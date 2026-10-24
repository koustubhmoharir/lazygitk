# Consistent command keys — item 8 design draft

This draft precedes changes to application defaults. The current source inventory,
instantiated [binding snapshots](keymap/bindings.json),
[default audit](keymap-default-audit.md) and
[dispatch audit](key-dispatch-audit.md), with the
[mode matrix](keymap-mode-matrix.md), provide the evidence. Snapshots are not
proof that every mode or dynamically built menu has been exercised.

The user requested a pause before implementing the final keybindings so they
can review the proposal. Keep the final bindings unimplemented/inactive until
that review; the existing opt-in matcher and override metadata are foundations.

[TortoiseGit's log dialog](https://tortoisegit.org/docs/tortoisegit/tgit-dug-showlog.html)
keeps working-tree and previous-revision comparisons distinct and names current
HEAD as the destination of reset/rebase actions. Its
[reference browser](https://tortoisegit.org/docs/tortoisegit/tgit-dug-browse-ref.html)
uses selected refs and explicit contextual choices. Those operation roles guide
this design; this fork's user requirements determine the actual keys.

## Command meanings

One complete default key/path has one meaning across normal panes. Selected
object types may differ; unrelated operations may not share that path. ASCII
uppercase and lowercase command letters are aliases. Punctuation stays literal.
Menus form visible namespaces: `d w` means working-tree diff, while `d`
opens the diff group. No invisible chord buffer or inter-key timeout is needed.

The latest user-specified bindings in the [allocation table](keymap-allocation.md)
replace earlier proposals. Space is an explicit scoped exception: in Files it
stages/unstages the selected files/directory/root; elsewhere it inspects details.
Enter opens the commit flow globally from every normal pane. The user confirmed
that dialogs, search and editors keep their normal Enter/input behavior. Modal
Enter confirms/executes, Esc cancels/returns, and multiline Enter inserts a newline.

| Key/path | Normal-pane meaning |
| --- | --- |
| `d w` | Diff selected historical version with working tree |
| `d p` | Diff selected historical version with previous commit |
| `d c` | Diff with a commit specified by pasted partial/full hash |
| `p` / `f` | Push / explicit fetch |
| F2 | Rename/reword selected object |
| `e` / `o` | Edit selected content / open selected object externally |
| F5 | Refresh repository state and re-render every pane; no fetch |
| Delete | Delete/drop selected object |
| `x` | Discard selected changes, with confirmation |
| `a` | Amend using selected/current commit, explicitly labeled |
| `r` / `m` | Rebase current onto / merge into current from selected source, with explanatory confirmation |
| `s h` / `s m` / `s s` | Hard / mixed / soft reset current branch to selected commit, with confirmation |
| `s c` | Switch/checkout selected branch/commit, with new-name and move-non-current options |
| `n b` | Create/move branch at selected branch/commit without checkout |
| Enter | Open commit flow globally from every normal pane |
| `i` | Ignore/exclude selected working-tree path |
| `u u` / `u r` | Undo / redo |
| `q` | Quit |
| `?` / `/` / `:` | Discover commands / search / enter shell command |
| Space in Files | Toggle staging; selecting root covers all files; no `s t` or stage-all shortcut |
| Space elsewhere | Inspect selected details, with no Git mutation |
| Left in Files | Collapse expanded directory; otherwise select visible parent |
| Right in Files | Expand collapsed directory; otherwise select first child |
| `c y` / `c x` | Mark / unmark selected commits for cherry-pick, without toggling unrelated marks |
| `c p` | Cherry-pick marked or selected commits after exact-list confirmation |
| `y` | Open copy choices |

Remaining proposed navigation: `v` inspects selected details in all normal panes,
including Files where Space stages; `j`/`k` and Up/Down move between items;
`h`/`l`, Tab/Backtab and number keys switch panes; Shift+Up/Down extend selection;
Home/End move to first/last item; PageUp/PageDown page the focused view;
Alt+Up/Down scroll main output; F3/Shift+F3 navigate search matches. These
remaining choices require review; physical Left/Right in Files follow the user's
explicit tree behavior and must not fall through to pane switching.

These are proposed meanings, not a completed allocation of every action. The
review must verify each handler's actual effect; metadata synonyms do not turn
unrelated behavior into a shared verb. An existing handler with mixed effects
must be split or given separate paths. A path is a no-op where the operation
has no supported target. Native precondition/overwrite failures after invocation
remain visible.

## Groups and displaced operations

`d` is the diff group; `s` contains reset/switch; `c` contains cherry-pick marking
and application; `u` contains undo/redo. `n` contains creation, `y` copying,
`g` less common Git operations and `t` view/navigation/tools. Prefixes do nothing
until the user chooses a leaf. The two-letter leaves `n b` and `s c` open their
parameter/review flows; there are no competing chord descendants beneath them.

The previous `w`/`v` diff shortcuts, `r` rename, single `d` deletion, `c` commit,
`s t`/`s a` staging, `g k …` cherry-pick and `g u r` redo assignments are removed.
F2 is reserved for rename/reword, so editor options move to proposed F4; Delete
also replaces the suggestions-list delete shortcut. Checkout variants belong in
`s c` options/palette entries; historical-file restoration remains a separate
proposed `g c f` action. Stage-all remains discoverable, without a default key;
working-tree line staging can use proposed `g h s`/`g h u` or the palette.

The [allocation draft](keymap-allocation.md) assigns concrete third letters and
names the existing operation IDs. Verify the handler/legacy-field projection
before activation. All prefix leaves must be independently discoverable in
`?`; a leaf may open an operation's existing review dialog. Modal parameter
choices and confirmations are not hidden key-chord continuations. Prefixes
never run Git mutations by themselves. Less common variants retain visible
paths rather than case-only shortcuts or Ctrl commands.

## Dispatch, origin and overrides

Keep the change additive: a fork-owned command/path catalog, ASCII command-key
matching helpers and a prefix-menu owner, with small registration, menu and
configuration seams. Keep stable command IDs independent of the chosen paths.
Do not rewrite arbitrary input events or case-fold typed paths/search strings.
Actual keyboard parent views and logical source-parent contexts are different;
this checkout's native keyboard parent links are the menu-filter views.

Preserve explicit key overrides and custom commands. The loader now records
explicit migrated YAML fields, separately from values and legacy alias folding;
equality with defaults cannot prove provenance. Use that record and each
command's source configuration fields when projecting new defaults.
A command-ID path override can coexist with legacy fields, but must have defined
precedence, disabling and collision behavior. No configuration is silently
rewritten. Custom Ctrl bindings retain their priority and exact input semantics.
Normal text-editing controls and terminal Enter/Tab byte aliases remain input
behavior; core editor submission and other command actions get non-Ctrl keys.

Reserve complete default action keys in normal command contexts. Unsupported
pane/selection routes must be consumed before an unrelated global action can
run. Mark applicability separately from actionable preconditions; do not hide
Git errors or required commit-first/overwrite explanations. Preserve explanatory
disabled palette/tooltips. Search navigation must no longer hijack `n`/`N`.

A prefix owns its original context, backend/generation and selection/range IDs.
Main-diff origins also need the associated side selection, diff render identity
and selection span. Validate those before invoking a leaf; never redirect to a
later selection. Reuse the existing action's captured operation flow after
invocation. Do not introduce a GUI forwarding proxy that hides worker boundaries.

Native menus reuse a single temporary context, so a prefix needs an explicit
instance token and parent-stack state. Esc in a child returns to its parent;
Esc at the root cancels. Focus/repository changes, menu replacement and
completion invalidate the old token. Invalid continuations are handled no-ops
in the current menu, never replayed into another pane. Prefix menus do not
filter as the user types; ordinary search/filter/editor input stays literal.

## Implemented foundation

`gocui.NormalizeCommandKey` and opt-in `SetCommandKeybinding` provide the shared
ASCII rule and protect editable views across local, parent and global scopes.
The audit generator uses the same normalizer. Application registration is still
exact: activation waits for the complete replacement catalog/defaults, so old
case-only meanings are not silently made ambiguous. The mode matrix records
the dynamic handlers that require separate applicability/routes.
