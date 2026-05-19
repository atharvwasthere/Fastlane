package live

import (
	"context"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

// Sources holds engine FACTORIES (not engine instances) for each step.
// bench.Engine implementations are single-shot — their internal channels
// close on Run() exit, so a cycling dashboard MUST construct a fresh engine
// per iteration. Factory closures keep internal/live free of concrete
// engine imports while supporting unbounded re-runs.
type Sources struct {
	Ping     func() bench.Engine
	Download func() bench.Engine
	Upload   func() bench.Engine
	Loss     func() bench.Engine
}

// Runner cycles the four sub-engines in a long loop, pushing a fresh Stats
// onto its output channel after each step. Closes the channel on cancel.
type Runner struct {
	sources Sources
	out     chan Stats
	stats   Stats
}

func NewRunner(s Sources) *Runner {
	return &Runner{sources: s, out: make(chan Stats, 4)}
}

// C exposes the read end of the stats channel for consumers.
func (r *Runner) C() <-chan Stats { return r.out }

// Run executes the cycle ping → download → upload → loss → (repeat) until
// ctx is done. After each sub-engine completes its Stats fields are merged
// into the running state and a snapshot is sent on the channel.
func (r *Runner) Run(ctx context.Context) {
	defer close(r.out)
	for ctx.Err() == nil {
		r.step(ctx, r.sources.Ping, mergePing)
		r.step(ctx, r.sources.Download, mergeDownload)
		r.step(ctx, r.sources.Upload, mergeUpload)
		r.step(ctx, r.sources.Loss, mergeLoss)
		// Small idle gap so the dashboard doesn't pin the link.
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (r *Runner) step(ctx context.Context, factory func() bench.Engine, merge func(*Stats, bench.Result)) {
	if factory == nil || ctx.Err() != nil {
		return
	}
	eng := factory()
	if eng == nil {
		return
	}
	res, err := eng.Run(ctx)
	if err != nil {
		return
	}
	merge(&r.stats, res)
	r.stats.UpdatedAt = time.Now()
	select {
	case r.out <- r.stats:
	default:
		// Drop if consumer is slow — UI re-renders on ticker anyway.
	}
}

func mergePing(s *Stats, r bench.Result) {
	s.LatencyMS = r.Metrics["latency_ms"]
	s.Jitter = r.Metrics["jitter_ms"]
}

func mergeDownload(s *Stats, r bench.Result) {
	s.DownloadMbps = r.Metrics["final_mbps"]
}

func mergeUpload(s *Stats, r bench.Result) {
	s.UploadMbps = r.Metrics["final_mbps"]
}

func mergeLoss(s *Stats, r bench.Result) {
	s.LossPercent = r.Metrics["loss_percent"]
	s.PacketsSent = r.Counters["packets_sent"]
	s.PacketsReceived = r.Counters["packets_received"]
}
