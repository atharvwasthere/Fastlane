# Fastlane Code Walkthrough

A guided tour of every file that runs when you type `fastlane <command>`. Written for someone newish to Go who wants to understand the codebase without grep-spelunking through every file. References use `path/file.go:LINE` so you can click through in your editor.

> Read this top-to-bottom once. After that, the per-section index is enough.

---

## Section 0 — Mental model in one paragraph

Fastlane is a CLI built with **cobra** (the same library `kubectl`, `gh`, and `hugo` use). It has one entry point (`main.go`), which delegates to a tree of commands defined in `cmd/`. Each command builds an "engine" from the `internal/` packages, runs it, and prints the result. There are four real engines (`ping`, `download`, `upload`, `loss`) plus a few stubs. Everything below is just elaboration on those two sentences.

```
main.go
   │
   ▼
cmd.Execute()                  ← cmd/root.go
   │
   ├── parses flags via cobra
   ├── routes to a subcommand (ping/download/upload/loss/...)
   │
   ▼
cmd/<name>.go                  ← thin wrapper, ~100-250 lines
   │
   ├── builds a Config struct
   ├── calls internal/<name>.NewEngine(config)
   ├── calls engine.Run()
   │
   ▼
internal/<name>/engine.go      ← the real work
   │
   ├── spawns goroutines
   ├── feeds samples into internal/stats (Welford, EWMA)
   ├── returns a Result struct
   │
   ▼
back to cmd/<name>.go
   │
   ├── prints via pkg/ui (text)        OR
   └── prints via pkg/output (JSON)
```

---

## Section 1 — The entry point

### `main.go` (12 lines)

```go
package main
import "github.com/atharvwasthere/Fastlane/cmd"
func main() { cmd.Execute() }
```

That's the entire file. It calls one function. The reason this is so small: **idiomatic Go CLIs put all command logic in a sub-package** (`cmd/`) so `main` is just a launcher. This makes the code testable — you can write `cmd.Execute()` from a test without spinning up a whole binary.

### Go gotcha #1: package vs directory
A directory in Go == one package. The file says `package main` because every executable Go program needs exactly one `main` package with one `main()` function. Subdirectories like `cmd/` are *different* packages. You can't put two `package main` files in the same directory.

### Go gotcha #2: imports use the module path, not the relative path
`"github.com/atharvwasthere/Fastlane/cmd"` is not a URL — it's the **module path** declared in `go.mod` (line 1: `module github.com/atharvwasthere/Fastlane`) joined with the subdirectory. The Go toolchain knows `github.com/atharvwasthere/Fastlane` resolves to the project root because `go.mod` says so.

---

## Section 2 — The cobra root command

### `cmd/root.go` (47 lines)

This file defines the root command (`fastlane` with no subcommand) and the **persistent flags** that apply to every subcommand.

Key pieces:

**Lines 11-14** — Two package-level variables:
```go
var (
    globalFlags config.GlobalFlags
    saveReport  bool
)
```
These are filled in by cobra when flags are parsed. Every subcommand can read `globalFlags.JSON`, `globalFlags.Verbose`, etc.

**Lines 17-29** — The root command itself:
```go
var rootCmd = &cobra.Command{
    Use:     "fastlane",
    Short:   "Fastlane ▸ Network benchmarking made simple.",
    Version: "0.1.0",
    Run:     func(cmd *cobra.Command, args []string) { ... },
}
```
`rootCmd` is a pointer to a `cobra.Command` struct. The `Run` field is a function — when cobra resolves the command tree and finds you typed just `fastlane` (no subcommand), this function fires and prints the help screen.

**Lines 33-38** — The `Execute()` function called from `main.go`:
```go
func Execute() {
    err := rootCmd.Execute()
    if err != nil { os.Exit(1) }
}
```
This is the cobra "kick the tires" call. It walks `os.Args`, matches the subcommand, parses flags, and invokes the matching `Run` function.

**Lines 40-47** — `init()`. This is where persistent flags are registered:
```go
func init() {
    rootCmd.PersistentFlags().BoolVar(&globalFlags.JSON, "json", false, "Output results as JSON")
    // ... etc
}
```

