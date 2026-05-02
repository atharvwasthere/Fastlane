package loss

import (
	"context"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

// NewBenchEngine wraps the loss Engine into a bench.Engine. Loss already has
// native ctx support so this is a straight pack-the-result adapter.
func NewBenchEngine(cfg Config) bench.Engine {
	return &benchEngine{cfg: cfg, engine: NewEngine(cfg)}
}

type benchEngine struct {
	cfg    Config
	engine *Engine
}

func (e *benchEngine) Run(ctx context.Context) (bench.Result, error) {
	started := time.Now()
	res, err := e.engine.Run(ctx)
	if err != nil {
		return bench.Result{}, err
	}

	r := bench.NewResult(bench.KindLoss, e.cfg.Host, started, res.Duration)
	r.Metrics["loss_percent"] = res.LossPercent
	r.Metrics["jitter_ms"] = res.JitterMS
	r.Counters["packets_sent"] = res.PacketsSent
	r.Counters["packets_received"] = res.PacketsReceived
	r.Counters["packets_lost"] = res.PacketsLost
	r.Flags["test_complete"] = res.TestComplete
	return r, nil
}
