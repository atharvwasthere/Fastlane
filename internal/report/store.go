// Package report persists bench.Result snapshots to disk and reads them back.
//
// Design: one JSON file per report, named for sortable lexical listing. The
// file format is intentionally identical to pkg/render/json.go output so a
// saved report and a fresh --json invocation are byte-equivalent (modulo
// non-zero metric ordering).
//
// The Store interface is small on purpose — Save/Load/List/Delete. A SQLite
// backend can slot in later without touching cmd/.
package report

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

// Store is the persistence contract. Implementations must be safe for
// concurrent use (FileStore relies on the filesystem for that).
type Store interface {
	Save(ctx context.Context, r bench.Result) (id string, err error)
	Load(ctx context.Context, id string) (bench.Result, error)
	List(ctx context.Context, filter Filter) ([]Meta, error)
	Delete(ctx context.Context, id string) error
}

// Filter narrows a List call.
type Filter struct {
	Kind  bench.Kind // empty = any
	Since time.Time  // zero = no lower bound
	Limit int        // 0 = unlimited
}

// Meta is the lightweight listing record — never loads the full Result body.
// Header-only parse keeps `report list` cheap even with thousands of files.
type Meta struct {
	ID        string     `json:"id"`
	Kind      bench.Kind `json:"kind"`
	Server    string     `json:"server"`
	Timestamp time.Time  `json:"timestamp"`
	Path      string     `json:"path"`
}

// envelope mirrors pkg/render/json.go output. Reports persist exactly what
// --json would have printed.
type envelope struct {
	Version         string             `json:"version"`
	Command         string             `json:"command"`
	Timestamp       string             `json:"timestamp"`
	Server          string             `json:"server"`
	DurationSeconds float64            `json:"duration_seconds"`
	Metrics         map[string]float64 `json:"metrics"`
	Counters        map[string]int64   `json:"counters"`
	Flags           map[string]bool    `json:"flags"`
	Layers          map[string]float64 `json:"layers,omitempty"`
	Notes           []string           `json:"notes"`
}

// FileStore writes JSON reports under root (defaults to ~/.fastlane/reports).
type FileStore struct {
	root string
}

// NewFileStore returns a FileStore. Empty root resolves to ~/.fastlane/reports.
func NewFileStore(root string) *FileStore {
	if root == "" {
		root = defaultRoot()
	}
	return &FileStore{root: root}
}

// Root exposes the resolved storage directory (used in `report list` headers).
func (s *FileStore) Root() string { return s.root }

func defaultRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".fastlane", "reports")
	}
	return filepath.Join(home, ".fastlane", "reports")
}

// Save serializes r into a JSON file and returns its id (filename stem).
// Filename: YYYY-MM-DDTHH-MM-SS_<kind>_<short-server-host>.json
func (s *FileStore) Save(ctx context.Context, r bench.Result) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return "", fmt.Errorf("report: mkdir %s: %w", s.root, err)
	}

	ts := r.StartedAt
	if ts.IsZero() {
		ts = time.Now()
	}
	id := buildID(ts.UTC(), r.Kind, r.Server)
	path := filepath.Join(s.root, id+".json")

	env := envelope{
		Version:         "2.0.0",
		Command:         string(r.Kind),
		Timestamp:       ts.UTC().Format(time.RFC3339),
		Server:          r.Server,
		DurationSeconds: r.Duration.Seconds(),
		Metrics:         nilToEmptyF(r.Metrics),
		Counters:        nilToEmptyI(r.Counters),
		Flags:           nilToEmptyB(r.Flags),
		Layers:          r.Layers,
		Notes:           nilToEmptyS(r.Notes),
	}
	buf, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return "", fmt.Errorf("report: marshal: %w", err)
	}
	// Append trailing newline for grep/tail friendliness.
	buf = append(buf, '\n')
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return "", fmt.Errorf("report: write %s: %w", path, err)
	}
	return id, nil
}

