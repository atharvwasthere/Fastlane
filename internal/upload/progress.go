package upload

import (
	"fmt"
	"strings"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/format"
	"github.com/fatih/color"
)

// LiveProgress displays real-time download progress
type LiveProgress struct {
	width      int
	lastUpdate time.Time
}

// NewLiveProgress creates a new live progress displayer
func NewLiveProgress(width int) *LiveProgress {
	if width < 40 {
		width = 40
	}
	return &LiveProgress{
		width:      width,
		lastUpdate: time.Now(),
	}
}


// RenderFrame renders a single frame of the download progress
func (lp *LiveProgress) RenderFrame(mean, ewma, stddev, cv float64, threads, samples int, converged bool, bytesDownloaded int64) string {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	var output strings.Builder

	// Header
	output.WriteString(cyan("╔════════════════════════════════════════════════════════╗") + "\n")
	output.WriteString(cyan("║") + " UPLOAD TEST IN PROGRESS                             " + cyan("║") + "\n")

	// Current speed bar
	speedBar := lp.renderSpeedBar(ewma)
	output.WriteString(cyan("║") + " Speed:  " + speedBar + "   " + cyan("║") + "\n")

	// EWMA value
	ewmaStr := fmt.Sprintf("%-13s", format.Mbps(ewma))
	output.WriteString(cyan("║") + " EWMA:   " + green(ewmaStr) + "                                    " + cyan("║") + "\n")

	// Mean ± StdDev
	meanStr := fmt.Sprintf("%s ± %s", format.Mbps(mean), format.Mbps(stddev))
	output.WriteString(cyan("║") + " Mean:   " + fmt.Sprintf("%-44s", meanStr) + " " + cyan("║") + "\n")

	// Coefficient of Variation with status
	cvStatus := "●"
	if cv < 0.05 {
		cvStatus = green("✓")
	} else if cv < 0.1 {
		cvStatus = yellow("○")
	}
	cvLine := fmt.Sprintf("CV:     %.3f %s Convergence threshold: 0.030", cv, cvStatus)
	output.WriteString(cyan("║") + " " + fmt.Sprintf("%-53s", cvLine) + " " + cyan("║") + "\n")

	// Samples and threads
	infoLine := fmt.Sprintf("Samples: %d  │  Threads: %d  │  Data: %s", samples, threads, format.Bytes(bytesDownloaded))
	output.WriteString(cyan("║") + " " + fmt.Sprintf("%-53s", infoLine) + " " + cyan("║") + "\n")

	// Convergence status
	statusLine := "Status: Testing..."
	if converged {
		statusLine = "Status: " + green("CONVERGED")
	}
	output.WriteString(cyan("║") + " " + fmt.Sprintf("%-53s", statusLine) + " " + cyan("║") + "\n")

	// Footer
	output.WriteString(cyan("╚════════════════════════════════════════════════════════╝") + "\n")

	return output.String()
}

// renderSpeedBar creates a visual bar representation of speed
func (lp *LiveProgress) renderSpeedBar(mbps float64) string {
	// Bar width: 36 characters
	barWidth := 36
	
	// Dynamic scaling based on current speed
	maxMbps := 100.0
	if mbps > maxMbps {
		maxMbps = mbps * 1.2
	}

	filled := int((mbps / maxMbps) * float64(barWidth))
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}

	bar := "[" + strings.Repeat("=", filled) + strings.Repeat(" ", barWidth-filled) + "]"
	return bar
}

// RenderSummary renders the final summary
func (lp *LiveProgress) RenderSummary(result *Result) string {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	var output strings.Builder

	output.WriteString(fmt.Sprintf("%s ╔════════════════════════════════════════════════════════╗\n", cyan("")))
	output.WriteString(fmt.Sprintf("%s ║ UPLOAD TEST COMPLETE%-36s║\n", cyan(""), " "))

	// Final speed metrics
	output.WriteString(fmt.Sprintf("%s ║ Final Speed:    %s%-38s%s│\n", cyan(""), 
		green(""), fmt.Sprintf("%.2f Mbps", result.FinalMbps), cyan("")))

	output.WriteString(fmt.Sprintf("%s ║ EWMA:           %s%-38s%s│\n", cyan(""),
		green(""), fmt.Sprintf("%.2f Mbps", result.EWMAMbps), cyan("")))

	output.WriteString(fmt.Sprintf("%s ║ Mean:           %s%-38s%s│\n", cyan(""),
		green(""), fmt.Sprintf("%.2f Mbps", result.MeanMbps), cyan("")))

	output.WriteString(fmt.Sprintf("%s ║ Std Dev:        %s%-38s%s│\n", cyan(""),
		green(""), fmt.Sprintf("%.2f Mbps", result.StdDevMbps), cyan("")))

	output.WriteString(fmt.Sprintf("%s ║ Min/Max:        %s / %s%-28s%s│\n", cyan(""),
		fmt.Sprintf("%.2f", result.MinMbps), fmt.Sprintf("%.2f", result.MaxMbps), " ", cyan("")))

	// Statistics
	output.WriteString(fmt.Sprintf("%s ║ Duration:       %s%-38s%s│\n", cyan(""),
		"", result.Duration.Round(10*time.Millisecond).String(), cyan("")))

	output.WriteString(fmt.Sprintf("%s ║ Bytes Uploaded: %s%-32s%s│\n", cyan(""),
		"", format.Bytes(result.BytesUploaded), cyan("")))

	output.WriteString(fmt.Sprintf("%s ║ Samples:        %d  │  Threads: %d%-24s│\n", cyan(""),
		result.SamplesCollected, result.Threads, " "))

	// Convergence info
	cvStr := fmt.Sprintf("CV: %.4f", result.ConvergenceCV)
	convergenceMsg := ""
	if result.Converged {
		convergenceMsg = green(fmt.Sprintf("✓ %s (Converged)", cvStr))
	} else {
		convergenceMsg = yellow(fmt.Sprintf("○ %s (No convergence)", cvStr))
	}
	output.WriteString(fmt.Sprintf("%s ║ %s%-35s%s│\n", cyan(""), convergenceMsg, " ", cyan("")))

	output.WriteString(fmt.Sprintf("%s ╚════════════════════════════════════════════════════════╝\n", cyan("")))

	return output.String()
}
