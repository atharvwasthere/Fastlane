package live

import (
	"fmt"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Stats is a snapshot of the cycling runner's state. Fields are filled as
// each sub-engine completes its step; partial Stats are normal.
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

// StatsMsg carries a fresh Stats reading into the bubbletea model.
type StatsMsg Stats

// Visual tokens — mirror pkg/render/card.go so the live view feels like the
// rest of the app. Bar character is U+2501 (heavy horizontal) for a sharp,
// thin look that scales cleanly with width.
var (
	colorGreen  = lipgloss.AdaptiveColor{Light: "#3a7a4f", Dark: "#6db380"}
	colorYellow = lipgloss.AdaptiveColor{Light: "#7a6630", Dark: "#b8a060"}
	colorRed    = lipgloss.AdaptiveColor{Light: "#7a3030", Dark: "#b06060"}
	colorPurple = lipgloss.AdaptiveColor{Light: "#5848aa", Dark: "#8878bb"}
	colorDim    = lipgloss.AdaptiveColor{Light: "#a0a0a0", Dark: "#3a3a3a"}
	colorBorder = lipgloss.AdaptiveColor{Light: "#cccccc", Dark: "#181818"}
)

const barRune = '━'

// metric describes one row in the dashboard. Direction governs verdict color
// (higher-better vs lower-better); Scale is the value at which the bar reaches
// 100% width (auto-grows if a sample exceeds it).
type metric struct {
	label string
	unit  string
	color lipgloss.AdaptiveColor
	scale float64 // soft cap for bar normalization; grows if exceeded
	lower bool    // true = lower is better (latency, loss)
}

// model is the bubbletea state for the cycling live dashboard. Keeps minimal
// state — the runner pushes Stats, the renderer maps them to bars.
type model struct {
	host    string
	width   int
	started time.Time
	current Stats

	// dynamic upper bounds so bars don't permanently saturate after one spike
	maxPing  float64
	maxDown  float64
	maxUp    float64
	maxLoss  float64

	cancel func()
}

// NewProgram builds the bubbletea program. Consumers feed Stats via Send;
// ctrl+c/q calls the supplied cancel and exits.
func NewProgram(out io.Writer, host string, cancel func()) *tea.Program {
	m := model{
		host:    host,
		started: time.Now(),
		cancel:  cancel,
		// initial scales picked from common home-broadband ranges; auto-grow
		maxPing: 100,
		maxDown: 200,
		maxUp:   100,
		maxLoss: 5,
	}
	return tea.NewProgram(m,
		tea.WithOutput(out),
		tea.WithAltScreen(),
	)
}

func (m model) Init() tea.Cmd { return tick() }

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc || msg.String() == "q" {
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
	case StatsMsg:
		m.current = Stats(msg)
		// Auto-grow scales so a sample never gets stuck at the cap.
		if msg.LatencyMS > m.maxPing {
			m.maxPing = msg.LatencyMS * 1.25
		}
		if msg.DownloadMbps > m.maxDown {
			m.maxDown = msg.DownloadMbps * 1.25
		}
		if msg.UploadMbps > m.maxUp {
			m.maxUp = msg.UploadMbps * 1.25
		}
		if msg.LossPercent > m.maxLoss {
			m.maxLoss = msg.LossPercent * 1.25
		}
		return m, nil
	case tickMsg:
		return m, tick()
	}
	return m, nil
}

