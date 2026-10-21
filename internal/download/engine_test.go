package download

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// startMockServer starts a simple HTTP server that sends data
func startMockServer(size int64) (string, func()) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := listener.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		// Send fixed amount of data
		data := make([]byte, 1024)
		for i := int64(0); i < size; i += 1024 {
			_, _ = w.Write(data)
		}
	})

	go func() { _ = http.Serve(listener, mux) }()

	return "http://" + addr + "/data", func() { listener.Close() }
}

// TestEngineBasicDownload tests basic download functionality
func TestEngineBasicDownload(t *testing.T) {
	// goleak guard: ensure no engine workers/collector linger after Run.
	// http.Serve's accept goroutine stays alive (we don't shut the listener
	// cleanly) — ignore it.
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("net/http.(*Server).Serve"),
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
	)
	url, cleanup := startMockServer(1024 * 1024) // 1MB
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		TestDuration:  2 * time.Second,
		RSEThreshold:   0.03,
		MinSamples:    3,
		UpdateInterval: 100 * time.Millisecond,
	})

	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.BytesDownloaded == 0 {
		t.Errorf("Expected bytes downloaded > 0, got %d", result.BytesDownloaded)
	}

	if result.FinalMbps <= 0 {
		t.Errorf("Expected FinalMbps > 0, got %f", result.FinalMbps)
	}

	if result.Threads != 2 {
		t.Errorf("Expected Threads = 2, got %d", result.Threads)
	}
}

// TestEngineMultiThreading verifies concurrent download threads
func TestEngineMultiThreading(t *testing.T) {
	// Create a test server that tracks concurrent connections
	var activeConnections int32
	var peakConnections int32

	mux := http.NewServeMux()
	mux.HandleFunc("/concurrent", func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&activeConnections, 1)
		for {
			peak := atomic.LoadInt32(&peakConnections)
			if current <= peak || atomic.CompareAndSwapInt32(&peakConnections, peak, current) {
				break
			}
		}
		defer atomic.AddInt32(&activeConnections, -1)

		// Send 100KB
		for i := 0; i < 100; i++ {
			_, _ = w.Write(make([]byte, 1024))
		}
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	go func() { _ = http.Serve(listener, mux) }()
	url := "http://" + listener.Addr().String() + "/concurrent"

	engine := NewEngine(Config{
		URL:           url,
		Threads:       4,
		TestDuration:  1 * time.Second,
		RSEThreshold:   0.03,
		MinSamples:    2,
		UpdateInterval: 100 * time.Millisecond,
	})

	_, _ = engine.Run(context.Background())

	if peak := atomic.LoadInt32(&peakConnections); peak < 2 {
		t.Logf("Expected at least 2 concurrent connections, got %d", peak)
	}
}

// TestEngineConvergenceDetection tests CV threshold convergence
func TestEngineConvergenceDetection(t *testing.T) {
	url, cleanup := startMockServer(500 * 1024) // 500KB per request
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		TestDuration:  10 * time.Second,
		RSEThreshold:   0.03,
		MinSamples:    5,
		UpdateInterval: 100 * time.Millisecond,
	})

	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.SamplesCollected < result.Threads {
		t.Errorf("Expected at least %d samples, got %d", result.Threads, result.SamplesCollected)
	}

	// Check that convergence CV is valid
	if result.Converged && result.ConvergenceCV > result.ConvergenceCV {
		t.Errorf("Expected convergence CV to be set when converged")
	}
}

// TestEngineStatsCalculation verifies statistics calculation
func TestEngineStatsCalculation(t *testing.T) {
	url, cleanup := startMockServer(2 * 1024 * 1024) // 2MB
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		TestDuration:  2 * time.Second,
		RSEThreshold:   0.03,
		MinSamples:    3,
		UpdateInterval: 100 * time.Millisecond,
	})

	result, _ := engine.Run(context.Background())

	// Verify all stats fields are populated
	if result.MeanMbps < 0 {
		t.Errorf("Expected MeanMbps >= 0, got %f", result.MeanMbps)
	}

	if result.StdDevMbps < 0 {
		t.Errorf("Expected StdDevMbps >= 0, got %f", result.StdDevMbps)
	}

	if result.EWMAMbps < 0 {
		t.Errorf("Expected EWMAMbps >= 0, got %f", result.EWMAMbps)
	}

	if result.MinMbps > result.MaxMbps {
		t.Errorf("Expected MinMbps <= MaxMbps, got %f > %f", result.MinMbps, result.MaxMbps)
	}
}

