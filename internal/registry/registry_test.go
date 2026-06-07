package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	// .opencode will be created lazily by Save.
	r := &Registry{}
	r.Upsert(Entry{
		Name:                "slack",
		SkillDir:            ".opencode/skills/slack",
		BundleHash:          "sha256:abc",
		RegisteredAt:        time.Unix(1700000000, 0).UTC(),
		DeclaredSecretNames: []string{"SLACK_BOT_TOKEN"},
	})
	if err := Save(dir, r); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Registered) != 1 || loaded.Registered[0].Name != "slack" {
		t.Fatalf("unexpected registry: %+v", loaded)
	}
	if loaded.Version != SchemaVersion {
		t.Fatalf("version = %d, want %d", loaded.Version, SchemaVersion)
	}
	if _, err := os.Stat(filepath.Join(dir, ".opencode", "sidecar.json")); err != nil {
		t.Fatalf("registry file missing: %v", err)
	}
}

func TestRemove(t *testing.T) {
	r := &Registry{}
	r.Upsert(Entry{Name: "a"})
	r.Upsert(Entry{Name: "b"})
	if !r.Remove("a") {
		t.Fatal("expected remove true")
	}
	if r.Remove("a") {
		t.Fatal("expected second remove false")
	}
	if len(r.Registered) != 1 || r.Registered[0].Name != "b" {
		t.Fatalf("unexpected after remove: %+v", r)
	}
}

func TestWithLock(t *testing.T) {
	dir := t.TempDir()
	if err := WithLock(dir, func() error { return nil }); err != nil {
		t.Fatalf("lock: %v", err)
	}
	if _, err := os.Stat(LockPath(dir)); err != nil {
		t.Fatalf("lock file missing: %v", err)
	}
}

// TestGlobalRoundTrip verifies the user-global registry writes to
// $XDG_CONFIG_HOME/omac/sidecar.json and round-trips independently of
// any workdir.
func TestGlobalRoundTrip(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	want := filepath.Join(xdg, "omac", "sidecar.json")
	if got := GlobalPath(); got != want {
		t.Fatalf("GlobalPath() = %q, want %q", got, want)
	}

	r := &Registry{}
	r.Upsert(Entry{Name: "tng-email", SkillDir: "/abs/skills/tng-email", BundleHash: "sha256:xyz"})
	if err := WithGlobalLock(func() error { return SaveGlobal(r) }); err != nil {
		t.Fatalf("SaveGlobal: %v", err)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("global registry file missing: %v", err)
	}
	loaded, err := LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal: %v", err)
	}
	if len(loaded.Registered) != 1 || loaded.Registered[0].Name != "tng-email" {
		t.Fatalf("unexpected global registry: %+v", loaded)
	}
}

// TestGlobalDirXDGPrecedence confirms XDG_CONFIG_HOME wins over HOME.
func TestGlobalDirXDGPrecedence(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", t.TempDir())
	if got, want := GlobalDir(), filepath.Join(xdg, "omac"); got != want {
		t.Fatalf("GlobalDir() = %q, want %q", got, want)
	}
}

// TestLoadGlobalMissingIsEmpty confirms a missing global file yields an
// empty registry rather than an error, so callers can always merge it.
func TestLoadGlobalMissingIsEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	r, err := LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal on empty: %v", err)
	}
	if len(r.Registered) != 0 {
		t.Fatalf("expected empty registry, got %+v", r)
	}
}

func TestUpsertPerHarnessCoexist(t *testing.T) {
	r := &Registry{}
	r.Upsert(Entry{Name: "slack", Harness: "opencode", SkillDir: "/oc/slack"})
	r.Upsert(Entry{Name: "slack", Harness: "claude-code", SkillDir: "/cc/slack"})
	if len(r.Registered) != 2 {
		t.Fatalf("want 2 coexisting entries, got %d", len(r.Registered))
	}
	// Updating one harness must not touch the other.
	r.Upsert(Entry{Name: "slack", Harness: "opencode", SkillDir: "/oc/slack-v2"})
	if len(r.Registered) != 2 {
		t.Fatalf("update should not add an entry, got %d", len(r.Registered))
	}
	oc, _ := r.FindForHarness("slack", "opencode")
	cc, _ := r.FindForHarness("slack", "claude-code")
	if oc == nil || oc.SkillDir != "/oc/slack-v2" {
		t.Errorf("opencode entry = %+v, want /oc/slack-v2", oc)
	}
	if cc == nil || cc.SkillDir != "/cc/slack" {
		t.Errorf("claude entry = %+v, want /cc/slack", cc)
	}
}

func TestFindForHarnessLegacyFallback(t *testing.T) {
	r := &Registry{}
	r.Upsert(Entry{Name: "old", SkillDir: "/legacy"}) // no harness
	// A harness-specific lookup falls back to the legacy entry.
	e, _ := r.FindForHarness("old", "claude-code")
	if e == nil || e.SkillDir != "/legacy" {
		t.Errorf("legacy fallback failed: %+v", e)
	}
}

func TestUpsertUpgradesLegacyEntry(t *testing.T) {
	r := &Registry{}
	r.Upsert(Entry{Name: "x", SkillDir: "/old"})                         // legacy
	r.Upsert(Entry{Name: "x", Harness: "opencode", SkillDir: "/scoped"}) // should replace legacy
	if len(r.Registered) != 1 {
		t.Fatalf("legacy entry should be upgraded in place, got %d entries", len(r.Registered))
	}
	if r.Registered[0].Harness != "opencode" || r.Registered[0].SkillDir != "/scoped" {
		t.Errorf("entry not upgraded: %+v", r.Registered[0])
	}
}
