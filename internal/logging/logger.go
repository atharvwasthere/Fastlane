// Package logging wires the global slog logger and a context carrier so any
// package can pick up a request-scoped logger via FromContext.
//
// Level mapping:
//   --debug    -> slog.LevelDebug
//   --verbose  -> slog.LevelInfo
//   default    -> slog.LevelWarn
//   --json     -> handler discards every record (clean stdout JSON,
//                 clean stderr too).
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/atharvwasthere/Fastlane/internal/config"
)

// ctxKey is unexported so only this package can stamp loggers into a ctx.
type ctxKey struct{}

// Init builds a *slog.Logger from the resolved global flags. JSON mode
// returns a logger backed by io.Discard so no log output leaks to stderr
// while structured data is on stdout.
func Init(flags config.GlobalFlags) *slog.Logger {
	if flags.JSON {
		// Drop everything — keep stdout/stderr clean for the JSON envelope.
		return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.Level(127), // higher than any real level
		}))
	}
	level := slog.LevelWarn
	switch {
	case flags.Debug:
		level = slog.LevelDebug
	case flags.Verbose:
		level = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}

// ContextWith returns a derived ctx carrying logger. A nil logger is
// silently ignored so callers don't need defensive checks.
func ContextWith(ctx context.Context, logger *slog.Logger) context.Context {
	if logger == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, logger)
}

// FromContext returns the logger stamped on ctx, or slog.Default() so callers
// never get a nil. slog.Default() is always usable, even before Init.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}