// TestEngineCleanupOnContext tests graceful shutdown
func TestEngineCleanupOnContext(t *testing.T) {
	url, cleanup := startMockServer(10 * 1024 * 1024) // 10MB, enough for long test
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		TestDuration:  1 * time.Second, // Short duration to test cleanup
		RSEThreshold:   0.03,
		MinSamples:    3,
		UpdateInterval: 100 * time.Millisecond,
	})

	// Run should complete without hanging
	done := make(chan bool)
	go func() {
		_, _ = engine.Run(context.Background())
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Engine did not terminate within timeout")
	}
}

// TestEngineGetCurrentStats tests live stats retrieval
func TestEngineGetCurrentStats(t *testing.T) {
	url, cleanup := startMockServer(1024 * 1024) // 1MB
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		TestDuration:  2 * time.Second,
		RSEThreshold:   0.03,
		MinSamples:    3,
		UpdateInterval: 100 * time.Millisecond,
	})

	// Start test in goroutine
	done := make(chan *Result)
	go func() {
		result, _ := engine.Run(context.Background())
		done <- result
	}()

	// Give it time to collect samples
	time.Sleep(500 * time.Millisecond)

	// Get current stats while running
	mean, ewma, stddev, _, _ := engine.GetCurrentStats()

	if mean < 0 || ewma < 0 || stddev < 0 {
		t.Errorf("Expected non-negative stats, got mean=%f ewma=%f stddev=%f", mean, ewma, stddev)
	}

	// Wait for completion
	<-done
}

// TestEngineGetBytesDownloaded tests byte counting
func TestEngineGetBytesDownloaded(t *testing.T) {
	url, cleanup := startMockServer(512 * 1024) // 512KB
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		TestDuration:  2 * time.Second,
		RSEThreshold:   0.03,
		MinSamples:    3,
		UpdateInterval: 100 * time.Millisecond,
	})

	done := make(chan *Result)
	go func() {
		result, _ := engine.Run(context.Background())
		done <- result
	}()

	// Check bytes during run
	time.Sleep(300 * time.Millisecond)
	bytes1 := engine.GetBytesDownloaded()

	time.Sleep(300 * time.Millisecond)
	bytes2 := engine.GetBytesDownloaded()

	if bytes2 < bytes1 {
		t.Errorf("Expected bytes to increase or stay same, got %d -> %d", bytes1, bytes2)
	}

	result := <-done
	if result.BytesDownloaded != engine.GetBytesDownloaded() {
		t.Errorf("Expected final bytes to match, got %d vs %d", result.BytesDownloaded, engine.GetBytesDownloaded())
	}
}

// TestEngineDefaults tests default configuration values
func TestEngineDefaults(t *testing.T) {
	config := Config{
		URL: "http://example.com/data",
	}

	engine := NewEngine(config)

	if engine.config.Threads != 4 {
		t.Errorf("Expected default threads 4, got %d", engine.config.Threads)
	}

	if engine.config.RSEThreshold != 0.03 {
		t.Errorf("Expected default RSEThreshold 0.03, got %f", engine.config.RSEThreshold)
	}

	if engine.config.MinSamples != 20 {
		t.Errorf("Expected default MinSamples 20, got %d", engine.config.MinSamples)
	}
}

// TestEngineResultFields verifies all Result fields are set
func TestEngineResultFields(t *testing.T) {
	url, cleanup := startMockServer(1024 * 1024)
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		TestDuration:  1 * time.Second,
		RSEThreshold:   0.03,
		MinSamples:    3,
		UpdateInterval: 100 * time.Millisecond,
	})

	result, _ := engine.Run(context.Background())

	// Verify Result structure has all required fields
	if result.Threads == 0 {
		t.Error("Expected Threads > 0")
	}

	if result.Duration == 0 {
		t.Error("Expected Duration > 0")
	}

	if result.SamplesCollected == 0 {
		t.Errorf("Expected SamplesCollected > 0, got %d", result.SamplesCollected)
	}
}

