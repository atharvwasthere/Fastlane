package nettest

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/atharvwasthere/Fastlane/types"
)

// Upload performs an upload speed test to a server over multiple TCP connections.
func Upload(server *types.Server, uploadThreadCount int) (*types.TestResult, error) {
	// Configuration
	const (
		testDuration = 10 * time.Second // Test runs for 10 seconds
		chunkSize    = 1024 * 1024     // 1MB chunks
		timeout      = 5 * time.Second  // Connection timeout
		writeDeadline = 2 * time.Second // Write deadline per chunk
	)

	// Initialize result
	result := &types.TestResult{
		Server:  server.Host,
		Success: false,
	}

	// Generate random data chunk to avoid server compression
	dataChunk := make([]byte, chunkSize)
	rand.Seed(time.Now().UnixNano())
	rand.Read(dataChunk) // Fill with random bytes

	// Track total bytes sent across all threads
	var totalBytes int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstError error
	errorCount := 0

	// Start upload threads
	if uploadThreadCount <= 0 {
		uploadThreadCount = 4 // Default from ios-config.php
	}
	start := time.Now()

	for i := 0; i < uploadThreadCount; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			// Establish TCP connection
			conn, err := net.DialTimeout("tcp", server.Host, timeout)
			if err != nil {
				mu.Lock()
				if firstError == nil {
					firstError = fmt.Errorf("thread %d failed to connect to %s: %v", threadID, server.Host, err)
				}
				errorCount++
				mu.Unlock()
				return
			}
			defer conn.Close()

			// Send UPLOAD command
			if err := conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
				mu.Lock()
				if firstError == nil {
					firstError = fmt.Errorf("thread %d failed to set write deadline: %v", threadID, err)
				}
				errorCount++
				mu.Unlock()
				return
			}
			_, err = conn.Write([]byte("UPLOAD\n"))
			if err != nil {
				mu.Lock()
				if firstError == nil {
					firstError = fmt.Errorf("thread %d failed to send UPLOAD: %v", threadID, err)
				}
				errorCount++
				mu.Unlock()
				return
			}

			// Send data chunks until test duration is reached
			var threadBytes int64
			for time.Since(start) < testDuration {
				if err := conn.SetWriteDeadline(time.Now().Add(writeDeadline)); err != nil {
					mu.Lock()
					if firstError == nil {
						firstError = fmt.Errorf("thread %d failed to set write deadline: %v", threadID, err)
					}
					errorCount++
					mu.Unlock()
					return
				}
				_, err = conn.Write(dataChunk)
				if err != nil {
					mu.Lock()
					if firstError == nil {
						firstError = fmt.Errorf("thread %d failed to send data: %v", threadID, err)
					}
					errorCount++
					mu.Unlock()
					return
				}
				threadBytes += int64(chunkSize)
			}

			// Update total bytes
			mu.Lock()
			totalBytes += threadBytes
			mu.Unlock()
		}(i)
	}

	// Wait for all threads to complete
	wg.Wait()

	// Check for errors
	if errorCount == uploadThreadCount {
		if firstError == nil {
			firstError = fmt.Errorf("all upload threads failed")
		}
		result.Error = firstError.Error()
		return result, firstError
	}

	// Calculate speed
	duration := time.Since(start)
	if duration == 0 {
		duration = time.Nanosecond // Avoid division by zero
	}
	speedBps := float64(totalBytes) * 8 / duration.Seconds()

	// Populate result
	result.Success = true
	result.Metrics = types.Metrics{
		BytesTransferred: totalBytes,
		SpeedBps:         speedBps,
		Duration:         duration,
	}
	return result, nil
}