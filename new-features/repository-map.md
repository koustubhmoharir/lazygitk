# Repository exploration map

Read against `4e1eebe278790f257a875da6633234391a5a85b3` on 2026-10-21.
Paths are relative to the repository root. Symbols are included so the notes
remain useful after line numbers change. This is a source inspection, not a
claim that proposed features have been implemented or integration-tested.

## Defaults and background work

- `pkg/config/user_config.go`: `GitConfig` and `GetDefaultConfig`. Current
  defaults: `AutoFetch=true`, `AutoRefresh=true`,
  `AutoForwardBranches="onlyMainBranches"`, `AutoStageResolvedConflicts=true`,
  and `AutoDetectExternalChanges=true`.
- The config comment explicitly says `AutoStageResolvedConflicts` also controls
  asking to continue after conflicts are resolved. `AutoRefresh` governs files
  and submodules; `AutoDetectExternalChanges` independently watches refs.
- `pkg/gui/background.go`: `startBackgroundRoutines` gates the three polling
  routines independently. `startBackgroundFetch` immediately fetches at startup
  before its timer loop; `triggerImmediateFetch` retriggers on repo switches.
- `pkg/gui/gui.go`: also inspect `AutoFetch` checks and restart-sensitive config
  keys when testing reload/repo-switch behavior.
- `pkg/gui/controllers/helpers/branches_helper.go`: `PostFetchRefresh`,
  `AutoForwardBranches`, `PrepareFastForward`, `forwardBranches`. Manual fetch
  and background fetch share post-fetch handling. Existing forwarding supports
  worktrees, so it is not always just a write to an unoccupied ref.
- `pkg/jsonschema/generate.go` and `pkg/jsonschema/generate_config_docs.go`:
  config docs and schema generation. `docs-master/Config.md` and
  `schema-master/config.json` are generated outputs.

## Keys, dispatch, and the palette

- `pkg/config/user_config.go`: keybinding config types and platform defaults.
  Examples of overloaded keys: `r` = refresh files / rebase branch / rename
  commit or stash; `f` = fetch / fast-forward / fixup; `c` = commit / checkout
  by name / fixup message / checkout historical file. Space is select/stage/
  checkout depending on controller. `n` also participates in search and
  focused-diff navigation.
- `pkg/config/keybinding.go`, `keynames.go`, and their tests: aliases and
  validation. Audit parsed keys, not just YAML spellings.
- `pkg/gui/types/keybindings.go`: `Binding` carries keys, descriptions, handler,
  tags, and disabled reasons; it has no stable command ID yet.
- `pkg/gui/keybindings.go`: `GetInitialKeybindings` gathers controller/context
  bindings; `GetInitialKeybindingsWithCustomCommands` prepends custom commands;
  `callKeybindingHandler` handles disabled states, sometimes showing a toast or
  allowing further dispatch. This is a shared entry point for shortcut and
  palette execution, but a nil error alone does not prove an action happened.
- `pkg/gocui/gui.go`: `execKeybindings` dispatches focused-view, parent-view,
  editor, then global bindings, and also has search-match handling. A disabled
  action returning `ErrKeybindingNotHandled` can trigger fallback. Do not fix
  key consistency by remapping config alone.
- `pkg/gui/controllers/options_menu_action.go`: `OptionsMenuAction.Call` builds
  `?` from current-view/global/navigation bindings in declaration order;
  `uniqueBindings` deduplicates by translated description. The menu executes
  through `CallKeybindingHandler`. Frequency needs stable identity, not that
  deduplication key.
- `pkg/gui/menu_panel.go`, `controllers/menu_controller.go`,
  `context/menu_context.go`: sections, item key precedence, filtering, physical
  navigation keys, and selection. Sorting only the input bindings may still
  leave section-based rendering to address.
- `pkg/gui/services/custom_commands/keybinding_creator.go`: custom commands
  become bindings for one/multiple contexts or globally. Existing custom Ctrl+T
  can shadow a new builtin; migration needs to explain removing it.
- `pkg/config/app_config.go`: `AppState`, `SaveAppState`, and state loading;
  `pkg/gui/gui.go` saves state at shutdown. Existing shell-command history is
  unrelated to usage counts and should not become the identity source.

## Diffs and arbitrary selected commits

- `pkg/gui/controllers/basic_commits_controller.go`: shared commit-list
  actions, including `openDiffTool`, `newBranch`, `checkout`, and reset menu.
  `openDiffTool` already invokes a directory difftool using parent→commit,
  adjusted by sticky diff mode. It currently has no target-choice submenu.
