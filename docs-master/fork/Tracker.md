# Current work: none

The fork review fixes are finished (see [reviews/TRIAGE.md](../../reviews/TRIAGE.md)).
Their decisions are recorded in the [user guide](Guide.md) and the
[design principles](Design.md). The full integration suite passed on
2026-11-07. `r s` (rebase onto the tracking branch) followed on the same day;
see the user guide.

## Open follow-ups (not started; need the user's go-ahead)

- Triage §3 quality items: 04-F1 (true-colour probe can eat keys typed during
  start-up), 04-F2 (graph cache key hashes the whole history per render),
  02-F3 (popup attribution across workers), 15-F4 (restore weakened graph
  assertions), 13-F3 (copy tests stop before real handlers), and the
  cherry-pick add/add conflict that the merge helper reports as a plain error.
- Remaining "01 merge notes" move: `focusPanelTitle` to a fork file, and
  optional extractions in `menu_context.go` and `menu_panel.go`.
- `docs-master/Custom_Command_Keybindings.md` still shows the ignored `key`
  setting in its tables, and the custom-command menu `key` comment was not
  updated.
- Reword in a path-filtered list takes the rebase base from the next filtered
  row, which with merges may not be an ancestor of the target (inherited).
- A completion callback registered by a cherry-pick or move-commits flow stays
  set if the operation is finished outside lazygitk (review 10, open question).

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
