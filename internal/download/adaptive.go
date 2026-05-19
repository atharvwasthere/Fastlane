package download

import (
	"context"
	"fmt"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
	"github.com/atharvwasthere/Fastlane/internal/logging"
)

// AdaptiveConfig steers the hill-climb sweep that picks an optimal thread count.
// The sweep runs short download passes at increasing concurrency until the
// marginal throughput gain falls below MarginalGainThreshold, then re-runs the
// best concurrency for FinalRunDuration to produce the reported Result.
type AdaptiveConfig struct {
	Base                  Config        // Base download config; Threads is overridden per sweep step.
	StartThreads          int           // Sweep start (default 2)
	StepThreads           int           // Increment per step (default 2)
	MaxThreads            int           // Hard cap (default 16)
	ProbeDuration         time.Duration // Duration of each sweep probe (default 2s)
	FinalRunDuration      time.Duration // Final run at best threads (default 6s)
	MarginalGainThreshold float64       // Stop when (new-prev)/prev < this (default 0.05 = 5%)
}

// DefaultAdaptive returns sensible defaults built on top of the supplied base.
func DefaultAdaptive(base Config) AdaptiveConfig {
	return AdaptiveConfig{
		Base:                  base,
		StartThreads:          2,
		StepThreads:           2,
		MaxThreads:            16,
		ProbeDuration:         2 * time.Second,
		FinalRunDuration:      6 * time.Second,
		MarginalGainThreshold: 0.05,
	}
}

// NewAdaptiveBenchEngine returns a bench.Engine that performs an adaptive
// thread-count sweep before the recorded run. The Run method returns a
// standard download bench.Result decorated with sweep telemetry in Notes:
// "adaptive: 4 threads, sweep 2→8".
func NewAdaptiveBenchEngine(cfg AdaptiveConfig) bench.Engine {
	cfg = withDefaults(cfg)
	return &adaptiveEngine{cfg: cfg}
}

type adaptiveEngine struct{ cfg AdaptiveConfig }

func withDefaults(c AdaptiveConfig) AdaptiveConfig {
	if c.StartThreads <= 0 {
		c.StartThreads = 2
	}
	if c.StepThreads <= 0 {
		c.StepThreads = 2
	}
	if c.MaxThreads <= 0 {
		c.MaxThreads = 16
	}
	if c.ProbeDuration <= 0 {
		c.ProbeDuration = 2 * time.Second
	}
	if c.FinalRunDuration <= 0 {
		c.FinalRunDuration = 6 * time.Second
	}
	if c.MarginalGainThreshold <= 0 {
		c.MarginalGainThreshold = 0.05
	}
	return c
}

// Run executes the sweep, then a final run at the chosen thread count. Each
// probe respects ctx; cancellation surfaces immediately with whatever final
// result was collected (possibly empty).
func (a *adaptiveEngine) Run(ctx context.Context) (bench.Result, error) {
	log := logging.FromContext(ctx)
	best := a.cfg.StartThreads
	var prevMbps float64
	sweepEnd := a.cfg.StartThreads

	for threads := a.cfg.StartThreads; threads <= a.cfg.MaxThreads; threads += a.cfg.StepThreads {
		if ctx.Err() != nil {
			break
		}
		mbps, err := a.probe(ctx, threads, a.cfg.ProbeDuration)
		if err != nil {
			log.Debug("adaptive: probe failed", "threads", threads, "err", err)
			break
		}
		log.Debug("adaptive: probe", "threads", threads, "mbps", mbps)
		sweepEnd = threads

		if prevMbps == 0 {
			prevMbps = mbps
			best = threads
			continue
		}
		gain := (mbps - prevMbps) / prevMbps
		if gain < a.cfg.MarginalGainThreshold {
			// Marginal gain too small — previous step was optimal.
			break
		}
		prevMbps = mbps
		best = threads
	}

	log.Info("adaptive: chose threads",
		"best", best, "sweep_start", a.cfg.StartThreads, "sweep_end", sweepEnd)

	// Final recorded run at the chosen thread count.
	finalCfg := a.cfg.Base
	finalCfg.Threads = best
	finalCfg.TestDuration = a.cfg.FinalRunDuration
	if finalCfg.Timeout < a.cfg.FinalRunDuration {
		finalCfg.Timeout = a.cfg.FinalRunDuration + 2*time.Second
	}

	started := time.Now()
	res, err := NewEngine(finalCfg).Run(ctx)
	out := packResult(finalCfg, started, res)
	out.Notes = append(out.Notes,
		fmt.Sprintf("adaptive: %d threads (sweep %d→%d)", best, a.cfg.StartThreads, sweepEnd))
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	return out, err
}

// probe runs a short download burst at the given thread count and returns its
// FinalMbps. Probes are intentionally brief — the goal is rank order, not
// final numbers.
func (a *adaptiveEngine) probe(ctx context.Context, threads int, dur time.Duration) (float64, error) {
	cfg := a.cfg.Base
	cfg.Threads = threads
	cfg.TestDuration = dur
	if cfg.Timeout < dur {
		cfg.Timeout = dur + 2*time.Second
	}
	res, err := NewEngine(cfg).Run(ctx)
	if err != nil {
		return 0, err
	}
	if res == nil {
		return 0, fmt.Errorf("nil result")
	}
	return res.FinalMbps, nil
}
