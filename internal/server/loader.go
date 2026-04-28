package server

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Loader handles loading servers from files
type Loader struct {
	servers []Server
}

// NewLoader creates a new server loader
func NewLoader() *Loader {
	return &Loader{servers: make([]Server, 0)}
}

// LoadFromJSON loads servers from a JSON file
func (l *Loader) LoadFromJSON(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	var serverList ServerList
	if err := json.Unmarshal(data, &serverList); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	l.servers = serverList.Servers
	return nil
}

// LoadDefault provides hardcoded fallback servers
func (l *Loader) LoadDefault() {
	l.servers = []Server{
		{
			ID:        "us-va-1",
			Name:      "Virginia, USA",
			Host:      "speedtest-va.example.com",
			IP:        "203.0.113.1",
			Location:  "Ashburn",
			Country:   "USA",
			Latitude:  38.9,
			Longitude: -77.6,
			Port:      80,
		},
		{
			ID:        "us-ca-1",
			Name:      "California, USA",
			Host:      "speedtest-ca.example.com",
			IP:        "198.51.100.1",
			Location:  "Los Angeles",
			Country:   "USA",
			Latitude:  34.0,
			Longitude: -118.2,
			Port:      80,
		},
		{
			ID:        "eu-de-1",
			Name:      "Germany",
			Host:      "speedtest-de.example.com",
			IP:        "192.0.2.1",
			Location:  "Frankfurt",
			Country:   "Germany",
			Latitude:  50.1,
			Longitude: 8.7,
			Port:      80,
		},
		{
			ID:        "eu-uk-1",
			Name:      "United Kingdom",
			Host:      "speedtest-uk.example.com",
			IP:        "192.0.2.2",
			Location:  "London",
			Country:   "UK",
			Latitude:  51.5,
			Longitude: -0.1,
			Port:      80,
		},
		{
			ID:        "sg-1",
			Name:      "Singapore",
			Host:      "speedtest-sg.example.com",
			IP:        "192.0.2.3",
			Location:  "Singapore",
			Country:   "Singapore",
			Latitude:  1.3,
			Longitude: 103.8,
			Port:      80,
		},
	}
}

// GetAll returns all loaded servers
func (l *Loader) GetAll() []Server {
	return l.servers
}

// Filter applies selection criteria and returns matching servers
func (l *Loader) Filter(criteria SelectionCriteria) []Server {
	var filtered []Server

	for _, s := range l.servers {
		// Filter by ID
		if criteria.ServerID != "" && s.ID != criteria.ServerID {
			continue
		}

		// Filter by city
		if criteria.City != "" && !strings.Contains(strings.ToLower(s.Location), strings.ToLower(criteria.City)) {
			continue
		}

		// Filter by keyword
		if criteria.Keyword != "" {
			keyword := strings.ToLower(criteria.Keyword)
			if !strings.Contains(strings.ToLower(s.Name), keyword) &&
				!strings.Contains(strings.ToLower(s.Country), keyword) &&
				!strings.Contains(strings.ToLower(s.Location), keyword) {
				continue
			}
		}

		filtered = append(filtered, s)
	}

	// Limit to max count
	if criteria.MaxCount > 0 && len(filtered) > criteria.MaxCount {
		filtered = filtered[:criteria.MaxCount]
	}

	return filtered
}

// SortByMetrics sorts servers by their score (ascending - lower is better)
func SortByMetrics(servers []*ServerWithMetrics) {
	sort.Slice(servers, func(i, j int) bool {
		// Available servers first
		if servers[i].Available != servers[j].Available {
			return servers[i].Available
		}
		// Then by score
		return servers[i].Score < servers[j].Score
	})
}

// SortByLatency sorts servers by latency (ascending)
func SortByLatency(servers []*ServerWithMetrics) {
	sort.Slice(servers, func(i, j int) bool {
		if servers[i].Available != servers[j].Available {
			return servers[i].Available
		}
		return servers[i].LatencyMS < servers[j].LatencyMS
	})
}
