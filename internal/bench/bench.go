// Package bench defines the engine contract every Fastlane benchmark obeys.
//
// The renderer and orchestrator depend only on these types. Concrete engines
// (ping, download, upload, loss) live in their own packages and adapt their
// internal results into bench.Result via a factory.
package bench

import (
	"context"
	"time"
)

// Kind identifies the benchmark type. Renderers MUST NOT type-switch on Kind
// to decide whether to render — only how.
type Kind string

const (
	KindPing     Kind = "ping"
	KindDownload Kind = "download"
	KindUpload   Kind = "upload"
	KindLoss     Kind = "loss"
)

// Engine is the canonical contract: take a context, return a Result or error.
type Engine interface {
	Run(ctx context.Context) (Result, error)
}

// LiveEngine is opt-in for engines that emit progress samples. Renderers that
// want a live view assert against this interface; engines without progress
// stay at the Engine level.
type LiveEngine interface {
	Engine
	Snapshot() Snapshot
}

// Result is the canonical structured benchmark output. New engines add metric
// keys to the maps — never new fields to this struct. Keeps the renderer
// generic and JSON schema stable.
type Result struct {
	Kind      Kind
	Server    string
	StartedAt time.Time
	Duration  time.Duration

	// Metrics holds floating-point measurements (latency_ms, mbps, percent...).
	Metrics map[string]float64
	// Counters holds integer tallies (samples, bytes, packets...).
	Counters map[string]int64
	// Flags holds boolean states (converged, degraded...).
	Flags map[string]bool
	// Layers holds optional protocol-layer breakdowns (dns_ms, tcp_ms...).
	Layers map[string]float64
	// Notes holds human-readable annotations surfaced by the renderer.
	Notes []string
}

// Snapshot is a single live progress reading. Bandwidth engines fill all
// fields; latency engines fill what makes sense (Mean, Samples).
type Snapshot struct {
	Mean      float64
	EWMA      float64
	StdDev    float64
	CV        float64
	Bytes     int64
	Samples   int
	Converged bool
}

// NewResult is a small constructor that pre-allocates the metric maps so
// engine adapters don't need nil-checks at every assignment.
func NewResult(kind Kind, server string, startedAt time.Time, duration time.Duration) Result {
	return Result{
		Kind:      kind,
		Server:    server,
		StartedAt: startedAt,
		Duration:  duration,
		Metrics:   make(map[string]float64),
		Counters:  make(map[string]int64),
		Flags:     make(map[string]bool),
		Layers:    make(map[string]float64),
		Notes:     []string{},
	}
}
