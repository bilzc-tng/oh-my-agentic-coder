## Context

Three independent consumers find skills, each with its own search paths:

1. **omac** (serves sidecar routes) — currently scans a flat union of `.agents/skills` + `.opencode/skills` across workdir and user-global layers (`internal/skillsource/skillsource.go`), harness-unaware.
2. **OpenCode** (reads `SKILL.md`) — its own loader scans `.opencode/skills` (workdir) and `~/.config/opencode/skills` (global).
3. **Claude Code** (reads `SKILL.md`) — verified from the binary: it scans `.claude/skills` (workdir) and `~/.claude/skills` (global), and nothing else.

Because (2) and (3) are non-overlapping, omac today can discover/register/bridge a skill the active harness cannot load. The harness is already a first-class concept (`internal/config/harness.go`, positional `omac start <harness>`); this change threads it into discovery.

Call sites that discover skills: `register.go` (`skillsource.Resolve`), `start.go:findUnregisteredSkills` (`Discover`), `serve.go` (`Discover` in `activate`/`rediscover`, `autoRegister`). The registry files (`.opencode/sidecar.json` workdir, `~/.config/omac/sidecar.json` global) are omac's own and are **not** harness-scoped — only skill *source* discovery is.

## Goals / Non-Goals

**Goals:**
- Each harness sees only skills it can load: its own dir + the shared `.agents` dir; never the other harness's dir. Applies to workdir and global layers.
- start/serve/bridge discover, auto-register, mount, and surface only in-scope skills; omac never inspects the other harness's registrations for the active run.
- `omac register <name>` is unambiguous: when a name exists under multiple harnesses or multiple scopes, stop and tell the user (nicely formatted), with `--harness <name>` and `--global` to resolve.
- Marketplace `/install` defaults to the active harness's skills dir.
- Keep the default `opencode` harness's behavior effectively unchanged (its scope is `.opencode/skills` + `.agents/skills`, which is what it discovered before).

**Non-Goals:**
- Changing the registry file format/locations.
- Auto-mirroring/symlinking skills between harness dirs (explicitly rejected earlier in favor of per-harness dirs).
- Making Claude/OpenCode scan extra dirs (we cannot reconfigure their native loaders; omac adapts to where each already looks).

## Decisions

### Decision 1: Harness owns a `SkillsDirs` identity; skillsource is parameterized by harness

Extend the `Harness` descriptor (`internal/config/harness.go`) with the skills-dir base names it owns:

- `opencode` → own base `opencode` (`.opencode/skills`, `~/.config/opencode/skills`, legacy `~/.opencode/skills`).
- `claude-code` → own base `claude` (`.claude/skills`, `~/.claude/skills`).

The **shared** base `agents` (`.agents/skills`, `~/.config/agents/skills`, `~/.agents/skills`) is scanned by every harness. `skillsource.Sources(workdir, harness)` builds roots from: the harness's own base + the shared `agents` base, in both workdir and global layers, **excluding** every other harness's base. Precedence within a layer: own-harness dir ranks above shared `.agents` (so a harness-specific skill overrides a neutral one), workdir over global (unchanged).

**Alternative considered:** keep `Sources(workdir)` flat and filter at the call site. Rejected — discovery scope is a harness property; centralizing it in skillsource keeps every call site correct by construction.

### Decision 2: skillsource API takes an explicit harness; callers pass the resolved one

`Sources`, `Resolve`, `Discover` gain a `config.Harness` parameter. `register.go` resolves the harness from `--harness` (default opencode), `start.go`/`serve.go` pass the already-resolved positional harness. There is no implicit global default deep in skillsource — the CLI is the single place harness is resolved.

### Decision 3: Register disambiguation — detect, stop, tell, resolve

`skillsource` grows a `Candidates(workdir, harness, name)` that returns **all** in-scope matches (across harness-of-record and scope) rather than just the first. `register.go`:

- 0 candidates → not-found (as today).
- 1 candidate → register it (as today).
- >1 candidates → **ambiguous**: print a formatted table of candidates (name, harness, scope=workdir/global, path) and the exact flags to disambiguate, then exit non-zero **unless** the provided `--harness` / `--global` narrow it to exactly one.
  - `--harness <name>` restricts candidates to that harness's scope.
  - `--global` restricts to the user-global layer (default is workdir when otherwise ambiguous).

Because a single register invocation targets one harness's scope, "harness ambiguity" only arises when the **shared** `.agents` dir and the harness's own dir both contain the name — or, more importantly, when the user wants to register *for the other harness*. `--harness` selects which harness's scope to resolve in. Scope ambiguity (workdir vs global within the chosen harness scope) is resolved by `--global`.

**Formatting:** a bordered/aligned block consistent with the existing register "NEXT STEP" callout style (`internal/cli/style.go` if present, else aligned text), so the user sees something like:

```
Multiple skills named "slack" found:
  HARNESS      SCOPE     PATH
  opencode     workdir   .opencode/skills/slack
  claude-code  workdir   .claude/skills/slack
Pick one:  omac register slack --harness claude-code
```

### Decision 4: Install target = active harness's skills dir

The marketplace skill's `/install` default (`ASML_SKILLS_DIR`) resolves to the active harness's workdir skills dir. The bridge already injects the manifest; it will pass the harness's dir as the `target_path` guidance (OpenCode → `.opencode/skills`, Claude → `.claude/skills`). The skill keeps `.agents/skills` as a valid explicit override. This keeps installed skills in the dir the running harness actually loads.

### Decision 5: Bridge surfaces only in-scope skills

Since start/serve discover only in-scope skills, the manifest the bridge renders already contains only loadable skills. No bridge-script change is needed beyond the install-target guidance (Decision 4) — the scoping happens upstream in omac.

## Risks / Trade-offs

- **[A neutral `.agents/skills` skill plus a harness-specific one of the same name]** → resolved by precedence (own-harness dir wins) and surfaced by register disambiguation when registering. Mitigation: documented precedence + the ambiguity message.
- **[Existing global skill installed under `~/.config/opencode/skills` becomes invisible to Claude]** → intended: Claude can't load it anyway. Mitigation: docs tell users to install per-harness or into `.agents/skills` for shared skills; marketplace install now targets the active harness's dir.
- **[skillsource signature change ripples to call sites/tests]** → contained: 3 call sites + tests. Mitigation: compiler enforces it; keep a thin helper if needed.
- **[Hidden behavior change for current opencode users]** → minimal: opencode scope = `.opencode/skills` + `.agents/skills`, exactly today's effective set minus the (previously also-scanned) global `agents`/`opencode` which remain in scope. No skill that worked before disappears for opencode.

## Migration Plan

1. Add `SkillsDirs`/base identity to `Harness`; add shared/own/excluded classification.
2. Parameterize skillsource by harness; update the 3 call sites to pass the resolved harness.
3. Add `Candidates` + register disambiguation with `--harness`/`--global` and formatted output.
4. Reconcile marketplace install target; update `../opencode-nono/skill-marketplace`.
5. Docs.
6. Tests: discovery scoping per harness, exclusion of the other harness's dir, ambiguity detection/resolution.

**Rollback:** revert the skillsource signature change; the harness descriptor addition is inert without it.

## Open Questions

- Should `--harness` on `register` default to the value of a harness positional if we later add one to `register`, or always default to `opencode`? (Current: defaults to `opencode`; `--harness` overrides.)
- For global installs of shared skills, do we want a first-class `.agents/skills` "shared" install path surfaced in the marketplace manifest, in addition to the active harness's dir? (Deferred; `.agents` remains a manual override.)
