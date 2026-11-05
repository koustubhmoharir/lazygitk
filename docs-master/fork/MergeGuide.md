# Upstream merge guide

How lazygitk diverges from upstream lazygit, and what to check when merging
upstream changes. Keep this guide current in the same commit as any change to
an integration point. It describes the code as it is, not its history; the
behaviour it protects is in the [user guide](Guide.md) and the reasons in the
[design principles](Design.md).

## Baseline

- Upstream: `https://github.com/jesseduffield/lazygit.git` (remote `origin`).
- Last integrated upstream commit: `4e1eebe278790f257a875da6633234391a5a85b3`.
- Against that base the fork modifies 104 upstream source files and 157
  upstream tests, and adds 380 files of its own (168 source, 212 test).

## Layout of fork code

Most fork behaviour lives in files upstream does not have, so upstream changes
rarely touch them:

- `pkg/gui/fork_*.go`: key projection, prefix menus, palette usage and
  ordering, hints and labels, focus frames, refresh, clipboard writer.
- `pkg/gui/keymap/`: the default key catalogue (`default_paths.go`), command
  trees and path resolution.
- `pkg/gui/controllers/fork_*.go`, `branch_sync_*.go`, `combine_*.go`,
  `working_file_diff_actions.go`, `tree_navigation.go` and similar:
  fork actions and bindings.
- `pkg/gui/controllers/helpers/captured_*.go`, `named_checkout.go`,
  `fork_open_repo_path.go`: flows that capture their target before prompting.
- `pkg/commands/git_commands/` additions (`branch_placement.go`,
  `branch_action_*.go`, `fork_ref_commit.go`, `pull_rebase_policy.go`, …):
  native Git plans for those flows.
- `pkg/commandusage/`: the palette usage store.
- `pkg/config/fork_*.go`: fork defaults, config directory, theme palette,
  diff renderers, ignored key settings.
- `pkg/integration/tests/**/fork_*.go`: fork scenarios.
- `new-features/inventory/` and `new-features/keymap/`: generators for the
  static binding inventory and the instantiated key snapshot.

Strings are added to `pkg/i18n/english.go` only; translations are untouched.

## Merging

1. Merge upstream into a branch and record the new base commit above.
2. Resolve conflicts with the integration points below. Also review upstream
   changes near those points that merged without conflicts: a new code path
   can bypass a fork hook silently.
