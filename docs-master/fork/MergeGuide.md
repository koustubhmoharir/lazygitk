# Upstream merge guide

How lazygitk diverges from upstream lazygit, and what to check when merging
upstream changes. Keep this guide current in the same commit as any change to
an integration point. It describes the code as it is, not its history; the
behaviour it protects is in the [user guide](Guide.md) and the reasons in the
[design principles](Design.md).

## Baseline

- Upstream: `https://github.com/jesseduffield/lazygit.git`, the remote
  `upstream`. The fork's own remote is `origin`.
- Last integrated upstream commit: `4e1eebe278790f257a875da6633234391a5a85b3`.
- Against that base, at commit `09ef9d41d`, the fork modifies 133 upstream
  source files, 203 upstream test files (199 modified, 4 deleted) and 6 other
  upstream files. It adds 620 files: 241 source, 346 test, 30 other and 3
  generated. Recompute with the recipe below after a merge, or whenever files
  are added or removed.

Counting recipe, from the repository root. Status `M` is an upstream file that
the fork modified, `D` an upstream file it deleted and `A` a file it adds. The
generated files are counted apart. Test files are `*_test.go` and the scenarios
under `pkg/integration/tests/`; source files are the other `.go` files.

```sh
B=4e1eebe278790f257a875da6633234391a5a85b3
git diff --no-renames --name-status $B HEAD | awk -F'\t' '{
  p = $2
  if (p ~ /^(docs-master\/keybindings\/|docs-master\/Config\.md$|schema-master\/|pkg\/integration\/tests\/test_list\.go$|pkg\/gui\/keymap\/legacy_fields_generated\.go$|new-features\/(keymap\/bindings|inventory\/inventory)\.json$)/) k = "generated"
  else if (p ~ /_test\.go$|^pkg\/integration\/tests\//) k = "test"
  else if (p ~ /\.go$/) k = "source"
  else k = "other"
  print $1, k }' | sort | uniq -c
```

The output lists, for each status and category, the number of paths. The
`docs-master/Config.md` file is generated, so it is counted as generated, though
its hand-written paragraphs are the fork's too.

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
- `pkg/commandusage/`: the palette usage store. Its counts file is updated
  under a lock from `pkg/utils/filelock/`.
- `pkg/utils/filelock/` (new): the exclusive OS file lock that serializes
  updates to lazygitk's own files. It is a lock on a `<file>.lock` file, which is
  never deleted. The OS releases the lock when the holding process exits.
- `pkg/config/fork_*.go`: fork defaults, config directory, theme palette,
  diff renderers, ignored key settings.
- `pkg/integration/tests/**/fork_*.go`: fork scenarios.
- `new-features/inventory/` and `new-features/keymap/`: generators for the
  static binding inventory and the instantiated key snapshot.

Fork strings live only in `pkg/i18n/fork_english.go`: fields of
`ForkTranslationSet`, which `TranslationSet` embeds, with values set by
`forkEnglishTranslationSet`. That function also applies the fork's changed
English values (`MustExitFilterModePrompt`, `IntroPopupMessage`). `english.go`
differs from upstream only by the embed and that call, and the translation
files match upstream. Add new fork strings there, never to `english.go`.

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
`docs-master/keybindings/*`, the generated block of `docs-master/Config.md`,
`schema-master/*`, `pkg/gui/keymap/legacy_fields_generated.go` and
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
- `pkg/config/app_config.go` `AppConfigurer` (addition): `PrepareRepoUserConfig`
  and `CommitRepoUserConfig` (`fork_repo_user_config.go`). A repo switch loads
  the new repo's config, checks it, and only then makes it current. Keep both
  methods on the interface; `ReloadUserConfigForRepo` stays as upstream.
- `pkg/gui/gui.go` `getPerRepoConfigFiles`: `lazygitk.yml` names. It takes the
  repo's Git command as an argument, because `onNewRepo` calls it before
  `gui.git` is assigned. New upstream scenarios that write `lazygit.yml` must
  write `lazygitk.yml`.
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

- `git_commands/captured_combine_plan.go`: a selection that is not in HEAD's
  history fails with `CombineCommitsNotInHeadError`, which lists the commits.
  The combine flow (`controllers/fork_combine.go`, `planError`) turns it into
  the translated `CombineCommitsNotInHead`. Keep that mapping if the error
  changes.
- `git_commands/rebase.go` (replacement): `PrepareInteractiveRebaseCommand`
  returns an error, checks for a clean start, passes `--no-autostash`, can set
  `rebase.updateRefs=false` and disables lock retries. It is the shared generic
  entry point: `runInteractiveRebase` calls it, as do `RewordCommitInEditor` and
  the captured branch rebase (`branch_action_rebase.go`). `AmendTo` has a
  clean-tree preflight before its fixup commit, and its rebase then goes through
  `runInteractiveRebase`. New generic upstream rebase starts must use
  `PrepareInteractiveRebaseCommand` or `runInteractiveRebase`. Some starts build
  their own command, each with its own checks: `SquashAllAboveFixupCommits`
  (clean check and ignored-file review in `rebase.go`), the selected-commit
  rebase (`selected_commit_rebase.go`, clean check), the combine rebase
  (`combine_command.go`, with the clean check in `captured_combine_plan.go`) and
  the reviewed pull rebase (`branch_sync_pull_review.go`, with its clean check in
  the plan). All of them pass `--no-autostash`. `GitRebaseEditTodo` edits an
  existing operation and needs no clean start.
  The same function calls `requireIgnoredRebaseStart` after the clean check
  (`git_commands/rebase_ignored_start.go`, new), which refuses with
  `IgnoredRebaseRiskError` when Git's rewind could overwrite ignored files.
  Approval reaches it through `PrepareInteractiveRebaseCommandOpts.approvedIgnoredFingerprint`
  (set by `BranchActionRebaseCommand` from its own review) or through the
  scope `RebaseCommands.ApproveIgnoredRebaseRisks`, which the GUI uses.
  `AmendTo` runs `CheckCompoundRebaseIgnoredRisks` before it creates the fixup,
  and `RebaseRiskAfterCommit` stops a retry after the fixup exists.
  `RebaseCommands` gained the field `ignoredRebaseApproval`. Upstream's clean
  check stays in front of it.