### Go gotcha #3: `init()` is magical
Every `.go` file can have an `init()` function. Go runs *every* `init()` in *every* imported package before `main()` ever starts. This is how cobra subcommands "register" themselves — each `cmd/<name>.go` has its own `init()` that does `rootCmd.AddCommand(<name>Cmd)`. By the time `main()` runs, the command tree is already built.

This is also a footgun: if your `init()` does something expensive or has side effects, that cost is paid on every run, even `fastlane --help`.

### Go gotcha #4: pointer-to-struct vs struct
`var rootCmd = &cobra.Command{...}` gives you a `*cobra.Command` (pointer). Cobra needs pointers because it mutates the struct (adds child commands, sets parent links). If you accidentally pass a value (`cobra.Command{...}` without `&`), you'd be passing a copy and any mutations would be lost.

---

## Section 3 — A concrete command flow: `fastlane ping cloudflare.com`

This is the simplest of the real commands. Walk through it once and the others click into place.

### `cmd/ping.go` (87 lines)

**Lines 17-86** — `pingCmd`:
```go
var pingCmd = &cobra.Command{
    Use:   "ping [host]",
    Short: "Measure latency to a server",
    Args:  cobra.MaximumNArgs(1),     // accept 0 or 1 positional arg
    Run:   func(cmd *cobra.Command, args []string) { ... },
}
```

The `Run` body, step by step:

1. **JSON early-exit** (lines 23-32) — if `--json`, build a fake-data result, write JSON, return. **Note: this branch returns hardcoded values right now; this is a known bug — see `ACTION_PLAN.md` §3.1.**

