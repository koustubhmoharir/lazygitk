# Current work: consistent Enter, Space and `v`

Enter, Space and `v` meant different things in different panes (Enter
committed in Files and did nothing elsewhere; Space staged, selected, opened a
menu or inspected; `v` was sometimes a key and sometimes a prefix). Agreed with
the user on 2026-11-25:

- **Enter goes in**: it opens a layer that Escape leaves (a list, a focused
  diff, conflict resolution, a submodule). Branches, Remote branches, Tags and
  Worktrees: commits; Remotes: branches; Commits, Sub-commits, Reflog and
  Stash: the changed files; Commit files: the file's diff or contents; Status:
  the main view; Submodules: enter. In Files, Enter focuses the diff (also on
  a directory), resolves a conflicted file and enters a submodule entry, as
  `v` did. Enter does nothing in the main views and the merge conflicts view.
  Switching worktree stays `s w`: it has nothing to return to.
- **Space toggles** and never navigates: staging in Files and the diff views,
  the selection in Commits, Sub-commits and Reflog (Sub-commits and Reflog
  gain Commits' selection model), the file in the custom patch in Commit files
  (as `g h t`), lines in the patch, conflict hunks. Elsewhere Space does
  nothing, including regular menus (upstream runs the item).
- **`v` chooses how to go in**, as a menu, only where there is more than one
  way; Enter is its first entry. Commits, Sub-commits, Reflog and Stash: `v d`
  changed files, `v f` all files. Files: on a conflicted file `r` resolve
  here, `m` merge tool, `d` show the diff; on a submodule entry `s` enter the
  submodule, `d` show the diff. Elsewhere `v` is unbound.
- **`c` commits or continues, in Files only.** With no operation in progress
  it commits (the message box, as Enter did). During a merge, cherry-pick,
  revert or rebase it continues, except at a rebase stop that is not a
  conflict (an edit stop, `rebase-merge/amend` present) with staged changes,
  where it commits: Git would refuse to continue after a reset, and would
  otherwise amend silently, which `a` offers explicitly. The options bar and
  `?` name the action `c` does now. `c p` has no route in Files; `c p` and
  `c c` stay as they are elsewhere.

Acceptance: each pane's keys behave as above, the options bar and `?` show
them, and the Guide, Design ("Shared roles") and merge guide say so. The full
integration suite passes.

## Tasks

1. Design and merge-guide updates, full integration suite; close this
   section.

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

`just` is unavailable; use the equivalent commands in `justfile`. Test times
depend heavily on what else the machine is running: `branch/delete` took 7.5s
to 30s alone. Use the `-timeout` values in `justfile`, because the unit tests can
exceed Go's default of 10 minutes per package. Clear
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
