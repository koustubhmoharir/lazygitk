# Current work

Roadmap item: [5. Checkout any commit under a supplied branch name](Roadmap.md#5-checkout-any-commit-under-a-supplied-branch-name).
Status: backend and modal flow verified; 170 affected headless scenarios and
targeted race checks pass. Finishing sparse-checkout/submodule acceptance cases
and the final documentation/checkpoint.

Scope: check out a captured selected commit under a literal new or existing
local branch name. Warn before moving an existing name; use Git's combined
operation, carry dirty changes when safe, preserve ignored collisions, and
never force files, require an unnecessarily clean tree, or offer stash.

## Remaining tasks for item 5

Agreed contract: [named-checkout-design.md](named-checkout-design.md) records
the verified late-ref/alias race and the user's choice of Git's native
concurrency model. Recheck before launch; other writers must be coordinated
while checkout runs. Do not promise atomic expected-old updates for checkout.

1. Finish sparse-checkout/submodule compatibility tests against Git's native
   behavior, including dirty nested-file protection. The remaining backend,
   modal and affected headless regressions have passed, including failed-hook
   state refresh, no retries, retained failure selection and fresh confirmation.
2. Update usage, inventory and merge guide; generate, debug-build, format, run
   full short tests, relevant headless regressions and lint. Record any limits.
3. Once acceptance criteria pass, complete item 5 accurately in the roadmap and
   replace this tracker with item 6's task-level plan in the same commit.

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
Finish generation successfully before starting headless checks, and keep
integration-test sources frozen during the run. Each scenario variable must be
its filename converted from snake_case to PascalCase, using unqualified
`NewIntegrationTest`; compile-only checks do not validate those generator rules.
The shared integration config explicitly disables fetch/file/ref polling;
default-policy tests must restore values from the real default constructor.
