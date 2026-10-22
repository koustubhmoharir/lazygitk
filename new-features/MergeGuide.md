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
- Built-in bindings in the basic-commit, local-branch, remote-branch, tag and
  stash controllers carry explicit operation IDs/verbs. Preserve the literals
  through upstream key/description/handler renames rather than recomputing IDs.
  Equivalent `RefsHelper.NewBranch`, reset-menu and ref-difftool invocations
  share identity across these controllers. Commit difftool identity remains
  separate because its default endpoints differ. This is metadata only; no
  keys, guards or Git handlers are replaced by the annotations.
- `Binding`/`MenuItem` also offer optional live metadata callbacks through
  `GetCommandID`/`GetVerb`. `controllers/diff_command_identity.go` distinguishes
  the existing main-diff staging, unstaging, patch-selection and commit-discard
  operations. Only their binding metadata hooks in `MainViewController` change;
  handlers and disabled reasons remain upstream-owned. If upstream adds another
  `FocusedMainViewDiffSource` implementation or changes pane semantics, review
  this resolver. Its 12-case matrix and callback tests protect the classification.
- `controllers/file_command_identity.go` classifies the single-file conflict
  dialog using the existing predicate. Review it if upstream changes primary
  staging/conflict dispatch. The selected path is never part of identity.
- Built-in menu leaves and the remaining binding factories carry literal IDs
  or finite internal-enum IDs. Parameter-only prompts and informational rows
  intentionally have none; see the identity design's dynamic-factory audit.
- `OptionsMenuAction` delegates to fork-owned `command_palette.go`, replacing
  translated-description deduplication with resolved identity deduplication.
  Palette copies merge shortcut hints without mutating dispatch bindings and
  preserve the preferred binding's handler/disabled check. `MenuItem.Binding`
  marks a palette proxy for future usage instrumentation; menu accelerator
  wrappers propagate live item identity without changing row identity or guards.
- `JumpToSideWindowController` retains all parameterized panel definitions when
  fewer keys are configured, instead of truncating definitions to key count.
  Positional dispatch is unchanged. Review this replacement if upstream changes
  panel layout or shortcut allocation; platform/locale/unbound metadata tests
  must remain valid. This does not add visible panel-chooser menu entries.
- The inventory generator and annotation-only AST checker live under
  `new-features/inventory` with standalone, explicitly invoked tests. Regenerate
  the source snapshot after command changes. Five unannotated parameter/info
  factories are intentional; do not add IDs derived from their user input.
- Item 2 verification: full short tests, generator/checker tests, generation,
  formatting and lint passed. The corrected 14-scenario headless selection
  passed in 11.796s, covering palette filtering/navigation, custom submenus and
  unbound branch rename through the normal prompt and Git result. The first
  attempt's new test omitted the Local section header; its assertion was fixed.
  This was a targeted run, not a new full-suite run.
  Final post-review debug build, full short tests, formatting and lint also
  passed before advancing to item 3.

### Direct historical comparison seams (item 3 complete)

- [direct-diff-design.md](direct-diff-design.md) records endpoints, literal
  paths, temporary-file ownership, default-key collision handling and the
  TortoiseGit guidance consulted before implementation.
- `oscommands/subprocess_exit_status.go` owns opt-in success normalization.
  `CmdObj` adds only a private allowlist; normal runner behavior is unchanged.
  `Gui.runSubprocess` calls `SubprocessError` immediately after `exec.Cmd.Run`,
  before its error/return prompt. Keep that ordering on merge: suppressing an
  accepted difference status after prompting is too late. No command accepts
  nonzero results unless its builder explicitly opts in. Tests cover wrapped
  exit errors, command-local copied lists, start errors and preserved failures.
- `historical_diff.go` is an additive Git backend; the existing sticky difftool
  builder remains intact. Literal path lookup, raw blob streaming and prepared
  comparison ownership are tested separately from UI flow. Preserve its use of
  the repo-bound builder/environment and explicit worktree cwd, not process cwd.
- `historical_diff_actions.go` adds bindings through two hooks in the shared
  commit and commit-file controllers. Existing sticky actions get a distinct
  display label but retain their IDs/handlers. New guards/captured values must
  not silently select another ref/file while a prompt is open. Filesystem
  completion is separate from upstream's tracked-only suggestion trie.
- `KeybindingUniversalConfig` adds three diff fields; defaults and displaced
  shortcut removal live only in the fork overlay. Preserve explicit config
  loading and regenerate development docs/schema/cheatsheets after changes.
  New English strings fall back normally for untranslated locales.
- The shared headless fixture supplies non-conflicting F6–F10 shortcuts for
  legacy operations whose old letters are now reserved. These are test bindings,
  not fork defaults. Default-keymap scenarios must restore the constructor's
  keybinding config (as the historical-diff scenario does). Keep new direct-diff
  bindings enabled in legacy tests rather than hiding collisions by disabling them.
- [Fork feature usage](../docs-master/Fork_Features.md) documents the new flows
  and transitional shortcuts; update it when behavior changes.
