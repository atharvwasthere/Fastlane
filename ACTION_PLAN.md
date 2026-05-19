# Fastlane v2 — Action Plan

> Source of truth for the v2 refactor. Opinionated, prioritized, code-pointed.
> Companion to `CLAUDE.md` (architecture rules) and `fastlane_pm_design.html` (visual target).

---

## 0. Verdict (be honest)

**What's actually good:**
- `internal/stats/welford.go` and `ewma.go` — clean, testable numerical primitives. Keep.
- `internal/ping/ping.go` `MeasureLayered()` — genuine layered DNS / TCP / TLS / HTTP measurement.
- `internal/download/engine.go`, `internal/upload/engine.go` — goroutine-fanout + Welford/EWMA + CV-convergence pipeline. This is the spine.
- `pkg/output/json.go` — generic envelope is the right shape.

**What's broken or weak:**
1. `cmd/ping.go` JSON path returns a hardcoded `45.32 ms`. Fake numbers in JSON invalidate every script consumer.
2. `cmd/download.go` does raw ANSI cursor math (`\033[u`, `\033[3B`, ...) to redraw a TUI box. Brittle on Windows, duplicates `internal/download/progress.go::RenderFrame` which is never called.
3. `cmd/stubs.go` mixes 4 stub commands and (per prior review) had a syntax issue. Split per file.
4. `internal/loss/engine.go` sends UDP to port 7 (Echo). Port 7 is firewalled everywhere → reports 100% loss every time. Feature is non-functional.
5. `pkg/ui/printer.go::PrintBox` uses fixed 44-char widths that don't match any inner content. Source of the ragged-edge boxes.
6. `formatBytes` is duplicated in `cmd/download.go` and `internal/download/progress.go`.
7. `--no-color`, `--debug`, `--save-report` are parsed but never honored. Dead flags lie to users.
8. No `os.Interrupt` / `SIGTERM` handling. Ctrl+C mid-test leaks goroutines and HTTP connections.
9. `internal/server/loader.go:6` still imports `io/ioutil` (deprecated since Go 1.16; module is on 1.24).
10. `docs/server_selection.md` referenced MaxMind GeoLite2, but mmdb was removed in commit `f1a6596`. Doc was stale; deleted.

**Architecture verdict:** the engine layer is good. The cmd, UI, and server-selection layers are where the rot is. Fix those.

---

## 1. Architectural seams (SOLID)

### 1.1 The three contracts

```go
// internal/bench/bench.go (new)
package bench

type Engine interface {
    Run(ctx context.Context) (Result, error)
}

// LiveEngine is opt-in: download/upload implement it; ping/loss don't.
type LiveEngine interface {
    Engine
    Snapshot() Snapshot
}

type Result struct {
    Kind      Kind
    Server    string
    StartedAt time.Time
    Duration  time.Duration
    Metrics   map[string]float64
    Counters  map[string]int64
    Flags     map[string]bool
    Layers    map[string]float64 // ping: dns/tcp/tls/http
    Notes     []string
}

type Snapshot struct {
    Mean, EWMA, StdDev, CV float64
    Bytes                  int64
    Samples                int
    Converged              bool
}
```

- **SRP:** each engine has one method.
- **OCP:** new test types register without touching `cmd/`.
- **DIP:** `cmd/` depends on `bench.Engine`, not on concrete engines.
- **LSP:** every engine honors the same `Result` shape; renderer doesn't care which engine produced it.
- **ISP:** ping/loss don't pay for `Snapshot()` they never call.

### 1.2 Renderer Strategy

```go
// pkg/render/render.go (new)
type Renderer interface {
    Header(title string)
    Live(snap bench.Snapshot)
    Final(r bench.Result)
    Error(err error)
}

func New(format string, w io.Writer, opts Options) Renderer {
    switch format {
    case "json":   return &jsonRenderer{w: w}
    case "compact": return &compactRenderer{w: w, opts: opts}
    case "minimal": return &minimalRenderer{w: w, opts: opts}
    default:        return &cardRenderer{w: w, opts: opts}
    }
}
```

This kills `if globalFlags.JSON { ... } else { ... }` duplicated across every cmd.

### 1.3 Single orchestrator

```go
// cmd/internal/run.go (new)
func RunBench(ctx context.Context, e bench.Engine, r render.Renderer) error {
    if le, ok := e.(bench.LiveEngine); ok {
        // bubbletea program: poll le.Snapshot() on tick, run e.Run(ctx) in goroutine
    }
    res, err := e.Run(ctx)
    if err != nil { r.Error(err); return err }
    r.Final(res); return nil
}
```

Every cmd file becomes ~30 lines: parse flags → build engine config → build renderer → `RunBench`. No raw ANSI, no duplicated JSON branches.

---

## 2. TUI overhaul — verdict-led card design

