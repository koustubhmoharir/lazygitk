# Upstream merge guide

Maintain this guide with every change to a fork integration point, replacement
codepath, or merge requirement. Link to [Roadmap.md](Roadmap.md) for behavior and
[Tracker.md](Tracker.md) for the current item's remaining work. Keep this guide
about the code that exists, not speculative implementation tasks.

Sync routing acceptance checkpoint: default focused push/pull hooks are now
integrated, with push limited to one branch and separately chosen wider updates.
The full short suite, generation, debug build, formatting, scoped lint and
inventory checks pass. The complete `sync`, `tag` and `demo` terminal group
passes after updating legacy implicit-flow fixtures; all affected push demos
include explicit reviews. The occupied-update fixture must make the target
fast-forwardable to test occupancy rather than stopping earlier on divergence.
This checkpoint does not complete Roadmap 7: the remaining non-sync menu audit
is recorded in [branch-menu-audit.md](branch-menu-audit.md). Preparatory/historical
checkpoints below describe their verification at the time, not remaining
requirements to restore a deferred multi-branch push.

Item 7 acceptance is complete. The full short suite passed, including backend
checks (457.330s), controllers (356.556s) and helpers (13.146s). Generation,
formatting, debug build, scoped lint and the 549-factory inventory/parser checks
pass. The full branch/sync terminal group passed with `-parallel 1` (405.161s),
retaining its existing intentional skip. The initial concurrent broad run hit
the fixed 40-second limit in `branch/delete` and `branch/delete_multiple` while
the short suite was running; both passed on focused rerun (19.208s) and in the
serialized group. No timeout or application behavior was changed to bypass the
deadline. The tracker now owns item 8; key defaults are still the pre-migration
ones. Verification is Linux/Git 2.51.1, not a minimum-Git or cross-platform matrix.

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

### Editable combine review

- `controllers/fork_combine.go` adds one keyless BasicCommits action, separate
  from upstream squash/fixup and the cherry-pick buffer. Capture selected OIDs
  and repository generation before worker inspection. Review full ordered IDs
  and subjects, current HEAD and the selected-ancestor keeper before opening
  the shared commit message editor with all original messages in its body.
- Ignored-file approval, review acceptance and mutation are explicit. A stale
  capture offers fresh review of the same selected commits or return to the
  draft. Pre-start errors, including a failed fresh capture, restore the edited
  draft on the original repository. Reopen the editor before raising its error
  popup so dismissal returns to the draft rather than replacing the error.
- Reuse the existing merge/rebase result and terminal-signing helpers. Keep
  private payload cleanup on worker, terminal failure and repository-change
  paths. A native conflict/hook failure remains an in-progress native rebase;
  do not reopen a normal commit operation or rerun the combine implicitly.
- The named terminal combine scenario covers cancel, editable full messages,
  both dirty-tree failure paths, explicit fresh-HEAD review, preserved external
  empty commits, unrelated refs and final file contents. Focused controller
  checks cover source guards, retained inputs, repeated mutation approval and
  terminal failure/cleanup. The palette command is available; `g c` awaits the
  full normal-profile activation.

### Captured combine topology and native execution

- `git_commands/captured_combine_plan.go` binds the linear topology to literal
  selected objects, native HEAD and ignored-file risks. Find the earliest
  selected ancestor using linear-count ancestry queries, then capture only its
  first-parent suffix through HEAD. Read actual object headers/messages, so a
  shallow boundary cannot masquerade as a root and dates cannot reorder picks.
- `LinearCombinePlan.SelectedCommits` returns independent display data for the
  final full-OID/subject review. Capture and getters own their slices; later
  model changes cannot mutate the operation. The clean-status query explicitly
  uses the captured worktree, independent of the process's current directory.
- `combine_command.go` revalidates capture and ignored-file approval immediately
  before building one native rebase. Explicit options disable autostash,
  autosquash, merge rewriting and updateRefs, retain empty/redundant commits and
  reschedule a failed exec for explicit user continuation. Preserve these flags
  when merging upstream rebase construction. No custom locks/CAS/retries occur.
- The caller owns cleanup of the private initial payload on every execution
  path, including terminal signing and a native pause. Only the sequence editor
  needs it; Git-owned continuation state survives cleanup. Tests use actual
  Git and the serialized daemon route, preserving large full messages,
  unrelated branch refs, harmless untracked files and native state cleanup.
- Focused capture checks cover raw message bytes, ancestry/date order,
  non-adjacent selections, root/detached history, merges, shallow parents,
  staged/unstaged changes, changed cwd, stale/foreign plans and ignored-path
  fingerprints. The GUI editor/default shortcut are integrated separately.

### Combine message persistence in native rebase

- `daemon/combine_rebase.go` introduces an appended daemon kind and one mapping
  entry in `daemon.go`. Keep existing kind numbers stable across merges.
  A private JSON file carries the logical TODO and approved full message;
  only its pathname travels in the environment.
- The sequence editor compiles the keeper's logical reword into a native pick
  followed by an exec amendment using `--cleanup=verbatim --file`. It installs
  the message next to Git's TODO. The exec line is constant, resolving the
  worktree's state path through quoted `git rev-parse --git-path`; it embeds no
  user text or repository path and works with spaces/newlines in paths.
- This replaces a native reword specifically because a pre-commit hook failure
  can cause native continue to bypass the reword editor and lose the approved
  message. The caller must use `--reschedule-failed-exec`: a failed amendment
  stops once and remains queued for the user's explicit continue. Hooks and
  signing remain native; there is no application retry or rollback.
- Git owns the message file in `rebase-merge` through success/abort. Later
  fixup/pick conflicts and process restarts need no original temporary payload
  or in-memory callback. The real-Git daemon tests cover root/nonroot,
  non-adjacent selections, large messages, hook rejection, both conflict stages,
  unselected messages and native abort cleanup. Capture/editor/executor wiring
  remains separate; this commit does not enable `g c`.

### Ref-only placement from branch lists

- `controllers/branch_placement_refs.go` adds the existing placement command to
  local/remote branch lists through one additive hook per controller. Resolve
  the captured full ref on a worker, then freeze its commit before name entry.
- The shared `branchPlacementFlow.promptName` and optional repository validator
  retain the existing create/move/no-op review and native mutation. Keep the
  ref-origin generation check through later prompts and warnings; a repository
  change must not redirect the pending operation.
- Git-backed checks cover local and remote refs, source movement before the
  read, frozen target placement, dirty-file preservation through the existing
  flow and repository changes during name entry. Normal `n b` registration
  remains part of the pending default-profile activation.

### Selected-ref checkout choices

- `helpers/selected_ref_checkout.go` captures a selected local/remote/tag ref,
  resolves its commit on a worker and presents independent local-branch,
  named-branch and detached choices. `gui/fork_selected_checkout.go` replaces
  only the default direct-ref checkout handler during semantic projection;
  explicit legacy bindings retain their original handlers. The default profile
  remains pending activation.
- Naming and branch-move approval reuse `namedCheckoutFlow`. Its optional
  repository validator retains the originating generation across the complete
  flow. Preserve that callback when merging changes to named checkout.
- `BranchCommands.CheckoutCapturedCommit` performs one detached native checkout
  with literal commit validation, no guessing, ignored-overwrite protection and
  no application lock retry. It carries harmless dirty files and preserves
  Git's overwrite rejection. The new flow never offers an implicit stash.
- Focused real-Git helper tests cover captured branch movement, independent
  remote naming choices, repository changes during name entry, dirty-file
  carry and overwrite refusal. Existing named-checkout checks remain green.

### Captured cherry-pick source and exact-list review

- `controllers/fork_cherry_pick.go` adds a global keyless selected/marked
  action and replaces `LocalCommitsController.paste`/`canPaste`. Freeze both
  source choices before the chooser; never combine marked and selected commits
  implicitly. The native plan supplies full ordered OIDs, subjects, destination,
  merge-parent choices and ignored-path approval.
- Repository pointer/generation and a one-use approval retain origin. Changed
  HEAD or ignored risks require a fresh review. Execution delegates conflict,
  empty-commit and paused-rebase handling to the existing
  `MergeAndRebase.CheckMergeOrRebaseWithRefreshOptions`; do not replace it with
  a normal commit operation or forbid otherwise valid native operation states.
- Successful marked paste retains the upstream buffer and sets `DidPaste` to
  hide it, only if it is still the captured buffer. Selected paste does not
  hide unrelated marks. Preserve the old reset/refresh/continuation semantics.
- Integration fixtures explicitly choose marked source and assert the actual
  full destination and ordered commits. Paused-rebase review uses detached
  HEAD, and raw commit subjects must not include rendered author prefixes.
  Checks cover all nine cherry-pick scenarios plus reflog and demo selection.
  Read the post-pick destination only after its refreshed commit list is visible.

### Captured native cherry-pick backend

- `git_commands/captured_cherry_pick.go` captures literal commit objects, current
  HEAD identity and affected ignored-file fingerprints. Git supplies ancestry
  order; canonicalized input ordering makes unrelated tips deterministic across
  preparation and validation. Getters return independent plan data.
- Merges require an explicit common mainline parent, validated for every merge.
  Ignored-path risks require approval and fresh fingerprints. The command is
  one native sequencer invocation with exact reviewed OIDs, explicit cwd and
  disabled application lock retries. Git preserves harmless dirty files or
  rejects overlaps; no stash, custom lock, rollback or CAS is introduced.
- `rebase_ignored_files.go` extracts `ignoredRisksForPaths` so existing rebase
  policy and cherry-pick share collision/fingerprint handling. Preserve the
  caller-specific candidate path sets when merging: cherry-pick uses changed
  paths, including every merge parent for conservative risk review.
- Focused real-Git checks cover misleading dates and unrelated tips, SHA-256,
  literal/hex-ref identity, detached/unborn states, paused rebases/bare repos, merge
  parents, ignored changes, native dirty-file protection and conflict sequence
  preservation. The controller review and shortcut migration integrate this
  backend separately.
- Preserve native/upstream operation-state behavior: a paused interactive rebase
  can accept a cherry-pick. Do not add blanket marker rejection to this path.
  Git enforces valid operation state. Retain upstream version-dependent empty
  handling and the existing post-action continuation/conflict helper; the new
  feature changes selection/review and removes implicit stashing, not those
  unrelated behaviors.


### Idempotent cherry-pick marking

- `modes/cherrypicking/fork_marks.go` keeps mark/unmark separate from upstream's
  copy-range toggle. Mark unions by object ID without removing existing marks;
  unmark removes only the selected objects. Stored commits own their hash,
  parent and tag data instead of pointing into mutable UI models.
