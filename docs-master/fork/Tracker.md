# Current work

## Repo switch feedback

Agreed: on a repo switch every pane clears at once and shows "Loading…" until
its own data arrives, so old content never appears under the new repo; a
toast "Opened <repo>" appears once all panes have loaded. TortoiseGit has no
in-app repo switch to follow.

Design: when a switch starts, every side pane and the main area show
"Loading…", drawn at once (`ForceFlushViewsContentOnly`) before the
synchronous switch work, so keys queue as before. If the switch fails, the
current repo's panes are rendered again. The panes fill in the existing
batched frame; the toast runs from `RefreshOptions.ThenIfRepoUnchanged`, so a
superseded switch shows none. Startup is not a switch. Known gaps: a lazy
list viewport re-render or a focus-in refresh may fill a pane early, always
with the opened repo's data.

Tasks:

- Show "Loading…" at switch start, restore on failure, toast on completion;
  integration test switching to a new and to a visited repo.

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
