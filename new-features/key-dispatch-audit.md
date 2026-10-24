# Roadmap 8: key dispatch and prefix-menu audit

Bounded read-only trace for Tracker item 8, task 1 and the dispatch/menu seam
needed by task 4. No key allocation is proposed here. Line references point to
the current checkout and may shift as implementation proceeds.

## Native dispatch flow and observed semantics

1. `pkg/gocui/key.go:32-73` represents printable keys as exact runes and
   `Key.Equals` compares key name, string and modifier exactly. `keybinding.go:35-37`
   delegates to that equality. Thus ASCII `a` and `A` are distinct today; there
   is no case-folding or key collision normalization in gocui.
2. `pkg/gui/keybindings.go:430-481` resets the registered key list, registers
   every key on each `types.Binding`, and runs `GetDisabledReason` at invocation.
   A disabled binding returns a handled nil unless `AllowFurtherDispatching`
   is true, in which case it returns `ErrKeybindingNotHandled`. The latter is
   explicitly a fallthrough request, not a safe no-op.
3. `pkg/gocui/gui.go:2038-2105` scans registrations in order. A matching local
   handler normally ends dispatch even when it returns nil. A declining local
   handler stops the scan and allows a matching **native view parent** handler,
   then the focused editor, then the remembered global handler. If there is no
   local match, the first matching native view parent and global handlers are
   remembered. This is the concrete path by which an unavailable local action
   can run an unrelated view-parent/global action or insert text. Key
   registration order is consequently observable for duplicate bindings.
   Repository search confirms `View.ParentView` is assigned only to the two
   menu filter child views in `pkg/gui/views.go:147-148`; this is not general
   main-pane keyboard bubbling.
4. Editable views get special treatment at `pkg/gocui/gui.go:2203-2217`:
   printable keys do not match ordinary bindings unless `KeybindOnEdit` is set.
   After local/parent dispatch, the editor gets the key before global dispatch.
   The menu filter editor deliberately returns false for unhandled keys
   (`pkg/gui/editors.go:52-75`), preserving global controls such as Ctrl-C while
   printable input is consumed as filter text. Search has a gocui-level special
   path for next/previous/escape at `gui.go:2046-2061`.

**Implication for reserved inactive keys:** the robust existing primitive is a
registered handler that returns nil after deciding the key is inapplicable.
Using `AllowFurtherDispatching` for an inactive verb explicitly invites the
fallback above. Capturing keys globally or forwarding from a generic GUI proxy
would also interfere with editable views, paste, search, and global controls.

## Menu, popup, editor and filter behavior

- `pkg/gui/menu_panel.go:14-27` appends a cancel row unless hidden. At lines
  32-58 it removes item accelerators conflicting with the *primary* configured
  confirm/return/previous/next keys (unless `KeepConflictingKeybindings`). It
  does not normalize case or inspect every alternate navigation key.
- The same file at lines 70-88 installs menu state, clears filter text, resets
  all keybindings and pushes the menu context. The TODO at line 91 says nested
  menu rendering is not ensured. Menus reuse one `MenuContext` and menu view;
  opening a replacement menu resets the active menu's model/selection. This is
  not a stack of independent menu instances.
- `pkg/gui/controllers/menu_controller.go:40-73` registers execute/select,
  configured Return and, for filter-as-you-type menus, physical Enter/Esc and
  arrow/page/home/end navigation. `physicalKeyBindings` only omits a physical
  alias when that exact key appears in the configured binding (`:75-115`).
  Enter confirms and Esc closes even if the user configured other keys. These
  aliases are important conventional behavior and a potential collision with
  any prefix recognizer.
- For filtered menus the menu filter is an editable child view of the menu
  (`pkg/gui/views.go:144-148`). Printable letters go into filter text rather
  than menu accelerators; physical navigation/confirm aliases still operate.
  `pkg/gui/context/menu_context.go:68-101` preprocesses `@` to search binding
  labels when allowed. The command palette opts into filter-as-you-type,
  `@`-key search, and keeping conflicts at
  `pkg/gui/controllers/options_menu_action.go:17-53`; its rows are built from
  current local/global/navigation bindings and can be invoked via
  `CallKeybindingHandler`.
- `pkg/gui/popup/popup_handler.go:58-60,106-149` forwards menu/prompt/confirm
  creation through callbacks. Prompt panels are editable, and their confirm,
  close and suggestion callbacks are supplied by each caller. There is no
  shared prefix/menu cancellation abstraction in this layer.
