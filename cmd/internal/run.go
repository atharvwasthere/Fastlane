// Package internal holds the cmd-layer orchestrator. It runs a bench.Engine
// against a render.Renderer so individual cmd/ files stay thin and never
// import concrete engine packages.
package internal

import (
	"context"

	"github.com/atharvwasthere/Fastlane/internal/bench"
	"github.com/atharvwasthere/Fastlane/pkg/render"
)

// RunBench is the canonical run loop. It draws the header, runs the engine,
// and routes the terminal Result (or error) to the renderer. Live progress
// is intentionally NOT wired here — Phase 4 layers it in via bubbletea.
func RunBench(ctx context.Context, title, server string, engine bench.Engine, r render.Renderer) error {
	r.Header(title, server)
	res, err := engine.Run(ctx)
	if err != nil {
		// Surface partial results when ctx was cancelled mid-test (the
		// bandwidth factories return Result + ctx.Err in that case).
		if ctx.Err() != nil && res.Kind != "" {
			res.Notes = append(res.Notes, "test cancelled — partial results")
			r.Final(res)
			return nil
		}
		r.Error(err)
		return err
	}
	r.Final(res)
	return nil
}
