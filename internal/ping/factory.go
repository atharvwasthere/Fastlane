package ping

import (
	"context"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

// BenchConfig configures the ping bench.Engine wrapper.
type BenchConfig struct {
	Host       string
	Iterations int
	Timeout    time.Duration
}

// NewBenchEngine returns a bench.Engine that runs MeasureLayered + Pauta
// filtering and packs the output into a bench.Result.
func NewBenchEngine(cfg BenchConfig) bench.Engine {
	if cfg.Iterations <= 0 {
		cfg.Iterations = 5
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &benchEngine{cfg: cfg}
}

type benchEngine struct {
	cfg BenchConfig
}

func (e *benchEngine) Run(ctx context.Context) (bench.Result, error) {
	started := time.Now()

	// MeasureLayered is synchronous and does its own timing internally. We
	// honor ctx by returning early if it's already cancelled before each
	// phase; the existing function does not accept ctx, so this is a
	// best-effort gate until ping.MeasureLayered grows ctx support.
	if err := ctx.Err(); err != nil {
		return bench.Result{}, err
	}

	raw, err := MeasureLayered(e.cfg.Host, e.cfg.Iterations, e.cfg.Timeout)
	if err != nil {
		return bench.Result{}, err
	}

	filter := NewPautaFilter()
	filtered, removed := filter.Filter(raw.Samples)

	r := bench.NewResult(bench.KindPing, e.cfg.Host, started, time.Since(started))
	r.Metrics["latency_ms"] = raw.MeanMS
	r.Metrics["jitter_ms"] = raw.JitterMS
	r.Metrics["min_ms"] = raw.MinMS
	r.Metrics["max_ms"] = raw.MaxMS
	r.Layers["dns_ms"] = raw.DNSLatencyMS
	r.Layers["tcp_ms"] = raw.TCPLatencyMS
	r.Layers["tls_ms"] = raw.TLSLatencyMS
	r.Layers["http_ms"] = raw.HTTPLatencyMS
	r.Counters["samples"] = int64(len(filtered))
	r.Counters["outliers_removed"] = int64(removed)
	r.Counters["errors"] = raw.Errors
	if raw.Errors > int64(len(filtered)) {
		r.Flags["degraded"] = true
	}
	return r, nil
}