// Load reads the report at id and returns the rehydrated bench.Result.
func (s *FileStore) Load(ctx context.Context, id string) (bench.Result, error) {
	if err := ctx.Err(); err != nil {
		return bench.Result{}, err
	}
	path := filepath.Join(s.root, id+".json")
	buf, err := os.ReadFile(path)
	if err != nil {
		return bench.Result{}, fmt.Errorf("report: read %s: %w", path, err)
	}
	var env envelope
	if err := json.Unmarshal(buf, &env); err != nil {
		return bench.Result{}, fmt.Errorf("report: unmarshal %s: %w", path, err)
	}
	t, _ := time.Parse(time.RFC3339, env.Timestamp)
	return bench.Result{
		Kind:      bench.Kind(env.Command),
		Server:    env.Server,
		StartedAt: t,
		Duration:  time.Duration(env.DurationSeconds * float64(time.Second)),
		Metrics:   nilToEmptyF(env.Metrics),
		Counters:  nilToEmptyI(env.Counters),
		Flags:     nilToEmptyB(env.Flags),
		Layers:    env.Layers,
		Notes:     nilToEmptyS(env.Notes),
	}, nil
}

// List walks the report directory and returns matching Meta records, sorted
// newest first.
func (s *FileStore) List(ctx context.Context, filter Filter) ([]Meta, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("report: readdir %s: %w", s.root, err)
	}

	out := make([]Meta, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		path := filepath.Join(s.root, e.Name())
		m, ok := readHeader(path)
		if !ok {
			continue
		}
		if filter.Kind != "" && m.Kind != filter.Kind {
			continue
		}
		if !filter.Since.IsZero() && m.Timestamp.Before(filter.Since) {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

// Delete removes id's file from disk. Missing files report ErrNotExist.
func (s *FileStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path := filepath.Join(s.root, id+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("report: delete %s: %w", path, err)
	}
	return nil
}

// readHeader extracts the minimal Meta from a report file without loading the
// metric maps. Cheap enough to scale to thousands of files.
func readHeader(path string) (Meta, bool) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, false
	}
	// Tiny header struct — json decoder ignores the heavy maps.
	var h struct {
		Command   string `json:"command"`
		Timestamp string `json:"timestamp"`
		Server    string `json:"server"`
	}
	if err := json.Unmarshal(buf, &h); err != nil {
		return Meta{}, false
	}
	t, _ := time.Parse(time.RFC3339, h.Timestamp)
	id := strings.TrimSuffix(filepath.Base(path), ".json")
	return Meta{
		ID:        id,
		Kind:      bench.Kind(h.Command),
		Server:    h.Server,
		Timestamp: t,
		Path:      path,
	}, true
}

// buildID composes a filesystem-safe, lexically sortable id.
func buildID(ts time.Time, kind bench.Kind, server string) string {
	stamp := ts.Format("2006-01-02T15-04-05")
	k := string(kind)
	if k == "" {
		k = "unknown"
	}
	host := shortHost(server)
	if host == "" {
		return fmt.Sprintf("%s_%s", stamp, k)
	}
	return fmt.Sprintf("%s_%s_%s", stamp, k, host)
}

// shortHost extracts a filename-safe host fragment from a server URL/host.
func shortHost(server string) string {
	if server == "" {
		return ""
	}
	s := server
	if u, err := url.Parse(server); err == nil && u.Host != "" {
		s = u.Host
	}
	// Strip port and replace any non-friendly chars.
	if i := strings.IndexByte(s, ':'); i >= 0 {
		s = s[:i]
	}
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, " ", "-")
	// Filesystem-safe: keep alnum, dot, dash.
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}

func nilToEmptyF(m map[string]float64) map[string]float64 {
	if m == nil {
		return map[string]float64{}
	}
	return m
}
func nilToEmptyI(m map[string]int64) map[string]int64 {
	if m == nil {
		return map[string]int64{}
	}
	return m
}
func nilToEmptyB(m map[string]bool) map[string]bool {
	if m == nil {
		return map[string]bool{}
	}
	return m
}
func nilToEmptyS(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
