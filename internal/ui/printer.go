package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/types"
	"github.com/atharvwasthere/Fastlane/internal/utils"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
	"github.com/schollz/progressbar/v3"
)

func stringRepeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

type Printer struct {
	verbose bool
	spinner *spinner.Spinner
	bar     *progressbar.ProgressBar
}

func NewPrinter(verbose bool) *Printer {
	return &Printer{verbose: verbose}
}

func (p *Printer) PrintLogo() {
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
		fmt.Println(line)
	}
}
func (p *Printer) PrintLogoBorder() {
	logo := []string{
		"                                                                     ",
		"╔───────────────────────────────────────────────────────────────────╗",
		"│███████╗ █████╗ ███████╗████████╗██╗      █████╗ ███╗   ██╗███████╗│",
		"│██╔════╝██╔══██╗██╔════╝╚══██╔══╝██║     ██╔══██╗████╗  ██║██╔════╝│",
		"│█████╗  ███████║███████╗   ██║   ██║     ███████║██╔██╗ ██║█████╗  │",
		"│██╔══╝  ██╔══██║╚════██║   ██║   ██║     ██╔══██║██║╚██╗██║██╔══╝  │",
		"│██║     ██║  ██║███████║   ██║   ███████╗██║  ██║██║ ╚████║███████╗│",
		"│╚═╝     ╚═╝  ╚═╝╚══════╝   ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝│",
		"╚───────────────────────────────────────────────────────────────────╝",
		"                                                                     ",
	}
	for _, line := range logo {
		fmt.Println(line)
	}
}

/* 


*/
func (p *Printer) PrintVersion() {
	logo := []string{
	"              ___   ____     ____ ",
	"       _   __/  /  / __ \\   / __ \\",
	"      | | / // /  / / / /  / / / /",
	"      | |/ // /_ / /_/ /_ / /_/ / ",
	"      |___//_/(_)\u005c____/(_)\u005c____/  ",
	"                                            ",
	"                                            ",
	}
	for _, line := range logo {
		fmt.Println(line)
	}
}


func (p *Printer) PrintVersionTaglineBox() {
	cyan := color.New(color.FgHiMagenta).SprintFunc()
	tagline := "Built for developers who hate lag."

	width := runewidth.StringWidth(tagline)
	total := width + 4

	top := "╔" + stringRepeat("═", total-2) + "╗"
	bottom := "╚" + stringRepeat("═", total-2) + "╝"

	fmt.Println(cyan(top))
	fmt.Printf("%s %s %s\n", cyan("║"), tagline, cyan("║"))
	fmt.Println(cyan(bottom))
}

func (p *Printer) PrintTaglineBox() {
	cyan := color.New(color.FgHiMagenta).SprintFunc()
	tagline := "Fastlane ▸ Benchmark your bandwidth. Beautifully."

	width := runewidth.StringWidth(tagline)
	total := width + 4

	top := "╔" + stringRepeat("═", total-2) + "╗"
	bottom := "╚" + stringRepeat("═", total-2) + "╝"

	fmt.Println(cyan(top))
	fmt.Printf("%s %s %s\n", cyan("║"), tagline, cyan("║"))
	fmt.Println(cyan(bottom))
}


