package loss

import (
	"context"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

// engineRunner is the minimal contract both UDPEngine and HTTPEngine satisfy
// for the bench wrapper.
type engineRunner interface {
	Run(ctx context.Context) (*Result, error)
}

// NewBenchEngine wraps a loss engine into bench.Engine. Mode picks the
// concrete engine: HTTP (default) or UDP (LAN-only).
func NewBenchEngine(cfg Config) bench.Engine {
	if cfg.Mode == "" {
		cfg.Mode = ModeHTTP
	}
	var runner engineRunner
	switch cfg.Mode {
	case ModeUDP:
		runner = NewUDPEngine(cfg)
	default:
		runner = NewHTTPEngine(cfg)
	}
	return &benchEngine{cfg: cfg, runner: runner}
}

type benchEngine struct {
	cfg    Config
	runner engineRunner
}

func (e *benchEngine) Run(ctx context.Context) (bench.Result, error) {
	started := time.Now()
	res, err := e.runner.Run(ctx)
	if err != nil {
		return bench.Result{}, err
	}

	r := bench.NewResult(bench.KindLoss, e.cfg.Host, started, res.Duration)
	r.Metrics["loss_percent"] = res.LossPercent
	r.Metrics["jitter_ms"] = res.JitterMS
	r.Metrics["rtt_min_ms"] = res.MinLatencyMS
	r.Metrics["rtt_avg_ms"] = res.AvgLatencyMS
	r.Metrics["rtt_max_ms"] = res.MaxLatencyMS
	r.Counters["packets_sent"] = res.PacketsSent
	r.Counters["packets_received"] = res.PacketsReceived
	r.Counters["packets_lost"] = res.PacketsLost
	r.Counters["errors"] = res.Errors
	r.Flags["test_complete"] = res.TestComplete
	r.Flags["complete"] = res.TestComplete
	// Sample count for the loss test = packets received (RTTs measured). If
	// connection-level errors swamp samples, mark the result degraded.
	samples := res.PacketsReceived
	if res.Errors > samples {
		r.Flags["degraded"] = true
	}
	if len(res.Notes) > 0 {
		r.Notes = append(r.Notes, res.Notes...)
	}
	return r, nil
}
