package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// LauncherConfig is the oh-my-agentic-coder.json file.
type LauncherConfig struct {
	Sandbox SandboxConfig `json:"sandbox"`
	Facade  FacadeConfig  `json:"facade"`
}

// SandboxConfig declares named sandbox profiles.
type SandboxConfig struct {
	DefaultProfile string                    `json:"default_profile"`
	Profiles       map[string]SandboxProfile `json:"profiles"`
}

// SandboxProfile describes how to launch the sandbox for a given runtime.
type SandboxProfile struct {
	// Command is a templated argv. Supported placeholders:
	//   {{socket}}, {{socket_dir}}, {{inner_cmd}}, {{inner_args}},
	//   {{skills_csv}}, {{per_skill_env_flags}}, {{workdir}}
	// Tokens that expand to multiple argv entries (inner_args,
	// per_skill_env_flags) must stand alone in their slot.
	Command  []string `json:"command"`
	InnerCmd []string `json:"inner_cmd"`
}

// FacadeConfig tunes the reverse proxy.
type FacadeConfig struct {
	IdleTimeoutSecs    int      `json:"idle_timeout_secs"`
	MaxBodyBytes       int64    `json:"max_body_bytes"`
	BaseEnvPassthrough []string `json:"base_env_passthrough"`
}

// DefaultLauncherConfig returns a config that ships as the compiled-in default.
// It matches the existing tng-opencode invocation at the repo root.
func DefaultLauncherConfig() LauncherConfig {
	return LauncherConfig{
		Sandbox: SandboxConfig{
			DefaultProfile: "nono",
			Profiles: map[string]SandboxProfile{
				"nono": {
					// Reference invocation for nono (https://nono.sh).
					//
					// Transport: omac binds the facade on BOTH a Unix
					// socket and a 127.0.0.1 TCP port. We tell nono to:
					//
					//   - --allow-file <socket>      grant open(2) on the
					//                                Unix socket inode
					//                                (Linux: this is enough;
					//                                macOS: necessary but
					//                                not sufficient under
					//                                proxy mode).
					//
					//   - --read <socket-dir>        path-component lookup
					//                                during connect(2).
					//
					//   - --open-port <tcp-port>     allow bidirectional
					//                                127.0.0.1:<port> from
					//                                inside the sandbox.
					//                                THIS is the transport
					//                                that works on macOS
					//                                under proxy mode (auto-
					//                                activated by any nono
					//                                profile with
					//                                custom_credentials,
					//                                network_profile,
					//                                --allow-domain,
					//                                --credential, or
					//                                --upstream-proxy).
					//                                Per the nono
					//                                "Networking" docs,
					//                                --open-port emits a
					//                                Seatbelt allow rule
					//                                that takes precedence
					//                                over the proxy-mode
					//                                `(deny network*)`.
					//
					// Inside the sandbox the agent reads OMAC_<SKILL>_BASE
					// (a TCP URL) by default, falling back to
					// OMAC_<SKILL>_SOCKET_BASE for the http+unix:// form.
					//
					// Env-var injection: nono no longer accepts a literal
					// `--env KEY=VAL` flag. Instead sandbox.Exec sets
					// OMAC_* in nono's own process environment, and nono
					// propagates the parent env to the inner process by
					// default. If you author a custom nono profile with
					// environment.allow_vars set, add `OMAC_*` to the
					// list.
					//
					// IMPORTANT: this profile does NOT use --block-net.
					// On macOS that installs `(deny network*)` plus a
					// `--open-port` allowance — but the interaction with
					// --network-profile and Seatbelt rule ordering is
					// untested for our use case. Use --network-profile
					// instead (see nono-netprofile below).
					Command: []string{
						"nono", "run",
						"--allow-cwd",
						"--profile", "tng-sandbox",
						"--allow-file", "{{socket}}",
						"--read", "{{socket_dir}}",
						"--open-port", "{{tcp_port}}",
						"--",
						"{{inner_cmd}}", "{{inner_args}}",
					},
					InnerCmd: []string{"opencode"},
				},
				// Same as above but adds --network-profile opencode so
				// outbound HTTP goes through nono's credential-injection
				// proxy. --open-port keeps the facade reachable; per the
				// nono docs it works alongside domain filtering.
				"nono-netprofile": {
					Command: []string{
						"nono", "run",
						"--allow-cwd",
						"--profile", "tng-sandbox",
						"--network-profile", "opencode",
						"--allow-file", "{{socket}}",
						"--read", "{{socket_dir}}",
						"--open-port", "{{tcp_port}}",
						"--",
						"{{inner_cmd}}", "{{inner_args}}",
					},
					InnerCmd: []string{"opencode"},
				},
				"no-sandbox-debug": {
					Command:  []string{"{{inner_cmd}}", "{{inner_args}}"},
					InnerCmd: []string{"bash"},
				},
			},
		},
		Facade: FacadeConfig{
			IdleTimeoutSecs:    300,
			MaxBodyBytes:       10 * 1024 * 1024,
			BaseEnvPassthrough: []string{"PATH", "HOME", "USER", "LANG", "LC_ALL", "LC_CTYPE", "TMPDIR"},
		},
	}
}

// LoadLauncher loads the launcher config from workdir/.opencode/oh-my-agentic-coder.json
// or, failing that, $XDG_CONFIG_HOME/omac/config.json (~/.config/omac/config.json).
// If neither exists, the compiled-in default is returned.
func LoadLauncher(workdir string) (LauncherConfig, string, error) {
	candidates := []string{
		filepath.Join(workdir, ".opencode", "oh-my-agentic-coder.json"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".config", "omac", "config.json"))
	}
	for _, p := range candidates {
		raw, err := os.ReadFile(p)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return LauncherConfig{}, "", fmt.Errorf("read %s: %w", p, err)
		}
		var lc LauncherConfig
		if err := json.Unmarshal(raw, &lc); err != nil {
			return LauncherConfig{}, "", fmt.Errorf("parse %s: %w", p, err)
		}
		lc = mergeDefaults(lc)
		return lc, p, nil
	}
	return DefaultLauncherConfig(), "", nil
}

func mergeDefaults(lc LauncherConfig) LauncherConfig {
	def := DefaultLauncherConfig()
	if lc.Sandbox.DefaultProfile == "" {
		lc.Sandbox.DefaultProfile = def.Sandbox.DefaultProfile
	}
	if lc.Sandbox.Profiles == nil {
		lc.Sandbox.Profiles = def.Sandbox.Profiles
	}
	if lc.Facade.IdleTimeoutSecs == 0 {
		lc.Facade.IdleTimeoutSecs = def.Facade.IdleTimeoutSecs
	}
	if lc.Facade.MaxBodyBytes == 0 {
		lc.Facade.MaxBodyBytes = def.Facade.MaxBodyBytes
	}
	if lc.Facade.BaseEnvPassthrough == nil {
		lc.Facade.BaseEnvPassthrough = def.Facade.BaseEnvPassthrough
	}
	return lc
}
