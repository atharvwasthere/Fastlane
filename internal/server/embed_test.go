package server

import (
	"strings"
	"testing"
)

func TestLoadEmbedded(t *testing.T) {
	sl, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	if sl.Version < 1 {
		t.Errorf("Version = %d, want ≥ 1", sl.Version)
	}
	if len(sl.Servers) < 20 {
		t.Errorf("embedded server count = %d, want ≥ 20", len(sl.Servers))
	}
	// Cloudflare anycast must be present — that's our default.
	var cfFound bool
	for _, s := range sl.Servers {
		if strings.Contains(s.Host, "speed.cloudflare.com") {
			cfFound = true
			break
		}
	}
	if !cfFound {
		t.Error("embedded list missing speed.cloudflare.com anchor")
	}
}

func TestEmbeddedServersHaveRequiredFields(t *testing.T) {
	sl, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	for _, s := range sl.Servers {
		if s.ID == "" || s.Host == "" || s.Port == 0 {
			t.Errorf("server %+v: missing required field (id/host/port)", s)
		}
	}
}
