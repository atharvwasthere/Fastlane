package server

import (
	"encoding/json"
	"fmt"
	"os"
)

type Server struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Location string `json:"location"`
	Country  string `json:"country"`
}

type Selector struct {
	Servers []Server
}


func NewSelector(filepath string) (*Selector, error) {
	var servers []Server
	
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read server file: %v", err)
	}
	
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, fmt.Errorf("failed to parse servers JSON: %v" , err) // %v prints the value in its default format 
	}
	fmt.Println(servers)

	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers found in %s" , filepath)
	}
	return &Selector{Servers: servers}, nil
}

func (s *Selector) SelectDefault() *Server {
	// Placeholder: Return first server (Phase 3 will use GeoIP)
	return &s.Servers[0]
}

func (s *Selector) GetServer(host string) *Server {
	for _, srv := range s.Servers {
		if srv.Host == host {
			return &srv
		}
	}
	return s.SelectDefault() // Fallback to default if not found
}