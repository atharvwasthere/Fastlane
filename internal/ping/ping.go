package ping

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
	"time"
)

// LatencyResult holds latency measurements for each layer
type LatencyResult struct {
	DNSLatencyMS   float64
	TCPLatencyMS   float64
	TLSLatencyMS   float64
	HTTPLatencyMS  float64 // time to first byte: request written → first response byte
	TotalLatencyMS float64
	JitterMS       float64
	MinMS          float64
	MaxMS          float64
	MeanMS         float64
	Samples        []float64 // TCP connect times; latency and jitter come from these
	Errors         int64     // probes that failed at any layer
}

// MeasureLayered sends iterations requests, each on a fresh connection, and
// times DNS, TCP connect, TLS handshake and TTFB of the same request with
// httptrace, so no layer's number includes another's.
func MeasureLayered(ctx context.Context, host string, iterations int, timeout time.Duration) (*LatencyResult, error) {
	if iterations < 1 {
		iterations = 5
	}
	target, err := targetURL(host)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{DisableKeepAlives: true} // a reused connection would skip DNS/TCP/TLS
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse // time the first response, not a redirect chain
		},
	}

	result := &LatencyResult{}
	var dnsS, tlsS, ttfbS []float64
	for i := 0; i < iterations; i++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		p, err := probe(ctx, client, target)
		if err != nil {
			result.Errors++
		}
		// Keep layers that completed even if a later one failed (e.g. TCP ok, TLS cert bad).
		if p.dns > 0 {
			dnsS = append(dnsS, p.dns)
		}
		if p.tcp > 0 {
			result.Samples = append(result.Samples, p.tcp)
		}
		if p.tls > 0 {
			tlsS = append(tlsS, p.tls)
		}
		if p.ttfb > 0 {
			ttfbS = append(ttfbS, p.ttfb)
		}
	}

	if len(result.Samples) == 0 {
		return result, fmt.Errorf("ping %s: no successful probes (%d errors)", host, result.Errors)
	}
	result.DNSLatencyMS = average(dnsS)
	result.TCPLatencyMS = average(result.Samples)
	result.TLSLatencyMS = average(tlsS)
	result.HTTPLatencyMS = average(ttfbS)
	result.MeanMS = result.TCPLatencyMS
	result.MinMS = minimum(result.Samples)
	result.MaxMS = maximum(result.Samples)
	result.JitterMS = stddev(result.Samples)
	result.TotalLatencyMS = result.MeanMS

	return result, nil
}

type phases struct{ dns, tcp, tls, ttfb float64 }

// probe runs one request and returns the duration of each phase it reached.
func probe(ctx context.Context, client *http.Client, target string) (phases, error) {
	var (
		mu                        sync.Mutex
		p                         phases
		dnsStart, tlsStart, wrote time.Time
		connStart                 = map[string]time.Time{}
	)
	// Callbacks can run on dialer goroutines (Happy Eyeballs races IPv4/IPv6), so guard everything.
	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) { mu.Lock(); dnsStart = time.Now(); mu.Unlock() },
		DNSDone:  func(httptrace.DNSDoneInfo) { mu.Lock(); p.dns = sinceMS(dnsStart); mu.Unlock() },
		ConnectStart: func(_, addr string) {
			mu.Lock()
			connStart[addr] = time.Now()
			mu.Unlock()
		},
		ConnectDone: func(_, addr string, err error) {
			mu.Lock()
			if err == nil && p.tcp == 0 {
				p.tcp = sinceMS(connStart[addr]) // per-address start: the winning dial, not the first one tried
			}
			mu.Unlock()
		},
		TLSHandshakeStart: func() { mu.Lock(); tlsStart = time.Now(); mu.Unlock() },
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			mu.Lock()
			if err == nil {
				p.tls = sinceMS(tlsStart)
			}
			mu.Unlock()
		},
		WroteRequest:         func(httptrace.WroteRequestInfo) { mu.Lock(); wrote = time.Now(); mu.Unlock() },
		GotFirstResponseByte: func() { mu.Lock(); p.ttfb = sinceMS(wrote); mu.Unlock() },
	}

	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), http.MethodGet, target, nil)
	if err != nil {
		return phases{}, err
	}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close() // only the first byte matters
	}
	mu.Lock()
	defer mu.Unlock()
	return p, err
}

// targetURL accepts "host", "host:port" or a URL and returns scheme://host/.
// Bare hosts default to https, the port every public server answers on.
func targetURL(host string) (string, error) {
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	u, err := url.Parse(host)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("ping: invalid host %q", host)
	}
	return u.Scheme + "://" + u.Host + "/", nil
}

// sinceMS keeps sub-millisecond precision that Duration.Milliseconds() truncates.
func sinceMS(start time.Time) float64 {
	return float64(time.Since(start)) / float64(time.Millisecond)
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