### 2.1 Stack

- **`github.com/charmbracelet/lipgloss`** — all styling, padding, alignment, borders, color themes. Replaces hand-counted `%-44s` and `pkg/ui/printer.go` box drawing.
- **`github.com/charmbracelet/bubbletea`** — runtime + event loop for live-redrawing screens (`download`, `upload`, `live`). Replaces the raw `\033[u` cursor math in `cmd/download.go`. Static screens (help, version, ping, loss results) render via lipgloss alone — no bubbletea program needed.

Drop `briandowns/spinner` and `schollz/progressbar` from `go.mod`. Bubbletea has its own spinner; lipgloss handles progress bars.

### 2.2 The visual target (per `fastlane_pm_design.html`)

Verdict-led, three-section card:

```
┌─ ping cloudflare.com ──────────────────────────────────┐
│ ✓ Connection is healthy                                │
│   22 ms average · no packet loss · low jitter          │
│                                                        │
│   22ms              51ms              3ms              │
│   AVG LATENCY       WORST CASE        JITTER           │
│                                                        │
│   p95 is 38 ms — occasional spikes but nothing         │
│   consistent. Good for real-time use.                  │
└────────────────────────────────────────────────────────┘
```

Three structural elements, in order:

1. **Verdict row** — colored icon (`✓`/`!`/`✗`) + title (one line, plain English) + subtitle (3 dot-separated facts).
2. **Metric grid** — 3 columns, big-number value over small-caps label. Collapses to 1 column when TTY width < 40.
3. **Hint footer** — one or two lines of context, low-contrast text, optional left border accent for severity.

Color tokens (lipgloss `lipgloss.AdaptiveColor`):
- `green` — healthy (`#6db380`)
- `yellow` — warning (`#b8a060`)
- `red` — failure (`#b06060`)
- `purple` — upload accent (`#8878bb`)
- `dim` — labels and footer text (`#3a3a3a`)
- `border` — faint card frame (`#181818`)

**Removed from prior plan:** sparkline glyph column, `▁▂▃▅▇` ASCII art, multi-width "default/compact/minimal" preset cards. Replaced with a single responsive layout that adapts to terminal width.

### 2.3 Layout sizing

| TTY width | Layout                          |
|-----------|---------------------------------|
| ≥ 80      | 3-col metric grid, full hints   |
| 60–79     | 3-col grid, single-line hints   |
| 40–59     | 2-col grid, no hints            |
| < 40      | 1-col grid, no hints            |
| `--json`  | bypass the renderer entirely    |

Detect via `golang.org/x/term`. Lipgloss handles re-flow.

### 2.4 What stays unchanged

- **Start screen** (`fastlane` no args / `fastlane help`): the existing logo + tagline approach is kept. Only the help body below it gets re-themed via lipgloss.
- **`fastlane version`** output: kept as-is; lipgloss styling but same content.

### 2.5 Live progress (download/upload)

Bubbletea model:

```go
type liveModel struct {
    engine    bench.LiveEngine
    snapshot  bench.Snapshot
    sparkline []float64 // ring buffer of last 32 EWMA samples
    spinner   spinner.Model
    quitting  bool
}

func (m liveModel) Init() tea.Cmd { return tea.Batch(m.spinner.Tick, tickCmd()) }
func (m liveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
func (m liveModel) View() string { /* lipgloss-rendered card */ }
```

Engine runs in a goroutine, posts `tea.Msg` updates on a channel via `tea.Cmd`. No more cursor math.

---

## 3. Server selection (two paths)

The old `docs/server_selection.md` referenced MaxMind. Deleted. New plan:

### 3.1 Default path — Cloudflare colo (always-on, zero-cost)

For `fastlane ping`, `download`, `upload` with no `--server` flag and no `--auto-server`:

```go
// internal/server/cloudflare.go (new, ~30 lines)
func DetectCloudflareColo(ctx context.Context) (Colo, error) {
    // GET https://speed.cloudflare.com/cdn-cgi/trace
    // Parse k=v lines: loc=IN, colo=DEL, ip=...
}
```

Returns `colo=DEL` (or whichever edge), no geo lookup, no server list, no probing. The CDN routes you to your nearest edge automatically — that's the whole point of Anycast. Default endpoint stays `speed.cloudflare.com`. This path needs zero work in `internal/server/`.

### 3.2 Opt-in path — `--auto-server` with embedded list + fetch-update

For users who want a non-CF server (privacy, regional ISP testing, library-grade benchmarks):

```
fastlane download --auto-server
fastlane download --server hetzner-fsn1
fastlane servers list                 # show embedded list
fastlane servers update               # pull fresh list from GitHub raw
```

**List sourcing — hybrid (1) + (3):**

