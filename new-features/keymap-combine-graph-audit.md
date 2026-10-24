# Combine and graph keymap source audit

Source evidence for the approved keybinding implementation. This records
existing behavior and integration seams; it does not claim the requested new
commands already exist.

## Combine selected commits

- `pkg/gui/controllers/local_commits_controller.go`,
  `LocalCommitsController.GetKeybindings` registers `commit.squash-down` and
  `commit.mark-fixup` for selected ranges. Their handlers are `squashDown` and
  `fixup`; both feed interactive-rebase TODO changes, not an operation that
  combines an selected commit set using the earliest selected commit as
  keeper.
- The same controller's `squashFixupCommits` presents separate current-branch
  and above-selection autosquash choices. `squashFixupsImpl` delegates to
  `Rebase.SquashAllAboveFixupCommits` and compensates selection for removed
  fixups. This is specifically autosquash of `fixup!` commits; it does not
  establish the requested combine-selected flow.
- `pkg/commands/git_commands/commit.go`, `CreateFixupCommit`, builds
  `git commit --fixup=<hash>`. `pkg/commands/git_commands/rebase.go`,
  `PrepareInteractiveRebaseCommand`, runs interactive rebase with
  `GIT_SEQUENCE_EDITOR`; `AmendTo`, `MoveFixupCommitDown`, and `EditRebaseTodo`
  create/move or edit fixup TODO entries. `pkg/utils/rebase_todo.go` contains
  TODO editing helpers. These are the Git operation seams.
- Message controls are separate: `LocalCommitsController.setFixupMessage`
  offers discard (`fixup`) and keep (`fixup -C`) choices. Commit rewording
  obtains the existing message and switches through the commit message context
  and editor. The shared message panel stores summary and description
  separately (`CommitsHelper.SplitCommitMessageAndDescription`,
  `JoinCommitMessageAndUnwrappedDescription`, and
  `CommitMessageContext.OnConfirm`); `CommitMessageContext.SwitchToEditor`
  exposes the full subject-plus-body editor route. A combine flow can reuse this
  seam while retaining all selected messages in ancestor-first order; no
  existing handler gathers messages for an selected commit set.
