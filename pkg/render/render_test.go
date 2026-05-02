package render

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// updateGolden lets us refresh fixtures with `go test ./pkg/render -update`.
var updateGolden = flag.Bool("update", false, "rewrite testdata/*.golden")

func init() {
	// Force ascii so goldens are deterministic across machines/terms.
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(os.Stdout, termenv.WithProfile(termenv.Ascii)))
}

func sampleResult(kind bench.Kind) bench.Result {
	started := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	dur := 8 * time.Second
	r := bench.NewResult(kind, "speed.example.com", started, dur)
	switch kind {
	case bench.KindPing:
		r.Server = "cloudflare.com"
		r.Metrics["latency_ms"] = 22
		r.Metrics["jitter_ms"] = 3
		r.Metrics["min_ms"] = 18
		r.Metrics["max_ms"] = 51
		r.Layers["dns_ms"] = 1.2
		r.Layers["tcp_ms"] = 12
		r.Layers["tls_ms"] = 24
		r.Layers["http_ms"] = 38
		r.Counters["samples"] = 10
		r.Counters["outliers_removed"] = 1
	case bench.KindDownload:
		r.Server = "https://speed.cloudflare.com/__down"
		r.Metrics["final_mbps"] = 243.5
		r.Metrics["mean_mbps"] = 240.1
		r.Metrics["ewma_mbps"] = 238.1
		r.Metrics["stddev_mbps"] = 5.2
		r.Metrics["min_mbps"] = 220
		r.Metrics["max_mbps"] = 251
		r.Metrics["convergence_cv"] = 0.021
		r.Counters["bytes"] = 255_852_544
		r.Counters["threads"] = 4
		r.Counters["samples"] = 22
		r.Flags["converged"] = true
	case bench.KindUpload:
		r.Server = "https://speed.cloudflare.com/__up"
		r.Metrics["final_mbps"] = 38.2
		r.Metrics["mean_mbps"] = 37.5
		r.Metrics["ewma_mbps"] = 37.9
		r.Metrics["stddev_mbps"] = 1.1
		r.Metrics["min_mbps"] = 35
		r.Metrics["max_mbps"] = 41
		r.Metrics["convergence_cv"] = 0.029
		r.Counters["bytes"] = 41_000_000
		r.Counters["threads"] = 4
		r.Counters["samples"] = 18
		r.Flags["converged"] = true
	case bench.KindLoss:
		r.Server = "8.8.8.8"
		r.Metrics["loss_percent"] = 0.0
		r.Metrics["jitter_ms"] = 1.4
		r.Counters["packets_sent"] = 100
		r.Counters["packets_received"] = 100
		r.Counters["packets_lost"] = 0
		r.Flags["test_complete"] = true
	}
	return r
}

func TestCardWidthSweep(t *testing.T) {
	widths := []int{30, 50, 70, 90}
	for _, w := range widths {
		w := w
		t.Run(fmt.Sprintf("ping_w%d", w), func(t *testing.T) {
			var buf bytes.Buffer
			r := New(FormatCard, &buf, Options{Width: w})
			r.Final(sampleResult(bench.KindPing))
			assertGolden(t, fmt.Sprintf("ping_w%d.golden", w), buf.Bytes())
			assertNoLineExceeds(t, buf.String(), w)
		})
	}
}

func TestCardKinds(t *testing.T) {
	kinds := []bench.Kind{bench.KindDownload, bench.KindUpload, bench.KindLoss}
	for _, k := range kinds {
		k := k
		t.Run(string(k), func(t *testing.T) {
			var buf bytes.Buffer
			r := New(FormatCard, &buf, Options{Width: 80})
			r.Final(sampleResult(k))
			assertGolden(t, fmt.Sprintf("%s_w80.golden", k), buf.Bytes())
			assertNoLineExceeds(t, buf.String(), 80)
		})
	}
}

func TestJSONSchema(t *testing.T) {
	var buf bytes.Buffer
	r := New(FormatJSON, &buf, Options{})
	r.Final(sampleResult(bench.KindDownload))

	var env map[string]any
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, k := range []string{"version", "command", "timestamp", "server", "duration_seconds", "metrics", "counters", "flags", "notes"} {
		if _, ok := env[k]; !ok {
			t.Errorf("missing required key %q", k)
		}
	}
	if env["version"] != "2.0.0" {
		t.Errorf("version=%v, want 2.0.0", env["version"])
	}
}

// TestCardJSONParity guards against drift: every metric key the card sources
// (latency_ms, final_mbps, etc.) must round-trip through the JSON renderer.
// The test asserts that the JSON envelope contains a non-empty "metrics" map
// and that all keys we put on the Result come back out.
func TestCardJSONParity(t *testing.T) {
	res := sampleResult(bench.KindDownload)

	var jsonBuf bytes.Buffer
	New(FormatJSON, &jsonBuf, Options{}).Final(res)

	var env struct {
		Metrics  map[string]float64 `json:"metrics"`
		Counters map[string]int64   `json:"counters"`
		Flags    map[string]bool    `json:"flags"`
	}
	if err := json.Unmarshal(jsonBuf.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for k, v := range res.Metrics {
		if got, ok := env.Metrics[k]; !ok || got != v {
			t.Errorf("metrics.%s missing or mismatch in JSON: got %v ok=%v", k, got, ok)
		}
	}
	for k, v := range res.Counters {
		if got, ok := env.Counters[k]; !ok || got != v {
			t.Errorf("counters.%s missing or mismatch in JSON: got %v ok=%v", k, got, ok)
		}
	}
	for k, v := range res.Flags {
		if got, ok := env.Flags[k]; !ok || got != v {
			t.Errorf("flags.%s missing or mismatch in JSON: got %v ok=%v", k, got, ok)
		}
	}
}

func TestPromExposition(t *testing.T) {
	var buf bytes.Buffer
	New(FormatProm, &buf, Options{}).Final(sampleResult(bench.KindDownload))
	out := buf.String()
	for _, want := range []string{
		"# HELP fastlane_download_final_mbps",
		"# TYPE fastlane_download_final_mbps gauge",
		`fastlane_download_final_mbps{server="https://speed.cloudflare.com/__down"} 243.5`,
		`fastlane_download_converged{server="https://speed.cloudflare.com/__down"} 1`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("prom output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

// helpers

func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run `go test ./pkg/render -update` to create)", path, err)
	}
	if !bytes.Equal(normalize(got), normalize(want)) {
		t.Errorf("golden mismatch %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

// normalize collapses CRLF→LF and trims trailing whitespace per line so the
// goldens survive being checked in on Windows.
func normalize(b []byte) []byte {
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return []byte(strings.Join(lines, "\n"))
}

func assertNoLineExceeds(t *testing.T, s string, width int) {
	t.Helper()
	for i, line := range strings.Split(s, "\n") {
		if w := lipgloss.Width(line); w > width {
			t.Errorf("line %d width=%d exceeds %d: %q", i, w, width, line)
		}
	}
}
