# Upstream merge guide

Maintain this guide with every change to a fork integration point, replacement
codepath, or merge requirement. Link to [Roadmap.md](Roadmap.md) for behavior and
[Tracker.md](Tracker.md) for the current item's remaining work. Keep this guide
about the code that exists, not speculative implementation tasks.

## Baseline and current divergence

- Upstream: `https://github.com/jesseduffield/lazygit.git`, currently remote
  `origin`. No separate hosted fork remote has been configured by this work.
- Last inspected upstream base:
  `4e1eebe278790f257a875da6633234391a5a85b3`.
- Fork planning/research lives under `new-features/`; root `AGENTS.md` is
  fork-specific. Application divergences are recorded below as they land.
- Treat root `AGENTS.md` as fork-owned. Review upstream guidance changes for
  relevant technical constraints and incorporate them deliberately while
  retaining this fork's concise workflow and usability principles.

## Live integration record

### Explicit-action defaults: audited integration seams

- `pkg/config/fork_defaults.go:applyForkDefaults` replaces the four upstream
  automation defaults. The constructor calls this overlay with a comment
  explaining the replacement; upstream's default table stays intact. Preserve
  this hook when merging constructor changes, and include new constructors if
  upstream adds any. `fork_defaults_test.go` protects all platform defaults,
  independent ref polling, config layering, opt-ins, reload and file contents.
- `pkg/config/user_config.go:GetDefaultConfigForPlatform` is the shared
  constructor for app loading, platform variants, dummy configs and generated
  docs/schema. Apply fork defaults here before `loadUserConfig` overlays YAML,
  not after loading. Loading/reloading starts with fresh defaults; no migration
  or rewrite of existing config is needed. Later config files win.
- `pkg/gui/background.go:startBackgroundRoutines` independently gates fetch,
  file polling and external-ref detection. Startup fetch only runs inside its
  enabled routine. `Gui.onSwitchToNewRepo` checks auto-fetch before retriggering.
  Preserve the independent read-only ref detector.
- `Gui.checkForChangedConfigsThatDontAutoReload` warns that fetch/file/ref
  polling settings and intervals need restart. Forwarding and conflict-staging
  settings are read live; do not introduce a separate fork reload policy.
- `Gui.runSubprocessWithSuspenseAndRefresh` explicitly refreshes on return.
  Disabling periodic file polling must not suppress this or action refreshes.
- `BranchesHelper.PostFetchRefresh` invokes `AutoForwardBranches` after both
  manual and background fetch. That method returns for `none`; no handler
  replacement is needed to prevent forwarding after a manual fetch.
- `RefreshHelper.refreshStateFiles` gates resolved-file staging on
  `AutoStageResolvedConflicts`. Its continue prompt depends on the previous
  conflict count collected within that gate, so disabling it also disables
  the automatic continue prompt, as upstream's option documents.
- Existing integration scenarios specifically testing automatic staging or
  forwarding must opt in explicitly. Default-policy scenarios should not set
  the option to its default: they must detect a regression in the constructor.
- `pkg/gui/fork_background_test.go` verifies the three independent startup
  gates and no-op fetch triggering deterministically with invalid intervals;
  it deliberately starts no background workers. Conflict integration scenarios
  explicitly opt into staging when that is their subject; `ResolveNoAutoStage`
  uses the fork default. `FetchAndAutoForwardBranchesNone` uses the forwarding
  default and verifies that a manual fetch leaves local branch tips alone.
- Generation uses `go generate ./...`: development defaults are written to
  `docs-master/Config.md` and `schema-master/config.json`, not the release
  snapshots in `docs/` and `schema/`. The fork-defaults explanation outside the
  generated YAML in `docs-master/Config.md` is hand-maintained. Keybinding
  cheatsheets should not change for this defaults-only item.
- `pkg/integration/components/eventual_content.go` adds a bounded UI-thread
  buffer observation for genuine background events; it never triggers refresh.
  Its unit test verifies retries and UI-thread reads. New sync scenarios
  `ExplicitActionDefaults` and `AutomationOptIn` use local bare remotes.
  The shared test YAML explicitly disables polling, so the default-policy
  scenario restores the three routine gates from `GetDefaultConfig` rather
  than mistaking harness overrides for application defaults.

### Item 1 verification (2026-10-21)

- Passed: `go build -gcflags='all=-N -l'`, isolated-config TUI start/quit,
  `--config` output, `go generate ./...`, `env -u NO_COLOR go test ./... -short`,
  `scripts/gofumpt-check.sh` and `scripts/golangci-lint-shim.sh run` (zero issues).
- Full headless attempt: 690 registered scenarios, one version-gated skip,
  two fixture failures. The custom-patch reverse/conflict scenario needed an
  explicit staging opt-in; the new default-policy scenario inherited the
  harness's disabled external-ref polling. Both fixtures were corrected.
- Corrected headless rerun passed in 14.924s: `TestIntegration/(sync|conflicts|main_view)/(explicit_action_defaults|automation_opt_in|fetch_and_auto_forward_branches_.*|apply_custom_patch_in_reverse_with_conflict|resolve_no_auto_stage)$`,
  with `-parallel 4`. This covers 11 scenarios, including all seven forwarding
  variants. Other full-run scenarios passed unchanged; the full suite was not
  repeated after the fixture-only corrections. The earlier focused conflict
  and no-forwarding run also passed. Do not describe the initial full attempt
  as a single green full-suite run.
- Used `justfile` command equivalents because `just` is absent. No system
  dependencies were installed; the pinned Go lint tool downloaded successfully.

### Command identity foundations (item 2 in progress)

- `types.Binding` and `types.MenuItem` have additive `CommandID` and `Verb`
  fields. Preserve `MenuItem.ID()` as row identity: many rows may share an
  operation ID while carrying different object arguments. The rationale and
  dispatch audit live in [command-identity-design.md](command-identity-design.md).
- `config.CustomCommand.ID` is an optional YAML field. `UserConfig.Validate`
  calls `validateCustomCommandIdentities` once over the whole nested tree,
  alongside upstream's existing definition validation. Preserve both checks
  when merging config changes; a command menu now permits `id` as metadata.
- `pkg/config/custom_command_identity.go` owns explicit namespaces and the
  version-1 fallback digest payload. No command contents are persisted. Changing
  the payload's schema or normalization can change fallback identities; explicit
  IDs survive definition changes. Keep the stability/privacy/validation tests.
- `custom_commands.KeybindingCreator` assigns one identity to all contexts of
  a definition. `Client` propagates identity to custom menu openers and leaves.
  Existing handlers, key parsing and context filtering are unchanged. The
  invocation verbs classify custom commands rather than inferring arbitrary
  shell semantics. Tests capture menu construction without executing shell code.
- Changing `CustomCommand` requires regenerating development docs and schema
  with the existing generation recipe. Never add command text or rendered form
  values to identity/usage persistence when extending this metadata.

## Reviewing an upstream merge

1. Record the upstream commit being integrated and review changes around all
   live integration points, including changes that merge without conflicts.
2. Preserve fork-owned additions and re-evaluate the narrow hooks into upstream.
   If upstream now solves the same problem, assess consolidation against the
   roadmap's agreed behavior instead of retaining redundant code.
3. Check replacement comments and this guide against the resulting code. Update
   obsolete paths, behavior descriptions, generated-file requirements and
   regression scenarios in the merge/resolution commit.
4. Regenerate affected outputs and run relevant regression checks plus the
   repository's build, unit-test, formatting and lint recipes. Validate user
   control and modal flows as well as Git results. Record the integrated base,
   verification results and any unresolved issue here before calling the merge
   complete.
