package testsuite

import (
	"context"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

// Config holds the four sub-engine factories that the orchestrator will run
// sequentially. Injecting bench.Engine values (not constructors) keeps the
// dependency on cmd/internal one-way: cmd/internal builds the engines and
// hands them to testsuite, not the other way around.
type Config struct {
	Server   string
	Ping     bench.Engine
	Download bench.Engine
	Upload   bench.Engine
	Loss     bench.Engine
}

// Engine implements bench.Engine and produces an aggregate Result with
// Kind=KindTest. Each sub-result is folded into the aggregate via a small
// adapter so the renderer never needs to know about four-results-at-once.
type Engine struct {
	cfg Config
}

func NewEngine(cfg Config) *Engine { return &Engine{cfg: cfg} }

// Run executes the four sub-engines in order. A failure in any one step
// records a Note and continues to the next — partial data is still useful.
// Honors context cancellation between steps.
func (e *Engine) Run(ctx context.Context) (bench.Result, error) {
	start := time.Now()
	agg := bench.NewResult(bench.KindTest, e.cfg.Server, start, 0)

	runStep := func(label string, eng bench.Engine, fold func(bench.Result)) {
		if eng == nil || ctx.Err() != nil {
			return
		}
		r, err := eng.Run(ctx)
		if err != nil {
			agg.Notes = append(agg.Notes, label+": "+err.Error())
			return
		}
		fold(r)
	}

	runStep("ping", e.cfg.Ping, func(r bench.Result) {
		agg.Metrics["latency_ms"] = r.Metrics["latency_ms"]
		agg.Metrics["jitter_ms"] = r.Metrics["jitter_ms"]
		agg.Layers = r.Layers
		agg.Flags["ping_ok"] = true
	})
	runStep("download", e.cfg.Download, func(r bench.Result) {
		agg.Metrics["download_mbps"] = r.Metrics["final_mbps"]
		agg.Flags["download_converged"] = r.Flags["converged"]
		agg.Flags["download_ok"] = true
	})
	runStep("upload", e.cfg.Upload, func(r bench.Result) {
		agg.Metrics["upload_mbps"] = r.Metrics["final_mbps"]
		agg.Flags["upload_converged"] = r.Flags["converged"]
		agg.Flags["upload_ok"] = true
	})
	runStep("loss", e.cfg.Loss, func(r bench.Result) {
		agg.Metrics["loss_percent"] = r.Metrics["loss_percent"]
		agg.Counters["packets_sent"] = r.Counters["packets_sent"]
		agg.Counters["packets_received"] = r.Counters["packets_received"]
		agg.Flags["loss_ok"] = true
	})

	agg.Duration = time.Since(start)
	agg.Notes = append([]string{"rating=" + rate(agg)}, agg.Notes...)
	return agg, ctx.Err()
}

// rate produces a coarse EXCELLENT/GOOD/FAIR/POOR string from the aggregate.
// Thresholds picked to match common-sense home-broadband expectations.
func rate(r bench.Result) string {
	score := 0
	if v := r.Metrics["latency_ms"]; v > 0 {
		switch {
		case v < 30:
			score += 3
		case v < 80:
			score += 2
		case v < 150:
			score += 1
		}
	}
	if v := r.Metrics["download_mbps"]; v > 0 {
		switch {
		case v >= 200:
			score += 3
		case v >= 50:
			score += 2
		case v >= 10:
			score += 1
		}
	}
	if v := r.Metrics["upload_mbps"]; v > 0 {
		switch {
		case v >= 50:
			score += 3
		case v >= 10:
			score += 2
		case v >= 2:
			score += 1
		}
	}
	if v := r.Metrics["loss_percent"]; v <= 0 {
		score += 2
	} else if v < 1 {
		score += 1
	}
	switch {
	case score >= 10:
		return "EXCELLENT"
	case score >= 7:
		return "GOOD"
	case score >= 4:
		return "FAIR"
	default:
		return "POOR"
	}
}
