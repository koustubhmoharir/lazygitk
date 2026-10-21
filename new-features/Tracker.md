# Current work

Roadmap item: [3. Separate diff actions for commits and historical files](Roadmap.md#3-separate-diff-actions-for-commits-and-historical-files).
Status: comparisons implemented; final scoped invocation tests and verification underway.

Scope: direct previous-commit and working-tree comparisons for commits and
historical files/directories, plus explicitly chosen worktree-file comparison.
Keep sticky diff mode independent, preserve captured selection and actual
working-file bytes, and use the configured external difftool with safe cleanup.

## Remaining tasks for item 3

1. Finish and review the historical materialization matrix for Unicode, empty,
   option-like/literal-magic paths and large binary blobs. Core backend,
   selection/capture, completion and exact directory/rename invocation tests pass.
2. Finish the last short-test/format/lint pass after that matrix; check generated
   artifacts for freshness and record limits. Debug build and lint passed;
   the 99-scenario regression run and final 15-scenario palette/diff run passed,
   including hint priority, staged-versus-working bytes, failure/recovery and cleanup.
3. Once acceptance criteria pass, complete item 3 accurately in the roadmap and
   replace this tracker with item 4's task-level plan in the same commit.

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