- `pkg/gui/controllers/filtering_menu_action.go:13-109` captures the selected
  path/author into local variables before building callbacks; this is a useful
  precedent for immutable origin values. Other ordinary menus may instead
  close over objects or re-read context at callback time, so capture safety is
  flow-specific, not a universal `MenuItem` guarantee.

**Two different parent relationships:** gocui `View.ParentView` is the native
keyboard fallback described above. Separately, GUI contexts implement
`ParentContexter.GetParentContext()` (`pkg/gui/types/context.go:35-38`), used by
flows such as remote-branch source selection and subcommit/history contexts.
That logical relationship can identify the originating context/selection for a
prefix snapshot, but it does not automatically inherit or merge keyboard
bindings. `BaseContext.GetKeybindings` (`pkg/gui/context/base_context.go:136-145`)
assembles only its attached controller functions. Native dispatch fallback is
still the local → native-view-parent → editor → global sequence in gocui; do
not propose or assume logical main-pane key bubbling.

## Case, overrides, aliases and accelerators

- Exact rune matching means ASCII case folding must happen in a deliberate
  shared registration/match seam. Folding only input or only configured keys
  risks duplicates where both cases are configured. Preserve literal text by
  keeping the editor's existing printable-key ownership and applying command
  matching only after the focused context has established that the key is a
  command candidate.
- `pkg/gui/controllers/command_palette.go:8-49` (palette-only binding merge) preserves local
  precedence when aliases share command identity; inspect alongside
  `command_palette_test.go:110-139`. Runtime registration itself is still a
  flat ordered list and does not implement a general user override resolver.
  Configured key lists feed bindings through `config.GetValidatedKeyBindingKeys`
  (`pkg/gui/keybindings.go:63-76`); custom bindings are prepended by
  `GetInitialKeybindingsWithCustomCommands` (`:407-428`), explicitly to give
  them precedence over defaults. An audit
  or implementation must distinguish identical local/global keys and custom
  Ctrl commands from built-in defaults, rather than silently deleting user
  intent.
- Menu item accelerators are native `MenuItem.Keys`, rendered as the first
  accelerator and matched by menu keybindings; menu key precedence is controlled
  by `MenuViewModel.keybindingsTakePrecedence` (`menu_context.go:147+`). The
  command palette intentionally preserves accelerator conflicts. Case-only
  accelerator duplicates remain distinct at the gocui equality layer.
- **Observed terminal conversion:** `pkg/gocui/tcell_driver.go:339-353`
  preserves `tcell.KeyRune` plus `EventKey.Str()`. For `tcell.KeyCtrlA` through
  `KeyCtrlZ`, it converts the key name to a rune (`a` through `z`) and changes
  the key name to `KeyRune`, while retaining the event modifier mask. Thus
  Ctrl-J/M/I arriving as those tcell control-key constants become rune
  `j`/`m`/`i` with Ctrl modifier. This is a general conversion, not proof of
  what every terminal emits for Enter or Tab.
- **Aliases established in this code:** during paste only,
  `pkg/gocui/gui.go:1753-1766` rewrites Ctrl-J to gocui `KeyEnter` when the
  event matches rune `j` + Ctrl. Menu `physicalKeyBindings`
  (`pkg/gui/controllers/menu_controller.go:79-112`) adds a `KeyEnter`
  execution binding for filter-as-you-type menus and adds `KeyEsc` cancellation,
  plus arrow/page/home/end navigation where those physical keys are not already
  configured. Gocui's default editor handles `KeyEnter` as acceptance
  (`pkg/gocui/edit.go:64-67`); GUI multiline editor handling inserts newline
  only when explicitly allowed (`pkg/gui/editors.go:7-12`). The traced GUI/menu
  code does not establish a generic Tab-to-command alias. Do not infer one;
  terminal emissions and tcell behavior require separate evidence.

## Search and other mode-specific hardcoded keys

- Gocui view search is a special dispatch path before ordinary binding lookup
  (`pkg/gocui/gui.go:2046-2061`). It checks `NextSearchMatchKeys`,
  `PrevSearchMatchKeys`, then `SearchEscapeKeys`, and consumes a match directly
  via next/previous match or search clearing. Defaults are `n`, `N`, and Esc
  (`gui.go:318-321`); lazygit replaces these from configured `NextMatch`,
  `PrevMatch`, and `Return` at `pkg/gui/gui.go:522-524`. While `v.IsSearching()`
  these keys are unavailable to local, parent, editor, or global dispatch.
- Search/filter prompt contexts also replace the normal binding set:
  `GetInitialKeybindingsWithCustomCommands` (`pkg/gui/keybindings.go:407-418`)
  returns only that context's bindings when current context is
  `SEARCH_CONTEXT_KEY`; custom bindings are not included on this special path.
  Search prompt Enter is explicitly bound in
  `pkg/gui/controllers/search_prompt_controller.go:20-30`. Filtering has its
  own editor/menu behavior described above.
