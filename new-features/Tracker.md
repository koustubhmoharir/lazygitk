# Current work

Roadmap item: [9. Frequency-ordered `?` palette](Roadmap.md#9-frequency-ordered--palette).
Status: item 8 is complete (full short suite, lint and all 723 integration
scenarios pass); this item has not started. Scope and validation
are in the roadmap. Counts are global per user, keyed by stable command IDs,
lifetime frequency, local only, with declaration order as the tie-breaker and
the displayed order frozen while the palette is open.

## Remaining tasks for item 9

1. Design the accepted-action recording seam and present it before
   implementing. Identify the single point where a direct key, prefix-menu leaf
   and palette execution of the same command converge, so each accepted action
   counts once under its leaf ID. Exclude typing, navigation repeats, opening
   `?`, prefix group menus, disabled/inapplicable actions and canceled
   confirmations; decide how flows with a confirmation or review record only on
   acceptance. Define the stable-ID policy for custom commands. Consult
   TortoiseGit for any comparable usage-ordered UI.
2. Design persistence and present the choice of app-state facility versus a
   dedicated state file: write frequency, batching, flush on normal exit, atomic
   writes, a lock or merge strategy across concurrent lazygit processes, and
   tolerance of corrupt or unwritable state. Store only IDs and counts.
3. Implement recording and persistence with unit tests for direct/menu/palette
   parity, aliases, cancellations, failures after acceptance, restart
   persistence, multiple processes and corrupt/unwritable state.
4. Sort the palette by count with deterministic ties, keeping Local/Global/
   Navigation as annotations or filters, context relevance, filtering and
   disabled items; keep order and selection stable while open. Add reset-usage
   and restore-static-ordering controls, with docs and merge-guide updates.
5. Verify with focused and terminal scenarios, generation, formatting, builds,
   full short and integration suites and lint. When item 9 acceptance passes,
   complete its roadmap description and update this tracker in that commit.

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
