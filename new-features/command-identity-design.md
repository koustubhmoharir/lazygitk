# Command identity and dispatch audit

Current scope: roadmap item 2. This is a preparatory identity change, not the
new keymap or usage ranking. Mechanical source evidence is in
[inventory/inventory.json](inventory/inventory.json), regenerated with
`go run new-features/inventory/generate.go`. Factory locations are audit
references, never persistent IDs. The static inventory still needs semantic
annotation and dynamic-factory review before it is considered complete.

## Existing invocation paths

- Controllers return `types.Binding` values. `controllers.AttachControllers`
  attaches their factories to contexts; `BaseContext.GetKeybindings` visits
  attached factories in reverse order, so later attachments take precedence.
  `pkg/gui/controllers.go:resetHelpersAndControllers` owns the attachment map.
- `Gui.GetInitialKeybindings` combines GUI-owned bindings and context bindings,
  assigning view names. Custom bindings are prepended, giving them precedence
  within the matching dispatch tier. Search-prompt mode returns only that
  context's bindings rather than the full application list.
- `gocui.Gui.execKeybindings` handles active search-match/escape keys before
  normal bindings. Normal dispatch considers the focused view, its parent,
  the editor, then a global binding. `ErrKeybindingNotHandled` permits fallback;
  a disabled binding can currently request that fallback. Paste and editable
  views impose additional routing rules. Item 2 must not change these rules.
- `Gui.callKeybindingHandler` checks disabled reasons before invoking a binding.
  The palette calls this same gateway, preserving guards and error behavior.
- `OptionsMenuAction` gathers local/global/navigation bindings, including
  bindings without shortcut keys, but currently deduplicates by translated
  description. Different commands with equal labels can disappear. Metadata
  will replace description-based deduplication, not controller ownership.
- Menu accelerators are converted to bindings by `MenuContext.GetKeybindings`.
  Enter and accelerators reach `MenuContext.OnMenuPress`, which checks disabled
  reasons, pops the menu and invokes `OnPress`. Filtering menus deliberately do
  not install item accelerators; their printable keys enter filter text.
- `MenuController.physicalKeyBindings` supplies hardcoded Enter/Escape and
  navigation fallbacks so filtering menus remain usable under custom keymaps.
  Editors, search handling, mouse bindings and positional panel-jump factories
  must be audited separately from visible palette entries.

## Identity contract

Bindings and relevant menu leaves carry `CommandID` and `Verb`. IDs are explicit,
non-translated strings describing operations, not keys, labels, handler-pointer
addresses, selected hashes, ref names, source locations or list positions.
Built-in IDs use lower-case dotted names; hyphens separate words within a part.
Equivalent shortcut/menu paths use the same ID. A verb describes a shared
operation on its target, not a reason to merge distinct command identities.

Examples that the annotation review must preserve:

| Existing action | Identity/semantic distinction |
| --- | --- |
| Branch/stash rename and commit reword | Distinct IDs; shared `rename` verb |
| Files discard and branch delete | Distinct IDs and verbs despite the same upstream key |
| Commit, branch, tag and remote-branch reset menu | Same current-branch reset operation with a different selected ref parameter |
| Basic-commit and ref external difftool | Different IDs: their default endpoints differ despite the same translated label |
| Push/pull from any pane | Currently target the checked-out branch; do not label these as selected-branch operations yet |
| Commit actions available in local history, subcommits and reflog | Share an ID only when the operation itself is equivalent |
| Dynamic branch/ref/file menu choices | Stable operation ID; object values are arguments, never part of that ID |
| Reset soft/mixed/hard leaves | Distinct operations, not interchangeable aliases |

`MenuItem.ID()` remains its existing row identity. It must not be replaced by
`CommandID`: a menu can contain several rows invoking the same operation with
different object arguments. Action identity and row-selection identity solve
different problems.

Palette items inherit binding identity and keep using the binding gateway.
Menu-accelerator wrappers inherit their item's identity. Core menu leaves keep
their existing handlers and disabled reasons. No counters or persistence are
introduced in this item. Later usage instrumentation must avoid counting both
a palette/menu wrapper and its underlying binding, and must not count generic
Enter/navigation or parameter-choice prompts as independent commands.

TortoiseGit provides a useful precedent: its
[log-menu implementation](https://raw.githubusercontent.com/TortoiseGit/TortoiseGit/master/src/TortoiseProc/GitLogListBase.cpp)
uses operation IDs separately from localized labels and passes the selected
objects as operation data. This informs the separation here; we are not copying
its Windows-specific dispatch machinery.

## Context/target review map

- Files: selected working files/directories, except commit/stash/reset actions
  that affect the index, current branch or whole worktree. Main diff views also
  expose some of these working-tree actions through guards.
- Commit files: historical paths and patch selection; checkout/discard have
  worktree or history effects and are not aliases for editing a working file.
- Basic commit controller: selected commits across local history, subcommits
  and reflog. Local-commit controller adds rewriting, rebase and stack actions.
- Branches/remotes/tags: selected refs or remote definitions, with exceptions
  for merge/rebase/reset that operate on the current branch using selection as
  a source/target. Fetch actions can target a selected remote or branch upstream.
- Stash: selected stash entries. Its branch-creation binding calls the same
  `RefsHelper.NewBranch` as commit/ref contexts, with the stash commit as base;
  it is not an implicit `git stash branch`/apply workflow.
- Worktrees/submodules: selected paths/repositories; bulk menus and initialization
  have wider scope. Panel switching is navigation, not a Git operation.
- Main/secondary diffs: selected lines/hunks/files and current diff mode; some
  operations are mode-dependent. Merge-conflict selection/undo is its own scope.
- Status/global: settings, repository switching, application state and current
  branch operations. Do not infer selected-object semantics from the pane name.
- Menus, prompts, suggestions, search and commit editors: modal input/navigation
  stays distinct from Git actions. Text entry retains normal case/modifier rules.

The machine inventory records factory keys, handler/availability expressions,
platform defaults and hardcoded dispatch leads. Per-action annotation must be
checked against this target map and the actual handler, especially dynamic or
mode-dependent factories; translated labels alone are insufficient evidence.

## Custom commands

An optional `id` supplies durable identity as `custom.user.<id>`. IDs are unique
across the loaded nested custom-command tree; one definition may list several
contexts or alternate keys without creating multiple identities. Valid IDs
start with an ASCII letter/digit and otherwise contain letters, digits, `.`,
`_` or `-`. Duplicate/invalid-ID errors identify config positions, not command
or prompt contents.

Without `id`, a leaf uses `custom.auto.<sha256>` of a versioned JSON payload of
its unrendered command, prompts, output mode and after-hook settings. Keys,
contexts, description, loading text and output title do not participate. A
definition change can change fallback identity; use an explicit ID when that is
undesirable. A menu without an explicit ID hashes sorted recursive child IDs,
so reordering its children does not change its identity.

No rendered selection/form values are involved. Only the identity string is
intended for eventual usage persistence, never the payload or raw shell/prompt
contents. A digest is a stable fingerprint, not encryption or anonymization.
Prompt menu options are parameters within their custom command, not new
top-level action identities.
