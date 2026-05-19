package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
	"github.com/atharvwasthere/Fastlane/internal/format"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SnapshotMsg carries a fresh engine snapshot to the live model.
type SnapshotMsg struct{ Snap bench.Snapshot }

// ResultMsg carries the terminal Result. Receipt triggers Quit.
type ResultMsg struct{ Result bench.Result }

// ErrMsg carries a fatal engine error.
type ErrMsg struct{ Err error }

// QuitMsg is sent by the parent to force a tea.Quit (e.g. on ctx cancel).
type QuitMsg struct{}

type tickMsg time.Time

// liveModel is bubbletea-driven progress for bandwidth tests.
type liveModel struct {
	title  string
	server string
	opts   Options
	kind   bench.Kind

	snap   bench.Snapshot
	final  bench.Result
	err    error
	done   bool

	startedAt time.Time
	cancel    func() // invoked on ctrl+c so engine ctx unwinds
}

// NewLiveProgram builds the bubbletea program used by RunBench when the
// engine is a bench.LiveEngine and the renderer is in card mode. Cancel
// callback fires on ctrl+c.
func NewLiveProgram(out io.Writer, title, server string, kind bench.Kind, opts Options, cancel func()) *tea.Program {
	if opts.Width <= 0 {
		opts.Width = 80
	}
	m := liveModel{
		title:     title,
		server:    server,
		opts:      opts,
		kind:      kind,
		startedAt: time.Now(),
		cancel:    cancel,
	}
	progOpts := []tea.ProgramOption{tea.WithOutput(out)}
	return tea.NewProgram(m, progOpts...)
}

func (m liveModel) Init() tea.Cmd { return tickCmd() }

func (m liveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SnapshotMsg:
		m.snap = msg.Snap
		return m, nil
	case ResultMsg:
		m.final = msg.Result
		m.done = true
		return m, tea.Quit
	case ErrMsg:
		m.err = msg.Err
		return m, tea.Quit
	case QuitMsg:
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc {
			if m.cancel != nil {
				m.cancel()
			}
			return m, nil // wait for engine to send ResultMsg/ErrMsg
		}
	case tickMsg:
		return m, tickCmd()
	}
	return m, nil
}

func (m liveModel) View() string {
	if m.err != nil {
		style := lipgloss.NewStyle().Foreground(colorRed)
		return style.Render(fmt.Sprintf("✗ %s\n", m.err.Error()))
	}
	if m.done {
		// Final view rendered by the static card renderer to keep parity.
		c := newCardRenderer(io.Discard, m.opts).(*cardRenderer)
		c.title = m.title
		c.host = m.server
		v := classify(m.final)
		return c.buildBody(m.final, v) + "\n"
	}
	return m.renderLive()
}

func (m liveModel) renderLive() string {
	innerW := m.opts.Width - 6
	if innerW < 30 {
		innerW = 30
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(colorPurple)
	dimStyle := lipgloss.NewStyle().Foreground(colorDim)
	greenStyle := lipgloss.NewStyle().Foreground(colorGreen)

	verb := "downloading"
	if m.kind == bench.KindUpload {
		verb = "uploading"
	}
	header := titleStyle.Render(fmt.Sprintf("◆ %s · %s", verb, shortHost(m.server)))

	bar := renderSpeedBar(m.snap.EWMA, 36)
	speedLine := fmt.Sprintf("%s  %s", bar, greenStyle.Render(format.Mbps(m.snap.EWMA)))

	meanLine := dimStyle.Render(fmt.Sprintf("mean %s · stddev %s",
		format.Mbps(m.snap.Mean), format.Mbps(m.snap.StdDev)))

	cvIcon := "○"
	cvStyle := dimStyle
	switch {
	case m.snap.Converged:
		cvIcon = "✓"
		cvStyle = greenStyle
	case m.snap.CV > 0 && m.snap.CV < 0.1:
		cvIcon = "○"
		cvStyle = lipgloss.NewStyle().Foreground(colorYellow)
	}
	cvLine := cvStyle.Render(fmt.Sprintf("%s CV %.3f", cvIcon, m.snap.CV))

	elapsed := time.Since(m.startedAt).Round(100 * time.Millisecond)
	infoLine := dimStyle.Render(fmt.Sprintf("samples %d · %s · %s",
		m.snap.Samples, format.Bytes(m.snap.Bytes), elapsed))

	body := strings.Join([]string{header, "", speedLine, meanLine, cvLine, infoLine}, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(1, 2).
		Width(innerW + 2)

	chip := fmt.Sprintf(" %s [%s] ", string(m.kind), shortHost(m.server))
	return injectTitleChip(box.Render(body), chip) + "\n"
}

// renderSpeedBar draws a [====    ] bar scaled dynamically.
func renderSpeedBar(mbps float64, width int) string {
	maxMbps := 100.0
	if mbps > maxMbps {
		maxMbps = mbps * 1.2
	}
	filled := int((mbps / maxMbps) * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	full := lipgloss.NewStyle().Foreground(colorGreen).Render(strings.Repeat("█", filled))
	empty := lipgloss.NewStyle().Foreground(colorDim).Render(strings.Repeat("░", width-filled))
	return "[" + full + empty + "]"
}

func tickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
