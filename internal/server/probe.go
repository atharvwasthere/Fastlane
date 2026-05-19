package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	defaultProbeConcurrency = 8
	defaultProbeTimeout     = 500 * time.Millisecond
)

// ProbeAll dials each server's host:port concurrently and records RTT.
// Concurrency is capped (defaults to 8) so we don't flood the user's link.
// Returned slice is aligned with the input order; failed dials carry
// Available=false and LatencyMS=0.
func ProbeAll(ctx context.Context, servers []Server, concurrency int) []*ServerWithMetrics {
	if concurrency <= 0 {
		concurrency = defaultProbeConcurrency
	}
	out := make([]*ServerWithMetrics, len(servers))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i := range servers {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, s Server) {
			defer wg.Done()
			defer func() { <-sem }()
			out[idx] = probeOne(ctx, s)
		}(i, servers[i])
	}
	wg.Wait()
	return out
}

func probeOne(ctx context.Context, s Server) *ServerWithMetrics {
	swm := &ServerWithMetrics{Server: &s}
	port := s.Port
	if port == 0 {
		port = 443
	}
	addr := fmt.Sprintf("%s:%d", s.Host, port)

	dialer := &net.Dialer{Timeout: defaultProbeTimeout}
	t0 := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return swm
	}
	swm.LatencyMS = float64(time.Since(t0).Microseconds()) / 1000.0
	swm.Available = true
	_ = conn.Close()
	return swm
}
