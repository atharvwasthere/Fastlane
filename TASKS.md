# Fastlane v2 — Task Tracker

> Working checklist for the v2 refactor. Use alongside `ACTION_PLAN.md` (the deep-dive) and `fastlane_pm_design.html` (the visual reference).
> Each task: [ ] open, [~] in progress, [x] done. Mark dates when closing.

**Design reference:** `fastlane_pm_design.html` — verdict-led cards, 3-col metric grid, hint footer. Match this. No sparkline columns, no ASCII art beyond the start screen / version logo.

---

## Phase 0 — Cleanup (do first; clears the noise)

- [x] Delete root-level `PHASE_1_COMPLETE.md` … `PHASE_7_COMPLETE.md` — done (already gone from tree).
- [x] Delete `DOWNLOAD_BUGFIX_REPORT.md`, `DOWNLOAD_ME.txt`, `IMPLEMENTATION_STATUS.md` — done.
- [x] Delete `README_FINAL.md`, `the lastest fastlane_analysis.md` — done 2026-05-13.
- [x] Delete `some api.txt` (Ookla XML, license-restricted, not shipping) — done 2026-05-13.
- [x] `structure.txt`, `phase wise dev.txt`, `PRD.docx` — already gone.
- [x] `docs/` created with `CODE_WALKTHROUGH.md` (kept); moved `fastlane_pm_design.html` into `docs/`. — done 2026-05-13.
- [x] `.gitignore` has `bin/`, `*.exe`, `coverage.out`; added `~/.fastlane/`, `.rtk/`, `bash.exe.stackdump` — done 2026-05-13.
- [x] Delete `docs/server_selection.md` (stale MaxMind reference) — done 2026-05-13.
- [x] Delete leftover dev artifacts at root: `bash.exe.stackdump`, stray `fastlane` binary — done 2026-05-13.

## Phase 1 — Critical bug fixes (~3h)

- [x] **fix(ping):** `cmd/ping.go:23-32` — JSON path returns hardcoded `45.32`. Move `MeasureLayered` above the JSON branch; both paths consume the same `*LatencyResult`. — done 2026-04-29 (commit 60e9cd1)
- [x] **fix(root):** `cmd/root.go::Execute` — honor `--no-color` and `NO_COLOR` env. Wire `color.NoColor` and `lipgloss.SetDefaultRenderer(termenv.Ascii)` at startup. — done 2026-04-29 (commit 8bc19be) + lipgloss hook 2026-04-30.
- [x] **fix(download/upload):** `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` in `cmd/download.go` and `cmd/upload.go`; thread into `engine.Run(ctx)`. — done 2026-04-29 (commit 438f013)
- [x] **fix(server):** `internal/server/loader.go:6` — `io/ioutil` → `os.ReadFile`. — done 2026-04-29 (commit 516e6ac)
- [x] **fix(stubs):** Split `cmd/stubs.go` into `cmd/{servers,report,live,version,test}.go`. Repair the stray `}, ` at line 133 in `serversCmd`. — done 2026-04-29 (commit 32c1125)
- [x] **fix(loss):** `internal/loss/engine.go:135` — `close(ackChan)` race. Receiver owns the channel; sender signals via `done`; close after drain. — done 2026-04-29 (commit eaf66a5)
- [x] **fix(formatBytes):** Move out of `cmd/download.go:20-34` and `internal/download/progress.go`. Land in `internal/format/bytes.go` (or `pkg/render/format.go` after §4). — done 2026-04-29 (commit d3b87ea)

## Phase 2 — Architecture seams (~3h)

- [x] **arch:** Create `internal/bench/bench.go` with `Engine`, `LiveEngine`, `Result`, `Snapshot`, `Kind` types (per ACTION_PLAN §1.1). — done 2026-04-30
- [x] **arch:** `cmd/internal/run.go::RunBench(ctx, engine, renderer)` orchestrator. — done 2026-04-30
- [x] **arch:** Per-engine factory: `internal/<test>/factory.go::NewBenchEngine(cfg) bench.Engine`. — done 2026-04-30 (ping/download/upload/loss)
- [x] **arch:** `cmd/{ping,download,upload,loss}.go` no longer import concrete engine packages. The single seam is `cmd/internal/factories.go`, which exposes `PingEngine/DownloadEngine/UploadEngine/LossEngine` returning `bench.Engine`. — done 2026-04-30. `cmd/live.go` still imports `internal/live` (Phase 9 stub).

