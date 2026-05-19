package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/logging"
)

const (
	autoServerCacheMaxAge = 1 * time.Hour
	shortlistSize         = 5
)

// AutoSelect picks the best server for the user via:
//  1. geoip lookup → user lat/lon
//  2. sort embedded/cached list by haversine distance
//  3. take top 5 + always-on anycast anchors
//  4. parallel TCP-probe shortlist
//  5. lowest-RTT wins (caches decision for 1h)
//
// Errors short-circuit at any step; callers fall back to the static default.
func AutoSelect(ctx context.Context) (*Server, error) {
	log := logging.FromContext(ctx)
	if s, ok := readSelectionCache(); ok {
		log.Debug("auto-select: cache hit", "host", s.Host, "name", s.Name)
		return s, nil
	}

	sl, err := LoadCachedOrEmbedded()
	if err != nil {
		return nil, fmt.Errorf("auto-select: load: %w", err)
	}
	if len(sl.Servers) == 0 {
		return nil, fmt.Errorf("auto-select: empty server list")
	}

	loc, err := DetectLocation(ctx)
	if err != nil {
		return nil, fmt.Errorf("auto-select: geoip: %w", err)
	}

	shortlist := shortlistByDistance(sl.Servers, loc.Latitude, loc.Longitude, shortlistSize)
	log.Debug("auto-select: shortlist built",
		"size", len(shortlist), "lat", loc.Latitude, "lon", loc.Longitude)
	metrics := ProbeAll(ctx, shortlist, defaultProbeConcurrency)
	SortByLatency(metrics)
	for _, m := range metrics {
		if m == nil {
			continue
		}
		log.Debug("auto-select: probe", "host", m.Server.Host,
			"rtt_ms", m.LatencyMS, "available", m.Available)
	}

	for _, m := range metrics {
		if m != nil && m.Available {
			_ = writeSelectionCache(m.Server)
			log.Info("auto-select: chose server",
				"host", m.Server.Host, "name", m.Server.Name,
				"location", m.Server.Location, "rtt_ms", m.LatencyMS)
			return m.Server, nil
		}
	}
	return nil, fmt.Errorf("auto-select: no reachable server in shortlist")
}

// shortlistByDistance picks the N nearest servers, always keeping anycast
// anchors (lat=lon=0) so they survive the cull as last-resort fallbacks.
func shortlistByDistance(servers []Server, lat, lon float64, n int) []Server {
	type pair struct {
		s    Server
		dist float64
	}
	pairs := make([]pair, 0, len(servers))
	anycast := make([]Server, 0)
	for _, s := range servers {
		if s.Latitude == 0 && s.Longitude == 0 {
			anycast = append(anycast, s)
			continue
		}
		pairs = append(pairs, pair{s: s, dist: HaversineDistance(lat, lon, s.Latitude, s.Longitude)})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].dist < pairs[j].dist })

	out := make([]Server, 0, n+len(anycast))
	for i := 0; i < len(pairs) && i < n; i++ {
		out = append(out, pairs[i].s)
	}
	out = append(out, anycast...)
	return out
}

func selectionCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fastlane", "cache", "server.json"), nil
}

func readSelectionCache() (*Server, bool) {
	path, err := selectionCachePath()
	if err != nil {
		return nil, false
	}
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > autoServerCacheMaxAge {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var s Server
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, false
	}
	return &s, true
}

func writeSelectionCache(s *Server) error {
	path, err := selectionCachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
