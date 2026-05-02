package render

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/atharvwasthere/Fastlane/internal/bench"
	"github.com/atharvwasthere/Fastlane/internal/format"
	"github.com/charmbracelet/lipgloss"
)

// Color tokens — see fastlane-tui-render skill.
var (
	colorGreen  = lipgloss.AdaptiveColor{Light: "#3a7a4f", Dark: "#6db380"}
	colorYellow = lipgloss.AdaptiveColor{Light: "#7a6630", Dark: "#b8a060"}
	colorRed    = lipgloss.AdaptiveColor{Light: "#7a3030", Dark: "#b06060"}
	colorPurple = lipgloss.AdaptiveColor{Light: "#5848aa", Dark: "#8878bb"}
	colorDim    = lipgloss.AdaptiveColor{Light: "#a0a0a0", Dark: "#3a3a3a"}
	colorBorder = lipgloss.AdaptiveColor{Light: "#cccccc", Dark: "#181818"}
)

// verdict captures icon + tone for the top row of the card.
type verdict struct {
	icon  string
	color lipgloss.AdaptiveColor
	title string
}

func newCardRenderer(w io.Writer, opts Options) Renderer {
	if opts.Width <= 0 {
		opts.Width = 80
	}
	return &cardRenderer{w: w, opts: opts}
}

type cardRenderer struct {
	w     io.Writer
	opts  Options
	title string
	host  string
}

func (c *cardRenderer) Header(title, server string) {
	c.title = title
	c.host = server
}

// Live is a no-op for the static card; live progress is bubbletea's job in Phase 4.
func (c *cardRenderer) Live(_ bench.Snapshot) {}

func (c *cardRenderer) Final(r bench.Result) {
	v := classify(r)
	body := c.buildBody(r, v)
	fmt.Fprintln(c.w, body)
}

func (c *cardRenderer) Error(err error) {
	style := lipgloss.NewStyle().
		Foreground(colorRed).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 2)
	fmt.Fprintln(c.w, style.Render(fmt.Sprintf("✗ %s", err.Error())))
}