- **(1) Embed at build time:** `assets/servers.json` baked in via `//go:embed`. Initial population: adapt **LibreSpeed's public server list** (https://github.com/librespeed/speedtest-servers, MIT-licensed, ~100 servers globally) + handcrafted CDN anchors (Cloudflare, CacheFly, OVH, Hetzner, Tele2). Credit LibreSpeed in README.
- **(3) Fresher cache at `~/.fastlane/servers.json`:** `fastlane servers update` fetches `https://raw.githubusercontent.com/atharvwasthere/Fastlane/master/assets/servers.json`. If cache exists and is < 30 days old, prefer it over embedded.

**Selection algorithm (when `--auto-server` is set):**

```
1. Detect user country: GET https://ipapi.co/json/  (no key, free tier)
   - cache to ~/.fastlane/location.json for the session
2. Filter servers.json to user country + same continent
   - Haversine sort (geo.go already does this)
3. Take top 5 by distance
4. Parallel TCP-ping shortlist (net.DialTimeout, 500ms timeout each)
   - new file: internal/server/probe.go
5. Pick lowest RTT, cache decision to ~/.fastlane/server.json (TTL 1h)
```

**What `some api.txt` (Ookla XML) is:** discarded. Ookla's server list is license-restricted; their upload-PHP endpoints aren't raw-HTTP throughput compatible. Not shipping.

**What's missing today:**
- `assets/servers.json` — does not exist
- `internal/server/cloudflare.go` — CF trace client, ~30 lines
- `internal/server/probe.go` — parallel TCP-ping, ~80 lines
- `internal/server/geoip.go` — ipapi.co client, ~40 lines
- `internal/server/embed.go` — `//go:embed` + cache merge logic
- `loader.go:6` — `ioutil` → `os.ReadFile`

### 3.3 Don't break the default path

`fastlane ping cloudflare.com` and `fastlane download` (no flags) must keep working with **zero** `internal/server/` involvement. The two-path split exists so server selection becomes opt-in code, not always-on.

---

## 4. Critical bug fixes (do these first)

Each names file:line and the fix. Total ~3h.

| # | File:Line | Fix |
|---|-----------|-----|
| 4.1 | `cmd/ping.go:23-32` | Move `MeasureLayered` above the JSON branch; both paths consume the same `*LatencyResult`. Kill the hardcoded `45.32`. |
| 4.2 | `cmd/root.go:33-38` | Honor `--no-color` and `NO_COLOR` env: `color.NoColor = ...` at start of `Execute()`. |
| 4.3 | `cmd/download.go`, `cmd/upload.go` | `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)`, thread into `engine.Run(ctx)`. |
| 4.4 | `internal/server/loader.go:6` | `io/ioutil` → `os` (`ioutil.ReadFile` → `os.ReadFile`). |
| 4.5 | `cmd/stubs.go` | Split into `servers.go`, `report.go`, `live.go`, `version.go`, `test.go`. |
| 4.6 | `internal/loss/engine.go` | Replace UDP-echo with HTTP-HEAD loss: N HEAD requests with `--rate` pacing, count timeouts. ICMP behind `--mode icmp` later. |
| 4.7 | `cmd/download.go` + `internal/download/progress.go` | De-dup `formatBytes` → `pkg/render/format.go` (created in §1.2). |
| 4.8 | `internal/loss/engine.go:135` | `close(ackChan)` race. Pattern: receiver owns the channel, sender signals via `done`, receiver closes after drain. |

---

## 5. Production-readiness

### 5.1 Errors

Engines currently `time.Sleep(100ms); continue` on any HTTP error. Hides systemic failures. Replace with:

```go
if err != nil {
    if errors.Is(err, context.Canceled) { return }
    e.recordError(err)
    backoff.Sleep()
    continue
}
```

Surface count in `Result.Counters["errors"]`. If `errors > samples`, mark `Result.Notes` with `"degraded: N errors during run"`.

### 5.2 Structured logging — `log/slog`

One logger at root, threaded via context. `--debug` → `slog.LevelDebug`. Now `--debug` is real, not dead.

### 5.3 Config precedence

```
CLI flag > env var (FASTLANE_*) > ~/.fastlane/config.json > built-in default
```

Same as `kubectl`, `gh`, `aws-cli`. `internal/config/loader.go` merges once at startup.

### 5.4 Tests

Add integration tests against `httptest.NewServer`. One per engine. Use `goleak.VerifyNone(t)` to catch goroutine leaks. Currently 0 integration tests; aim for >80% coverage on engines.

---

## 6. Stand-out features (pick top 3, not all)