- `controllers/fork_cherry_marks.go` adds keyless actions through the existing
  BasicCommits fork hook and repaints local/reflog/branch histories. The normal
  catalogue assigns `c y` / `c x`; the old toggle has no default path, while
  explicitly configured legacy toggle bindings remain supported.
- Mode and controller checks cover repeated marking, cross-selection union,
  selected-only removal, invalid/synthetic selections, copied ownership and
  repainting. Normal shortcut activation and Escape terminal checks remain in
  the current tracker item.


### Semantic mode routes and pane target identities

- `Binding.CommandActive` distinguishes an inactive shared-key route from a
  disabled operation. The projection always keeps per-mode semantic routes,
  including explicit legacy keys. A specific command override wins over an
  explicit shared-route override, then legacy field provenance, then defaults.
  Dispatch skips inactive routes before checking operation errors; aliases of
  the same action coalesce only after explicit-override priority is resolved.
- `gui/fork_command_labels.go` names each mode's actual verb, including inactive
  palette entries. Do not reuse a shared upstream binding's dynamic label for
  every split action: it describes only the currently active mode.
- `JumpToSideWindowController` assigns stable target IDs such as
  `navigation.focus-panel.files`, and labels them for discovery. Positional
  `jumpToBlock` keys retain their configured targets; a stable-ID override can
  now address one pane independently. Preserve the target suffix through
  palette deduplication and future usage tracking. The source identity test
  covers all default platforms/locales and cleared key configuration.
- These refine the pending dispatcher migration. The normal registration hook
  is still unchanged until the accepted operation flows and matrix checks pass.


### Diff discard approval and staged unstage exception

- `WorkingTreeDiffActions.DiscardSelection` delegates confirmation to
  `controllers/fork_diff_discard.go`. The staged diff retains upstream's
  immediate unstage behavior, explicitly requested by the user. Destructive
  unstaged discards always confirm, regardless of the upstream skip-warning
  setting, with the repository instance/generation checked before application.
- `fork_command_projection.go` exposes the staged discard handler as an `x`
  alias of the same unstage operation while retaining the normal unstage path.
  Keep that live mode guard separate from true worktree discard and historical
  commit rewrite. This is a documented user-approved key meaning exception.
- Preserve the fork prompt: upstream's old message suggests disabling the
  optional warning, which does not apply to this path. Focused tests cover both
  warning settings, cancellation, real errors and repository changes.


### Directional tree navigation

- `filetree/directional_navigation.go` handles visible trees independently of
  pane switching: Left collapses or selects a visible parent, Right expands or
  selects the first visible child. Respect flat mode, compressed/filtered paths,
  hidden roots, and the visible root's no-parent boundary. Selection indices
  returned by the helper must be applied before the controller refresh.
- The two additive keyless action hooks in Files and CommitFiles use
  `controllers/tree_navigation.go`. The fork projection will assign arrows
  together with the final navigation profile; upstream bindings remain active
  until that migration. Keep the historical diff-action hook separate.
- Focused tree checks exercise real working-tree and historical-tree models,
  compressed/filtered paths, root expansion/collapse, leaves and flat mode.


### Pasted-hash historical comparisons

- `git_commands/historical_commit_comparison.go` resolves only unambiguous
  hexadecimal commit IDs and builds literal-pathspec difftool commands with
  fixed left/right OIDs. Separate stdout from Git diagnostics, including full
  hex-named ref warnings; never recover object identity from mixed output.
- The additive keyless `diff.commit.selected-commit` and
  `diff.history-file.selected-commit` actions in `historical_diff_actions.go`
  capture the selected OID/path and repository generation before prompting.
  The selected revision is left, pasted revision right. Files/directories and
  deleted paths use native two-tree comparison; explicit rename/arbitrary
  worktree-file selection remains the separate existing flow.
- `controllers/historical_commit_comparison.go` refuses repo switches and
  preserves the captured source even if models change. The normal `d c` route
  is in the catalogue, awaiting dispatcher activation; the action is already
  discoverable through the palette. Focused Git-backed checks cover partial,
  full, ambiguous, non-commit and SHA-256 IDs, ref shadows, exact argv/cwd,
  cancellation, copied paths and repository changes. Existing binding-metadata
  checks also include the pasted-hash action, which is keyless before the
  normal `d c` profile is activated.


### Captured command menus and disabled defaults

- `gui/fork_prefix_menu.go` owns visible prefix frames with one-use acceptance
  tokens. `fork_command_origin.go` pins the repository instance/generation,
  static and side contexts, selected IDs/ranges, focused diff content/mode,
  and conflict file/content/marker size/hunk selection. A pending prefix must
  not resolve a newly selected conflict side or follow a refreshed file.
  Freeze dynamic labels before opening the popup; run guards again after Pop.
- `CreateMenuOptions.BeforeAccept` runs before native Pop. Its matching model
  callback is reset on every `createMenu`; `BlockUnmatchedKeys` consumes invalid
  printable continuations before inherited/global actions. Ordinary menus keep
  their previous behavior. The small `createMenu` and `ContextMgr.Activate`
  hooks invalidate old tokens when menus/focus are replaced. Preserve this
  pre-pop/post-pop order when merging upstream menu or focus changes.
- Explicit single-key overrides take precedence over default prefix descendants.
  If both the leaf and descendants are explicitly configured, present the
  visible choice; do not silently discard either user-configured action.
- `registerForkCommands` merges global paths into each normal pane and reserves
  other core root keys as no-ops. It retains exact custom/editor/mouse routes;
  global custom routes precede compiled defaults. Registration is implemented
  but not yet enabled at `resetKeybindings`; activate it with projection and the
  remaining accepted action flows, rather than partially switching defaults.
- `Binding.DefaultCommand` opts into command matching and silent inapplicability.
  List guards distinguish a missing target from genuine operation preconditions;
  explicit bindings retain upstream error/fallback handling. Keep genuine Git
  failures visible. Prefix menu accelerators use the opt-in command matcher.
- Palette alias copies, menu labels/filtering and cheatsheets retain complete
  paths through `Binding.KeyLabels`, without binding continuation letters alone.
  Palette filtering skips other panes before calling dynamic descriptions.
  Source key lists and path slices remain independent of display-only copies.


### Direct reset-mode actions

- `helpers/captured_reset.go` adds `ResetCurrentToCapturedSourceMode` beside
  the existing mode menu. It shares source/HEAD/repository capture and native
  execution, but always confirms the requested hard/mixed/soft mode with its
  effects and captured endpoints. Never route these direct actions through the
  warning-skipping menu callbacks. Replayed approvals remain single-use.
- `controllers/selected_reset_bindings.go` adds separate keyless mode actions to
  branch and basic commit controllers; the fork catalogue supplies `s h`, `s m`
  and `s s` when the dispatcher is enabled. Preserve the two additive controller
  hooks and existing reset-menu behavior for explicit legacy overrides.
- Real-Git focused helper checks cover all three effects, cancellation, changed
  HEAD/source ref/repository, invalid mode and replayed approval. Normal-pane
  shortcut activation and terminal checks remain in the current tracker item.


### Command identity and path projection

- The terminal `interactive_rebase/combine_selected` scenario resets the entire
  loaded configuration to real defaults and invokes uppercase `G C`. Its graph
  assertions include original history retained by an unrelated branch; separate
  native log assertions verify only the rewritten branch's topology/messages.
  Do not mistake the full-graph rows for extra commits on the rewritten branch.

- The integration-only `AppConfig.SaveGlobalUserConfig` delegates keybinding
  serialization to `fork_test_config_save.go`. Preserve explicit legacy leaves
  and programmatic changes, but omit unchanged implicit defaults; otherwise
  saving the harness configuration silently tests legacy overrides instead of
  fork defaults. Loader provenance is nonnil even with no explicit leaves;
  constructor replacements remain nil. Narrowed primary bindings neutralize
  unchanged implicit legacy aliases. Round-trip tests cover all platforms,
  disabled bindings, explicit aliases and constructor resets with overrides.

- `gui/types/command_paths.go` keeps direct keys separate from sequential menu
  paths and provides full labels. `gui/keymap/command_tree.go` compiles paths
  without evaluating handlers or metadata: local routes shadow globals only at
  the same complete path, not an entire shared prefix. Keep mode candidates
  together for live applicability checks.
- Normal-pane leaves that return `gocui.ErrKeybindingNotHandled` pass the key
  to the node's pruned lower-priority owners (`CommandTree.Fallbacks`) in
  priority order, so upstream's Escape decline reaches global `app.cancel`.
  Fallbacks are never menu choices or hints. If every owner declines, the key is
  consumed; inapplicable/disabled leaves still consume without fallback. Global
  single keys are also registered natively (`setNativeGlobalKeys`) for non-fork
  views such as popups, with a guard that leaves fork views to their compiled
  roots.
- `gui/keymap/resolve_paths.go` resolves stable-ID overrides before explicitly
  configured legacy fields, then the fork catalogue. Provenance uses the loader's
  field-presence metadata, never equality to an old default. `legacy_fields.go`
  carries view-specific exceptions; its generator scans upstream constructors
  and legacy alias folding. Regenerate `legacy_fields_generated.go` after
  changing source bindings or aliases and review exceptions during merges.
- `gui/fork_command_projection.go` owns the normal-pane migration. It
  splits shared upstream stage/patch/conflict keys into semantic actions with
  live guards, preserves mouse routes, restricts Enter to Files, and uses a
  read-only Files inspector. Explicit legacy mode overrides retain their
  configured behavior. Numbered pane jumps retain their individual targets.
  `GetInitialKeybindings` projects core routes; `resetKeybindings` compiles them
  through `registerForkCommands`. Custom, modal, editor and mouse routes retain
  native registration. Focus restoration rebuilds bindings after `HandleFocus`.
  Offline cheatsheet generation sets a canonical Files context before projection.
- Complete-path ownership is explicit local, explicit global, default local,
  then default global. Prefix descendants remain available unless an explicit
  leaf displaces only default descendants. Never infer explicit configuration
  from equality to a default. `commit.search` aliases `search.open` after path
  resolution, preserving the first history-loading handler instead of presenting
  duplicate search choices. Keep explicit historical ID overrides resolvable.
- `fork_effective_command_paths.go` supplies private display copies for the
  palette and hints after current-pane ownership, explicit leaves and exact
  custom-root claims. Foreign-view customs must not clear current paths. Footer
  alias deduplication is local-first; mode hints use the effective command path
  or the palette path. `fork_panel_labels.go` uses owned routes for border jump
  labels, hiding normal shortcuts while a popup/editor is focused.