## Phase 3 — TUI overhaul (lipgloss + bubbletea, ~5h)

Reference: `fastlane_pm_design.html`. See also `.claude/skills/fastlane-tui-render/SKILL.md`.

- [x] **deps:** `go get github.com/charmbracelet/lipgloss github.com/charmbracelet/bubbletea github.com/charmbracelet/bubbles` (also pulled `muesli/termenv` for the ascii-profile hook). — done 2026-04-30
- [x] **deps:** Drop `briandowns/spinner` and `schollz/progressbar` from `go.mod`. — done 2026-05-13. Slimmed `pkg/ui/printer.go` to logo + tagline only; `go mod tidy` purged deps.
- [x] **render:** `pkg/render/render.go` — `Renderer` interface + `New(format, w, opts)` factory. — done 2026-04-30
- [x] **render:** `pkg/render/card.go` — verdict-led card (icon + title + subtitle + 3-col grid + hint). — done 2026-04-30
- [x] **render:** `pkg/render/json.go` — schema-locked JSON output (parity with card metrics, asserted by `TestCardJSONParity`). — done 2026-04-30
- [x] **render:** `pkg/render/prom.go` — Prometheus exposition. — done 2026-04-30
- [x] **render:** Width detection via `golang.org/x/term.GetSize`. Resolved once in `cmd/internal/renderer.go`. — done 2026-04-30
- [x] **render:** Responsive collapse (3-col → 2-col → 1-col by width). One responsive `FormatCard`, no preset sizes. — done 2026-04-30
- [x] **render:** Color tokens — green `#6db380`, yellow `#b8a060`, red `#b06060`, purple `#8878bb`, dim `#3a3a3a`, border `#181818`. — done 2026-04-30 (lipgloss.AdaptiveColor, with light variants)
- [x] **render:** Golden-file tests at widths 30, 50, 70, 90 + per-kind goldens at width 80, parity test, prom assertion test. `testdata/*.golden` rebuildable via `go test ./pkg/render -update`. — done 2026-04-30
- [ ] **render:** Keep `Printer.PrintLogo()` for start screen + version banner (per user direction; logo + version stay unchanged). (Untouched — Phase 12 work.)

## Phase 4 — Live runtime (bubbletea, ~3h)

- [x] **live:** `pkg/render/live.go` — `liveModel` with `Init/Update/View`. `SnapshotMsg`, `ResultMsg`, `ErrMsg`. — done 2026-05-09
- [x] **download:** ANSI cursor math gone with `internal/download/progress.go` deletion; live path now flows through `cmd/internal/run.go::runLive`. — done 2026-05-09
- [x] **upload:** Same as download — bubbletea program drives the live view; `internal/upload/progress.go` deleted. — done 2026-05-09
- [x] **engine:** `download.Engine.Run(ctx)` and `upload.Engine.Run(ctx)` are ctx-native; `Cancel()` shim removed. — done 2026-05-09
- [x] **engine:** `cmd/internal/run.go::runLive` ticks 150ms, calls `engine.Snapshot()`, sends `render.SnapshotMsg`. — done 2026-05-09
- [x] **delete:** `internal/download/progress.go` and `internal/upload/progress.go` removed. — done 2026-05-09

## Phase 5 — Loss replacement (~2h)

Skill: `.claude/skills/fastlane-loss/SKILL.md`.

- [x] **loss:** `internal/loss/http.go` — `HTTPEngine` with HEAD requests, rate-paced ticker, RTT collection. Default mode. — done 2026-05-09
- [x] **loss:** Rename `internal/loss/engine.go` → `udp.go`; type `UDPEngine`; gated behind `--mode udp`. — done 2026-05-09
- [x] **loss:** `cmd/loss.go` — `--mode http|udp` flag (icmp deferred). Default `http`. — done 2026-05-09
- [x] **loss:** Update `cmd/loss.go` config — port 7 default replaced; http default `https://www.cloudflare.com`, udp default `127.0.0.1`. — done 2026-05-09
- [x] **loss:** Tests: `httptest.Server` near-zero loss; half-loss via per-probe timeout; cancellation; conn-refused→errors; URL normalize. — done 2026-05-09

