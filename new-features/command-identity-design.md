# Command identity and dispatch audit

Foundation completed in roadmap item 2. Later features extend this identity
contract without adding usage ranking yet. Mechanical source evidence is in
[inventory/inventory.json](inventory/inventory.json), regenerated with
`go run new-features/inventory/generate.go`. Factory locations are audit
references, never persistent IDs. The semantic review below supplements the
static inventory; source expressions are evidence, not inferred action names.

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
  bindings without shortcut keys. The fork deduplicates by resolved operation
  ID, combines alias shortcuts and retains distinct equal-label commands.
  Local aliases retain precedence over global aliases; copies prevent palette
  construction from mutating installed bindings. Controller ownership stays intact.
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

Mode-dependent bindings can supply `CommandIDFunc` and `VerbFunc`; consumers
use `GetCommandID`/`GetVerb` to resolve them when invoked, not when shortcuts
were first registered. These callbacks must be cheap, make no Git calls and
return fixed operation identifiers rather than hashes/names of selected data.
Static fields remain useful generic identities for off-stack tooling.

The main diff selection demonstrates why this matters: working-tree lines stage
or unstage depending on pane side; historical lines toggle patch membership,
and the custom-patch pane removes selected patch lines. Discarding working-tree
lines differs from removing lines from a commit. The identity resolver describes
these existing operations without changing their handlers or disabled guards.

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
`MenuItem.Binding` explicitly identifies palette proxies. A successful handler
return is not necessarily acceptance: some handlers open conditional prompts or
menus even without `OpensMenu`. Usage instrumentation must follow acceptance,
not assume that every nil return means a completed leaf action.

## Dynamic and low-level factory review

- Item 2 reviewed its 312 binding and 176 menu factories. Later additions are
  recorded in the regenerated inventory and their feature designs. Five intentionally
  lack independent action IDs: the two worktree-location choices, the two custom
  prompt-option factories and the no-applicable-custom-commands placeholder.
  The former four supply parameters to their parent action; the last does nothing.
- Branch reset and operation-resume factories concatenate only finite internal
  enums (`soft`/`mixed`/`hard`, `continue`/`abort`/`skip`), never translated text.
  Dynamic checkout/recent-repository/diff-file rows share an operation ID but
  retain separate row identity and selected arguments. Regular merge preserves
  one ID even when its label changes with Git config; explicit `--ff` and
  `--no-ff` variants are distinct. `--ff` is not `--ff-only`.
- Panel-jump rows are parameterized navigation, not different Git actions. Their
  IDs do not include positional shortcut numbers. The fork retains definitions
  for panels with no assigned key, preserving the original positional mapping
  and hidden shortcut-only presentation. Next/previous panel navigation remains
  available in the palette; this item does not introduce a panel chooser.
- The primary file binding resolves to conflict options for a single dialog-only
  conflict, otherwise toggle staging. The resolver shares the existing predicate
  and never incorporates a selected path or file contents into an ID.
- The three low-level registration calls are GUI binding installation, per-key
  gocui installation and editable word-motion configuration. No alternate Git
  action registry exists there. `gocui/keybinding.go` owns view/parent/editor/
  global routing; `gocui/edit.go` owns text editing (including Ctrl word/delete/
  cursor/paste behavior). Text editing is not a case-insensitive core command.
- Hardcoded dispatch leads include search-match/escape interception, editor
  keys, physical menu safety keys, mouse handling and normal non-input enums.
  The raw case list deliberately retains non-key cases. Menu `essentialKeys`
  protects confirm/cancel/navigation from accelerators; printable filtering keys
  type text rather than execute displayed shortcut reminders. These paths and
  platform modifier defaults are inventoried without changing their behavior.
- Item 3 adds five fixed-endpoint diff bindings with distinct IDs for commit,
  historical-file and explicitly chosen-file comparisons. Their shared verbs
  express the same comparisons at each scope. Guards reject ambiguous ranges,
  missing commits and unsupported historical entries; captured OIDs/paths are
  operation data, never action identity. See [direct-diff-design.md](direct-diff-design.md).

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
