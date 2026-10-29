# Current work

## Item 25: stable scoped logs, branch / tag filter, flat `w` keys

Agreed behaviour:

- Scoped logs (`w a`, `w d`, `w b`) keep the order of the unfiltered log: they
  read the same `git log` (revisions, `--all`, order) and drop the commits
  outside the scope, so the filtered rows are a subsequence of the unfiltered
  rows. Ancestors are kept by propagation from the selected commits (a commit
  is kept when it is selected or a parent of a kept commit), which is exact
  because the walk lists children before parents. `git.log.order: default`
  walks scopes with `--date-order`. When the log shows only the current
  branch, a branch / tag scope adds its tips to the walk.
- `w b` is "Show branch / tag". In Commits it is offered when a local branch,
  remote branch or tag points at the commit; the refs are looked up when it
  runs. A local branch shows with its upstream; a remote branch or tag shows
  alone. One ref applies directly; several are chosen from a menu (local
  branches, remote branches, then tags). Branches, Remote branches and Tags
  offer it on the selected ref. The scope stores the full ref name and
  resolves it on every refresh. The command ID becomes `filter.ref`.
- `w o` loses its key. `w p` and `w u` open the path and author prompts,
  prefilled with the selected file's path (Files, Commit files) or the
  selected commit's author (Commits). `w c` clears the log filter and is
  offered only while one is active.

Acceptance: integration scenarios show a scoped ancestors log of two diverged
branches in the unfiltered relative order; `w b` on a remote branch tip, a tag
and a commit with several refs; `w p`, `w u` and `w c` from the `w` group.

Tasks:

- Flat `w p`, `w u`, `w c`.

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
