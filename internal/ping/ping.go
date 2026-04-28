package ping

import (
	"crypto/tls"
	"math"
	"net"
	"net/http"
	"strings"
	"time"
)

// LatencyResult holds latency measurements for each layer
type LatencyResult struct {
	DNSLatencyMS   float64
	TCPLatencyMS   float64
	TLSLatencyMS   float64
	HTTPLatencyMS  float64
	TotalLatencyMS float64
	JitterMS       float64
	MinMS          float64
	MaxMS          float64
	MeanMS         float64
	Samples        []float64
}

// MeasureLayered measures latency at each protocol layer
func MeasureLayered(host string, iterations int, timeout time.Duration) (*LatencyResult, error) {
	if iterations < 1 {
		iterations = 5
	}

	result := &LatencyResult{
		Samples: make([]float64, 0),
	}

	// Normalize host (remove protocol prefix if present)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.Split(host, "/")[0] // Remove path

	// 1. Measure DNS latency
	dnsSamples := make([]float64, 0)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := net.LookupIP(host)
		if err == nil {
			dnsSamples = append(dnsSamples, float64(time.Since(start).Milliseconds()))
		}
	}
	if len(dnsSamples) > 0 {
		result.DNSLatencyMS = average(dnsSamples)
	}

	// 2. Measure TCP latency
	tcpSamples := make([]float64, 0)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", host+":80", timeout)
		if err == nil {
			tcpSamples = append(tcpSamples, float64(time.Since(start).Milliseconds()))
			conn.Close()
		}
	}
	if len(tcpSamples) > 0 {
		result.TCPLatencyMS = average(tcpSamples)
	}

	// 3. Measure TLS latency
	tlsSamples := make([]float64, 0)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		conn, err := tls.DialWithDialer(
			&net.Dialer{Timeout: timeout},
			"tcp",
			host+":443",
			&tls.Config{InsecureSkipVerify: true},
		)
		if err == nil {
			tlsSamples = append(tlsSamples, float64(time.Since(start).Milliseconds()))
			conn.Close()
		}
	}
	if len(tlsSamples) > 0 {
		result.TLSLatencyMS = average(tlsSamples)
	}

	// 4. Measure HTTP latency (fallback)
	httpSamples := make([]float64, 0)
	client := &http.Client{Timeout: timeout}
	for i := 0; i < iterations; i++ {
		start := time.Now()
		resp, err := client.Get("http://" + host)
		if err == nil {
			httpSamples = append(httpSamples, float64(time.Since(start).Milliseconds()))
			resp.Body.Close()
		}
	}
	if len(httpSamples) > 0 {
		result.HTTPLatencyMS = average(httpSamples)
	}

	// Collect all samples for overall statistics
	result.Samples = append(result.Samples, dnsSamples...)
	result.Samples = append(result.Samples, tcpSamples...)

	if len(result.Samples) > 0 {
		result.MeanMS = average(result.Samples)
		result.MinMS = minimum(result.Samples)
		result.MaxMS = maximum(result.Samples)
		result.JitterMS = stddev(result.Samples)
		result.TotalLatencyMS = result.MeanMS
	}

	return result, nil
}

// helper functions
func average(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func minimum(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	min := vals[0]
	for _, v := range vals {
		if v < min {
			min = v
		}
	}
	return min
}

func maximum(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	max := vals[0]
	for _, v := range vals {
		if v > max {
			max = v
		}
	}
	return max
}

func stddev(vals []float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	mean := average(vals)
	sumSq := 0.0
	for _, v := range vals {
		diff := v - mean
		sumSq += diff * diff
	}
	variance := sumSq / float64(len(vals)-1)
	return math.Sqrt(variance)
}
