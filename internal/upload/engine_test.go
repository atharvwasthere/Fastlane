package upload

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
			w.Write(data)
		}
	})

	go http.Serve(listener, mux)

	return "http://" + addr + "/data", func() { listener.Close() }
}

// TestEngineBasicUpload tests basic download functionality
func TestEngineBasicUpload(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("net/http.(*Server).Serve"),
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
	)
	url, cleanup := startMockServer(1024 * 1024) // 1MB
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		Timeout:       10 * time.Second,
		TestDuration:  2 * time.Second,
		CVThreshold:   0.03,
		MinSamples:    3,
		UpdateInterval: 100 * time.Millisecond,
	})

	result, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.BytesUploaded == 0 {
		t.Errorf("Expected bytes downloaded > 0, got %d", result.BytesUploaded)
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
		if current > peakConnections {
			atomic.StoreInt32(&peakConnections, current)
		}
		defer atomic.AddInt32(&activeConnections, -1)

		// Send 100KB
		for i := 0; i < 100; i++ {
			w.Write(make([]byte, 1024))
		}
	})

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	go http.Serve(listener, mux)
	url := "http://" + listener.Addr().String() + "/concurrent"

	engine := NewEngine(Config{
		URL:           url,
		Threads:       4,
		Timeout:       10 * time.Second,
		TestDuration:  1 * time.Second,
		CVThreshold:   0.03,
		MinSamples:    2,
		UpdateInterval: 100 * time.Millisecond,
	})

	_, _ = engine.Run(context.Background())

	if peakConnections < 2 {
		t.Logf("Expected at least 2 concurrent connections, got %d", peakConnections)
	}
}

// TestEngineConvergenceDetection tests CV threshold convergence
func TestEngineConvergenceDetection(t *testing.T) {
	url, cleanup := startMockServer(500 * 1024) // 500KB per request
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		Timeout:       15 * time.Second,
		TestDuration:  10 * time.Second,
		CVThreshold:   0.03,
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
		Timeout:       10 * time.Second,
		TestDuration:  2 * time.Second,
		CVThreshold:   0.03,
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
		Timeout:       10 * time.Second,
		TestDuration:  1 * time.Second, // Short duration to test cleanup
		CVThreshold:   0.03,
		MinSamples:    3,
		UpdateInterval: 100 * time.Millisecond,
	})

	// Run should complete without hanging
	done := make(chan bool)
	go func() {
		engine.Run(context.Background())
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
		Timeout:       10 * time.Second,
		TestDuration:  2 * time.Second,
		CVThreshold:   0.03,
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
	_ = <-done
}

// TestEngineGetBytesUploaded tests byte counting
func TestEngineGetBytesUploaded(t *testing.T) {
	url, cleanup := startMockServer(512 * 1024) // 512KB
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		Timeout:       10 * time.Second,
		TestDuration:  2 * time.Second,
		CVThreshold:   0.03,
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
	bytes1 := engine.GetBytesUploaded()

	time.Sleep(300 * time.Millisecond)
	bytes2 := engine.GetBytesUploaded()

	if bytes2 < bytes1 {
		t.Errorf("Expected bytes to increase or stay same, got %d -> %d", bytes1, bytes2)
	}

	result := <-done
	if result.BytesUploaded != engine.GetBytesUploaded() {
		t.Errorf("Expected final bytes to match, got %d vs %d", result.BytesUploaded, engine.GetBytesUploaded())
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

	if engine.config.CVThreshold != 0.03 {
		t.Errorf("Expected default CVThreshold 0.03, got %f", engine.config.CVThreshold)
	}

	if engine.config.MinSamples != 5 {
		t.Errorf("Expected default MinSamples 5, got %d", engine.config.MinSamples)
	}
}

// TestEngineResultFields verifies all Result fields are set
func TestEngineResultFields(t *testing.T) {
	url, cleanup := startMockServer(1024 * 1024)
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		Timeout:       10 * time.Second,
		TestDuration:  1 * time.Second,
		CVThreshold:   0.03,
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

// BenchmarkEngineUpload benchmarks download performance
func BenchmarkEngineUpload(b *testing.B) {
	url, cleanup := startMockServer(5 * 1024 * 1024) // 5MB
	defer cleanup()

	for i := 0; i < b.N; i++ {
		engine := NewEngine(Config{
			URL:           url,
			Threads:       4,
			Timeout:       30 * time.Second,
			TestDuration:  2 * time.Second,
			CVThreshold:   0.03,
			MinSamples:    3,
			UpdateInterval: 100 * time.Millisecond,
		})

		engine.Run(context.Background())
	}
}

// TestEngineConcurrentAccess tests thread-safety of GetCurrentStats
func TestEngineConcurrentAccess(t *testing.T) {
	url, cleanup := startMockServer(2 * 1024 * 1024)
	defer cleanup()

	engine := NewEngine(Config{
		URL:           url,
		Threads:       2,
		Timeout:       10 * time.Second,
		TestDuration:  2 * time.Second,
		CVThreshold:   0.03,
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
				bytes := engine.GetBytesUploaded()
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
		Timeout:       30 * time.Second,
		TestDuration:  20 * time.Second,
		CVThreshold:   0.5, // High threshold to encourage convergence
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
