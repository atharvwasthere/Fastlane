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

// Check returns every violated threshold against the aggregate Result.
// Metric keys read here MUST match what the test engine emits:
//   download_mbps, upload_mbps, latency_ms, jitter_ms, loss_percent
func (t Thresholds) Check(r bench.Result) []Violation {
	var out []Violation
	if t.MinDownloadMbps > 0 {
		got := r.Metrics["download_mbps"]
		if got < t.MinDownloadMbps {
			out = append(out, Violation{Metric: "download_mbps", Want: fmt.Sprintf(">=%.2f", t.MinDownloadMbps), Got: got})
		}
	}
	if t.MinUploadMbps > 0 {
		got := r.Metrics["upload_mbps"]
		if got < t.MinUploadMbps {
			out = append(out, Violation{Metric: "upload_mbps", Want: fmt.Sprintf(">=%.2f", t.MinUploadMbps), Got: got})
		}
	}
	if t.MaxLatencyMS > 0 {
		got := r.Metrics["latency_ms"]
		if got > t.MaxLatencyMS {
			out = append(out, Violation{Metric: "latency_ms", Want: fmt.Sprintf("<=%.2f", t.MaxLatencyMS), Got: got})
		}
	}
	if t.MaxJitterMS > 0 {
		got := r.Metrics["jitter_ms"]
		if got > t.MaxJitterMS {
			out = append(out, Violation{Metric: "jitter_ms", Want: fmt.Sprintf("<=%.2f", t.MaxJitterMS), Got: got})
		}
	}
	if t.MaxLossPercent > 0 {
		got := r.Metrics["loss_percent"]
		if got > t.MaxLossPercent {
			out = append(out, Violation{Metric: "loss_percent", Want: fmt.Sprintf("<=%.2f", t.MaxLossPercent), Got: got})
		}
	}
	return out
}
