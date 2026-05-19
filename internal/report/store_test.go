package report

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

func sample(kind bench.Kind, server string, at time.Time) bench.Result {
	r := bench.NewResult(kind, server, at, 5*time.Second)
	r.Metrics["latency_ms"] = 42.5
	r.Metrics["download_mbps"] = 123.4
	r.Counters["samples"] = 7
	r.Flags["converged"] = true
	r.Notes = append(r.Notes, "ok")
	return r
}

func TestFileStore_SaveLoad_RoundTrip(t *testing.T) {
	s := NewFileStore(t.TempDir())
	ctx := context.Background()

	in := sample(bench.KindDownload, "https://speed.cloudflare.com/__down", time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC))
	id, err := s.Save(ctx, in)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if id == "" {
		t.Fatal("empty id")
	}

	got, err := s.Load(ctx, id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// Compare field-by-field — duration may lose sub-second precision via JSON float.
	if got.Kind != in.Kind || got.Server != in.Server {
		t.Errorf("kind/server mismatch: got %+v want %+v", got, in)
	}
	if !got.StartedAt.Equal(in.StartedAt) {
		t.Errorf("started: got %v want %v", got.StartedAt, in.StartedAt)
	}
	if !reflect.DeepEqual(got.Metrics, in.Metrics) {
		t.Errorf("metrics: got %v want %v", got.Metrics, in.Metrics)
	}
	if !reflect.DeepEqual(got.Counters, in.Counters) {
		t.Errorf("counters: got %v want %v", got.Counters, in.Counters)
	}
	if !reflect.DeepEqual(got.Flags, in.Flags) {
		t.Errorf("flags: got %v want %v", got.Flags, in.Flags)
	}
	if !reflect.DeepEqual(got.Notes, in.Notes) {
		t.Errorf("notes: got %v want %v", got.Notes, in.Notes)
	}
}

func TestFileStore_List_KindAndSinceFilter(t *testing.T) {
	s := NewFileStore(t.TempDir())
	ctx := context.Background()

	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		kind bench.Kind
		at   time.Time
	}{
		{bench.KindPing, old},
		{bench.KindDownload, mid},
		{bench.KindUpload, now},
		{bench.KindDownload, now},
	}
	for _, c := range cases {
		if _, err := s.Save(ctx, sample(c.kind, "speed.cloudflare.com", c.at)); err != nil {
			t.Fatalf("save: %v", err)
		}
	}

	tests := []struct {
		name   string
		filter Filter
		want   int
	}{
		{"no filter", Filter{}, 4},
		{"kind=download", Filter{Kind: bench.KindDownload}, 2},
		{"kind=ping", Filter{Kind: bench.KindPing}, 1},
		{"since mid", Filter{Since: mid}, 3},
		{"since now", Filter{Since: now}, 2},
		{"kind+since", Filter{Kind: bench.KindDownload, Since: now}, 1},
		{"limit", Filter{Limit: 2}, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.List(ctx, tc.filter)
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			if len(got) != tc.want {
				t.Errorf("len=%d want %d (got %+v)", len(got), tc.want, got)
			}
		})
	}
}

func TestFileStore_List_SortedNewestFirst(t *testing.T) {
	s := NewFileStore(t.TempDir())
	ctx := context.Background()
	times := []time.Time{
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	for _, ts := range times {
		if _, err := s.Save(ctx, sample(bench.KindPing, "h", ts)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.List(ctx, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len=%d", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Timestamp.Before(got[i].Timestamp) {
			t.Errorf("not sorted desc: %v before %v", got[i-1].Timestamp, got[i].Timestamp)
		}
	}
}

func TestFileStore_Delete(t *testing.T) {
	s := NewFileStore(t.TempDir())
	ctx := context.Background()
	id, err := s.Save(ctx, sample(bench.KindLoss, "h", time.Now().UTC()))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Load(ctx, id); err == nil {
		t.Error("expected load after delete to fail")
	} else if !errors.Is(err, os.ErrNotExist) {
		// Wrapped, that's fine — just sanity check.
	}
	list, _ := s.List(ctx, Filter{})
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}
}

func TestFileStore_List_EmptyDir(t *testing.T) {
	s := NewFileStore(t.TempDir())
	got, err := s.List(context.Background(), Filter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestFileStore_List_MissingDir(t *testing.T) {
	s := NewFileStore(t.TempDir() + "-does-not-exist")
	got, err := s.List(context.Background(), Filter{})
	if err != nil {
		t.Fatalf("list missing dir: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestShortHost(t *testing.T) {
	cases := map[string]string{
		"https://speed.cloudflare.com/__down":         "speed.cloudflare.com",
		"https://speed.cloudflare.com:443/__down":     "speed.cloudflare.com",
		"google.com":                                  "google.com",
		"":                                            "",
		"127.0.0.1:9":                                 "127.0.0.1",
	}
	for in, want := range cases {
		if got := shortHost(in); got != want {
			t.Errorf("shortHost(%q)=%q want %q", in, got, want)
		}
	}
}
