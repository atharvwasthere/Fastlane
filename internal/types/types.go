package types

import "time"

type ServerConfig struct {
    Name      string   `json:"name"`
    Host      string   `json:"host"`
    Port      int      `json:"port"`
    PortRange string   `json:"port_range,omitempty"` // e.g., "5200-5209"
    Options   []string `json:"options"`
    Country   string   `json:"country"`
    Location  string   `json:"location"`
    Continent string   `json:"continent"`
    Provider  string   `json:"provider"`
    Bandwidth string   `json:"bandwidth"`
}


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

type Iperf3Result struct {
    Start struct {
        Connected []struct {
            RemoteHost string `json:"remote_host"`
            RemotePort int    `json:"remote_port"`
        } `json:"connected"`
    } `json:"start"`
    Intervals []struct {
        Sum struct {
            BitsPerSecond float64 `json:"bits_per_second"`
        } `json:"sum"`
    } `json:"intervals"`
    End struct {
        SumSent struct {
            BitsPerSecond float64 `json:"bits_per_second"`
            Retransmits   int64   `json:"retransmits"`
            Bytes         int64   `json:"bytes"`
        } `json:"sum_sent"`
        SumReceived struct {
            BitsPerSecond float64 `json:"bits_per_second"`
            Bytes         int64   `json:"bytes"`
        } `json:"sum_received"`
        Sum struct {
            BitsPerSecond float64 `json:"bits_per_second"`
            JitterMs      float64 `json:"jitter_ms"`
            LostPackets   int64   `json:"lost_packets"`
            Packets       int64   `json:"packets"`
            LostPercent   float64 `json:"lost_percent"`
        } `json:"sum"`
        CPUUtilizationPercent struct {
            HostTotal   float64 `json:"host_total"`
            RemoteTotal float64 `json:"remote_total"`
        } `json:"cpu_utilization_percent"`
    } `json:"end"`
    Error string `json:"error"`
}


type FullResult struct {
	Ping     *PingResult     `json:"ping"`
	Download *DownloadResult `json:"download"`
	Upload   *UploadResult   `json:"upload"`
}

type Report struct {
	Timestamp string      `json:"timestamp"`
	Server    string      `json:"server"`
	Results   interface{} `json:"results"`
}