# Current work

Roadmap item: [4. Create or move a branch at a selected commit, without checkout](Roadmap.md#4-create-or-move-a-branch-at-a-selected-commit-without-checkout).
Status: reviewing branch-ref primitives, worktree guards and concurrency semantics.

Scope: create or move a named local branch to a captured selected commit,
without checkout. Existing-tip moves require a warning; preserve HEAD, index,
working files and upstream metadata. Current-branch movement remains Reset's
responsibility. Never bypass another worktree's branch occupancy.

## Remaining tasks for item 4

1. Audit existing branch creation, validation, ref-update, worktree occupancy,
   branch-name suggestions and undo/reflog paths. Consult TortoiseGit and verify
   expected-old-OID updates, symbolic refs and occupancy behavior in disposable
   repositories. Record the additive design and raise unresolved choices.
2. Add backend preparation/application APIs for a validated literal local branch
   name and captured full commit OID. Distinguish create/no-op/move, preserve
   tracking metadata, reject prohibited occupied/current-branch moves, and
   recheck occupancy plus expected ref state immediately before writing.
3. Add a separate discoverable commit-list action and branch-name prompt, keeping
   create-and-checkout clearly distinct. Warn with old/new tips before a move;
   if state changes after confirmation, show the new state and require fresh
   confirmation. Refresh markers without changing checkout or selected commit.
4. Add unit, real-Git and headless regressions for create/move/no-op/cancel,
   invalid/conflicting names, dirty index/worktree, detached HEAD, occupied
   worktrees, concurrent ref changes, upstream retention and reflog recovery.
5. Update usage, inventory and merge guide; generate, build, format, test and lint.
   Record limits and avoid claiming undo support without a verified integration.
6. Once acceptance criteria pass, complete item 4 accurately in the roadmap and
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
