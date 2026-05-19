package server

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

//go:embed data/servers.json
var embeddedList []byte

// CacheMaxAge is the freshness window for ~/.fastlane/servers.json. Older
// caches are ignored in favor of the embedded list.
const CacheMaxAge = 30 * 24 * time.Hour

// LoadEmbedded returns the compiled-in list.
func LoadEmbedded() (*ServerList, error) {
	var sl ServerList
	if err := json.Unmarshal(embeddedList, &sl); err != nil {
		return nil, fmt.Errorf("embedded servers.json: %w", err)
	}
	return &sl, nil
}

// LoadCachedOrEmbedded prefers a fresh ~/.fastlane/servers.json (<30d old);
// falls back to the embedded list otherwise. Errors reading the cache are
// non-fatal — we just return the embedded copy.
func LoadCachedOrEmbedded() (*ServerList, error) {
	path, err := cachePath()
	if err == nil {
		if info, statErr := os.Stat(path); statErr == nil {
			if time.Since(info.ModTime()) < CacheMaxAge {
				data, readErr := os.ReadFile(path)
				if readErr == nil {
					var sl ServerList
					if jsonErr := json.Unmarshal(data, &sl); jsonErr == nil && len(sl.Servers) > 0 {
						return &sl, nil
					}
				}
			}
		}
	}
	return LoadEmbedded()
}

// SaveCache writes the list to ~/.fastlane/servers.json. Used by
// `fastlane servers update`.
func SaveCache(sl *ServerList) error {
	path, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sl, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func cachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fastlane", "servers.json"), nil
}
