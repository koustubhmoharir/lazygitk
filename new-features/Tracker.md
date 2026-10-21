# Current work

Roadmap item: [4. Create or move a branch at a selected commit, without checkout](Roadmap.md#4-create-or-move-a-branch-at-a-selected-commit-without-checkout).
Status: implementation and headless regressions verified; finishing final checks.

Scope: create or move a named local branch to a captured selected commit,
without checkout. Existing-tip moves require a warning; preserve HEAD, index,
working files and upstream metadata. Current-branch movement remains Reset's
responsibility. Never bypass another worktree's branch occupancy.

## Remaining tasks for item 4

1. Finish final full-short-test and lint reruns after the generated test list is
   stable. Usage, inventory, generation, debug build, formatting, targeted race
   tests and 127 branch/commit/reflog headless scenarios have passed. Record
   verification limits; do not claim Undo support or a new full headless run.
2. Once acceptance criteria pass, complete item 4 accurately in the roadmap and
   replace this tracker with item 5's task-level plan in the same commit.

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
