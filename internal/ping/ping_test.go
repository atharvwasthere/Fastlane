package ping

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMeasureLayeredUnreachableIsError(t *testing.T) {
	// .invalid never resolves (RFC 2606), so every layer fails.
	res, err := MeasureLayered(context.Background(), "no-such-host.invalid", 1, time.Second)
	if err == nil {
		t.Fatalf("expected an error, got median %.2f ms", res.MedianMS)
	}
}

func TestMeasureLayeredSeparatesLayers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond) // server think time belongs to TTFB, never to TCP
	}))
	defer srv.Close()

	res, err := MeasureLayered(context.Background(), srv.URL, 3, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Samples) != 3 {
		t.Fatalf("want one TCP sample per probe (fresh connections), got %d", len(res.Samples))
	}
	if res.TCPLatencyMS < 0 || res.TCPLatencyMS >= 30 { // 0 is valid: Windows' clock can't resolve a loopback connect
		t.Fatalf("TCP connect %.2f ms should be loopback-fast and exclude server time", res.TCPLatencyMS)
	}
	if res.HTTPLatencyMS < 30 {
		t.Fatalf("TTFB %.2f ms should include the 30 ms server delay", res.HTTPLatencyMS)
	}
	if res.TLSLatencyMS != 0 || res.DNSLatencyMS != 0 {
		t.Fatalf("plain-HTTP IP target has no TLS or DNS, got tls=%.2f dns=%.2f", res.TLSLatencyMS, res.DNSLatencyMS)
	}
	if res.MedianMS != res.TCPLatencyMS {
		t.Fatalf("latency must be the TCP connect median: median=%.2f tcp=%.2f", res.MedianMS, res.TCPLatencyMS)
	}
}

func TestMeasureLayeredHonorsContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := MeasureLayered(ctx, srv.URL, 20, 5*time.Second)
	if err == nil {
		t.Fatal("expected a context error")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("cancel took %v; 20 probes should stop at the 300ms deadline", elapsed)
	}
}

func TestTargetURL(t *testing.T) {
	cases := map[string]string{
		"google.com":      "https://google.com/",
		"google.com:8443": "https://google.com:8443/",
		"https://speed.cloudflare.com/__down?bytes=1": "https://speed.cloudflare.com/",
		"http://192.168.1.1":                          "http://192.168.1.1/",
		"[::1]:8080":                                  "https://[::1]:8080/",
	}
	for in, want := range cases {
		got, err := targetURL(in)
		if err != nil || got != want {
			t.Errorf("targetURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestRobustStatsOnSkewedLatency(t *testing.T) {
	// One queueing spike: the mean would read 57.2 ms, the median stays on the typical 22 ms.
	samples := []float64{20, 21, 22, 23, 200}
	if got := median(samples); got != 22 {
		t.Fatalf("median = %.1f, want 22", got)
	}
	if got := median([]float64{30, 10, 20, 40}); got != 25 {
		t.Fatalf("even-length median = %.1f, want 25", got)
	}
	if got := meanAbsDiff(samples); got != 45 {
		t.Fatalf("meanAbsDiff = %.1f, want 45 ((1+1+1+177)/4)", got)
	}
	if samples[4] != 200 {
		t.Fatal("median must not reorder the caller's samples")
	}
}
