package nettest

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/ui"
	"github.com/atharvwasthere/Fastlane/internal/types"
	
)


func Download(ctx context.Context , server string ,verbose bool, printer *ui.Printer) (*types.DownloadResult ,error) {
	result := &types.DownloadResult{Server: server , Success: true}
	const downloadSize = 1 * 1024 *1024 // 2 MB

	// TCP Donwload
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(server,"443"),25*time.Second)
	if err != nil { 
		// Fallback to HTTP
		return httpDownload(ctx, server, verbose ,printer ,downloadSize)
	} 
	defer conn.Close()

	//Read data with progress
	buf := make([]byte, 64*1024) // 32 KB buffer
	// not saving this data just measuring how fast it arrives
	var totalBytes int64
	progress := printer.StartProgressBar(downloadSize, "Downloading")
	outer:
	for totalBytes < downloadSize {
		select {
		case <-ctx.Done():
			// Context expired or cancelled
			result.Success = false
			result.Error = "download tomed out"
			return result, ctx.Err()
		default:
			// Safe to continue: Context is still active
			n, err := conn.Read(buf) //streaming test data into memory 
			if err != nil && err != io.EOF {
				// Fallback to HTTP
				conn.Close()
				return httpDownload(ctx, server, verbose, printer, downloadSize)
			}
			totalBytes += int64(n)
			progress.Add64(int64(n))
			if verbose {
				fmt.Printf("Read %d bytes, total: %d\n", n, totalBytes)
			}
			if err == io.EOF {
				break outer
			}
		}
	}
	progress.Finish()

	result.BytesTransferred = totalBytes
	result.Duration = time.Since(start)
	speedMbps := float64(totalBytes*8) / (1024 * 1024) / result.Duration.Seconds()
	if speedMbps < 1 {
		printer.ClearProgressBar()
		printer.PrintError("Your connection is too slow for accurate benchmarking. Try a better network.")
	}
	result.SpeedMbps = speedMbps
	return result, nil

}

func httpDownload(ctx context.Context , server string, verbose bool , printer *ui.Printer , downloadSize int64 )(*types.DownloadResult , error){
	result := &types.DownloadResult{Server: server , Success: true }
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://%s/10Mb.dat", server), nil) 
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP request creation failed: %v", err)
		return result, err
	}
		
	resp, err := client.Do(req)
	if err != nil {
		result.Success = false 
		result.Error = fmt.Sprintf("HTTP download failed: %v", err)
		return result, err
	}
	defer resp.Body.Close()

	start := time.Now()
	progress := printer.StartProgressBar(downloadSize , "Downloading (HTTP)")
	buf := make([]byte, 32*1024)
	var totalBytes int64
	outer:
	for totalBytes < downloadSize {
		select {
		case <- ctx.Done():
			result.Success = false;
			result.Error = "HTTP download timed-out"
			return result, ctx.Err()
		default:
			n, err := resp.Body.Read(buf)
			if err != nil && err != io.EOF {
				result.Success = false
				result.Error = fmt.Sprintf("HTTP read failed: %v", err)
				return result, err
			}
			totalBytes += int64(n)
			progress.Add64(int64(n))
			if verbose {
				fmt.Printf("Read %d bytes,total: %d\n", n, totalBytes)
			}
			if err == io.EOF {
				break outer
			}
		}
	}
	progress.Finish()

	result.BytesTransferred = totalBytes
	result.Duration = time.Since(start)
	result.SpeedMbps = float64(totalBytes*8) / (1024 * 1024) / result.Duration.Seconds()
	return result, nil
}