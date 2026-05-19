package config

import (
	"os"
	"path/filepath"
	"testing"
)

// withHome sets HOME (and USERPROFILE on Windows) to a temp dir for the
// duration of the test so config lookups land in a sandbox. Returns the
// sandbox path so callers can drop a config.json into it.
func withHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
	return dir
}

// clearEnv wipes the FASTLANE_* env vars so prior test runs (or CI shells)
// don't leak in.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"FASTLANE_JSON", "FASTLANE_VERBOSE", "FASTLANE_DEBUG",
		"FASTLANE_NO_COLOR", "FASTLANE_TIMEOUT",
	} {
		t.Setenv(k, "")
	}
}

func TestLoadDefaultsWhenNothingSet(t *testing.T) {
	withHome(t)
	clearEnv(t)

	got, err := Load(GlobalFlags{}, ChangedFlags{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := DefaultGlobalFlags()
	if got != want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func TestLoadMissingFileIsSilent(t *testing.T) {
	withHome(t)
	clearEnv(t)
	// no config.json on disk — should not error.
	if _, err := Load(GlobalFlags{}, ChangedFlags{}); err != nil {
		t.Fatalf("expected nil err on missing file, got %v", err)
	}
}

func TestLoadMalformedFileReturnsError(t *testing.T) {
	home := withHome(t)
	clearEnv(t)
	if err := os.MkdirAll(filepath.Join(home, ".fastlane"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".fastlane", "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(GlobalFlags{}, ChangedFlags{}); err == nil {
		t.Fatal("expected error on malformed config")
	}
}

func TestLoadFileBeatsDefaults(t *testing.T) {
	home := withHome(t)
	clearEnv(t)
	if err := os.MkdirAll(filepath.Join(home, ".fastlane"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".fastlane", "config.json")
	body := `{"debug": true, "timeout": 99, "no_color": true}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(GlobalFlags{}, ChangedFlags{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !got.Debug || got.Timeout != 99 || !got.NoColor {
		t.Errorf("file-precedence Load() = %+v", got)
	}
}

func TestLoadEnvBeatsFile(t *testing.T) {
	home := withHome(t)
	clearEnv(t)
	if err := os.MkdirAll(filepath.Join(home, ".fastlane"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"debug": false, "timeout": 10}`
	_ = os.WriteFile(filepath.Join(home, ".fastlane", "config.json"), []byte(body), 0o644)

	t.Setenv("FASTLANE_DEBUG", "true")
	t.Setenv("FASTLANE_TIMEOUT", "77")

	got, err := Load(GlobalFlags{}, ChangedFlags{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !got.Debug {
		t.Errorf("env should beat file: Debug=%v", got.Debug)
	}
	if got.Timeout != 77 {
		t.Errorf("env should beat file: Timeout=%d", got.Timeout)
	}
}

func TestLoadFlagBeatsEnv(t *testing.T) {
	withHome(t)
	clearEnv(t)
	t.Setenv("FASTLANE_DEBUG", "true")
	t.Setenv("FASTLANE_TIMEOUT", "77")

	flags := GlobalFlags{Debug: false, Timeout: 5}
	changed := ChangedFlags{Debug: true, Timeout: true}

	got, err := Load(flags, changed)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Debug {
		t.Errorf("flag should beat env: Debug=%v", got.Debug)
	}
	if got.Timeout != 5 {
		t.Errorf("flag should beat env: Timeout=%d", got.Timeout)
	}
}

func TestLoadEnvBoolForms(t *testing.T) {
	withHome(t)

	truthy := []string{"1", "true", "TRUE", "t", "yes", "y", "Yes"}
	for _, s := range truthy {
		clearEnv(t)
		t.Setenv("FASTLANE_DEBUG", s)
		got, err := Load(GlobalFlags{}, ChangedFlags{})
		if err != nil {
			t.Fatalf("Load(%q): %v", s, err)
		}
		if !got.Debug {
			t.Errorf("FASTLANE_DEBUG=%q should be truthy, got Debug=false", s)
		}
	}

	falsy := []string{"0", "false", "no", "nope", "off"}
	for _, s := range falsy {
		clearEnv(t)
		t.Setenv("FASTLANE_DEBUG", s)
		got, err := Load(GlobalFlags{}, ChangedFlags{})
		if err != nil {
			t.Fatalf("Load(%q): %v", s, err)
		}
		if got.Debug {
			t.Errorf("FASTLANE_DEBUG=%q should be falsy, got Debug=true", s)
		}
	}
}

func TestLoadInvalidTimeoutErrors(t *testing.T) {
	withHome(t)
	clearEnv(t)
	t.Setenv("FASTLANE_TIMEOUT", "not-a-number")
	if _, err := Load(GlobalFlags{}, ChangedFlags{}); err == nil {
		t.Fatal("expected error for malformed FASTLANE_TIMEOUT")
	}
}
