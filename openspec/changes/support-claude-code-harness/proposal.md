## Why

Today omac hard-codes OpenCode as the inner harness: launcher-profile defaults (`InnerCmd: ["opencode"]`), the `omac serve` server-subcommand logic (`opencode` → inject `serve`), and a single OpenCode-only plugin (`.opencode/plugins/omac-multidir.ts`) that delivers per-directory skill activation, the system-prompt skills manifest, and per-session `OMAC_*` env injection. Teams that standardize on Claude Code — or any other agentic coder — cannot use the sandbox, the skill marketplace, or any skills. The Go core is already harness-agnostic (it execs whatever `inner_cmd` resolves to and exposes a stable `OMAC_*` + `/__omac__/` contract), so the remaining work is to make the harness an explicit, extensible choice and to supply per-harness bridges — starting with Claude Code alongside OpenCode.

## What Changes

- Select the inner harness as a **positional subcommand**: `omac start claude`, `omac start opencode` (and likewise `omac serve claude` / `omac serve opencode`). No `--harness` flag. Omitting the harness keeps today's behavior (defaults to `opencode`).
- Introduce a **harness registry** built for extensibility: adding a new agentic harness (e.g. `cursor`, `aider`, `gemini`, …) is a matter of registering one descriptor (name + aliases, inner command, server-launch convention, bridge), not editing call sites. `omac start` with an unknown/known harness name is validated against this registry.
- Replace the hard-coded `opencode → serve` subcommand injection with per-harness "server-launch" metadata, and add the Claude Code launch convention.
- Ship a **Claude Code bridge** at parity with the OpenCode plugin (per-directory skill activation/promotion, system-prompt skills manifest, per-session skill base-URL injection), built on Claude Code's native extension mechanism (settings/hooks), not the OpenCode plugin API.
- Make discovery, config, and registry paths **harness-neutral**: keep `.opencode/` working and treat the already-parallel `.agents/` paths as the neutral home; bridge assets live under each harness's native dir (`.opencode/`, `.claude/`, …).
- Establish that **skills are harness-agnostic by contract**: a skill MUST work unchanged under any harness. Update skill metadata/docs so no skill embeds OpenCode- or Claude-specific assumptions; the manifest/`target_path` guidance becomes harness-neutral.
- **Documentation plan:** rewrite `CREATING_A_SKILL.md` to be harness-agnostic with explicit Claude + OpenCode coverage, update `README.md` and `docs/MULTI_DIR_DESKTOP.md` for the positional-harness UX and Claude Code, and reconcile `oh-my-agentic-coder.md` §4/§17.

## Capabilities

### New Capabilities
- `inner-harness`: A first-class, extensible harness model selected by positional subcommand (`omac start <harness>`), backed by a harness registry that decouples omac from any single agentic coder and defines what each harness provides (name/aliases, inner command, server-launch convention, bridge).
- `agent-bridge`: The per-harness client-side integration that wires an agentic coder into the omac control plane and `OMAC_*` env contract — per-directory skill activation/promotion, system-prompt skills manifest injection, and per-session skill base-URL injection. Defines the bridge interface generally and the concrete OpenCode and Claude Code bridges at parity.
- `agnostic-skills`: The requirement and authoring guidance that skills are harness-agnostic by contract, plus the documentation deliverables (skill-authoring guide, README, multi-dir guide) that cover Claude and OpenCode equally.

### Modified Capabilities
<!-- No existing specs in openspec/specs/ yet; all capabilities are new. -->

## Impact

- **Code (Go):** `internal/cli/cli.go` (usage), `internal/cli/start.go` and `internal/cli/serve.go` (consume a leading positional harness token before flag parsing; resolve via registry), new harness registry in `internal/config`, `internal/config/launcher.go` (profile defaults derive `InnerCmd` from the resolved harness), `internal/cli/serve.go:ensureServeSubcommand` (→ per-harness server-launch metadata). Harness-agnostic core (`facade`, `supervisor`, `sandbox`, `registry`, `skillsource`, control plane) reused unchanged.
- **Bridges:** existing OpenCode plugin retained as the `opencode` bridge; new Claude Code bridge assets under `.claude/`; a documented bridge interface so further harnesses can add their own.
- **Env/contract:** `OMAC_*` naming in `internal/sandbox/launcher.go` reused as-is; every bridge reads the same vars.
- **Skills & marketplace:** no per-skill code changes; skills already speak the `OMAC_*`/REST contract. Marketplace install `target_path` guidance becomes harness-neutral.
- **Docs:** `CREATING_A_SKILL.md` (harness-agnostic rewrite, Claude + OpenCode), `README.md`, `docs/MULTI_DIR_DESKTOP.md`, `oh-my-agentic-coder.md` §4/§17.
- **Backward compatibility:** omitting the harness defaults to `opencode`; existing `.opencode/` layouts, registries, and the OpenCode plugin keep working unchanged.
