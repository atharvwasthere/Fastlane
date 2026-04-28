package server

// Server represents a test server
type Server struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Host     string  `json:"host"`
	IP       string  `json:"ip"`
	Location string  `json:"location"`
	Country  string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Port     int     `json:"port"`
}

// ServerWithMetrics extends Server with computed metrics
type ServerWithMetrics struct {
	Server              *Server
	LatencyMS           float64
	DistanceKM          float64
	ThroughputMbps      float64
	Score               float64
	Available           bool
}

// ServerList represents a collection of servers
type ServerList struct {
	Servers []Server `json:"servers"`
	Updated string   `json:"updated"`
}

// SelectionCriteria holds filtering options
type SelectionCriteria struct {
	ServerID string
	City     string
	Coords   string // "latitude,longitude"
	Keyword  string
	MaxCount int
}

// UserLocation represents user geographic info
type UserLocation struct {
	IP        string
	Latitude  float64
	Longitude float64
	ISP       string
}
