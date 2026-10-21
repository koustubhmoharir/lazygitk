# Current work

Roadmap item: [1. Explicit-action defaults](Roadmap.md#1-explicit-action-defaults).
Status: config/runtime audit complete; implementation and verification underway.

Scope: default `autoFetch`, `autoRefresh`, and `autoStageResolvedConflicts` to
`false`, and `autoForwardBranches` to `none`. Keep read-only external-ref
polling enabled and honor explicit user/repository configuration. Manual
operations and refreshes following explicit actions remain available.

## Remaining tasks for item 1

1. Finish verification of the timer-driven default and opt-in scenarios: no
   automatic fetch or file polling under defaults, explicit fetch/refresh
   opt-ins still work, and read-only external-ref polling remains active.
   Keep all Git mutations in disposable repositories with local fake remotes.
2. Finish final validation, including lint and the full headless integration
   run; check generated outputs remain current after any fixes. Record results
   and any verification limits
   in the roadmap/merge guide. Once acceptance criteria are satisfied, mark item
   1 complete in the roadmap, reconcile its description with actual behavior,
   and replace this task list with the newly prepared plan for item 2 in the
   same commit. Do not advance while a required check or scope issue is unresolved.

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

The initial environment lacks `just`; install it or use equivalent recipe
commands and report that choice. Color-sensitive tests require clearing
`NO_COLOR`. A previous lint attempt could not fetch the pinned tool because
`sum.golang.org:443` refused the connection; resolve that before claiming item
1's required verification is complete. These notes are environment observations,
not completed feature tasks.