## Phase 6 — Server selection (~6h)

Skill: `.claude/skills/fastlane-servers/SKILL.md`. Reference: `ACTION_PLAN.md` §3.

### Path A — Cloudflare default (always-on)

- [x] **cf:** `internal/server/cloudflare.go` — `DetectCloudflareColo(ctx) (Colo, error)` parsing `cdn-cgi/trace`. — done 2026-05-13.
- [x] **cf:** Display `colo=` code in card subtitle when default endpoint is `speed.cloudflare.com`. — done 2026-05-13 via `cmd/internal/run.go::detectColoAsync` + `Result.Notes` surfaced by `pkg/render/card.go`. 250ms await deadline so colo never delays rendering.

### Path B — opt-in `--auto-server`

- [x] **assets:** `internal/server/data/servers.json` v2 — **22 verified LibreSpeed-compatible backends + Cloudflare anycast**, each with `download_path` and `upload_path` so `--auto-server` works for download AND upload on non-CF picks. Sourced from librespeed.org/backend-servers/servers.json, probed individually, only kept hosts that returned 200 on both download GET and upload POST. Coverage: NA=8, EU=12, RU=1, JP=1 (Tokyo A573), Anycast=1. **India gap:** all major Indian ISPs (Airtel/Jio/ACT/BSNL/Excitel/Spectra/Tata) migrated to Ookla proprietary protocol — zero verified open HTTPS backends. CF anycast covers IN traffic via Mumbai/Delhi/Chennai edges. Self-hosting a Mumbai/Bangalore VPS with LibreSpeed is the path to native IN coverage. — upgraded + IN/JP hunt 2026-05-13.
- [x] **embed:** `internal/server/embed.go` — `//go:embed data/servers.json` + `LoadEmbedded`, `LoadCachedOrEmbedded` (prefer `~/.fastlane/servers.json` if < 30d), `SaveCache`. — done 2026-05-13.
- [x] **probe:** `internal/server/probe.go` — `ProbeAll(ctx, servers, concurrency=8)` parallel TCP dial, 500ms timeout. — done 2026-05-13.
- [x] **geoip:** `internal/server/geoip.go` — ipapi.co, no API key, 24h cache to `~/.fastlane/cache/location.json`, 3s fetch timeout. — done 2026-05-13.
- [x] **select:** `internal/server/select.go::AutoSelect(ctx)` — geoip → haversine sort → top-5 shortlist (+ anycast anchors as fallback) → probe → lowest RTT, 1h decision cache. — done 2026-05-13.
- [x] **flag:** `--auto-server` on `download`, `upload`, `ping`. Routes through `cmd/internal/autoserver.go::AutoServerURL`. Now uses **per-server `download_path` / `upload_path`** from servers.json so non-CF picks actually serve the right endpoint (LibreSpeed `/backend/garbage.php` + `/backend/empty.php`). Falls back to CF defaults if a server lacks a path. — done 2026-05-13.
- [x] **cmd:** `cmd/servers.go` rewritten with `list`, `probe`, `update` subcommands; reads from embedded/cached list; tabwriter for text, JSON envelope for `--json`. — done 2026-05-13.
- [x] **update:** `fastlane servers update` — fetches `https://raw.githubusercontent.com/atharvwasthere/Fastlane/master/internal/server/data/servers.json` → writes to `~/.fastlane/servers.json`. — done 2026-05-13.

## Phase 7 — Reports (~4h)

Skill: `.claude/skills/fastlane-report/SKILL.md`.

