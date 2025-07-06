package types

import "time"

// Server represents an Ookla Speedtest server.
type Server struct {
    ID          string  `json:"id"`          // Unique server ID from Ookla
    Host        string  `json:"host"`        // Server hostname (e.g., speedtest.example.com:8080)
    Name        string  `json:"name"`        // Server name (e.g., New York)
    Country     string  `json:"country"`     // Country of the server
    Location    string  `json:"location"`    // City or region
    Sponsor     string  `json:"sponsor"`     // Server provider/sponsor
    Latitude    float64 `json:"lat"`         // Server latitude
    Longitude   float64 `json:"lon"`         // Server longitude
    Distance    float64 `json:"distance"`    // Distance from user (km)
    Latency     time.Duration `json:"latency"` // Measured ping latency
}

// ConnectionMetrics captures detailed timing metrics for a connection.
type ConnectionMetrics struct {
    DNSResolution time.Duration `json:"dns_resolution"` // Time to resolve hostname
    TCPHandshake  time.Duration `json:"tcp_handshake"`  // Time for TCP 3-way handshake
    TLSSetup      time.Duration `json:"tls_setup"`      // Time for TLS negotiation
    TTFB          time.Duration `json:"ttfb"`           // Time to first byte
    TotalRTT      time.Duration `json:"total_rtt"`      // Sum of above components
}

// BandwidthSample represents a single speed measurement sample.
type BandwidthSample struct {
    Timestamp       time.Time `json:"timestamp"`        // Time of sample
    BytesTransferred int64     `json:"bytes_transferred"` // Bytes transferred in sample
    SpeedMbps        float64   `json:"speed_mbps"`       // Speed in Mbps
}

// PingResult captures the results of a ping test.
type PingResult struct {
    Server      string            `json:"server"`       // Target server hostname
    Metrics     ConnectionMetrics `json:"metrics"`      // Detailed connection timings
    Success     bool              `json:"success"`      // Whether the test completed
    Error       string            `json:"error,omitempty"` // Error message if failed
}

// DownloadResult captures the results of a download test.
type DownloadResult struct {
    Server           string           `json:"server"`          // Target server hostname
    Samples          []BandwidthSample `json:"samples"`         // Time-series speed samples
    TotalBytes       int64            `json:"total_bytes"`     // Total bytes transferred
    Duration         time.Duration    `json:"duration"`        // Test duration
    AverageSpeedMbps float64          `json:"average_speed_mbps"` // Average speed in Mbps
    Success          bool             `json:"success"`         // Whether the test completed
    Error            string           `json:"error,omitempty"` // Error message if failed
}

// UploadResult captures the results of an upload test.
type UploadResult struct {
    Server           string           `json:"server"`          // Target server hostname
    Samples          []BandwidthSample `json:"samples"`         // Time-series speed samples
    TotalBytes       int64            `json:"total_bytes"`     // Total bytes transferred
    Duration         time.Duration    `json:"duration"`        // Test duration
    AverageSpeedMbps float64          `json:"average_speed_mbps"` // Average speed in Mbps
    Success          bool             `json:"success"`         // Whether the test completed
    Error            string           `json:"error,omitempty"` // Error message if failed
}

// TestReport aggregates results from all tests and metadata.
type TestReport struct {
    Timestamp time.Time       `json:"timestamp"` // Time of test
    Server    Server          `json:"server"`    // Selected server details
    Ping      *PingResult     `json:"ping"`      // Ping test results
    Download  *DownloadResult `json:"download"`  // Download test results
    Upload    *UploadResult   `json:"upload"`    // Upload test results
    Summary   string          `json:"summary"`   // Human-readable summary
}

type Userinfo struct {
	IP      string
	Lat     float64
	Lon     float64
	City    string
	Region  string
	Country string
	ISP     string
	Source  string // "api" or "mmdb"
}
