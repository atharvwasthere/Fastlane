package nettest

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
	"github.com/atharvwasthere/Fastlane/internal/types"

)



func Ping(ctx context.Context, server string, verbose bool) (*types.PingResult, error) {
	// using a ptr so as to apply direct changes to returning result
	// pre defining the values
	result := &types.PingResult{Server: server, Success: true}

	// 1. DNS Resolution
	start := time.Now()
	resolver := &net.Resolver{}
	addrs, err := resolver.LookupHost(ctx, server)
	if err != nil {
		result.Success = false
		//fmt.Sprintf returns a string, it doesn't print to the console.
		result.Error = fmt.Sprintf("DNS resolution failed: %v", err)
		return result, err //return partical result with error
	}
	result.DNSResolution = time.Since(start)
	if verbose {
		fmt.Printf("DNS resolved: %v (%s)\n", addrs, result.DNSResolution)
	}

	addr := addrs[0]
	if !net.ParseIP(addr).IsLoopback() { //loopback ip states that it is running locally
		addr = net.JoinHostPort(addr, "443")
	}

	// 2. TCP Handshake
	start = time.Now()
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("TCP handshake failed: %v", err)
		return result, err
	}
	defer conn.Close()
	result.TCPHandshake = time.Since(start)
	if verbose {
		fmt.Printf("TCP handshake completed: %s\n", result.TCPHandshake)
	}

	// 3. TLS Setup
   // encryption	
   start = time.Now()
   tlsConn := tls.Client(conn, &tls.Config{ServerName: server})
   if err := tlsConn.HandshakeContext(ctx); err != nil {
	result.Success = false
	result.Error = fmt.Sprintf("TLS setup failed: %v", err)
	return result, err
   }
   defer tlsConn.Close()
   result.TLSSetup = time.Since(start)
   if verbose {
		fmt.Printf("TLS setup Completed: %s\n", result.TLSSetup)
   }

   // 4. HTTP TTFB
   start = time.Now()
   client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn ,error) {
				return net.DialTimeout("tcp", addr, 3*time.Second)
			},
			TLSClientConfig: &tls.Config{ServerName: server},
		},
   }

   req, err := http.NewRequestWithContext(ctx,"GET", fmt.Sprintf("https://%s", server), nil)
   if err != nil {
	result.Success = false
	result.Error = fmt.Sprintf("HTTP request creation failed: %v", err)
	return result, err
   }
   resp, err := client.Do(req)
   if err != nil {
	result.Success = false
	result.Error = fmt.Sprintf("HTTP request failed: %v", err)
	return result, err
   }
   defer resp.Body.Close()
   result.TTFB = time.Since(start)
   if verbose {
	fmt.Printf("HTTP TTFB: %s\n", result.TTFB)
   }

   // Total RTT
   result.TotalRTT = result.DNSResolution + result.TCPHandshake + result.TLSSetup + result.TTFB 
   return result, nil
}
