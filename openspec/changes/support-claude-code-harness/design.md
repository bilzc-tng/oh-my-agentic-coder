## Context

omac bridges host-side HTTP "sidecar" services into a sandboxed coding agent through a single facade (Unix socket + loopback TCP), with per-skill secrets injected only into sidecars. The Go core is deliberately harness-agnostic: `omac start` execs whatever `inner_cmd` resolves to (`internal/cli/start.go`), and the facade/supervisor/sandbox/registry/control-plane subsystems speak a stable contract — `OMAC_*` environment variables (`internal/sandbox/launcher.go`) plus a `/__omac__/{activate,deactivate,reload,reload-global,dirs,global}` control plane.

Three OpenCode-specific couplings remain:

1. **Launcher-profile defaults** — `internal/config/launcher.go` hard-codes `InnerCmd: ["opencode"]` in the `nono` profiles.
2. **Server-subcommand injection** — `internal/cli/serve.go:ensureServeSubcommand` checks for the `opencode` basename and injects the `serve` subcommand.
3. **The OpenCode plugin** — `.opencode/plugins/omac-multidir.ts`, loaded automatically by OpenCode, is the entire client-side integration (OpenCode plugin API: hooks `event`, `experimental.chat.system.transform`, `shell.env`, and `client.session.get`). It drives `/__omac__/activate|deactivate` on session lifecycle, injects the "## omac skills available in this workspace" manifest, and injects per-session `OMAC_<MOUNT>_BASE` / `OMAC_G_<MOUNT>_BASE`.

The CLI dispatches subcommands by the first positional token (`internal/cli/cli.go:75`), and `start`/`serve` then parse their own flags (`internal/cli/start.go:28`). This makes a leading positional harness token (`omac start <harness> [flags]`) a clean fit. The design doc `oh-my-agentic-coder.md` §4/§17 already anticipates a swappable harness.

## Goals / Non-Goals

**Goals:**
- Select the inner harness by positional subcommand: `omac start opencode`, `omac start claude` (and `omac serve <harness>`). No `--harness` flag.
- Make the harness set **extensible**: adding another agentic coder is registering one descriptor, not editing dispatch/launch sites.
- Deliver a Claude Code bridge at parity with the OpenCode plugin.
- Define a general **bridge interface** so OpenCode and Claude Code are two implementations of one contract, and future harnesses follow the same shape.
- Establish and document that **skills are harness-agnostic by contract** and verify it for both Claude and OpenCode.
- Keep the Go core and the `OMAC_*` / `/__omac__/` contract unchanged; preserve backward compatibility (omitting the harness ⇒ `opencode`).
- Produce a concrete documentation plan: harness-agnostic skill-authoring guide plus README / multi-dir updates.

**Non-Goals:**
- Rewriting the sandbox runtime (nono) interaction.
- Changing the skill metadata schema beyond what is needed to remove harness-specific assumptions.
- Auto-detecting which harness is installed (selection is explicit; default stays `opencode`).
- Implementing every conceivable harness now — only OpenCode (existing) and Claude Code (new), with the registry/bridge shaped so others are easy to add.

## Decisions

### Decision 1: Positional harness subcommand, not a flag

The harness is the **first positional argument** to `start`/`serve`: `omac start <harness> [flags] [-- inner args]`. Parsing: in `runStart`/`runServe`, peek at the first non-`--`, non-flag token; if it matches a known harness name/alias in the registry, consume it as the harness and pass the remainder to the existing flag parser. If the first token is a flag (begins with `-`) or absent, fall back to the default harness (`opencode`).

- `omac start` → default `opencode` (unchanged behavior).
- `omac start opencode` → explicit OpenCode.
- `omac start claude` → Claude Code.
- `omac start claude --verbose -- --model X` → harness `claude`, omac flag `--verbose`, inner args after `--`.

**Alternatives considered:** (a) `--harness` flag — rejected per product direction; positional reads as a mode/persona and matches how users think ("start claude"). (b) separate top-level subcommands `omac claude` / `omac opencode` — rejected: it duplicates `start`/`serve`/flag surface per harness and fights the existing subcommand table in `cli.go:116`.

**Edge case:** a user whose inner command literally needs a leading bareword that collides with a harness name uses `--` (everything after `--` is verbatim inner args) or `--inner` to disambiguate.

### Decision 2: A harness registry as the single source of harness knowledge

Add a registry in `internal/config`: `name`, `aliases` (e.g. `claude` ⇄ `claude-code`, `oc` ⇄ `opencode`), `InnerCmd` default, `ServerLaunch` convention (how `serve` makes a long-lived server), and `Bridge` descriptor (which client assets/dir). `start`/`serve`/`launcher` consult the registry; `ensureServeSubcommand` is replaced by `harness.ServerLaunch`. Adding a harness = appending one descriptor (+ its bridge assets).

**Alternative considered:** scattered `if name == ...` branches. Rejected — does not scale to "more agentic harnesses" and spreads knowledge across files.

### Decision 3: General bridge interface; OpenCode + Claude Code as implementations