- `git_commands/rebase_ignored_files.go` (fork-owned): `rebaseIgnoredPaths`
  delegates to `rebaseCandidateRisks(rebaseReplay)`. Its candidates are the
  destination tree, the replayed paths, the extras and the paths predicted by
  `predictedRebaseCandidates`. The selected-commit and branch-action reviews use
  it unchanged.
- `git_commands/rebase_ignored_prediction.go` (new, fork-owned): the extra
  paths that Git's merge backend creates by directory-rename detection (07-F1).
  - Mirrored-path rule: a directory rename moves a replayed `old/<rest>` to
    `new/<rest>`. Renames come from the fork point (`merge-base(upstream, head)`)
    to the destination, from the replayed range and from the staged extras. Each
    rename `a/b/c/f -> x/y/c/f` implies `a/b/c -> x/y/c`, then `a/b -> x/y` while
    the trailing components keep matching. A replayed or staged path under an old
    directory gives a candidate FILE path in the new directory. No directory is
    ever a candidate. A new directory that is the root is allowed. The diffs pass
    `diff.renameLimit` set to `merge.renameLimit` (default 7000), so detection
    finds at least what the merge backend finds. It runs no merge drivers.
  - Gate: the mirrored candidates are checked against the ignored files first. If
    none collides, nothing is added and the merge-tree chain is skipped. Git's
    merge drivers, which the chain can run, then do not run at all.
  - Chain: when the gate passes, a linear replay onto a destination (Git 2.40 or
    later, within `rebaseMergeTreeChainLimit` = 200 commits, not reordered) runs
    `merge-tree --write-tree --merge-base=<commit>^` and `commit-tree` for each
    commit. Its intermediate trees' paths replace the mirrored candidates, since
    they are exact. A conflicted step continues from its conflicted tree, so a
    manual resolution is not predicted. The chain runs in private object storage
    and changes no repository objects, refs, index or worktree files. A failure
    is logged at debug level and falls back to the mirrored candidates.
  - Trade-off: the gate assumes the mirrored rule covers every path that Git's
    directory-rename detection can create. The raised rename limit is what makes
    that assumption reasonable. Any change to the detection needs a new look here.
  - Merge commits, root rebases, Git older than 2.40 and reordered replays use the
    mirrored candidates. A reordered replay never runs the chain, because its
    order is not known here.
  - Counter: `mergeTreeChainRuns` counts chain starts, for the gate test only.
- `git_commands/private_object_store.go` (new, fork-owned): `privateObjectStore`,
  the temporary `GIT_OBJECT_DIRECTORY` whose alternate is the repository's objects,
  with `writeMergeTreeInStore`, `commitTreeInStore` and `treePaths`. The isolated
  merge preview (`PreviewBranchActionOrtMerge` in `branch_action_merge_preview.go`)
  uses the same helpers. Keep them shared, and check the preview tests after a
  change to either side.
- Replay order by caller (`rebase_ignored_start.go`). `rebaseStartReplay` builds
  the replay for every guard. The selected-commit rebase, the branch-action rebase
  (plain or marked base) and generic starts without a moving instruction use the
  original order, which is the order Git replays. `instructionMovesCommits` marks
  `MoveFixupCommitDown` and `MoveTodosUp`/`MoveTodosDown` starts reordered. `AmendTo`
  and fixup placement (`CheckCompoundRebaseIgnoredRisks`) and squash-above
  (`--autosquash`) are reordered too. No caller passes a computed order. The
  mirrored candidates cover a reordered replay.
  `CheckCompoundRebaseIgnoredRisks` adds the staged paths and the renames of the
  staged changes (`parseNameStatus`).
- `controllers/fork_rebase_ignored_review.go` (new): `runWithIgnoredRebaseReview`
  wraps every generic start in the GUI. A refusal opens the selected-commit
  review menu, and approval retries the start under the reviewed fingerprint.
  Wrapped call sites: `local_commits_controller.go` (reword in the editor,
  drop merge commit, edit, squash/fixup/drop, move, amend-to, the three author
  edits, the fixup placement paths), `fork_reword_target.go` (both reword
  submit paths), `custom_patch_options_menu_action.go` and `commit_diff_actions.go`
  (custom-patch moves and removals), and `commits_files_controller.go` (discard
  from a commit). `fixup_placement.go` (fork-owned) runs the placement
  preflight before the commit, and `placementRiskAfterCommit` stops a retry
  after it.
- `SquashAllAboveFixupCommits` builds its own `--autosquash` command, so it runs
  `requireIgnoredRebaseSquashAbove` itself before the command. Its GUI entry is
  wrapped in `squashFixupsImpl`.
- The approval scope is protected by `RebaseCommands.ignoredRebaseApprovalMu`,
  which guards the field only. Overlapping scopes can see each other's approval,
  which is harmless because each start still compares fresh fingerprints.
- Sentinel refusals reach the user as fork strings: `runWithIgnoredRebaseReview`
  maps `ErrRebaseRiskAfterCommit` and `ErrRebaseIgnoredApprovalRequired` through
  `mapIgnoredRebaseError` (`controllers/fork_rebase_ignored_review.go`). Fixup and
  amend flows that refuse after their commit use `finishAfterFixupPlacement`
  (`controllers/fixup_placement.go`), which keeps the amend message out of the draft
  and refreshes, so a resubmit cannot create a second commit.
  `BeginInteractiveRebaseForCommit` and `BeginInteractiveRebaseForCommitRange`
  refuse an index outside the commits list, from either bound, through
  `requireCommitIndex` (`git_commands/fork_commit_index.go`).
- `git_commands/patch.go` (replacement): `MovePatchIntoIndex` lost its stash
  parameter and autostash; `MovePatchToSelectedCommit` uses the wrapper and
  checks both indexes first; `PullPatchIntoNewCommitBefore` checks its index
  before it adds one.
