package download

import (
	"context"
	"fmt"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
	"github.com/atharvwasthere/Fastlane/internal/logging"
)

// AdaptiveConfig steers the hill-climb sweep that picks an optimal thread count.
// The sweep doubles concurrency from StartThreads, keeps the best count seen,
// and stops after Patience probes in a row fail to beat it by
// MarginalGainThreshold; then it re-runs the best count for FinalRunDuration.
type AdaptiveConfig struct {
	Base                  Config        // Base download config; Threads is overridden per sweep step.
	StartThreads          int           // Sweep start (default 1)
	MaxThreads            int           // Hard cap (default 16)
	Patience              int           // Non-improving probes in a row before stopping (default 2)
	ProbeDuration         time.Duration // Duration of each sweep probe (default 2s)
	FinalRunDuration      time.Duration // Final run at best threads (default 6s)
	MarginalGainThreshold float64       // A probe improves only if (new-best)/best >= this (default 0.05 = 5%)
}

// DefaultAdaptive returns sensible defaults built on top of the supplied base.
func DefaultAdaptive(base Config) AdaptiveConfig {
	return AdaptiveConfig{
		Base:                  base,
		StartThreads:          1,
		MaxThreads:            16,
		Patience:              2,
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
		c.StartThreads = 1
	}
	if c.MaxThreads <= 0 {
		c.MaxThreads = 16
	}
	if c.Patience <= 0 {
		c.Patience = 2
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
	best, sweepEnd := climb(a.cfg, func(threads int) (float64, error) {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		mbps, err := a.probe(ctx, threads, a.cfg.ProbeDuration)
		log.Debug("adaptive: probe", "threads", threads, "mbps", mbps, "err", err)
		return mbps, err
	})

	log.Info("adaptive: chose threads",
		"best", best, "sweep_start", a.cfg.StartThreads, "sweep_end", sweepEnd)

	// Final recorded run at the chosen thread count.
	finalCfg := a.cfg.Base
	finalCfg.Threads = best
	finalCfg.TestDuration = a.cfg.FinalRunDuration

	started := time.Now()
	res, err := NewEngine(finalCfg).Run(ctx)
	out := packResult(finalCfg, started, res)
	out.Notes = append(out.Notes,
		fmt.Sprintf("adaptive: %d threads (sweep %d→%d)", best, a.cfg.StartThreads, sweepEnd))
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	if err == nil {
		err = errNoData(res)
	}
	return out, err
}

// climb doubles the thread count, tracking the best rate seen. One noisy
// probe can't end the sweep: it stops after Patience non-improving probes in
// a row, or on a probe error. Returns the best count and the last one probed.
func climb(c AdaptiveConfig, measure func(threads int) (float64, error)) (best, last int) {
	best, last = c.StartThreads, c.StartThreads
	var bestMbps float64
	misses := 0
	for threads := c.StartThreads; threads <= c.MaxThreads; threads *= 2 {
		mbps, err := measure(threads)
		if err != nil {
			break
		}
		last = threads
		if bestMbps == 0 || (mbps-bestMbps)/bestMbps >= c.MarginalGainThreshold {
			best, bestMbps, misses = threads, mbps, 0
			continue
		}
		if misses++; misses >= c.Patience {
			break
		}
	}
	return best, last
}

// probe runs a short download burst at the given thread count and returns its
// FinalMbps. Probes are intentionally brief — the goal is rank order, not
// final numbers.
func (a *adaptiveEngine) probe(ctx context.Context, threads int, dur time.Duration) (float64, error) {
	cfg := a.cfg.Base
	cfg.Threads = threads
	cfg.TestDuration = dur
	res, err := NewEngine(cfg).Run(ctx)
	if err != nil {
		return 0, err
	}
	if res == nil {
		return 0, fmt.Errorf("nil result")
	}
	return res.FinalMbps, nil
}