Define what every bridge must do against the stable contract: (1) call `/__omac__/activate|deactivate` on session lifecycle using `OMAC_CONTROL_BASE`; (2) surface the skills manifest as agent context; (3) expose `OMAC_<MOUNT>_BASE` / `OMAC_G_<MOUNT>_BASE` to the agent. The OpenCode bridge is the existing plugin. The Claude Code bridge uses Claude Code's native settings/hooks (e.g. `SessionStart`/teardown hooks POSTing to the control plane; manifest delivered as session context / generated context file; env exposed via the supported passthrough). Each bridge consumes the **same source of truth** for manifest text and env naming (shared generator) to prevent drift.

**Alternative considered:** porting the OpenCode plugin API onto Claude Code via a shim. Rejected — couples two third-party APIs and is brittle; native extension points per harness are simpler.

### Decision 4: Skills are harness-agnostic by contract

A skill MUST NOT depend on any harness-specific behavior; it speaks only the documented `OMAC_*`/REST/facade contract. The change audits the skill schema and docs to remove OpenCode-specific assumptions (e.g. `.opencode`-only path guidance, OpenCode-only manifest wording) and makes the marketplace `target_path` guidance harness-neutral. This is the load-bearing guarantee that "the spec covers both Claude and OpenCode."

### Decision 5: Harness-neutral paths; bridge assets in native dirs

Treat `.agents/` as the neutral skills home (already searched and ranked above `.opencode/`), keep `.opencode/` working, and place each harness's bridge assets in its native location (`.opencode/plugins/`, `.claude/`). No existing path is removed.

### Decision 6: Documentation & skill-authoring plan (deliverable, not just code)

- `CREATING_A_SKILL.md`: restructure to a harness-agnostic core ("write to the `OMAC_*`/REST contract; never assume a harness") with a short, symmetric "Running under OpenCode" / "Running under Claude Code" section. Replace OpenCode-only examples with neutral ones.
- `README.md`: document the positional-harness UX (`omac start opencode|claude`), default behavior, and Claude Code setup.
- `docs/MULTI_DIR_DESKTOP.md`: document the Claude Code bridge and any `serve` differences/limitations vs OpenCode.
- `oh-my-agentic-coder.md` §4/§17: update to reference the implemented registry/bridge model.
- A short **"Adding a new harness"** note (registry descriptor + bridge interface checklist) so the extensibility goal is documented.

## Risks / Trade-offs

- **[Positional token vs inner bareword collision]** → A harness name as the first token could shadow an intended inner bareword. Mitigation: only consume the first token if it matches a known harness AND is not preceded by flags; document `--`/`--inner` to disambiguate; cover with tests.
- **[Claude Code lacks a true per-session `shell.env` hook]** → Mitigation: use the single-directory flat aliases already exported by `start`/`serve`; document the limitation; pursue full multi-dir parity if/when Claude Code exposes the hook.
- **[Claude Code system-prompt/context injection differs from OpenCode]** → Mitigation: deliver identical manifest text via the closest supported context mechanism; verify with the echo-rest smoke test.
- **[Claude Code `serve`/headless mode may differ from `opencode serve`]** → Mitigation: encode the launch convention in `ServerLaunch`; if Claude Code has no server mode, support Claude under `start` first and document `serve` as OpenCode-first.
- **[Bridge drift over time]** → Mitigation: shared manifest/env generator as single source of truth; bridge-interface checklist in docs.
- **[Doc rewrite churn]** → Restructuring `CREATING_A_SKILL.md` risks dropping detail. Mitigation: keep all existing normative content; only relocate/neutralize harness-specific wording; review diff against the current file.

## Migration Plan

1. Land harness registry + positional parsing with `opencode` default and aliases → no behavior change when harness omitted.
2. Add Claude Code descriptor, `ServerLaunch` metadata, and `.claude/` bridge assets behind `omac start claude`.
3. Audit + neutralize skill schema/docs; make marketplace `target_path` guidance harness-neutral.
4. Rewrite/extend docs (`CREATING_A_SKILL.md`, `README.md`, `docs/MULTI_DIR_DESKTOP.md`, `oh-my-agentic-coder.md`).
5. Verify end-to-end with `echo-rest` (JSON / `/whoami` secret / `/tick` SSE) and `skill-marketplace` install under both `claude` and `opencode`.

**Rollback:** harness defaults to `opencode`; not passing `claude` reverts to current behavior with no data migration.

## Open Questions

- Does the locally-run Claude Code CLI expose a stable server/headless mode suitable for `omac serve claude`, or should Claude initially be `start`-only?
- What is Claude Code's supported mechanism for per-session shell env and additional system/context text (hooks vs settings vs generated context file), and does it support per-directory differentiation for multi-dir parity?
- Should bridge assets be auto-installed by omac (like the OpenCode plugin is auto-discovered) or committed by the user into `.claude/`?
- Alias policy: which short names are reserved (`claude`, `opencode`, `oc`, …) and how are collisions across future harnesses prevented?
