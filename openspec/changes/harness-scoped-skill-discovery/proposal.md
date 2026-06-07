## Why

`SKILL.md` is read by each harness's own skill loader, not by omac: OpenCode scans `.opencode/skills` and Claude Code scans `.claude/skills` — non-overlapping native directories. omac, meanwhile, currently discovers skills from a flat union of `.agents/skills` + `.opencode/skills` across workdir and global layers, with no notion of which harness is running. This means omac will discover, register, and bridge a skill that the *active* harness cannot actually load (e.g. an OpenCode-only skill surfaced to a Claude session), and it forces users to keep duplicate copies with no way to say "this skill belongs to that harness." We need discovery to be harness-scoped so each harness only ever sees skills it can use, and we need registration to be unambiguous when the same skill name exists in more than one place.

## What Changes

- **Harness-scoped discovery roots.** The active harness determines which skill roots omac scans:
  - OpenCode → `.opencode/skills` (its own dir) **+** `.agents/skills` (shared/neutral).
  - Claude Code → `.claude/skills` (its own dir) **+** `.agents/skills` (shared/neutral).
  - A harness **never** scans another harness's dir. This applies to **both** the workdir layer and the user-global layer.
- **Bridge & start/serve honor the scope.** `omac start <harness>` / `omac serve <harness>` discover, auto-register, mount, and surface to the bridge **only** the active harness's in-scope skills. omac does not check or use registrations belonging to the other harness.
- **Disambiguation on register.** When `omac register <name>` finds the same skill name in more than one in-scope source:
  - **Harness ambiguity** (same name under two different harness dirs): by default, **stop and tell the user**, listing each candidate and the harness it belongs to; the user picks with **`--harness <name>`**.
  - **Scope ambiguity** (same name workdir vs global): by default, **stop and tell the user**; the user picks the global one with **`--global`** (workdir is the implied default when disambiguated, consistent with existing precedence).
  - Both messages are clearly formatted and tell the user exactly which flag resolves them.
- **Install target follows the active harness.** The marketplace `/install` default directory becomes the active harness's skills dir (`.opencode/skills` / `.claude/skills`) rather than a single hard-coded location; the bridge passes the harness's dir.
- **Docs + skill audit.** Update `README.md`, `CREATING_A_SKILL.md`, `docs/MULTI_DIR_DESKTOP.md`. Verify the `../opencode-nono/skill-marketplace` skill is consistent with the new install-target behavior.

## Capabilities

### New Capabilities
- `harness-scoped-discovery`: The rules for which skill source roots omac scans given an active harness — own-dir + shared `.agents`, never the other harness's dir — across workdir and global layers, and how start/serve/bridge consume that scope.
- `skill-registration-disambiguation`: The behavior of `omac register <name>` when a name is ambiguous across harness and/or scope, including the default "tell the user" output and the `--harness` / `--global` selectors.

### Modified Capabilities
<!-- The `inner-harness` capability from the support-claude-code-harness change
     is not yet archived into openspec/specs/, so there is no base spec to delta
     against. The harness's new "skills-dir identity" is captured as an ADDED
     requirement under `harness-scoped-discovery` instead of a MODIFIED delta. -->


## Impact

- **Code:** `internal/skillsource/skillsource.go` (harness-parameterized `Sources`/`Resolve`/`Discover`, add `.claude/skills` roots, exclude the inactive harness's dir), `internal/config/harness.go` (per-harness skills-dir identity), `internal/cli/register.go` (`--harness`/`--global` flags + ambiguity detection + formatted output), `internal/cli/start.go` & `internal/cli/serve.go` (pass the resolved harness into discovery), control-plane discovery in `serve.go`/`start_reload.go`.
- **Skill / marketplace:** `../opencode-nono/skill-marketplace` `ASML_SKILLS_DIR` default + `/install` target reconciled to "active harness dir"; bridge passes it.
- **Docs:** README, CREATING_A_SKILL, MULTI_DIR_DESKTOP.
- **Backward compatibility:** existing `.opencode/skills` + `.agents/skills` keep working for the default `opencode` harness (its scope is exactly those two). The change is additive (`.claude/skills`) plus a stricter exclusion (OpenCode no longer needs the other harness's dir, which never existed before). Registries (`sidecar.json`) are unchanged.
