// Package internal holds the cmd-layer orchestrator. It runs a bench.Engine
// against a render.Renderer so individual cmd/ files stay thin and never
// import concrete engine packages.
package internal

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
	"github.com/atharvwasthere/Fastlane/internal/logging"
	"github.com/atharvwasthere/Fastlane/internal/report"
	"github.com/atharvwasthere/Fastlane/internal/server"
	"github.com/atharvwasthere/Fastlane/pkg/render"
)

// reportStore is overridable in tests; production code uses a FileStore rooted
// at ~/.fastlane/reports. Keeping this as a package-level var avoids threading
// a store through every cmd.
var reportStore report.Store = report.NewFileStore("")

// persistResult is the single point where successful Results land on disk.
// All cmd/*.go files flow through RunBench, so wiring save here gives every
// command --save-report support without per-command code. Failures are
// surfaced to stderr but never fail the command — the measurement itself
// already succeeded.
func persistResult(ctx context.Context, save bool, res bench.Result) {
	if !save || res.Kind == "" {
		return
	}
	if _, err := reportStore.Save(ctx, res); err != nil {
		fmt.Fprintf(os.Stderr, "warning: save report: %v\n", err)
	}
}

// PersistResult is the exported wrapper for commands that bypass RunBench
// (currently only `test`, which needs the Result back for CI exit-code logic).
func PersistResult(ctx context.Context, save bool, res bench.Result) {
	persistResult(ctx, save, res)
}

// detectColoAsync kicks off a best-effort Cloudflare colo lookup if the server
// URL targets speed.cloudflare.com. Returns a channel that emits the parsed
// "colo=XXX loc=YY" annotation, or closes empty on failure/timeout/non-CF.
func detectColoAsync(ctx context.Context, serverURL string) <-chan string {
	out := make(chan string, 1)
	if !strings.Contains(serverURL, "speed.cloudflare.com") {
		close(out)
		return out
	}
	go func() {
		defer close(out)
		cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		c, err := server.DetectCloudflareColo(cctx)
		if err != nil {
			return
		}
		if s := c.String(); s != "" {
			out <- s
		}
	}()
	return out
}

// awaitColo reads from a colo channel with a short deadline so we never delay
// the user. Returns "" if nothing arrives in time.
func awaitColo(ch <-chan string) string {
	select {
	case s, ok := <-ch:
		if !ok {
			return ""
		}
		return s
	case <-time.After(250 * time.Millisecond):
		return ""
	}
}

// RunBench is the canonical run loop. For static renderers (json, prom) it
// runs the engine to completion and routes the Result to the renderer. For
// the card renderer with a bench.LiveEngine, it spawns a bubbletea program
// that pumps Snapshot() at a fixed cadence — replacing the raw-ANSI cursor
// math previously inlined in cmd/download.go and cmd/upload.go.
func RunBench(ctx context.Context, title, serverURL string, engine bench.Engine, r render.Renderer, save bool) error {
	log := logging.FromContext(ctx)
	log.Debug("runbench start", "kind", title, "server", serverURL, "save", save)

	live, hasLive := engine.(bench.LiveEngine)
	if hasLive && render.IsCard(r) {
		log.Debug("runbench live mode", "kind", title)
		return runLive(ctx, title, serverURL, live, r, save)
	}

	coloCh := detectColoAsync(ctx, serverURL)

	r.Header(title, serverURL)
	res, err := engine.Run(ctx)
	if err != nil {
		log.Debug("engine error", "kind", title, "err", err)
		if ctx.Err() != nil && res.Kind != "" {
			res.Notes = append(res.Notes, "test cancelled — partial results")
			r.Final(res)
			persistResult(ctx, save, res)
			return nil
		}
		r.Error(err)
		return err
	}
	if colo := awaitColo(coloCh); colo != "" {
		res.Notes = append(res.Notes, colo)
	}
	log.Info("runbench done",
		"kind", title,
		"duration_ms", res.Duration.Milliseconds(),
		"errors", res.Counters["errors"],
		"degraded", res.Flags["degraded"],
	)
	r.Final(res)
	persistResult(ctx, save, res)
	return nil
}

// runLive drives a bubbletea program against the live engine. The engine
// runs in a goroutine; a ticker pumps Snapshot() into the program; when the
// engine returns, ResultMsg/ErrMsg quits the program and the static card
// renderer prints the final view.
func runLive(ctx context.Context, title, serverURL string, engine bench.LiveEngine, r render.Renderer, save bool) error {
	opts, _ := render.CardOptions(r)
	w, ok := render.CardWriter(r)
	if !ok {
		w = os.Stdout
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	coloCh := detectColoAsync(runCtx, serverURL)

	// Title is the canonical kind string ("download"/"upload"/"ping"/"loss").
	prog := render.NewLiveProgram(w, title, serverURL, bench.Kind(title), opts, cancel)

	// Engine goroutine — sends terminal Result/Err into the program.
	go func() {
		res, err := engine.Run(runCtx)
		if err != nil && runCtx.Err() == nil {
			prog.Send(render.ErrMsg{Err: err})
			return
		}
		if runCtx.Err() != nil && res.Kind != "" {
			res.Notes = append(res.Notes, "test cancelled — partial results")
		}
		if colo := awaitColo(coloCh); colo != "" {
			res.Notes = append(res.Notes, colo)
		}
		// Persist before sending the terminal message — Save is fast (<5ms)
		// and the program quit shouldn't outrun a synchronous file write.
		persistResult(ctx, save, res)
		prog.Send(render.ResultMsg{Result: res})
	}()

	// Snapshot pump — ticks until the program quits.
	pumpDone := make(chan struct{})
	go func() {
		defer close(pumpDone)
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				prog.Send(render.SnapshotMsg{Snap: engine.Snapshot()})
			}
		}
	}()

	if _, err := prog.Run(); err != nil {
		cancel()
		<-pumpDone
		return err
	}
	cancel()
	<-pumpDone
	return nil
}