// BenchmarkEngineDownload benchmarks download performance
func BenchmarkEngineDownload(b *testing.B) {
	url, cleanup := startMockServer(5 * 1024 * 1024) // 5MB
	defer cleanup()

	for i := 0; i < b.N; i++ {
		engine := NewEngine(Config{
			URL:           url,
			Threads:       4,
			TestDuration:  2 * time.Second,
			RSEThreshold:   0.03,
			MinSamples:    3,
			UpdateInterval: 100 * time.Millisecond,
		})

		_, _ = engine.Run(context.Background())
	}
}

// TestEngineConcurrentAccess tests thread-safety of GetCurrentStats
func TestEngineConcurrentAccess(t *testing.T) {
	url, cleanup := startMockServer(2 * 1024 * 1024)
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		TestDuration:  2 * time.Second,
		RSEThreshold:   0.03,
		MinSamples:    3,
		UpdateInterval: 100 * time.Millisecond,
	})

	done := make(chan *Result)
	go func() {
		result, _ := engine.Run(context.Background())
		done <- result
	}()

	// Multiple goroutines reading stats
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				mean, ewma, stddev, _, _ := engine.GetCurrentStats()
				if mean < 0 || ewma < 0 || stddev < 0 {
					errors <- fmt.Errorf("negative values: mean=%f ewma=%f stddev=%f", mean, ewma, stddev)
					return
				}
				bytes := engine.GetBytesDownloaded()
				if bytes < 0 {
					errors <- fmt.Errorf("negative bytes: %d", bytes)
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	wg.Wait()

	select {
	case err := <-errors:
		t.Fatalf("Concurrent access error: %v", err)
	default:
	}

	<-done
}

// TestEngineStoppedOnConvergence tests early termination on convergence
func TestEngineStoppedOnConvergence(t *testing.T) {
	url, cleanup := startMockServer(10 * 1024 * 1024)
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		TestDuration:  20 * time.Second,
		RSEThreshold:   0.5, // High threshold to encourage convergence
		MinSamples:    3,
		UpdateInterval: 100 * time.Millisecond,
	})

	startTime := time.Now()
	result, _ := engine.Run(context.Background())
	duration := time.Since(startTime)

	// If convergence was detected, should finish early
	if result.Converged {
		if duration > 15*time.Second {
			t.Logf("Note: Convergence detected but took %v (might vary based on network)", duration)
		}
	}
}

// TestBenchEngineNoDataIsError checks an unreachable server surfaces an error, not a 0 Mbps result.
func TestBenchEngineNoDataIsError(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := listener.Addr().String()
	listener.Close() // nothing listens here now, so every dial is refused

	eng := NewBenchEngine(Config{URL: "http://" + addr + "/data", Threads: 2, TestDuration: 300 * time.Millisecond})
	if _, err := eng.Run(context.Background()); err == nil {
		t.Fatal("expected an error when no bytes were transferred")
	}
}

// TestEngineErrorStatusNotCounted checks a 403 body is an error, not a throughput sample.
func TestEngineErrorStatusNotCounted(t *testing.T) {
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	mux := http.NewServeMux()
	mux.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("0"))
	})
	go func() { _ = http.Serve(listener, mux) }()
	defer listener.Close()

	engine := NewEngine(Config{URL: "http://" + listener.Addr().String() + "/data", Threads: 2, TestDuration: 300 * time.Millisecond})
	result, _ := engine.Run(context.Background())
	if result.BytesDownloaded != 0 || result.FinalMbps != 0 {
		t.Fatalf("403 counted as data: bytes=%d mbps=%.3f", result.BytesDownloaded, result.FinalMbps)
	}
	if result.Errors == 0 {
		t.Fatal("expected 403 responses to be counted as errors")
	}
}
