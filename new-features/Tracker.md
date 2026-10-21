# Current work

Roadmap item: [2. Stable command identity and inventory](Roadmap.md#2-stable-command-identity-and-inventory).
Status: current-item task plan ready; inventory work begins next.

Scope: establish stable, non-translated action identity and a complete key/action
inventory. Preserve existing shortcuts, handlers, guards and modal behavior.
Expose unbound actions in the palette and carry identities through equivalent
invocation paths; do not implement usage ranking or remapping in this item.

## Remaining tasks for item 2

1. Inventory all binding and menu construction paths, hardcoded accelerators,
   modal/navigation keys, platform differences, custom commands and dispatch
   precedence. Record verb, target, availability and aliases with source
   references under `new-features`. Decide the narrow metadata/invocation seams
   from that evidence, consulting TortoiseGit for design questions. Present the
   concrete design before code changes; raise unresolved user choices.
2. Add explicit command-identity and semantic-verb metadata to bindings and
   relevant menu leaves, with validation/tests. Design IDs to survive locale,
   shortcut, selection and source-order changes. Add optional custom-command
   IDs and a documented stable fallback without persisting commands or secrets.
3. Annotate the inventory's built-in actions and aliases through bounded,
   reviewable changes. Share identity for equivalent invocation paths, not
   unrelated operations with the same translated label. Keep behavior ownership
   in existing controllers/helpers and record integration points in the merge
   guide. Check completeness mechanically where practical.
4. Connect palette deduplication and menu/binding identity propagation. Ensure
   commands with no shortcut remain discoverable and executable, with disabled
   reasons intact. Add focused tests for aliases, distinct equal-label actions,
   custom commands, unbound actions, locale/keymap stability and unchanged
   invocation behavior. Do not introduce frequency recording yet.
5. Review the inventory against the implemented metadata, regenerate affected
   docs/schema/test lists, and run formatting, build, short tests, lint and
   relevant headless scenarios. Record verification limits. Complete item 2 in
   the roadmap and replace this plan with item 3's task plan in the same final
   implementation commit, only after its acceptance criteria are satisfied.

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
