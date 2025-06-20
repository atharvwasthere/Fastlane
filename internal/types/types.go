package types

import "time"

type DownloadResult struct {
	BytesTransferred int64         `json:"bytes_transferred"`
	Duration         time.Duration `json:"duration"`
	SpeedMbps        float64       `json:"speed_mbps"`
	Server           string        `json:"server"`
	Success          bool          `json:"success"`
	Error            string        `json:"error,omitempty"`
}

type PingResult struct {
	DNSResolution time.Duration `json:"dns_resolution"`  // Time to resolve hostname
	TCPHandshake  time.Duration `json:"tcp_handshake"`   // Time for TCP 3-way handshake
	TLSSetup      time.Duration `json:"tls_setup"`       // Time for TLS negotiation
	TTFB          time.Duration `json:"ttfb"`            // Time to first byte of HTTP response
	TotalRTT      time.Duration `json:"total_rtt"`       // Sum of components
	Server        string        `json:"server"`          // Target server hostname
	Success       bool          `json:"success"`         // Whether the test completed fully
	Error         string        `json:"error,omitempty"` // Error message if failed
}

type UploadResult struct {
    BytesTransferred int64         `json:"bytes_transferred_upload"`
    Duration         time.Duration `json:"duration"`
    SpeedMbps        float64       `json:"speed_mbps"`
    Server           string        `json:"server"`
    Success          bool          `json:"success"`
    Error            string        `json:"error,omitempty"`
}
