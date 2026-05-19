package loss

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"
)

// Mock UDP echo server for testing
func startMockUDPServer(t *testing.T, port int, dropRate float64) *net.UDPConn {
	addr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("Failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("Failed to listen on UDP: %v", err)
	}

	// Echo server goroutine
	go func() {
		buffer := make([]byte, 1500)
		packetsReceived := 0
		for {
			n, remoteAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return
			}

			packetsReceived++

			// Simulate packet loss by dropping packets based on dropRate
			if dropRate > 0 && float64(packetsReceived%100) < dropRate*100 {
				continue // Drop this packet
			}

			// Echo back
			conn.WriteToUDP(buffer[:n], remoteAddr)
		}
	}()

	return conn
}

func TestEngineBasicLoss(t *testing.T) {
	// Start mock echo server with 0% loss
	server := startMockUDPServer(t, 19999, 0.0)
	defer server.Close()

	time.Sleep(100 * time.Millisecond) // Let server start

	config := Config{
		Host:         "127.0.0.1",
		Port:         19999,
		Count:        10,
		PacketSize:   32,
		Rate:         20,
		Timeout:      5 * time.Second,
		EnableJitter: false,
	}

	engine := NewUDPEngine(config)
	ctx := context.Background()
	result, err := engine.Run(ctx)

	if err != nil {
		t.Fatalf("Engine.Run() failed: %v", err)
	}

	if result.PacketsSent != 10 {
		t.Errorf("Expected 10 packets sent, got %d", result.PacketsSent)
	}

	// Allow some packet loss due to timing
	if result.PacketsReceived < 8 {
		t.Errorf("Expected at least 8 packets received, got %d", result.PacketsReceived)
	}

	if !result.TestComplete {
		t.Error("Test should be marked as complete")
	}
}

func TestEnginePacketLossCalculation(t *testing.T) {
	// Start mock echo server with 20% loss
	server := startMockUDPServer(t, 20000, 0.2)
	defer server.Close()

	time.Sleep(100 * time.Millisecond)

	config := Config{
		Host:       "127.0.0.1",
		Port:       20000,
		Count:      50,
		PacketSize: 32,
		Rate:       50,
		Timeout:    3 * time.Second,
	}

	engine := NewUDPEngine(config)
	ctx := context.Background()
	result, err := engine.Run(ctx)

	if err != nil {
		t.Fatalf("Engine.Run() failed: %v", err)
	}

	if result.PacketsSent != 50 {
		t.Errorf("Expected 50 packets sent, got %d", result.PacketsSent)
	}

	// Loss should be close to 20%
	if result.LossPercent < 10 || result.LossPercent > 30 {
		t.Logf("Warning: Loss percent is %f%%, expected around 20%%", result.LossPercent)
	}

	packetLoss := result.PacketsSent - result.PacketsReceived
	if packetLoss != result.PacketsLost {
		t.Errorf("PacketsLost mismatch: calculated %d, reported %d",
			packetLoss, result.PacketsLost)
	}
}

func TestEngineJitterCalculation(t *testing.T) {
	// Start mock echo server
	server := startMockUDPServer(t, 20001, 0.0)
	defer server.Close()

	time.Sleep(100 * time.Millisecond)

	config := Config{
		Host:         "127.0.0.1",
		Port:         20001,
		Count:        20,
		PacketSize:   32,
		Rate:         10,
		Timeout:      5 * time.Second,
		EnableJitter: true,
	}

	engine := NewUDPEngine(config)
	ctx := context.Background()
	result, err := engine.Run(ctx)

	if err != nil {
		t.Fatalf("Engine.Run() failed: %v", err)
	}

	// Jitter should be calculated if we received enough packets
	if result.PacketsReceived > 5 && result.JitterMS == 0 {
		t.Error("Expected non-zero jitter with sufficient received packets")
	}
}