- Fork-owned command-hint helpers replace the paste-key-only tooltip and mode
  hint when paste is unbound. Keep the hooks in `BasicCommitsController` and
  `OptionsMapMgr`; otherwise reserved diff keys leave blank instructions in the
  copied-commit workflow. Explicit paste bindings keep their normal hints.
  `OptionsMapMgr` also prioritizes working-tree comparison in the display-only
  hint list, without reordering installed bindings or dispatch precedence.
- Verification so far: debug build, full short tests, generation, standalone
  inventory/checker tests, formatting and lint passed. The affected-operation
  headless selection matched 99 scenarios and passed in 104.892s. A final
  15-scenario palette/diff selection passed in 24.201s, including normal terminal
  return, fixed endpoints under sticky/reverse mode, both letter cases, ignored
  destination selection, staged/index preservation, failure/recovery and cleanup.
  Exact controller invocation tests additionally cover directory scope, recognized
  rename paths and historical deletion. Final content-matrix review passed for
  Unicode/nested/spaced/option-like/literal-magic paths, empty blobs and a large
  binary blob. The final short-test/format/lint pass was green. Verification is
  Linux/Git 2.51.1, not a GUI-tool or cross-platform matrix; do not describe the
  targeted headless runs as a new full-suite run.

### Ref-only branch placement (item 4 complete)

- [branch-placement-design.md](branch-placement-design.md) and the linked probe
  report explain why ordinary `git branch -f`/OID-only updates are insufficient.
  `branch_placement.go` adds immutable prepare/apply snapshots and literal-name
  validation; existing creation/checkout/reset paths are not replaced.
- `branch_ref_transaction.go` uses the repo-bound command builder but owns the
  interactive process lifecycle instead of the runner's whole-command retry.
  Its wait goroutine uses upstream's `utils.Safe` terminal-restoration wrapper.
  Git's prepared transaction holds the target ref lock during the final type,
  value and occupancy checks. Preserve EOF-abort, ACK/hook-output handling,
  bounded diagnostics and no retry after a possibly committed write.
- `branch_placement_guard.go` reads shared worktree metadata, including missing
  physical worktrees and rebase/bisect/update-refs reservations. It does not
  depend on UI worktree models or parse path-oriented porcelain output. Review
  Git operation-metadata formats on upgrades; malformed active state fails
  closed. A fresh occupancy check is not a cross-process checkout/worktree lock.
- Exact Git ref queries support packed refs without loose-file assumptions.
  Symbolic aliases (including broken aliases) are rejected, not dereferenced.
  No-op placement writes nothing. Independent branch reflogs support manual
  recovery; the upstream HEAD-based Undo action does not cover this operation.
- Backend real-Git tests passed uncached on Linux/Git 2.51.1, including dirty
  index/files, tracking metadata, occupied/unborn/detached states, CAS races,
  same-OID aliases, hooks, lock cleanup and process failures. The prepared
  protocol predates the minimum supported Git version; no minimum-version or
  cross-platform runtime matrix has been performed.
- `BasicCommitsController` adds one binding hook; the old creation/checkout
  action keeps its handler/identity but has an explicit checkout label.
  `branch_placement_actions.go` owns captured source/repository identity, worker
  preparation/application, move warnings and renewed confirmation. Refresh
  branches/commits/subcommits/reflog by identity, not position or checkout.
  `selected_commit.go` shares full-OID guards with historical comparisons;
  the latter's behavior is unchanged. The config field is initially unbound.
- Development docs/schema/cheatsheets and the static inventory must be
  regenerated after the binding/config/English additions. Controller tests
  verify captured selection, cancellation, changed/deleted refs, no-op refresh,
  repository changes and unchanged dirty files.
- The 127-scenario branch/commit/reflog headless selection passed in 107.551s,
  including all three new placement scenarios (local history with dirty files,
  stale/deleted destinations and independent branch history/reflog sources).
  The first broad attempt was invalidated by adding a source scenario while
  its generated list was frozen; regeneration and a stable-source rerun fixed
  that harness coordination issue. This is not a new full-suite run.
- Debug build, generation, formatting, inventory/checker tests and targeted
  backend/controller race tests passed. Real SHA-256 creation/movement passed
  separately. Final stable-source full short tests, formatting and lint passed
  (zero issues). Earlier checks caught and corrected the missing terminal-safe
  goroutine wrapper and a test allocation lint warning; a concurrent generated-
  list change also required restarting lint against the stable source tree.

### Named checkout (item 5 complete)

- `oscommands/lock_retries.go` adds an opt-in, command-local
  `DisableLockRetries` flag, copied by `CmdObj.Clone`. Two early hooks in
  `gitCmdObjRunner` honor it for combined/separate captured output; `Run` uses
  the existing combined-output path. Default retries and credential handling
  are unchanged. Preserve this distinction: a hook may emit lock-error text
  after a mutating operation, so retrying based on text alone is unsafe.
