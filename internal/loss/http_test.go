package loss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// TestHTTPEngineNearZeroLoss: server accepts every HEAD → loss ≈ 0.
func TestHTTPEngineNearZeroLoss(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("net/http.(*Server).Serve"),
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := NewHTTPEngine(Config{
		Host:    srv.URL,
		Count:   20,
		Rate:    50,
		Timeout: 5 * time.Second,
	})
	res, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.PacketsSent != 20 {
		t.Errorf("expected 20 sent, got %d", res.PacketsSent)
	}
	if res.LossPercent > 5 {
		t.Errorf("expected near-zero loss, got %.1f%%", res.LossPercent)
	}
	if res.AvgLatencyMS == 0 {
		t.Errorf("expected non-zero RTT avg")
	}
}

// TestHTTPEngineHalfLoss: server hangs half the requests past the per-probe
// 2s timeout → those count as loss.
func TestHTTPEngineHalfLoss(t *testing.T) {
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := atomic.AddInt32(&n, 1)
		if i%2 == 0 {
			// Hang past the 2s per-probe timeout — probe will deadline-exceed.
			time.Sleep(3 * time.Second)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := NewHTTPEngine(Config{
		Host:    srv.URL,
		Count:   10,
		Rate:    5, // slow rate so 2s deadlines have room to play out serially
		Timeout: 30 * time.Second,
	})
	res, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.PacketsSent != 10 {
		t.Errorf("expected 10 sent, got %d", res.PacketsSent)
	}
	// Every other probe times out → loss should be 30–70% range.
	if res.LossPercent < 30 || res.LossPercent > 70 {
		t.Errorf("expected ~50%% loss, got %.1f%%", res.LossPercent)
	}
}

// TestHTTPEngineCancellation: cancel ctx mid-test, ensure no panic and
// partial counts make sense.
func TestHTTPEngineCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	e := NewHTTPEngine(Config{
		Host:    srv.URL,
		Count:   1000,
		Rate:    10,
		Timeout: 30 * time.Second,
	})
	res, err := e.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.PacketsSent >= 1000 {
		t.Errorf("expected partial sends after cancel, got %d", res.PacketsSent)
	}
	if res.PacketsSent > 0 && res.PacketsReceived == 0 {
		t.Errorf("expected some receives if anything was sent")
	}
}

// TestHTTPEngineConnRefusedAsError: dial-error to a closed port → counts as
// errors, not loss.
func TestHTTPEngineConnRefusedAsError(t *testing.T) {
	// Port 1 is reserved/closed; dial fails fast.
	e := NewHTTPEngine(Config{
		Host:    "http://127.0.0.1:1",
		Count:   5,
		Rate:    20,
		Timeout: 5 * time.Second,
	})
	res, err := e.Run(context.Background())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.Errors == 0 {
		t.Errorf("expected non-zero errors on closed port, got %d", res.Errors)
	}
}

// TestNormalizeURL: bare host gets http://; existing scheme preserved.
func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"example.com":         "http://example.com",
		"http://x.com":        "http://x.com",
		"https://x.com/path":  "https://x.com/path",
		"127.0.0.1:8080":      "http://127.0.0.1:8080",
	}
	for in, want := range cases {
		if got := normalizeURL(in); got != want {
			t.Errorf("normalizeURL(%q) = %q, want %q", in, got, want)
		}
	}
}
