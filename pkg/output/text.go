package output

import (
	"fmt"
	"io"
)

// TextWriter formats and writes test results as human-readable text
type TextWriter struct {
	writer  io.Writer
	verbose bool
}

// NewTextWriter creates a new text output writer
func NewTextWriter(w io.Writer, verbose bool) *TextWriter {
	return &TextWriter{
		writer:  w,
		verbose: verbose,
	}
}

// WritePingResult writes ping test results in human-readable format
func (t *TextWriter) WritePingResult(latencyMS float64, jitterMS float64, minMS, maxMS float64) error {
	output := fmt.Sprintf(
		"  Latency:  %.2f ms\n  Jitter:   %.2f ms\n  Min:      %.2f ms\n  Max:      %.2f ms\n",
		latencyMS, jitterMS, minMS, maxMS,
	)
	_, err := io.WriteString(t.writer, output)
	return err
}

// WriteDownloadResult writes download test results in human-readable format
func (t *TextWriter) WriteDownloadResult(mbps float64, stddev float64) error {
	output := fmt.Sprintf(
		"  Download: %.2f Mbps (stddev: %.2f)\n",
		mbps, stddev,
	)
	_, err := io.WriteString(t.writer, output)
	return err
}

// WriteUploadResult writes upload test results in human-readable format
func (t *TextWriter) WriteUploadResult(mbps float64, stddev float64) error {
	output := fmt.Sprintf(
		"  Upload:   %.2f Mbps (stddev: %.2f)\n",
		mbps, stddev,
	)
	_, err := io.WriteString(t.writer, output)
	return err
}

// WriteLossResult writes packet loss test results in human-readable format
func (t *TextWriter) WriteLossResult(lossPercent float64, sent, received int64) error {
	output := fmt.Sprintf(
		"  Loss:     %.2f%% (%d sent, %d received)\n",
		lossPercent, sent, received,
	)
	_, err := io.WriteString(t.writer, output)
	return err
}

// WriteSection writes a section header
func (t *TextWriter) WriteSection(title string) error {
	output := fmt.Sprintf("\n=== %s ===\n", title)
	_, err := io.WriteString(t.writer, output)
	return err
}

// WriteVerbose writes a verbose message if verbose flag is set
func (t *TextWriter) WriteVerbose(msg string) error {
	if !t.verbose {
		return nil
	}
	output := fmt.Sprintf("[DEBUG] %s\n", msg)
	_, err := io.WriteString(t.writer, output)
	return err
}