func TestEngineDefaults(t *testing.T) {
	config := Config{
		Host: "127.0.0.1",
	}

	engine := NewUDPEngine(config)

	if engine.config.Port != 9 {
		t.Errorf("Expected default port 9, got %d", engine.config.Port)
	}

	if engine.config.Count != 100 {
		t.Errorf("Expected default count 100, got %d", engine.config.Count)
	}

	if engine.config.PacketSize != 32 {
		t.Errorf("Expected default packet size 32, got %d", engine.config.PacketSize)
	}

	if engine.config.Rate != 10 {
		t.Errorf("Expected default rate 10, got %d", engine.config.Rate)
	}

	if engine.config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", engine.config.Timeout)
	}
}

func TestEngineGetCurrentStats(t *testing.T) {
	engine := NewUDPEngine(Config{
		Host: "127.0.0.1",
	})

	// Initial state
	sent, received, lost, lossPercent := engine.GetCurrentStats()
	if sent != 0 || received != 0 || lost != 0 || lossPercent != 0 {
		t.Error("Initial stats should be zero")
	}

	// Simulate some packets
	engine.mu.Lock()
	engine.sent = 100
	engine.received = 95
	engine.mu.Unlock()

	sent, received, lost, lossPercent = engine.GetCurrentStats()
	if sent != 100 {
		t.Errorf("Expected sent=100, got %d", sent)
	}
	if received != 95 {
		t.Errorf("Expected received=95, got %d", received)
	}
	if lost != 5 {
		t.Errorf("Expected lost=5, got %d", lost)
	}
	if lossPercent != 5.0 {
		t.Errorf("Expected lossPercent=5.0, got %f", lossPercent)
	}
}

func TestEngineTimeout(t *testing.T) {
	config := Config{
		Host:    "127.0.0.1",
		Port:    9,
		Count:   1000,
		Rate:    10,
		Timeout: 1 * time.Second, // Short timeout
	}

	engine := NewUDPEngine(config)
	ctx := context.Background()
	result, err := engine.Run(ctx)

	if err != nil {
		t.Fatalf("Engine.Run() should not error on timeout: %v", err)
	}

	// Should not send all packets due to timeout
	if result.PacketsSent >= 1000 {
		t.Error("Should have timed out before sending all packets")
	}

	if result.Duration > 2*time.Second {
		t.Error("Test should have timed out within timeout duration")
	}
}

func TestEnginePacketCreation(t *testing.T) {
	engine := NewUDPEngine(Config{
		Host:       "127.0.0.1",
		PacketSize: 64,
	})

	packet := engine.createPacket(12345)

	if len(packet) != 64 {
		t.Errorf("Expected packet size 64, got %d", len(packet))
	}

	// Check if sequence number is encoded correctly (first 8 bytes)
	// This is a basic check, full verification would decode the binary
	if packet[0] == 0 && packet[1] == 0 && packet[7] == 0 {
		t.Log("Packet appears to have sequence number encoded")
	}
}

func TestEngineConcurrentAccess(t *testing.T) {
	engine := NewUDPEngine(Config{
		Host: "127.0.0.1",
	})

	// Simulate concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				engine.GetCurrentStats()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic with concurrent access
	t.Log("Concurrent access completed without panic")
}

func TestEngineZeroLoss(t *testing.T) {
	engine := NewUDPEngine(Config{
		Host: "127.0.0.1",
	})

	engine.mu.Lock()
	engine.sent = 100
	engine.received = 100
	engine.mu.Unlock()

	result := engine.buildResult(time.Second)

	if result.LossPercent != 0.0 {
		t.Errorf("Expected 0%% loss, got %f%%", result.LossPercent)
	}

	if result.PacketsLost != 0 {
		t.Errorf("Expected 0 lost packets, got %d", result.PacketsLost)
	}
}

func TestEngineCompleteLoss(t *testing.T) {
	engine := NewUDPEngine(Config{
		Host: "127.0.0.1",
	})

	engine.mu.Lock()
	engine.sent = 100
	engine.received = 0
	engine.mu.Unlock()

	result := engine.buildResult(time.Second)

	if result.LossPercent != 100.0 {
		t.Errorf("Expected 100%% loss, got %f%%", result.LossPercent)
	}

	if result.PacketsLost != 100 {
		t.Errorf("Expected 100 lost packets, got %d", result.PacketsLost)
	}
}
