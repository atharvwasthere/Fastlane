package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	geoipURL         = "https://ipapi.co/json/"
	geoipCacheMaxAge = 24 * time.Hour
)

// geoipResponse mirrors the ipapi.co /json/ shape (subset we use).
type geoipResponse struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Continent   string  `json:"continent_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Org         string  `json:"org"`
}

// DetectLocation returns the user's approximate location, preferring a fresh
// 24h cache over the network. Configured to fail fast (3s timeout) so callers
// using --auto-server don't hang on a flaky upstream.
func DetectLocation(ctx context.Context) (*UserLocation, error) {
	if loc, ok := readLocationCache(); ok {
		return loc, nil
	}
	loc, err := fetchLocation(ctx, geoipURL)
	if err != nil {
		return nil, err
	}
	_ = writeLocationCache(loc)
	return loc, nil
}

func fetchLocation(ctx context.Context, url string) (*UserLocation, error) {
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "fastlane-cli")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geoip: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	if err != nil {
		return nil, err
	}
	var g geoipResponse
	if err := json.Unmarshal(body, &g); err != nil {
		return nil, fmt.Errorf("geoip: %w", err)
	}
	return &UserLocation{
		IP:        g.IP,
		Latitude:  g.Latitude,
		Longitude: g.Longitude,
		ISP:       g.Org,
	}, nil
}

func locationCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fastlane", "cache", "location.json"), nil
}

func readLocationCache() (*UserLocation, bool) {
	path, err := locationCachePath()
	if err != nil {
		return nil, false
	}
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > geoipCacheMaxAge {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var loc UserLocation
	if err := json.Unmarshal(data, &loc); err != nil {
		return nil, false
	}
	return &loc, true
}

func writeLocationCache(loc *UserLocation) error {
	path, err := locationCachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(loc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