- [x] **store:** `internal/report/store.go` — `Store` interface, `FileStore` impl, `Save/Load/List/Delete`, `Filter` (Kind/Since/Limit), lightweight `Meta` for listings. Path: `~/.fastlane/reports/` (falls back to `./.fastlane/reports` if home detect fails). Windows-safe filename: `YYYY-MM-DDTHH-MM-SS_<kind>_<short-host>`. — done 2026-05-13.
- [x] **wire:** `--save-report` honored once in `cmd/internal/run.go::RunBench` via `persistResult` helper. All four wired commands (ping/download/upload/loss) thread their `Flags.SaveReport` through. Save errors warn to stderr but never fail the command. — done 2026-05-13.
- [x] **list:** `fastlane report list [--limit N] [--kind X] [--since YYYY-MM-DD]` — tabwriter for text, JSON envelope for `--json`. — done 2026-05-13.
- [x] **show:** `fastlane report show <id>` — loads JSON, hands to `cmdint.NewRenderer(globalFlags).Final(r)` so the normal card renders. — done 2026-05-13.
- [x] **compare:** `fastlane report compare <a> <b>` — colored ▲/▼ per metric. Direction table (`metricDirection` map) encodes higher-better vs lower-better; unknown keys show delta with dim `·`. — done 2026-05-13.
- [x] **trend:** `fastlane report trend --metric KEY --days N [--kind X]` — ASCII sparkline via `github.com/guptarohit/asciigraph v0.9.0`. — done 2026-05-13.
- [x] **followup:** `cmd/test.go` now calls `cmdint.PersistResult` directly via the exported wrapper (keeps CI exit-code control AND persists). Added `--save-report` flag to test. — done 2026-05-13.

## Phase 8 — Test orchestrator (~2h)

Skill: `.claude/skills/fastlane-test-suite/SKILL.md`.

- [x] **test:** `cmd/test.go` — sequential ping → download → upload → loss. Aggregated via `internal/testsuite/engine.go` into a `bench.Result{Kind: KindTest}` with metric keys `latency_ms / download_mbps / upload_mbps / loss_percent` + rating note. `internal/report.BenchmarkSnapshot` didn't exist in tree, so the aggregate lives entirely in the bench schema. — done 2026-05-13.
- [x] **ci:** `--ci --min-download / --min-upload / --max-latency / --max-jitter / --max-loss`. Exit 0 pass / 1 violated / 2 couldn't-run. — done 2026-05-13.
- [x] **ci:** `testsuite.Thresholds.Check(result) []Violation`; violations printed to stderr. Tests in `internal/testsuite/thresholds_test.go`. — done 2026-05-13.
- [x] **render:** `KindTest` wired in `pkg/render/card.go` (verdict by rating, 4-cell metric grid PING/DOWN/UP/LOSS). — done 2026-05-13.

## Phase 9 — Live dashboard (~2h)

Skill: `.claude/skills/fastlane-live/SKILL.md`.

- [x] **live:** `internal/live/runner.go` — cycle ping → 3s download → 3s upload → loss probe; `Sources` struct accepts injected `bench.Engine`s so internal/live stays free of concrete engine imports. Stats merge per step, channel emits incremental snapshots. — done 2026-05-13.
- [x] **live:** Replace `rand.Float64` simulation in `cmd/live.go` with the real runner wired through `cmd/internal/factories.go`. — done 2026-05-13.
- [x] **live:** `internal/live/dashboard.go` — bubbletea model with channel-driven re-render and per-metric 60-sample sparkline. AltScreen mode so it redraws cleanly on any terminal (no more append-on-each-render). Replaces legacy ANSI `internal/live/ui.go` (deleted). — done 2026-05-13.
- [x] **live fix:** Runner switched to engine **factories** instead of instances. `bench.Engine` is single-shot (channels close on Run exit); reusing the same instance caused `panic: close of closed channel` in `download.collector` on the second cycle. — done 2026-05-13.
- [x] **live:** `cmd/live.go` (already split from stubs in earlier refactor). — done.

## Phase 10 — Production-readiness (~2h)