- `git_commands/fork_bisect_log_ref.go` (new): `BisectCommands.LogRef` is the
  ref the unfiltered Commits log walks. `helpers/refresh_helper.go` `refForLog`
  and `IsCommitInCommitsLog` (`w v`) both use it.
- `git_commands/sync.go`: `Pull` resolves `effectivePullRebasePolicy`; a
  rebasing pull checks for a clean start and passes `--no-autostash`.
- `git_commands/sync.go` (replacement): `Fetch` and `FetchCmdObj` are removed,
  so the global f no longer reads `git.fetchAll` (U1). Its replacements are
  `FetchRemote` (kept here, used by Remotes) and `FetchAllRemotes` in
  `git_commands/fork_fetch.go`, which also holds the refspec risk check and the
  automatic fetch's remote choice. `FetchBackgroundCmdObj` still follows
  `git.fetchAll`. Re-apply the removal if upstream changes the fetch path.
- `git_cmd_obj_runner.go` and `oscommands/cmd_obj.go`: `DisableLockRetries`
  and subprocess success exit codes.
- `oscommands/os.go` (replacement): `CopyToClipboard` runs
  `os.copyToClipboardCmd` when set and otherwise the writer injected with
  `SetClipboardWriter` (OSC 52, `pkg/gui/fork_clipboard.go`, wired right after
  `gui.os = osCommand`). Paste still uses `atotto/clipboard`. Keep the writer
  injected so `oscommands` never imports gocui. The writer needs the running
  gocui screen and returns an error before it starts.
- `git_commands/fork_external_tool.go`: `ConfiguredExternalTool` (keys in Git's
  order, with the GUI keys first in GUI mode, as `get_configured_merge_tool` in
  `git-mergetool--lib`; GUI mode follows `GIT_MERGETOOL_GUI`, then the mode's
  `guiDefault`, where `auto` means `DISPLAY` is set), `AvailableExternalTools`
  (parsed from `--tool-help`), `ApplyExternalTool` (inserts `--tool=<name>`
  after the subcommand, so Git never guesses) and `SaveExternalToolGlobally`.
  Shared by the difftool and the merge tool through `ExternalToolMode`.
- `git_commands/fork_merge_tool.go`: `MergeToolCmdObj` builds
  `git --literal-pathspecs mergetool --no-prompt -- <paths>` and adds
  `-c mergetool.keepBackup=false` only while that key is unset. It reads the key
  fresh on each call and refuses an empty path list.
- `git_commands/fork_literal_pathspecs.go` (new): `OpenLiteralDiffToolCmdObj`,
  `GetLiteralPathsDiff` and `ShowLiteralFileDiffCmdObj` wrap upstream's
  `OpenDiffToolCmdObj`, `getDiffCmdObj` and `ShowFileDiffCmdObj` and set
  `GIT_LITERAL_PATHSPECS=1`, so upstream changes to those builders carry over.
  Only the fork's call sites below use them; other callers keep Git's pathspec
  interpretation.
- `git_commands/working_tree.go` (replacement): `OpenMergeToolCmdObj`, which ran
  `git mergetool` with no paths, is removed.
- `git_commands/commit_loader.go`: `GetCommitsOptions` carries the commit
  scope (`ScopeHashes`, `ScopeDescendants`, `ScopeRef`, `ScopeTips`). `GetCommits`
  loads a scoped list with `loadCommitScope` and skips rebase todos for it;
  `getLogCmd` takes its revisions, order, `--all` and limit from
  `commitScopeLog` (`fork_commit_scope.go`). New `git log` arguments that depend on the ref or
  limit must go through `commitScopeLog`. A scope walks the unfiltered
  revisions and order and drops commits while it reads, so an upstream change
  to the unfiltered log arguments changes the scoped walk with it. The
  descendant scope is the exception: it walks `--all` in the unfiltered order,
  reads only until its reachable selected commits have been read, and parses
  only the rows it keeps (`loadCommitDescendants`, `descendantWalk`). Its
  reachability check streams `rev-list --all` (`reachableFromAll`), and does not
  use `rev-list <hash> --not --all`, whose limit walk trusts commit dates. Its
  limit is not a git `-300`: a limited scope keeps the newest `commitScopeLimit`
  rows, which are the first kept rows of the walk. The scoped ancestors filter
  and the descendant rows both read `commitScopeLimit`, the one place that sets
  the scope limit. When the upstream `Limit` bool becomes an int, the `-300` in
  `getLogCmd` and that constant must follow it together.
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
  panel jump labels; `callKeybindingHandler` records usage through
  `callHandlerWithForkUsage` (`fork_command_usage.go`). Adds the Status-only
  `s n` binding (`repo.open-path`). `resetKeybindings` stores the bindings it registers, and
  `GetInitialKeybindingsWithCustomCommands` (the options bar each frame and
  the palette) returns them for the view current at the reset instead of
  rebuilding (`fork_binding_cache.go`). Applicability is still evaluated per
  call. `controllers.go` `resetHelpersAndControllers` clears the cache; any new
  upstream path that replaces controllers or bindings without
  `resetKeybindings` must clear it too.
- `pkg/gocui/keybinding.go` and `gui.go`: command bindings match through
  `NormalizeCommandKey` (case-insensitive letters) and are skipped while an
  editable view has focus.
- `pkg/gui/context/menu_context.go` (replacement): prefix menus, labels from
  `Binding.KeyLabels()`, physical navigation keys, and `CommandActive`. The
  setters `SetBeforeAccept` and `SetBlockUnmatchedKeys` are in
  `context/fork_prefix_menu.go`. The prefix key overrides and blocked-key
  bindings in `GetKeybindings` are in `context/fork_menu_context.go`.
- `pkg/gui/menu_panel.go` and `pkg/gui/context.go`: invalidate an open prefix
  on a new menu or a focus change; `Activate` updates focus frames. The physical
  essential keys of a prefix menu are in `pkg/gui/fork_menu_panel.go`.
