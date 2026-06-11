# sandbox-launch

Integration of the built-in sandbox with the omac launcher (`omac start` / `omac serve`).

## ADDED Requirements

### Requirement: Built-in launcher profile is the default
The launcher SHALL provide a compiled-in launch profile named `builtin` that runs the inner harness under the built-in sandbox by re-executing the current omac binary:

```
{{self}} sandbox run --profile default --allow-file {{socket}} --read {{socket_dir}} {{tmpdir_flags}} --open-port {{tcp_port}} -- {{inner_cmd}} {{inner_args}}
```

`{{self}}` SHALL expand to the absolute path of the running omac executable. `builtin` SHALL be the default launch profile. The existing template mechanism and placeholders SHALL be preserved so users can configure external sandboxes (including the retained, non-default `nono` template and `no-sandbox-debug`).

#### Scenario: Default start uses built-in sandbox
- **WHEN** `omac start opencode` is run with no launcher configuration
- **THEN** the harness is launched via `omac sandbox run` with the bridge socket file, socket directory, tmpdir, and facade TCP port granted

#### Scenario: User overrides launcher
- **WHEN** the omac config selects the `nono` launch profile
- **THEN** the external `nono run ...` template is used unchanged

### Requirement: Bridge connectivity inside the sandbox
Under the default `builtin` profile, the inner harness MUST be able to reach the omac facade over both transports: the Unix domain socket (`bridge.sock`, granted via `--allow-file` and explicitly allowed in the macOS Seatbelt profile despite the network deny) and the loopback TCP port (granted via `--open-port`). All `OMAC_*` environment variables exported by the launcher MUST be visible to the inner harness (the default profile's env filtering passes `OMAC_*`).

#### Scenario: Unix socket transport on macOS
- **WHEN** the harness connects to `OMAC_SOCKET` from inside the sandbox on macOS with network filtering active
- **THEN** the connection succeeds

#### Scenario: TCP transport
- **WHEN** the harness sends a request to `OMAC_BASE` (`http://127.0.0.1:<tcp_port>/...`) from inside the sandbox
- **THEN** the request reaches the facade and the response streams back (including SSE without buffering)

### Requirement: Default sandbox profile content
The compiled-in `default` sandbox profile SHALL provide the equivalent of today's `tng-sandbox.json`: readwrite workdir; `filesystem.allow` for the harness state/cache paths (`~/.local/share/opencode`, `~/.local/state/opencode`, `~/.claude`, `~/.cache`, `~/Library/Caches`, `~/go`, `~/.rustup`, `~/.cargo`); `filesystem.read` for config paths (`~/.config/opencode`, `~/.opencode/bin`, `~/.nvm`, `~/.gitconfig`, `~/.gitignore_global`, `~/.claude.json`); `network.listen_port: [4097]`; `network.allow_tcp_connect: [22]`; prompt enabled with 60 s timeout and `on_unavailable: deny`; `environment.allow_vars` unset (pass-through minus blocklist).

#### Scenario: Harness state persists
- **WHEN** the inner harness writes session state under `~/.local/share/opencode`
- **THEN** the write succeeds and the data is visible on the host after exit

### Requirement: Doctor checks for sandbox prerequisites
`omac doctor` SHALL verify the platform prerequisites of the built-in sandbox and report actionable failures: on Linux, that `bwrap` is installed and the kernel supports Landlock ABI ≥ 4; on macOS, that `sandbox-exec` is present; on both, whether an interactive dialog backend is available (warning only).

#### Scenario: Missing bwrap reported
- **WHEN** `omac doctor` runs on a Linux host without bubblewrap
- **THEN** the report flags the missing dependency with an install hint

#### Scenario: Headless host warning
- **WHEN** no dialog backend is available
- **THEN** doctor warns that network prompts will fall back to the `on_unavailable` policy
