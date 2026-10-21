# Current work

Roadmap item: [3. Separate diff actions for commits and historical files](Roadmap.md#3-separate-diff-actions-for-commits-and-historical-files).
Status: designing the diff backend and subprocess lifecycle.

Scope: direct previous-commit and working-tree comparisons for commits and
historical files/directories, plus explicitly chosen worktree-file comparison.
Keep sticky diff mode independent, preserve captured selection and actual
working-file bytes, and use the configured external difftool with safe cleanup.

## Remaining tasks for item 3

1. Finalize the additive backend/UI design against the recorded probes and
   TortoiseGit guidance. Resolve exact literal-path blob lookup, root/merge/
   rename/deletion handling, trusted no-index exit status and temporary-file
   lifetime before delegating bounded implementation. Raise unresolved choices.
2. Implement and test endpoint/selection validation and exact historical blob
   extraction. Stream binary bytes into private read-only temporary sources;
   validate existing regular-file destinations inside the current worktree.
3. Add the narrow subprocess success-status integration needed for trusted
   no-index diffs; Git 0/1 are normal only for this action. Preserve other
   subprocess errors/prompts and ensure cleanup on every completion path.
4. Wire separate commit/file previous and worktree actions with stable IDs,
   accurate labels/disabled reasons, captured OIDs/paths and priority hints.
   Add arbitrary-file prompt with completion and explicit source/destination.
   Keep old sticky comparison distinct. Introduce only the agreed provisional
   case-insensitive diff keys, resolving collisions without a full remapping.
5. Add fake-tool backend and headless UI coverage for endpoints, exact bytes,
   unusual paths, renames/deletions/root/merge, missing/unsupported destinations,
   cancellation/failure, sticky isolation and case variants. Update user docs,
   inventory and merge guide; generate/build/test/lint and record limits.
6. Once acceptance criteria pass, complete item 3 accurately in the roadmap and
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