- `pkg/gui/options_map.go` and `pkg/gui/command_log_panel.go` (replacement):
  hints and tips name effective key paths through `CommandKeyLabel`; tips for
  keys with no route are dropped. `keyTip`, `commandRoute` and the options-map
  hint filters (`localFirstForkHintBindings`, `forkHintBindingInContext`) are in
  `fork_command_hints.go`. The operation-in-progress block adds the
  `operation.continue-in-progress` hint after the `g o` hint, so it renders
  first (`fork_command_hints.go`).
- `pkg/cheatsheet/generate.go`: labels from `binding.KeyLabels()`.
- `controllers/list_controller_trait.go`: selection disabled reasons are
  `Inapplicable`, so an unusable key does nothing quietly.
- `controllers/jump_to_side_window_controller.go`: numbered pane jumps keep
  distinct identities even without keys. `focusPanelTitle` is in
  `controllers/fork_jump_to_side_window.go`.
- Hints that used to read upstream key fields now ask `CommandKeyLabel`
  (`commit_diff_actions.go`, `helpers/gpg_helper.go`). Do the same for new
  upstream hints that name a key.

### Palette

- `controllers/options_menu_action.go` (replacement): `Call` delegates to
  `openPalette` in `controllers/fork_palette_menu.go`, which lists every pane's
  commands, merged by command ID, in pane sections ordered by key group
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
  (`fork_continue_operation.go`), fetch (`fork_fetch.go`, with the palette's Fetch all remotes), previous branch
  (`fork_checkout_previous.go`) and usage reset. `GetKeybindings` wraps its
  list in `addFlatFilterBindings` (`fork_flat_filter_keys.go`), which adds
  `w p`, `w u` and `w c`; their prompts copy those of upstream's filtering
  menu (`filtering_menu_action.go`), so check them against it when merging.
  `filter.options-menu` keeps its command but has no default key.
- `controllers/files_controller.go`: adds the flat copy menu, `d w`/`d s`/`d c`
  and tree navigation. The copy menu's items (`copyMenuItems`) are in
  `controllers/fork_files_copy.go`, beside the flat bindings; `openCopyMenu`
  stays here as the seam. `repo.fetch` and `fetch` are removed (now global);
  re-apply that removal or fetch is registered twice. Conflict hints use
  `resolveKeyLabel`. The copy-selected-diff item calls
  `GetLiteralPathsDiff` (literal pathspecs) rather than upstream's `GetDiff`.
- `controllers/undo_controller.go` (replacement, CHECKOUT, REBASE and COMMIT
  redo cases): undo and redo of a checkout confirm with `UndoCheckoutPrompt` and
  then call `CheckoutReflogTarget`. Undo of a rebase, and redo of a rebase or a
  commit, call `Refs.ResetReflogTarget` (`helpers/fork_undo_reset.go`). It
  refuses tracked changes and otherwise runs the captured hard reset with the
  undo or redo reflog marker. It replaces upstream's `hardResetWithAutoStash`,
  which stashed and popped; that method is removed. Undo of a commit still
  soft-resets with `ResetToRef`. The upstream `HardResetAutostashPrompt` string
  is no longer used by undo.
- `controllers/branches_controller.go` (replacement): wrapped in
  `addSelectedResetBindings`; merge, rebase, fast-forward, reset, rename,
  delete and new branch go through captured flows; `branch.checkout-previous`
  and `checkoutPreviousBranch` are removed (now global; re-apply the removal).
  `checkoutByName` (`branch.checkout-by-name`) keeps upstream's prompt and
  suggestions, but its confirm delegates to `CheckoutByNameFlow`
  (`helpers/fork_checkout_by_name.go`), so upstream's `CheckoutRef` and
  `CheckoutRemoteBranch` are not reached from it.
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
  points that cancel a range need an override there. The set is never pruned
  for absence (a filter or the commit limit hides members without removing
  them), and the sync sets the Commits subtitle. The commands' hashes come
  from `SelectedCommitHashes`, through `forkSelectionHashes` in
  `controllers/fork_commit_multi_select.go`; `GetSelectedItems` still lists only
  the loaded commits.
- `controllers/local_commits_controller.go` `selectRebaseResultCommit`, which
  drop, fixup and squash call before they rewrite, calls
  `SetSelectionAfterRewrite` instead of `SetSelection`. The override would add a
  Shift run to the set first, and the rewrite would then leave those commits
  unreachable. Upstream's use of `SetSelection` there is otherwise unchanged.
- `controllers/local_commits_controller.go` `move`, which the keys and the
  drag release both call, calls `ForgetRewrittenCommits` before it rewrites the
  block. A drag whose cursor moves mid-drag ends its run into the set first, so
  the moved commits leave the set there and are selected again as the run.
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
- `controllers/tags_controller.go`: `GetKeybindings` adds the captured-ref
  placement binding `branch.place-at-selected-commit` (`branchPlacementBinding`
  in `branch_placement_refs.go`, the same hook Branches and Remote branches
  use). `ResolveBranchActionCommit` peels an annotated tag to its commit. If
  upstream adds its own binding with that command ID, keep one.
- `controllers/branches_controller.go`, `remote_branches_controller.go`,
  `tags_controller.go`, `reflog_commits_controller.go` and
  `sub_commits_controller.go`: `GetKeybindings` adds `w v` show in Commits
  (`fork_show_in_commits.go`, `helpers/fork_show_in_commits.go`); Reflog and
  Sub-commits gain a `GetKeybindings` for it. If upstream adds its own
  `GetKeybindings` there, keep the binding in it.
- `controllers/commits_files_controller.go`: tree navigation, flat copy and
  historical diffs. The copy menu's items (`copyMenuItems`) are in
  `controllers/fork_commit_files_copy.go`, shared with the flat bindings;
  `openCopyMenu` stays here as the seam.
  `copyDiffToClipboard` calls `ShowLiteralFileDiffCmdObj` (literal pathspecs)
  rather than upstream's `ShowFileDiffCmdObj`.
- `controllers/sync_controller.go` (replacement): push and pull act on the
  captured focused branch through the branch-sync flows.
- `controllers/main_view_controller.go`, `working_tree_diff_actions.go`
  (replacement), `merge_conflicts_controller.go`: mode-dependent command
  identities for Space and `x`; `x` confirms discards except the staged-side
  unstage.
- `controllers/custom_patch_options_menu_action.go` (replacement): no stash
  confirmation; the backend refuses a dirty tree. The source-editing items are
  disabled, and their handlers refuse, when the patch's source commit is not in
  the commits list (`controllers/fork_patch_source.go`).
- `controllers/stash_controller.go`: a new branch from a stash uses the
  entry's hash, not its shifting index.
- `controllers/prompt_controller.go`, `helpers/confirmation_helper.go`,
  `context/confirmation_context.go`, `types/common.go`: the optional
  `HandleTab` prompt hook used by `s n`. Keep the pass-throughs when upstream
  changes prompt options. The `s n` hook returns handled at once and does its
  directory reads on a worker (`helpers/fork_open_repo_path.go`); when the
  result arrives, focus moves to the suggestions if nothing completed, as the
  prompt's own fallback does. Keep that in mind if the hook contract changes.
- Default checkout routes: `projectForkBinding`, for a binding the user has
  not set explicitly, replaces the handlers of `branch.checkout`,
  `remote-branch.checkout`, `tag.checkout` and `commit.checkout-menu` with
  `forkSelectedCheckoutHandler` (`pkg/gui/fork_selected_checkout.go`), which
  calls the flat checkouts in `helpers/fork_flat_checkout.go`. `s d`
  (`commit.checkout-detached`) attaches through `fork_checkout_detached.go` and
  `RefCommitCopyController`. `s p` runs `CheckoutPreviousFlat` through
  `controllers/fork_checkout_previous.go`; `CheckoutPreviousFlat` reads the
  previous checkout from HEAD's reflog (`git_commands/fork_previous_checkout.go`,
  `ClassifyPreviousCheckout`), not from `@{-1}`, which is ambiguous when a
  branch and a tag share a name. The palette's "Checkout by name"
  (`branch.checkout-by-name`) runs `CheckoutByNameFlow`, which dispatches to
  the same flat checkouts (`branches_controller.go`, `helpers/fork_checkout_by_name.go`,
  classifier `git_commands/fork_checkout_by_name.go`).
