# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common commands

Build/test/run is driven by the Makefile (uses Unix shell syntax — on Windows run via Git Bash / WSL, or invoke `go` directly):

- `make build` — compiles to `./bin/fastlane` (`go build -v -o ./bin/fastlane .`)
- `make test` — `go test -v -cover ./...`
- `make bench` — `go test -bench=. -benchmem ./...`
- `make cross-build` — produces linux-amd64, darwin-arm64, windows-amd64 binaries in `./bin/`
- `make fmt` / `make lint` (golangci-lint optional)
- Run a single package's tests: `go test ./internal/download/...`
- Run a single test: `go test ./internal/stats -run TestWelford`

The CLI entry point is `main.go` → `cmd.Execute()`. After building, invoke subcommands like `./bin/fastlane ping`, `./bin/fastlane download`, `./bin/fastlane upload`, `./bin/fastlane loss`. Global flags (`--json`, `--verbose`, `--timeout`, `--debug`, `--no-color`) are persistent on the root command.

Module path: `github.com/atharvwasthere/Fastlane` (Go 1.24.4). Dependencies are minimal: cobra, fatih/color, briandowns/spinner, schollz/progressbar.

## Architecture

Fastlane is a network-benchmarking CLI. The codebase is split into three layers, and the layering is load-bearing — keep them separated when adding features:

1. **`cmd/`** — Cobra command definitions. Each file (`ping.go`, `download.go`, `upload.go`, `loss.go`) is a thin wrapper that parses flags into a `config.CommandFlags` struct, instantiates an engine from `internal/<test>`, and renders results via `pkg/ui` (text) or `pkg/output` (JSON). `stubs.go` holds commands that aren't fully wired up yet (`test`, `live`, `servers`, `report`, `version`) — when implementing one of these, replace the stub in place rather than adding a new file. `root.go` owns global flag wiring.

2. **`internal/`** — All measurement logic. The pattern is one package per test type, each exposing an `Engine` with `NewEngine(Config) *Engine` and `Run() (*Result, error)`. Long-running engines (download/upload) also expose `GetCurrentStats()`, `GetBytesDownloaded()`, `GetSampleCount()` for live progress polling from `cmd/` while `Run()` executes in a goroutine. Shared building blocks:
   - `internal/stats/` — `Welford` (numerically-stable running mean/variance) and `EWMA` (exponentially-weighted moving average). Both are used by download/upload engines for streaming statistics and convergence detection (CV threshold default 0.03).
   - `internal/config/` — `GlobalFlags` and `CommandFlags` structs shared by all commands.
   - `internal/ping/` — layered latency (DNS → TCP → TLS → HTTP) plus a Pauta outlier filter applied after sampling.
   - `internal/server/` — server list loader + GeoIP-based selection (`geo.go`). Currently used by stub `servers` command.
   - `internal/report/` — `BenchmarkSnapshot` aggregate type with a `Rate()` method that scores ping/download/upload/loss into `EXCELLENT/GOOD/FAIR/POOR`. Used as the canonical structured result; not yet persisted.
   - `internal/live/` — terminal UI for live visualization (used by the `live` stub).

3. **`pkg/`** — Output and presentation, importable by external code in principle.
   - `pkg/output/` — `JSONWriter` plus the generic `Result{Timestamp, Command, Server, Data map[string]interface{}, Error}` envelope used by every command's `--json` path.
   - `pkg/ui/` — `Printer` with logo/box/section/spinner helpers and color theme. Commands print headers via `Printer.PrintLogo()` + `PrintTaglineBox()` + `PrintBox(title)` then a final `PrintBoxFooter(msg)`.

### Conventions to preserve

- **JSON vs text dual path:** every command checks `globalFlags.JSON` first and emits via `output.NewResult(...)` + `JSONWriter.WriteResult(...)`; otherwise it renders through `ui.Printer`. Keep both paths in sync when adding fields.
- **Live progress in download/upload** uses raw ANSI escapes (`\033[u`, `\033[<n>B`, `\033[K`) to redraw a fixed box in place rather than reprinting. If you touch the layout, update both the static box drawn before the loop and the cursor offsets inside the ticker case.
- **Convergence detection** is the stop condition for bandwidth tests: a test ends when CV (stddev/mean of recent samples) drops below `CVThreshold` *or* `TestDuration` elapses. Don't replace it with fixed-duration timing.
- **Pauta filter** (3σ) is applied to ping samples after collection, not during. Mean/jitter/min/max are reported on the filtered set.

### Repo state notes

The git status shows a large pending refactor: many files under `cmd/`, `internal/nettest/`, `server/`, `types/`, `ui/`, `utils/` are deleted in favor of the new `internal/<test>/engine.go` layout described above. The `readme.md` project-structure section is **stale** — it still documents the pre-refactor `internal/nettest/` layout. Trust the live tree, not the readme. Numerous `PHASE_*_COMPLETE.md` files at the repo root are development logs; they describe historical milestones, not current behavior.

<!-- rtk-instructions v2 -->
# RTK (Rust Token Killer) - Token-Optimized Commands