- The displayed commit sequence is supplied by `CommitLoader.GetCommits` and
  `CommitLoader.getLogCmd` in `pkg/commands/git_commands/commit_loader.go`;
  ordering comes from `git.log.order` and the configured `git log` order. The
  commit context's selection range is index based. The existing squash/fixup
  paths operate on that ordered list/TODO sequence. `getLogCmd` uses configured
  `git.log.order` (otherwise Git's default ordering); a future combine-message
  builder must explicitly produce ancestor-first order. The earliest-ancestor
  keeper semantics for a selected commit set are not implemented by those
  paths.

## Cherry-pick mode cancellation

- `pkg/gui/controllers/helpers/cherry_pick_helper.go`, `CherryPickHelper.Reset`,
  clears `ContextKey` and copied commits, then rerenders. It does not call Git.
  Git cherry-pick is only started by `CherryPickHelper.Paste`; conflict/abort
  state is handled separately by merge/rebase helpers.
- `pkg/gui/controllers/quit_actions.go`, `QuitActions.Escape`, first cancels
  list range selection, then exits filtering, then returns to a logical parent
  context. Only if none applies does it scan `Mode.Statuses()` and reset the
  first active mode. In `pkg/gui/controllers/helpers/mode_helper.go`,
  `ModeHelper.Statuses` orders filtering, marked-base, cherry-picking, working
  tree operation state, and bisect. Thus Escape resets cherry-pick marks after
  range/filter/parent handling, and only if no earlier status (notably
  marked-base) is active. `EscapeEnabled` and `EscapeDescription` mirror this
  order and expose the selected action's cancel label. The information-panel
  reset action also resets the first active mode; Escape is the general reset
  route described above.
- Menus bind Escape in `pkg/gui/controllers/menu_controller.go`; active search
  escape is handled by gocui through `OnSearchEscape` in `pkg/gui/gui.go`.
  `QuitActions.Escape` explicitly handles range selection and filtering but
  does not enumerate menu/popup state. Popup/menu dispatch relative to this
  handler is not established by these functions alone.

## Whole graph versus graph rendering

- Pre-change baseline: `pkg/config/user_config.go`,
  `GitLogConfig.ShowWholeGraph`, documents the option as controlling whether
  `git log --all` loads the whole graph; the user-config constructor sets it to
  `false`. The fork overlay now overrides it to `true` before loading user or
  repository config. This leaves the upstream constructor untouched and retains
  explicit false overrides; do not confuse the two default-table layers.
- `pkg/gui/context/local_commits_context.go`,
  `NewLocalCommitsViewModel`, initializes `showWholeGitGraph` from that
  configuration value. `SetShowWholeGitGraph` / `GetShowWholeGitGraph` hold
  the live value in the local-commits view model. `RefreshHelper.captureCommitsState`
  captures it before worker refresh, and `refreshCommitsWithLimit` passes it as
  `GetCommitsOptions.All`.
- `pkg/commands/git_commands/commit_loader.go`, `CommitLoader.getLogCmd`, adds
  `--all` only when `opts.All` is true. This is the all-ref query seam; it is
  distinct from drawing graph characters.
- `LocalCommitsContext.shouldShowGraph` in
  `pkg/gui/context/local_commits_context.go` reads `git.log.showGraph` and
  chooses whether the renderer draws the graph (`always`, `never`, or
  `when-maximised`). In particular, `always` means draw graph rendering; it
  does not set `ShowWholeGraph` or cause `git log --all`.
- The source establishes the default and live refresh path. It does not by
  itself establish persistence of a changed live `showWholeGitGraph` value
  across context recreation/reload. That lifecycle remains an implementation
  seam to inspect before promising persistence.

## Evidence limits

Escape's mode-reset precedence is explicit in `QuitActions`; popup/menu
dispatch relative to that handler is outside the inspected function. The live
full-graph toggle's persistence across context recreation/reload is not
established by the captured refresh path. Graph details above are pre-change
baseline evidence; the main implementation work owns the default change.

## Confirmed requirements and delivery boundary

The combined description preserves every selected full message (subject and
body), including the first, in ancestor-first order; the summary starts with
the first subject. This is a selected-commit operation, independent of the
cherry-pick mark buffer. The combine operation is implemented for linear history and discoverable in
the palette; the normal `g c` replacement for squash/fixup awaits full-profile
activation. The user confirmed non-adjacent
selections with intervening commits preserved, refusing merges in the rewritten
ancestry. Merges before the first selected commit need not be rewritten.

The independent graph default is implemented in `pkg/config/fork_defaults.go`.
The shared terminal harness explicitly retains branch-only history for existing
scenarios; `commit/full_graph_default` restores the real default and checks local
branch, remote-only and current-branch commits together. Default changes do not
promise new persistence for the existing live view toggle.

## TortoiseGit guide

The [log dialog guide](https://tortoisegit.org/docs/tortoisegit/tgit-dug-showlog.html)
describes combining continuous commits. The [rebase guide](https://tortoisegit.org/docs/tortoisegit/tgit-dug-rebase.html)
describes ancestor-to-descendant execution order and merge-preservation limits.
These guide the review; neither establishes arbitrary graph-selection support
for the fork. The requested selected-ancestor keeper and preservation of
unselected history require an explicit topology-aware plan.

## Implemented linear planning core

`git_commands/combine_plan.go` now validates a captured first-parent chain,
canonicalizes full object IDs, finds the selected ancestor independently of UI
order, and refuses merges in the rewritten suffix. Its TODO puts every selected
fixup directly after the keeper before replaying unselected commits, so an
unselected intervening commit can never accidentally absorb a fixup. A merge
before the keeper stays outside the rewritten range. The editor seed preserves
all selected full messages, including blank lines and the keeper's own message.

The pure planner is now complemented by `captured_combine_plan.go`,
`combine_command.go`, `daemon/combine_rebase.go` and the editable controller
flow in `fork_combine.go`. Capture reads native ancestry and exact object
messages, refuses unavailable shallow parents and checks clean tracked state
and ignored-file approval before one native interactive rebase.

The sequence editor puts the approved message in Git-owned rebase state and
compiles the keeper message action to a native exec amendment. Native reword
can bypass its editor after a pre-commit hook failure; the exec route with
`--reschedule-failed-exec` preserves the approved message for explicit continue.
Its constant command resolves the worktree's state path without embedding user
text or repository paths. Original temporary payloads can be removed when the
initial command returns, even after a pause. Native success/abort removes the
Git-owned message file.

Real-Git checks cover root/nonroot, non-adjacent selections, large/raw messages,
hook failure, selected fixup and unselected pick conflicts, abort, shallow
boundaries, stale/foreign captures, ignored fingerprints and normal dirty-file
protection. The terminal GUI scenario covers all-message editing, cancel,
dirty-tree draft recovery, explicit fresh-HEAD review, preserved external empty
commits and unrelated refs. Terminal signing reuses existing routing; focused
checks exercise cleanup/error handling, not an actual signing device. Final
normal-profile activation and its `g c` regression checks remain. No implicit
stash, application retry, transaction lock or rollback is introduced.
