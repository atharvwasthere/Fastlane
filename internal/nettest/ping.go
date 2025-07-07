package nettest

import (
	"fmt"
	"io"
	"math"
	"net"
	"time"

	"github.com/atharvwasthere/Fastlane/types"
)

// Ping sends multiple PING commands to a server over TCP and measures latency and jitter.
func Ping(server *types.Server) (*types.PingResult, error) {
\	const (
		pingCount    = 5           // Number of pings to send
		timeout      = 5 * time.Second // Connection timeout
		readDeadline = 2 * time.Second // Read deadline per ping
	)

	result := &types.PingResult{
		Server:  server.Host,
		Success: false,
	}

	// Establish TCP connection
	conn, err := net.DialTimeout("tcp", server.Host, timeout)
	if err != nil {
		result.Error = fmt.Sprintf("failed to connect to %s: %v", server.Host, err)
		return result, err
	}
	defer conn.Close()

	// Store RTTs for calculating average and jitter
	rtts := make([]time.Duration, 0, pingCount)

	// Send multiple pings
	for i := 0; i < pingCount; i++ {
		// Set read deadline to avoid hanging
		if err := conn.SetReadDeadline(time.Now().Add(readDeadline)); err != nil {
			result.Error = fmt.Sprintf("failed to set read deadline: %v", err)
			return result, err
		}

		// Send PING command
		start := time.Now()
		_, err = conn.Write([]byte("PING\n"))
		if err != nil {
			result.Error = fmt.Sprintf("failed to send PING: %v", err)
			return result, err
		}

		// Read response (expect PONG\n)
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			result.Error = fmt.Sprintf("failed to read PONG: %v", err)
			return result, err
		}

		// Verify response
		response := string(buffer[:n])
		if !strings.HasPrefix(response, "PONG") {
			result.Error = fmt.Sprintf("invalid response from %s: %s", server.Host, response)
			return result, fmt.Errorf("invalid PONG response")
		}

		rtt := time.Since(start)
		rtts = append(rtts, rtt)
	}

	var totalRTT time.Duration
	for _, rtt := range rtts {
		totalRTT += rtt
	}
	avgRTT := totalRTT / time.Duration(len(rtts))

	var sumSquaredDiff float64
	for _, rtt := range rtts {
		diff := float64(rtt - avgRTT)
		sumSquaredDiff += diff * diff
	}
	jitter := time.Duration(math.Sqrt(sumSquaredDiff / float64(len(rtts))))

	result.Success = true
	result.Metrics = types.Metrics{
		TotalRTT: avgRTT,
		Jitter:   jitter,
	}
	return result, nil
}