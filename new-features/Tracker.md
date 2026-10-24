# Current work

Roadmap item: [8. Consistent default keys and no-op behavior](Roadmap.md#8-consistent-default-keys-and-no-op-behavior).
Status: the default registration audit and source-reviewed mode matrix are
recorded in [keymap/README.md](keymap/README.md),
[keymap-mode-matrix.md](keymap-mode-matrix.md) and the linked dispatch/default
audits. Complete the command/path allocation before changing application
defaults. [keymap-design.md](keymap-design.md) is a draft; Space's proposed
staging role is still a pending user preference.

Scope: one command meaning per default key/path across panes. ASCII command
letters are case-insensitive; inapplicable actions consume their key without
side effects or global fallback. Reserve Ctrl defaults for custom commands,
while preserving normal text entry and terminal Enter/Tab byte aliases. Use
visible accelerator menus for two-/three-letter paths. Keep working-tree and
previous-version diff separate and directly accessible; Pull has no single-letter
shortcut. Preserve explicit overrides and custom commands. Frequency persistence
and sorting remain item 9.

## Remaining tasks for item 8

2. Review [keymap-allocation.md](keymap-allocation.md) against each actual
   handler, context and legacy configuration field before enabling defaults.
   The TortoiseGit guide and concrete verb/prefix table are recorded; the table
   names every instantiated operation ID and the known dynamic variants.
   Resolve Space's remaining role with the user. Preserve direct `w`/`v`
   comparisons and Pull's deliberate menu path. Define source-field override
   projection for commands with several legacy aliases or context-specific fields.
3. Integrate the opt-in ASCII command matching/collision normalization and add
   inactive-key consumption at small central integration points. The command
   matcher is implemented independently of exact keys and input events; the
   loader records explicit YAML field presence separately from values/legacy
   alias folding. The application still uses its existing registration path.
   Preserve literal input, search/filter text, subprocess typing, explicit
   override behavior and custom
   Ctrl commands; retain genuine errors after an action is invoked.
4. Add visible two-/three-key command menus with captured origin/repository and
   selection checks. Keep prefixes free of mutations and hidden timing state;
   define nested Esc/back behavior and safely handle invalid continuations,
   focus/repository changes and popup replacement. Expose leaves independently
   through `?` and show their complete paths.
5. Apply the reviewed defaults across Linux/macOS/Windows. Remove core Ctrl
   defaults and case-only distinctions, resolve navigation/action collisions,
   retain direct history/file comparisons, and leave user overrides explicit.
   Update command hints, generated cheatsheets/schema, usage and a migration
   table; maintain the merge guide at integration changes.
6. Verify the default key × pane/mode matrix: case parity, no core Ctrl defaults,
   unique paths, disabled no-ops, no parent/global fallback, prefix cancellation
   and captured targets, editors/filtering, custom commands/overrides and actual
   side effects. Run generation, formatting, builds, focused/full short tests,
   affected terminal scenarios and lint. When item 8 acceptance passes, complete
   its roadmap description and replace this tracker with item 9's task plan in
   that same commit.

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