- Upstream checkout handlers, by status. Each one that uses `CheckoutRef`
  runs plain `git checkout` and offers an autostash, so fork defaults must not
  reach it:
  - Explicit legacy only. A key the user binds to the command keeps the
    upstream handler, since the replacement above skips explicit bindings:
    `BranchesController.checkoutBranch` (`branch.checkout`, `CheckoutRef`),
    `TagsController.checkout` (`tag.checkout`, `CheckoutRef`),
    `RemoteBranchesController.checkoutBranch` (`remote-branch.checkout`,
    `CheckoutRemoteBranch`, which shows upstream's new-or-detached menu and
    then calls `CheckoutRef`), and `BasicCommitsController.checkout`
    (`commit.checkout-menu`, `CreateCheckoutMenu`, which lists `CheckoutRef`
    actions).
  - Unused. `CheckoutPreviousRef` has no callers and is marked `// Fork:
    unreachable`; do not reconnect it. `CreateSelectedRefCheckoutMenu` and
    its tests (`helpers/selected_ref_checkout.go`, `selected_ref_checkout_test.go`)
    were removed: no production code called them, and their tests covered the
    unused menu rather than the default handlers.
  - Replaced for undo and redo of a checkout (`controllers/undo_controller.go`,
    `CHECKOUT` action, which calls `CheckoutReflogTarget` in
    `helpers/fork_undo_checkout.go`). The reflog action is passed as
    `envVars` to `ApplyNamedCheckout` and `CheckoutCapturedCommit`, so the
    entry keeps its undo or redo marker. `CheckoutAutostashPrompt` is no
    longer used by undo; the upstream string stays.
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
callback after `apply` has set the tool on it (`--tool=<name>`, on every
launch). Pass a `cancel` that releases anything the caller holds, since it runs
whenever nothing launches. The integration points are:

- `controllers/basic_commits_controller.go` `openDiffTool`: a commit against
  its parent or a range.
- `controllers/commits_files_controller.go` `openDiffTool`: a commit's file or
  directory, launched with `OpenLiteralDiffToolCmdObj` (literal pathspecs).
- `controllers/files_controller.go` `openDiffTool`: a Files entry, launched with
  `OpenLiteralDiffToolCmdObj` (literal pathspecs).
- `controllers/working_file_diff_actions.go` `diffWorkingFileWithStaged`: the
  index against the working file, launched with `OpenLiteralDiffToolCmdObj`
  (literal pathspecs).
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
  `runMergeTool` (`helpers/fork_merge_tool.go`), which refreshes Files whether
  the tool succeeds or fails; upstream `RunSubprocessAndRefresh` skips the
  refresh on failure. `CreateMergeConflictMenu` passes the files the
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
  `captured_merge*.go` and `captured_rebase*.go`. The three unused upstream
  merge methods are marked and listed under "Unreachable upstream code". While
  `SequencePaused`, its skip item stays in the menu, disabled with
  `SkipCommittedPickDisabled`.
  Fork: `SetOnOperationFinished` holds a callback, which `fork_cherry_pick.go`
  registers when a pick stops on conflicts. Both it and
  `settleOperationFinished` are in `helpers/fork_operation_settle.go`.
  `genericMergeCommandImpl` calls `settleOperationFinished` after each
  continue, skip or abort, in both its subprocess and worker paths, so the
  auto-skip is covered. A continue or skip
  that ends the operation runs the callback once; an abort drops it. Upstream
  changes to these functions must keep that call. When the operation ends
  outside lazygitk, the files refresh in `refresh_helper.go` drops a pending
  callback with its notice (`DropAbandonedOperationStep`), guarded by an epoch
  and an in-flight count so a refresh cannot drop a step that a lazygitk run is
  still settling.
- `helpers/fork_undo_checkout.go`: `CheckoutReflogTarget`, the undo and redo
  checkout (`git_commands/fork_undo_checkout.go`, `ClassifyReflogCheckoutTarget`).
  It replaces upstream's `CheckoutRef` there, which runs plain `git checkout`
  and offers an autostash; see the controllers' undo entry above.
