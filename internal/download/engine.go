package download

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/stats"
)

// Result holds the final download test results
type Result struct {
	FinalMbps        float64 // steady-state rate, excluding warm-up
	StdDevMbps       float64
	EWMAMbps         float64
	MeanMbps         float64
	MinMbps          float64
	MaxMbps          float64
	Threads          int
	Duration         time.Duration
	BytesDownloaded  int64
	ConvergenceCV    float64
	SamplesCollected int
	Converged        bool
	Errors           int64 // dial/status/read failures across worker lifetime
}

// Config holds download test configuration
type Config struct {
	URL            string        // Download URL (typically large file or stream)
	Threads        int           // Number of concurrent streams (default 4)
	RSEThreshold   float64       // Stop when the mean's relative standard error (CV/√n) is below this (default 0.03)
	UpdateInterval time.Duration // Sampling interval (default 100ms)
	MinSamples     int           // Minimum steady-state samples before convergence check (default 20)
	TestDuration   time.Duration // Maximum test duration (default 60s)
	WarmUp         time.Duration // Discarded ramp-up window, capped at TestDuration/3 (default 1s)
}

// Engine manages the download test. Workers add bytes to one atomic counter
// as they arrive; Run samples that counter every UpdateInterval.
type Engine struct {
	config          Config
	sampler         *stats.Sampler
	bytesDownloaded int64
	errCount        int64 // dial/status/read failures across all workers (atomic)
}

// NewEngine creates a new download test engine
func NewEngine(config Config) *Engine {
	if config.Threads <= 0 {
		config.Threads = 4
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

	return &Engine{
		config:  config,
		sampler: stats.NewSampler(config.WarmUp, config.MinSamples, config.RSEThreshold),
	}
}

// Run executes the download test. The caller's context governs lifecycle —
// cancel it (or wait for TestDuration) to terminate the run. Returns the
// partial Result accumulated up to that point.
func (e *Engine) Run(ctx context.Context) (*Result, error) {
	startTime := time.Now()
	testCtx, testCancel := context.WithTimeout(ctx, e.config.TestDuration)
	defer testCancel()
	e.sampler.Start(startTime)

	var wg sync.WaitGroup
	for i := 0; i < e.config.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.worker(testCtx)
		}()
	}

	ticker := time.NewTicker(e.config.UpdateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-testCtx.Done():
			wg.Wait()
			return e.buildResult(startTime), nil
		case now := <-ticker.C:
			if e.sampler.Observe(now, atomic.LoadInt64(&e.bytesDownloaded)) {
				testCancel()
				wg.Wait()
				return e.buildResult(startTime), nil
			}
		}
	}
}

// worker downloads data from URL until ctx ends
func (e *Engine) worker(ctx context.Context) {
	// Own HTTP/1.1 transport per worker: an h2 server would otherwise multiplex every worker onto one TCP connection.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Protocols = new(http.Protocols)
	transport.Protocols.SetHTTP1(true)
	transport.TLSClientConfig = &tls.Config{NextProtos: []string{"http/1.1"}} // without ALPN some servers assume h2
	// Bound stalls before headers only; a body timeout would cut slow links mid-file.
	transport.ResponseHeaderTimeout = 10 * time.Second
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}

	for ctx.Err() == nil {
		req, err := http.NewRequestWithContext(ctx, "GET", e.config.URL, nil)
		if err != nil {
			atomic.AddInt64(&e.errCount, 1)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			// Ignore ctx-cancel: that's a clean shutdown, not a real fault.
			if ctx.Err() == nil {
				atomic.AddInt64(&e.errCount, 1)
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// An error page (e.g. 403) is not throughput; count it as a failure.
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			atomic.AddInt64(&e.errCount, 1)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		_, err = io.Copy(io.Discard, stats.CountingReader{R: resp.Body, N: &e.bytesDownloaded})
		resp.Body.Close()
		if err != nil && ctx.Err() == nil {
			atomic.AddInt64(&e.errCount, 1)
		}
	}
}

// buildResult creates the final result
func (e *Engine) buildResult(startTime time.Time) *Result {
	now := time.Now()
	bytesTotal := atomic.LoadInt64(&e.bytesDownloaded)
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
		BytesDownloaded:  bytesTotal,
		ConvergenceCV:    s.CV,
		SamplesCollected: s.Samples,
		Converged:        s.Converged,
		Errors:           atomic.LoadInt64(&e.errCount),
	}
}

// GetErrorCount returns the current dial/status/read error tally.
func (e *Engine) GetErrorCount() int64 {
	return atomic.LoadInt64(&e.errCount)
}

// GetCurrentStats returns current statistics for live display
func (e *Engine) GetCurrentStats() (mean, ewma, stddev float64, converged bool, cv float64) {
	s := e.sampler.Stats()
	return s.Mean, s.EWMA, s.StdDev, s.Converged, s.CV
}

// GetBytesDownloaded returns total bytes downloaded
func (e *Engine) GetBytesDownloaded() int64 {
	return atomic.LoadInt64(&e.bytesDownloaded)
}

// GetSampleCount returns current number of steady-state samples collected
func (e *Engine) GetSampleCount() int {
	return e.sampler.Stats().Samples
}
