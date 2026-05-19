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
		CVThreshold:    0.5, // loose so probes finish quickly
		UpdateInterval: 50 * time.Millisecond,
		MinSamples:     2,
	})
	cfg.StartThreads = 2
	cfg.StepThreads = 2
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