- Tests verify default retry behavior, copied/isolated flags, single execution
  and unchanged stdout/stderr/error identity with the opt-out. Both command
  packages' short tests passed. The named-checkout backend explicitly opts out;
  existing upstream callers retain their retry behavior.
- `git_commands/object_id.go` now owns the shared full-OID syntax check formerly
  private to historical diffs. The commit-controller helper delegates to it;
  the forthcoming shared checkout-menu helper can use the same cheap check
  without importing controllers. This refactor changes neither existence/type
  verification nor diff/placement behavior. Both affected packages' short tests
  passed, including a dedicated SHA-1/SHA-256/revision/synthetic-input matrix.
- [named-checkout-design.md](named-checkout-design.md) records the paused design
  boundary; [named-checkout-probes.md](named-checkout-probes.md) and its standalone
  build-ignored probe reproduce native Git behavior. In particular, plain OID
  branch-name shadowing, late direct/symbolic ref changes and post-transition
  failures must not be overlooked when implementing or merging named checkout.
  The user chose native Git concurrency: recheck immediately before launch,
  and coordinate other ref-mutating tools while checkout runs. Do not add hook
  replacement or claim expected-old atomicity for named checkout. Probe writes
  stay in owned temporary repos.
- `branch_target.go` verifies exact native object identity and commit type with
  `cat-file --batch-check`, keeping stderr warnings separate. Ref-only placement
  now uses it too, replacing its type-only combined-output check. Preserve this
  seam when merging: hash-looking names of the wrong object-format length must
  not stand in for actual commit IDs; native hash shadows must not redirect or
  disable a valid operation. Dedicated real SHA-1/SHA-256 target tests passed,
  including identity/type failures, shadows and unchanged repository state.
- `named_checkout.go` owns immutable destination/HEAD snapshots and the combined
  `checkout --no-overwrite-ignore --no-guess -B name OID^{commit}` invocation.
  Keep the peel suffix: plain OIDs can be shadowed by a same-named branch.
  Actual current-context bare detection must not reject a linked worktree of a
  bare common repository. Operation markers are guarded without dirty polling.
- `branch_placement_guard.go` now shares its scanner with named checkout. The
  opt-in permits only the current worktree's HEAD reservation, never another
  worktree or an active-operation reservation. Continue scanning after that
  permitted match; otherwise a duplicated checkout can mask another worktree.
- `RefsHelper.CreateCheckoutMenu` has one additive row hook into fork-owned
  `helpers/named_checkout.go`. Existing checkout handlers remain unchanged.
  The new flow captures source/repository identity, preserves literal input,
  warns before tip moves and rechecks before execution. It never routes into
  upstream's autostash helper. Successful refresh switches panes in its batched
  `Then` callback; failure refresh shows actual state without forcing a pane.
- Apply uses input-blocking status. Reprepare a stale snapshot on that same
  worker and enqueue the fresh modal before releasing buffered input; starting
  a second read worker from the UI would replay keys before the warning exists.
  Preserve errors and refresh HEAD/branches/files/commits/subcommits/reflog/
  worktree/bisect state even after a partially completed Git failure.
  Failure refresh retains the selected commit/reflog position, unlike successful
  checkout's deliberate navigation to HEAD. An overwrite rejection must not
  lose the target the user may want to inspect or retry.
- The new English labels and menu factory require regenerated cheatsheets and
  inventory. Existing exact checkout-menu integration expectations include the
  additive row; do not remove existing detached/current-branch entries.
- The uncached backend short suite passed. Real-Git coverage includes current/
  detached/unborn checkouts, carried staged/unstaged/untracked changes, rejected
  tracked/untracked/ignored collisions, upstream retention, active markers,
  duplicate worktree reservations, a linked worktree of a bare repository,
  native OID shadows, SHA-256, stale ref/HEAD identity and hook partial failures.
  The captured checkout command explicitly verifies its no-lock-retry flag and
  exact argv. Shared command-runner tests verify that flag prevents retries;
  a dummy Git runner alone cannot prove the opt-out was integrated.
- The 170-scenario branch/commit/reflog/worktree/undo headless selection passed
  in 433.042s, including all three new named-checkout scenarios. An early run
  found rejected checkout selecting HEAD instead of retaining its source; the
  failure refresh now keeps the commit/reflog selection, with modal assertions.
  A later fixture-only error used an IsSelected matcher with a content assertion;
  the corrected selected-line assertion passed in the full affected rerun.
- Full short tests, debug build, generation, standalone inventory/checker tests,
  formatting, lint (zero issues) and targeted backend/helper race checks passed.
  Generation initially caught a scenario variable not matching its filename;
  headless checks must wait for successful generation and a stable source list.
  Additional sparse-checkout/submodule tests passed, preserving the cone and
  respecting explicit recursion while rejecting dirty nested-file overwrites.
  Their direct-Git control checks optional parent-index preservation against
  native behavior without comparing unrelated repositories' commit IDs. The
  final full short suite and scoped test-file lint passed (zero issues).
  These are Linux/Git 2.51.1 checks, not a new full-suite, minimum-Git or
  cross-platform/ref-storage matrix.

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
