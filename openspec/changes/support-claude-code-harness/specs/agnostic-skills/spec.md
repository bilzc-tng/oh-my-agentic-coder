## ADDED Requirements

### Requirement: Skills are harness-agnostic by contract

A skill SHALL depend only on the documented omac contract — the `OMAC_*` environment variables, the facade REST routes, the sidecar lifecycle, and the skill metadata schema — and MUST NOT depend on any harness-specific behavior, file path, or extension API. The same skill, unmodified, MUST work under every supported harness.

#### Scenario: Unmodified skill runs under multiple harnesses

- **WHEN** a skill authored for omac is run under `opencode` and then under `claude` without changes
- **THEN** its sidecar is reachable through the same facade routes and `OMAC_*` names, and it behaves identically

#### Scenario: Skill metadata contains no harness assumptions

- **WHEN** a skill's `omac.yaml` and bundled files are reviewed
- **THEN** they contain no harness-specific paths, plugin APIs, or agent-specific instructions required for the skill to function

#### Scenario: Marketplace install guidance is harness-neutral

- **WHEN** the skills manifest presents install guidance for a global skill (e.g. the marketplace `target_path`)
- **THEN** the guidance is expressed in harness-neutral terms (the project's skills directory), not OpenCode-specific wording

### Requirement: Harness-agnostic skill-authoring documentation

The skill-authoring guide (`CREATING_A_SKILL.md`) SHALL be harness-agnostic: it MUST instruct authors to target the omac contract and explicitly state that skills must not assume a harness. It MUST cover running a skill under both OpenCode and Claude Code with symmetric, equivalent instructions.

#### Scenario: Guide states the agnostic contract

- **WHEN** an author reads `CREATING_A_SKILL.md`
- **THEN** it clearly states that skills target `OMAC_*`/REST and must work unchanged across harnesses

#### Scenario: Guide covers OpenCode and Claude Code equally

- **WHEN** an author looks for how their skill is consumed by an agent
- **THEN** the guide provides equivalent, symmetric "running under OpenCode" and "running under Claude Code" instructions, with neither treated as the only option

#### Scenario: Examples are harness-neutral

- **WHEN** the guide shows example invocations or env usage
- **THEN** the examples use harness-neutral `OMAC_*`/REST patterns rather than OpenCode-only constructs

### Requirement: User-facing documentation reflects harness selection

`README.md` and `docs/MULTI_DIR_DESKTOP.md` SHALL document harness selection via the positional subcommand and SHALL cover Claude Code alongside OpenCode, including the default-when-omitted behavior and any per-harness limitations.

#### Scenario: README documents positional harness UX

- **WHEN** a user reads `README.md`
- **THEN** it documents `omac start opencode`, `omac start claude`, that omitting the harness defaults to `opencode`, and how to set up Claude Code

#### Scenario: Multi-dir guide covers the Claude Code bridge

- **WHEN** a user reads `docs/MULTI_DIR_DESKTOP.md`
- **THEN** it describes the Claude Code bridge and any differences or limitations of `omac serve` under Claude Code versus OpenCode

### Requirement: Documented path for adding a new harness

The documentation SHALL describe how to add support for a further agentic harness, listing the registry descriptor fields and the bridge-interface obligations, so the extensibility goal is actionable.

#### Scenario: "Adding a harness" guidance exists

- **WHEN** a contributor wants to add another agentic harness
- **THEN** the docs describe the required registry descriptor (name, aliases, inner command, server-launch convention, bridge) and the bridge interface it must implement