func (p *Printer) PrintHelpSectionInline() {
	command := color.New(color.FgHiCyan).SprintFunc()     // For command names
	flag := color.New(color.FgHiYellow).SprintFunc()      // For flags
	section := color.New(color.FgHiMagenta).SprintFunc()  // For section titles

	fmt.Println()
	fmt.Println(section("Usage:"))
	fmt.Printf("  %s [command]\n", command("Fastlane"))
	fmt.Println()

	fmt.Println(section("Available Commands:"))
	fmt.Printf("  %s    %s\n", command("completion"), "Generate the autocompletion script for the specified shell")
	fmt.Printf("  %s       %s\n", command("download"), "Run a download speed test")
	fmt.Printf("  %s           %s\n", command("full"), "Run a full network test (ping + download + upload)")
	fmt.Printf("  %s          %s\n", command("help"), "Help about any command")
	fmt.Printf("  %s           %s\n", command("live"), "Continuously monitor your network in real-time")
	fmt.Printf("  %s           %s\n", command("ping"), "Measure latency to the nearest server")
	fmt.Printf("  %s         %s\n", command("report"), "Generate a test summary report")
	fmt.Printf("  %s          %s\n", command("upload"), "Run an upload speed test")
	fmt.Printf("  %s            %s\n", command("xray"), "Deep-dive into your network diagnostics")
	fmt.Println()

	fmt.Println(section("Flags:"))
	fmt.Printf("  %s     %s\n", flag("-h, --help"), "help for Fastlane")
	fmt.Printf("  %s  %s\n", flag("-t, --toggle"), "Help message for toggle")
	fmt.Println()

	fmt.Printf("Use \"%s [command] %s\" for more information about a command.\n",
		command("Fastlane"), flag("--help"))
}




func (p *Printer) PrintBox(header string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("┌────────────────────────────────────────────┐\n")
	fmt.Printf("│ %s │\n", cyan(fmt.Sprintf("%-42s", header)))
	fmt.Printf("└────────────────────────────────────────────┘\n")
}

func (p *Printer) PrintDetails(server, location, country, timestamp string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("%s Target Server: %s (%s, %s)\n", cyan("▶"), server, location, country)
	fmt.Printf("%s Timestamp: %s\n", cyan("▶"), timestamp)
}

func (p *Printer) StartSpinner(command, message string) {
	var emoji string
	switch command {
	case "PING":
		emoji = "📶"
	case "DOWNLOAD":
		emoji = "⬇️"
	case "UPLOAD":
		emoji = "⬆️"
	default:
		emoji = "🚀"
	}
	p.spinner = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	cyan := color.New(color.FgCyan).SprintFunc()
	p.spinner.Prefix = fmt.Sprintf("%s %s %s: %s ", cyan("▶"), emoji, command, message)
	p.spinner.Start()
}

func (p *Printer) StopSpinner() {
	if p.spinner != nil {
		p.spinner.Stop()
		p.spinner = nil
		fmt.Printf("\033[1A\033[K")
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

func (p *Printer) PrintUploadResult(result *types.UploadResult) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s UPLOAD:\n", green("▶"))
	fmt.Printf("  → %.1f MB in %s\n", float64(result.BytesTransferred)/(1024*1024), utils.FormatDuration(result.Duration))
	fmt.Printf("  → Speed: %.1f Mbps\n", result.SpeedMbps)
}

func (p *Printer) PrintFullResult(result *types.FullResult) {
	p.PrintLatencyBreakdown(result.Ping)
	p.PrintDownloadResult(result.Download)
	p.PrintUploadResult(result.Upload)
}

func (p *Printer) PrintReportSaved(filepath string) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Report saved at %s\n", green("✔"), filepath)
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

func getEmojiFromDescription(desc string) string {
	desc = strings.ToLower(desc)
	switch {
	case strings.Contains(desc,"download"):
		return "⬇️"
	
	case strings.Contains(desc,"upload"):
	   return "⬆️"
	default:
		return "🚀"   
	}
}

func (p *Printer) StartProgressBar(max int64, description string) *progressbar.ProgressBar {
	
	emoji := getEmojiFromDescription(description)
	bar := progressbar.NewOptions64(
		max,
		progressbar.OptionSetDescription( emoji + " " + description),
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

func (p *Printer) ClearProgressBar() {
	if p.bar != nil {
		_ = p.bar.Clear()
		p.bar = nil 
	}
}
func (p *Printer) FinishProgressBar() {
	if p.bar != nil {
		p.bar.Finish()
		p.bar = nil
	}
}