- Prefix menus reserve physical Up/Down, Enter and Escape even when ordinary
  modal navigation uses letters. Validation rejects these unmodified keys after
  a command path's root, and rejects mouse paths. Ordinary dialogs retain their
  configured controls. Keep prefix continuation accelerators ahead of the
  unmatched-key blockers, and both ahead of inherited navigation.
- `fork_full_refresh.go` replaces default global and Files refresh with one
  `app.refresh` identity and handler. Explicit `file.refresh` overrides retain
  the native limited operation. F5 clears the Git config cache, reloads every
  refresh scope (including selected-ref history), renders contexts and redraws
  the terminal without fetch. Layout-only windows have no context: skip them.
  Preserve repository-generation guards on completion.


### Command path configuration foundation

- `UserConfig.CommandBindings` and its validation hook introduce paths keyed by
  stable command ID. Each outer list contains alternate paths; each inner list
  is a sequence of one to three keys. Empty/null values remove keyboard paths;
  actions remain discoverable in the palette.
  Keep this distinct from legacy `Keybinding`, whose list means alternate keys.
- `config/fork_command_bindings.go` supplies JSON decoding/schema; YAML uses the
  existing Keybinding decoder. Loader coverage checks layered maps, explicit
  disabling, and coexistence with legacy bindings and custom commands.
- This is configuration support for the pending dispatcher integration. Parsing
  alone does not activate the new defaults. Regenerate `docs-master/Config.md`
  and `schema-master/config.json` when these definitions change.


### Modal editor word movement

- `config/fork_editor_defaults.go` adds Ctrl+Left/Right before user config is
  loaded, retaining macOS Alt+Left/Right and avoiding duplicate defaults on
  Linux/Windows. The single integration call in `fork_defaults.go` extends the
  upstream editor defaults; it does not register normal-pane commands. Explicit
  editor key replacements and empty-list disabling remain authoritative.
- `gocui/editor_word_motion_test.go` exercises the real editor with ASCII,
  Unicode, multiline content and boundary movement. Preserve literal input and
  native editor dispatch when merging command-key registration changes.


### Full-graph history default

- `config/fork_defaults.go` now sets `Git.Log.ShowWholeGraph = true` before
  loading user/repository config. Keep the upstream constructor unchanged and
  retain explicit false overrides. `showGraph` remains the separate rendering
  setting; this change uses the existing all-ref query (`GetCommitsOptions.All`,
  `git log --all`) without changing pagination, sorting, live toggle persistence
  or fetching refs. The user has since authorized the final keybinding and combine implementation.
- `test/default_test_config/config.yml` pins `git.log.showWholeGraph: false` for
  existing terminal scenarios whose row/range assumptions require branch-only
  history. `commit/full_graph_default` restores the real default constructor
  value and checks local-side, remote-only and current-branch commits together,
  without changing HEAD. Keep that targeted default-policy coverage when
  updating the common harness. Config tests cover all platform defaults,
  explicit user opt-out, repository precedence and restoration after reload.
- Generation updates the config guide/schema default and registers the terminal
  scenario. The only application-policy change is the fork overlay assignment;
  no command key or graph-loader codepath is replaced by this commit.
  Config/context/helper short tests passed (2.110s / 0.050s / 47.427s), as did
  generation, debug build, formatting and scoped lint. The verbose terminal run
  explicitly executed `commit/full_graph_default`, `commit/highlight` and
  `commit/do_not_show_branch_marker_for_head_commit` (4.899s, all passed). The
  initial incorrect filter ran no scenarios and is not counted as coverage.

### Command-key foundation (item 8 in progress)

- `config/fork_keybinding_overrides.go` records explicitly loaded keybinding
  fields. Its only upstream seams are private metadata on `UserConfig` and a
  loader call after successful migration/unmarshal/validation, before legacy
  alias folding. `KeybindingIsExplicit` uses YAML tags (`universal.pushFiles`,
  for example), not Go field names or value equality. Deprecated aliases keep
  their separate provenance. Only recognized schema group/leaf fields count;
  unknown accepted fields cannot mask a neighboring configured binding.
  Mapping-to-Node decoding retains accepted inline YAML merges. Existing alias
  rejection and null-to-disabled migration remain the loader's own behavior.
  Metadata does not serialize or set new key values. A fresh load starts from
  fresh defaults; removed repository/user overrides do not retain stale presence.
  Independent config short tests passed (0.269s), including nested inline
  merges, legacy alias separation, explicit disable/omission and unknown keys.
  Full generation and debug build passed without generated-file changes;
  formatting and combined config/gocui scoped lint passed (zero issues).
- Override tracking is a foundation, not yet the command projection policy.
  Do not treat one command ID as one configuration field: worktree switching,
  modal confirmation, tag creation and other factories use multiple legacy
  fields or different fields in different views. Preserve those distinctions
  when integrating the catalog. The allocation draft records proposed paths;
  no application defaults have been replaced yet.
- `pkg/gocui/command_key.go` owns ASCII-letter command normalization. It folds
  only a single ASCII letter with no modifiers or Shift alone. `Key.Equals`,
  text, Unicode, punctuation and Ctrl/Alt/Meta combinations remain exact.
- `SetCommandKeybinding` in `command_binding.go` is an opt-in alternative to
  `SetKeybinding`. A small `keybinding.matchKeypress` seam normalizes candidates
  only for those registrations. `execKeybindings` excludes printable command
  registrations from local, native-parent and global dispatch while editing,
  even if an editor declines a shifted-letter event. Editors receive the
  original event. Preserve this check before remembering global candidates.
  Normal handled/fallthrough ordering and paste guards remain native.
- Normal panes now use this API through compiled command routes. Do not globally
  normalize key equality or incoming events as a substitute. Native text input
  and custom commands remain exact. Cross-mode and operation regression checks
  remain in the current tracker item until acceptance is complete.
- `new-features/keymap/generate.go` instantiates the exported cheatsheet path
  for three platforms after legacy alias folding. Its default/cleared-key
  snapshots use the command normalizer for collision candidates, preserve raw
  key tuples and declaration order, and read actual native parent pointers.
  Rows record complete sequential command paths; collisions compare whole
  paths, so a shared prefix root is not a collision. The cleared snapshot adds
  an explicit empty `commandBindings` override per projected command ID so
  catalogue defaults cannot reappear once legacy fields are cleared.
  Logical parent contexts are separate metadata. The tool never calls action,
  availability or dynamic metadata callbacks; the source-reviewed mode matrix
  supplements its deliberately non-exhaustive runtime snapshot. These are
  standalone explicit-file tools/tests, excluded from application generation.
  Independent verification: full gocui short tests passed (3.795s), standalone
  audit tests passed (0.023s), two generated snapshots matched byte-for-byte,
  and scoped gocui lint reported zero issues. This foundation does not activate
  the new defaults, so no new terminal interaction flow is claimed as tested.

### Selected-branch operations (item 7 complete)

- `helpers/captured_branch_delete*.go` now replaces the local/remote branch
  controllers' deletion paths. Copy full source refs and named remotes before
  inspection; keep native targets/configuration/proofs and repository generation
  across menus, manual destination prompts, approval, workers and refresh-Then.
  Ambiguous destinations are Cancel-first; compound approval includes unmerged
  history after excluding caches scheduled for deletion. Consume execution once.
  Use responsive waiting workers: blocking keyboard events deadlocks credential
  entry. Collapse only the original unchanged selection before refresh; retain
  local range selection for remote-only deletion. Refresh actual partial state
  on errors, and do not revalidate refs intentionally deleted by the operation.
  Old private controller deletion wrappers are removed. Legacy BranchesHelper
  worktree helpers still serve explicit Worktrees-pane removal; preserve those
  distinct operations when merging upstream. The shared headless deletion
  helper checks full refs/OIDs and reservation warnings. Captured flow tests
  passed (39.887s); all six branch deletion terminal scenarios passed across an
  initial four-case success and two-case rerun (25.990s). Three occupied-worktree
  refusal cases passed (14.745s), with the filter and two explicit worktree
  removal baselines passing in the preceding run. Native deletion checks,
  generation, debug build and inventory checks also passed; scoped backend,
  controller and i18n lint reports zero issues. This completes the
  deletion subtask; creation audit and final item 7 acceptance remain pending.
- `RefsHelper.NewBranch` and `TagsHelper.OpenCreateTagPrompt` are thin hooks
  into fork-owned `captured_branch_create.go` and `captured_tag_create.go`.
  They replace upstream name sanitization/live callbacks and creation autostash
  fallback while retaining native creation, tracking, tag description/signing
  and overwrite choices. Branch policy rows retain worker-prepared native plans;
  do not recapture changed config only after a row is chosen. Keep default row
  ordering based on the suggested/entered name, and explicit no-setup wording.
  Confirm/cancel/duplicate callbacks retain repository generation and consumed
  execution state. Tag signing revalidates again before terminal launch; its
  guarded refresh-Then invokes the success callback at most once. Preserve
  annotation versus signing as distinct captured facts. Refresh actual errors;
  failed checkout keeps selection instead of selecting a presumed new HEAD.
  Remote branch creation supplies a full ref; stash creation supplies its full
  immutable commit, never a mutable stash index. `checkoutByName` captures its
  original selected source and backend/generation before name entry; its
  missing-ref creation uses `NewBranchWithNameFromRef` without another name
  prompt. Generic existing checkout behavior outside creation remains separate.
  Focused captured branch tests passed (29.781s), tag tests passed independently
  (16.824s), all nine affected branch/commit/stash terminal cases passed
  (32.157s), and the full tag terminal group passed (13.937s). Generation,
  formatting, debug build, scoped lint and the 549-factory inventory/parser
  checks pass.
  This completes the creation audit; final item 7 acceptance is recorded above.
- The branch-creation terminal fixtures now review source/HEAD and select
  tracking policy explicitly through `Common.ReviewBranchCreateMenu`. Check
  policy rows independently because their default order depends on name entry;
  wrapped source lines must not require the OID twice on one line. Use full refs
  for stash/branch shadowing and the actual selected commit, including one-row
  history lists. The former autostash fixture now verifies native overwrite
  refusal, preserved dirty bytes/HEAD and no branch or stash creation. Missing-ref
  assertions must propagate query errors rather than treating any Git error as
  absence. All nine affected branch/commit/stash scenarios passed (32.157s)
  against the captured creation integration being finalized below.
