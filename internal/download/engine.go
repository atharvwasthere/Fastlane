package download

import (
	"context"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/stats"
)

// Result holds the final download test results
type Result struct {
	FinalMbps        float64
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
}

// Config holds download test configuration
type Config struct {
	URL              string        // Download URL (typically large file or stream)
	Threads          int           // Number of concurrent streams (default 4)
	Timeout          time.Duration // Overall test timeout (default 30s)
	CVThreshold      float64       // Convergence threshold (default 0.03)
	UpdateInterval   time.Duration // Progress update interval (default 100ms)
	MinSamples       int           // Minimum samples before convergence check (default 5)
	TestDuration     time.Duration // Maximum test duration (default 60s)
}

// Engine manages the download test
type Engine struct {
	config Config
	ctx    context.Context
	cancel context.CancelFunc

	// Synchronization
	mu sync.RWMutex

	// Statistics
	welford *stats.Welford
	ewma    *stats.EWMA

	// Data collection
	samplesChan chan float64
	samples     []float64

	// Progress tracking
	bytesDownloaded int64
	converged       bool
	convergenceCV   float64

	// Worker coordination
	workerDone chan struct{}
}

// NewEngine creates a new download test engine
func NewEngine(config Config) *Engine {
	if config.Threads <= 0 {
		config.Threads = 4
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.CVThreshold == 0 {
		config.CVThreshold = 0.03
	}
	if config.UpdateInterval == 0 {
		config.UpdateInterval = 100 * time.Millisecond
	}
	if config.MinSamples == 0 {
		config.MinSamples = 5
	}
	if config.TestDuration == 0 {
		config.TestDuration = 60 * time.Second
	}

	// Use background context - test duration will control overall timeout
	ctx, cancel := context.WithCancel(context.Background())

	return &Engine{
		config:      config,
		ctx:         ctx,
		cancel:      cancel,
		welford:     stats.NewWelford(),
		ewma:        stats.NewEWMA(0.2), // α = 0.2
		samplesChan: make(chan float64, config.Threads*10),
		samples:     make([]float64, 0),
		workerDone:  make(chan struct{}),
	}
}

// Run executes the download test
func (e *Engine) Run() (*Result, error) {
	startTime := time.Now()
	testCtx, testCancel := context.WithTimeout(e.ctx, e.config.TestDuration)
	defer testCancel()

	// Start collector goroutine
	go e.collector()

	// Start worker threads
	var wg sync.WaitGroup
	for i := 0; i < e.config.Threads; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			e.worker(testCtx, threadID)
		}(i)
	}

	// Monitor convergence
	ticker := time.NewTicker(e.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-testCtx.Done():
			wg.Wait()             // Wait for all workers to finish
			close(e.samplesChan)  // Signal collector to stop
			<-e.workerDone        // Wait for collector to drain and finish
			return e.buildResult(time.Since(startTime)), nil

		case <-ticker.C:
			e.mu.RLock()
			if e.converged {
				e.mu.RUnlock()
				testCancel()
				wg.Wait()             // Wait for all workers to finish
				close(e.samplesChan)  // Signal collector to stop
				<-e.workerDone        // Wait for collector to drain and finish
				return e.buildResult(time.Since(startTime)), nil
			}
			e.mu.RUnlock()
		}
	}
}

// worker downloads data from URL
func (e *Engine) worker(ctx context.Context, threadID int) {
	// Use per-request timeout that's reasonable for large downloads
	client := &http.Client{
		Timeout: 10 * time.Second, // 10s per request is plenty for 100MB
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Create request
		req, err := http.NewRequestWithContext(ctx, "GET", e.config.URL, nil)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Download with timing
		startTime := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Read to /dev/null with byte counting
		n, _ := io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		duration := time.Since(startTime).Seconds()

		if n > 0 && duration > 0 {
			atomic.AddInt64(&e.bytesDownloaded, n)

			// Calculate Mbps based on actual download time
			mbps := float64(n*8) / 1e6 / duration
			
			// Safely send sample
			select {
			case e.samplesChan <- mbps:
				// Sample sent
			case <-ctx.Done():
				return
			}
		}
	}
}

// collector processes samples and updates statistics
func (e *Engine) collector() {
	defer close(e.workerDone) // Signal completion when done

	// Collect samples until channel is closed
	for sample := range e.samplesChan {
		e.mu.Lock()
		e.welford.Add(sample)
		e.ewma.Add(sample)
		e.samples = append(e.samples, sample)

		// Check convergence after minimum samples
		if len(e.samples) >= e.config.MinSamples {
			cv := e.welford.CoefficientOfVariation()
			e.convergenceCV = cv // Always track current CV
			// Update convergence status based on current CV
			e.converged = (cv < e.config.CVThreshold && cv > 0)
		} else if len(e.samples) >= 2 {
			// Track CV even before MinSamples, but don't consider converged
			e.convergenceCV = e.welford.CoefficientOfVariation()
			e.converged = false
		}
		e.mu.Unlock()
	}
}

// buildResult creates the final result
func (e *Engine) buildResult(duration time.Duration) *Result {
	e.mu.RLock()
	defer e.mu.RUnlock()

	bytesTotal := atomic.LoadInt64(&e.bytesDownloaded)
	mbps := (float64(bytesTotal) * 8) / 1e6 / duration.Seconds()

	// Use the tracked convergence CV, or compute final CV if not set
	finalCV := e.convergenceCV
	if finalCV == 0 && len(e.samples) >= 2 {
		finalCV = e.welford.CoefficientOfVariation()
	}

	return &Result{
		FinalMbps:       mbps,
		StdDevMbps:      e.welford.SampleStdDev(),
		EWMAMbps:        e.ewma.Value(),
		MeanMbps:        e.welford.Mean(),
		MinMbps:         e.welford.Min(),
		MaxMbps:         e.welford.Max(),
		Threads:         e.config.Threads,
		Duration:        duration,
		BytesDownloaded: bytesTotal,
		ConvergenceCV:   finalCV,
		SamplesCollected: len(e.samples),
		Converged:       e.converged,
	}
}

// GetCurrentStats returns current statistics for live display
func (e *Engine) GetCurrentStats() (mean, ewma, stddev float64, converged bool, cv float64) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return current CV even if not stored yet
	currentCV := e.convergenceCV
	if currentCV == 0 && len(e.samples) >= 2 {
		currentCV = e.welford.CoefficientOfVariation()
	}

	return e.welford.Mean(), e.ewma.Value(), e.welford.SampleStdDev(), e.converged, currentCV
}

// GetBytesDownloaded returns total bytes downloaded
func (e *Engine) GetBytesDownloaded() int64 {
	return atomic.LoadInt64(&e.bytesDownloaded)
}

// GetSampleCount returns current number of samples collected
func (e *Engine) GetSampleCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.samples)
}
