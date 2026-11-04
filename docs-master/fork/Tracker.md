# Current work

Conflict flow and external tools. Agreed with the user on 2026-11-04.

## Agreed behaviour

- **`w v` on a detached HEAD.** Shows HEAD's commit in Commits, like any other
  Branches entry. This includes the entry shown during a rebase.
- **A paused sequence is in progress.** After a pick or revert of a sequence is
  committed, Git removes `CHERRY_PICK_HEAD` / `REVERT_HEAD` but keeps
  `sequencer/`. lazygitk treats this as the cherry-pick or revert still being in
  progress, as `git status` does. The commit already made is not listed as a
  todo. Skip is crossed out with the reason, because Git has nothing to skip.
- **`c c` continues.** Continues the cherry-pick, rebase, merge or revert in
  progress, from any pane, committing the resolved pick if it is not committed
  yet. It refuses and names the files while any are still unmerged, including
  files whose markers are gone but which are not staged. The status line shows
  it with the operation's name while an operation is in progress. `g o c` stays.
- **Cherry-pick marks are dropped.** `c y`, `c x` and the source menu go. `c p`
  acts on the selection, which is cleared after a successful pick and kept when
  it stops on conflicts.
- **`d m` opens the merge tool.** In Files and the conflicts view, for the
  selected conflicted files, including delete/modify conflicts. No `.orig`
  backups unless `mergetool.keepBackup` is set. The item in `g h c` does the
  same.
- **No guessing of external tools.** With no `diff.tool` or `merge.tool` for
  the difftool, or no `merge.tool` for the merge tool, a menu lists the tools Git
  reports as available. The chosen tool is passed to Git for this launch, with
  the option to save it to Git's global config. Cancel launches nothing. Git's
  "not configured" message no longer appears.
- **Up and Down recall only a cancelled draft.** In the commit message box
  (commit, reword, combine, tag), they no longer step through earlier commits'
  messages. Escape saves the text as the worktree's one draft if it is not empty
  and differs from what the box opened with, and the draft survives a restart.
  In the summary line, Up shows the draft and keeps the text it replaces; Down
  returns to that text, and edits made to the draft are kept for the next Up.
  Up does nothing with no draft or when the draft is already shown. Submitting
  the draft's text clears the draft; submitting anything else keeps it. A new
  commit's box still reopens with its cancelled text. The description box still
  moves its cursor.

## Remaining tasks

- Drop the marks.
- External tool chooser for every difftool launch.
- `d m` and the merge tool chooser.
- Replace message history with draft recall in the commit message box.

## How this tracker works

- This file holds the current piece of work: its agreed behaviour, acceptance
  criteria and remaining tasks. New work starts by writing that section here
  and reviewing it with the user. If many items are planned at once, create a
  separate roadmap for them at that point.
- Plan tasks for the current work only. Work through them in reviewable
  increments; a task may span commits. Remove each task in the commit that
  completes it, and do not keep a completed-task log: Git history records it.
- When the work is finished, move its lasting decisions into the
  [user guide](Guide.md) (keys and behaviour) and the
  [design principles](Design.md) (rules), then replace its section here with
  the next work, or say explicitly that there is none. Do not leave a stale
  section.
- Keep the tracker accurate when tasks change, record concrete blockers against
  the affected task, and present unresolved user choices in a modal flow with
  sensible defaults. Do not mark unfinished work complete.

## Verification notes

`just` is unavailable; use the equivalent commands in `justfile`. Clear
`NO_COLOR` for color-sensitive tests. Headless invocations share
`/tmp/lazygit/test_lazygit` and fixture directories: serialize their builds and
runs. Use bounded eventual checks for background events, not immediate UI
assertions or synthetic refreshes. Use verbose named terminal runs and verify
the expected scenario names actually ran; a successful exit with "no tests to
run" is not coverage. Test names are `TestIntegration/<group>/<scenario>`.
Finish generation successfully before starting headless checks, and keep
integration-test sources frozen during the run. Each scenario variable must be
its filename converted from snake_case to PascalCase, using unqualified
`NewIntegrationTest`; compile-only checks do not validate those generator rules.
The shared integration config explicitly disables fetch/file/ref polling;
default-policy tests must restore values from the real default constructor.
