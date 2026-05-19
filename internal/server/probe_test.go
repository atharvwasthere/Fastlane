package server

import (
	"context"
	"net"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestProbeAll_KnownUpAndDown(t *testing.T) {
	srv := httptest.NewServer(nil)
	defer srv.Close()

	// Parse host:port from the test server URL.
	u := strings.TrimPrefix(srv.URL, "http://")
	host, portStr, err := net.SplitHostPort(u)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	port, _ := strconv.Atoi(portStr)

	servers := []Server{
		{ID: "up", Host: host, Port: port, Latitude: 1, Longitude: 1},
		{ID: "down", Host: "127.0.0.1", Port: 1, Latitude: 1, Longitude: 1}, // port 1 should refuse
	}
	out := ProbeAll(context.Background(), servers, 4)
	if len(out) != 2 {
		t.Fatalf("len(out) = %d", len(out))
	}
	if !out[0].Available {
		t.Errorf("expected up server available, got %+v", out[0])
	}
	if out[1].Available {
		t.Errorf("expected down server unavailable, got %+v", out[1])
	}
}

func TestProbeAll_ConcurrencyCap(t *testing.T) {
	// 20 unreachable servers with concurrency 4 should still return 20 entries.
	servers := make([]Server, 20)
	for i := range servers {
		servers[i] = Server{ID: "x", Host: "127.0.0.1", Port: 1}
	}
	out := ProbeAll(context.Background(), servers, 4)
	if len(out) != 20 {
		t.Fatalf("len(out) = %d, want 20", len(out))
	}
}
