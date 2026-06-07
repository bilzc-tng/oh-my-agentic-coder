## ADDED Requirements

### Requirement: Harness owns a skills-directory identity

Each harness SHALL declare the skills-directory base name it owns. omac MUST classify every skills root as belonging to exactly one harness's own base, or to the shared neutral base, so discovery can include or exclude roots by harness.

#### Scenario: OpenCode owns the opencode base

- **WHEN** the active harness is `opencode`
- **THEN** its own skills base is `opencode` (e.g. `.opencode/skills`, `~/.config/opencode/skills`)

#### Scenario: Claude Code owns the claude base

- **WHEN** the active harness is `claude-code`
- **THEN** its own skills base is `claude` (e.g. `.claude/skills`, `~/.claude/skills`)

#### Scenario: The agents base is shared

- **WHEN** any harness is active
- **THEN** the `agents` base (`.agents/skills`, `~/.config/agents/skills`, `~/.agents/skills`) is in scope as the shared/neutral root

### Requirement: Discovery is scoped to the active harness

omac SHALL discover skills only from the active harness's own skills roots plus the shared `agents` roots, and MUST NOT scan any other harness's skills roots. This applies to both the workdir layer and the user-global layer.

#### Scenario: OpenCode scope excludes the Claude dir

- **WHEN** the active harness is `opencode` and a skill exists only under `.claude/skills`
- **THEN** omac does not discover, register, mount, or surface that skill

#### Scenario: Claude scope excludes the OpenCode dir

- **WHEN** the active harness is `claude-code` and a skill exists only under `.opencode/skills`
- **THEN** omac does not discover, register, mount, or surface that skill

#### Scenario: Shared skills are visible to every harness

- **WHEN** a skill exists under `.agents/skills`
- **THEN** it is discovered regardless of the active harness

#### Scenario: Own-harness dir overrides shared on name collision

- **WHEN** the same skill name exists under both the active harness's own dir and `.agents/skills`
- **THEN** the active harness's own dir wins

#### Scenario: Scoping applies to the global layer

- **WHEN** the active harness is `claude-code` and a skill exists only under `~/.config/opencode/skills`
- **THEN** omac does not discover it (the global OpenCode root is out of scope)

### Requirement: start and serve honor the active harness scope

`omac start <harness>` and `omac serve <harness>` SHALL discover, auto-register, mount, and surface to the harness bridge only the in-scope skills for the active harness. omac MUST NOT check or use registrations belonging to another harness for the active run.

#### Scenario: Start mounts only in-scope skills

- **WHEN** the user runs `omac start claude` and both a Claude-scope and an OpenCode-only skill exist on disk
- **THEN** only the Claude-scope skill (its own dir or `.agents`) is mounted on the facade and surfaced to the Claude bridge

#### Scenario: Bridge manifest reflects scope

- **WHEN** a harness bridge requests the skills manifest for a directory
- **THEN** the manifest lists only skills in the active harness's scope

### Requirement: Install target follows the active harness

The marketplace `/install` default target directory SHALL be the active harness's workdir skills directory (`.opencode/skills` for OpenCode, `.claude/skills` for Claude Code), so installed skills land where the running harness loads them. An explicit `target_path` (including `.agents/skills` for a shared skill) MUST still override the default.

#### Scenario: Install under Claude targets the Claude dir

- **WHEN** a skill is installed via `/install` with no explicit `target_path` under the `claude-code` harness
- **THEN** it is unpacked into `.claude/skills/`

#### Scenario: Install under OpenCode targets the OpenCode dir

- **WHEN** a skill is installed via `/install` with no explicit `target_path` under the `opencode` harness
- **THEN** it is unpacked into `.opencode/skills/`

#### Scenario: Explicit target_path overrides

- **WHEN** `/install` is called with `target_path` set to `.agents/skills`
- **THEN** the skill is unpacked there regardless of the active harness