- `branch_action_create.go`, `branch_action_tag_create.go` and
  `branch_action_head_target.go` add native creation plans for the remaining
  creation audit. Bind backend/source/current HEAD/configuration and destination
  state before command construction. Normal branch creation refuses existing
  names; it does not replace the separate reviewed placement/checkout actions.
  No-track creation uses the captured commit OID. Configured/direct tracking
  passes the original full start ref so Git owns `autoSetupMerge`, inheritance
  and `autoSetupRebase`; the final exact-tip/config check is an observation,
  followed by Git's normal source-ref race, not expected-old atomicity. Do not
  substitute a partial application implementation of native tracking policy.
  `--no-track` suppresses automatic setup but can preserve orphan metadata;
  use accurate UI wording. Native tracking can reject non-branch namespaces.
  Checkout protects ignored files and preserves harmless dirt, without stash,
  retries, custom locks or rollback. Pointer-only creation refuses reservations,
  including the unborn current branch; a bare common HEAD alone is permitted.
  Tag plans capture native signing, description and the exact old tag object;
  overwrite approval is required even when the old tag peels to the same commit.
  Commands target the captured commit and retain native errors. The HEAD source
  helper supports named/detached HEAD and rejects an unborn source. Focused
  native creation checks passed (26.521s), including raw dirty-state retention,
  stale/foreign plans, tracking policies and real checkout obstructions.
  [Design](branch-create-design.md) and [native probes](branch-create-probes.md)
  record these constraints. Live creation uses the captured GUI hooks; final acceptance is recorded above.