- Other hardcoded mode-sensitive paths for the effective matrix include gocui
  Ctrl/Alt editing (`pkg/gocui/edit.go:28-80`), paste-only Ctrl-J conversion,
  editable prompt/commit-message/description editors (`pkg/gui/editors.go:7-49`),
  search navigation, menu filter physical keys, and operation-specific contexts
  whose controller factories replace normal pane bindings. Trace each factory's
  active-context behavior; these are not globally available verbs.

## Existing capture primitives and safe prefix validation

- Repository generation exists: `pkg/gui/gui.go:126-129` increments on repo
  switch at `:607-615`, and `StateAccessor.GetRepoGeneration` exposes it
  (`:189-191`). Capture this integer when opening a prefix and reject continuation
  if it changes.
- The generic selected-ID contract is `types.IListContext.GetSelectedItemId()`
  and `GetSelectedItemIds()` at `pkg/gui/types/context.go:327-338`. It is
  available after establishing that the origin context implements
  `IListContext` (for example, a type assertion on captured `types.Context`);
  `IListPanelState` itself (`:449-453`) exposes selected line indices only, not
  IDs. Implementations delegate to `ListViewModel` (`pkg/gui/context/list_view_model.go:28-61`),
  returning IDs in range order. File-tree view models expose selected node
  and selected ranges (`pkg/gui/filetree/file_tree_view_model.go:40-73`;
  commit tree equivalent `commit_file_tree_view_model.go:84-117`). A prefix
  origin snapshot can record context identity/key + selected item ID(s) + range
  endpoints/IDs + repository generation. On each continuation compare those
  captured values directly with the origin context; do not call the eventual
  leaf handler against whichever item happens to be selected later.
- A small additive prefix-flow value can own origin context, immutable captured
  target IDs, repository generation, and menu identity/token. Its leaf closure
  should first validate that the active prefix is still that token, current
  context is the expected prefix context, repo generation matches, and origin
  selection/range IDs still match; then invoke a leaf built from captured target
  values. The origin need not remain focused while the prefix menu is open.
  This avoids a generic GUI event-forwarding proxy and avoids accidentally
  sampling an unrelated current selection at leaf time.
- Replacement/cancellation needs an explicit invalidation seam. `createMenu`
  currently mutates the shared menu context and pushes it; `OnCancel` belongs
  to the menu options, and ordinary menu replacement does not universally call
  the old cancellation callback first. A prefix flow therefore needs a token
  invalidated on menu replacement, Esc/back, repository change and completion.
  A nested menu must define whether Esc closes only the child or cancels the
  whole prefix; current menu close uses Return and calls `OnMenuPress(nil)`
  (`menu_controller.go:135-147`), with no generic nested-back stack.

## Hardcoded paths and outstanding seams

Known command paths outside ordinary configurable action bindings include:

- gocui search match navigation and search escape, including configured n/N
  overrides that consume those keys before global dispatch;
- paste CR/Ctrl-J conversion;
- gocui editor Ctrl/Alt word and line editing (`pkg/gocui/edit.go:28-80`);
- GUI editor Enter handling and menu filter behavior (`pkg/gui/editors.go`);
- menu physical Enter/Esc/arrows/page/home/end aliases;
- menu accelerator filtering and its primary-key-only conflict set;
- hardcoded extra-panel accelerators (`pkg/gui/extras_panel.go:20,37`);
- context-specific prompt Enter handlers and context navigation; plus terminal
  driver decoding/aliases, which still require a complete inventory.

The central seams to evaluate later are: (1) ASCII-only canonicalization and
collision detection at configured command registration with explicit user and
custom-command precedence; (2) a handled no-op binding for inactive command
letters, keeping editor-owned text and menu filtering intact; (3) an additive,
`MenuItem`-native prefix flow using existing menus/accelerators and captured
origin values; and (4) prefix invalidation tied to menu replacement, Escape,
repo generation, and selection identity. These are seam candidates, not an
implementation plan or key allocation.

### Companion evidence and implementation boundary

The [effective snapshots](keymap/README.md), [platform field audit](keymap-default-audit.md)
and [mode matrix](keymap-mode-matrix.md) supplement this bounded trace with
factory expansion, dynamic operation roles and vendored terminal decoding.
The [design draft](keymap-design.md) records the TortoiseGit guide and proposed
keys. Complete command/path allocation and runtime acceptance tests remain
Tracker tasks; this baseline does not prove collision freedom or no-op behavior.
