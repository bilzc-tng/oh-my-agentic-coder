## ADDED Requirements

### Requirement: Harness selected by positional subcommand

omac SHALL select the inner agent harness via an optional positional token immediately following `start` or `serve`: `omac start <harness>` and `omac serve <harness>`. omac MUST NOT require a `--harness` flag. When the positional harness token is omitted, omac SHALL default to `opencode`, preserving current behavior.

#### Scenario: Default harness when omitted

- **WHEN** the user runs `omac start` with no harness token
- **THEN** omac uses the `opencode` harness and launches exactly as it does today

#### Scenario: Explicit OpenCode harness

- **WHEN** the user runs `omac start opencode`
- **THEN** omac uses the OpenCode harness profile, server-launch convention, and bridge

#### Scenario: Claude Code harness

- **WHEN** the user runs `omac start claude`
- **THEN** omac uses the Claude Code harness profile, server-launch convention, and bridge

#### Scenario: Harness token coexists with flags and inner args

- **WHEN** the user runs `omac start claude --verbose -- --model X`
- **THEN** omac resolves harness `claude`, applies the omac flag `--verbose`, and passes `--model X` through as inner args

#### Scenario: Leading flag means default harness

- **WHEN** the first token after `start` begins with `-` (a flag) rather than a harness name
- **THEN** omac does not treat it as a harness, defaults the harness to `opencode`, and parses flags normally

#### Scenario: Unknown harness name is rejected

- **WHEN** the user runs `omac start <unknown>` where `<unknown>` is not a registered harness name or alias
- **THEN** omac exits with a non-zero status and an error listing the supported harness names

### Requirement: Extensible harness registry

omac SHALL describe every supported harness with a registry descriptor containing at least: the canonical name, optional aliases, the default inner command, the server-launch convention, and the bridge descriptor. Adding support for a new agentic harness MUST be achievable by adding one descriptor (plus its bridge assets) without modifying command dispatch or launch call sites.

#### Scenario: Registry drives inner command

- **WHEN** a harness is resolved from the registry
- **THEN** the launcher derives the default inner command from that descriptor instead of a hard-coded value

#### Scenario: Aliases resolve to a canonical harness

- **WHEN** the user runs `omac start claude` and the registry maps the alias `claude` to canonical `claude-code`
- **THEN** omac resolves to the `claude-code` descriptor

#### Scenario: Adding a harness requires no call-site edits

- **WHEN** a new harness descriptor (name, inner command, server-launch, bridge) is added to the registry
- **THEN** `omac start <new-harness>` and `omac serve <new-harness>` work without changes to `start`/`serve` dispatch logic

### Requirement: Per-harness server-launch convention

`omac serve` SHALL derive how the inner command becomes a long-lived server from the resolved harness descriptor, rather than branching on a specific inner-command basename.

#### Scenario: OpenCode server launch from descriptor

- **WHEN** the resolved harness is `opencode` under `omac serve` with no explicit subcommand
- **THEN** omac applies the OpenCode server-launch convention (the `serve` subcommand) as declared in the descriptor

#### Scenario: Claude Code server launch from descriptor

- **WHEN** the resolved harness is `claude-code` under `omac serve`
- **THEN** omac applies the Claude Code server-launch convention declared in its descriptor rather than the OpenCode `serve` subcommand

#### Scenario: Explicit inner override wins over harness default

- **WHEN** the user supplies an explicit inner command (via `--inner` or a profile) together with a harness token
- **THEN** omac uses the explicit inner command and applies the resolved harness's server-launch and bridge conventions to it

### Requirement: Harness-agnostic core contract preserved

Selecting a different harness SHALL NOT change the `OMAC_*` environment-variable contract, the skill metadata schema, the sidecar lifecycle, the secret/keychain model, or the `/__omac__/` control-plane surface. All harnesses MUST consume the same contract.

#### Scenario: Control plane is harness-independent

- **WHEN** the active harness is `claude-code`
- **THEN** the `/__omac__/{activate,deactivate,reload,reload-global}` endpoints behave exactly as they do for `opencode`

#### Scenario: Env naming is harness-independent

- **WHEN** the active harness changes
- **THEN** a given skill is reachable under the same `OMAC_<MOUNT>_BASE` / `OMAC_G_<MOUNT>_BASE` name regardless of harness

### Requirement: Harness-neutral discovery paths

omac SHALL keep existing OpenCode-named paths (`.opencode/`) working while treating the harness-neutral paths (`.agents/`) as the shared home for skills, so that no harness selection removes or breaks an existing layout. Each harness's client bridge assets MAY live in that harness's native directory.

#### Scenario: Existing OpenCode layout still resolves

- **WHEN** skills are installed under `.opencode/skills/` and the harness is `claude-code`
- **THEN** omac discovers and serves those skills

#### Scenario: Neutral path is honored

- **WHEN** skills are installed under `.agents/skills/`
- **THEN** omac discovers them regardless of the selected harness, with `.agents/` ranked above `.opencode/`
