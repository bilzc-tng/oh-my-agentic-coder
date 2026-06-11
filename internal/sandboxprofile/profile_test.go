package sandboxprofile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseValidProfile(t *testing.T) {
	data := []byte(`{
	  "meta": {"name": "tng-sandbox"},
	  "workdir": {"access": "readwrite"},
	  "filesystem": {
	    "allow": [".", "~/.cache"],
	    "read": ["~/.gitconfig"],
	    "override_deny": ["~/.git-credentials"]
	  },
	  "network": {
	    "listen_port": [4097],
	    "allow_tcp_connect": [22],
	    "network_prompt": {"enabled": true, "prompt_timeout_secs": 60, "on_unavailable": "deny"}
	  }
	}`)
	p, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if p.Workdir.Access != AccessReadWrite {
		t.Errorf("workdir.access = %q", p.Workdir.Access)
	}
	if !p.Network.PromptEnabled() {
		t.Error("prompt should be enabled")
	}
	if p.Network.PromptTimeoutSecs() != 60 {
		t.Errorf("timeout = %d", p.Network.PromptTimeoutSecs())
	}
	if p.Network.OnUnavailable() != OnUnavailableDeny {
		t.Errorf("on_unavailable = %q", p.Network.OnUnavailable())
	}
	if p.Network.EffectiveMode() != ModeFiltered {
		t.Errorf("mode = %q", p.Network.EffectiveMode())
	}
	if got := p.Filesystem.OverrideDeny; len(got) != 1 || got[0] != "~/.git-credentials" {
		t.Errorf("override_deny = %v", got)
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	cases := []string{
		`{"security": {"groups": []}}`,
		`{"filesystem": {"allow_file": []}}`,
		`{"network": {"credentials": ["tng_skills"]}}`,
		`{"network": {"network_prompt": {"learned_policy_path": "/x"}}}`,
	}
	for _, c := range cases {
		if _, err := Parse([]byte(c)); err == nil {
			t.Errorf("Parse(%s) should fail on unknown field", c)
		}
	}
}

func TestParseValidationErrors(t *testing.T) {
	cases := []string{
		`{"workdir": {"access": "rw"}}`,
		`{"network": {"mode": "openish"}}`,
		`{"network": {"enforcement": "none"}}`,
		`{"network": {"listen_port": [70000]}}`,
		`{"network": {"network_prompt": {"on_unavailable": "ask"}}}`,
		`{"environment": {"allow_vars": [" "]}}`,
	}
	for _, c := range cases {
		if _, err := Parse([]byte(c)); err == nil {
			t.Errorf("Parse(%s) should fail validation", c)
		}
	}
}

func TestPromptDefaultsWhenObjectPresent(t *testing.T) {
	p, err := Parse([]byte(`{"network": {"network_prompt": {}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !p.Network.PromptEnabled() {
		t.Error("prompt object present means enabled (nono semantics)")
	}
	if p.Network.PromptTimeoutSecs() != DefaultPromptTimeoutSecs {
		t.Errorf("timeout = %d", p.Network.PromptTimeoutSecs())
	}
	p2, err := Parse([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if p2.Network.PromptEnabled() {
		t.Error("absent prompt object means disabled")
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	got, err := ExpandPath("~/.gitconfig")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(home, ".gitconfig") {
		t.Errorf("got %q", got)
	}
	t.Setenv("OMAC_TEST_DIR", "/tmp/omac-test")
	got, err = ExpandPath("$OMAC_TEST_DIR/sub")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/omac-test/sub" {
		t.Errorf("got %q", got)
	}
	if _, err := ExpandPath(""); err == nil {
		t.Error("empty path should fail")
	}
}

func TestExpandExistingSkipsMissing(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nope")
	var buf strings.Builder
	out, err := ExpandExisting([]string{dir, missing}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != dir {
		t.Errorf("out = %v", out)
	}
	if !strings.Contains(buf.String(), "skipping nonexistent path") {
		t.Errorf("notice missing: %q", buf.String())
	}
}

func TestResolveBuiltinDefault(t *testing.T) {
	// Point HOME at an empty dir so no user profile shadows the builtin.
	t.Setenv("HOME", t.TempDir())
	p, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if p.Workdir.Access != AccessReadWrite {
		t.Errorf("builtin default workdir.access = %q", p.Workdir.Access)
	}
	if len(p.Network.ListenPort) != 1 || p.Network.ListenPort[0] != 4097 {
		t.Errorf("builtin default listen_port = %v", p.Network.ListenPort)
	}
	if len(p.Network.AllowTCPConnect) != 1 || p.Network.AllowTCPConnect[0] != 22 {
		t.Errorf("builtin default allow_tcp_connect = %v", p.Network.AllowTCPConnect)
	}
	if !p.Network.PromptEnabled() {
		t.Error("builtin default prompt should be enabled")
	}
}

func TestResolveUserProfileOverridesBuiltin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "omac", "profiles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "default.json"),
		[]byte(`{"workdir": {"access": "read"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Resolve("default")
	if err != nil {
		t.Fatal(err)
	}
	if p.Workdir.Access != AccessRead {
		t.Errorf("user profile should win, got access %q", p.Workdir.Access)
	}
}

func TestResolveExplicitPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.json")
	if err := os.WriteFile(path, []byte(`{"meta": {"name": "custom"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Resolve(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.Meta.Name != "custom" {
		t.Errorf("name = %q", p.Meta.Name)
	}
}

func TestResolveUnknownProfileListsSearchLocations(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_, err := Resolve("nosuch")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "searched") || !strings.Contains(err.Error(), "builtin") {
		t.Errorf("error should list search locations: %v", err)
	}
}

func TestParseFlagsLauncherStyle(t *testing.T) {
	f, err := ParseFlags([]string{
		"--profile", "tng-sandbox",
		"--allow-file", "/tmp/x/bridge.sock",
		"--read", "/tmp/x",
		"--read", "/tmp/t", "--write", "/tmp/t",
		"--open-port", "49152",
		"--", "opencode", "--port", "4097",
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.ProfileRef != "tng-sandbox" {
		t.Errorf("profile = %q", f.ProfileRef)
	}
	if len(f.AllowFile) != 1 || f.AllowFile[0] != "/tmp/x/bridge.sock" {
		t.Errorf("allow-file = %v", f.AllowFile)
	}
	if len(f.Read) != 2 || len(f.Write) != 1 {
		t.Errorf("read=%v write=%v", f.Read, f.Write)
	}
	if len(f.OpenPort) != 1 || f.OpenPort[0] != 49152 {
		t.Errorf("open-port = %v", f.OpenPort)
	}
	if len(f.InnerArgv) != 3 || f.InnerArgv[0] != "opencode" {
		t.Errorf("inner = %v", f.InnerArgv)
	}
}

func TestParseFlagsErrors(t *testing.T) {
	cases := [][]string{
		{"--open-port", "0", "--", "x"},
		{"--open-port", "abc", "--", "x"},
		{"--workdir-access", "rw", "--", "x"},
		{"--unknown", "--", "x"},
		{"--read", "/x"},       // no -- separator
		{"--read", "/x", "--"}, // no command after --
		{"--read"},             // missing value
	}
	for _, c := range cases {
		if _, err := ParseFlags(c); err == nil {
			t.Errorf("ParseFlags(%v) should fail", c)
		}
	}
}

func TestParseFlagsInlineValues(t *testing.T) {
	f, err := ParseFlags([]string{"--profile=p", "--open-port=8080", "--", "true"})
	if err != nil {
		t.Fatal(err)
	}
	if f.ProfileRef != "p" || len(f.OpenPort) != 1 || f.OpenPort[0] != 8080 {
		t.Errorf("inline parse failed: %+v", f)
	}
}

func TestMergeAdditive(t *testing.T) {
	p := &Profile{
		Filesystem: Filesystem{Read: []string{"~/.gitconfig"}},
		Network:    Network{OpenPort: []int{1000}, AllowDomain: []string{"github.com"}},
		Workdir:    Workdir{Access: AccessRead},
	}
	f := &Flags{
		Read:          []string{"/tmp/x"},
		AllowFile:     []string{"/tmp/x/s.sock"},
		OpenPort:      []int{49152},
		AllowDomain:   []string{"npmjs.org"},
		WorkdirAccess: AccessReadWrite,
	}
	merged, warnings := Merge(p, f)
	if len(warnings) != 0 {
		t.Errorf("warnings = %v", warnings)
	}
	if len(merged.Filesystem.Read) != 2 {
		t.Errorf("read = %v", merged.Filesystem.Read)
	}
	if len(merged.Filesystem.Allow) != 1 {
		t.Errorf("allow = %v (allow-file should merge into allow)", merged.Filesystem.Allow)
	}
	if len(merged.Network.OpenPort) != 2 {
		t.Errorf("open_port = %v", merged.Network.OpenPort)
	}
	if len(merged.Network.AllowDomain) != 2 {
		t.Errorf("allow_domain = %v", merged.Network.AllowDomain)
	}
	if merged.Workdir.Access != AccessReadWrite {
		t.Errorf("workdir = %q", merged.Workdir.Access)
	}
	// base profile untouched
	if len(p.Filesystem.Read) != 1 || p.Workdir.Access != AccessRead {
		t.Error("Merge mutated the base profile")
	}
}

func TestMergeBlockNetOverrides(t *testing.T) {
	p := &Profile{
		Network: Network{
			AllowDomain:   []string{"github.com"},
			NetworkPrompt: &NetworkPrompt{},
		},
	}
	merged, warnings := Merge(p, &Flags{BlockNet: true})
	if merged.Network.EffectiveMode() != ModeBlocked {
		t.Errorf("mode = %q", merged.Network.EffectiveMode())
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "--block-net overrides") {
		t.Errorf("warnings = %v", warnings)
	}
}

func TestEffectiveProtectedPaths(t *testing.T) {
	t.Setenv("HOME", "/home/u")
	b := Baseline{ProtectedPaths: []string{"~/.git-credentials", "~/.netrc", "~/.ssh"}}
	got := EffectiveProtectedPaths(b, []string{"~/.git-credentials"})
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	for _, p := range got {
		if strings.Contains(p, ".git-credentials") {
			t.Errorf("override_deny hole not punched: %v", got)
		}
		if !strings.HasPrefix(p, "/home/u/") {
			t.Errorf("protected path not expanded: %q", p)
		}
	}
}

func TestFilterEnv(t *testing.T) {
	environ := []string{
		"HOME=/home/u",
		"PATH=/usr/bin",
		"OMAC_BASE=http://127.0.0.1:1/x",
		"AWS_SECRET_ACCESS_KEY=oops",
		"DYLD_INSERT_LIBRARIES=/evil.dylib",
		"NODE_OPTIONS=--require evil",
		"OP_SESSION_team=tok",
		"HTTP_PROXY=http://old:1",
	}
	injected := map[string]string{"HTTP_PROXY": "http://omac:tok@127.0.0.1:9"}

	// With allowlist.
	got := FilterEnv(environ, []string{"HOME", "PATH", "OMAC_*"}, injected)
	want := map[string]string{
		"HOME":       "/home/u",
		"PATH":       "/usr/bin",
		"OMAC_BASE":  "http://127.0.0.1:1/x",
		"HTTP_PROXY": "http://omac:tok@127.0.0.1:9",
	}
	gotMap := envMap(got)
	if len(gotMap) != len(want) {
		t.Errorf("got %v want %v", gotMap, want)
	}
	for k, v := range want {
		if gotMap[k] != v {
			t.Errorf("%s = %q want %q", k, gotMap[k], v)
		}
	}

	// Without allowlist: everything but the blocklist passes.
	got = FilterEnv(environ, nil, nil)
	gotMap = envMap(got)
	if _, ok := gotMap["AWS_SECRET_ACCESS_KEY"]; !ok {
		t.Error("no allowlist: AWS var should pass")
	}
	for _, k := range []string{"DYLD_INSERT_LIBRARIES", "NODE_OPTIONS", "OP_SESSION_team"} {
		if _, ok := gotMap[k]; ok {
			t.Errorf("blocklisted %s leaked", k)
		}
	}

	// Blocklist beats allowlist.
	got = FilterEnv(environ, []string{"NODE_OPTIONS"}, nil)
	if len(got) != 0 {
		t.Errorf("blocklist must beat allowlist: %v", got)
	}
}

func envMap(env []string) map[string]string {
	m := map[string]string{}
	for _, kv := range env {
		if i := strings.IndexByte(kv, '='); i > 0 {
			m[kv[:i]] = kv[i+1:]
		}
	}
	return m
}
