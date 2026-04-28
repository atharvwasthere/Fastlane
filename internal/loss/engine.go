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
	LossPercent     float64
	JitterMS        float64
	MinLatencyMS    float64
	MaxLatencyMS    float64
	AvgLatencyMS    float64
	Duration        time.Duration
	TestComplete    bool
}

// Config holds loss test configuration
type Config struct {
	Host           string        // Target host for UDP packets
	Port           int           // UDP port (default 9)
	Count          int           // Number of packets to send (default 100)
	PacketSize     int           // Size of each UDP packet (default 32 bytes)
	Rate           int           // Packets per second (default 10)
	Timeout        time.Duration // Overall test timeout (default 30s)
	EnableJitter   bool          // Calculate jitter from inter-arrival times
}

// Engine manages the UDP loss test
type Engine struct {
	config Config
	mu     sync.RWMutex
	
	// Metrics
	sent     int64
	received int64
	
	// Jitter tracking
	arrivalTimes []time.Time
	interArrivals []float64
}

// NewEngine creates a new loss test engine
func NewEngine(config Config) *Engine {
	if config.Port == 0 {
		config.Port = 9 // Discard protocol (standard for testing)
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
	
	return &Engine{
		config:        config,
		arrivalTimes:  make([]time.Time, 0, config.Count),
		interArrivals: make([]float64, 0, config.Count),
	}
}

// Run executes the UDP loss test
func (e *Engine) Run(ctx context.Context) (*Result, error) {
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
	
	// doneChan signals the receiver to stop. The receiver owns ackChan and
	// closes it on the way out, so we never close from the sender side
	// (which would race the receiver's send and panic).
	ackChan := make(chan int64, e.config.Count)
	doneChan := make(chan struct{})
	receiverDone := make(chan struct{})

	go func() {
		e.receiver(conn, ackChan, doneChan)
		close(ackChan)
		close(receiverDone)
	}()

	interval := time.Second / time.Duration(e.config.Rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	testCtx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	var seq int64
sendLoop:
	for seq = 0; seq < int64(e.config.Count); seq++ {
		select {
		case <-testCtx.Done():
			break sendLoop
		case <-ticker.C:
			packet := e.createPacket(seq)
			_, _ = conn.Write(packet)
			e.mu.Lock()
			e.sent++
			e.mu.Unlock()
		}
	}

	// Give late responses a moment to land, then signal the receiver to exit.
	// Unblocking the blocking ReadFromUDP requires bumping the deadline.
	select {
	case <-testCtx.Done():
	case <-time.After(time.Second):
	}
	close(doneChan)
	_ = conn.SetReadDeadline(time.Now())
	<-receiverDone
	for range ackChan {
		// drain
	}

	return e.buildResult(time.Since(startTime)), nil
}

// createPacket creates a UDP packet with sequence number and timestamp
func (e *Engine) createPacket(seq int64) []byte {
	packet := make([]byte, e.config.PacketSize)
	
	// First 8 bytes: sequence number
	binary.BigEndian.PutUint64(packet[0:8], uint64(seq))
	
	// Next 8 bytes: timestamp (nanoseconds)
	binary.BigEndian.PutUint64(packet[8:16], uint64(time.Now().UnixNano()))
	
	// Rest: padding (zeros)
	return packet
}

// receiver listens for echo/response packets
func (e *Engine) receiver(conn *net.UDPConn, ackChan chan<- int64, done <-chan struct{}) {
	buffer := make([]byte, 1500) // MTU size
	conn.SetReadDeadline(time.Now().Add(e.config.Timeout))
	
	for {
		select {
		case <-done:
			return
		default:
		}
		
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			// Timeout or error - normal for UDP
			continue
		}
		
		if n < 16 {
			continue // Invalid packet
		}
		
		// Extract sequence number and timestamp
		seq := int64(binary.BigEndian.Uint64(buffer[0:8]))
		_ = int64(binary.BigEndian.Uint64(buffer[8:16])) // sentTime (unused for now)
		recvTime := time.Now()
		
		e.mu.Lock()
		e.received++
		e.arrivalTimes = append(e.arrivalTimes, recvTime)
		
		// Track inter-arrival times for jitter
		if e.config.EnableJitter && len(e.arrivalTimes) > 1 {
			lastArrival := e.arrivalTimes[len(e.arrivalTimes)-2]
			interArrival := recvTime.Sub(lastArrival).Seconds() * 1000 // ms
			e.interArrivals = append(e.interArrivals, interArrival)
		}
		e.mu.Unlock()
		
		ackChan <- seq
	}
}

// buildResult creates the final result
func (e *Engine) buildResult(duration time.Duration) *Result {
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
		LossPercent:     lossPercent,
		JitterMS:        jitter,
		Duration:        duration,
		TestComplete:    true,
	}
}

// GetCurrentStats returns current statistics for live display
func (e *Engine) GetCurrentStats() (sent, received, lost int64, lossPercent float64) {
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