- `helpers/fork_undo_reset.go` (new): `ResetReflogTarget`, the undo and redo
  reset. It passes `capturedResetOptions` to `captured_reset.go`: a `preflight`
  that calls `git_commands/branch_action_reset_risks.go`
  (`TrackedChangesPresent`), the title and warning for undo or redo, and the
  reflog `envVars`. Undo and redo of a rebase and of a commit use it; see the
  controllers' undo entry above.
- `helpers/captured_reset.go` (replacement of the reset flow): `s h`, the
  hard row of the reset menu, and undo or redo all reach `reviewHard`. It
  computes the hard reset's ignored or untracked risks on a worker
  (`HardResetIgnoredRisks`, `git_commands/branch_action_reset_risks.go`). With
  none it confirms as before; with risks it opens `ShowIgnoredRiskMenu`. The
  approval passes its fingerprint to `BranchActionResetCommand`, which refuses
  a changed list with `ResetIgnoredRiskError`; the flow then reviews afresh.
  `BranchActionResetCommand` takes the approval and the env vars as parameters
  after the strength. Its callers are only the captured flow and its tests.
- `helpers/fork_ignored_risk_review.go` (new): `ShowIgnoredRiskMenu`, the review
  menu shared by the hard reset and by the rebase starts. The rebase review
  (`controllers/fork_rebase_ignored_review.go`, `showIgnoredRebaseMenu`)
  delegates to it, keeping its command IDs and strings. Upstream merges that
  change either menu must keep the approve-first, Cancel-second layout.
- `helpers/refs_helper.go`: the reset menu and new branch use captured
  selections (`captured_reset.go`, `captured_branch_create.go`).
  `MoveCommitsToNewBranch` is a fork delegation to `helpers/fork_move_commits.go`
  (`git_commands/fork_move_commits.go`). It replaces upstream's reset-first
  variants, which were removed from `refs_helper.go` rather than kept as
  unreachable code. The flow captures the branch, upstream, base and unpushed
  commits at invocation, refuses tracked changes and never stashes, validates
  the name, and moves the current branch only after the new branch has the
  commits. For the base variant's conflicts, it registers its completion with
  `MergeAndRebaseHelper.SetOnOperationFinished` (see the cherry-pick entry).
  Its menu, labels and command IDs are upstream's. The flat checkouts (`fork_flat_checkout.go`,
  `fork_checkout_by_name.go`) use the captured named checkout
  (`named_checkout.go`). Its upstream checkout menus and `CheckoutRemoteBranch`
  are the legacy handlers listed under "Default checkout routes" above.
  Remote checkout with no local branch of that name is in
  `fork_remote_checkout.go`: `PrepareRemoteBranchCheckout` runs the named
  checkout's checks and requires the remote branch to still name the captured
  commit, then `ApplyRemoteBranchCheckout` runs one
  `git checkout --no-overwrite-ignore --no-guess -b <name> --track
  refs/remotes/<remote>/<branch>`. Git creates the branch only when the
  checkout succeeds. Detached checkout (`captured_detached_checkout.go`) runs
  `rejectWorktreeOperation` immediately before `git checkout --detach`.
- `helpers/tags_helper.go`: tag creation uses `captured_tag_create.go`.
- `helpers/refresh_helper.go` `performRefresh` and `types/refresh.go`:
  `RefreshOptions.ThenIfRepoUnchanged` runs on the UI thread after all of a
  refresh's updates (and a batch's single-frame apply), before input is
  unblocked, and only if no repo switch happened meanwhile. Unlike `Then`, it
  may be combined with `DontBlockRepoSwitch`.
- `helpers/refresh_helper.go` `refForLog` delegates to `BisectCommands.LogRef`;
  keep it and the `w v` check (`IsCommitInCommitsLog`) on that one decision.
- `helpers/refresh_helper.go` `refreshCommitsWithLimit` calls
  `checkSelectedCommitsAfterRefresh` (`helpers/fork_commit_set_refresh.go`) at
  the end of its UI-thread model update, after the selection is restored. It
  classifies the selected commits that the new list lacks, with
  `CommitCommands.ClassifySelectedCommits` (`git_commands/fork_commit_set_classify.go`),
  on a worker, and applies the result on the UI thread for the same repository.
  Keep the call inside that closure, so that a repository switch drops it.

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
  `ClearFiltering` (all filter kinds) restores the selection through
  `selectCommitAfterClearingFilter` (`helpers/fork_show_in_commits.go`), which
  loads the unfiltered list further when the limit hides both the cursor
  commit and the pre-filter commit. `changeFilteringWithLayout`'s `selectCommit`
  callbacks receive `land`, which they call once the selection is in place.
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
  (`i18n/fork_english.go`), so the palette lists them. Their CommandIDs and keys
  are upstream's. Upstream's `handleCommitIndexChange` and
  `setCommitMessageAtIndex` are removed; re-apply that removal if upstream
  changes them.
- `helpers/commits_helper.go`: `HandleCommitConfirm` takes the draft session's
  state before `OnConfirm`, because `OnConfirm` closes the panel, which ends the
  session. `CloseCommitMessagePanel` ends it too. `OpenCommitMessagePanel` records
  the text the box shows as the session's opened text, after setting it in the
  view. `SwitchToEditor` keeps the text that Up replaced. For a reported submit
  (`submitReportsResult`) it keeps the typed text as a draft and sets
  `pendingDraftUse` around `SwitchToEditor(filepath)`, as `HandleCommitConfirm`
  does, so `rewordTarget.switchToEditor` can take the reporter.
  `SaveCommitDraftIfChanged`, `CancelCommitMessagePanel`,
  `RecallOlderCommitDraft`, `RecallNewerCommitDraft` and
  `TakeCommitMessageSubmitReporter` are fork code in
  `helpers/fork_commit_drafts.go`.
  Draft saves return errors, and the callers decide. `CancelCommitMessagePanel`
  returns one before closing, so the box stays open. `HandleCommitConfirm` does
  not call `OnConfirm` when the pre-submit save fails. `SwitchToEditor` saves the
  drafts before it creates the message file, and does not switch on failure.
  `RecallOlderCommitDraft` and `RecallNewerCommitDraft` return the error and do
  not move. After a successful commit, the drafts' update is only logged,
  because the commit has already happened. `GetOnQuit` (in both controllers)
  logs a failed save: it runs after the main loop has ended, when nothing can
  be vetoed, and no quit key reaches the app while the box has focus.
