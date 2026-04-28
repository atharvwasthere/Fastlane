package live

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// Stats holds real-time test statistics
type Stats struct {
	LatencyMS       float64
	DownloadMbps    float64
	UploadMbps      float64
	LossPercent     float64
	PacketsSent     int64
	PacketsReceived int64
	Jitter          float64
	UpdatedAt       time.Time
}

// UI manages the live mode display
type UI struct {
	stats         Stats
	width         int
	height        int
	supportsANSI  bool
	firstRender   bool
	lastRenderAt  time.Time
}

// NewUI creates a new live mode UI
func NewUI() *UI {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width == 0 {
		width = 80
		height = 24
	}

	return &UI{
		stats:        Stats{},
		width:        width,
		height:       height,
		supportsANSI: supportsANSI(),
		firstRender:  true,
	}
}

// supportsANSI checks if terminal supports ANSI escape codes
func supportsANSI() bool {
	// Windows CMD historically doesn't support ANSI (pre-Windows 10)
	// Modern Windows Terminal and ConEmu do support it
	if runtime.GOOS == "windows" {
		// Check if running in Windows Terminal or ConEmu
		if os.Getenv("WT_SESSION") != "" || os.Getenv("ConEmuANSI") == "ON" {
			return true
		}
		// Fallback to no ANSI on old Windows
		return false
	}
	return true
}

// UpdateStats updates the statistics for display
func (ui *UI) UpdateStats(stats Stats) {
	ui.stats = stats
	ui.stats.UpdatedAt = time.Now()
}

// Render draws the live UI
func (ui *UI) Render() {
	if !ui.firstRender && ui.supportsANSI {
		// Move cursor to top-left and clear screen
		fmt.Print("\033[H\033[2J")
	}
	ui.firstRender = false

	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	// Header
	fmt.Println(cyan("╔══════════════════════════════════════════════════════════════════════╗"))
	fmt.Println(cyan("║") + "           " + green("FASTLANE LIVE MODE") + " - Real-Time Network Stats            " + cyan("║"))
	fmt.Println(cyan("╚══════════════════════════════════════════════════════════════════════╝"))
	fmt.Println()

	// Latency bar
	fmt.Printf("  %s  ", cyan("LATENCY"))
	ui.renderBar(ui.stats.LatencyMS, 0, 200, 40, "ms", green, yellow, red)

	// Download bar
	fmt.Printf("  %s ", cyan("DOWNLOAD"))
	ui.renderBar(ui.stats.DownloadMbps, 0, 500, 40, "Mbps", green, yellow, red)

	// Upload bar
	fmt.Printf("  %s   ", cyan("UPLOAD"))
	ui.renderBar(ui.stats.UploadMbps, 0, 500, 40, "Mbps", green, yellow, red)

	// Packet Loss bar
	fmt.Printf("  %s ", cyan("PKT LOSS"))
	ui.renderBar(ui.stats.LossPercent, 0, 10, 40, "%", green, yellow, red)

	fmt.Println()

	// Detailed stats
	fmt.Println(cyan("  ─────────────────────────────────────────────────────────────────────"))
	fmt.Printf("  Latency: %s%-8.2f ms%s  │  Jitter: %-8.2f ms  │  Packets: %d/%d\n",
		green(""), ui.stats.LatencyMS, color.New(color.Reset).Sprint(""),
		ui.stats.Jitter, ui.stats.PacketsReceived, ui.stats.PacketsSent)
	fmt.Printf("  Download: %s%-7.2f Mbps%s │  Upload: %-7.2f Mbps │  Loss: %.2f%%\n",
		green(""), ui.stats.DownloadMbps, color.New(color.Reset).Sprint(""),
		ui.stats.UploadMbps, ui.stats.LossPercent)
	fmt.Println(cyan("  ─────────────────────────────────────────────────────────────────────"))

	// Footer
	elapsed := time.Since(ui.stats.UpdatedAt)
	if elapsed < 2*time.Second {
		fmt.Printf("\n  %s Last updated: just now\n", green("●"))
	} else {
		fmt.Printf("\n  %s Last updated: %s ago\n", yellow("●"), elapsed.Round(time.Second))
	}

	fmt.Println("\n  Press Ctrl+C to exit")

	ui.lastRenderAt = time.Now()
}

// renderBar renders a horizontal ASCII bar
func (ui *UI) renderBar(value, min, max float64, width int, unit string, greenFunc, yellowFunc, redFunc func(a ...interface{}) string) {
	// Normalize value to 0-1 range
	normalized := (value - min) / (max - min)
	if normalized < 0 {
		normalized = 0
	}
	if normalized > 1 {
		normalized = 1
	}

	filled := int(normalized * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}

	// Choose color based on metric type and value
	var colorFunc func(a ...interface{}) string
	if strings.Contains(unit, "%") || unit == "ms" {
		// Lower is better (loss, latency)
		if value < max*0.3 {
			colorFunc = greenFunc
		} else if value < max*0.6 {
			colorFunc = yellowFunc
		} else {
			colorFunc = redFunc
		}
	} else {
		// Higher is better (bandwidth)
		if value > max*0.6 {
			colorFunc = greenFunc
		} else if value > max*0.3 {
			colorFunc = yellowFunc
		} else {
			colorFunc = redFunc
		}
	}

	// Render bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Printf("[%s] %s%7.2f %s%s\n",
		colorFunc(bar),
		colorFunc(""),
		value,
		unit,
		color.New(color.Reset).Sprint(""))
}

// Clear clears the terminal
func (ui *UI) Clear() {
	if ui.supportsANSI {
		fmt.Print("\033[2J\033[H")
	} else {
		// Windows fallback: just print newlines
		for i := 0; i < ui.height; i++ {
			fmt.Println()
		}
	}
}

// UpdateTerminalSize updates the cached terminal dimensions
func (ui *UI) UpdateTerminalSize() {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil && width > 0 {
		ui.width = width
		ui.height = height
	}
}

// RenderLoop runs the render loop at specified interval
func (ui *UI) RenderLoop(interval time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial render
	ui.Render()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			ui.UpdateTerminalSize()
			ui.Render()
		}
	}
}