- [x] **slog:** `internal/logging/logger.go` exposes `Init(GlobalFlags) *slog.Logger` + `FromContext`. `cmd/root.go::PersistentPreRunE` wires it via `slog.SetDefault`. `--debug` → Debug, `--verbose` → Info, default → Warn. JSON mode mutes the logger (io.Discard). Real call-sites in `RunBench` and `select.go::AutoSelect`. — done 2026-05-13.
- [x] **config:** `internal/config/loader.go` — precedence flag > env (`FASTLANE_*`) > `~/.fastlane/config.json` > defaults. Uses `cmd.Flags().Changed()` to detect user-set flags. Bool env accepts `1/true/t/yes/y` (case-insensitive). 7 precedence tests in `loader_test.go`. — done 2026-05-13.
- [x] **errors:** Engines (ping/download/upload/loss-udp) carry `errCount` atomic; factories write `Result.Counters["errors"]` and set `Flags["degraded"]` when errors > samples/received. `pkg/render/card.go::buildSubtitle` appends `· degraded (N errors)`. — done 2026-05-13.
- [x] **goleak:** `go.uber.org/goleak v1.3.0` added. `goleak.VerifyNone` guards on one long-running integration test per engine package (download/upload/loss). — done 2026-05-13.
- [x] **ci-yaml:** `.github/workflows/ci.yml` — 3-OS test matrix with `-race`, golangci-lint, 5-target cross-build with artifact upload. — done 2026-05-13.

## Phase 11 — Standout features (pick 3, not all)

- [x] **ci-mode:** Done in Phase 8. 🥇
- [x] **adaptive-threads:** `internal/download/adaptive.go` — hill-climb (start 2 threads, step +2, max 16, 2s probe, 5% marginal-gain stop, then 6s final run at chosen count). `--adaptive-threads` flag on `fastlane download`. Decorates Result.Notes with `adaptive: N threads (sweep S→E)`. — done 2026-05-13. 🥇
- [x] **prom:** Done in Phase 3. 🥈
- [x] **trend:** Done in Phase 7. 🥈
- [x] **compare:** Done in Phase 7. 🥉
- [ ] **sqlite:** Deferred — `modernc.org/sqlite` history backing. Punt until file-store gets slow. 🥉

## Phase 12 — Polish

- [x] **readme:** Rewrote `readme.md` — install via `go install`, complete command surface, server-picking explainer + India/Asia note, architecture overview with output-format breakdown, build/config sections, credits. — done 2026-05-13.
- [ ] **help:** Restyle help body via lipgloss (logo + tagline kept). Per-command help with algorithm description + example. (Deferred — readme + version banner are the immediate user-facing surface; cobra default help is functional and the lipgloss restyle is pure polish.)
- [x] **version:** `cmd/version.go` with ldflags injection (`Version` / `Commit` / `BuildDate`) — Makefile auto-injects `git describe`, short SHA, UTC date. Logo + content preserved. — done 2026-05-13.
- [x] **ci-yaml:** `.github/workflows/ci.yml` — 3-OS test matrix, golangci-lint job, 5-target cross-build with artifact upload. — done 2026-05-13.

---

## Skill files

Per-command rules live under `.claude/skills/`. **Load the relevant skill before editing.**

| Skill                          | Load before editing                                              |
|--------------------------------|------------------------------------------------------------------|
| `fastlane-architecture`        | Anything in `cmd/`, `internal/`, or `pkg/`. Read this first.     |
| `fastlane-tui-render`          | `pkg/render/`, `pkg/ui/`, any `--json`/output code               |
| `fastlane-ping`                | `cmd/ping.go`, `internal/ping/`                                  |
| `fastlane-download`            | `cmd/download.go`, `internal/download/`                          |
| `fastlane-upload`              | `cmd/upload.go`, `internal/upload/`                              |
| `fastlane-loss`                | `cmd/loss.go`, `internal/loss/`                                  |
| `fastlane-servers`             | `cmd/servers.go`, `internal/server/`                             |
| `fastlane-report`              | `cmd/report.go`, `internal/report/`, `--save-report` wiring      |
| `fastlane-test-suite`          | `cmd/test.go`, `--ci` mode, threshold logic                      |
| `fastlane-live`                | `cmd/live.go`, `internal/live/`                                  |
| `fastlane-version`             | `cmd/version.go`, ldflags injection                              |
| `fastlane-help`                | `cmd/root.go` Long/Help, help templates                          |

Maintenance: invoke `/fastlane-skills-sync` after substantive refactors to bring skills back in sync.
