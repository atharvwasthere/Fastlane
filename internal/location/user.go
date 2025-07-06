package location

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/atharvwasthere/Fastlane/types"
	"github.com/atharvwasthere/Fastlane/utils"
)

func fetchUserLocation(ctx context.Context, cmd string, servers []types.Server, mmdbPath string) (*location.Location, error) {
	if cmd == "myserver" {
		// use GeoIP Lookup
		ip, err := utils.GetPublicIP() 
		if err != nil {
			return nil, fmt.Errorf("failed to get public IP: %v", err)
		}

		geo, err := location.LookupGeoFromIP(ip, mmdbPath)
		if err != nil {
			return nil, fmt.Errorf("GeoIP lookup failed: %v", err)
		}

		return &location.Location{
			Name: geo.ISP,
			Lat: geo.Lat,
			Lon: geo.Lon,
			City: geo.City,
			Region: geo.Region,
		}, nil
	}

	// for every other command: fetch from speedtest.net ios-config API
	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers available for user location fetch")
	}
	serverID := servers[0].ID

	u, err := url.Parse("https://www.speedtest.net/api/ios-config.php")
	if err != nil {
		return nil, fmt.Errorf("failed to parse ios-config URL: %v", err)
	}

	query := u.Query()
	query.Set("serverID", serverID)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ios-config request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ios-config: %v", err)
	}
	defer resp.Body.Close();

	var serverList ServerList
	if err := xml.NewDecoder(resp.Body).Decode(&serverList); err != nil {
		return nil, fmt.Errorf("failed to parse ios-config XML: %v", err)
	}

	if serverList.Client.Lat =="" || serverList.Client.Lon == "" {
		return nil, fmt.Errorf("no user location data in ios-config response")
	}
	lat, err := strconv.ParseFloat(serverList.Client.Lat, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid latitude: %v", err)
	}
		lon, err := strconv.ParseFloat(serverList.Client.Lon, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid longitude: %v", err)
	}

	return &location.Location{
		Name: serverList.Client.Isp,
		Lat:  lat,
		Lon:  lon,
	}, nil

}