func (m model) View() string {
	width := m.width
	if width == 0 {
		width = 100
	}
	innerW := width - 4 // border + padding
	if innerW < 40 {
		innerW = 40
	}

	verdictStyle := lipgloss.NewStyle().Bold(true).Foreground(verdictColor(m.current))
	dimStyle := lipgloss.NewStyle().Foreground(colorDim)

	verdict := verdictIcon(m.current) + " " + verdictTitle(m.current)
	verdictLine := verdictStyle.Render(verdict)

	elapsed := time.Since(m.started).Round(time.Second)
	subtitle := dimStyle.Render(fmt.Sprintf("%s · running %s · jitter %.1f ms",
		m.host, elapsed, m.current.Jitter))

	rows := []string{
		verdictLine,
		subtitle,
		"",
		m.bar(innerW, "ping", "ms", m.current.LatencyMS, m.maxPing, colorGreen, true),
		m.bar(innerW, "download", "Mbps", m.current.DownloadMbps, m.maxDown, colorGreen, false),
		m.bar(innerW, "upload", "Mbps", m.current.UploadMbps, m.maxUp, colorPurple, false),
		m.bar(innerW, "packet loss", "%", m.current.LossPercent, m.maxLoss, colorYellow, true),
		"",
		dimStyle.Render(footerHint(m.current)),
	}

	body := strings.Join(rows, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(1, 2).
		Width(width - 4)
	return box.Render(body) + "\n"
}

// bar renders one metric row: label left, value right, then a thin
// proportional bar on the line below. Matches the static-card design.
func (m model) bar(width int, label, unit string, value, scale float64, color lipgloss.AdaptiveColor, lowerIsBetter bool) string {
	labelStyle := lipgloss.NewStyle().Foreground(colorDim)
	valueStyle := lipgloss.NewStyle().Foreground(color).Bold(true)

	var valStr string
	if value == 0 {
		valStr = labelStyle.Render(fmt.Sprintf("— %s", unit))
	} else {
		valStr = valueStyle.Render(fmt.Sprintf("%.1f %s", value, unit))
	}

	// Compose the "label … value" header line.
	header := alignRow(label, valStr, width, labelStyle)

	// Bar: filled portion proportional to value/scale, dim remainder absent
	// (just whitespace) so the bar visually shrinks rather than always filling.
	if scale <= 0 {
		scale = 1
	}
	frac := value / scale
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat(string(barRune), filled))

	return header + "\n" + bar
}

// alignRow right-aligns the value within `width` printable cells.
func alignRow(label, value string, width int, labelStyle lipgloss.Style) string {
	labelWidth := lipgloss.Width(label)
	valueWidth := lipgloss.Width(value)
	gap := width - labelWidth - valueWidth
	if gap < 1 {
		gap = 1
	}
	return labelStyle.Render(label) + strings.Repeat(" ", gap) + value
}

func verdictIcon(s Stats) string {
	switch verdictTone(s) {
	case "good":
		return "✓"
	case "warn":
		return "!"
	case "bad":
		return "✗"
	}
	return "·"
}

func verdictColor(s Stats) lipgloss.AdaptiveColor {
	switch verdictTone(s) {
	case "good":
		return colorGreen
	case "warn":
		return colorYellow
	case "bad":
		return colorRed
	}
	return colorDim
}

func verdictTitle(s Stats) string {
	switch verdictTone(s) {
	case "good":
		return "Connection looks healthy"
	case "warn":
		return "Connection is degraded"
	case "bad":
		return "Connection is poor"
	}
	return "Warming up…"
}

// verdictTone collapses the latest Stats into a coarse status. Conservative —
// any one bad metric tips to "warn" or "bad".
func verdictTone(s Stats) string {
	if s.LatencyMS == 0 && s.DownloadMbps == 0 && s.UploadMbps == 0 {
		return "neutral"
	}
	if s.LatencyMS > 200 || s.LossPercent > 5 || s.DownloadMbps < 5 && s.DownloadMbps > 0 {
		return "bad"
	}
	if s.LatencyMS > 80 || s.LossPercent > 1 || (s.DownloadMbps > 0 && s.DownloadMbps < 25) {
		return "warn"
	}
	return "good"
}

func footerHint(s Stats) string {
	if s.UpdatedAt.IsZero() {
		return "press q · ctrl+c to quit"
	}
	since := time.Since(s.UpdatedAt).Round(time.Second)
	return fmt.Sprintf("last update %s ago · press q · ctrl+c to quit", since)
}
