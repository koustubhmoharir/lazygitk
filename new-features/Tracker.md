# Current work

Roadmap item: [2. Stable command identity and inventory](Roadmap.md#2-stable-command-identity-and-inventory).
Status: metadata, inventory and palette integration implemented; final verification underway.

Scope: establish stable, non-translated action identity and a complete key/action
inventory. Preserve existing shortcuts, handlers, guards and modal behavior.
Expose unbound actions in the palette and carry identities through equivalent
invocation paths; do not implement usage ranking or remapping in this item.

## Remaining tasks for item 2

1. Finish the final build/short-test/lint pass after inventory and palette review.
   Generation, standalone inventory/checker tests and 14 targeted headless
   scenarios passed, including executing unbound branch rename via the palette.
   Record verification limits. Complete item 2 in
   the roadmap and replace this plan with item 3's task plan in the same final
   implementation commit, only after its acceptance criteria are satisfied.

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
The shared integration config explicitly disables fetch/file/ref polling;
default-policy tests must restore values from the real default constructor.
