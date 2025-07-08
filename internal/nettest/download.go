package nettest

import (
	"fmt"
	"net"
	"sync"
	"time"
	"github.com/atharvwasthere/Fastlane/types"
)

func Download(server *types.Server, downloadThreadCount int) (*types.TestResult, error) {
	const (
		testDuration  = 10 * time.Second
		chunkSize     = 1024 * 1024
		timeout       = 5 * time.Second
		readDeadline  = 2 * time.Second
	)

	result := &types.TestResult{Server: server.Host}
	var totalBytes int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstError error
	errorCount := 0

	if downloadThreadCount <= 0 {
		downloadThreadCount = 4 // Default from ios-config.php
	}
	start := time.Now()

	for i := 0; i < downloadThreadCount; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", server.Host, timeout)
			if err != nil {
				mu.Lock()
				if firstError == nil {
					firstError = fmt.Errorf("thread %d failed to connect: %v", threadID, err)
				}
				errorCount++
				mu.Unlock()
				return
			}
			defer conn.Close()

			if err := conn.SetWriteDeadline(time.Now().Add(readDeadline)); err != nil {
				mu.Lock()
				if firstError == nil {
					firstError = fmt.Errorf("thread %d failed to set write deadline: %v", threadID, err)
				}
				errorCount++
				mu.Unlock()
				return
			}
			_, err = conn.Write([]byte("DOWNLOAD\n"))
			if err != nil {
				mu.Lock()
				if firstError == nil {
					firstError = fmt.Errorf("thread %d failed to send DOWNLOAD: %v", threadID, err)
				}
				errorCount++
				mu.Unlock()
				return
			}

			var threadBytes int64
			buffer := make([]byte, chunkSize)
			for time.Since(start) < testDuration {
				if err := conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
					mu.Lock()
					if firstError == nil {
						firstError = fmt.Errorf("thread %d failed to set read deadline: %v", threadID, err)
					}
					errorCount++
					mu.Unlock()
					return
				}
				n, err := conn.Read(buffer)
				if err != nil {
					mu.Lock()
					if firstError == nil {
						firstError = fmt.Errorf("thread %d failed to read data: %v", threadID, err)
					}
					errorCount++
					mu.Unlock()
					return
				}
				threadBytes += int64(n)
			}

			mu.Lock()
			totalBytes += threadBytes
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	if errorCount == downloadThreadCount {
		if firstError == nil {
			firstError = fmt.Errorf("all download threads failed")
		}
		result.Error = firstError.Error()
		return result, firstError
	}

	duration := time.Since(start)
	if duration == 0 {
		duration = time.Nanosecond
	}
	speedBps := float64(totalBytes) * 8 / duration.Seconds()

	result.Success = true
	result.Metrics = types.Metrics{
		BytesTransferred: totalBytes,
		SpeedBps:         speedBps,
		Duration:         duration,
	}
	return result, nil
}