## 1. Harness registry (Go core)

- [x] 1.1 Define a harness descriptor + registry in `internal/config` (fields: `Name`, `Aliases`, `InnerCmd`, `ServerLaunch`, `Bridge`)
- [x] 1.2 Register the `opencode` descriptor reproducing today's defaults (inner `opencode`; server-launch = inject `serve`; bridge = `.opencode/plugins`); aliases e.g. `oc`
- [x] 1.3 Register the `claude-code` descriptor (inner command; server-launch convention; bridge = `.claude/`); alias `claude`
- [x] 1.4 Add lookup-by-name-or-alias with a clear error listing supported harnesses for unknown names
- [x] 1.5 Refactor `internal/config/launcher.go` profile defaults to derive `InnerCmd` from the resolved descriptor instead of hard-coded `["opencode"]`

## 2. Positional harness parsing (CLI)

- [x] 2.1 In `runStart` (`internal/cli/start.go`), peek the first non-flag, pre-`--` token; consume it as the harness iff it matches the registry, else default to `opencode`
- [x] 2.2 Mirror the same parsing in `runServe` (`internal/cli/serve.go`)
- [x] 2.3 Ensure flags, `--inner`/`--sandbox` overrides, and `-- inner args` still parse correctly when a harness token is present
- [x] 2.4 Reject an unknown positional harness with a non-zero exit and a message listing supported names
- [x] 2.5 Update usage/help in `internal/cli/cli.go`, `start.go`, `serve.go` to show `omac start <harness>` / `omac serve <harness>`

## 3. Replace OpenCode-specific server-launch logic

- [x] 3.1 Replace `ensureServeSubcommand` in `internal/cli/serve.go` with a call into the resolved descriptor's `ServerLaunch`
- [x] 3.2 Verify OpenCode still injects `serve` and behavior is unchanged for the default harness
- [x] 3.3 Implement the Claude Code server-launch convention (or mark Claude `start`-only initially if no server mode exists)

## 4. Bridge interface + bridges

- [x] 4.1 Extract a single source of truth for the skills-manifest text and `OMAC_*` env naming, consumable by every bridge
      (Shared contract: control-plane manifest JSON + `internal/sandbox/launcher.go` env naming; both bridges render from the same activate-response shape.)
- [x] 4.2 Document the bridge interface (activate/deactivate via control plane; manifest as context; expose `OMAC_*_BASE`) — `.claude/README.md`
- [x] 4.3 Wire the existing OpenCode plugin to the shared manifest/env source (no behavior change) — plugin already renders from the control-plane manifest; unchanged
- [x] 4.4 Create Claude Code bridge assets under `.claude/` (settings + hook scripts)
- [x] 4.5 Implement Claude Code `SessionStart`/teardown hooks POSTing to `OMAC_CONTROL_BASE` `/__omac__/activate` and `/__omac__/deactivate`
- [x] 4.6 Inject the skills manifest into Claude Code session context from the shared source (SessionStart `additionalContext`)
- [x] 4.7 Expose per-skill base URLs under Claude Code using `OMAC_<MOUNT>_BASE` / `OMAC_G_<MOUNT>_BASE`, with the single-directory flat-alias fallback (process-level aliases from `start`/`serve`)
- [x] 4.8 Make both bridges inert when `OMAC_CONTROL_BASE` is unset

## 5. Skill agnosticism

- [x] 5.1 Audit the skill metadata schema (`internal/config/meta.go`, `omac.yaml`) for harness-specific assumptions — none found; the schema has no harness-coupled fields (skills speak only `OMAC_*`/REST). `.opencode` mentions are omac storage-path comments, not skill-author assumptions.
- [x] 5.2 Make the marketplace `/install` `target_path` guidance in the manifest harness-neutral (OpenCode plugin manifest + Claude bridge manifest both name the harness and point at the harness's skills dir)
- [x] 5.3 Neutralize harness-specific wording in example skills (`echo-rest`) — only benign storage-path comments remain; no functional coupling


## 6. Tests & verification

- [x] 6.1 Unit tests for harness resolution (positional token, alias, default-when-omitted, leading-flag fallback, unknown rejection, `--inner` precedence) — `internal/config/harness_test.go`, `internal/cli/harness_cli_test.go`
- [x] 6.2 Unit tests for per-harness server-launch (OpenCode injects `serve`; Claude uses its convention) — `TestApplyServerLaunch`
- [x] 6.3 Bridge verified against the live control plane: the Claude Code hook activates the directory and renders the real skills manifest (`additionalContext`). Full in-sandbox `echo-rest` (`/echo`, `/whoami`, `/tick`) under `omac start claude` requires the `claude` binary + nono runtime present at runtime; the omac plumbing (control plane, manifest, env aliases) is exercised and unchanged across harnesses.
- [x] 6.4 Marketplace `/install` is harness-neutral (manifest `target_path` guidance updated for both bridges); install + reload flow is unchanged by harness selection.
- [x] 6.5 Regression-check that `omac start` (no harness) is unchanged: default harness resolves to `opencode`, nono profiles keep `inner_cmd ["opencode"]`, and `ensureServeSubcommand` behavior is preserved by `ApplyServerLaunch` for opencode (existing + new unit tests green).

## 7. Documentation plan

- [x] 7.1 Rewrite `CREATING_A_SKILL.md` to a harness-agnostic core (target `OMAC_*`/REST; never assume a harness) with a new §0 "Running under a harness" covering OpenCode + Claude Code symmetrically and neutral install guidance
- [x] 7.2 Update `README.md` for the positional-harness UX (`omac start opencode|claude`), default-when-omitted, bridge table, and Claude Code setup
- [x] 7.3 Update `docs/MULTI_DIR_DESKTOP.md` with the harness-bridge interface (§0), the Claude Code bridge, and its `serve`/env limitations
- [x] 7.4 Add "Adding a new harness" guidance (registry descriptor fields + bridge-interface checklist) — README "Choosing an inner harness" + MULTI_DIR_DESKTOP §0
- [x] 7.5 Reconcile `oh-my-agentic-coder.md` §14.4 (and §4 harness-agnostic statement) with the implemented registry/positional-harness model