- `pkg/commands/git_commands/diff.go`: `DiffToolCmdOptions` and
  `OpenDiffToolCmdObj` already support selected-commit→working-tree by setting
  `FromCommit` and leaving `ToCommit` empty. Uses argv, `--no-prompt`, optional
  `--dir-diff`, and a `--` path delimiter.
- `pkg/gui/controllers/diffing_menu_action.go` and
  `controllers/helpers/diff_helper.go`: existing in-app sticky comparisons;
  keep these distinct from explicit external-difftool choices.
- `pkg/gui/controllers/helpers/refs_helper.go`: `CreateCheckoutMenu` currently
  offers detached checkout or existing branches already at the selected commit.
  It does not prompt to assign an arbitrary name there. `NewBranch` checks the
  newly created branch out and offers autostash after certain errors.
- `pkg/gui/controllers/helpers/merge_and_rebase_helper.go`: `RebaseOntoRef`
  already rebases the checked-out branch onto a supplied ref through a menu;
  branches/remotes call it. The shared commit controller currently exposes
  checkout/new/reset/diff, but no equivalent “rebase current onto this commit”.
- `pkg/commands/git_commands/rebase.go`: `RebaseBranch`,
  `RebaseBranchFromBaseCommit`, `EditRebase`, and
  `PrepareInteractiveRebaseCommand`. Existing preparation includes
  `--interactive`, `--autostash`, `--rebase-merges`, and editor/daemon setup.
  Adding an entry point must account for these semantics and marked-base mode.

## Selected branch versus current branch

- `pkg/gui/controllers/branches_controller.go`: selected-branch rename/delete,
  upstream settings, fast-forward, difftool and branch creation already exist.
  `rebase` uses selection as the upstream/source, not the branch to rewrite.
  `createResetMenu` resets the current branch toward the selected ref.
  `pkg/gui/controllers/remote_branches_controller.go` has the same distinction.
- `pkg/gui/controllers/sync_controller.go`: `HandlePush` and `HandlePull` call
  `branchCheckedOut`, which resolves `Refs.GetCheckedOutRef` regardless of the
  branch selection. Push remote fallback, refspec generation, setting upstream,
  current-worktree pull, status and stacked-branch menus all need auditing;
  swapping only the initial branch pointer is insufficient.
- `pkg/commands/git_commands/branch.go`: `New` and `NewWithoutTracking` check
  out; `NewWithoutCheckout` and `CreateWithUpstream` only create refs.
  `Checkout` accepts `Force` but has no named-branch reset/checkout API.
  See the separate Git investigation for `checkout -B` and dirty-tree probes.
- `pkg/gui/controllers/helpers/branches_helper.go`: current selected-branch
  fast-forward support is a foundation, including fetch and per-branch status.
  It can update occupied worktrees; the first-version selected-ref workflow
  needs a deliberate occupancy policy instead of inheriting that side effect.

## Existing test footholds

- Unit tests: `pkg/config/app_config_test.go`, `keybinding_test.go`,
  `user_config_validation_test.go`; `pkg/commands/git_commands/branch_test.go`
  (add difftool argv coverage alongside command tests);
  `pkg/gocui/parent_view_test.go`.
- Commit actions: `pkg/integration/tests/commit/checkout.go`, `new_branch.go`,
  `directory_diff_with_renamed_files.go`; `pkg/integration/tests/reflog/checkout.go`.
- Branch operations: `pkg/integration/tests/branch/checkout_autostash.go`,
  `new_branch_autostash.go`, `rebase_onto_base_branch.go`, branch-prefix tests,
  and branch-stack scenarios under `branch/` and `sync/`.
- Defaults/forwarding: `pkg/integration/tests/sync/fetch_and_auto_forward_branches_*.go`.
- UI: `pkg/integration/tests/ui/keybinding_suggestions_*.go`,
  `pkg/integration/tests/filter_and_search/filter_menu_*.go`, and
  `pkg/integration/tests/custom_commands/`.
- Worktree safety: `pkg/integration/tests/worktree/fast_forward_worktree_branch*.go`
  and occupied-branch scenarios under `sync/`.
- `justfile`: use named `just e2e` scenarios during iteration; `just generate`
  updates generated test lists/keybinding/config outputs. This environment
  currently lacks `just`; see the main plan's workflow note.
