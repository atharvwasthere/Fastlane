// Package testsuite orchestrates the four primitive bench engines in
// sequence (ping → download → upload → loss) and aggregates their results.
package testsuite

import (
	"fmt"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

// Thresholds captures the CI-mode policy. A field with value 0 is treated as
// "don't check this threshold." Only fields the user explicitly set apply.
type Thresholds struct {
	MinDownloadMbps float64
	MinUploadMbps   float64
	MaxLatencyMS    float64
	MaxJitterMS     float64
	MaxLossPercent  float64
}

// Violation describes one failed threshold. Stringified for stderr.
type Violation struct {
	Metric string
	Want   string
	Got    float64
}

func (v Violation) String() string {
	return fmt.Sprintf("%s: want %s, got %.2f", v.Metric, v.Want, v.Got)
}

type gate struct {
	metric string
	limit  float64
	min    bool // true: got must be >= limit; false: got must be <= limit
}

// gates lists the active thresholds. Metric keys MUST match what the test
// engine emits: download_mbps, upload_mbps, latency_ms, jitter_ms, loss_percent
func (t Thresholds) gates() []gate {
	all := []gate{
		{"download_mbps", t.MinDownloadMbps, true},
		{"upload_mbps", t.MinUploadMbps, true},
		{"latency_ms", t.MaxLatencyMS, false},
		{"jitter_ms", t.MaxJitterMS, false},
		{"loss_percent", t.MaxLossPercent, false},
	}
	active := all[:0]
	for _, g := range all {
		if g.limit > 0 {
			active = append(active, g)
		}
	}
	return active
}

// Check returns every violated threshold against the aggregate Result.
// Gates whose metric is absent are skipped here and reported by Missing.
func (t Thresholds) Check(r bench.Result) []Violation {
	var out []Violation
	for _, g := range t.gates() {
		got, ok := r.Metrics[g.metric]
		if !ok {
			continue
		}
		if g.min && got < g.limit {
			out = append(out, Violation{Metric: g.metric, Want: fmt.Sprintf(">=%.2f", g.limit), Got: got})
		}
		if !g.min && got > g.limit {
			out = append(out, Violation{Metric: g.metric, Want: fmt.Sprintf("<=%.2f", g.limit), Got: got})
		}
	}
	return out
}

// Missing returns the active gates whose metric the run never produced, so a
// failed step can't pass a max-gate by reading the map's zero value.
func (t Thresholds) Missing(r bench.Result) []string {
	var out []string
	for _, g := range t.gates() {
		if _, ok := r.Metrics[g.metric]; !ok {
			out = append(out, g.metric)
		}
	}
	return out
}
