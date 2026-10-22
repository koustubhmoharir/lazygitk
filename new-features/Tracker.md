# Current work

Roadmap item: [7. Selected-branch operations without checkout](Roadmap.md#7-selected-branch-operations-without-checkout).
Status: source/remote inspection, guarded ref-only advancement and explicit
push/fetch backend plans (including current-branch fetch) are implemented.
Additive push-dialog/target helpers are implemented and directly tested.
The additive off-branch update dialog is implemented and directly tested.
Default binding integration, explicit stack choices and the remaining
end-to-end selected-branch workflows remain.

Scope: operate on the explicitly selected local branch wherever no worktree is
needed. Preserve current HEAD, index, worktree and unrelated refs. Keep general
off-branch merge/rebase/cherry-pick deferred; never create another worktree or
switch branches silently. Keep stack-wide operations explicit choices.

## Remaining tasks for item 7

1. Integrate the captured branch/repository target into source/destination
   review and routing, using the backend's fresh approved-input validation and
   transport/config change detection. Include custom tracking namespaces,
   local upstreams and explicit/excluded automatic-lease choices.
   Every prompt, command, upstream-setting action and refresh must retain that
   target; neither a later selection nor current HEAD may substitute for it.
2. Make push target the selected local branch with an explicit remote/refspec.
   The branch-only backend now handles explicit destinations, immutable
   multi-URL approval, reviewed leases and upstream-setting snapshots.
   Finish modal routing and protection/no-upstream/triangular-workflow
   verification. Keep pushing other stack branches an explicit separate choice.
   The one-branch dialog and upstream-refusal fallback are implemented;
   integrate them without losing the explicit wider stack workflow.
   Agreed upstream policy: refuse ambiguous changes before pushing; offer
   Cancel or an explicitly reviewed push without changing tracking. Do not
   repair or replace multi-valued/inherited upstream settings automatically.
3. Make off-branch pull/update an explicit upstream fetch and verified
   fast-forward without checkout. The scoped fetch and guarded ref-only update
   builders are implemented; integrate their modal flow and explain divergence
   without attempting integration. Reject tip changes to branches occupied by
   another worktree; do not reuse paths that mutate that worktree. Integrate the
   current-branch fetch-first flow: review the exact fetched commit and
   integration choices (defaulting from Git configuration), check ignored-file
   risks, and integrate only that captured target without another network pull.
   Retain clean/no-autostash behavior. Integrate local `.` upstreams through
   direct off-branch advancement or captured current integration, as appropriate.
   The earlier one-step current-pull primitive is not the approved UI path.
   Current-branch fetch and exact observed-tip inspection are implemented;
   the reviewed native merge/rebase backend is implemented, including
   pre-fetch fork points and ignored-file approval. Finish modal choices and
   routing without dropping those reviewed backend guarantees.
4. Integrate branch-bearing versus global/current-branch actions, retaining
   selected-ref metadata support and clearly named selected-source merge/
   rebase actions. Do not change shortcut policy ahead of item 8.
5. Verify with local fake remotes: selected != current, absent/gone upstream,
   distinct push/pull remotes, force-with-lease rejection, fast-forward,
   divergence/rewritten upstream, concurrent ref changes, occupied worktrees
   and branch stacks. Assert HEAD/index/worktree and unrelated refs unchanged.
   Maintain usage, inventory and merge guide; run generation, formatting,
   build, short tests, relevant headless scenarios and lint.
6. When acceptance passes, complete item 7 accurately in the roadmap and
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
