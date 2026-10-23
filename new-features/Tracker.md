# Current work

Roadmap item: [7. Selected-branch operations without checkout](Roadmap.md#7-selected-branch-operations-without-checkout).
Status: selected-branch sync, tracking metadata, reset, rebase and merge flows are
integrated, including captured local/remote deletion. Remaining work is the
branch/tag creation audit and its acceptance checks; see
[the inspected integration points](branch-menu-audit.md).

Scope: operate on the explicitly selected local branch wherever no worktree is
needed. Ref-only actions preserve current HEAD, index and worktree; explicitly
current-destination reset, rebase and merge retain their reviewed effects.
Preserve unrelated refs. Keep general off-branch merge/rebase/cherry-pick
deferred; never create another worktree or
switch branches silently. Push only the selected branch; wider-stack push is
deferred and must not be offered. Keep wider updates explicit choices.

## Remaining tasks for item 7

1. Finish captured targets in the remaining branch actions. Do not change
   shortcut policy ahead of item 8. Bind prompts, commands, result inspection and
   refresh completion to the original source, current destination and repository.
   Use the native source/current-HEAD snapshot and captured repository-generation
   guard; keep mutation-specific reservation and data-loss checks in each flow.

   - Audit branch/tag creation, retaining existing checkout/tracking choices,
     annotation/signing and overwrite warnings without later model or repository
     substitution. Native creation plans are implemented; finish the captured GUI
     flows and acceptance checks in [the creation design](branch-create-design.md).
     Keep pointer-only placement separate from named checkout.

   Preserve Git's normal concurrency model, errors and actual partial state;
   no implicit retry, rollback, checkout or stash. Reset/rebase/merge remain explicitly
   current-destination actions; do not invent off-branch worktree operations.
2. Verify remaining menus and preserve the sync acceptance baseline with local
   fake remotes: selected != current, absent/gone upstream,
   distinct push/pull remotes, force-with-lease rejection, fast-forward,
   divergence/rewritten upstream, concurrent ref changes, occupied worktrees
   and branch stacks. Assert HEAD/index/worktree and unrelated refs unchanged.
   Maintain usage, inventory and merge guide; run generation, formatting,
   build, short tests, relevant headless scenarios and lint.
3. When acceptance passes, complete item 7 accurately in the roadmap and
   replace this tracker with item 8's task-level plan in the same commit.

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
