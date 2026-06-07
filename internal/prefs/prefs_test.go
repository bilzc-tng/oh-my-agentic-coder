package prefs

import (
	"os"
	"path/filepath"
	"testing"
)

func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	isolate(t)
	p, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p.SuppressPluginWarning {
		t.Error("fresh prefs should not suppress the plugin warning")
	}
	if p.Version != SchemaVersion {
		t.Errorf("version = %d, want %d", p.Version, SchemaVersion)
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	isolate(t)
	if err := Save(&Store{SuppressPluginWarning: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// File should exist at the resolved path with 0600.
	p := Path()
	if p == "" {
		t.Fatal("Path() empty")
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !got.SuppressPluginWarning {
		t.Error("SuppressPluginWarning did not round-trip")
	}
}

func TestGlobalDirHonorsXDG(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if got, want := GlobalDir(), filepath.Join(xdg, "omac"); got != want {
		t.Errorf("GlobalDir = %q, want %q", got, want)
	}
}