- `controllers/local_commits_controller.go`: `reword` captures the target with
  `captureRewordTarget` (`controllers/fork_reword_target.go`), and the message
  panel's `OnConfirm` and `OnSwitchToEditor` are `rewordTarget.submit` and
  `rewordTarget.switchToEditor`. These replace upstream's `handleReword` and
  `switchFromCommitMessagePanelToEditor`, which are removed. Those read the live
  commits and selection when the message was submitted, so a commit that moved
  while the box was open was rewritten. Both submit paths verify the captured
  repository, HEAD's ref and ID, and (for HEAD) HEAD against the target, before
  any write, and refuse with `RewordTargetChanged` or `RewordRepositoryChanged`.
  The rebase path's failure goes to `rewordFailure`
  (`controllers/fork_reword_failure.go`), which explains the stopped rebase. It
  refreshes the repository state first, as the success path does, so the stopped
  rebase shows before input is released. Re-apply these if upstream changes
  either function.
- `git_commands/fork_reword_target.go` (new): `BranchCommands.CurrentHead`
  returns the checked-out ref and HEAD's ID for the capture and the checks.
- Submissions that finish on a worker: a panel opened with
  `OpenCommitMessagePanelOpts.ReportsSubmitResult` writes the text as a draft
  before `OnConfirm`, and its `OnConfirm` takes a reporter with
  `TakeCommitMessageSubmitReporter` and calls it on success. Those call sites are
  `helpers/working_tree_helper.go` (`HandleCommitPressWithMessage`,
  `handleCommit`), `controllers/fork_reword_target.go` (`rewordTarget.submit`),
  `controllers/local_commits_controller.go` (`createAmendCommit`) and
  `controllers/custom_patch_options_menu_action.go` (both
  `handlePullPatchIntoNewCommit` functions). A new upstream panel whose work runs
  on a worker should do the same, or a failure there loses its text.
  `CommitsHelper.SwitchToEditor` does the same for such a panel, so its
  `OnSwitchToEditor` must take the reporter too: `rewordTarget.switchToEditor`
  calls it once the editor is about to get the text, and
  `WorkingTreeHelper.switchFromCommitMessagePanelToEditor` calls it at once.
- `context/fork_commit_draft.go` (new): the drafts file,
  `LAZYGITK_COMMIT_DRAFTS` (a JSON list, newest first), beside
  `LAZYGIT_PENDING_COMMIT`, and `commitDraftSession`. Every change goes through
  `updateCommitDrafts`, which takes the lock `<drafts>.lock`, reads the file,
  applies the change, and writes a unique temporary file before renaming it.
  A malformed file is moved aside to `<drafts>.corrupt-<timestamp>`, and a toast
  names the new path (`CommitDraftsMovedAside`). Up and Down use
  `StepToOlderCommitDraft` and `StepToNewerCommitDraft`, which now return an
  error too. In
  `context/commit_message_context.go`, the view model's `drafts` field holds the
  session, `SetPanelState` resets it, and `RenderSubtitle` shows the position.
- Retained on purpose: `OpenCommitMessagePanelOpts.CommitIndex`, the index
  parameter of `SetPanelState` and the `selectedindex` field. Upstream's call
  sites (local commits, custom patch options, working tree) still pass indexes,
  and they now drive nothing. The field's `// Fork:` comment says so, and the
  call sites pass it unchanged. Removing them
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
  `rerender`. `Paste` was already replaced by `c p` and is marked
  unreachable (see "Unreachable upstream code"); `Reset` only serves the
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

### Unreachable upstream code

These upstream functions have no callers. They are kept rather than deleted,
because deleting them adds merge conflicts. Each carries a `// Fork:
unreachable` marker that names its replacement. Do not reconnect them.

- `helpers/merge_and_rebase_helper.go`: `RegularMerge`,
  `SquashMergeUncommitted` and `SquashMergeCommitted`. Replaced by the captured
  merge flow (`helpers/captured_merge.go`). They read live refs and config
  after the prompt.
- `helpers/cherry_pick_helper.go`: `Paste`. Replaced by the captured
  cherry-pick flow (`controllers/fork_cherry_pick.go`). It stashes implicitly.
- `helpers/refs_helper.go`: `CheckoutPreviousRef`. Replaced by
  `CheckoutPreviousFlat` (`controllers/fork_checkout_previous.go`). It runs
  plain `git checkout` and offers an autostash.

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
- `pkg/gui/layout.go`: focus frames. The startup notices are not here any
  more; they are queued by `onNewRepo` (below).

### Other

- `controllers/remotes_controller.go` `fetchAndCheckout` (replacement): the
  upstream body is now `fetchAndCheckoutReviewed`, and `fetchAndCheckout` runs
  the refspec review from `controllers/fork_fetch.go` before it, so every
  Remotes fetch, including the ones that check out a branch afterwards, is
  reviewed. Keep that wrapper when merging.
- `pkg/gui/background.go` `backgroundFetch`: skips the fetch when
  `fork_background_fetch.go` finds a remote it would fetch whose refspecs write
  into local branches, with a once-per-session notice. The
  `forkFetchGuard` field on `BackgroundRoutineMgr` holds that notice's state.
- `helpers/repos_helper.go` `switchToLocation`: shows "Loading…" in every
  side and main pane and draws it with `ForceFlushViewsContentOnly` before
  the synchronous switch. If the switch fails before the new repo is in place
  (including a failure of `onNewRepo`), the deferred restore puts back the
  working directory and process environment it snapshotted, and renders the
  current repo again (`restoreCurrentRepo`, `helpers/fork_repo_switch.go`).
  Keep the named error result the deferred restore relies on. The
  `RecordCurrentDirectory` call (the shell hand-off file,
  `LAZYGIT_NEW_DIR_FILE`) runs after `onNewRepo` succeeds; upstream ran it
  before. The not-a-repo VerifyInGitRepo failure returns `notAGitRepoError`,
  which the `s n` prompt checks with `errors.As` (`fork_open_repo_path.go`).
  The `ReposHelper.repoPathSeq` field belongs to `s n`.
