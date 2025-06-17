package ui

import (
	"fmt"
	"github.com/atharvwasthere/Fastlane/internal/nettest"
)

type Printer struct {
	verbose bool
}

func NewPrinter(verbose bool) *Printer {
	return &Printer{verbose: verbose}
}

func (p *Printer) PrintBox(header, subheader, timestamp string) {
	fmt.Printf("┌────────────────────────────────────────────┐\n")
	fmt.Printf("│ %-42s │\n", header)
	fmt.Printf("├────────────────────────────────────────────┤\n")
	fmt.Printf("│ ▶ %-40s │\n", subheader)
	fmt.Printf("│ ▶ Timestamp: %-31s │\n", timestamp)
	fmt.Printf("├────────────────────────────────────────────┤\n")
}

func (p *Printer) PrintSpinner(message string) {
	fmt.Printf("│ 📶 PING        | %-28s ⠉ │\n", message)
}

func (p *Printer) StopSpinner() {
	// Placeholder: Stop spinner animation
}

func (p *Printer) PrintLatencyBreakdown(result *nettest.PingResult) {
	fmt.Printf("│ 📶 Latency Analysis:                       │\n")
	fmt.Printf("│   → DNS Resolution:     %-15s │\n", result.DNSResolution)
	fmt.Printf("│   → TCP Handshake:      %-15s │\n", result.TCPHandshake)
	fmt.Printf("│   → TLS Setup:          %-15s │\n", result.TLSSetup)
	fmt.Printf("│   → First Byte (TTFB):  %-15s │\n", result.TTFB)
	fmt.Printf("│   → Total Ping RTT:     %-15s │\n", result.TotalRTT)
}

func (p *Printer) PrintError(message string) {
	fmt.Printf("│ ❌ Error: %-34s │\n", message)
}

func (p *Printer) PrintBoxFooter(message string) {
	fmt.Printf("├────────────────────────────────────────────┤\n")
	fmt.Printf("│ ✔ %-40s │\n", message)
	fmt.Printf("└────────────────────────────────────────────┘\n")
}