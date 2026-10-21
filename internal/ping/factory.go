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

// NewBenchEngine returns a bench.Engine that runs MeasureLayered and packs the
// output into a bench.Result.
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

	raw, err := MeasureLayered(ctx, e.cfg.Host, e.cfg.Iterations, e.cfg.Timeout)
	if err != nil {
		return bench.Result{}, err
	}

	r := bench.NewResult(bench.KindPing, e.cfg.Host, started, time.Since(started))
	r.Metrics["latency_ms"] = raw.MedianMS
	r.Metrics["jitter_ms"] = raw.JitterMS
	r.Metrics["min_ms"] = raw.MinMS
	r.Metrics["max_ms"] = raw.MaxMS
	r.Layers["dns_ms"] = raw.DNSLatencyMS
	r.Layers["tcp_ms"] = raw.TCPLatencyMS
	r.Layers["tls_ms"] = raw.TLSLatencyMS
	r.Layers["http_ms"] = raw.HTTPLatencyMS
	r.Counters["samples"] = int64(len(raw.Samples))
	r.Counters["errors"] = raw.Errors
	if raw.Errors > int64(len(raw.Samples)) {
		r.Flags["degraded"] = true
	}
	return r, nil
}
