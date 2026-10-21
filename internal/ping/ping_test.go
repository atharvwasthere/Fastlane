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
		t.Fatalf("expected an error, got mean %.2f ms", res.MeanMS)
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
	if res.TCPLatencyMS <= 0 || res.TCPLatencyMS >= 30 {
		t.Fatalf("TCP connect %.2f ms should be loopback-fast and exclude server time", res.TCPLatencyMS)
	}
	if res.HTTPLatencyMS < 30 {
		t.Fatalf("TTFB %.2f ms should include the 30 ms server delay", res.HTTPLatencyMS)
	}
	if res.TLSLatencyMS != 0 || res.DNSLatencyMS != 0 {
		t.Fatalf("plain-HTTP IP target has no TLS or DNS, got tls=%.2f dns=%.2f", res.TLSLatencyMS, res.DNSLatencyMS)
	}
	if res.MeanMS != res.TCPLatencyMS {
		t.Fatalf("latency must be the TCP connect mean: mean=%.2f tcp=%.2f", res.MeanMS, res.TCPLatencyMS)
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
