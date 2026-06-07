## ADDED Requirements

### Requirement: Bridge interface common to all harnesses

omac SHALL define a single bridge interface that every harness implements against the stable omac contract. A bridge MUST, when active: (1) call the control plane on session lifecycle using `OMAC_CONTROL_BASE`; (2) surface the omac skills manifest as agent-visible context; and (3) expose per-skill base URLs to the agent using the `OMAC_*` naming from the Go core. Bridges MUST source the manifest text and env naming from a single shared definition to prevent divergence between harnesses.

#### Scenario: Bridge activates and deactivates via control plane

- **WHEN** a session starts or ends in an omac-served directory under any harness with `OMAC_CONTROL_BASE` set
- **THEN** the bridge POSTs to `/__omac__/activate` (or `/__omac__/reload`) on start and `/__omac__/deactivate` on end

#### Scenario: Bridge surfaces the skills manifest

- **WHEN** a session is active in a directory with one or more skills under any harness
- **THEN** the agent's context includes the skills manifest listing each skill's mount, base URL, and readiness state

#### Scenario: Bridge exposes skill base URLs

- **WHEN** a workdir skill `echo-rest` and a global skill `skill-marketplace` are active under any harness
- **THEN** the agent can resolve `OMAC_ECHO_REST_BASE` and both `OMAC_G_SKILL_MARKETPLACE_BASE` and the flat `OMAC_SKILL_MARKETPLACE_BASE` alias, identically across harnesses

#### Scenario: Bridge is inert without a control base

- **WHEN** `OMAC_CONTROL_BASE` is not set in the agent environment
- **THEN** the bridge performs no control-plane calls and does not interfere with normal operation of the harness

### Requirement: OpenCode bridge

The OpenCode bridge SHALL implement the bridge interface using the OpenCode plugin mechanism, preserving today's behavior.

#### Scenario: OpenCode bridge behavior unchanged

- **WHEN** the active harness is `opencode`
- **THEN** the existing OpenCode plugin drives activation/deactivation, manifest injection, and per-session env injection exactly as before this change

### Requirement: Claude Code bridge

The Claude Code bridge SHALL implement the bridge interface using Claude Code's native extension mechanism (settings/hooks), at functional parity with the OpenCode bridge.

#### Scenario: Activate/deactivate via Claude Code hooks

- **WHEN** a Claude Code session starts or ends in an omac-served directory
- **THEN** a Claude Code hook POSTs to the control plane `/__omac__/activate` (or `/__omac__/reload`) on start and `/__omac__/deactivate` on end

#### Scenario: Manifest visible to the Claude Code agent

- **WHEN** a Claude Code session is active in an omac-served directory with skills
- **THEN** the agent's context includes the same skills manifest content the OpenCode bridge injects, including harness-neutral marketplace `target_path` install guidance

#### Scenario: Skill base URLs available under Claude Code

- **WHEN** skills are active under Claude Code
- **THEN** the agent can resolve their base URLs via the same `OMAC_<MOUNT>_BASE` / `OMAC_G_<MOUNT>_BASE` names used under OpenCode

#### Scenario: Fallback when per-session env injection is unavailable

- **WHEN** Claude Code does not support per-session shell-env injection in the current setup
- **THEN** the bridge falls back to the process-level single-directory base aliases that `omac start`/`omac serve` already export, and skills remain callable for the single active directory

### Requirement: Bridge parity verified end-to-end

Each shipped bridge SHALL be verifiable end-to-end with the existing smoke-test skills, demonstrating that the sandbox, skills, and skill marketplace work under that harness.

#### Scenario: Echo-rest smoke test passes per harness

- **WHEN** the `echo-rest` skill is exercised under a shipped harness (`opencode` or `claude`)
- **THEN** the JSON round-trip on `POST /echo`, the secret-injection proof on `GET /whoami`, and the SSE stream on `GET /tick` all succeed through the facade

#### Scenario: Marketplace install works per harness

- **WHEN** the agent installs a skill via the skill-marketplace `/install` endpoint under a shipped harness
- **THEN** the skill is unpacked into the target directory and, after a control-plane reload, becomes reachable on the facade