2. **Pick the host** (lines 34-37) — use `--server` flag, fall back to `"google.com"`. (It should also accept the positional `args[0]` but currently doesn't.)

3. **Print the header** (lines 39-45) — logo, tagline, box. All via `pkg/ui/printer.go`.

4. **Spinner on, do the work, spinner off** (lines 46-63):
   ```go
   printer.StartSpinner("PING", "Testing latency...")
   result, err := ping.MeasureLayered(server, 5, 10*time.Second)
   if err != nil { ... return }
   filter := ping.NewPautaFilter()
   filtered, removed := filter.Filter(result.Samples)
   printer.StopSpinner()
   ```
   This is the entire test. Two function calls.

5. **Print results** (lines 65-78) — section headers + `fmt.Printf` with the layer breakdown.

**Lines 82-86** — `init()` registers flags and adds `pingCmd` to `rootCmd`:
```go
func init() {
    pingCmd.Flags().StringVar(&pingFlags.Server, "server", "", "Target server host")
    rootCmd.AddCommand(pingCmd)
}
```

### `internal/ping/ping.go` — what `MeasureLayered` actually does

This is where the real network work lives. Read it in full — it's only 167 lines and it's a great example of straight-line Go.

**Lines 12-24** — The result type:
```go
type LatencyResult struct {
    DNSLatencyMS   float64
    TCPLatencyMS   float64
    TLSLatencyMS   float64
    HTTPLatencyMS  float64
    // ... etc
    Samples []float64
}
```

**Lines 27-115** — `MeasureLayered`:
1. Strip `https://`, `http://`, and any path from the host (lines 37-39).
2. **DNS layer** (lines 42-52): call `net.LookupIP(host)` N times, time each, average.
3. **TCP layer** (lines 55-66): call `net.DialTimeout("tcp", host+":80", timeout)` N times, time each.
4. **TLS layer** (lines 69-85): call `tls.DialWithDialer(...)` to port 443 N times.
5. **HTTP layer** (lines 88-100): full HTTP GET, time each.
6. **Aggregate** (lines 102-112): combine samples, compute mean/min/max/jitter.

### Go gotcha #5: `defer` and `Close()`
Notice line 61: `conn.Close()` after recording the sample. In production Go you'd write `defer conn.Close()` immediately after creating the connection so it always runs even if the function panics. The current code is fine because nothing between `Dial` and `Close` can fail, but it's a pattern to watch for.

### Go gotcha #6: error handling style
Every network call returns `(value, error)` and the code does `if err == nil { ... record sample ... }`. **It does NOT** propagate errors up — if DNS fails, that iteration silently skips. This is intentional for benchmarking (one bad sample shouldn't kill the test) but in normal Go you'd usually return the error.

The standard idiom you'll see everywhere:
```go
result, err := someCall()
if err != nil {
    return nil, fmt.Errorf("some context: %w", err)
}
// use result
```
The `%w` wraps the error so callers can `errors.Is(err, target)` to check the root cause.

### `internal/ping/filter.go` — Pauta outlier filter

`PautaFilter` removes samples more than 3 standard deviations from the mean. Read `filter.go:17-61`. It's straightforward: compute mean and stddev, drop anything outside `mean ± 3*stddev`, return the survivors and the count removed.

The fallback at lines 46-58 (if everything got filtered, keep the median) is sloppy — but harmless in practice because real network data rarely triggers it.

---

## Section 4 — The bandwidth tests: `download` and `upload`

These two are nearly identical. Once you understand `download`, `upload` is a 90% mirror with `POST` instead of `GET`.

### `cmd/download.go` (250 lines, half of which is ANSI cursor math)

**Lines 36-242** — `downloadCmd`. Walking through `Run`:

1. **Pick URL** (lines 42-46) — default is Cloudflare's 100 MB endpoint.
2. **Build config** (lines 49-67) — `download.Config` struct with threads, timeout, convergence threshold (CV = 0.03), update interval (100ms), test duration.
3. **JSON path** (lines 73-97) — run the engine, format JSON, write, return.
4. **Text path with live UI** (lines 100-241):
   - Clear screen, print header.
   - **Run engine on a goroutine** (lines 111-120):
     ```go
     resultChan := make(chan *download.Result)
     errChan := make(chan error)
     go func() {
         result, err := engine.Run()
         if err != nil { errChan <- err } else { resultChan <- result }
     }()
     ```
   - **Live update loop** (lines 124-216) — every 200ms, poll `engine.GetCurrentStats()`, redraw the box using raw ANSI escape codes.

The ANSI escape soup (`\033[u`, `\033[<n>B`, `\033[K`) means "save cursor, move down N lines, clear to end of line." It's painful to read and breaks on Windows CMD. **The TUI refactor (`ACTION_PLAN.md` §2) replaces this with `lipgloss` + `bubbletea`.**

### Go gotcha #7: goroutines + channels
The pattern at lines 111-120 is the bread-and-butter Go concurrency model:
- `make(chan *download.Result)` creates an unbuffered channel for results.
- `go func() {...}()` spawns a goroutine (cheap thread).
- Inside the goroutine, the work runs and sends the result on the channel.
- Outside, the main goroutine reads from the channel via `select` (line 129).

**Unbuffered means** the sender blocks until someone reads, and the reader blocks until someone sends. This is a built-in synchronization primitive — no mutex needed.

### Go gotcha #8: `select` is like `switch` for channels
```go
select {
case result = <-resultChan:    // case 1: result arrived
    break
case err := <-errChan:          // case 2: error arrived
    os.Exit(1)
case <-ticker.C:                // case 3: ticker fired
    // update UI
}
```
Whichever case is *ready first* fires. If multiple are ready, Go picks one pseudo-randomly. There's also a `default:` case (not used here) for "if nothing is ready, do this and don't block."

### `internal/download/engine.go` — the real engine (281 lines)

This file is the heart of the project. Read it carefully.

**Lines 14-28** — `Result` struct: final test results.

**Lines 31-39** — `Config` struct: tunable parameters.

**Lines 41-65** — `Engine` struct: holds runtime state.
```go
type Engine struct {
    config Config
    ctx    context.Context
    cancel context.CancelFunc
    mu sync.RWMutex                  // protects the fields below
    welford *stats.Welford           // running mean/variance
    ewma    *stats.EWMA              // smoothed bandwidth
    samplesChan chan float64         // workers → collector
    samples     []float64            // recorded samples
    bytesDownloaded int64            // atomic counter
    converged       bool             // true when CV < threshold
    convergenceCV   float64
    workerDone chan struct{}         // signals collector finished
}
```

### Go gotcha #9: `sync.RWMutex`
`mu` protects shared fields (`samples`, `converged`, `convergenceCV`) because the collector goroutine writes to them while the main goroutine reads them via `GetCurrentStats()`. Without the mutex, you'd have a data race.

`RWMutex` allows multiple readers OR one writer:
- `mu.RLock()` / `mu.RUnlock()` — for reading. Multiple goroutines can hold this at once.
- `mu.Lock()` / `mu.Unlock()` — for writing. Exclusive.

Use `RWMutex` when reads vastly outnumber writes (true here: the UI polls many times per second, the collector writes per sample).

### Go gotcha #10: `atomic.AddInt64`
`bytesDownloaded int64` is mutated by every worker. Instead of locking the mutex for a single integer increment, use `atomic.AddInt64(&e.bytesDownloaded, n)`. This is faster and lock-free. Read with `atomic.LoadInt64(&e.bytesDownloaded)`.

**Lines 104-147** — `Run()`: the orchestration. Spawns the collector goroutine, spawns N worker goroutines, runs a ticker loop checking for convergence. When done (timeout or converged), it carefully shuts down in this order:

```go
case <-testCtx.Done():
    wg.Wait()             // wait for all workers to exit
    close(e.samplesChan)  // signal collector to stop
    <-e.workerDone        // wait for collector to drain & exit
    return e.buildResult(...), nil
```

### Go gotcha #11: NEVER close a channel from the receiver
The pattern above is the canonical Go fanout/fanin pattern. Workers SEND on `samplesChan`. The collector RECEIVES. **Always close from the sender side, never the receiver.** If a worker tries to send to a closed channel, it panics. So the orchestrator waits for all senders (`wg.Wait()`) before closing.

`workerDone` is a `chan struct{}` (zero-byte channel) used purely as a signal — `<-e.workerDone` blocks until somebody calls `close(e.workerDone)`. Idiomatic Go for "tell me when you're done."

**Lines 150-198** — `worker()`: each goroutine runs this in a loop. Repeatedly:
1. Build an HTTP request with a context (so cancellation propagates).
2. `client.Do(req)`.
3. `io.Copy(io.Discard, resp.Body)` — drain the response body to nowhere, counting bytes.
4. Compute Mbps, send on `samplesChan`.
5. Loop.

### Go gotcha #12: `io.Discard`
`io.Discard` is a writer that throws everything away. `io.Copy(io.Discard, body)` is the fastest way to "consume and ignore" a response body. You **must** drain the body before closing it, otherwise the underlying connection can't be reused (HTTP keep-alive breaks).

**Lines 201-224** — `collector()`: reads samples from the channel, feeds them into Welford and EWMA, checks convergence:
```go
for sample := range e.samplesChan {       // loops until channel is closed
    e.mu.Lock()
    e.welford.Add(sample)
    e.ewma.Add(sample)
    e.samples = append(e.samples, sample)
    if len(e.samples) >= e.config.MinSamples {
        cv := e.welford.CoefficientOfVariation()
        e.convergenceCV = cv
        e.converged = (cv < e.config.CVThreshold && cv > 0)
    }
    e.mu.Unlock()
}
```

### Go gotcha #13: `for x := range channel`
This idiom loops until the channel is closed. Each iteration reads one value. When the channel closes, the loop exits. This is why "always close from the sender" matters — closing the channel is how you tell the receiver "no more data, exit cleanly."

### `internal/upload/engine.go` — same pattern, different verb

The only differences from download:
- `worker()` (lines 156-214) builds a POST request with a random-bytes body via `bytes.NewReader(buffer)`.
- It pre-fills a buffer with `crypto/rand.Read(buffer)` per request — this is overkill (`crypto/rand` is slow), see `ACTION_PLAN.md` §upload bug #2.
- Bytes counted on success: `atomic.AddInt64(&e.bytesUploaded, e.config.ChunkSize)`.

Everything else is byte-identical.

---

## Section 5 — The math: `internal/stats/`

Two tiny files, both worth reading in full.

### `internal/stats/welford.go` (99 lines)

Implements **Welford's online algorithm** for mean and variance. The naive way to compute variance is `sum((x - mean)²) / N`, but this requires storing every sample. Welford updates `mean` and `M2` (sum of squared deviations) one sample at a time, in O(1) memory.

Read lines 24-43 — `Add(x)`:
```go
w.count++
delta := x - w.mean
w.mean = w.mean + delta/float64(w.count)
delta2 := x - w.mean
w.M2 = w.M2 + delta*delta2
```
Five lines. This is the famous Welford recurrence. Numerically stable even with millions of samples; survives floating-point cancellation that would wreck the naive formula.

`CoefficientOfVariation()` (lines 93-98) returns `stddev / mean` — a normalized "spread" metric. The download/upload engines use this as the convergence signal: when CV drops below 3%, the bandwidth has stabilized and the test can stop.

### `internal/stats/ewma.go` (40 lines)

**Exponential Weighted Moving Average.** A smoother that weights recent samples more than old ones:
```go
new_value = α * sample + (1 - α) * old_value
```

With `α = 0.2` (the default), each new sample contributes 20% and the existing average contributes 80%. The output lags slightly but rejects spikes well — perfect for a "current bandwidth" reading on the live UI.

---

## Section 6 — The other commands (briefly)

### `cmd/loss.go` + `internal/loss/engine.go`

Sends UDP packets, counts replies. **Currently broken** — port 7 is firewalled on real targets, so it always reports 100% loss. See `ACTION_PLAN.md` §3.6 for the HTTP-HEAD replacement plan.

### `cmd/stubs.go` (testCmd, liveCmd, serversCmd, reportCmd, versionCmd)

Five commands in one file, all stubs. They print fake data or call simulated random sources. Each will eventually become its own file under `cmd/` with a real implementation. See `ACTION_PLAN.md` §6 for the order.

### `internal/server/`

Three files (types, geo, loader) implementing server discovery and ranking — geographically-aware "which server should we test against?" The Haversine formula in `geo.go:9-23` calculates great-circle distance between lat/lon pairs. The loader has a hardcoded fallback list and a JSON loader, but nothing in the project actually calls it yet. Wired up in the upcoming refactor.

### `internal/live/ui.go`

A bar-chart TUI for live mode. Renders four bars (latency, download, upload, loss) with colors that change based on value. Currently fed by `rand.Float64()` from `cmd/stubs.go::liveCmd` — not real data.

### `internal/report/types.go`

Types for saved reports, plus a `Rate()` method that scores network health as EXCELLENT/GOOD/FAIR/POOR. Persistence (writing to `~/.fastlane/reports/`) is **not implemented yet** — the `--save-report` flag is parsed but ignored.

---

## Section 7 — The output layer

### `pkg/ui/printer.go` (209 lines)

A wrapper around `fatih/color`, `briandowns/spinner`, and `schollz/progressbar`. Provides `PrintLogo`, `PrintBox`, `StartSpinner`, `StopSpinner`, `PrintError`, etc.

The boxes have hardcoded widths (44 chars), which is why the output looks ragged — the inner content the commands print doesn't match those widths. Will be replaced by `lipgloss` (see `ACTION_PLAN.md` §2).

### `pkg/output/json.go` + `text.go`

`JSONWriter.WriteResult(result)` → `json.MarshalIndent` → write to stdout. The `Result` struct is a generic envelope with a `Data map[string]interface{}` for arbitrary fields. Each command builds its own `Result` and stuffs metric fields into `Data`.

### Go gotcha #14: `interface{}` (now `any`)
`map[string]interface{}` accepts any value type. In Go 1.18+, `interface{}` got a shorter alias: `any`. They're identical. The codebase still uses `interface{}` because it predates the alias.

This is also called the "empty interface" — every type implements it (because every type has at least zero methods), so it's the Go equivalent of `Object` in Java or `Any` in TypeScript.

---

## Section 8 — The dependency tree

```
main.go
└── cmd
    ├── internal/config       (flag structs only)
    ├── internal/ping         ──┐
    ├── internal/download     ──┤── all import internal/stats
    ├── internal/upload       ──┤
    ├── internal/loss           │
    ├── internal/live           │
    ├── internal/server         │
    ├── internal/report         │
    ├── pkg/output             (json + text writers)
    └── pkg/ui                 (printer)

External:
    spf13/cobra              (CLI framework)
    fatih/color              (ANSI colors)
    briandowns/spinner       (loading animations)
    schollz/progressbar/v3   (progress bars)
```

No circular imports — Go forbids them at compile time.

### Go gotcha #15: `internal/` is a real keyword
A package under `internal/` can only be imported by code inside the same module. So `github.com/atharvwasthere/Fastlane/internal/ping` is importable from anywhere in *this* repo, but if someone else `go get`s your module, they CAN'T import your `internal/` packages. This is a Go-language-level enforcement, not a convention.

`pkg/` has no special meaning — it's just a convention for "stuff intended to be importable by outsiders."

---

## Section 9 — A real run, end to end

When you type:
```bash
./fastlane download --threads 8 --json
```

Here's what happens, frame-by-frame:

1. **OS** loads `fastlane.exe`, jumps to `main()` in `main.go`.
2. **Go runtime** runs every `init()` in every imported package — `cmd/root.go::init()` registers persistent flags; `cmd/download.go::init()` registers `--threads`/`--server`/`--save-report` and adds `downloadCmd` as a child of `rootCmd`. Similarly for ping/upload/loss/stubs.
3. `main()` calls `cmd.Execute()`.
4. **cobra** walks `os.Args` (`["./fastlane", "download", "--threads", "8", "--json"]`), finds `downloadCmd`, parses the flags into `downloadFlags` and `globalFlags`.
5. **cobra** invokes `downloadCmd.Run(cmd, args)`.
6. The Run function reads `globalFlags.JSON` → true → takes the JSON branch.
7. Builds `download.Config{URL: "https://speed.cloudflare.com/__down...", Threads: 8, ...}`.
8. `download.NewEngine(cfg)` allocates the engine, makes the channels, makes Welford and EWMA.
9. `engine.Run()` (this is on the main goroutine in JSON mode):
    - Spawns the collector goroutine.
    - Spawns 8 worker goroutines via a `sync.WaitGroup`.
    - Each worker loops: GET request → drain body → compute Mbps → send on `samplesChan`.
    - Collector loops: read from `samplesChan` → feed Welford + EWMA → check convergence.
    - Main goroutine ticks every 100ms, checks `e.converged` under `mu.RLock()`.
    - When converged (or timeout fires): `wg.Wait()`, `close(samplesChan)`, `<-workerDone`, return `Result`.
10. `cmd/download.go` builds a `pkg/output.Result`, stuffs metrics into `Data`, calls `JSONWriter.WriteResult(jsonResult)`.
11. JSON appears on stdout. Process exits.

Total elapsed for a typical fast connection: ~6-12 seconds (until convergence).

---

## Section 10 — Common Go gotchas summarized

A cheatsheet you can revisit. Numbered to match the inline references above.

| #  | Gotcha                                  | Where                                                |
|----|-----------------------------------------|------------------------------------------------------|
| 1  | One package per directory, `main` once  | `main.go`                                            |
| 2  | Imports use module path, not relative   | every file's `import (...)`                          |
| 3  | `init()` runs before `main()`           | `cmd/*.go::init()`                                   |
| 4  | Pointer vs value matters for mutation   | `&cobra.Command{}` not `cobra.Command{}`             |
| 5  | `defer` for cleanup                     | should be used more in `internal/ping/ping.go`       |
| 6  | `(value, error)` return + `if err != nil` | every network call                                 |
| 7  | Goroutines + channels for concurrency   | `cmd/download.go:111-120`                            |
| 8  | `select` for multi-channel reads        | `cmd/download.go:129`                                |
| 9  | `sync.RWMutex` for read-heavy sharing   | `internal/download/engine.go::Engine.mu`             |
| 10 | `atomic.AddInt64` for lock-free counters| `internal/download/engine.go:184`                    |
| 11 | Close channels from sender, never receiver | `internal/download/engine.go:128-132`             |
| 12 | `io.Discard` to drain response bodies   | `internal/download/engine.go:179`                    |
| 13 | `for x := range chan` exits on close    | `internal/download/engine.go:205`                    |
| 14 | `interface{}` == `any` (1.18+)          | `pkg/output/json.go:33`                              |
| 15 | `internal/` is enforced by the compiler | the entire `internal/` tree                          |

### Bonus gotchas you'll hit soon

**16. `nil` slices vs empty slices.** `var s []int` and `s := []int{}` look similar but `var` gives you a nil slice. `len(nil_slice)` returns 0 (safe!), `append(nil_slice, x)` works (allocates new backing array). The difference matters mostly for JSON marshaling: nil slices become `null`, empty slices become `[]`.

**17. Maps must be initialized before use.** `var m map[string]int` is nil; writing to it panics. Use `m := make(map[string]int)` or `m := map[string]int{}`. Reading from a nil map is safe (returns zero value), but writing crashes.

**18. Capitalized = exported.** `ping.MeasureLayered` is callable from other packages because it starts with a capital letter. `ping.average` isn't. This is the only access control Go has — no `public/private/protected` keywords.

**19. Struct field tags are strings.** `json:"latency_ms"` on a struct field tells `encoding/json` what key to use. They look like comments but they're real metadata; misspelling them silently does the wrong thing.

**20. Goroutine leaks are silent.** A goroutine stuck waiting on a channel that no one will ever send to just sits there forever. Your program keeps running, the goroutine never returns, and nothing tells you. Tools like `go.uber.org/goleak` check this in tests — recommended for the engine tests (mentioned in `ACTION_PLAN.md` §4.4).

**21. `defer` runs in LIFO order.** Multiple `defer` calls execute reverse-order on function exit. So:
```go
defer fmt.Println("first")
defer fmt.Println("second")
// prints: second, first
```

**22. `defer` arguments are evaluated immediately.** `defer fmt.Println(time.Now())` captures `time.Now()` at the `defer` line, not at function exit. Use a closure if you want late evaluation: `defer func() { fmt.Println(time.Now()) }()`.

---

## Section 11 — Where to look next

| If you want to...                                     | Read                                              |
|-------------------------------------------------------|---------------------------------------------------|
| Understand how a command goes from CLI to engine       | This doc, sections 2-3                            |
| Modify the engine for a specific test                  | `internal/<test>/engine.go`                       |
| Change how results are displayed                       | `pkg/ui/printer.go` and `cmd/<cmd>.go` Run body   |
| Add a new test type                                    | Copy `internal/loss/`, register in a new `cmd/x.go` |
| Fix the JSON ping bug                                  | `cmd/ping.go:23-32` and `ACTION_PLAN.md` §3.1     |
| Plan a refactor                                        | `ACTION_PLAN.md` §6                               |
| Update agent skills after refactor                     | `.claude/skills/fastlane-skills-sync/SKILL.md`    |
| Stage your messy commits cleanly                       | `.claude/skills/fastlane-commit-staging/SKILL.md` |

When in doubt: `go doc github.com/atharvwasthere/Fastlane/internal/download` will print the exported API of any package without you needing to open the file.

---

## Section 12 — One-shot quiz to confirm you've got it

Try answering without scrolling up. Answers in `docs/CODE_WALKTHROUGH_ANSWERS.md` (not included — work it out yourself).

1. Why does `main.go` have only 12 lines?
2. What does `cmd.Execute()` actually do?
3. Where does `--threads` get registered?
4. Why is the `samplesChan` always closed by the `Run()` method, not by a worker?
5. What does `CoefficientOfVariation()` represent and why does the engine use it as a stop condition?
6. If I add a new file `internal/foo/foo.go`, can I import it from `cmd/`? Could a third party using `go get github.com/atharvwasthere/Fastlane` import it?
7. What's wrong with `cmd/ping.go` lines 23-32 right now?
8. Why does `worker()` use `atomic.AddInt64` for `bytesDownloaded` but `mu.Lock()` for `samples`?

If you can answer all 8, you've understood the codebase. If you can't, re-read the section that covers the question.
