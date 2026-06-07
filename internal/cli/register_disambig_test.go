package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// stageHarnessSkill drops an omac.yaml under <workdir>/.<base>/skills/<name>/.
func stageHarnessSkill(t *testing.T, workdir, base, name string) {
	t.Helper()
	dir := filepath.Join(workdir, "."+base, "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "name: " + name + "\nsidecar:\n  command: [\"true\"]\n  mount: " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "omac.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveRegisterTarget_SingleOpenCode(t *testing.T) {
	isolateHome(t)
	wd := t.TempDir()
	stageHarnessSkill(t, wd, "opencode", "solo")
	env := makeEnv(wd)

	dir, _, harness, code := resolveRegisterTarget(env, "solo", "", false)
	if code != ExitOK {
		t.Fatalf("code = %d, want ExitOK", code)
	}
	if harness != "opencode" {
		t.Errorf("harness = %q, want opencode", harness)
	}
	if filepath.Base(dir) != "solo" {
		t.Errorf("dir = %q, want .../solo", dir)
	}
}

func TestResolveRegisterTarget_HarnessAmbiguous(t *testing.T) {
	isolateHome(t)
	wd := t.TempDir()
	stageHarnessSkill(t, wd, "opencode", "slack")
	stageHarnessSkill(t, wd, "claude", "slack")
	env := makeEnv(wd)

	// No --harness: ambiguous across harnesses -> misuse.
	_, _, _, code := resolveRegisterTarget(env, "slack", "", false)
	if code != ExitMisuse {
		t.Fatalf("code = %d, want ExitMisuse (ambiguous)", code)
	}

	// --harness claude resolves it.
	dir, _, harness, code := resolveRegisterTarget(env, "slack", "claude", false)
	if code != ExitOK {
		t.Fatalf("with --harness claude: code = %d, want ExitOK", code)
	}
	if harness != "claude-code" {
		t.Errorf("harness = %q, want claude-code", harness)
	}
	if !filepath.IsAbs(dir) || filepath.Base(dir) != "slack" {
		t.Errorf("dir = %q", dir)
	}
}

func TestResolveRegisterTarget_SharedNotAmbiguous(t *testing.T) {
	isolateHome(t)
	wd := t.TempDir()
	stageHarnessSkill(t, wd, "agents", "neutral")
	env := makeEnv(wd)

	dir, _, harness, code := resolveRegisterTarget(env, "neutral", "", false)
	if code != ExitOK {
		t.Fatalf("shared skill should not be ambiguous; code = %d", code)
	}
	// A shared skill records no harness on the entry.
	if harness != "" {
		t.Errorf("harness = %q, want empty (shared)", harness)
	}
	if filepath.Base(dir) != "neutral" {
		t.Errorf("dir = %q", dir)
	}
}

func TestResolveRegisterTarget_NotFound(t *testing.T) {
	isolateHome(t)
	wd := t.TempDir()
	env := makeEnv(wd)
	_, _, _, code := resolveRegisterTarget(env, "ghost", "", false)
	if code != ExitPrerequisiteMissing {
		t.Errorf("code = %d, want ExitPrerequisiteMissing", code)
	}
}

func TestResolveRegisterTarget_UnknownHarness(t *testing.T) {
	isolateHome(t)
	env := makeEnv(t.TempDir())
	_, _, _, code := resolveRegisterTarget(env, "x", "bogus", false)
	if code != ExitMisuse {
		t.Errorf("code = %d, want ExitMisuse for unknown harness", code)
	}
}
