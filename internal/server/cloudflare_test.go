package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseTrace(t *testing.T) {
	body := "fl=24f123\nh=speed.cloudflare.com\nip=1.2.3.4\nloc=IN\ncolo=DEL\nts=1700000000\n"
	got := parseTrace(body)
	if got.IP != "1.2.3.4" || got.Loc != "IN" || got.Colo != "DEL" {
		t.Fatalf("parseTrace = %+v", got)
	}
	if s := got.String(); s != "colo=DEL loc=IN" {
		t.Fatalf("String() = %q", s)
	}
}

func TestParseTracePartial(t *testing.T) {
	got := parseTrace("ip=10.0.0.1\nrandom=field\n")
	if got.IP != "10.0.0.1" || got.Loc != "" || got.Colo != "" {
		t.Fatalf("parseTrace = %+v", got)
	}
	if got.String() != "" {
		t.Fatalf("String() = %q, want empty", got.String())
	}
}

func TestDetectCloudflareColoOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("colo=BLR\nloc=IN\nip=9.9.9.9\n"))
	}))
	defer srv.Close()

	c, err := detectColoFrom(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("detectColoFrom: %v", err)
	}
	if c.Colo != "BLR" || c.Loc != "IN" {
		t.Fatalf("got %+v", c)
	}
}

func TestDetectCloudflareColoNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()
	if _, err := detectColoFrom(context.Background(), srv.URL); err == nil {
		t.Fatal("expected error on 503")
	}
}
