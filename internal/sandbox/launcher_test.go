package sandbox

import (
	"reflect"
	"testing"

	"github.com/tngtech/oh-my-agentic-coder/internal/config"
)

func TestExpand_Nono(t *testing.T) {
	lc := config.DefaultLauncherConfig()
	prof := lc.Sandbox.Profiles["nono"]
	got, err := Expand(prof, Inputs{
		Workdir:  "/work",
		Socket:   "/tmp/omac-abc/bridge.sock",
		TCPPort:  41017,
		Mounts:   []string{"slack", "himalaya-email"},
		InnerCmd: []string{"opencode", "--model", "opus"},
	})
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	want := []string{
		"nono", "run",
		"--allow-cwd",
		"--profile", "tng-sandbox",
		"--allow-file", "/tmp/omac-abc/bridge.sock",
		"--read", "/tmp/omac-abc",
		"--open-port", "41017",
		"--",
		"opencode", "--model", "opus",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expand mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

// TestExpand_NonoNetprofile asserts the --network-profile variant.
// Both profiles use --open-port on the facade's TCP port, because both
// custom_credentials (in tng-sandbox.json) and --network-profile
// activate nono's proxy mode, which installs `(deny network*)` on
// macOS — including Unix-socket connects. --open-port emits a more-
// specific allow rule for that loopback port that takes precedence.
func TestExpand_NonoNetprofile(t *testing.T) {
	lc := config.DefaultLauncherConfig()
	prof := lc.Sandbox.Profiles["nono-netprofile"]
	got, err := Expand(prof, Inputs{
		Workdir:  "/work",
		Socket:   "/tmp/omac-abc/bridge.sock",
		TCPPort:  41017,
		Mounts:   []string{"slack"},
		InnerCmd: []string{"opencode"},
	})
	if err != nil {
		t.Fatalf("Expand: %v", err)
	}
	want := []string{
		"nono", "run",
		"--allow-cwd",
		"--profile", "tng-sandbox",
		"--network-profile", "opencode",
		"--allow-file", "/tmp/omac-abc/bridge.sock",
		"--read", "/tmp/omac-abc",
		"--open-port", "41017",
		"--",
		"opencode",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expand mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestOmacEnvName(t *testing.T) {
	cases := map[string]string{
		"slack":          "OMAC_SLACK_BASE",
		"himalaya-email": "OMAC_HIMALAYA_EMAIL_BASE",
		"mail2":          "OMAC_MAIL2_BASE",
		"a-b_c":          "OMAC_A_B_C_BASE",
	}
	for in, want := range cases {
		if got := OmacEnvName(in); got != want {
			t.Errorf("OmacEnvName(%q) = %q, want %q", in, got, want)
		}
	}
}
