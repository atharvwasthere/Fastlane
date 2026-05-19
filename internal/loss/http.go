package loss

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// HTTPEngine measures loss using HEAD probes. "Loss" is the fraction of
// probes that fail to return a response within the per-request deadline,
// classified separately from DNS/connection-reset errors (those land in
// errors). HEAD avoids body bytes — pure round-trip.
type HTTPEngine struct {
	cfg    Config
	client *http.Client

	mu       sync.RWMutex
	sent     int64
	received int64
	errCount int64
	rtts     []float64 // ms
	notes    []string
}

// NewHTTPEngine builds an HTTPEngine. Defaults match the skill spec.
func NewHTTPEngine(cfg Config) *HTTPEngine {
	if cfg.Count == 0 {
		cfg.Count = 100
	}
	if cfg.Rate == 0 {
		cfg.Rate = 10
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	transport := &http.Transport{
		DisableKeepAlives:     true, // each probe is its own round trip
		MaxIdleConnsPerHost:   1,
		ResponseHeaderTimeout: 2 * time.Second,
	}
	return &HTTPEngine{
		cfg: cfg,
		client: &http.Client{
			Transport: transport,
			Timeout:   2 * time.Second,
		},
		rtts: make([]float64, 0, cfg.Count),
	}
}

// Run sends Count HEAD probes at Rate/sec. Caller's context governs
// lifecycle; ctx cancel returns whatever was collected so far.
func (e *HTTPEngine) Run(ctx context.Context) (*Result, error) {
	startTime := time.Now()
	url := normalizeURL(e.cfg.Host)

	testCtx, cancel := context.WithTimeout(ctx, e.cfg.Timeout)
	defer cancel()

	interval := time.Second / time.Duration(e.cfg.Rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for i := 0; i < e.cfg.Count; i++ {
		select {
		case <-testCtx.Done():
			goto done
		case <-ticker.C:
		}
		e.probe(testCtx, url)
	}
done:
	return e.buildResult(time.Since(startTime)), nil
}

func (e *HTTPEngine) probe(ctx context.Context, url string) {
	atomic.AddInt64(&e.sent, 1)
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, url, nil)
	if err != nil {
		atomic.AddInt64(&e.errCount, 1)
		return
	}
	t0 := time.Now()
	resp, err := e.client.Do(req)
	if err != nil {
		// DeadlineExceeded counts as loss (timeout); other errors are
		// classified — DNS/connection-refused isn't loss, surface in errors.
		if errors.Is(err, context.DeadlineExceeded) {
			return // sent counted, received not — counts toward loss
		}
		atomic.AddInt64(&e.errCount, 1)
		return
	}
	resp.Body.Close()
	rtt := float64(time.Since(t0).Milliseconds())
	e.mu.Lock()
	e.received++
	e.rtts = append(e.rtts, rtt)
	e.mu.Unlock()
}

// GetCurrentStats mirrors UDPEngine for parity with future live renderers.
func (e *HTTPEngine) GetCurrentStats() (sent, received, lost int64, lossPercent float64) {
	sent = atomic.LoadInt64(&e.sent)
	e.mu.RLock()
	received = e.received
	e.mu.RUnlock()
	lost = sent - received
	if sent > 0 {
		lossPercent = float64(lost) / float64(sent) * 100.0
	}
	return
}

func (e *HTTPEngine) buildResult(duration time.Duration) *Result {
	e.mu.RLock()
	defer e.mu.RUnlock()

	sent := atomic.LoadInt64(&e.sent)
	received := e.received
	errCount := atomic.LoadInt64(&e.errCount)
	// Errors don't get blamed as loss — they're connection-class faults.
	probedSent := sent - errCount
	if probedSent < 0 {
		probedSent = 0
	}
	lost := probedSent - received
	if lost < 0 {
		lost = 0
	}
	lossPct := 0.0
	if probedSent > 0 {
		lossPct = float64(lost) / float64(probedSent) * 100.0
	}

	min, avg, max, jitter := rttStats(e.rtts)

	notes := append([]string(nil), e.notes...)
	if errCount > 0 {
		notes = append(notes, "non-loss errors: connection failures excluded from loss%")
	}

	return &Result{
		PacketsSent:     sent,
		PacketsReceived: received,
		PacketsLost:     lost,
		Errors:          errCount,
		LossPercent:     lossPct,
		JitterMS:        jitter,
		MinLatencyMS:    min,
		MaxLatencyMS:    max,
		AvgLatencyMS:    avg,
		Duration:        duration,
		TestComplete:    true,
		Notes:           notes,
	}
}

func rttStats(rtts []float64) (min, avg, max, stddev float64) {
	if len(rtts) == 0 {
		return 0, 0, 0, 0
	}
	min = rtts[0]
	max = rtts[0]
	sum := 0.0
	for _, v := range rtts {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}
	avg = sum / float64(len(rtts))
	if len(rtts) > 1 {
		variance := 0.0
		for _, v := range rtts {
			d := v - avg
			variance += d * d
		}
		variance /= float64(len(rtts))
		stddev = math.Sqrt(variance)
	}
	return
}

// normalizeURL ensures the target has a scheme. Bare host/IP gets http://
// prefix; existing http:// or https:// is left alone.
func normalizeURL(host string) string {
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host
	}
	return "http://" + host
}
