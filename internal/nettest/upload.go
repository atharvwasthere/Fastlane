package nettest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/types"
	"github.com/atharvwasthere/Fastlane/internal/ui"
)

func Upload(ctx context.Context, server string, verbose bool, printer *ui.Printer) (*types.UploadResult, error) {
	result := &types.UploadResult{Server: server , Success: true}
	const uploadSize = 2 * 1024 *1024 // X MB

	// TCP Upload
	start := time.Now()
	conn, err := net.DialTimeout("tcp",net.JoinHostPort(server,"443"), 25*time.Second)
	if err != nil {
		// Fallback to HTTP
		return httpUpload(ctx, server ,verbose , printer, uploadSize)

	}
	defer conn.Close()

	// Writing data with progress
	data := strings.Repeat("a",32*1024) // 32KB buffer
	var totalBytes int64
	progress := printer.StartProgressBar(uploadSize, "Uploading")
	for totalBytes < uploadSize {
		select {
		case <-ctx.Done():
			result.Success = false
			result.Error ="Upload timed-out"
			return result, ctx.Err()
		default:
			n, err := conn.Write([]byte(data))
			if err != nil {
				conn.Close()
				return httpUpload(ctx,server,verbose,printer,uploadSize)
			}
			totalBytes += int64(n)
			progress.Add64(int64(n))
			if verbose {
				fmt.Printf("Wrote %d bytes, total: %d\n", n, totalBytes)
			}
		}
	}
	progress.Finish()

	result.BytesTransferred = totalBytes
	result.Duration = time.Since(start)
	speedMbps :=  float64(totalBytes*8) / (1024 * 1024) / result.Duration.Seconds()
	result.SpeedMbps = speedMbps
	if speedMbps < 1 {
		printer.PrintError("Your connection is too slow for accurate benchmarking. Try a better network.")
	}
	return result, nil
}

func httpUpload(ctx context.Context, server string , verbose bool , printer *ui.Printer, uploadSize int64) (*types.UploadResult, error) {
	result := &types.UploadResult{Server: server, Success: true}
	client := &http.Client{}
	data := strings.Repeat("a", int(uploadSize))
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://%s/post", server), strings.NewReader(data))
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP request creation failed: %v", err)
		return result, err
	}

	start := time.Now()
	progress := printer.StartProgressBar(uploadSize , "Uploading (HTTP)")
	resp, err := client.Do(req)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP upload failed: %v", err)
		return result, err
	}
	defer resp.Body.Close()

	// Simulating progress (HTTP client does not provide write progress)
		for i := 0; i < int(uploadSize); i += 32*1024 {
		progress.Add(32 * 1024)
		time.Sleep(100 * time.Millisecond) // Simulate upload time
	}
	progress.Finish()

	result.BytesTransferred = uploadSize
	result.Duration = time.Since(start)
	speedMbps := float64(uploadSize*8) / (1024 * 1024) / result.Duration.Seconds()
	result.SpeedMbps = speedMbps
	if speedMbps < 1 {
		printer.PrintError("Your connection is too slow for accurate benchmarking. Try a better network.")
	}
	return result, nil
}