| # | Feature | Hours | Impact |
|---|---------|-------|--------|
| 6.1 | `--ci` mode with thresholds (`--min-download 50 --max-latency 100`), exit 1 on violation | 1 | 🥇 |
| 6.2 | Adaptive thread scaling (hill-climb until marginal Mbps gain < 5%) | 3 | 🥇 |
| 6.3 | Prometheus exposition format (`--format prometheus`) | 1 | 🥈 |
| 6.4 | `fastlane report trend --metric download.mbps --days 7` (sparkline history) | 3 | 🥈 |
| 6.5 | `fastlane report compare <a> <b>` (colored deltas) | 2 | 🥉 |
| 6.6 | SQLite-backed history (`modernc.org/sqlite`, no CGO) | 4 | 🥉 |

---

## 7. Execution sequence

| Step | What | Hours | Skill |
|------|------|-------|-------|
| 1 | Critical fixes §4.1–4.5 | 2 | general |
| 2 | Split `cmd/stubs.go` into per-command files | 0.5 | general |
| 3 | Introduce `internal/bench` interfaces (§1.1) | 1 | architecture |
| 4 | Introduce `pkg/render` Renderer strategy (§1.2) | 2 | tui-render |
| 5 | Add lipgloss + bubbletea, build verdict-card renderer (§2) | 5 | tui-render |
| 6 | Migrate `download` and `upload` cmds to renderer + bubbletea live | 3 | download, upload |
| 7 | Migrate `ping` and `loss` cmds to renderer | 1.5 | ping, loss |
| 8 | Replace UDP-echo loss with HTTP-HEAD (§4.6) | 2 | loss |
| 9 | CF colo client `internal/server/cloudflare.go` (§3.1) | 1 | servers |
| 10 | `assets/servers.json` from LibreSpeed + CDN anchors | 1 | servers |
| 11 | `internal/server/{embed,geoip,probe}.go` for `--auto-server` (§3.2) | 3 | servers |
| 12 | `fastlane servers list/update` subcommands | 1.5 | servers |
| 13 | `internal/report/store.go` + `--save-report` honored | 2 | report |
| 14 | `fastlane report list/show/compare` | 2 | report |
| 15 | `fastlane test` orchestrator (ping → down → up → loss) | 1.5 | test-suite |
| 16 | Wire `fastlane live` to real engines | 2 | live |
| 17 | `--ci` flag with thresholds | 1 | general |
| 18 | slog wiring, `--debug` honored | 1 | general |
| 19 | Adaptive threading (optional) | 3 | download |
| 20 | Prometheus output | 1 | tui-render |
| 21 | GitHub Actions CI (`go test`, `golangci-lint`, build matrix) | 1 | general |
| 22 | Rewrite `readme.md` with screenshots | 0.5 | general |

**Total:** ~37h for finished v2. First 12 steps (~22h) deliver the visible quality jump.

---

## 8. Cleanup

Already deleted: `docs/server_selection.md` (stale MaxMind reference).

Still to delete from repo root:
- `DOWNLOAD_BUGFIX_REPORT.md`, `DOWNLOAD_ME.txt`, `IMPLEMENTATION_STATUS.md`
- `PHASE_1_COMPLETE.md` … `PHASE_7_COMPLETE.md`
- `phase wise dev.txt`, `some api.txt` (Ookla XML, not shipping)
- `README_FINAL.md`, `the lastest fastlane_analysis.md`, `Why the wait.md`, `structure.txt`

Move to `docs/`: keep only authoritative material. `PRD.docx` only if still current.

`.gitignore` additions: `bin/`, `*.exe`, `coverage.out`, `~/.fastlane/`.

---

## 9. SOLID self-check

| Letter | Question | Where |
|--------|----------|-------|
| S | One reason to change per `internal/*` package? | engine + its progress mate only |
| O | Can I add a new test type without editing `cmd/*`? | `bench.Engine` registration |
| L | Can any `Engine` impl swap for another and the renderer still works? | `bench.Result` shape |
| I | Is `bench.Engine` minimal? Live features in `LiveEngine`? | `bench/bench.go` |
| D | Does `cmd/` depend on `internal/download` directly? | After §1.1, only via `bench.Engine` |

If any regress, the PR isn't done.

---

## 10. What "done" looks like

- `fastlane ping cloudflare.com` and `fastlane ping cloudflare.com --json` return the **same numbers**.
- `fastlane download` (no flags) uses CF Anycast, zero `internal/server/` involvement.
- `fastlane download --auto-server` picks the lowest-RTT server from the embedded list.
- Verdict cards render correctly at 30, 60, 80, 120 column widths.
- `Ctrl+C` mid-test cleanly cancels, prints partial results, exits 130.
- `fastlane test --ci --min-download 50` exits 1 on slow networks, drops in a GitHub Action.
- `fastlane report compare <a> <b>` shows colored deltas.
- `go test ./...` hits >80% coverage on engines, `goleak.VerifyNone(t)` everywhere.
- `readme.md` has a screenshot of the new TUI and `go install github.com/atharvwasthere/Fastlane@latest`.