3. Give every new upstream binding and menu item a `CommandID` and `Verb`.
   `TestBuiltinCommandIdentityMetadata` (`pkg/gui/command_identity_test.go`)
   fails until they have them. A command whose ID is not in
   `pkg/gui/keymap/default_paths.go` has no key and is reachable only from
   `?`; decide whether it deserves a key under the
   [key principles](Design.md#keys), and if so ask the user.
4. If upstream solves a problem the fork also solves, weigh consolidating
   against the agreed behaviour in the user guide rather than keeping
   redundant code.
5. Regenerate (`go generate ./...`), refresh the snapshots (below), then run
   formatting, build, `go test ./... -short`, lint and the integration suite
   (`go test -timeout 60m pkg/integration/clients/*.go`, about 7 minutes).
6. Update this guide's paths and notes in the merge commit.

Generated outputs: `go generate ./...` rewrites
`docs-master/keybindings/*`, `schema-master/*` and
`pkg/integration/tests/test_list.go`. Snapshots, from the repository root:
`env GOMAXPROCS=2 go run new-features/keymap/generate.go >
new-features/keymap/bindings.json` and
`go run new-features/inventory/generate.go`.

## Integration points

Each entry names the upstream file, what the fork changed and what to keep
when merging. "Replacement" means upstream code was removed or rerouted;
re-apply it if upstream changes that code.

### Configuration

- `pkg/config/user_config.go`: `GetDefaultConfigForPlatform` wraps its literal
  in `applyForkDefaults` (`fork_defaults.go`: explicit-action defaults,
  `showWholeGraph: true`, and the palette's theme fields cleared so the
  Catppuccin palette applies). Adds fork
  fields: `CommandBindings`, the historical
  diff and placement key fields, `CustomCommand.ID`, and the `yaml:"-"`
  provenance fields.
- `pkg/config/app_config.go` (replacement): `findConfigFile` and
  `stateFilePath` use only `AppDirName` (`lazygitk`); upstream's `lazygit/` and
  `jesseduffield/lazygit/` lookups are removed. `loadUserConfig` discards key
  settings per file (`fork_ignore_key_config.go`: one capture before the file
  loop, one discard before `Validate`). Keep new upstream path logic on
  `AppDirName`.
- `pkg/gui/gui.go` `getPerRepoConfigFiles`: `lazygitk.yml` names. New upstream
  scenarios that write `lazygit.yml` must write `lazygitk.yml`.
- `pkg/config/user_config_validation.go`: validates `commandBindings` and
  custom command IDs.
- `pkg/config/theme_config.go`: `ThemeForBackground` merges
  `forkThemePalette` between `gui.theme` and upstream's `themeDefaults`.
- `pkg/gui/gui.go` `applyTheme`: calls `theme.UpdateBasicColors`
  (`pkg/theme/fork_basic_colors.go`), which replaces `style.FgBlue` with the
  palette's Blue. Upstream code that hard-codes `style.FgBlue` gets it too;
  `style.ColorMap` keeps the terminal's blue for config values.
- `pkg/config/diff_renderer_config_manager.go`: every read of
  `Git.DiffRenderers` goes through `forkEffectiveDiffRenderers`, which turns
  an empty list into delta plus Git's diff. Route any new read through it.
  The default delta renderer's command is built in Go from the render values
  (`forkDeltaCommand`): `resolveCommand` returns
  `DiffRendererConfig.forkResolveCommand` (`user_config.go`) when it is set,
  and `DiffRendererValues.TrueColor`, set in `main_view_render.go`, decides
  whether it passes its dark tints. It leaves out settings that the git
  config sets; `gui.go` connects the git config through
  `SetForkGitDeltaConfig` (`pkg/gui/fork_delta_config.go`).
- Key configuration is ignored for users, but the integration runner sets
  `LAZYGITK_HONOUR_KEY_CONFIG` (`pkg/integration/components/runner.go`) so
  upstream scenarios that drive legacy key fields run unchanged.
  `recordExplicitKeybindingFields` records nothing without it. New upstream
  config sections that carry keys need adding to the discard.

### Git commands

- `git_commands/rebase.go` (replacement): `PrepareInteractiveRebaseCommand`
  returns an error, checks for a clean start, passes `--no-autostash`, can set
  `rebase.updateRefs=false` and disables lock retries. Every start site goes
  through `runInteractiveRebase`. `SquashAllAboveFixupCommits` and `AmendTo`
  have clean-tree preflights. New upstream rebase starts must use the wrapper.
- `git_commands/patch.go` (replacement): `MovePatchIntoIndex` lost its stash
  parameter and autostash; `MovePatchToSelectedCommit` uses the wrapper.
- `git_commands/sync.go`: `Pull` resolves `effectivePullRebasePolicy`; a
  rebasing pull checks for a clean start and passes `--no-autostash`.
- `git_cmd_obj_runner.go` and `oscommands/cmd_obj.go`: `DisableLockRetries`
  and subprocess success exit codes.
- `oscommands/os.go` (replacement): `CopyToClipboard` runs
  `os.copyToClipboardCmd` when set and otherwise the writer injected with
  `SetClipboardWriter` (OSC 52, `pkg/gui/fork_clipboard.go`, wired right after
  `gui.os = osCommand`). Paste still uses `atotto/clipboard`. Keep the writer
  injected so `oscommands` never imports gocui.
- `git_commands/fork_external_tool.go`: `ConfiguredExternalTool`,
  `AvailableExternalTools` (parsed from `--tool-help`), `ApplyExternalTool`
  (one launch, through `GIT_CONFIG_COUNT`) and `SaveExternalToolGlobally`. Shared
  by the difftool and the merge tool through `ExternalToolMode`.
- `git_commands/fork_merge_tool.go`: `MergeToolCmdObj` builds
  `git --literal-pathspecs mergetool --no-prompt -- <paths>` and adds
  `-c mergetool.keepBackup=false` only while that key is unset. It reads the key
  fresh on each call and refuses an empty path list.
- `git_commands/working_tree.go` (replacement): `OpenMergeToolCmdObj`, which ran
  `git mergetool` with no paths, is removed.
- `git_commands/commit_loader.go`: `GetCommitsOptions` carries the commit
  scope (`ScopeHashes`, `ScopeDescendants`, `ScopeRef`, `ScopeTips`). `GetCommits`
  loads a scoped list with `loadCommitScope` and skips rebase todos for it;
  `getLogCmd` takes its revisions, order, `--all` and limit from
  `commitScopeLog` (`fork_commit_scope.go`). New `git log` arguments that depend on the ref or
  limit must go through `commitScopeLog`. A scope walks the unfiltered
  revisions and order and drops commits while it reads, so an upstream change
  to the unfiltered log arguments changes the scoped walk with it.
- `git_commands/fork_unmerged_paths.go`: `UnmergedPaths` asks Git for the
  unmerged paths (`git diff --name-only --diff-filter=U -z`) for the `c c`
  refusal. Keep it fresh: the Files model may be stale.
- `git_commands/status.go`: `IsInCherryPick` and `IsInRevert` also report a
  paused sequence, one whose CHERRY_PICK_HEAD or REVERT_HEAD is absent but
  whose `sequencer/todo` starts with that command (`fork_paused_sequence.go`).
  `WorkingTreeState` sets `SequencePaused` for it. Keep the existing
  CHERRY_PICK_HEAD and stopped-sha logic as it is.
- `git_commands/commit_loader.go`: `getHydratedSequencerCommits` drops the first
  sequencer entry when `SequencePaused` and that entry is committed
  (`withoutCommittedSequencerPick`), and marks no commit conflicted then. It
  does not synthesize a conflicted commit from CHERRY_PICK_HEAD while paused.
- `models/working_tree_state.go`: the `SequencePaused` field. `CanSkip` is
  unchanged. Code that builds a `WorkingTreeState` by hand must set it.

### Keys and dispatch

- `pkg/gui/types/keybindings.go`: `Binding` gains `CommandID`,
  `CommandIDFunc`, `Verb`, `VerbFunc`, `CommandPaths`, `DefaultCommand` and
  `CommandActive`.
- `pkg/gui/keybindings.go` (replacement): bindings pass once through
  `projectForkBindings` (`fork_command_projection.go`), which assigns the
  catalogue paths. `resetKeybindings` registers fork commands, key labels and
  panel jump labels; `callKeybindingHandler` records usage. Adds the global
  `s n` binding. `resetKeybindings` stores the bindings it registers, and
  `GetInitialKeybindingsWithCustomCommands` (the options bar each frame and
  the palette) returns them for the view current at the reset instead of
  rebuilding (`fork_binding_cache.go`). Applicability is still evaluated per
  call. `controllers.go` `resetHelpersAndControllers` clears the cache; any new
  upstream path that replaces controllers or bindings without
  `resetKeybindings` must clear it too.
- `pkg/gocui/keybinding.go` and `gui.go`: command bindings match through
  `NormalizeCommandKey` (case-insensitive letters) and are skipped while an
  editable view has focus.
- `pkg/gui/context/menu_context.go` (replacement): prefix menus
  (`SetBeforeAccept`, `SetBlockUnmatchedKeys`), labels from
  `Binding.KeyLabels()`, physical navigation keys, and `CommandActive`.
- `pkg/gui/menu_panel.go` and `pkg/gui/context.go`: invalidate an open prefix
  on a new menu or a focus change; `Activate` updates focus frames.
- `pkg/gui/options_map.go` and `pkg/gui/command_log_panel.go` (replacement):
  hints and tips name effective key paths through `CommandKeyLabel`; tips for
  keys with no route are dropped. The operation-in-progress block adds the
  `operation.continue-in-progress` hint after the `g o` hint, so it renders
  first (`fork_command_hints.go`).
- `pkg/cheatsheet/generate.go`: labels from `binding.KeyLabels()`.
- `controllers/list_controller_trait.go`: selection disabled reasons are
  `Inapplicable`, so an unusable key does nothing quietly.
- `controllers/jump_to_side_window_controller.go`: numbered pane jumps keep
  distinct identities even without keys.
- Hints that used to read upstream key fields now ask `CommandKeyLabel`
  (`commit_diff_actions.go`, `helpers/gpg_helper.go`). Do the same for new
  upstream hints that name a key.

### Palette

- `controllers/options_menu_action.go` (replacement): the palette lists every
  pane's commands, merged by command ID, in pane sections ordered by key group
  (`command_palette.go`, `fork_command_palette_order.go`). Other panes'
  commands are static copies whose callbacks are never run. Selections count
  as palette uses.
- `context/menu_context.go` `GetDisplayStrings`: an inapplicable disabled
  reason is dimmed instead of struck through (all menus). `OnMenuPress` shows
  the reason for items with `MenuItem.ExplainDisabled` (palette items) even
  when an inapplicable default command would otherwise be silent.
- `pkg/gui/popup/popup_handler.go`: `Menu`, `Confirm`, `ConfirmIf` and `Prompt`
  pass through the usage hook (`popup/fork_usage_hook.go`), so a use counts
  only when its first popup is accepted. Popups opened later by background
  workers are not tracked.
- `pkg/gui/gui_common.go`, `types/common.go`: `CommandKeyLabel`, palette
  recording and usage count reset. `gui.go` installs the hook.
- `services/custom_commands/client.go`: custom commands carry a stable
  `CommandIdentity`.

### Controllers

- `controllers/global_controller.go`: appends the fork globals: cherry-pick
  (`fork_cherry_pick.go`), continue the operation in progress
  (`fork_continue_operation.go`), fetch (`fork_fetch.go`), previous branch
  (`fork_checkout_previous.go`) and usage reset. `GetKeybindings` wraps its
  list in `addFlatFilterBindings` (`fork_flat_filter_keys.go`), which adds
  `w p`, `w u` and `w c`; their prompts copy those of upstream's filtering
  menu (`filtering_menu_action.go`), so check them against it when merging.
  `filter.options-menu` keeps its command but has no default key.
- `controllers/files_controller.go`: adds the flat copy menu, `d w`/`d s`/`d c`
  and tree navigation. `repo.fetch` and `fetch` are removed (now global);
  re-apply that removal or fetch is registered twice. Conflict hints use
  `resolveKeyLabel`.
- `controllers/branches_controller.go` (replacement): wrapped in
  `addSelectedResetBindings`; merge, rebase, fast-forward, reset, rename,
  delete and new branch go through captured flows; `branch.checkout-previous`
  and `checkoutPreviousBranch` are removed (now global; re-apply the removal).
- `controllers/remote_branches_controller.go` (replacement): delete, merge,
  rebase and new local branch go through captured flows; adds placement.
- `controllers/basic_commits_controller.go`: appends placement, rebase onto
  commit, combine, historical diffs, reset modes and flat copy bindings.
- `controllers/local_commits_controller.go` (replacement): paste is the
  reviewed cherry-pick of the selection (`fork_cherry_pick.go`); fixup and
  amend creation preflight their placement;
  `GetKeybindings` appends the commit scope filters
  (`fork_commit_scope_filters.go`) and Space's selection toggle
  (`fork_commit_multi_select.go`).
- `context/local_commits_context.go`: `LocalCommitsViewModel` holds the
  multi-select set; `fork_commit_multi_select.go` in context overrides the
  cursor methods that end a range, `GetSelectedItems`, focus and render.
  `GetSelectedRefRangeForDiffFiles` and `RefForAdjustingLineNumberInDiff`
  use the cursor commit while a set exists. New upstream `ListCursor` entry
  points that cancel a range need an override there.
- `controllers/local_commits_controller.go`, `basic_commits_controller.go` and
  `bisect_controller.go`: `GetKeybindings` returns through
  `guardCommitSelectionBindings`, which disables the commands in
  `commitSelectionScoped` while Commits holds a set. Add new upstream Commits
  commands that use the selected commit to that list. `handleCommitDragPress`
  falls through while a set exists. `pkg/gui/global_handlers.go`
  `getCopySelectedSideContextItemToClipboardDisabledReason` disables `y i`
  then.
- `controllers/quit_actions.go`: `Escape`, `EscapeEnabled` and
  `EscapeDescription` first try the Commits steps (filter, then selection;
  `forkCommitSelectionEscape`).
- `controllers/branches_controller.go`, `remote_branches_controller.go` and
  `tags_controller.go`: `GetKeybindings` adds `w b`
  (`fork_branch_scope_filter.go`, `fork_ref_scope_filter.go`).
- `controllers/branches_controller.go`, `remote_branches_controller.go`,
  `tags_controller.go`, `reflog_commits_controller.go` and
  `sub_commits_controller.go`: `GetKeybindings` adds `w v` show in Commits
  (`fork_show_in_commits.go`, `helpers/fork_show_in_commits.go`); Reflog and
  Sub-commits gain a `GetKeybindings` for it. If upstream adds its own
  `GetKeybindings` there, keep the binding in it.
- `controllers/commits_files_controller.go`: tree navigation, flat copy and
  historical diffs; copy menu items are shared with the flat bindings.
- `controllers/sync_controller.go` (replacement): push and pull act on the
  captured focused branch through the branch-sync flows.
- `controllers/main_view_controller.go`, `working_tree_diff_actions.go`
  (replacement), `merge_conflicts_controller.go`: mode-dependent command
  identities for Space and `x`; `x` confirms discards except the staged-side
  unstage.
- `controllers/custom_patch_options_menu_action.go` (replacement): no stash
  confirmation; the backend refuses a dirty tree.
- `controllers/stash_controller.go`: a new branch from a stash uses the
  entry's hash, not its shifting index.
- `controllers/prompt_controller.go`, `helpers/confirmation_helper.go`,
  `context/confirmation_context.go`, `types/common.go`: the optional
  `HandleTab` prompt hook used by `s n`. Keep the pass-throughs when upstream
  changes prompt options.
- Default `s c` routes: `projectForkBinding` replaces the handlers of
  `branch.checkout`, `remote-branch.checkout`, `tag.checkout` and
  `commit.checkout-menu` with the flat checkouts in
  `helpers/fork_flat_checkout.go` (`pkg/gui/fork_selected_checkout.go`).
  Upstream's `CreateCheckoutMenu`, `CheckoutRemoteBranch` and
  `CheckoutPreviousRef` stay for explicit legacy configuration only; they use
  `CheckoutRef`, which offers an autostash, so fork defaults must not call
  them. `s d` (`commit.checkout-detached`) attaches through
  `fork_checkout_detached.go` and `RefCommitCopyController`.
- In Status, `s` is for repositories: the global `repo.switch-menu` binding in
  `keybindings.go` gets no path (Status's own binding keeps `s r`),
  `repo.open-path` is a Status binding, and `s p` is inapplicable there.
- `controllers/switch_to_focused_main_view_controller.go`: `WithInspect` makes
  Files' `v` inspect, and adds the focus-only `navigation.focus-main` (`0`).
- `controllers/switch_to_sub_commits_controller.go`: `WithoutDoubleClick`, so
  Worktrees can show commits on `v` while its double-click still switches.
  A context accepts only one double-click handler; a second one panics.
- `pkg/gui/controllers.go`: attaches the fork controllers: Files inspection,
  Worktrees commits, and `RefCommitCopyController` for Branches, Remote
  branches and Tags.

### Difftool launches

Every difftool launch goes through `helpers/fork_external_tool.go`
(`ExternalToolHelper.WithExternalTool`), which asks for a tool when none is
configured (`docs-master/fork/Design.md`, "Tools are never guessed"). A new
difftool launch must use it: build the command, then run it from the `launch`
callback after `apply` has set the tool on it. Pass a `cancel` that releases
anything the caller holds, since it runs whenever nothing launches. The
integration points are:

- `controllers/basic_commits_controller.go` `openDiffTool`: a commit against
  its parent or a range.
- `controllers/commits_files_controller.go` `openDiffTool`: a commit's file or
  directory.
- `controllers/files_controller.go` `openDiffTool`: a Files entry.
- `controllers/working_file_diff_actions.go` `diffWorkingFileWithStaged`: the
  index against the working file.
- `controllers/historical_diff_actions.go` `runHistoricalRevisionDiff`: the
  revision and directory diffs from commits and Files.
- `controllers/historical_diff_actions.go` `runPreparedHistoricalFileDiff`: the
  temporary historical copy of a file, closed exactly once after the tool exits
  or when nothing launches.
- `controllers/historical_commit_comparison.go` `prompt`: the pasted-hash
  comparison, keeping its generation-guarded refresh.
- `helpers/diff_helper.go` `OpenDiffToolForRef`: branches, remote branches and
  tags.

These run on the UI thread, where menus open. The merge tool has its own
launch, described under "Merge tool" below.

### Merge tool

- `helpers/working_tree_helper.go` `OpenMergeTool` (replacement): upstream ran
  `git mergetool` with no paths, which guesses a tool. It now takes the paths
  and launches through `WithExternalTool(ExternalToolMerge, ...)`, then
  `RunSubprocessAndRefresh`. `CreateMergeConflictMenu` passes the files the
  menu was opened on. The helper takes the `ExternalToolHelper` as a
  constructor argument, wired in `pkg/gui/controllers.go`; a test fixture that
  builds the helper passes nil, which is safe as long as it does not launch.
- `controllers/fork_merge_tool.go`: the `conflict.open-merge-tool-selected`
  bindings for Files (flattened selection, conflicted files only, disabled as
  inapplicable with none) and for the conflicts view (the file being resolved).
  `controllers/files_controller.go` and `merge_conflicts_controller.go` each
  append one line to `GetKeybindings` to include them.
- `keymap/default_paths.go`: `d m` routes to that command.

### Helpers (replacements)

- `helpers/merge_and_rebase_helper.go`: upstream's merge and rebase menus are
  removed; `CreateRebaseOptionsMenu`, `RebaseOntoRef`,
  `MergeRefIntoCheckedOutBranch` and related functions delegate to
  `captured_merge*.go` and `captured_rebase*.go`. While `SequencePaused`, its
  skip item stays in the menu, disabled with `SkipCommittedPickDisabled`.
  Fork: `SetOnOperationFinished` holds a callback, which `fork_cherry_pick.go`
  registers when a pick stops on conflicts. `genericMergeCommandImpl` calls
  `settleOperationFinished` after each continue, skip or abort, in both its
  subprocess and worker paths, so the auto-skip is covered. A continue or skip
  that ends the operation runs the callback once; an abort drops it. Upstream
  changes to these functions must keep that call.
- `helpers/refs_helper.go`: checkout of remote branches, the reset menu,
  the checkout menu, new branch and move-commits-to-new-branch use captured
  selections (`captured_reset.go`, `named_checkout.go`,
  `captured_branch_create.go`). Named checkout bypasses autostash.
- `helpers/tags_helper.go`: tag creation uses `captured_tag_create.go`.
- `helpers/refresh_helper.go` `performRefresh` and `types/refresh.go`:
  `RefreshOptions.ThenIfRepoUnchanged` runs on the UI thread after all of a
  refresh's updates (and a batch's single-frame apply), before input is
  unblocked, and only if no repo switch happened meanwhile. Unlike `Then`, it
  may be combined with `DontBlockRepoSwitch`.

Upstream changes to these menus should be ported into the captured flows, not
restored.

Commit scope filters (`w a`, `w d`, `w b`) are a third kind of filtering mode:

- `modes/filtering/filtering.go`: a `commitScope` field
  (`fork_commit_scope.go`); `Active()` counts it so that upstream's
  filter-mode restrictions apply, and `Reset()` clears it.
- `helpers/mode_helper.go`: the filtering status label defers to
  `commitScopeInfoLabel`; `changeFiltering` takes its screen layout from
  `changeFilteringWithLayout`, and `ClearFiltering` keeps the screen mode for a
  scope (`fork_commit_scope_filter.go`). `pathFilteringLayout` (replacement)
  only focuses Commits: upstream's switch to a half screen on applying a path
  or author filter, and back to normal on clearing any filter, is removed.
  The startup half screen for `--filter` and git arguments is upstream's.
- `helpers/refresh_helper.go`: the commits refresh passes the scope; the
  reflog filters only for a path or author (`HasPathOrAuthorFilter`).
- `helpers/diff_helper.go`: `DiffArgs` appends the filter path only when one is
  set.
- Upstream code that tests `Filtering.Active()` to mean a path or author
  filter should use `HasPathOrAuthorFilter` or `GetPath`.

### Integration test watchdog

`gui/test_mode.go` allows 120 seconds per scenario, not upstream's 40. Slow
scenarios take over 40 seconds when other work is using the machine, whatever
the test parallelism. Keep this value when upstream changes that line.

### Commit message drafts

Up and Down in the commit message box step through the worktree's cancelled
drafts, instead of earlier commits' messages. The behaviour is in the
[user guide](Guide.md#commit-message-box).

- `controllers/commit_message_controller.go` and
  `commit_description_controller.go`: Escape calls
  `Commits.CancelCommitMessagePanel`, which saves the drafts and then closes the
  panel. `GetOnQuit` also calls `SaveCommitDraftIfChanged`. Up and Down are
  `handleRecallOlderDraft` and `handleRecallNewerDraft`, and carry the
  descriptions `RecallOlderCommitDraft` and `RecallNewerCommitDraft`
  (`i18n/english.go`), so the palette lists them. Their CommandIDs and keys are
  upstream's. Upstream's `handleCommitIndexChange` and
  `setCommitMessageAtIndex` are removed; re-apply that removal if upstream
  changes them.
- `helpers/commits_helper.go`: `HandleCommitConfirm` takes the draft session's
  state before `OnConfirm`, because `OnConfirm` closes the panel, which ends the
  session. `CloseCommitMessagePanel` ends it too. `OpenCommitMessagePanel` records
  the text the box shows as the session's opened text, after setting it in the
  view. `SwitchToEditor` keeps the text that Up replaced.
  `SaveCommitDraftIfChanged`, `CancelCommitMessagePanel`,
  `RecallOlderCommitDraft`, `RecallNewerCommitDraft` and
  `TakeCommitMessageSubmitReporter` are fork code.
- `controllers/local_commits_controller.go`: `handleReword` passes a rebase-path
  failure to `rewordFailure` (`controllers/fork_reword_failure.go`), which explains
  the stopped rebase. Re-apply this if upstream changes that error path.
- Submissions that finish on a worker: a panel opened with
  `OpenCommitMessagePanelOpts.ReportsSubmitResult` writes the text as a draft
  before `OnConfirm`, and its `OnConfirm` takes a reporter with
  `TakeCommitMessageSubmitReporter` and calls it on success. Those call sites are
  `helpers/working_tree_helper.go` (`HandleCommitPressWithMessage`,
  `handleCommit`), `controllers/local_commits_controller.go` (`reword`,
  `handleReword`, `createAmendCommit`) and
  `controllers/custom_patch_options_menu_action.go` (both
  `handlePullPatchIntoNewCommit` functions). A new upstream panel whose work runs
  on a worker should do the same, or a failure there loses its text.
- `context/fork_commit_draft.go` (new): the drafts file,
  `LAZYGITK_COMMIT_DRAFTS` (a JSON list, newest first), beside
  `LAZYGIT_PENDING_COMMIT`, and `commitDraftSession`. In
  `context/commit_message_context.go`, the view model's `drafts` field holds the
  session, `SetPanelState` resets it, and `RenderSubtitle` shows the position.
- Retained on purpose: `OpenCommitMessagePanelOpts.CommitIndex`, the index
  parameter of `SetPanelState` and the `selectedindex` field. Upstream's call
  sites (local commits, custom patch options, working tree) still pass indexes,
  and they now drive nothing. Their `// Fork:` comments say so. Removing them
  would mean editing those call sites for no gain.
- `git_commands/commit.go`: `GetCommitMessageFromHistory` and
  `ErrInvalidCommitIndex` are removed, with `TestGetCommitMessageFromHistory`.
  `TryRemoveHardLineBreaks` stays; other callers use it.
- Generated: `docs-master/keybindings` lists the two descriptions. Run
  `go generate ./...` after merges.

### Dormant upstream cherry-pick marks

Upstream's copy and paste of commits (`commit.copy-for-cherry-pick`, `C`) kept
a set of marked commits in `Modes().CherryPicking`. The fork has no way to
mark commits any more: `c p` picks the selection (`fork_cherry_pick.go`), and
the mark bindings, the source menu and `CherryPickHelper.CopyRange` are
removed. Nothing sets the set, so upstream's marks stay disconnected and their
code below never runs. It is left in place because removing it means editing
many upstream files that upstream merges also touch. When merging, keep these
dormant rather than restoring the marks:

- `modes/cherrypicking/cherry_picking.go`: the `CherryPicking` state and its
  `Active`, `CanPaste`, `SelectedHashSet`, `Add`, `Remove`. `Active` and
  `CanPaste` are always false.
- `types/modes.go` and `gui.go`: the `Modes().CherryPicking` field and its
  construction.
- `helpers/mode_helper.go`: the cherry-pick mode banner, its reset, and
  `cherryPickHelper` in `NewModeHelper`.
- `context/local_commits_context.go`, `reflog_commits_context.go` and
  `sub_commits_context.go`: the presentation highlight of marked commits,
  which is always empty.
- `helpers/cherry_pick_helper.go`: `Paste`, `CanPaste` and `Reset`, with
  `rerender`. `Paste` was already replaced by `c p`; `Reset` only serves the
  dormant banner. `CopyRange` and its `resetIfNecessary` are removed.
- `pkg/i18n/english.go`: `CherryPickCopy`, `CherryPickCopyTooltip`,
  `NoCopiedCommits`, `ResetCherryPick`, `ResetCherryPickShort`,
  `CommitsCopied` and `CommitCopied`, which are used only by the dormant
  banner or are unused. The fork-added ones were removed. `NoCommitsSelected`
  is `c p`'s reason for an empty selection.
- `options_map.go`: the options-bar hint for an active cherry-pick mode is
  removed, as the mode can no longer be active.
- `config/user_config.go`: `Commits.CherryPickCopy` (`cherryPickCopy`) and
  `Commits.ResetCherryPick` (`resetCherryPick`) are kept, so the schema, the
  docs and the test config still list them. Nothing binds them now, so a
  user's setting for either does nothing. Removing them is a separate change.

The generated legacy-field map and the keybinding cheatsheets no longer list
the removed commands.

### Appearance

- `pkg/gocui/tcell_driver.go` `tcellInit`: passes
  `trueColorScreenOptions` (`fork_true_color.go`), which asks the terminal
  whether it shows true colour before tcell starts, when `COLORTERM` doesn't
  say so. Keep it before `NewTerminfoScreenFromTty` if tcell's start-up
  changes.
- `pkg/gocui/view.go`: `setCharacter` takes the cell's content line, and an
  explicit set of selected lines (`SetSelectedLines`,
  `fork_selected_lines.go`) replaces the cursor and range highlight while set.
  New upstream callers of `setCharacter` must pass the content line.
  `SelectSearchResult` skips the sideways scroll for views with
  `KeepOriginXOnSearch`, which `context/setup.go` `NewContextTree` sets on
  every list context's view (`fork_list_search_scroll.go`).
- `pkg/gui/presentation/commits.go` `loadPipesets`: the pipe set cache key's
  `commitHash` is a digest of every hash in the list (`commitListKey`,
  `fork_pipe_set_key.go`), because scoped and whole-graph lists can share
  their first commit and length. F5 drops the cache (`DropPipeSetCache`,
  called from `fork_full_refresh.go`).
- `pkg/gui/views.go` (replacement): upstream's border-style switch is removed;
  frame runes come from `updateFocusFrames` (`fork_focus_frame.go`), called
  from `configureViewProperties`, `ContextMgr.Activate` and initial layout.
  Port new upstream border styles into `focusFrameStyle`.
  It also sets the Commits cursor marker (`▶`) for multi-select mode.
- Screen mode area (`types/fork_screen_mode_area.go`): `GuiRepoState` has a
  `ScreenModeArea` field, and `SetScreenMode` resets it to the side area;
  `IRepoStateAccessor` gains `GetScreenModeArea` and `SetScreenModeInArea`.
  `window_arrangement_helper.go` `getMidSectionWeights` and
  `mainSectionChildren` decide the enlarged area from `args.ScreenModeArea`
  instead of `CurrentWindow`; upstream changes that test `CurrentWindow`
  for the screen mode should use the area. `screen_mode_actions.go` records
  the focused area (`controllers/fork_screen_mode_area.go`), and
  `ContextMgr.Activate` (`context.go`) calls `applyScreenModeForFocus`
  (`gui/fork_screen_mode_area.go`). The integration `GuiDriver` gains
  `ScreenMode` and `ScreenModeArea` (`TestDriver.ExpectScreenMode`,
  `ViewDriver.HasScreenArea`, `components/fork_screen_mode.go`).
- `pkg/gui/layout.go`: focus frames, and the startup notices for an
  unreadable usage file, ignored key settings and missing delta.

### Other

- `helpers/repos_helper.go` `switchToLocation`: shows "Loading…" in every
  side and main pane and draws it with `ForceFlushViewsContentOnly` before
  the synchronous switch, and renders the current repo again if the switch
  fails before `onNewRepo` succeeds (`helpers/fork_repo_switch.go`). Keep the
  named error result the deferred restore relies on.
- `pkg/gui/gui.go` `loadNewRepo`: a switch's refresh sets
  `ThenIfRepoUnchanged` to the "Opened <repo>" toast (`fork_repo_switch.go`).
  Integration tests that switch repos must expect that toast.
- `pkg/gui/gui.go` `onNewRepo`: calls `updateForkTerminalTitle`
  (`fork_terminal_title.go`) after the key-config notices, so startup and
  every repository switch set the terminal title. tcell saves and restores the
  terminal's title itself; keep that behaviour in mind when upstream changes
  screen setup. The harness reads the title through
  `gocui.MockTerminalTitle` (`pkg/gocui/fork_mock_title.go`),
  `GuiDriver.TerminalTitle` and `TestDriver.ExpectTerminalTitle`.

- `pkg/app/daemon/daemon.go`: registers `DaemonKindCombineRebase`.
- `pkg/i18n/english.go` `englishIntroPopupMessage`: the first-run popup only
  names the fork and keeps the note on donating to lazygit's author.
  `pkg/i18n/i18n.go` `newTranslationSet` restores it after merging a
  translation, whose text is upstream's. `misc/initial_open` checks the fork
  text.
- `AGENTS.md`: replaced by the fork's short version; expect a conflict on
  every merge and keep the fork's file, folding in relevant upstream technical
  constraints deliberately.
- `justfile` `dist` recipe and `/dist/` in `.gitignore`; the build scripts set
  `buildSource=forkDist` so upstream's self-updater never replaces a fork
  build.
- `test/default_test_config/config.yml`: commented legacy key overrides for
  scenarios.

### Binding metadata only

These files only gained `CommandID`/`Verb` metadata on bindings or menu items.
Conflicts there are mechanical; keep upstream's change and re-add the
metadata:

In `pkg/gui/controllers/`: `bisect_controller.go`,
`confirmation_controller.go`, `context_lines_controller.go`,
`diffing_menu_action.go`, `edit_config_action.go`, `filter_controller.go`,
`filtering_menu_action.go`, `git_flow_controller.go`, `list_controller.go`,
`menu_controller.go`, `remotes_controller.go`,
`rename_similarity_threshold_controller.go`, `search_controller.go`,
`search_prompt_controller.go`, `side_window_controller.go`,
`snake_controller.go`, `status_controller.go`, `submodules_controller.go`,
`suggestions_controller.go`, `switch_to_diff_files_controller.go`,
`tags_controller.go`, `undo_controller.go`, `workspace_reset_controller.go`,
`worktrees_controller.go`. In `helpers/`: `branches_helper.go`,
`diff_line_file_menu.go`, `refresh_helper.go`,
`repos_helper.go`, `worktree_helper.go`. Also
`pkg/gui/extras_panel.go` and
`pkg/gui/services/custom_commands/keybinding_creator.go`.
`go run new-features/inventory/check_annotations.go -base <commit> <files…>`
confirms that a file differs from a baseline only in this metadata.

### Upstream tests

Modified upstream tests fall into a few groups; update new upstream tests in
the same way:

- Scenarios that rely on automation now off by default set it explicitly (for
  example `AutoStageResolvedConflicts = true`).
- Scenarios through reviewed flows confirm the review
  (`ReviewBranchRebase` and similar helpers in
  `pkg/integration/components/common.go`).
- Per-repo config scenarios write `lazygitk.yml`.
- Scenarios that press Space in Commits to open a commit's files press `v`
  instead; Space toggles the selection there.
- `TestBackgroundDefaultsAreDerivedFromTheBackgroundColor` calls
  `themeDefaults` directly; `TestCurrentDiffRendererNameWithoutDiffRenderers`
  expects `default`.
- The shared fixture sets an explicit `rawGit` renderer so scenarios never use
  delta and disables fetch/file/ref polling.

Integration components gained `ExpectClipboard`, `ExplicitSelection`, `Views().Extras()`, a
menu `Content` matcher and `ReviewBranch*` helpers. Drivers need `Title()` and
`Content()` on a confirmation and `Title()` on a menu before confirming.
