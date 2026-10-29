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
  fields: `CommandBindings`, `GuiConfig.CommandPaletteOrder`, the historical
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
- `pkg/config/user_config_validation.go`: validates `commandBindings`,
  `gui.commandPaletteOrder` and custom command IDs.
- `pkg/config/theme_config.go`: `ThemeForBackground` merges
  `forkThemePalette` between `gui.theme` and upstream's `themeDefaults`.
- `pkg/config/diff_renderer_config_manager.go`: every read of
  `Git.DiffRenderers` goes through `forkEffectiveDiffRenderers`, which turns
  an empty list into delta plus Git's diff. Route any new read through it.
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
- `git_commands/commit_loader.go`: `GetCommitsOptions` carries the commit
  scope (`ScopeHashes`, `ScopeDescendants`, `ScopeRef`, `ScopeTips`). `GetCommits`
  loads a scoped list with `loadCommitScope` and skips rebase todos for it;
  `getLogCmd` takes its revisions, order, `--all` and limit from
  `commitScopeLog` (`fork_commit_scope.go`). New `git log` arguments that depend on the ref or
  limit must go through `commitScopeLog`. A scope walks the unfiltered
  revisions and order and drops commits while it reads, so an upstream change
  to the unfiltered log arguments changes the scoped walk with it.

### Keys and dispatch

- `pkg/gui/types/keybindings.go`: `Binding` gains `CommandID`,
  `CommandIDFunc`, `Verb`, `VerbFunc`, `CommandPaths`, `DefaultCommand` and
  `CommandActive`.
- `pkg/gui/keybindings.go` (replacement): bindings pass once through
  `projectForkBindings` (`fork_command_projection.go`), which assigns the
  catalogue paths. `resetKeybindings` registers fork commands, key labels and
  panel jump labels; `callKeybindingHandler` records usage. Adds the global
  `s n` binding.
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
  keys with no route are dropped.
- `pkg/cheatsheet/generate.go`: labels from `binding.KeyLabels()`.
- `controllers/list_controller_trait.go`: selection disabled reasons are
  `Inapplicable`, so an unusable key does nothing quietly.
- `controllers/jump_to_side_window_controller.go`: numbered pane jumps keep
  distinct identities even without keys.
- Hints that used to read upstream key fields now ask `CommandKeyLabel`
  (`commit_diff_actions.go`, `helpers/gpg_helper.go`). Do the same for new
  upstream hints that name a key.

### Palette

- `controllers/options_menu_action.go` (replacement): palette entries are
  merged by command ID and ordered by usage (`command_palette.go`,
  `fork_command_palette_order.go`); selections count as palette uses.
- `pkg/gui/popup/popup_handler.go`: `Menu`, `Confirm`, `ConfirmIf` and `Prompt`
  pass through the usage hook (`popup/fork_usage_hook.go`), so a use counts
  only when its first popup is accepted. Popups opened later by background
  workers are not tracked.
- `pkg/gui/gui_common.go`, `types/common.go`: `CommandKeyLabel`, palette
  recording, usage counts and reset. `gui.go` installs the hook.
- `services/custom_commands/client.go`: custom commands carry a stable
  `CommandIdentity`.

### Controllers

- `controllers/global_controller.go`: appends the fork globals: cherry-pick
  (`fork_cherry_pick.go`), fetch (`fork_fetch.go`), previous branch
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
  commit, combine, historical diffs, reset modes, cherry-pick marks and flat
  copy bindings.
- `controllers/local_commits_controller.go` (replacement): paste is the
  reviewed cherry-pick; fixup and amend creation preflight their placement;
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

### Helpers (replacements)

- `helpers/merge_and_rebase_helper.go`: upstream's merge and rebase menus are
  removed; `CreateRebaseOptionsMenu`, `RebaseOntoRef`,
  `MergeRefIntoCheckedOutBranch` and related functions delegate to
  `captured_merge*.go` and `captured_rebase*.go`.
- `helpers/refs_helper.go`: checkout of remote branches, the reset menu,
  the checkout menu, new branch and move-commits-to-new-branch use captured
  selections (`captured_reset.go`, `named_checkout.go`,
  `captured_branch_create.go`). Named checkout bypasses autostash.
- `helpers/tags_helper.go`: tag creation uses `captured_tag_create.go`.

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

### Appearance

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
`commit_description_controller.go`, `commit_message_controller.go`,
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
`commits_helper.go`, `diff_line_file_menu.go`, `refresh_helper.go`,
`repos_helper.go`, `working_tree_helper.go`, `worktree_helper.go`. Also
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
  delta, disables fetch/file/ref polling, and selects the static palette
  order.

Integration components gained `ExpectClipboard`, `ExplicitSelection`, `Views().Extras()`, a
menu `Content` matcher and `ReviewBranch*` helpers. Drivers need `Title()` and
`Content()` on a confirmation and `Title()` on a menu before confirming.
