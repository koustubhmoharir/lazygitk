# Current work: browse a commit's files

Agreed with the user on 2026-11-23. TortoiseGit's equivalent is the
Repository Browser at a revision (open, compare with working tree, show log).

## Behaviour

- In Commits, Reflog and Sub-commits, `v` opens a menu: `d` changed files (the
  Commit files pane as today) and `f` all files. Outside Commits, Space does
  the same as `v`. Stash keeps `v` going straight to its changed files. `v`
  in Reflog currently does nothing (no controller is attached); this work
  makes it work.
- `f` needs a single commit; with a range selected it is crossed out.
- The full listing shows every file in the commit's tree. Files the commit
  changed keep their status letter and colour, and files it deleted are listed
  as D, so it is a superset of `d`. Unchanged files show a blank status column,
  aligned with the others. Its title says it lists all files.
- Each entry into the full listing starts with folders collapsed (top-level
  entries visible); flat view still lists every file. Collapsing in the full
  listing does not change the changed-files listing's folders.
- In the full listing, a selected file shows its contents at the commit in the
  main view, not a diff. A file the commit deleted shows its contents before
  the commit, and the title says so. A binary file shows a one-line note
  instead of its bytes. A directory or submodule shows the commit's changes
  under it, as in `d`.
- Commands in the full listing: `d w`, `d c`, `g v f`, `y c`, `e`, `o` and
  `w p` work on any file. `d p`, `t d e`, `x` and the custom-patch toggles need
  a file the commit changed, and are crossed out on an unchanged file with the
  reason. On a directory they act only on the changed files under it.
- Filtering Commits to the selected file's history is `w p`, prefilled with
  the file.

## Tasks

1. Commands in the full listing on unchanged files and directories, with
   tests.
2. Finish the guide, design and merge-guide updates; close this section.

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
