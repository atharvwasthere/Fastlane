package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

func stringRepeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

// Printer handles all terminal UI output
type Printer struct {
	verbose bool
	spinner *spinner.Spinner
	bar     *progressbar.ProgressBar
}

// NewPrinter creates a new UI printer
func NewPrinter(verbose bool) *Printer {
	return &Printer{verbose: verbose}
}

// PrintLogo prints the Fastlane ASCII logo
func (p *Printer) PrintLogo() {
	cyan := color.New(color.FgHiCyan).SprintFunc()
	logo := []string{
		"                                                                   ",
		"███████╗ █████╗ ███████╗████████╗██╗      █████╗ ███╗   ██╗███████╗",
		"██╔════╝██╔══██╗██╔════╝╚══██╔══╝██║     ██╔══██╗████╗  ██║██╔════╝",
		"█████╗  ███████║███████╗   ██║   ██║     ███████║██╔██╗ ██║█████╗  ",
		"██╔══╝  ██╔══██║╚════██║   ██║   ██║     ██╔══██║██║╚██╗██║██╔══╝  ",
		"██║     ██║  ██║███████║   ██║   ███████╗██║  ██║██║ ╚████║███████╗",
		"╚═╝     ╚═╝  ╚═╝╚══════╝   ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝",
		"                                                                   ",
	}
	for _, line := range logo {
		fmt.Println(cyan(line))
	}
}

// PrintTaglineBox prints the Fastlane tagline in a styled box
func (p *Printer) PrintTaglineBox() {
	magenta := color.New(color.FgHiMagenta).SprintFunc()
	tagline := "Fastlane - Benchmark your bandwidth. Beautifully."

	width := len(tagline)
	total := width + 4

	top := "╔" + stringRepeat("═", total-2) + "╗"
	bottom := "╚" + stringRepeat("═", total-2) + "╝"

	fmt.Println(magenta(top))
	fmt.Printf("%s %s %s\n", magenta("║"), tagline, magenta("║"))
	fmt.Println(magenta(bottom))
	fmt.Println()
}

// PrintBox prints a section header box
func (p *Printer) PrintBox(header string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("%s┌────────────────────────────────────────────┐\n", cyan(""))
	fmt.Printf("%s│ %-42s │\n", cyan(""), header)
	fmt.Printf("%s└────────────────────────────────────────────┘\n", cyan(""))
}

// PrintBoxFooter prints a footer box with completion message
func (p *Printer) PrintBoxFooter(message string) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s┌────────────────────────────────────────────┐\n", green(""))
	fmt.Printf("%s│ [OK] %-37s │\n", green(""), message)
	fmt.Printf("%s└────────────────────────────────────────────┘\n", green(""))
	fmt.Println()
}

// PrintDetails prints server details
func (p *Printer) PrintDetails(server, location, country, timestamp string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("%s Target Server: %s (%s, %s)\n", cyan(">>"), server, location, country)
	fmt.Printf("%s Timestamp: %s\n", cyan(">>"), timestamp)
}

// StartSpinner starts a loading spinner
func (p *Printer) StartSpinner(command, message string) {
	p.spinner = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	cyan := color.New(color.FgCyan).SprintFunc()
	p.spinner.Prefix = fmt.Sprintf("%s [%s] %s: ", cyan(">>"), command, message)
	p.spinner.Start()
}

// StopSpinner stops the spinner and clears the line
func (p *Printer) StopSpinner() {
	if p.spinner != nil {
		p.spinner.Stop()
		p.spinner = nil
		fmt.Printf("\033[1A\033[K")
	}
}

// StartProgressBar creates and returns a progress bar
func (p *Printer) StartProgressBar(max int64, description string) *progressbar.ProgressBar {
	bar := progressbar.NewOptions64(
		max,
		progressbar.OptionSetDescription("["+strings.ToUpper(description)+"]"),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
	p.bar = bar
	return p.bar
}

// FinishProgressBar completes and clears the progress bar
func (p *Printer) FinishProgressBar() {
	if p.bar != nil {
		p.bar.Finish()
		p.bar = nil
	}
}

// PrintSection prints a section header
func (p *Printer) PrintSection(title string) {
	magenta := color.New(color.FgHiMagenta).SprintFunc()
	fmt.Printf("\n%s=== %s ===\n", magenta(""), title)
}

// PrintLatencyResult prints latency test results
func (p *Printer) PrintLatencyResult(latencyMS, jitterMS, minMS, maxMS float64) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Latency Analysis:\n", green(">>"))
	fmt.Printf("    Mean:    %.2f ms\n", latencyMS)
	fmt.Printf("    Jitter:  %.2f ms\n", jitterMS)
	fmt.Printf("    Min:     %.2f ms\n", minMS)
	fmt.Printf("    Max:     %.2f ms\n", maxMS)
}

// PrintDownloadResult prints download speed results
func (p *Printer) PrintDownloadResult(mbps, stddev float64) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Download Speed:\n", green(">>"))
	fmt.Printf("    Speed:   %.2f Mbps\n", mbps)
	fmt.Printf("    Std Dev: %.2f Mbps\n", stddev)
}

// PrintUploadResult prints upload speed results
func (p *Printer) PrintUploadResult(mbps, stddev float64) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Upload Speed:\n", green(">>"))
	fmt.Printf("    Speed:   %.2f Mbps\n", mbps)
	fmt.Printf("    Std Dev: %.2f Mbps\n", stddev)
}

// PrintLossResult prints packet loss results
func (p *Printer) PrintLossResult(lossPercent float64, sent, received int64) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Packet Loss:\n", green(">>"))
	fmt.Printf("    Loss:     %.2f%%\n", lossPercent)
	fmt.Printf("    Sent:     %d packets\n", sent)
	fmt.Printf("    Received: %d packets\n", received)
}

// PrintError prints an error message
func (p *Printer) PrintError(message string) {
	red := color.New(color.FgRed).SprintFunc()
	fmt.Printf("%s Error: %s\n", red("!!"), message)
}

// PrintSuccess prints a success message
func (p *Printer) PrintSuccess(message string) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s %s\n", green("[OK]"), message)
}

// PrintInfo prints an info message
func (p *Printer) PrintInfo(message string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("%s %s\n", cyan("[*]"), message)
}

// PrintVerbose prints a verbose debug message if verbose is enabled
func (p *Printer) PrintVerbose(message string) {
	if !p.verbose {
		return
	}
	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Printf("%s %s\n", yellow("[DEBUG]"), message)
}

// PrintServerInfo prints server discovery info
func (p *Printer) PrintServerInfo(name, location, ip string, latencyMS, distanceKM float64) {
	fmt.Printf("  %-30s %-20s RTT: %6.2f ms  Distance: %7.1f km\n",
		name, location, latencyMS, distanceKM)
}
