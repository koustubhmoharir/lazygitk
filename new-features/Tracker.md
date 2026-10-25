# Current work

Roadmap item: [9. Frequency-ordered `?` palette](Roadmap.md#9-frequency-ordered--palette).
Status: design agreed with the user; implementation in progress.

Agreed design (user decisions 2026-10-25, refining the roadmap):

- Scoring: a direct shortcut or prefix-menu leaf subtracts 1 (the user has
  learned it); palette execution adds 1. Counts may go negative. Sort by count
  descending, declaration order breaking ties; order is fixed while open.
- Recording seam: `callKeybindingHandler` receives the source. The use is
  pending while the handler runs; a confirmation, prompt or menu opened through
  `PopupHandler` during the handler commits it on acceptance and drops it on
  cancel; otherwise it commits immediately. Upstream merge ease outweighs exact
  acceptance tracking, so popups opened later from workers are not tracked.
  Prefix group menus, disabled/inapplicable actions, navigation commands and
  typing are not counted. Custom commands use `CommandIdentity`.
- Storage: dedicated `command_usage.json` beside `state.yml`, IDs and counts
  only. In-memory deltas flush (batched ~2 s, and on exit) under a lock file:
  re-read, add deltas, write a temp file, rename, release. A corrupt file is
  dropped with a warning that counts were reset; write failures are logged.
- Palette: local and global commands merge into one usage-sorted list with a
  dim local/global tag; Navigation stays a static section at the bottom. Add a
  confirmed reset-counts command and a config option for static ordering.

## Remaining tasks for item 9

1. Palette ordering, tags, Navigation section, reset command, static-order
   option, schema/docs and merge-guide updates.
2. Terminal scenarios (ordering after palette and shortcut use, cancel,
   restart persistence, corrupt file warning), then generation, formatting,
   builds, full short and integration suites and lint. When acceptance passes,
   complete item 9 in the roadmap and update this tracker in that commit.

## How this tracker works

- [Roadmap.md](Roadmap.md) owns the item sequence, agreed scope, acceptance
  criteria and item completion status. This file owns only the current item
  and its remaining tasks. The merge guide owns upstream integration knowledge.
- Create the task-level plan when an item becomes current, not earlier. Do
  not add task breakdowns for future items here or elsewhere. Roadmap design
  notes and acceptance criteria are not advance task lists.
- Work through the current tasks in reviewable increments. A task may span
  commits. Keep it here until its stated outcome and relevant checks are done;
  narrow its wording if part of the work remains.
- Remove each completed task from this file in the same commit that completes
  it. Do not retain checked-off tasks or a completed-task log; Git history
  records them. Include merge-guide updates with changes to integration seams.
- When an entire item is finished, use that same completion commit to mark it
  complete in the roadmap, update its description for any implementation
  differences, and replace this file's current-item scope and remaining tasks
  with a task-level plan for the next roadmap item. Do not leave a stale or
  empty tracker between items. If all items are complete, say so explicitly.
- Keep the tracker accurate when tasks change, record concrete blockers against
  the affected task, and present unresolved user choices in a modal flow with
  sensible defaults. Do not mark unfinished work complete to advance the queue.

## Verification notes

`just` is unavailable; use the equivalent commands in `justfile`. Clear
`NO_COLOR` for color-sensitive tests. The pinned lint shim now downloads and
runs successfully. Headless invocations share `/tmp/lazygit/test_lazygit` and
fixture directories: serialize their builds/runs. Use bounded eventual checks
for background events, not immediate UI assertions or synthetic refreshes.
Use verbose named terminal runs and verify the expected scenario names actually
ran; a successful exit with "no tests to run" is not coverage. Test names are
`TestIntegration/<group>/<scenario>` (there is no literal `group` segment).
Finish generation successfully before starting headless checks, and keep
integration-test sources frozen during the run. Each scenario variable must be
its filename converted from snake_case to PascalCase, using unqualified
`NewIntegrationTest`; compile-only checks do not validate those generator rules.
The shared integration config explicitly disables fetch/file/ref polling;
default-policy tests must restore values from the real default constructor.
