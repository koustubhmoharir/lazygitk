# Current work

Roadmap item: [8. Consistent default keys and no-op behavior](Roadmap.md#8-consistent-default-keys-and-no-op-behavior).
Status: the default registration audit and source-reviewed mode matrix are
recorded in [keymap/README.md](keymap/README.md),
[keymap-mode-matrix.md](keymap-mode-matrix.md) and the linked dispatch/default
audits. The user has approved implementation of the agreed keymap; the earlier
review pause is lifted. [keymap-allocation.md](keymap-allocation.md) records the
accepted table, with source/override projection and topology details to resolve
as implementation proceeds. Modal text fields must support Ctrl+Left/Right word
movement; these are editor controls, not core command bindings.
The latest user table reserves `d` for diff, `s` for reset/switch, `c` for
cherry-pick and `u` for undo/redo. Enter commits only while Files is focused;
it is a no-op in other normal panes. Dialogs/search/editors retain normal Enter.
`v` inspection, `h`/`l` pane switching, Tab/Shift+Tab tab switching and F3/Shift+F3
search navigation are approved. Escape exits marking mode and clears all
cherry-pick marks; remove the separate `c a` shortcut. `g c` combines selected
commits at their first selected ancestor, replacing squash/fixup, with editable
first subject and all full messages in ancestor-first order (confirmed).
Space stages only in Files and inspects elsewhere. Left/Right navigate the
Files tree; number keys remain available for pane switching.

The independent full-graph default uses `git.log.showWholeGraph: true` before
user config; explicit false overrides remain supported. Final keybindings and
combine implementation are now authorized.

Scope: one command meaning per default key/path across panes. ASCII command
letters are case-insensitive; inapplicable actions consume their key without
side effects or global fallback. Reserve Ctrl defaults for custom commands,
while preserving normal text entry and terminal Enter/Tab byte aliases. Use
visible accelerator menus for two-/three-letter paths. Keep working-tree and
previous-version diff separate as `d w` / `d p`, add pasted-hash comparison
on `d c`; Pull has no single-letter shortcut. Preserve explicit overrides and
custom commands. Frequency persistence and sorting remain item 9.

## Remaining tasks for item 8

2. Review [keymap-allocation.md](keymap-allocation.md) against each actual
   handler, context and legacy configuration field before enabling defaults.
   The TortoiseGit guide and concrete verb/prefix table are recorded; the table
   names every instantiated operation ID and the known dynamic variants.
   The fork-owned path catalogue is present, including distinct previous/next
   hunk, file and conflict paths. Implementation is authorized. Incorporate
   the latest user table, including Files-only commit Enter, Escape cancellation, combine `g c`, scoped Space, tree
   arrow registration (directional tree handlers are implemented), `d w`/`d p`/`d c`, `s h`/`s m`/`s s`/`s c`, `n b` and
   `c y`/`c x`/`c p`. Keep Pull in a deliberate menu path. Verify the implemented source-field projection against the remaining handlers
   and aliases before enabling it. Stable-ID paths override explicitly configured
   legacy fields, followed by catalogue defaults; numbered jumps keep their targets.
   The stable-ID path configuration parser/schema is present; connect it through
   the projection and dispatcher before treating overrides as active.
3. Integrate the opt-in ASCII command matching/collision normalization and add
   inactive-key consumption at small central integration points. The command
   matcher is implemented independently of exact keys and input events; the
   loader records explicit YAML field presence separately from values/legacy
   alias folding. The application still uses its existing registration path.
   Preserve literal input, search/filter text, subprocess typing, explicit
   override behavior and custom Ctrl commands; retain genuine errors after an action is invoked.
4. Extend the captured operation flows required by the
   requested bindings: Files-only commit Enter,
   full-refresh/re-render F5 without fetch, mandatory discard confirmation,
   explanatory rebase/merge confirmation,
   unified selected branch/commit checkout and ref-only branch placement,
   idempotent cherry-pick mark/unmark, Escape mark clearing and exact-list
   marked/selected review, plus ancestry-based combine with an editable message.
   Combine must replace the first selected ancestor (not an unselected parent),
   preserve all selected full messages in ancestor-first order and leave
   unselected history intact. The first version supports non-adjacent selections on linear ancestry and
   refuses selections containing or crossing merges (user confirmed). Derive
   ancestry from Git parents, never graph row/date order.
   Retain native concurrency/overwrite protection and no implicit stash. Use
   [keymap-behavior-gaps.md](keymap-behavior-gaps.md) as source evidence, not proof
   that these flows are already implemented.
5. Connect the implemented visible two-/three-key menu flow to normal-pane
   registration and verify captured origin/repository and selection checks. Keep prefixes free of mutations and hidden timing state;
   define nested Esc/back behavior and safely handle invalid continuations,
   focus/repository changes and popup replacement. Expose leaves independently
   through `?` and show their complete paths.
6. Apply the reviewed defaults across Linux/macOS/Windows. Remove core Ctrl
   defaults and case-only distinctions, resolve navigation/action collisions,
   retain direct history/file comparisons, and leave user overrides explicit.
   Update command hints, generated cheatsheets/schema, usage and a migration
   table; maintain the merge guide at integration changes.
7. Verify the default key × pane/mode matrix: case parity, no core Ctrl defaults,
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
Use verbose named terminal runs and verify the expected scenario names actually
ran; a successful exit with "no tests to run" is not coverage. Test names are
`TestIntegration/<group>/<scenario>` (there is no literal `group` segment).
Finish generation successfully before starting headless checks, and keep
integration-test sources frozen during the run. Each scenario variable must be
its filename converted from snake_case to PascalCase, using unqualified
`NewIntegrationTest`; compile-only checks do not validate those generator rules.
The shared integration config explicitly disables fetch/file/ref polling;
default-policy tests must restore values from the real default constructor.
