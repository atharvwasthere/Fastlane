package upload

import (
	"context"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

// NewBenchEngine wraps the concrete upload Engine into a bench.Engine.
// Mirrors internal/download/factory.go — engine is ctx-native.
func NewBenchEngine(cfg Config) bench.Engine {
	return &benchEngine{cfg: cfg, engine: NewEngine(cfg)}
}

type benchEngine struct {
	cfg    Config
	engine *Engine
}

func (e *benchEngine) Snapshot() bench.Snapshot {
	mean, ewma, stddev, converged, cv := e.engine.GetCurrentStats()
	return bench.Snapshot{
		Mean:      mean,
		EWMA:      ewma,
		StdDev:    stddev,
		CV:        cv,
		Bytes:     e.engine.GetBytesUploaded(),
		Samples:   e.engine.GetSampleCount(),
		Converged: converged,
	}
}

func (e *benchEngine) Run(ctx context.Context) (bench.Result, error) {
	started := time.Now()
	res, err := e.engine.Run(ctx)
	out := packResult(e.cfg, started, res)
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	return out, err
}

func packResult(cfg Config, started time.Time, res *Result) bench.Result {
	r := bench.NewResult(bench.KindUpload, cfg.URL, started, 0)
	if res == nil {
		return r
	}
	r.Duration = res.Duration
	r.Metrics["final_mbps"] = res.FinalMbps
	r.Metrics["mean_mbps"] = res.MeanMbps
	r.Metrics["stddev_mbps"] = res.StdDevMbps
	r.Metrics["ewma_mbps"] = res.EWMAMbps
	r.Metrics["min_mbps"] = res.MinMbps
	r.Metrics["max_mbps"] = res.MaxMbps
	r.Metrics["convergence_cv"] = res.ConvergenceCV
	r.Counters["bytes"] = res.BytesUploaded
	r.Counters["threads"] = int64(res.Threads)
	r.Counters["samples"] = int64(res.SamplesCollected)
	r.Counters["errors"] = res.Errors
	r.Flags["converged"] = res.Converged
	if res.Errors > int64(res.SamplesCollected) {
		r.Flags["degraded"] = true
	}
	return r
}
