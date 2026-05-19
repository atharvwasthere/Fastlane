package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// fileConfig mirrors GlobalFlags as a JSON-tagged optional document. All
// fields use pointer types so we can tell "absent" from "set to zero".
type fileConfig struct {
	JSON    *bool   `json:"json,omitempty"`
	Verbose *bool   `json:"verbose,omitempty"`
	Timeout *uint32 `json:"timeout,omitempty"`
	Debug   *bool   `json:"debug,omitempty"`
	NoColor *bool   `json:"no_color,omitempty"`
}

// ChangedFlags reports which GlobalFlags names the user actually set on the
// CLI. Pass this from cobra's PersistentPreRun via cmd.Flags().Changed(name).
type ChangedFlags struct {
	JSON    bool
	Verbose bool
	Timeout bool
	Debug   bool
	NoColor bool
}

// Load resolves global config with precedence:
//
//	flag (if Changed) > env (FASTLANE_*) > ~/.fastlane/config.json > defaults
//
// The caller pre-fills `flags` with whatever cobra parsed (already merged
// with cobra defaults). `changed` tells us which of those fields were
// actually user-set so file/env values can override unset zero values.
func Load(flags GlobalFlags, changed ChangedFlags) (GlobalFlags, error) {
	// Start from defaults.
	out := DefaultGlobalFlags()

	// Overlay file config (silent miss).
	fc, err := readFileConfig()
	if err != nil {
		return out, err
	}
	if fc != nil {
		if fc.JSON != nil {
			out.JSON = *fc.JSON
		}
		if fc.Verbose != nil {
			out.Verbose = *fc.Verbose
		}
		if fc.Timeout != nil {
			out.Timeout = *fc.Timeout
		}
		if fc.Debug != nil {
			out.Debug = *fc.Debug
		}
		if fc.NoColor != nil {
			out.NoColor = *fc.NoColor
		}
	}

	// Overlay env.
	if v, ok := envBool("FASTLANE_JSON"); ok {
		out.JSON = v
	}
	if v, ok := envBool("FASTLANE_VERBOSE"); ok {
		out.Verbose = v
	}
	if v, ok := envBool("FASTLANE_DEBUG"); ok {
		out.Debug = v
	}
	if v, ok := envBool("FASTLANE_NO_COLOR"); ok {
		out.NoColor = v
	}
	if s := os.Getenv("FASTLANE_TIMEOUT"); s != "" {
		n, perr := strconv.ParseUint(s, 10, 32)
		if perr != nil {
			return out, fmt.Errorf("invalid FASTLANE_TIMEOUT %q: %w", s, perr)
		}
		out.Timeout = uint32(n)
	}

	// Overlay user-set flags (highest precedence).
	if changed.JSON {
		out.JSON = flags.JSON
	}
	if changed.Verbose {
		out.Verbose = flags.Verbose
	}
	if changed.Timeout {
		out.Timeout = flags.Timeout
	}
	if changed.Debug {
		out.Debug = flags.Debug
	}
	if changed.NoColor {
		out.NoColor = flags.NoColor
	}

	return out, nil
}

// configPath returns the canonical config location. Falls back to "" if
// the home directory cannot be resolved — the caller treats that as "no
// file config available".
func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".fastlane", "config.json")
}

func readFileConfig() (*fileConfig, error) {
	path := configPath()
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &fc, nil
}

// envBool parses a FASTLANE_* boolean env var. Returns (value, true) when
// the env var is set; (false, false) when unset or empty. Accepted truthy:
// 1, true, t, yes, y (case-insensitive). Everything else parses as false.
func envBool(name string) (bool, bool) {
	s := os.Getenv(name)
	if s == "" {
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "t", "yes", "y":
		return true, true
	default:
		return false, true
	}
}
