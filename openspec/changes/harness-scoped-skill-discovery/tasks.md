## 1. Harness skills-dir identity

- [x] 1.1 Add per-harness skills-dir base to the `Harness` descriptor in `internal/config/harness.go` (opencode→`opencode`, claude-code→`claude`)
- [x] 1.2 Add a shared base constant (`agents`) and helpers: own-base for a harness, all-known harness bases, and "is this base in scope for harness H?"
- [x] 1.3 Unit-test the classification (own / shared / other-harness-excluded)

## 2. Harness-scoped skillsource

- [x] 2.1 Change `skillsource.Sources` to take a `config.Harness`; build roots from own-base + shared `agents` base only, workdir + global, excluding other harnesses' bases
- [x] 2.2 Update `Resolve` and `Discover` to take the harness and use the scoped sources
- [x] 2.3 Add `Candidates(workdir, harness, name)` returning ALL in-scope matches with harness + scope (workdir/global) + path
- [x] 2.4 Update package doc comment to describe harness scoping
- [x] 2.5 Update skillsource tests for scoping + exclusion + global layer

## 3. Register disambiguation

- [x] 3.1 Add `--harness <name>` and `--global` flags to `omac register` (`internal/cli/register.go`)
- [x] 3.2 Resolve the register harness (default opencode; `--harness` overrides) and use `Candidates`
- [x] 3.3 0 → not-found; 1 → register; >1 → narrow by `--harness`/`--global`, else print formatted ambiguity and exit non-zero
- [x] 3.4 Implement the aligned candidate table + literal resolving command (reuse `style.go`)
- [x] 3.5 Update register usage/help text
- [x] 3.6 Registry per-harness coexistence: add `Entry.Harness`, `FindForHarness`/`RemoveForHarness`, harness-keyed `Upsert` (legacy entries match any harness); record harness on register; add `--harness` to `omac deregister`

## 4. Wire scope through start / serve / control plane

- [x] 4.1 `start.go:findUnregisteredSkills` and discovery pass the resolved harness
- [x] 4.2 `serve.go` discovery (`activate`/`rediscover`/`autoRegister`) passes the harness; thread the harness into the serve server
- [x] 4.3 `start_reload.go` reload discovery passes the harness
- [x] 4.4 Confirm only in-scope skills are mounted + surfaced to the bridge manifest

## 5. Install target follows harness

- [x] 5.1 Reconcile `../opencode-nono/skill-marketplace` `/install` default to the active harness's skills dir (config `ASML_SKILLS_DIR` resolution + bridge `target_path` guidance)
- [x] 5.2 Update the bridge manifest install guidance (OpenCode plugin + Claude hook) to name the active harness's dir
- [x] 5.3 Keep `.agents/skills` as an explicit shared override; sync the edited skill to its harness dir(s)

## 6. Docs

- [x] 6.1 `README.md`: document per-harness skill dirs, the shared `.agents/skills`, and `register --harness`/`--global`
- [x] 6.2 `CREATING_A_SKILL.md`: where each harness loads `SKILL.md`; install per-harness vs shared
- [x] 6.3 `docs/MULTI_DIR_DESKTOP.md`: scoping note for discovery + bridge

## 7. Tests, build, validate

- [x] 7.1 Tests: discovery scope per harness, other-harness exclusion (workdir + global), `.agents` shared, precedence
- [x] 7.2 Tests: register ambiguity detection + `--harness`/`--global` resolution
- [x] 7.3 `go build ./...`, `go vet ./...`, `gofmt -l`, full `go test ./...`
- [x] 7.4 `openspec validate harness-scoped-skill-discovery --strict`
