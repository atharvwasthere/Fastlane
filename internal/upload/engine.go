package upload

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/stats"
)

// Result holds the final upload test results
type Result struct {
	FinalMbps        float64 // steady-state rate, excluding warm-up
	StdDevMbps       float64
	EWMAMbps         float64
	MeanMbps         float64
	MinMbps          float64
	MaxMbps          float64
	Threads          int
	Duration         time.Duration
	BytesUploaded    int64
	ConvergenceCV    float64
	SamplesCollected int
	Converged        bool
	Errors           int64 // dial/write/status failures across worker lifetime
}

// Config holds upload test configuration
type Config struct {
	URL            string        // Upload URL endpoint
	Threads        int           // Number of concurrent streams (default 4)
	Timeout        time.Duration // Overall test timeout (default 30s)
	RSEThreshold   float64       // Stop when the mean's relative standard error (CV/√n) is below this (default 0.03)
	UpdateInterval time.Duration // Sampling interval (default 100ms)
	MinSamples     int           // Minimum steady-state samples before convergence check (default 20)
	TestDuration   time.Duration // Maximum test duration (default 60s)
	WarmUp         time.Duration // Discarded ramp-up window, capped at TestDuration/3 (default 1s)
	ChunkSize      int64         // Size of each upload request body in bytes (default 1MB)
}

// Engine manages the upload test. Workers add bytes to one atomic counter
// as the transport reads the body; Run samples that counter every UpdateInterval.
type Engine struct {
	config        Config
	sampler       *stats.Sampler
	bytesUploaded int64
	errCount      int64 // dial/write/status failures across all workers (atomic)
}

// NewEngine creates a new upload test engine
func NewEngine(config Config) *Engine {
	if config.Threads <= 0 {
		config.Threads = 4
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RSEThreshold == 0 {
		config.RSEThreshold = 0.03
	}
	if config.UpdateInterval == 0 {
		config.UpdateInterval = 100 * time.Millisecond
	}
	if config.MinSamples == 0 {
		config.MinSamples = 20
	}
	if config.TestDuration == 0 {
		config.TestDuration = 60 * time.Second
	}
	if config.WarmUp == 0 {
		config.WarmUp = time.Second
	}
	config.WarmUp = min(config.WarmUp, config.TestDuration/3)
	if config.ChunkSize == 0 {
		config.ChunkSize = 1024 * 1024 // 1MB default
	}

	return &Engine{
		config:  config,
		sampler: stats.NewSampler(config.WarmUp, config.MinSamples, config.RSEThreshold),
	}
}

// Run executes the upload test. Caller's context governs lifecycle.
func (e *Engine) Run(ctx context.Context) (*Result, error) {
	startTime := time.Now()
	testCtx, testCancel := context.WithTimeout(ctx, e.config.TestDuration)
	defer testCancel()
	e.sampler.Start(startTime)

	var wg sync.WaitGroup
	for i := 0; i < e.config.Threads; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			e.worker(testCtx, threadID)
		}(i)
	}

	ticker := time.NewTicker(e.config.UpdateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-testCtx.Done():
			wg.Wait()
			return e.buildResult(startTime), nil
		case now := <-ticker.C:
			if e.sampler.Observe(now, atomic.LoadInt64(&e.bytesUploaded)) {
				testCancel()
				wg.Wait()
				return e.buildResult(startTime), nil
			}
		}
	}
}

// worker uploads data to URL until ctx ends
func (e *Engine) worker(ctx context.Context, threadID int) {
	// Own HTTP/1.1 transport per worker: an h2 server would otherwise multiplex every worker onto one TCP connection.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Protocols = new(http.Protocols)
	transport.Protocols.SetHTTP1(true)
	transport.TLSClientConfig = &tls.Config{NextProtos: []string{"http/1.1"}} // without ALPN some servers assume h2
	// Bound the wait for a response after the body is sent; a whole-request timeout would cut slow uplinks mid-body.
	transport.ResponseHeaderTimeout = 10 * time.Second
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}

	// Pre-allocate one random buffer per worker, reuse across requests.
	// math/rand is fine here — payload entropy is irrelevant to bandwidth.
	buffer := make([]byte, e.config.ChunkSize)
	rng := rand.New(rand.NewSource(time.Now().UnixNano() ^ int64(threadID)))
	rng.Read(buffer)

	for ctx.Err() == nil {
		var reqBytes int64
		body := stats.CountingReader{R: stats.CountingReader{R: bytes.NewReader(buffer), N: &reqBytes}, N: &e.bytesUploaded}
		req, err := http.NewRequestWithContext(ctx, "POST", e.config.URL, body)
		if err != nil {
			atomic.AddInt64(&e.errCount, 1)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		req.ContentLength = e.config.ChunkSize // a wrapped reader would otherwise be sent chunked
		req.Header.Set("Content-Type", "application/octet-stream")

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() == nil {
				atomic.AddInt64(&e.errCount, 1)
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// A rejected upload is not throughput; take back the bytes this request counted.
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			atomic.AddInt64(&e.bytesUploaded, -atomic.LoadInt64(&reqBytes))
			atomic.AddInt64(&e.errCount, 1)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// buildResult creates the final result
func (e *Engine) buildResult(startTime time.Time) *Result {
	now := time.Now()
	bytesTotal := atomic.LoadInt64(&e.bytesUploaded)
	s := e.sampler.Stats()
	return &Result{
		FinalMbps:        e.sampler.Mbps(now, bytesTotal),
		StdDevMbps:       s.StdDev,
		EWMAMbps:         s.EWMA,
		MeanMbps:         s.Mean,
		MinMbps:          s.Min,
		MaxMbps:          s.Max,
		Threads:          e.config.Threads,
		Duration:         now.Sub(startTime),
		BytesUploaded:    bytesTotal,
		ConvergenceCV:    s.CV,
		SamplesCollected: s.Samples,
		Converged:        s.Converged,
		Errors:           atomic.LoadInt64(&e.errCount),
	}
}

// GetErrorCount returns the current dial/write/status error tally.
func (e *Engine) GetErrorCount() int64 {
	return atomic.LoadInt64(&e.errCount)
}

// GetCurrentStats returns current statistics for live display
func (e *Engine) GetCurrentStats() (mean, ewma, stddev float64, converged bool, cv float64) {
	s := e.sampler.Stats()
	return s.Mean, s.EWMA, s.StdDev, s.Converged, s.CV
}

// GetBytesUploaded returns total bytes uploaded
func (e *Engine) GetBytesUploaded() int64 {
	return atomic.LoadInt64(&e.bytesUploaded)
}

// GetSampleCount returns current number of steady-state samples collected
func (e *Engine) GetSampleCount() int {
	return e.sampler.Stats().Samples
}
