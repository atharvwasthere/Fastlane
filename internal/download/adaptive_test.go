package download

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// largePayload serves bytes forever until the client disconnects.
func largePayloadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		buf := make([]byte, 64*1024)
		flusher, _ := w.(http.Flusher)
		for {
			if _, err := w.Write(buf); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
			if r.Context().Err() != nil {
				return
			}
		}
	}
}

func TestAdaptive_RunsSweepAndFinalRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	srv := httptest.NewServer(largePayloadHandler())
	defer srv.Close()

	cfg := DefaultAdaptive(Config{
		URL:            srv.URL,
		RSEThreshold:    0.5, // loose so probes finish quickly
		UpdateInterval: 50 * time.Millisecond,
		MinSamples:     2,
	})
	cfg.StartThreads = 2
	cfg.MaxThreads = 4
	cfg.ProbeDuration = 400 * time.Millisecond
	cfg.FinalRunDuration = 400 * time.Millisecond

	eng := NewAdaptiveBenchEngine(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	res, err := eng.Run(ctx)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Notes) == 0 {
		t.Fatalf("expected adaptive Note, got none")
	}
	// "adaptive: N threads (sweep S→E)"
	found := false
	for _, n := range res.Notes {
		if len(n) > 0 && n[:9] == "adaptive:" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("missing adaptive note in %v", res.Notes)
	}
	if res.Metrics["final_mbps"] <= 0 {
		t.Errorf("final_mbps = %v, want > 0", res.Metrics["final_mbps"])
	}
}

// Sanity: ensure compile-time interface compliance.
var _ = io.ReadAll
var _ = httptest.NewServer

// curve fakes a throughput-vs-threads measurement.
func curve(m map[int]float64) func(int) (float64, error) {
	return func(threads int) (float64, error) { return m[threads], nil }
}

func TestClimbSurvivesOneNoisyProbe(t *testing.T) {
	c := withDefaults(AdaptiveConfig{})
	// 2 threads looks flat (noise), but the real knee is at 4. One-step-and-stop picked 1.
	best, last := climb(c, curve(map[int]float64{1: 10, 2: 10.2, 4: 20, 8: 20.5, 16: 19}))
	if best != 4 || last != 16 {
		t.Fatalf("best=%d last=%d, want best=4 after probing through 16", best, last)
	}
}

func TestClimbKeepsOneThreadWhenMoreHurts(t *testing.T) {
	c := withDefaults(AdaptiveConfig{})
	// A slow link that degrades with more streams: the answer is below the old start of 2.
	best, last := climb(c, curve(map[int]float64{1: 8, 2: 7, 4: 6, 8: 5}))
	if best != 1 || last != 4 {
		t.Fatalf("best=%d last=%d, want best=1, stopping after 2 misses at 4", best, last)
	}
}

func TestClimbStopsOnProbeError(t *testing.T) {
	c := withDefaults(AdaptiveConfig{})
	best, last := climb(c, func(threads int) (float64, error) {
		if threads == 4 {
			return 0, context.Canceled
		}
		return float64(threads * 10), nil
	})
	if best != 2 || last != 2 {
		t.Fatalf("best=%d last=%d, want 2/2", best, last)
	}
}
