// Package render owns Fastlane's output surface. It exposes a Renderer
// interface (Header / Live / Final / Error) plus a New(format, w, opts)
// factory that returns the right concrete renderer for a given output mode.
//
// Phase 2 only needs the type. Phase 3 brings the lipgloss-backed card,
// JSON, and Prometheus implementations.
package render

import (
	"io"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

// Format selects the output mode.
type Format string

const (
	FormatCard Format = "card"       // verdict-led terminal card (default)
	FormatJSON Format = "json"       // schema-locked structured output
	FormatProm Format = "prometheus" // Prometheus exposition format
)

// Options holds render-time settings resolved once at startup.
type Options struct {
	// Width is the terminal width discovered at process start. The card
	// renderer uses this to pick a 1-, 2-, or 3-column metric grid. Never
	// re-read mid-render.
	Width int
	// NoColor forces the ASCII profile.
	NoColor bool
	// Verbose surfaces extra detail in the card body.
	Verbose bool
}

// Renderer is the contract every output mode satisfies.
type Renderer interface {
	// Header is called once before the engine runs (title + server line).
	Header(title, server string)
	// Live is called whenever a fresh Snapshot is available. Static
	// renderers (json, prom) treat it as a no-op.
	Live(snap bench.Snapshot)
	// Final is called once with the engine's terminal Result.
	Final(r bench.Result)
	// Error is called when the engine returns a non-nil error.
	Error(err error)
}

// New is the renderer factory. Phase 3 fills in the concrete cases.
func New(format Format, w io.Writer, opts Options) Renderer {
	switch format {
	case FormatJSON:
		return newJSONRenderer(w, opts)
	case FormatProm:
		return newPromRenderer(w, opts)
	default:
		return newCardRenderer(w, opts)
	}
}

// IsCard reports whether r is the live-capable card renderer (Phase 4 picks
// the bubbletea live path only when this is true).
func IsCard(r Renderer) bool {
	_, ok := r.(*cardRenderer)
	return ok
}

// CardOptions returns the Options used to build r if it is a card renderer.
// Used by the live pump to size the bubbletea program identically.
func CardOptions(r Renderer) (Options, bool) {
	c, ok := r.(*cardRenderer)
	if !ok {
		return Options{}, false
	}
	return c.opts, true
}

// CardWriter returns the writer the card renderer prints to (used by the
// bubbletea live program so live + final share an output stream).
func CardWriter(r Renderer) (io.Writer, bool) {
	c, ok := r.(*cardRenderer)
	if !ok {
		return nil, false
	}
	return c.w, true
}
