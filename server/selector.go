package server

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"

	"github.com/atharvwasthere/Fastlane/internal/location"
	"github.com/atharvwasthere/Fastlane/internal/nettest"
	"github.com/atharvwasthere/Fastlane/types"
)

// Selector manages server selection for Fastlane.
type Selector struct {
	Servers   []types.Server
	UserInfo  *types.Userinfo
}

// ServerList represents the XML response from ios-config.php for server validation.
type ServerList struct {
	Servers []struct {
		URL     string  `xml:"url,attr"`
		Lat     string  `xml:"lat,attr"`
		Lon     string  `xml:"lon,attr"`
		Name    string  `xml:"name,attr"`
		Sponsor string  `xml:"sponsor,attr"`
		ID      string  `xml:"id,attr"`
		Country string  `xml:"country,attr"`
	} `xml:"servers>server"`
}



// This initializes a Selector, fetching servers from Ookla APIs or a local file.
func NewSelector(ctx context.Context, filepath string, userInfo *types.Userinfo , mmdbPath string) (*Selector, error){
	selector :=  &Selector{UserInfo: userInfo}

	// Fetching from Primary  JSON API
	servers, err := fetchServerJSON(ctx)
	if err != nil {
		// Falback to XML API
		servers, err = fetchServersXML(ctx)
		if err != nil {
			// Fallback to local server.json
			servers, err = loadServersFromFile(filepath)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch from all sources: %v", err)
			}
		}
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers available")
	}

	selector.Servers = servers

	// Fetching user location 
	if userInfo == nil {
		userInfo, err = location.fetchUserLocation(ctx,"",servers, mmdbPath)
		if err != nil {
			// will proceed without it but log the error 
			fmt.Printf("Warning: failed to fetch user location: %v\n", err)
		} else {
			selector.UserInfo = userInfo
		}
	}
	return selector, nil

}

// Fetches Servers from Ookla's JSON API 

func fetchServersJSON(ctx context.Context) ([]types.Server, error) {
	url := "https://www.speedtest.net/api/js/servers"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil{
		return nil, fmt.Errorf("failed to create JSON request : %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON servers: %v", err)
	}
	defer resp.Body.Close()

	var servers []types.Server
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		return nil, fmt.Errorf("failed to parse JSON servers: %v", err)
	}

	return servers, nil
}

// fetches server from servers form XML fallback API.

func fetchServersXML(ctx context.Context) ([]types.Server, error) {
	url := "https://www.speedtest.net/speedtest-servers-static.php"
	req, err := http.NewRequestWithContext(ctx , http.MethodGet, url, nil)
	if err != nil {
		return nil , fmt.Errorf("failed to create XML requests: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch XML Servers: %v", err)
	}
	defer resp.Body.Close()

	var serverList struct {
		Servers []types.Server 	`xml:"server>server"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&serverList); err != nil {
		return nil, fmt.Errorf("failed to parse XML servers: %v", err)
	}

	return serverList.Servers, nil
}

// loads servers from server.json file as a fallback 

func loadServersFromFile(filepath string) ([]types.Server, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read server file %s: %v", filepath, err)
	}

	var servers []types.Server
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, fmt.Errorf("failed to parse servers JSON: %v", err)
	}

	return servers, nil
}

// Select the default choose the best server using the Score obtained 
func (s *Selector) SelectDefault() (*types.Server, *types.PingResult, error) {
	if len(s.Servers) == 0 {
		return nil, nil ,fmt.Errorf("No servers available")
	}

	//Ping all the servers Concurrently
	var wg sync.WaitGroup
	results := make([]*types.PingResult, len(s.Servers))
	for i, srv  range s.Server {
		wg.Add(1)
		go func(idx int, server types.Server) {
			defer wg.Done()
			pingResult, err := nettest.Ping(&server)
			if err != nil {
				pingResult = &types.PingResult{
					Server: server.Host,
					Success: false,
					Error: err.Error(),
				}
			}
			results[idx] = pingResult
			s.Servers[idx].Latency = pingResult.Metrics.TotalRTT
		}(1, srv)
	}
	wg.Wait()

	// ranking servers
	scorer := NewScorer(s.UserInfo)
	bestServer, bestPing := scorer.RankServers(s.Servers, results)
	if bestServer == nil {
		return nil, nil, fmt.Errorf("No valid servers found after scoring")
	}

	return bestServer, bestPing, nil
}

/* 
	 for cmd myserver which displays server info 
*/
func (s *Selector) GetServer(hostOrID string) (*types.Server, *&types.PingResult, error) {
	 for _, srv := range s.Servers {
		if srv.Host == hostOrID || srv.ID == hostOrID {
			pingResult, err := nettest.Ping(&srv) 
			if err != nil {
				return nil, &types.PingResult{
					Server: srv.Host,
					Success: false,
					Error: err.Error(),
				}, err
			}
			return &srv, pingResult, nil
		}
	 }
	 return s.SelectDefault()
}

func (s *Selector) myServer() (*types.Server, error) {
	server, _, err := s.SelectDefault()
	if err != nil {
		return nil, fmt.Errorf("failed to select server: %v", err)
	}
	server, err = s.validateServerDetails(context.Background(), server.ID)
	if err != nil {
		return server, nil
	}
	return server, nil
}

// fetching server details from ios-config.php to ensure TCP compatibility

func (s *Selector) validateServerDetails(ctx context.Context, serverID string) ( *types.Server, error) {
	u, err := url.Parse("https://www.speedtest.net/api/ios-config.php")
	if err != nil {
		return nil, fmt.Errorf("failed to parse ios-config URL: %v", err)
	}
	query := u.Query()
	query.Set("serverID", serverID)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx,http.MethodGet,u.String(),nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ios-config request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ios-config: %v", err)
	}
	defer resp.Body.Close()

	var serverList ServerList
	if err := xml.NewDecoder(resp.Body).Decode(&serverList); err != nil {
		return nil, fmt.Errorf("failed to parse ios-config XML: %v", err)
	}

	for _, srv := range serverList.Server {
		if srv.ID == serverID {
			// extracting the Host(Host:port)
			parsedURL, err := url.Parse(srv.URL)
			if err != nil {
				return nil, fmt.Errorf("failed to parse server URL %s: %v", srv.URL, err)
			}
			host := parsedURL.Host
			if host == "" {
				host = parsedURL.Hostname() + ":8080" // Default to 8080 if no port
			}
			lat, err := strconv.ParseFloat(srv.Lat, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid latitude for server %s: %v", srv.ID, err)
			}
			lon, err := strconv.ParseFloat(srv.Lon, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid longitude for server %s: %v", srv.ID, err)
			}
			return &types.Server{
				ID:        srv.ID,
				Host:      host,
				Name:      srv.Name,
				Country:   srv.Country,
				Location:  srv.Name, // Use Name as Location (city)
				Sponsor:   srv.Sponsor,
				Latitude:  lat,
				Longitude: lon,
			}, nil
		}
	}
	return nil, fmt.Errorf("server ID %s not found in ios-config response", serverID) 
}

	