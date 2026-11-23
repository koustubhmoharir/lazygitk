# Current work: branch targets from selections

Agreed with the user on 2026-11-23.

- `n b` on a remote branch prefills the name prompt with the remote branch's
  name without its remote (`origin/feature` gives `feature`). The text stays
  editable; an existing local branch of that name gets the usual move warning.
- In Commits, with one commit selected, `p` and `g l` act on the local branches
  whose tip is that commit, not on the checked-out branch. One such branch is
  used directly, as if selected in Branches. Several open a menu to choose
  one: the checked-out branch first when it is among them, then the Branches
  pane's order. With no branch at the commit, or a range selected, the keys
  act on the checked-out branch as before. (TortoiseGit's log offers "Push
  <branch>" for a commit with branch labels.)

## Tasks

1. Commit-tip push and pull in Commits, with unit and integration tests.
2. Guide, design and merge-guide updates; close this section.

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
