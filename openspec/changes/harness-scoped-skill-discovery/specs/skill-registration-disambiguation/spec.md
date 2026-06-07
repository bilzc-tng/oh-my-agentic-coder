## ADDED Requirements

### Requirement: Register resolves an unambiguous single skill

`omac register <name>` SHALL register the single in-scope skill matching `<name>` when exactly one exists. When none match, it MUST fail with a not-found error as before.

#### Scenario: Single match registers

- **WHEN** exactly one in-scope source contains a skill named `<name>`
- **THEN** omac registers it without prompting for disambiguation

#### Scenario: No match fails clearly

- **WHEN** no in-scope source contains `<name>`
- **THEN** omac exits non-zero with a not-found message

### Requirement: Register detects and reports harness ambiguity

When `<name>` matches skills under more than one harness's scope, `omac register <name>` SHALL by default stop and tell the user, listing each candidate with its harness, scope, and path, and naming the `--harness <name>` flag that resolves it. omac MUST NOT silently pick one.

#### Scenario: Ambiguous across harnesses stops with guidance

- **WHEN** `<name>` exists under both an OpenCode-scope dir and a Claude-scope dir
- **THEN** omac prints a formatted list of the candidates (harness, scope, path) and the `--harness` command to choose, and exits non-zero

#### Scenario: --harness resolves the ambiguity

- **WHEN** the user re-runs `omac register <name> --harness claude-code`
- **THEN** omac registers the candidate in the Claude Code scope and does not consider the OpenCode candidate

### Requirement: Register detects and reports scope ambiguity

When `<name>` matches both a workdir-local and a user-global skill within the chosen harness scope, `omac register <name>` SHALL by default stop and tell the user, and name the `--global` flag. The workdir candidate is the implied default once disambiguated.

#### Scenario: Ambiguous across scope stops with guidance

- **WHEN** `<name>` exists both workdir-local and user-global in the active harness scope
- **THEN** omac prints both candidates and the `--global` hint, and exits non-zero

#### Scenario: --global selects the global candidate

- **WHEN** the user re-runs `omac register <name> --global`
- **THEN** omac registers the user-global candidate

#### Scenario: Combined selectors resolve fully

- **WHEN** `<name>` is ambiguous across both harness and scope and the user passes `--harness <h> --global`
- **THEN** omac registers exactly the global candidate in harness `<h>`'s scope, or errors if that still does not identify exactly one

### Requirement: Disambiguation output is clearly formatted

The ambiguity messages SHALL be presented in an aligned, easy-to-read form (columns for harness, scope, and path) and MUST explicitly state the flag(s) to use, consistent with omac's existing register call-out style.

#### Scenario: Output lists columns and the resolving command

- **WHEN** an ambiguity is reported
- **THEN** the output shows one row per candidate with harness/scope/path columns and a literal example command containing the resolving flag(s)

### Requirement: A skill name may be registered per harness

The registry SHALL allow the same skill name to be registered once per harness, each entry pointing at that harness's own skill directory, so registering a skill for one harness does not overwrite or invalidate its registration under another. A registration records the harness it belongs to. Entries written before harness scoping (no harness recorded) MUST continue to match any harness.

#### Scenario: Registering for one harness preserves the other

- **WHEN** `<name>` is registered for `opencode` and later registered for `claude-code`
- **THEN** both registrations coexist, each pointing at its harness's skill dir, and neither `omac start opencode` nor `omac start claude` reports the skill as unregistered

#### Scenario: Deregister can target a specific harness

- **WHEN** `<name>` is registered under two harnesses and the user runs `omac deregister <name> --harness claude`
- **THEN** only the Claude Code registration is removed; the OpenCode one remains

#### Scenario: Legacy unscoped entry still matches

- **WHEN** a registry entry was written before harness scoping (no harness field) and a harness-scoped run looks it up
- **THEN** the legacy entry is treated as a match for that harness
