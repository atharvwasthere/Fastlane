package ui

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/atharvwasthere/Fastlane/internal/types"
	"github.com/atharvwasthere/Fastlane/internal/utils"
)

type Printer struct {
	verbose bool
	spinner *spinner.Spinner
	bar     *progressbar.ProgressBar
}

func NewPrinter(verbose bool) *Printer {
	return &Printer{verbose: verbose}
}

func (p *Printer) PrintBox(header string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("┌────────────────────────────────────────────┐\n")
	fmt.Printf("│ %s │\n", cyan(fmt.Sprintf("%-42s", header)))
	fmt.Printf("└────────────────────────────────────────────┘\n")
}

func (p *Printer) PrintDetails(server, location, country, timestamp string) {
	fmt.Printf("▶ Target Server: %s (%s, %s)\n", server, location, country)
	fmt.Printf("▶ Timestamp: %s\n", timestamp)
}

func (p *Printer) StartSpinner(command, message string) {
	p.spinner = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	p.spinner.Prefix = fmt.Sprintf("▶ %s: %s ", command, message)
	p.spinner.Start()
}

func (p *Printer) StopSpinner() {
	if p.spinner != nil {
		p.spinner.Stop()
		p.spinner = nil
		fmt.Printf("\033[1A\033[K") // Clear the spinner line
	}
}

func (p *Printer) PrintLatencyBreakdown(result *types.PingResult) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Latency Analysis:\n", green("▶"))
	fmt.Printf("  → DNS Resolution:     %s\n", utils.FormatDuration(result.DNSResolution))
	fmt.Printf("  → TCP Handshake:      %s\n", utils.FormatDuration(result.TCPHandshake))
	fmt.Printf("  → TLS Setup:          %s\n", utils.FormatDuration(result.TLSSetup))
	fmt.Printf("  → First Byte (TTFB):  %s\n", utils.FormatDuration(result.TTFB))
	fmt.Printf("  → Total Ping RTT:     %s\n", utils.FormatDuration(result.TotalRTT))
}

func (p *Printer) PrintDownloadResult(result *types.DownloadResult) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s DOWNLOAD:\n", green("▶"))
	fmt.Printf("  → %.1f MB in %s\n", float64(result.BytesTransferred)/(1024*1024), utils.FormatDuration(result.Duration))
	fmt.Printf("  → Speed: %.1f Mbps\n", result.SpeedMbps)
}

func (p *Printer) PrintError(message string) {
	red := color.New(color.FgRed).SprintFunc()
	fmt.Printf("%s Error: %s\n", red("❌"), message)
}

func (p *Printer) PrintBoxFooter(message string) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("┌────────────────────────────────────────────┐\n")
	fmt.Printf("│ %s %-40s │\n", green("✔"), message)
	fmt.Printf("└────────────────────────────────────────────┘\n")
}

func (p *Printer) StartProgressBar(max int64, description string) *progressbar.ProgressBar {
	p.bar = progressbar.DefaultBytes(max, description)
	return p.bar
}

func (p *Printer) FinishProgressBar() {
	if p.bar != nil {
		p.bar.Finish()
		p.bar = nil
	}
}