- `pkg/gui/gui.go` `loadNewRepo`: a switch's refresh sets
  `ThenIfRepoUnchanged` to the "Opened <repo>" toast (`fork_repo_switch.go`).
  Integration tests that switch repos must expect that toast.
- `pkg/gui/gui.go` `onNewRepo`: nothing is published until the new repo's Git
  command and config are loaded (`PrepareRepoUserConfig`) and the language is
  checked (`checkRepoSwitchLanguage`, `pkg/gui/fork_repo_switch_check.go`, which
  uses `onUserConfigLoaded`'s own condition and call). Then the config
  (`CommitRepoUserConfig`) and `gui.git` are published together. Keep that
  order when merging. The function also queues the startup notices in a fixed order
  (`queueForkUsageResetNotice`, `showForkIgnoredKeyConfigNotices`,
  `queueForkDiffRendererFallbackNotice`). They are shown one after another, each
  for its full time, by the queue in `fork_startup_notices.go`, which does not
  change the status manager. Its `forkNoticeDuration` repeats the status
  manager's inline two seconds for info toasts and must match it. The
  runtime usage reset in `fork_command_usage.go` also goes through the queue.
  It calls `updateForkTerminalTitle`
  (`fork_terminal_title.go`) after the key-config notices, so startup and
  every repository switch set the terminal title. tcell saves and restores the
  terminal's title itself; keep that behaviour in mind when upstream changes
  screen setup. The harness reads the title through
  `gocui.MockTerminalTitle` (`pkg/gocui/fork_mock_title.go`),
  `GuiDriver.TerminalTitle` and `TestDriver.ExpectTerminalTitle`.

- `pkg/app/daemon/daemon.go`: registers `DaemonKindCombineRebase`.
- `pkg/i18n/fork_english.go` `forkIntroPopupMessage`: the first-run popup only
  names the fork and keeps the note on donating to lazygit's author. It is set
  by `forkEnglishTranslationSet`, and `pkg/i18n/i18n.go` `newTranslationSet`
  sets it again after merging a translation, whose intro text is upstream's
  `englishIntroPopupMessage` in `english.go`. `pkg/integration/tests/misc/initial_open.go` checks the
  fork text.
- `AGENTS.md`: replaced by the fork's short version; expect a conflict on
  every merge and keep the fork's file, folding in relevant upstream technical
  constraints deliberately.
- `justfile` `dist` recipe and `/dist/` in `.gitignore`; the build scripts set
  `buildSource=forkDist`. Only `buildBinary` builds may self-update, so
  `pkg/updates/updates.go` `CanSelfUpdate` gates both the update check (also
  for manual checks, which skip `skipUpdateCheck`) and `Update`, and
  `update_helper.go` `CheckForUpdateInForeground` reports the refusal instead
  of checking. Keep these gates when merging the updater.
- `test/default_test_config/config.yml`: commented legacy key overrides for
  scenarios.

### Binding metadata only

These files only gained `CommandID`/`Verb` metadata on bindings or menu items.
Conflicts there are mechanical; keep upstream's change and re-add the
metadata:

In `pkg/gui/controllers/`: `confirmation_controller.go`,
`context_lines_controller.go`, `diffing_menu_action.go`,
`edit_config_action.go`, `filter_controller.go`, `filtering_menu_action.go`,
`git_flow_controller.go`, `list_controller.go`, `menu_controller.go`,
`rename_similarity_threshold_controller.go`, `search_controller.go`,
`search_prompt_controller.go`, `side_window_controller.go`,
`snake_controller.go`, `status_controller.go`, `submodules_controller.go`,
`suggestions_controller.go`, `switch_to_diff_files_controller.go`,
`workspace_reset_controller.go`, `worktrees_controller.go`. In `helpers/`:
`branches_helper.go`, `diff_line_file_menu.go`, `worktree_helper.go`. Also
`pkg/gui/extras_panel.go`.

These files also changed behaviour and are not metadata-only. Keep them with
their integration entries: `bisect_controller.go` (the commit-selection guard in
`GetKeybindings`), `tags_controller.go` (the `w b`, `w v` and placement
bindings), `undo_controller.go` (Controllers), `helpers/refresh_helper.go` and
`helpers/repos_helper.go` (Helpers), and
`pkg/gui/services/custom_commands/keybinding_creator.go`, which computes each
custom command's identity (Palette).

`go run new-features/inventory/check_annotations.go -base <commit> <files…>`
confirms that a file differs from a baseline only in this metadata. On the 30
files above it passes 24; the six named here fail.

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
- `controllers/fork_daemon_main_test.go` adds a `TestMain` to the controllers
  package. It handles the daemon invocation, because a test that starts a real
  interactive rebase runs the test binary as the sequence editor. Keep one
  `TestMain` per package when upstream adds its own.
- `TestBackgroundDefaultsAreDerivedFromTheBackgroundColor` calls
  `themeDefaults` directly. `TestCurrentDiffRendererNameWithoutDiffRenderers`
  stubs delta's lookup and expects `default` without delta, `delta` with it.
- The shared fixture sets an explicit `rawGit` renderer so scenarios never use
  delta and disables fetch/file/ref polling.
- Four upstream scenarios were deleted: `cherry_pick/cherry_pick_across_worktrees`,
  `cherry_pick/cherry_pick_range_after_paste`, `commit/history` and
  `commit/history_complex`. Do not restore them without checking the fork's
  replacement behaviour for each.

Integration components gained `ExpectClipboard`, `ExplicitSelection`, `Views().Extras()`, a
menu `Content` matcher, `ReviewBranch*` helpers and `Git().SameCommitAs`. Drivers need `Title()` and
`Content()` on a confirmation and `Title()` on a menu before confirming.
