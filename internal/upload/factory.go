package upload

import (
	"context"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

// NewBenchEngine wraps the concrete upload Engine into a bench.Engine.
// Mirrors internal/download/factory.go — same ctx-bridge pattern.
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

	type runResult struct {
		res *Result
		err error
	}
	done := make(chan runResult, 1)
	go func() {
		res, err := e.engine.Run()
		done <- runResult{res, err}
	}()

	select {
	case <-ctx.Done():
		e.engine.Cancel()
		out := <-done
		return packResult(e.cfg, started, out.res), ctx.Err()
	case out := <-done:
		if out.err != nil {
			return bench.Result{}, out.err
		}
		return packResult(e.cfg, started, out.res), nil
	}
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
	r.Flags["converged"] = res.Converged
	return r
}