// buildBody composes verdict row, metric grid, and footer hint inside the
// rounded outer border. Width drives column count.
func (c *cardRenderer) buildBody(r bench.Result, v verdict) string {
	cols := columnsFor(c.opts.Width)
	innerW := c.opts.Width - 4 // account for border + padding
	if innerW < 30 {
		innerW = 30
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(v.color)
	subStyle := lipgloss.NewStyle().Foreground(colorDim)

	verdictLine := fmt.Sprintf("%s %s",
		titleStyle.Render(v.icon),
		titleStyle.Render(v.title),
	)
	subtitle := subStyle.Render(buildSubtitle(r))

	grid := buildGrid(r, cols, innerW)

	var hint string
	if c.opts.Width >= 60 {
		hint = subStyle.Render(buildHint(r))
	}

	header := fmt.Sprintf(" %s [%s] ", string(r.Kind), shortHost(r.Server))
	border := lipgloss.RoundedBorder()
	// lipgloss Width sets content width; the rounded border adds 2 cols, so
	// subtract those (plus padding 2*2) so the rendered box never exceeds
	// the requested terminal width.
	contentW := c.opts.Width - 2 - 4
	if contentW < 10 {
		contentW = 10
	}
	box := lipgloss.NewStyle().
		Border(border).
		BorderForeground(colorBorder).
		Padding(1, 2).
		Width(contentW)

	parts := []string{verdictLine, subtitle, "", grid}
	if hint != "" {
		parts = append(parts, "", hint)
	}
	body := strings.Join(parts, "\n")

	rendered := box.Render(body)
	return injectTitleChip(rendered, header)
}

// classify maps an engine Result to a verdict (icon + colour + title).
func classify(r bench.Result) verdict {
	switch r.Kind {
	case bench.KindPing:
		latency := r.Metrics["latency_ms"]
		switch {
		case latency == 0:
			return verdict{"!", colorYellow, "No latency data"}
		case latency < 50:
			return verdict{"✓", colorGreen, "Connection is healthy"}
		case latency < 150:
			return verdict{"!", colorYellow, "Latency is elevated"}
		default:
			return verdict{"✗", colorRed, "Latency is poor"}
		}
	case bench.KindDownload, bench.KindUpload:
		mbps := r.Metrics["final_mbps"]
		converged := r.Flags["converged"]
		switch {
		case mbps == 0:
			return verdict{"✗", colorRed, "Throughput test failed"}
		case mbps < 5:
			return verdict{"✗", colorRed, "Throughput is poor"}
		case mbps < 50:
			return verdict{"!", colorYellow, "Throughput is moderate"}
		case !converged:
			return verdict{"!", colorYellow, "Throughput did not converge"}
		default:
			c := colorGreen
			if r.Kind == bench.KindUpload {
				c = colorPurple
			}
			return verdict{"✓", c, "Throughput is healthy"}
		}
	case bench.KindLoss:
		loss := r.Metrics["loss_percent"]
		switch {
		case loss == 0:
			return verdict{"✓", colorGreen, "No packet loss detected"}
		case loss < 1:
			return verdict{"!", colorYellow, "Low packet loss"}
		case loss < 5:
			return verdict{"!", colorYellow, "Moderate packet loss"}
		default:
			return verdict{"✗", colorRed, "High packet loss"}
		}
	}
	return verdict{"·", colorDim, string(r.Kind)}
}

// buildSubtitle fills the dot-separated facts line under the verdict.
func buildSubtitle(r bench.Result) string {
	switch r.Kind {
	case bench.KindPing:
		return fmt.Sprintf("%.0f ms average · jitter %.1f ms · %d samples",
			r.Metrics["latency_ms"], r.Metrics["jitter_ms"], r.Counters["samples"])
	case bench.KindDownload, bench.KindUpload:
		state := "converged"
		if !r.Flags["converged"] {
			state = "no convergence"
		}
		return fmt.Sprintf("%s · %d threads · %s",
			format.Mbps(r.Metrics["final_mbps"]),
			r.Counters["threads"], state)
	case bench.KindLoss:
		return fmt.Sprintf("%d/%d packets · %.2f%% loss · jitter %.1f ms",
			r.Counters["packets_received"], r.Counters["packets_sent"],
			r.Metrics["loss_percent"], r.Metrics["jitter_ms"])
	}
	return ""
}

// buildHint produces the dim-text footer line. Empty string for narrow widths.
func buildHint(r bench.Result) string {
	switch r.Kind {
	case bench.KindPing:
		return fmt.Sprintf("DNS %.1fms · TCP %.1fms · TLS %.1fms · HTTP %.1fms",
			r.Layers["dns_ms"], r.Layers["tcp_ms"], r.Layers["tls_ms"], r.Layers["http_ms"])
	case bench.KindDownload, bench.KindUpload:
		return fmt.Sprintf("CV %.3f · EWMA %s · %s transferred",
			r.Metrics["convergence_cv"],
			format.Mbps(r.Metrics["ewma_mbps"]),
			format.Bytes(r.Counters["bytes"]))
	case bench.KindLoss:
		dur := r.Duration.Round(10_000_000)
		return fmt.Sprintf("test ran %s", dur)
	}
	return ""
}

// buildGrid lays out the headline metrics in cols columns.
func buildGrid(r bench.Result, cols, innerW int) string {
	cells := metricCells(r)
	if len(cells) == 0 {
		return ""
	}
	if cols > len(cells) {
		cols = len(cells)
	}
	if cols < 1 {
		cols = 1
	}

	colW := innerW / cols
	if colW < 12 {
		colW = 12
	}

	bigStyle := lipgloss.NewStyle().Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(colorDim)

	var rows []string
	for i := 0; i < len(cells); i += cols {
		end := i + cols
		if end > len(cells) {
			end = len(cells)
		}
		valLine := make([]string, 0, cols)
		labLine := make([]string, 0, cols)
		for _, cell := range cells[i:end] {
			valLine = append(valLine, padCell(bigStyle.Render(cell.value), colW))
			labLine = append(labLine, padCell(labelStyle.Render(cell.label), colW))
		}
		rows = append(rows, strings.Join(valLine, ""))
		rows = append(rows, strings.Join(labLine, ""))
		rows = append(rows, "")
	}
	if n := len(rows); n > 0 && rows[n-1] == "" {
		rows = rows[:n-1]
	}
	return strings.Join(rows, "\n")
}

type cell struct{ value, label string }

func metricCells(r bench.Result) []cell {
	switch r.Kind {
	case bench.KindPing:
		return []cell{
			{fmt.Sprintf("%.0fms", r.Metrics["latency_ms"]), "AVG LATENCY"},
			{fmt.Sprintf("%.0fms", r.Metrics["max_ms"]), "WORST CASE"},
			{fmt.Sprintf("%.0fms", r.Metrics["jitter_ms"]), "JITTER"},
		}
	case bench.KindDownload, bench.KindUpload:
		return []cell{
			{format.Mbps(r.Metrics["final_mbps"]), "FINAL"},
			{format.Mbps(r.Metrics["ewma_mbps"]), "EWMA"},
			{format.Mbps(r.Metrics["mean_mbps"]), "MEAN"},
		}
	case bench.KindLoss:
		return []cell{
			{fmt.Sprintf("%.1f%%", r.Metrics["loss_percent"]), "LOSS"},
			{fmt.Sprintf("%d", r.Counters["packets_lost"]), "DROPPED"},
			{fmt.Sprintf("%.1fms", r.Metrics["jitter_ms"]), "JITTER"},
		}
	}
	// Fallback: dump sorted Metrics keys.
	keys := make([]string, 0, len(r.Metrics))
	for k := range r.Metrics {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]cell, 0, len(keys))
	for _, k := range keys {
		out = append(out, cell{fmt.Sprintf("%.2f", r.Metrics[k]), strings.ToUpper(k)})
	}
	return out
}

func padCell(s string, w int) string {
	plain := lipgloss.Width(s)
	if plain >= w {
		return s
	}
	return s + strings.Repeat(" ", w-plain)
}

// columnsFor selects 3 / 2 / 1 columns based on width, per skill spec.
func columnsFor(width int) int {
	switch {
	case width >= 60:
		return 3
	case width >= 40:
		return 2
	default:
		return 1
	}
}

// shortHost trims https:// and any path/query so the title chip stays clean.
func shortHost(s string) string {
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	if i := strings.IndexAny(s, "/?"); i >= 0 {
		s = s[:i]
	}
	return s
}

// injectTitleChip overlays a chip into the top border at offset 2.
func injectTitleChip(box, chip string) string {
	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}
	top := lines[0]
	chipW := lipgloss.Width(chip)
	topW := lipgloss.Width(top)
	if chipW+4 >= topW {
		return box
	}
	// Splice chip at byte offset corresponding to display offset 2.
	// The top border is pure ASCII "─", so byte == rune. To be safe,
	// use rune slicing.
	runes := []rune(top)
	if len(runes) < 4+chipW {
		return box
	}
	for i := 0; i < chipW; i++ {
		runes[2+i] = []rune(chip)[i]
	}
	lines[0] = string(runes)
	return strings.Join(lines, "\n")
}
