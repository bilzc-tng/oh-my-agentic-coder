package cli

// Tests for the drift-detection helpers in start.go:
//   - autoDeregisterMissing: prunes registry entries whose skill dir
//     no longer exists, leaves the rest alone.
//   - findUnregisteredSkills: finds top-level dirs under
//     .opencode/skills/ that contain a meta.yaml but aren't registered.

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tngtech/oh-my-agentic-coder/internal/registry"
)

// stageWorkdir creates a workdir layout suitable for the drift
// helpers: .opencode/ exists, with optional skill directories under
// .opencode/skills/<name>/ each containing a meta.yaml.
func stageWorkdir(t *testing.T, skills ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range skills {
		skillDir := filepath.Join(dir, ".opencode", "skills", name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", skillDir, err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "meta.yaml"), []byte("name: "+name+"\n"), 0o644); err != nil {
			t.Fatalf("write meta.yaml: %v", err)
		}
	}
	return dir
}

func makeEnv(workdir string) *Env {
	null, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	return &Env{
		Workdir: workdir,
		Stdout:  null,
		Stderr:  null,
		Stdin:   null,
		Version: "test",
	}
}

func TestAutoDeregisterMissing_DeletesGoneSkills(t *testing.T) {
	dir := stageWorkdir(t, "alpha") // alpha exists on disk; bravo doesn't
	reg := &registry.Registry{Registered: []registry.Entry{
		{Name: "alpha", SkillDir: ".opencode/skills/alpha"},
		{Name: "bravo", SkillDir: ".opencode/skills/bravo"}, // never created
	}}
	if err := registry.Save(dir, reg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	pruned, err := autoDeregisterMissing(makeEnv(dir), reg)
	if err != nil {
		t.Fatalf("autoDeregisterMissing: %v", err)
	}
	if !reflect.DeepEqual(pruned, []string{"bravo"}) {
		t.Errorf("pruned = %v, want [bravo]", pruned)
	}

	// Caller's view must be updated: only alpha remains.
	if len(reg.Registered) != 1 || reg.Registered[0].Name != "alpha" {
		t.Errorf("in-memory reg after prune = %+v, want only alpha", reg.Registered)
	}
	// Persisted registry must agree.
	persisted, err := registry.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(persisted.Registered) != 1 || persisted.Registered[0].Name != "alpha" {
		t.Errorf("persisted reg after prune = %+v, want only alpha", persisted.Registered)
	}
}

func TestAutoDeregisterMissing_NoOpWhenAllPresent(t *testing.T) {
	dir := stageWorkdir(t, "alpha", "bravo")
	reg := &registry.Registry{Registered: []registry.Entry{
		{Name: "alpha", SkillDir: ".opencode/skills/alpha"},
		{Name: "bravo", SkillDir: ".opencode/skills/bravo"},
	}}
	pruned, err := autoDeregisterMissing(makeEnv(dir), reg)
	if err != nil {
		t.Fatalf("autoDeregisterMissing: %v", err)
	}
	if len(pruned) != 0 {
		t.Errorf("pruned = %v, want []", pruned)
	}
	if len(reg.Registered) != 2 {
		t.Errorf("reg should still have both skills, got %+v", reg.Registered)
	}
}

func TestAutoDeregisterMissing_EmptyRegistry(t *testing.T) {
	dir := stageWorkdir(t)
	reg := &registry.Registry{}
	pruned, err := autoDeregisterMissing(makeEnv(dir), reg)
	if err != nil {
		t.Fatalf("autoDeregisterMissing: %v", err)
	}
	if pruned != nil {
		t.Errorf("pruned = %v, want nil", pruned)
	}
}

func TestFindUnregisteredSkills_FindsNew(t *testing.T) {
	dir := stageWorkdir(t, "alpha", "bravo", "charlie")
	reg := &registry.Registry{Registered: []registry.Entry{
		{Name: "alpha"}, // bravo and charlie are unregistered
	}}
	got, err := findUnregisteredSkills(dir, reg)
	if err != nil {
		t.Fatalf("findUnregisteredSkills: %v", err)
	}
	want := []string{"bravo", "charlie"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findUnregisteredSkills = %v, want %v", got, want)
	}
}

func TestFindUnregisteredSkills_AllRegistered(t *testing.T) {
	dir := stageWorkdir(t, "alpha", "bravo")
	reg := &registry.Registry{Registered: []registry.Entry{
		{Name: "alpha"},
		{Name: "bravo"},
	}}
	got, err := findUnregisteredSkills(dir, reg)
	if err != nil {
		t.Fatalf("findUnregisteredSkills: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("findUnregisteredSkills = %v, want []", got)
	}
}

func TestFindUnregisteredSkills_SkipsDirsWithoutMetaYaml(t *testing.T) {
	dir := stageWorkdir(t, "alpha")
	// Stage a directory under skills/ but without a meta.yaml. It's
	// an incidental subdirectory (e.g. _template/), not a real skill,
	// so the helper must NOT flag it as unregistered.
	if err := os.MkdirAll(filepath.Join(dir, ".opencode", "skills", "_template"), 0o755); err != nil {
		t.Fatal(err)
	}
	reg := &registry.Registry{Registered: []registry.Entry{{Name: "alpha"}}}
	got, err := findUnregisteredSkills(dir, reg)
	if err != nil {
		t.Fatalf("findUnregisteredSkills: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("findUnregisteredSkills = %v, want [] (template dir lacks meta.yaml)", got)
	}
}

func TestFindUnregisteredSkills_NoSkillsDir(t *testing.T) {
	// Workdir without an .opencode/skills/ at all should yield nil
	// (no error). This is the fresh-clone case before any skill has
	// been installed.
	dir := t.TempDir()
	got, err := findUnregisteredSkills(dir, &registry.Registry{})
	if err != nil {
		t.Fatalf("findUnregisteredSkills: %v", err)
	}
	if got != nil {
		t.Errorf("findUnregisteredSkills with no skills dir = %v, want nil", got)
	}
}