## Golden Rule

**Always prefix commands with `rtk`**. If RTK has a dedicated filter, it uses it. If not, it passes through unchanged. This means RTK is always safe to use.

**Important**: Even in command chains with `&&`, use `rtk`:
```bash
# ❌ Wrong
git add . && git commit -m "msg" && git push

# ✅ Correct
rtk git add . && rtk git commit -m "msg" && rtk git push
```

## RTK Commands by Workflow

### Build & Compile (80-90% savings)
```bash
rtk cargo build         # Cargo build output
rtk cargo check         # Cargo check output
rtk cargo clippy        # Clippy warnings grouped by file (80%)
rtk tsc                 # TypeScript errors grouped by file/code (83%)
rtk lint                # ESLint/Biome violations grouped (84%)
rtk prettier --check    # Files needing format only (70%)
rtk next build          # Next.js build with route metrics (87%)
```

### Test (60-99% savings)
```bash
rtk cargo test          # Cargo test failures only (90%)
rtk go test             # Go test failures only (90%)
rtk jest                # Jest failures only (99.5%)
rtk vitest              # Vitest failures only (99.5%)
rtk playwright test     # Playwright failures only (94%)
rtk pytest              # Python test failures only (90%)
rtk rake test           # Ruby test failures only (90%)
rtk rspec               # RSpec test failures only (60%)
rtk test <cmd>          # Generic test wrapper - failures only
```

### Git (59-80% savings)
```bash
rtk git status          # Compact status
rtk git log             # Compact log (works with all git flags)
rtk git diff            # Compact diff (80%)
rtk git show            # Compact show (80%)
rtk git add             # Ultra-compact confirmations (59%)
rtk git commit          # Ultra-compact confirmations (59%)
rtk git push            # Ultra-compact confirmations
rtk git pull            # Ultra-compact confirmations
rtk git branch          # Compact branch list
rtk git fetch           # Compact fetch
rtk git stash           # Compact stash
rtk git worktree        # Compact worktree
```

Note: Git passthrough works for ALL subcommands, even those not explicitly listed.

### GitHub (26-87% savings)
```bash
rtk gh pr view <num>    # Compact PR view (87%)
rtk gh pr checks        # Compact PR checks (79%)
rtk gh run list         # Compact workflow runs (82%)
rtk gh issue list       # Compact issue list (80%)
rtk gh api              # Compact API responses (26%)
```

### JavaScript/TypeScript Tooling (70-90% savings)
```bash
rtk pnpm list           # Compact dependency tree (70%)
rtk pnpm outdated       # Compact outdated packages (80%)
rtk pnpm install        # Compact install output (90%)
rtk npm run <script>    # Compact npm script output
rtk npx <cmd>           # Compact npx command output
rtk prisma              # Prisma without ASCII art (88%)
```

### Files & Search (60-75% savings)
```bash
rtk ls <path>           # Tree format, compact (65%)
rtk read <file>         # Code reading with filtering (60%)
rtk grep <pattern>      # Search grouped by file (75%). Format flags (-c, -l, -L, -o, -Z) run raw.
rtk find <pattern>      # Find grouped by directory (70%)
```

### Analysis & Debug (70-90% savings)
```bash
rtk err <cmd>           # Filter errors only from any command
rtk log <file>          # Deduplicated logs with counts
rtk json <file>         # JSON structure without values
rtk deps                # Dependency overview
rtk env                 # Environment variables compact
rtk summary <cmd>       # Smart summary of command output
rtk diff                # Ultra-compact diffs
```

### Infrastructure (85% savings)
```bash
rtk docker ps           # Compact container list
rtk docker images       # Compact image list
rtk docker logs <c>     # Deduplicated logs
rtk kubectl get         # Compact resource list
rtk kubectl logs        # Deduplicated pod logs
```

### Network (65-70% savings)
```bash
rtk curl <url>          # Compact HTTP responses (70%)
rtk wget <url>          # Compact download output (65%)
```

### Meta Commands
```bash
rtk gain                # View token savings statistics
rtk gain --history      # View command history with savings
rtk discover            # Analyze Claude Code sessions for missed RTK usage
rtk proxy <cmd>         # Run command without filtering (for debugging)
rtk init                # Add RTK instructions to CLAUDE.md
rtk init --global       # Add RTK to ~/.claude/CLAUDE.md
```

## Token Savings Overview

| Category | Commands | Typical Savings |
|----------|----------|-----------------|
| Tests | vitest, playwright, cargo test | 90-99% |
| Build | next, tsc, lint, prettier | 70-87% |
| Git | status, log, diff, add, commit | 59-80% |
| GitHub | gh pr, gh run, gh issue | 26-87% |
| Package Managers | pnpm, npm, npx | 70-90% |
| Files | ls, read, grep, find | 60-75% |
| Infrastructure | docker, kubectl | 85% |
| Network | curl, wget | 65-70% |

Overall average: **60-90% token reduction** on common development operations.
<!-- /rtk-instructions -->