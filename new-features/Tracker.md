# Current work

Roadmap item: [6. Rebase the current branch onto a selected commit](Roadmap.md#6-rebase-the-current-branch-onto-a-selected-commit).
Status: backend and shared clean-start policy implemented; UI integration and
the ignored-file collision policy remain open.

Scope: rebase the checked-out branch onto a captured selected commit, including
other-branch history/reflog. Require no staged/unstaged tracked changes; allow
harmless untracked files and let Git reject collisions. Never invoke or offer
autostash, and do not rewrite additional branch refs. Keep conflict
continue/skip/abort usable. Audit other rebase-start paths for the same policy.

## Remaining tasks for item 6

1. Finish the signing/recovery integration design and resolve the ignored-file
   safety choice. Native Git can overwrite an ignored target-path collision;
   the user has been asked whether to warn and require explicit approval or
   retain native behavior. The design/probe documents record the audit and
   settled harmless-untracked-file policy.
2. Add a discoverable shared commit-list action and modal warning naming the
   checked-out source branch/tip and captured target. Preserve repository/source
   identity, require fresh review if pre-launch state changes, and reuse normal
   conflict recovery and refresh without silent target/base substitution.
3. Complete regression coverage around the UI: other history/reflog, captured
   source/repository, fresh review, cancellation, marked-base isolation,
   signing routing, ignored-file policy and conflict continue/abort. Expand
   affected headless coverage beyond the four start-policy scenarios already
   verified. Backend real-Git tests cover target graphs, identity, topology,
   clean/untracked states, SHA-256 and auxiliary-ref preservation.
4. Update usage, inventory and merge guide; generate, debug-build, format, run
   full short tests, affected headless regressions and lint. Record limits.
5. Once acceptance criteria pass, complete item 6 accurately in the roadmap and
   replace this tracker with item 7's task-level plan in the same commit.

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
