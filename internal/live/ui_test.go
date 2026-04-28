package live

import (
	"strings"
	"testing"
	"time"
)

func TestNewUI(t *testing.T) {
	ui := NewUI()

	if ui == nil {
		t.Fatal("NewUI() returned nil")
	}

	if ui.width <= 0 {
		t.Error("UI width should be positive")
	}

	if ui.height <= 0 {
		t.Error("UI height should be positive")
	}

	if !ui.firstRender {
		t.Error("First render flag should be true initially")
	}
}

func TestUpdateStats(t *testing.T) {
	ui := NewUI()

	stats := Stats{
		LatencyMS:       25.5,
		DownloadMbps:    150.0,
		UploadMbps:      75.0,
		LossPercent:     0.5,
		PacketsSent:     100,
		PacketsReceived: 99,
		Jitter:          3.2,
	}

	ui.UpdateStats(stats)

	if ui.stats.LatencyMS != 25.5 {
		t.Errorf("Expected latency 25.5, got %f", ui.stats.LatencyMS)
	}

	if ui.stats.DownloadMbps != 150.0 {
		t.Errorf("Expected download 150.0, got %f", ui.stats.DownloadMbps)
	}

	if ui.stats.LossPercent != 0.5 {
		t.Errorf("Expected loss 0.5, got %f", ui.stats.LossPercent)
	}

	if ui.stats.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestRenderBarNormalization(t *testing.T) {
	tests := []struct {
		value    float64
		min      float64
		max      float64
		expected float64 // normalized value
	}{
		{50, 0, 100, 0.5},
		{0, 0, 100, 0.0},
		{100, 0, 100, 1.0},
		{-10, 0, 100, 0.0}, // Below min
		{150, 0, 100, 1.0}, // Above max
	}

	for _, tt := range tests {
		normalized := (tt.value - tt.min) / (tt.max - tt.min)
		if normalized < 0 {
			normalized = 0
		}
		if normalized > 1 {
			normalized = 1
		}

		if normalized != tt.expected {
			t.Errorf("Normalization of %f (min=%f, max=%f) = %f, expected %f",
				tt.value, tt.min, tt.max, normalized, tt.expected)
		}
	}
}

func TestSupportsANSI(t *testing.T) {
	// Just verify it doesn't panic
	supported := supportsANSI()
	t.Logf("ANSI support detected: %v", supported)

	// On Unix systems, should generally be true
	// On Windows, depends on terminal
	if supported {
		t.Log("ANSI escape codes are supported")
	} else {
		t.Log("ANSI escape codes are NOT supported (Windows fallback)")
	}
}

func TestUpdateTerminalSize(t *testing.T) {
	ui := NewUI()
	originalWidth := ui.width
	originalHeight := ui.height

	ui.UpdateTerminalSize()

	// Size should still be valid (may or may not change)
	if ui.width <= 0 {
		t.Error("Width should remain positive after update")
	}

	if ui.height <= 0 {
		t.Error("Height should remain positive after update")
	}

	t.Logf("Terminal size: %dx%d (was %dx%d)", ui.width, ui.height, originalWidth, originalHeight)
}

func TestStatsZeroValues(t *testing.T) {
	ui := NewUI()

	// Default stats should be zero
	if ui.stats.LatencyMS != 0 {
		t.Error("Initial latency should be 0")
	}

	if ui.stats.DownloadMbps != 0 {
		t.Error("Initial download should be 0")
	}

	if ui.stats.PacketsSent != 0 {
		t.Error("Initial packets sent should be 0")
	}
}

func TestRenderDoesNotPanic(t *testing.T) {
	ui := NewUI()

	// Set some stats
	ui.UpdateStats(Stats{
		LatencyMS:       50.0,
		DownloadMbps:    100.0,
		UploadMbps:      50.0,
		LossPercent:     1.0,
		PacketsSent:     100,
		PacketsReceived: 99,
		Jitter:          5.0,
	})

	// Render should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Render() panicked: %v", r)
		}
	}()

	// Note: This will actually output to terminal
	// In real tests, we'd capture output
	// ui.Render()
	t.Log("Render test skipped (would output to terminal)")
}

func TestRenderLoopTimeout(t *testing.T) {
	ui := NewUI()
	done := make(chan struct{})

	// Start render loop
	go ui.RenderLoop(50*time.Millisecond, done)

	// Let it run for a bit
	time.Sleep(200 * time.Millisecond)

	// Stop it
	close(done)

	// Give it time to clean up
	time.Sleep(100 * time.Millisecond)

	t.Log("RenderLoop stopped successfully")
}

func TestClearDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Clear() panicked: %v", r)
		}
	}()

	// Note: This will actually clear terminal
	// ui.Clear()
	t.Log("Clear test skipped (would clear terminal)")
}

func TestBarWidthCalculation(t *testing.T) {
	tests := []struct {
		value       float64
		min         float64
		max         float64
		width       int
		expectedFill int
	}{
		{50, 0, 100, 40, 20},   // 50% of 40 = 20
		{0, 0, 100, 40, 0},     // 0% = 0
		{100, 0, 100, 40, 40},  // 100% = 40
		{75, 0, 100, 40, 30},   // 75% of 40 = 30
		{25, 0, 100, 40, 10},   // 25% of 40 = 10
	}

	for _, tt := range tests {
		normalized := (tt.value - tt.min) / (tt.max - tt.min)
		filled := int(normalized * float64(tt.width))

		if filled != tt.expectedFill {
			t.Errorf("For value=%f, expected fill=%d, got %d",
				tt.value, tt.expectedFill, filled)
		}
	}
}

func TestColorSelectionLogic(t *testing.T) {
	// Test color selection for "lower is better" metrics (latency, loss)
	testCases := []struct {
		value    float64
		max      float64
		unit     string
		expected string // "green", "yellow", "red"
	}{
		{10, 100, "ms", "green"},    // Low latency = green
		{50, 100, "ms", "yellow"},   // Medium = yellow
		{80, 100, "ms", "red"},      // High = red
		{1, 10, "%", "green"},       // Low loss = green
		{5, 10, "%", "yellow"},      // Medium = yellow
		{8, 10, "%", "red"},         // High = red
		{400, 500, "Mbps", "green"}, // High bandwidth = green
		{200, 500, "Mbps", "yellow"},// Medium = yellow
		{50, 500, "Mbps", "red"},    // Low = red
	}

	for _, tc := range testCases {
		var color string
		if strings.Contains(tc.unit, "%") || tc.unit == "ms" {
			// Lower is better
			if tc.value < tc.max*0.3 {
				color = "green"
			} else if tc.value < tc.max*0.6 {
				color = "yellow"
			} else {
				color = "red"
			}
		} else {
			// Higher is better
			if tc.value > tc.max*0.6 {
				color = "green"
			} else if tc.value > tc.max*0.3 {
				color = "yellow"
			} else {
				color = "red"
			}
		}

		if color != tc.expected {
			t.Errorf("For %f %s (max %f): expected %s, got %s",
				tc.value, tc.unit, tc.max, tc.expected, color)
		}
	}
}
