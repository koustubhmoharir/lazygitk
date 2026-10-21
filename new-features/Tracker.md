# Current work

Roadmap item: [5. Checkout any commit under a supplied branch name](Roadmap.md#5-checkout-any-commit-under-a-supplied-branch-name).
Status: awaiting the user's checkout-concurrency choice after completed Git
probes. Lock-retry opt-out and shared pure OID validation are committed.

Scope: check out a captured selected commit under a literal new or existing
local branch name. Warn before moving an existing name; use Git's combined
operation, carry dirty changes when safe, preserve ignored collisions, and
never force files, require an unnecessarily clean tree, or offer stash.

## Remaining tasks for item 5

Design question: [named-checkout-design.md](named-checkout-design.md) records
the verified late-ref/alias race. Choose Git's native concurrency contract
(coordinate other writers during checkout) or investigate stronger in-command
protection before implementing the backend. No checkout mutation is implemented.

1. Audit checkout-menu construction, branch validation, operation guards,
   autostash offers, refresh and worker ownership. Consult TortoiseGit and probe
   `checkout --no-overwrite-ignore -B` with genuinely different trees, dirty
   index/files, current/other-worktree destinations, hooks and concurrent ref
   changes. Record the additive design and resolve substantive ambiguities.
2. Add backend preparation/application APIs reusing literal-name/full-OID/ref
   validation where appropriate. Permit a safe current-branch destination,
   reject reserved/in-progress states, revalidate before the combined checkout,
   preserve tracking settings and report Git failures without stash or retry.
   Use the verified command-local lock-retry opt-out, not the default retrying
   runner path after a possibly partial checkout.
   The shared pure full-OID guard is now available to both controller packages.
3. Add **Checkout as branch…** to the shared checkout menu. Capture repository
   and source identity, prompt for the destination, warn with old/new tips and
   require fresh review of changed destinations. Refresh all checkout-related
   models on actual completion, including uncertain hook/process failures.
4. Add backend/controller/headless regressions for new/existing/current names,
   detached HEAD, differing trees, carried dirty changes, overwritten tracked/
   staged/untracked/ignored paths, cancellation, concurrent changes, occupied
   worktrees and failure recovery. Assert no partial ref move on rejection.
5. Update usage, inventory and merge guide; generate, debug-build, format, run
   full short tests, relevant headless regressions and lint. Record any limits.
6. Once acceptance criteria pass, complete item 5 accurately in the roadmap and
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
The shared integration config explicitly disables fetch/file/ref polling;
default-policy tests must restore values from the real default constructor.