- `git_literal_config.go` extends the command builder in a fork-owned file for
  literal subsection names containing `=`. Git's `-c key=value` splits at the
  first equals sign; use native `--config-env=key=VARIABLE` for those names and
  supply its fixed per-command environment value. Normal keys retain existing
  argv. Preserve inherited `GIT_CONFIG_COUNT` settings and never edit config on
  disk to emulate the override. Captured merge, current-Pull integration and the
  older internal pull primitive use this for branch merge options and remote
  tag-pruning policy; branch-only remote deletion uses it for mirror policy.
  Callers must keep their AddEnvVars values paired with the builder arguments.
  Native regression checks cover duplicate branch messages, explicit merge
  overriding configured squash, inherited config and a remote named `peer=one`.
  The helper requires Git's `--config-env` support (2.31+, already below the
  merge-preview requirement). Focused merge/current-Pull/helper checks and
  backend lint passed; see [Git's option documentation](https://git-scm.com/docs/git).
- `branch_action_delete_{local,remote,choices,context}.go` adds backend-owned
  deletion reviews. Bind source refs, current HEAD, effective configuration,
  reservations and retained-history proofs to the original backend. Each source
  uses its own upstream plus current/main proofs; selected refs and tracking
  caches scheduled for deletion cannot prove retained history. Refuse occupied
  local branches, including paused-operation reservations; bare common HEAD
  alone is not a checkout. Local execution is one native `branch -D --` after
  applicable unmerged-history approval. Remote inspection retains every merge
  entry and inverse-fetch candidate; the GUI must resolve ambiguity explicitly.
  Remote execution uses exact full server branch refs and configured named
  remotes, suppresses mirroring, followed tags and recursive submodule pushes,
  and rejects protected/symbolic cache writes. Keep native multiple-push-URL
  partial success, one push per remote, and no retries or rollback. Between
  remote groups validate HEAD/configuration/cache safety without requiring a
  cached source intentionally deleted by an earlier push to reappear. Compound
  callers must retain local branches on any remote error. Getter results are
  copied; config digests stay private. Live deletion now uses these APIs through the captured GUI helpers. Native tests cover literal names, dirty/index byte preservation,
  reservations, stale/foreign plans, cache mappings and actual partial failures.
- `branch_sync_reverse_refspec.go` adds preparatory inverse fetch mapping for
  remote-pane deletion. Return every deduplicated full source in configuration
  order; multiple candidates require explicit choice. Retain positive mapping
  knowledge when a negative exclusion exists: this is not proof of server state
  or what was last fetched. Reject relevant abbreviated sources instead of DWIM
  guessing, validate all specs and preserve native query errors. Malformed
  configuration errors must not echo credential-bearing spec values. Focused
  real-Git/parser checks and backend lint passed. This does not route deletion.
  [Deletion design](branch-delete-design.md) and
  [the reproducible native evidence](branch-delete-probes.md) record reservations,
  literal names, remapped storage and partial multi-URL deletion. The standalone
  probe mutates only fresh owned temp repos, uses read-only observations and
  compares real index/HEAD/worktree bytes; its full log stays in that temp dir.
- `helpers/captured_merge*.go` now routes local/remote branch merge entry points
  through captured reviews. `MergeAndRebaseHelper.MergeRefIntoCheckedOutBranch`
  is a thin replacement hook; its old public merge methods remain dormant for
  upstream structure and must not acquire new live callers during merges.
  The obsolete private FF-preference helper was removed. Branch controllers
  supply full refs, not display names. Five fixed modal rows retain configured,
  allow-FF, no-FF and both squash modes. Allow-FF must remain available for
  divergent histories; it is not FF-only. Keep source/current commit identities
  and configured options visible, with captured message/sign-off/hook settings.
  UI navigation performs repository checks; native review checks stay on worker
  or terminal-launch boundaries. Ignored/staged loss reviews are Cancel-first,
  and selecting a path cannot approve it. Squash commit has its own queued
  validation; native conflicts/errors cannot fall through into commit.
  Native operation identity guards queued abort; squash conflicts do not offer
  normal merge abort. Guard result inspection, refresh and refresh-Then against
  repository generation/backend replacement, including an actual second repo.
  Shared status/repository-change text is now action-neutral for reset/rebase/
  merge and sync. Regenerate inventory after factory/line changes. The shared
  `Common.ReviewBranchMerge` headless helper checks roles and mode; menu headers
  contain review text and rows have shortcut prefixes/ellipsis, so avoid exact
  five-line assertions and label-only anchored selectors. All packages' short
  tests passed in separate backend/remaining-package runs, including a final
  controller rerun (264.050s). Three branch merge scenarios (22.075s), the UI
  mode-suggestion scenario (53.169s), generation, build, inventory checks,
  formatting and scoped lint passed. This completes the merge subtask, not
  Roadmap 7's remaining deletion and creation audit.
- `branch_action_merge_plan.go` prepares immutable-source merge commands from
  those reviews. Clear implicit `branch.<current>.mergeOptions` and supply its
  captured arguments exactly once before app arguments and the selected mode.
  Disable autostash and command-runner retries. Preserve native `--ff` semantics
  (divergence permits a merge), FF-only rejection, no-FF/squash incompatibilities
  and unborn-HEAD restrictions. Reservation checks depend on actual tip movement;
  the selected source remains read-only even when occupied elsewhere.
  Ignored-file approval binds paths, bytes and modes. Cache ort predictions;
  final validation must not rerun configured drivers. Preserve other strategies
  with an explicitly explained conservative review of all ignored files.
  `branch_action_merge_commit.go` separately captures the post-squash index,
  message and commit options. No-op/unchanged results never authorize an
  unrelated-only commit; conflicts and stale HEAD/config/index stop phase two.
  Keep terminal signing/editor/hook options and actual native failure state.
  Normal merge abort checks original HEAD, MERGE_HEAD and ORIG_HEAD, not a later
  source-ref lookup. Native plan/commit/abort tests and backend lint passed;
  normal concurrency remains Git's, without custom locks, retries or rollback.
- `branch_action_merge_review.go` adds backend-owned, read-only merge reviews
  binding source/current HEAD, effective configuration, semantic staged entries
  and staged path names. Native last-value branch merge options precede captured
  application options. Keep the configuration digest private; it can cover
  credentials. Native ancestry exit 1 means false, while query errors propagate.
  Detached/bare, active-operation and unmerged-index checks remain explicit;
  unborn HEAD is retained for native fast-forward support. Reviews neither run
  drivers nor authorize mutation. `branch_action_merge_policy.go` folds only
  normalized option records, respecting Git's sticky message presence,
  no-op `--no-strategy`, ordered FF flags and squash/explicit-commit conflicts.
  Real-Git review/policy checks and backend lint passed. Live GUI routing is
  recorded separately when integrated.
- `branch_action_merge_preview.go` is preparatory native merge inspection, not
  live routing. It runs ort's `merge-tree` with private temporary object storage
  and C-quoted original/inherited alternates. Preserve complete NUL-protocol
  validation: configured driver stdout must not masquerade as a result tree.
  Exit 1 supplies valid conflict checkout paths, including directory-renames
  absent from either input tree. Source/current-HEAD snapshots are checked on
  both sides; generated objects are removed on all paths. Invoke only during
  explicit merge review because Git may run configured low-level drivers.
  `branch_action_merge_arguments.go` tokenizes configured options without shell
  expansion and validates exact supported options; positional sources and
  abort/continue dispatch are forbidden. Its normalized records distinguish
  option values from flags. Neither helper authorizes mutation or completes
  the merge audit. See [the current design](branch-merge-design.md) and
  [native evidence](branch-merge-probes.md). Focused real-Git/parser tests,
  formatting and backend lint passed; the existing merge menu is unchanged.
- Audit and captured-source contracts are in
  [selected-branch-operations.md](selected-branch-operations.md); native
  routing/scope evidence is in
  [selected-branch-sync-probes.md](selected-branch-sync-probes.md). The probes
  use owned disposable repositories, not configured application remotes.
- `git_commands/branch_sync_info.go` adds read-only native inspection using
  the existing literal-name, exact-ref, HEAD and worktree-reservation readers.
  Preserve their error handling and repository-pinned builder when merging
  upstream changes. Parse NUL-framed fields from raw stdout, with record
  terminators separate: remote/config metadata can contain newlines. Preserve
  the exact tracking namespace; do not derive it from a remote name.
- `git_commands/branch_action_target.go` prepares a read-only source/current-HEAD
  snapshot for remaining selected-branch actions. Accept only full direct refs
  or full commit OIDs; preserve native query errors and distinguish a missing
  or moved ref from an inspection failure. The opaque snapshot is bound to the
  original backend instance, including when another instance opens the same
  filesystem repository. Check both HEAD's ref identity and OID; detached and
  unborn HEAD remain representable because reset can operate there. This guard
  does not authorize a mutation or replace operation-specific occupancy/risk
  checks, and it must not blanket-reject a merge that reset may explicitly abort.
  Callers still bind their repository generation, commands and refreshes, use
  the captured source OID, and preserve Git's normal concurrency after launch.
- `git_commands/branch_action_rebase.go` adds prepared, backend-owned branch
  rebase plans. Reuse the selected-commit clean-tree, operation, named-HEAD and
  reservation checks, then retain the exact source/current HEAD and optional
  marked base. Ignored-file approval covers path/content/mode fingerprints.
  `rebase_ignored_files.go` now shares a scanner whose destination tree and
  replay range may differ: marked `--onto` rebases must scan base..HEAD, not
  only target..HEAD. Preserve ordinary selected-commit behavior through its
  existing wrapper. `PrepareInteractiveRebaseCommandOpts.disableUpdateRefs`
  enables `rebase.updateRefs=false` only for this captured path; retain the
  existing todo daemon and optional initial-break instruction. Commands use
  immutable commit OIDs, captured working directory and no lock-error retries.
  `CheckBranchActionRebaseInProgress` checks merge-backend head-name/orig-head/
  onto metadata for queued conflict-abort actions after HEAD intentionally moves;
  it must not use the pre-launch HEAD check or accept another repository's plan.
  New native and existing selected-commit/start-policy tests passed (23.906s),
  including a real replaced conflicting rebase and native index.lock failure.
  Backend lint passed with zero issues. GUI integration is described below.
- `MergeAndRebaseHelper.RebaseOntoRef` now routes to the additive
  `helpers/captured_rebase.go` flow; local and remote branch controllers pass
  full literal refs, and the remote binding names the current destination too.
  Preserve ordinary, interactive-with-initial-break, marked-base and closest-base
  choices. Capture marked-base mode, exact source/current HEAD and base ref on
  entry; never read a later selection or replace the reviewed range silently.
  The shared `captured_action_repository.go` guard was extracted from the fork's
  reset flow; retain its backend identity and UI repository generation checks.
  `captured_rebase_execution.go` revalidates native plans on the worker and again
  immediately before terminal launch, which retains signing interaction. A
  consumed approval cannot run again. Guard native result inspection, operation
  flags, refresh and its completion callback against repository replacement.
  Clear only the original marked base once Git starts; preserve a newer mark.
  Conflict choices retain the reviewed repository; before abort, verify the
  original in-progress rebase rather than a later operation's detached HEAD.
  Paused signing/hook/empty-replay errors stay errors, with actionable context;
  do not reintroduce the upstream helper's automatic empty-commit skip. Explicit
  continue/skip/abort remain available through Rebase options. The integration
  component's `ReviewBranchRebase` records the additional review step in the
  existing branch/interactive/amend/demo scenarios; it must not replace their
  native post-state assertions.
  Validation: combined reset/upstream/rebase controller checks passed (64.679s),
  including real signing failure and replaced-operation rejection. Corrected
  pointer-replacement tests use a second repository and passed separately
  (7.695s). The full short suite passed, including native commands (145.127s)
  and controllers (126.736s). Seventeen relevant terminal scenarios passed in
  the broader rebase run; its old empty-replay auto-skip expectation was updated
  to explicit user skip and passed separately (7.617s). Generation, inventory,
  formatting, build and scoped lint passed after removing a constant test-helper
  parameter flagged by lint.
- `helpers/captured_reset.go` replaces the shared reset menu through
  `RefsHelper.CreateGitResetMenu`; the local-branch controller calls the same
  flow directly. Preserve this routing for commit, tag, remote-branch and
  Files/upstream entry points. The flow captures repository generation, source
  ref/commit and current HEAD, then rechecks them before launching one native
  soft/mixed/hard reset. All modes affect current HEAD, never a selected
  off-branch ref. Hard reset always requires an explicit data-loss warning,
  including clean tracked state with obstructing untracked/ignored paths.
  Preserve native errors and partial effects; do not retry or repair failures.
  Annotated/nested tag sources peel the immutable object read from the exact
  tag ref. Files' `@{upstream}` entry captures the current branch name before
  inspection and resolves its native tracking ref, including custom namespaces
  and local `.` upstreams. Reset may abort this worktree's own merge; the
  `branchOtherWorktreeOccupancy` wrapper shares the existing metadata reader
  while excluding only this worktree's reservations. A tip-changing reset must
  still refuse reservations in other worktrees; a same-tip reset may repair
  the current index/worktree without moving that shared branch.
  Validation: native source/reset and controller reset tests passed, including
  custom/local upstreams, stale callbacks, raw dirty bytes, native index locks,
  detached/unborn HEAD and active merges. All seven branch/tag/reflog reset
  terminal scenarios passed (68.997s); generation, inventory checks, formatting,
  build and scoped lint passed. The full short suite subsequently passed,
  including native commands (347.452s) and controllers (190.559s).
- `controllers/branch_sync_source.go` defines context-to-source resolution,
  including stored local-branch history and rejection of non-local history,
  range selections and detached pseudo-branches. Capture the name, not a model
  pointer. Other panes must not inherit an old local-branch selection.
- Additive GUI modules `branch_sync_bindings.go`, `branch_sync_target.go` and
  `branch_sync_push_flow.go` implement focused-source resolution, captured
  repository/generation guards and one-branch push review. The default
  `SyncController.GetKeybindings` route now uses the focused local branch (or
  explicitly labelled current branch outside branch-bearing contexts), with
  context-specific command IDs/descriptions and disabled reasons. Preserve the
  guard and `OpensMenu` metadata at this integration point. Wider-stack push is
  deferred; do not restore its push-menu offer. The old internal current-only
  helpers remain for upstream merge isolation, not as the default binding path.
  Preserve synchronous repository-scoped busy flags, worker cleanup/refresh,
  callback repository checks, immutable approved plans and Cancel-first lease,
  multi-URL and upstream-refusal menus. A tracking-disabled fallback must
  reprepare and review the captured source/options, never execute immediately
  or turn a force-with-lease request into an ordinary/unleased push. Explicit
  lease prompts retain their captured options/OID across later callbacks.
  This preparation adds English fallback strings; default shortcut policy is
  unchanged. Regenerate the command inventory/key docs with final binding
  integration, not as evidence that an unused menu is already accessible.
- `branch_sync_update_flow.go` is the off-branch update dialog, reached through
  the captured default Pull route. The selected Fast-forward action also
  uses the captured route described below. Fetch and ref-only
  advancement have separate reviews. Local `.` upstreams read exact local
  branch refs without contacting a transport. Keep custom storage explicit,
  preserve one-off options through stale-source review, and never use a
  changed selection or a newly checked-out target as permission to integrate.
  Divergence and occupied worktrees do not fall through to reset, checkout or
  worktree creation. English fallback strings belong with this helper, not
  with a later default-binding change. Source/options are copied before worker
  dispatch; normalized equivalent branch input preserves tracking suggestions.
  Stale-source review preserves explicit one-off options rather than mixing
  them with newly loaded upstream defaults. URL inputs are refused without
  echoing them, and local-source menus say review rather than fetch.
  Independent sync-controller tests passed (55.406s), covering divergent and
  occupied linked-worktree refusal, cache-only fetch effects, cancellation,
  generation/pointer changes and literal input capture. Scoped lint is clean.
  This helper alone does not complete the selected-branch slice.
- `branch_sync_stack_push_flow.go` composes one-branch dialogs into an explicit
  wider push. This flow is deferred and must not be reachable from ordinary
  push: the user limited push to the selected branch. Keep its isolated
  preparation/tests separate from acceptance of the shipped push route; do not
  restore its offer during upstream merges. Pull/update and explicit
  multi-selection Fast-forward are separate, unchanged scopes.
  The push discovery entry point and the one-branch menu's optional stack
  callback were removed; dormant batch primitives have no ordinary push entry.
  Selected/current menu-scope tests and four focused terminal push scenarios
  pass, including an unpublished related lower branch, tracking preservation
  and stale review. The debug build and scoped lint pass. Default binding
  integration and legacy multi-branch-push fixture migration remain separate.
  `branch_sync_push_flow.go` collects already validated opaque
  plans through `acceptPlan` instead of running push during member review.
  Collection is read-only and holds no one-branch push flag across callbacks;
  the final Cancel-first batch menu claims every source flag synchronously.
  Preserve captured member tips, full-OID destination/lease review, per-member
  upstream consent and immutable multiple-URL approval. Refuse overlapping
  configured destinations, including exact URL overlaps between remote aliases,
  without echoing credential-bearing URLs. Inspection rows never authorize writes.
  Preflight all plans before the first push, then revalidate each at launch.
  Stop on any native failure; report completed branches and possible partial
  success of the failing push. Never replay or roll back a consumed approval.
  This additive flow does not itself commit the default sync binding hooks.
- `branch_sync_stack_target.go` shares native candidate routing between wider
  push and update dialogs, retaining the captured repository/name and configured
  local-main exclusions. A stale-source refresh must pass its newly inspected
  snapshot into a subsequently requested stack review, not retain the original
  obsolete OID or read another pane's selection.
- `branch_sync_integration_result.go` handles terminal merge/rebase results
  for both one-branch pull and the current phase of a wider update. Terminal
  output is not captured in the exit error: do not search `exit status 1` for
  conflict diagnostics. Inspect native merge/rebase markers and unmerged index
  entries using the captured backend, retain repository-generation checks, and
  record that the operation was started here. Only actual unmerged entries
  open conflict handling; paused hook/signing failures preserve their errors.
  Never automatically skip or continue. The earlier text-only callback missed
  real conflicts, which the terminal acceptance scenarios exposed.
  Independent integration/current-pull/stack-update unit tests and scoped lint
  pass. The focused terminal run passed the merge/rebase conflicts, configured
  interactive todo/drop behavior, all five force-lease scenarios and dirty
  current/stack integration. Its three fixture failures (menu-row matching and
  the repository-fetch forwarding route) were corrected and passed on a
  separate three-scenario rerun; the whole group was not repeated afterward.
  Interactive fixtures explicitly accept the initial native todo list with
  `sequence.editor=true`, rather than leaving a headless editor waiting.
- `branch_sync_update_collection.go` adds read-only plan collection to the
  existing off-branch and current-pull dialogs. An approved fetch may update its
  reviewed tracking ref; collecting a local-tip/integration approval must not
  apply it. No-op plans require an explicit Continue during wider review.
  Keep the captured mode and source; changed collection inputs fail rather than
  silently reinspecting and fetching a replacement source.
- `branch_sync_stack_update_flow.go` reviews every member before a final
  Cancel-first approval. Preflight every plan before the first local tip moves.
  Execute guarded off-branch fast-forwards first, then the exact reviewed current
  integration through the terminal executor. Refuse divergence and occupied
  tip changes; there is no rewritten-upstream reset or worktree creation.
  Targets remain independently pinned even when a local `.` upstream is another
  selected member: advancing that member does not implicitly change an already
  approved integration target. The final prompt states this; using a different
  target requires another explicit review. Label current integration as an
  integration, not a promised resulting tip (merge/rebase may create new OIDs),
  and label ref-only no-ops as unchanged rather than suggesting a rewind.
  The current phase releases its initial busy flag before queuing the terminal
  handoff, so deferred ref-phase cleanup cannot clear the executor's new flag.
  Preserve all launch-time validation and report completed ref advances on
  current-phase errors, including native/query/repository errors crossing async
  boundaries. The idempotent progress error preserves native causes through
  `Unwrap`; `runReviewedIntegrationReporting` adds context without changing
  ordinary one-branch execution when its reporter is nil. Never automatically
  repeat a consumed batch or retry/roll back a native partial result.
- `BranchesController.fastForward` now delegates copied literal selection names
  to `branch_sync_fast_forward_flow.go`, replacing cached-status gates and the
  upstream helper's rewritten-history/occupied-worktree updates. Inspect fresh
  native inputs, use a simple one-branch flow or explicit multi-selection batch,
  and set one `fastForwardOnly` policy flag. A selected current branch uses
  explicit native fast-forward even when pull is configured to rebase. It must
  not inherit autostash or offer other integration methods. Fork translation
  fields replace the old tooltip that promised divergent resets; shortcut
  policy and the existing command ID remain unchanged.
- Verification for the reviewed stack-update/Fast-forward route: all 15 existing
  affected terminal scenarios pass, including rewritten history, dirty files,
  disabled reflogs, multi-selection and occupied/rebasing linked worktrees.
  The full controller/translation and native Git-command short suites pass;
  additional execution tests cover late stale/risk/config failures, native hook
  failure and completion-dispatch failure after an earlier ref advance, plus
  configured rebase with unrelated-ref and harmless-untracked-file preservation.
  Generation, debug build, inventory tests, formatting and scoped lint pass.
  This is not yet acceptance of the remaining default push/pull binding hooks.
- These modules are preparatory and not wired into push/pull yet. Native
  `PushRef` inspection can return only one of several configured destinations;
  it is a suggestion, not unique resolution or permission to execute. Future
  execution plans must revalidate source, occupancy and remote configuration.
- `branch_sync_remote.go` adds uncached remote inspection, including multiple
  mappings/URLs and included configuration. Its signature frames raw selected
  settings, valueless versus empty keys, URL-rewrite rules and native mirror
  normalization. Dotted remote names must not contaminate a shorter remote's
  profile. The config is not stored in app state or logged as query output;
  future modals must redact credentials when displaying URLs.
  Empty URL/push-URL values clear earlier values, matching Git's native
  remote reader; a later nonempty URL remains valid. Keep config parsing
  shared with execution-plan validation, without logging credential values.
  This API accepts configured remote names, not literal URLs. Reject URL-like
  input without echoing it: native push-remote metadata can contain credentials.
- `branch_sync_stack.go` discovers stack candidates from native direct local
  heads and one topological commit walk, not whichever history pane is loaded.
  It pins the selected OID, excludes commits reachable from exact supplied main
  refs, skips symbolic head aliases and orders equal-tip names consistently.
  Results are inspection only: review and revalidate every chosen source before
  writes. Callers must also retain configured local-main-name exclusions when
  the corresponding main ref is a remote branch. Discovery never checks out or
  modifies an occupied worktree. Preserve raw stdout framing and native query
  errors, rather than treating failed inspection as an empty/safe stack.
  Independent focused stack-inspection tests passed (9.173s), including a real
  split DAG, stable shared-tip order, root tips, symbolic heads, raw dirty-index
  preservation and linked-worktree occupancy. Native query-failure injection
  is not covered by this checkpoint; failed commands still return their errors.
- `branch_fast_forward.go` is the ref-only forward/update primitive. Preserve
  ancestry-error distinction, source-OID revalidation, ancestor-target no-ops,
  worktree/reservation checks and single delegation to guarded placement.
  Preserve uncertain commit errors before changed-state classification. The
  scoped ambiguity-warning override is for validated literal OID ancestry
  only, not a global suppression of Git diagnostics.
  Plans are bound to the original backend instance, not just equivalent
  repository data; a repository replacement must not reuse an old plan.
  `CheckBranchFastForward` exposes the same read-only owner/source/occupancy/
  plan validation for multi-branch preflight. `ApplyBranchFastForward` still
  repeats it and delegates a real update once to guarded placement. Passing
  preflight is not a reservation or permission to omit launch-time checks.
  Independent fast-forward tests passed (42.395s); read-only checks compare
  refs, raw index/HEAD/config/reflogs and worktree bytes, and preserve occupied
  no-ops, changed-state refusals and backend ownership. Backend lint is clean.
- `branch_sync_refspec.go` resolves native push's tracking lookup through
  configured fetch refspecs, rather than inventing
  `refs/remotes/<remote>/<branch>`. Do not reuse it as fetch's DWIM source
  resolver: a native probe distinguishes the two for abbreviated sources.
  Keep ordered canonical preference separate from the set of positive storage
  destinations to inspect for side effects. An explicit empty destination or
  matching exclusion suppresses automatic lease selection; do not fetch to
  replace missing lease evidence. This is conservative lease policy, not a
  reimplementation of every version's negative-fetch behavior. Native ref
  validation, wildcard captures and exact source matching need regression tests.
- `branch_sync_push.go` adds a backend-bound branch-only push plan. Preserve
  exact source/destination refspecs, explicit recursion/upstream choices,
  no-follow-tags, no automatic upstream setup, mirror refusal and disabled
  lock retries. Multi-URL approval belongs to one immutable reviewed profile;
  revalidation must invalidate it on transport/config changes. URLs stay in
  memory and require redaction before UI display. Native partial success is
  not grounds to retry the whole command.
- Explicit expected-OID leases use locally reviewed refs, never a preparatory
  fetch or a fallback to plain force. Keep the selected branch's upstream
  snapshot (including every merge entry) separate from its push destination.
  Native push can opportunistically update local tracking refs: validate all
  possible storage mappings, refusing branch/tag/special namespaces and
  symbolic aliases before any network operation. Recheck force restrictions
  when constructing the command, not only when opening a menu.
- `branch_rename.go` adds literal, backend-bound metadata rename plans. Recheck
  the exact source OID and destination before native `branch --move`, with no
  force, checkout, retry or rollback. Refuse direct or symbolic destinations
  (including dangling aliases) and existing destination branch configuration;
  subsection names are case-sensitive. Same-name plans are zero-write no-ops.
  Native rename may update symbolic branch names in linked worktrees without
  changing their commit, index or files. Refuse active-operation reservations.
  Preserve native partial failures: a config lock can leave the ref renamed
  even though config migration failed. `BranchesController.rename` now replaces
  the upstream sanitizing callback with `branch_rename_flow.go`: copy the name
  and repository generation, inspect before prompting, confirm the literal
  old/new names and full source OID, and guard worker refresh/reselection.
  Keep native upstream warnings even when the tracking cache is gone. Refresh
  after native failures without retry/rollback or successful-rename reselection.
- `branch_sync_metadata.go` adds captured upstream set/unset plans using exact
  direct tracking refs, including custom fetch namespaces and local branches.
  Native `branch --set-upstream-to` resolves remote mapping; the selected tip,
  tracking tip, effective/primary-local upstream entries and remote config are
  revalidated before launch. Metadata changes do not move a checked-out tip or
  write its index/files. The set/unset entries in
  `BranchesController.viewUpstreamOptions` now use `branch_upstream_flow.go`,
  capturing repository/name when the parent menu opens. Full tracking refs
  replace the old assumed standard namespace. Confirm the old tracking values
  and exact new target before writing; redact URL credentials in prior remote
  metadata. Each worker/callback and refresh retains the captured repository.
  `branch_upstream_menu.go` now replaces the parent menu's live selection and
  reconstructed upstream namespace. Inspect native metadata on a worker;
  construct a new branch model without copying its atomic fields, and retain
  the original context/repository generation in every callback. Divergence
  uses the native full tracking ref, including custom namespaces and local `.`
  upstreams. Missing refs disable dependent rows; other resolution errors remain
  visible while set/unset stay available for repair. Base-query errors propagate.
  Before opening reset/rebase submenus, freshly check the selected tip, HEAD
  identity/tip, exact upstream mapping and target commit. Reset uses the shared
  captured flow. Rebase receives a full ref; preserve the full-ref self-rebase
  check in the captured rebase mode menu. That helper's nested execution
  now uses the captured rebase flow described above; retain the nested execution
  checks as well as the parent-menu checks when merging upstream changes.
  Independent reset/upstream controller checks passed (33.501s). All nine
  selected-upstream terminal scenarios passed (39.263s), including custom/local
  divergence with dirty files, standard/base divergence, tracking edits and
  current-destination reset/rebase. Generation, inventory checks, build and
  controller/helper/i18n lint passed.
  Independent metadata/rename/push tests passed (74.433s), including raw
  dirty-state preservation, linked worktrees, custom namespaces, inherited
  configuration refusals and native rename partial failure. Backend lint
  passed with zero issues after correcting two test-only unused assignments.
- Upstream setup is independent of the explicit push destination and defaults
  off. `branch_sync_metadata.go` shares the guard with `branch_sync_push.go`,
  refusing multiple, empty or valueless upstream
  entries and settings inherited from outside the primary local config before
  allowing `--set-upstream`. Compare fresh effective entries with an uncached
  `config --local --no-includes` query; effective values alone cannot establish
  that native setup can replace them safely. Keep this check at preparation
  and command construction. Do not repair configuration or silently remove
  the upstream-setting choice: the UI must offer Cancel or a new review with
  tracking changes disabled. Preserve `push.autoSetupRemote=false` on the
  fallback invocation, even if the user's normal Git configuration enables it.
  Explicit setup (both push and metadata-only) also sets
  `branch.autoSetupRebase=never` for that invocation: native Git otherwise
  overwrites an existing branch.rebase value when automatic setup is enabled.
  Preserve existing preferences rather than changing reconciliation policy
  alongside a tracking-only edit. Unset already leaves that preference intact.
  Native probes confirmed this on Git 2.51.1. Regression tests cover absent,
  false, true and merges preferences under global/repository automatic setup;
  independent focused cases passed (5.646s) and backend lint reported zero
  issues. The expanded upstream/push suite also passed (44.236s).
  Malformed metadata that Git itself cannot inspect retains the native error;
  never infer an ambiguity sentinel by matching stderr text. Native tests cover
  primary duplicates and included, global, worktree and command-environment
  entries. The push backend tests passed independently (38.656s), followed by
  the expanded config-origin cases (6.458s); scoped lint reported zero issues.
- `branch_sync_fetch.go` composes an explicit fetch into a reviewed local
  storage ref with the guarded fast-forward primitive. Preserve empty refmap,
  no tags/prune/prune-tags/submodule recursion/FETCH_HEAD writes and no retry.
  Only call `PrepareFetchedBranchUpdate` after a successful command. Missing,
  divergent or rewritten upstreams must not fall through to reset/integration.
  A fetch may inspect an occupied branch; advancing its ref still obeys the
  existing worktree reservation guard. Current-branch pull is a separate path.
  `PrepareCurrentBranchFetch` now exposes the same bounded fetch for an exact
  native HEAD branch, without requiring a clean tree merely to download refs.
  Plans retain their original current/off-branch mode during revalidation;
  switching HEAD must not silently promote or demote the operation.
  `FetchedBranchTip` reads a direct, full commit OID from the reviewed storage
  only after successful fetch. It observes local state under Git's normal
  concurrency model, not an atomic remote/ref transaction. Review that OID
  before integration. Current fetch plans cannot use ref-only advancement.
  Independent current/off-branch fetch tests passed (14.685s), including a
  genuinely newer server tip with raw HEAD/index and dirty files preserved.
- `branch_sync_tracking.go` now shares local-storage validation across branch
  network builders. Exact special namespaces and their children are both
  forbidden. Preserve native ref-format checks and broken-alias detection;
  generic diagnostics must not describe a fetch refusal as a push failure.
- `branch_sync_current_pull.go` retains native pull rather than emulating its
  merge/rebase/ff-only precedence with separate commands. Keep the shared
  clean-rebase refusal before network activity, actual HEAD validation,
  no-autostash/no-retry policy, and branch-only `rebase.updateRefs=false` scope.
  Fingerprint native pull/rebase/merge settings, including the exact
  case-sensitive branch subsection's rebase and merge options. Do not swallow
  query/cleanliness errors as generic stale-plan errors. `pull` rejects
  `--no-prune-tags`; preserve invocation-only overrides for both global and
  selected-remote tag pruning instead. No persistent config rewrite is needed.
  This is the earlier one-step primitive, not the subsequently approved Pull
  UI: do not wire it into the new fetch-then-review action. That action must
  integrate the reviewed incoming OID without another network pull.
- `branch_sync_local.go` resolves exact direct local upstream refs without a
  fetch for off-branch advancement. Keep its source checks and guarded
  fast-forward application; approval captures a commit, not a live ref that
  may silently retarget later. For a current-branch `.` pull, verify transport
  resolution with the read-only `ls-remote --get-url`, fingerprint local source
  state and rewrite settings, and refuse redirected/shadowed local transports.
  Only the exact remote named `.` is a shadow; unrelated dotted names must not
  be rejected. Diagnostics must not reveal a rewritten credential-bearing URL.
- `CurrentBranchPullPlan.WithFastForwardOnly` adds only the stricter native
  ff-only option. Revalidation retains that reviewed restriction without
  broadening a plan's source, destination, repository or ref-update scope.
- `branch_sync_pull_review.go` is the replacement backend for fetch-first
  current-branch Pull. Capture `CurrentPullReview` before fetching: the old
  explicit tracking ref/reflog supplies a fork point for rewritten upstreams.
  Integration plans capture a full incoming OID and replay boundary, use native
  merge/rebase rather than another network pull, and remain backend-bound.
  Source HEAD/tip, reservations and native policy are checked again at launch.
  Preserve native branch/pull rebase precedence, pull.ff precedence, merge.ff
  fallback and refusal of unconfigured divergent reconciliation. Explicit
  choices override defaults; never silently choose merge when no policy exists.
  Default custom branch.mergeOptions remain native and must be displayed using
  `MergeOptions`/`ConfiguredMergeFF`; explicit merge choices clear that bundle
  only for the invocation. `UsesConfiguredRebaseOptions` signals that the
  default retains configured topology/autosquash behavior. The UI must label
  that as configured rebase, not promise explicitly linear history.
  All choices prohibit autostash, recursive submodule updates and automatic
  rebase updates of other branch refs. Optional index/stat-cache writes are
  disabled during cleanliness review. Explicit linear rebase suppresses
  autosquash and merge preservation; interactive rebase needs a terminal.
  Reuse the conservative ignored-path/content fingerprint scan for merge,
  fast-forward and rebase. Approval is immutable and invalidated by changed
  risk contents; actual merge no-ops do not request irrelevant approval.
  FF-only divergence is refused before presenting overwrite-risk approval.
  Local current upstreams read exact local branch refs directly, so a shadowed
  or URL-rewritten remote named `.` cannot redirect this action.
- `branch_sync_current_pull_flow.go` composes the additive current-branch
  dialog with the existing update-options flow. Capture review/fork-point
  context before fetch, inspect the fetched storage tip once, then review a
  method and exact incoming OID separately. Keep current/off-branch mode fixed
  for the flow; native validation must reject later HEAD changes. Stale-source
  or policy changes return to inspection without silently fetching again.
  Typed method-eligibility errors disable Default while retaining explicit
  alternatives; query/config errors remain errors. Configured topology and
  merge option bundles must not be labeled as explicitly linear/default-free.
  Ignored-file approval is Cancel-first, content-bound and not granted by
  selecting an individual path row. Reprepare from the immutable plan's
  `Review()` context, not mutable dialog state.
- `branch_sync_integration_exec.go` launches reviewed integration through the
  terminal lifecycle, including interactive rebase and signing. Preserve its
  synchronous scoped busy flag across worker/UI/terminal/completion handoffs,
  repository-generation guards, and revalidation immediately before terminal
  execution (a queued UI callback is not launch approval). Completion records
  merge/rebase state and checks conflicts; do not substitute the older helper
  that can auto-skip commits after an error. Never retry a native partial
  result or refresh a replacement repository. Default sync binding/stack
  integration remains pending; these modules alone are not feature acceptance.

Current-pull dialog checkpoint (2026-10-22): independent sync-controller tests
passed (108.023s), followed by current-pull/rename tests (43.252s) and a targeted
fetch-to-choice stale-policy regression (2.220s). The seven new/updated focused
push/pull headless scenarios passed (44.461s) against experimental binding hooks;
those hooks remain separate work until explicit stack choices are preserved.
Controller/i18n lint passed with zero issues. A first headless run found test
driver misuse (new popup driver instances lost required title checks), corrected
in the fixtures; use anchored path-segment expressions when selecting tests so
`push` does not also select legacy `force_push` scenarios.

Metadata-dialog checkpoint (2026-10-22): independent rename/current-pull tests
passed (43.252s), upstream-dialog tests passed (8.735s), and nine focused
rename/upstream/view/reset/rebase headless scenarios passed (9.219s). Controller/
i18n lint reported zero issues. The rename fixture now expects literal input
and final approval rather than silently replacing spaces with dashes. Native
failure tests retain the error and refresh observed state without retry or
rollback. These checks do not finish the remaining upstream-menu/delete audit.

Stack-push checkpoint (2026-10-23): independent focused controller tests passed
(19.413s), covering edited destinations, retained leases and stale-lease refusal,
explicit multi-URL consent and changed-profile refusal, pointer/generation guards,
Cancel-first inspection, raw dirty-state preservation, busy cleanup and native
three-branch partial failure without attempting the third branch. Eight focused
push/pull headless scenarios passed (101.769s), including the actual selected-stack
menu and no remote writes before final approval, against experimental hooks.
Default routing and stack-update integration still require their own acceptance.

Reviewed integration backend checkpoint (2026-10-22): independent focused
tests passed (74.603s) and the full backend short package passed (208.045s).
Native tests cover merge policy and custom option bundles, actual divergent
rebases, dropped-upstream-commit exclusion, captured local targets, config/source
changes, dirty-tree rules and approved/stale ignored-file risks. Debug build
also passed. Scoped backend/controller/i18n lint reported zero issues; a final
ignored-risk case rerun passed (20.371s) after a test-only exhaustive-switch
correction. This is backend verification, not default Pull UI acceptance.

Additive push-modal checkpoint (2026-10-22): independent focused controller
tests passed (18.609s), including normal single-upstream replacement,
ambiguity introduced after review, Cancel/no-tracking fallback, retained
explicit force leases and stale repository callbacks. Formatting and scoped
backend/controller/i18n/sync-fixture lint passed with zero issues. A working-tree
binding experiment passed five serial push TUI scenarios (10.626s), including
exact config preservation; that hook and its scenarios remain uncommitted
pending wider integration. These are not item 7 acceptance or a full sync-suite
run, and no new default routing is claimed by the additive-modal commit.
The staged source was also exported and checked without the experimental
binding: controller tests passed (33.208s) and scoped lint reported zero issues.
That isolation initially found an unused binding entry point; it remains with
the uncommitted integration hook rather than introducing a lint suppression.

Read-only checkpoint (2026-10-22): main-agent native probe rerun passed,
including selected != current with staged/unstaged/untracked preservation,
explicit stale-lease rejection, multiple configured push destinations and
suppressed extra fetch mappings/pruning/tags/`FETCH_HEAD` writes. Independently
rerun source tests passed (backend 6.648s, controller 0.030s), including an
unborn checkout with another valid selected branch. Formatting and scoped lint
passed with zero issues. No executable push/pull routing has changed yet;
these checks are not end-to-end acceptance for item 7.

Ref-only/remote checkpoint (2026-10-22): independent focused tests passed
(4.752s), followed by the full backend short package (35.469s). Formatting and
scoped lint passed with zero issues. Coverage includes staged/index/HEAD bytes,
unstaged and untracked content, same-tree rewritten divergence, occupied and
newly reserved worktrees, stale sources, no rewind, literal OID shadow refs,
uncertain commits, included configuration, multiple mappings/URLs and native
boolean normalization. The additional standalone probe passed its mirror
assertions: both configured mirror plus an explicit refspec and an attempted
`--no-mirror` override fail on Git 2.51.1. No executable sync UI is wired yet.
Follow-up focused tests passed (2.273s), including native URL-list reset
comparison and rejection of fast-forward plans on a replacement backend.
Independent push/refspec tests passed (20.659s), including real pushes to two
approved bare repositories, invalidated approvals, stale server/local leases,
upstream snapshot changes, raw HEAD/index preservation and unrelated refs.
The subsequent refspec/fetch-focused run passed (7.863s), including a native
abbreviated-source probe with an actual changed remote tip. No sync UI routing
has changed in these backend commits.
The independent backend short package passed (136.522s); the subsequent focused
fetch/tracking/current-pull run passed (46.223s), including real advancement,
an existing FETCH_HEAD sentinel, valid and dangling aliases, and native pull
precedence. Scoped backend lint passed with zero issues after removing an
unused fixture return. Earlier verification attempts encountered test files
still being written; those compile errors are not successful verification.
Local-upstream follow-up: focused backend tests passed (69.428s), the added
unrelated-dotted-remote regression passed (4.168s), and actual local-source
rebase replay passed (2.630s). The backend short package passed (145.401s) and
backend lint passed with zero issues after the declaration-only cleanup.

### Selected-commit rebase and clean-start policy (item 6)

- `git_commands/rebase_start_policy.go` owns the shared fresh tracked/index
  clean-state query and commit-first error. Untracked files, including those
  inside submodules, remain allowed by explicit user choice. The query overrides
  submodule ignore configuration so tracked child changes are not hidden.
- `RebaseCommands.PrepareInteractiveRebaseCommand` now returns `(command,
  error)` and checks before constructing an executable command. Preserve this
  error contract for editor subprocesses too. Its private prepare/run wrapper
  replaces direct chained execution; `SquashAllAboveFixupCommits` is a separate
  start builder with the same check. Both explicitly use `--no-autostash` and
  disable whole-command lock retries. Existing topology/update-ref behavior of
  these upstream history-editing actions is otherwise retained.
- `PatchCommands.MovePatchIntoIndex` no longer takes a stash argument or
  pushes/pops changes. `CustomPatchOptionsMenuAction` replaces its stash modal
  with the normal worker/backend check. Keep clean patch extraction working;
  dirty extraction must leave changes/history alone and explain commit-first.
- `SyncCommands.Pull` consults the additive `pull_rebase_policy.go` before
  fetching. Fresh native config reads resolve branch-specific rebase settings
  before `pull.rebase`, including bool normalization and explicit worktree
  context. Only configured rebase pulls get the clean check, no-autostash flag
  and no-retry policy; merge pulls retain their existing behavior. Do not pin
  `--rebase=<mode>` on the command: that changes Git's `pull.ff=only` precedence.
  Recognize legacy `preserve`/`p` modes accepted by the minimum Git 2.32;
  do not translate them into a modern topology choice. Native Git still rejects
  modes removed from the installed version. See its
  [2.32 parser](https://github.com/git/git/blob/v2.32.0/rebase.c).
- Do not move start checks into generic rebase continuation, skip, abort,
  edit-todo, ordinary commits or HEAD amendments. Conflict resolution is
  intentionally allowed to stage changes. Arbitrary custom commands and
  third-party `git flow` executables are not intercepted by these builders.
- `git_commands/rebase_commit_preflight.go` complements the late clean-start
  check for `RebaseCommands.AmendTo`: compare the worktree with the index
  before creating its fixup commit. Staged changes are allowed because that
  commit consumes them; other tracked edits must be rejected before mutation.
  Do not replace this with the ordinary clean-index check or remove the late
  check, which still detects changes made by hooks. HEAD-only amendments stay
  on their existing path. The user chose refusal without a fixup-only fallback.
- `controllers/fixup_placement.go` replaces post-commit stacked placement with
  a pre-commit placement decision and clean-state preflight. Preserve the two
  integration calls in `LocalCommitsController` for fixup and amend creation.
  No placement means no rebase restriction. Planned placement consumes staged
  changes only for a content-bearing commit; message-only amendments require
  both index and tracked worktree clean. Capture the owner index before the
  commit and retain the backend's post-commit check for hook-created dirt.
  Keep the existing version, operation-state, merged-target, updateRefs and
  owner-branch eligibility checks when reconciling upstream stack changes.
- `worktree_operation_guard.go` extracts named checkout's operation-marker
  guard without changing its diagnostics. The new selected-commit backend
  reuses it and existing exact OID/current-HEAD/ref-occupancy readers through
  the same pinned `GitCommon`. Its plan captures source branch/tip, target and
  explicit topology, never marked-base state. A command-scoped
  `rebase.updateRefs=false` protects other branch refs even after continuation;
  the command is not automatically retried or blindly rolled back on failure.
  `GenericMergeOrRebaseActionCmdObj` disables whole-command retries for rebase
  continue/skip/abort too: a failed hook can follow a completed replay step.
  Its merge behavior is unchanged.
- Feature design and native behavior evidence are in
  [rebase-selected-commit-design.md](rebase-selected-commit-design.md) and
  [rebase-selected-commit-probes.md](rebase-selected-commit-probes.md).
  The selected-commit flow now has an additive controller hook; ignored-file
  risks require explicit approval as described below. Item 6 remains under
  acceptance testing at that checkpoint; final acceptance is recorded below.

- `git_commands/rebase_ignored_files.go` conservatively intersects ignored paths
  with target-tree and potential replay-change paths (including merge diffs),
  with literal NUL parsing and `core.ignoreCase`. Indexed path-prefix lookups
  avoid an ignored-files × history-paths scan. Fingerprints frame each path,
  type/mode and content digest; symlinks are read, not followed. The plan copies
  exposed paths and gates command creation on an explicit approved copy.
  Fresh state invalidates approval when source, risk set or file bytes change.
  This is not an atomic file lock and cannot constrain arbitrary hook effects.
  A real-Git submodule regression verifies that native parent rebase leaves
  the child checkout and ignored child collision alone even with recursion
  configured, on the tested Git. If that native behavior changes, extend the
  safety scan/explicit choices before loosening the regression; do not silently
  accept newly unreviewed child-file overwrites.
  Path and unmerged-index queries must use raw stdout from `RunWithOutputs`,
  not the combined/sanitized `RunWithOutput`: ambiguity warnings or ANSI bytes
  in literal filenames must not corrupt the risk list or conflict detection.
- `controllers/selected_commit_rebase_actions.go` owns captured repository/OID,
  topology choice, history confirmation and risk review. Its one basic-commit
  hook covers local history, subcommits and reflog. The initially unbound
  `commits.rebaseOntoSelectedCommit` remains discoverable through `?`. Do not
  route this action through `RebaseOntoRef`, which reads/clears marked-base mode.
  Risk-menu Cancel is first; approval is separate, and path rows reopen review.
  Paths are individually quoted, scrollable and retain full tooltips; a long
  header must not conceal the list. `HideCancel` avoids a duplicate appended
  Cancel row without disabling Escape.
- The controller queues fresh review on its existing blocking worker before
  buffered keys can replay. Signing starts run through the terminal lifecycle;
  terminal return uses `helpers/rebase_subprocess_result.go` and the backend's
  read-only `HasUnmergedFiles` query to distinguish actual conflicts from
  signing/hook failures whose exit errors do not include displayed diagnostics.
  The continuation subprocess predicate also includes interactive signing;
  preserve that fork replacement when upstream changes exec/merge routing.
- New translation/config fields require development docs/schema generation and
  command-inventory regeneration. Approval does not imply changing the marked
  base, stashing, updating another branch or automatically retrying a failure.

The existing out-of-date-submodule conflict scenario now changes its submodule
after the rebase starts: this preserves its recovery/auto-staging subject while
respecting the new clean-start rule. The dirty patch-extraction scenario now
expects refusal rather than an automatic stash.

Backend/start-policy checkpoint (2026-10-22): debug build, full short tests,
generation, formatting and scoped lint (zero issues) passed. Focused backend/
named-checkout tests passed in 23.775s; the additional pull tests for actual
divergent replay and clean/dirty linked-worktree isolation passed in 6.480s.
The four headless scenarios `branch/rebase`,
`branch/rebase_conflicts_fix_build_errors_with_out_of_date_submodule`,
`main_view/move_patch_to_index` and
`main_view/move_patch_to_index_with_modified_file` passed together (11.664s).
The first headless attempt found one obsolete secondary-pane expectation in
the dirty extraction fixture; it was corrected before the four-scenario rerun.
The standalone probe also passed its explicit assertions. Broader affected
headless coverage, signing/UI acceptance and minimum-Git/cross-platform runs
have not been completed. No system dependencies were installed.

Selected-commit UI/approval checkpoint (2026-10-22): debug build, full short
tests, development generation, inventory generator tests, formatting and scoped
lint passed. Controller flow tests passed (11.411s), covering captured identity,
fresh risk review, quoted/scrollable names, cancellation, dirty rejection,
marked-base isolation and terminal signing dispatch. Backend tests cover raw
ANSI paths and ambiguity warnings as well as target graphs, SHA-256, submodule
behavior and approval invalidation by paths/type/mode/bytes. Terminal-result
tests distinguish real conflicts from other failures; signing continuation
tests verify continue/skip versus abort and explicit GPG override routing.
The two new TUI scenarios passed together (30.433s), covering other-branch
history, risk cancellation/fresh approval, tracked dirt, abort and staged
conflict continuation. Initial fixture failures were a wrong key namespace,
missing required setup callbacks and selecting file-conflict options instead
of rebase options; those were corrected before the passing run. The broader
409-scenario affected-area run completed in 375.526s: 407 Go passes and two
failures. One passing subtest was internally skipped by the harness
(`sync/push_no_follow_tags`), so 406 scenarios actually passed. Both failures
were existing find-base/amend workflows: upstream creates a fixup commit before
the clean-rebase check rejects other unstaged tracked changes. The fixup-only
fallback versus refusal was awaiting a user choice at that checkpoint.
The user subsequently chose refusal without a
fallback; automatic stacked placement now gets the same pre-mutation check.
The dedicated deleted-branch reflog scenario passed
(6.61s, invocation 20.939s). Its regenerated test list and the new early amend
guard pass formatting and scoped lint (zero issues); independent focused guard
tests passed in 2.391s; the full backend short package then passed in 63.983s.
Final acceptance reruns after that choice remain open. The early guard
preserves HEAD, index, worktree and stash on rejection,
and allows staged content plus harmless untracked files.
Actual verification is Linux/Git 2.51.1; no system dependencies were installed.

Refusal/compound-operation checkpoint (2026-10-22): the user chose refusal,
leaving all decisions about other edits to them. The updated amendment tests
check that HEAD, the staged file blob, working files and stash remain unchanged.
The preflight must inspect the wrapped native exit error, not the original
`CmdObj.ProcessState`: the application Git runner executes a clone. Controller
tests retain the real wrapped runner to protect that integration contract.
Full short tests and the final debug build passed; backend short tests took
351.999s under high host load. Generation, inventory tests and formatting pass.
An intermediate 410-scenario run built before the exit-error correction had
407 Go passes (one internal skip) and three failures: the two expected
amendment-message mismatches and a 40-second harness deadline in the longer
selected-rebase conflict scenario. The fresh serial rerun of both amendments,
clean stacked fixup placement and selected-rebase conflict recovery passed
together in 20.955s; the conflict scenario itself took 6.66s. Independently
rerun compound-operation controller tests passed in 14.079s. Final scoped lint
initially caught a test-helper naming/attribution issue; that helper was
corrected, and scoped lint then passed with zero issues. The final broad
rerun passed all 410 selected Go subtests in 547.765s: 409 scenarios executed
successfully, and `sync/push_no_follow_tags` was intentionally skipped inside
the harness. Item 6 is complete; this is affected-area coverage, not a claim
that all headless categories or the minimum-Git/cross-platform matrix ran.

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
