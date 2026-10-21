package loss

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"sync"
	"time"
)

// Result holds the packet loss test results
type Result struct {
	PacketsSent     int64
	PacketsReceived int64
	PacketsLost     int64
	Errors          int64 // connection/DNS failures (not counted as loss)
	LossPercent     float64
	JitterMS        float64
	MinLatencyMS    float64
	MaxLatencyMS    float64
	AvgLatencyMS    float64
	Duration        time.Duration
	TestComplete    bool
	Notes           []string
}

// Mode selects the loss-measurement strategy.
type Mode string

const (
	ModeHTTP Mode = "http" // HEAD requests over HTTP/HTTPS (default, public-internet safe)
	ModeUDP  Mode = "udp"  // UDP echo (LAN-only, requires controlled receiver)
	ModeICMP Mode = "icmp" // reserved — not implemented yet
)

// Config holds loss test configuration. Mode picks engine; the rest are
// shared knobs (Port is UDP-only, ignored for HTTP).
type Config struct {
	Mode         Mode          // "http" (default) | "udp" | "icmp" (TODO)
	Host         string        // Target host or URL
	Port         int           // UDP port (default 7, echo, when Mode=udp)
	Count        int           // Number of probes to send (default 100)
	PacketSize   int           // UDP packet size (default 32 bytes)
	Rate         int           // Probes per second (default 10)
	Timeout      time.Duration // Overall test timeout (default 30s)
	EnableJitter bool          // Calculate jitter (UDP: inter-arrival, HTTP: RTT stddev)
}

// UDPEngine manages the UDP echo loss test (LAN-only, requires controlled
// receiver). Public Internet hosts firewall port 7/9 — use HTTPEngine for
// real-world loss measurement.
type UDPEngine struct {
	config Config
	mu     sync.RWMutex
	
	// Metrics
	sent     int64
	received int64
	errCount int64 // send/recv errors (separate from packet loss)

	// Jitter tracking
	arrivalTimes []time.Time
	interArrivals []float64
}

// NewEngine creates a new loss test engine
func NewUDPEngine(config Config) *UDPEngine {
	if config.Port == 0 {
		config.Port = 7 // Echo (RFC 862); port 9 is Discard and never replies
	}
	if config.Count == 0 {
		config.Count = 100
	}
	if config.PacketSize == 0 {
		config.PacketSize = 32
	}
	if config.Rate == 0 {
		config.Rate = 10
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	return &UDPEngine{
		config:        config,
		arrivalTimes:  make([]time.Time, 0, config.Count),
		interArrivals: make([]float64, 0, config.Count),
	}
}

// Run executes the UDP loss test
func (e *UDPEngine) Run(ctx context.Context) (*Result, error) {
	startTime := time.Now()
	
	// Create UDP connection
	addr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}
	
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial UDP: %w", err)
	}
	defer conn.Close()
	
	done := make(chan struct{})
	receiverDone := make(chan struct{})
	go func() {
		e.receiver(conn, done)
		close(receiverDone)
	}()

	interval := time.Second / time.Duration(e.config.Rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	testCtx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

sendLoop:
	for seq := int64(0); seq < int64(e.config.Count); seq++ {
		select {
		case <-testCtx.Done():
			break sendLoop
		case <-ticker.C:
			_, werr := conn.Write(e.createPacket(seq))
			e.mu.Lock()
			if werr != nil {
				e.errCount++ // never left the host: an error, not a sent probe
			} else {
				e.sent++
			}
			e.mu.Unlock()
		}
	}

	// Give late echoes a moment to land, then stop the receiver.
	select {
	case <-testCtx.Done():
	case <-time.After(time.Second):
	}
	close(done)
	_ = conn.SetReadDeadline(time.Now())
	<-receiverDone

	return e.buildResult(time.Since(startTime)), nil
}

// createPacket creates a UDP packet with sequence number and timestamp
func (e *UDPEngine) createPacket(seq int64) []byte {
	packet := make([]byte, e.config.PacketSize)
	
	// First 8 bytes: sequence number
	binary.BigEndian.PutUint64(packet[0:8], uint64(seq))
	
	// Next 8 bytes: timestamp (nanoseconds)
	binary.BigEndian.PutUint64(packet[8:16], uint64(time.Now().UnixNano()))
	
	// Rest: padding (zeros)
	return packet
}

// receiver counts each echoed sequence number once until done closes.
func (e *UDPEngine) receiver(conn *net.UDPConn, done <-chan struct{}) {
	buffer := make([]byte, 1500) // MTU size
	seen := make(map[int64]bool, e.config.Count)

	for {
		select {
		case <-done:
			return
		default:
		}

		// Short per-read deadline: a single long one would spin on instant timeouts once it passed.
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			// Timeouts are expected for UDP; anything else (e.g. ECONNREFUSED) is counted.
			if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
				e.mu.Lock()
				e.errCount++
				e.mu.Unlock()
			}
			continue
		}
		if n < 16 {
			continue // Invalid packet
		}

		seq := int64(binary.BigEndian.Uint64(buffer[0:8]))
		// Duplicates and foreign packets would push received past sent (negative loss).
		if seq < 0 || seq >= int64(e.config.Count) || seen[seq] {
			continue
		}
		seen[seq] = true
		recvTime := time.Now()

		e.mu.Lock()
		e.received++
		e.arrivalTimes = append(e.arrivalTimes, recvTime)
		if e.config.EnableJitter && len(e.arrivalTimes) > 1 {
			lastArrival := e.arrivalTimes[len(e.arrivalTimes)-2]
			e.interArrivals = append(e.interArrivals, recvTime.Sub(lastArrival).Seconds()*1000)
		}
		e.mu.Unlock()
	}
}

// buildResult creates the final result
func (e *UDPEngine) buildResult(duration time.Duration) *Result {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	sent := e.sent
	received := e.received
	lost := sent - received
	lossPercent := 0.0
	if sent > 0 {
		lossPercent = (float64(lost) / float64(sent)) * 100.0
	}
	
	// Calculate jitter (standard deviation of inter-arrival times)
	jitter := 0.0
	if e.config.EnableJitter && len(e.interArrivals) > 1 {
		// Calculate mean inter-arrival time
		sum := 0.0
		for _, t := range e.interArrivals {
			sum += t
		}
		mean := sum / float64(len(e.interArrivals))
		
		// Calculate standard deviation
		variance := 0.0
		for _, t := range e.interArrivals {
			diff := t - mean
			variance += diff * diff
		}
		variance /= float64(len(e.interArrivals))
		jitter = math.Sqrt(variance)
	}
	
	return &Result{
		PacketsSent:     sent,
		PacketsReceived: received,
		PacketsLost:     lost,
		Errors:          e.errCount,
		LossPercent:     lossPercent,
		JitterMS:        jitter,
		Duration:        duration,
		TestComplete:    true,
	}
}

// GetCurrentStats returns current statistics for live display
func (e *UDPEngine) GetCurrentStats() (sent, received, lost int64, lossPercent float64) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	sent = e.sent
	received = e.received
	lost = sent - received
	lossPercent = 0.0
	if sent > 0 {
		lossPercent = (float64(lost) / float64(sent)) * 100.0
	}
	
	return sent, received, lost, lossPercent
}
