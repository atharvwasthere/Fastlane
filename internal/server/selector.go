package server

import (
	"encoding/json"
	"fmt"
	"go/types"
	"os"

	"github.com/atharvwasthere/Fastlane/internal/location"
	"github.com/atharvwasthere/Fastlane/internal/nettest"
	"github.com/atharvwasthere/Fastlane/internal/types"
)

type Selector struct {
	Servers []types.ServerConfig
}


func NewSelector(filepath string) (*Selector, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read server file: %v", err)
	}

	var servers []types.ServerConfig

	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, fmt.Errorf("failed to parse servers JSON: %v" , err) // %v prints the value in its default format 
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers found in %s" , filepath)
	}
	return &Selector{Servers: servers}, nil
}


func (s *Selector) SelectDefault() (*types.ServerConfig, *types.PingResult , error) {
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