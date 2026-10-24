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
Menus form visible namespaces: `g c` means checkout, not the meaning of `c`
pressed by itself. No invisible chord buffer or inter-key timeout is needed.

Space staging is agreed: stage/unstage selected working files or working-tree
diff lines, otherwise do nothing in normal panes. The proposed Enter role means
following the selected item: branch → commit list, commit → changed-file list,
file → diff. It must not check out a ref, apply a stash or switch worktrees.
Modal Enter confirms/executes, Esc
cancels/returns, and editable inputs retain normal typing. These are explicit
modal/input roles, not permission for normal pane commands to change meaning.

| Key | Proposed normal-pane meaning |
| --- | --- |
| `w` | Compare selected historical version with the working tree |
| `v` | Compare selected historical version with its previous version |
| `p` | Push the selected/current local branch, as labeled in the review |
| `f` | Explicit fetch |
| `r` | Rename/reword the selected object |
| `e` | Edit the selected object's editable content |
| `o` | Open the selected object externally |
| `d` | Delete/drop the selected object; never discard working-file edits |
| `x` | Discard selected changes; never substitute ref/commit deletion |
| `a` | Amend using the selected/current commit, explicitly labeled |
| `b` | Rebase current onto the selected source |
| `m` | Merge the selected source into current |
| `z` | Reset current HEAD to the selected/reviewed source |
| `c` | Create a commit from prepared changes |
| `i` | Ignore the selected working-tree path |
| `u` | Undo the last applicable Git operation |
| `n` | Open creation choices |
| `g` | Open Git-operation choices |
| `s` | Open staging choices |
| `y` | Open copy choices |
| `t` | Open view/navigation/tool choices |
| `q` | Quit the application |
| `j` / `k` | Next / previous item, also available on Down / Up |
| `h` / `l` | Previous / next pane; number keys also switch panes |
| Left / Right | Collapse / expand the Files tree; never switch panes there |
| Space | Toggle staging in working-tree files/diffs; otherwise no-op (agreed) |
| Enter | Inspect/open details; otherwise no-op |
| Shift+Up / Shift+Down | Extend selection where supported |
| Home / End | First / last item |
| PageUp / PageDown | Previous / next page in the focused view |
| Alt+Up / Alt+Down | Scroll main output without moving side selection |
| F3 / Shift+F3 | Next / previous search match |
| F5 | Explicit refresh |
| `?` | Command palette |
| `/` | Search |
| `:` | Shell-command prompt |

These are proposed meanings, not a completed allocation of every action. The
review must verify each handler's actual effect; metadata synonyms do not turn
unrelated behavior into a shared verb. An existing handler with mixed effects
must be split or given separate paths. A path is a no-op where the operation
has no supported target. Native precondition/overwrite failures after invocation
remain visible.

## Proposed groups

| Group/path | Meaning |
| --- | --- |
| `n b p`, `n b c` | Create/move a branch pointer here; create and check out a new branch |
| `n t`, `n w` | Create tag, worktree |
| `n r`, `n f`, `n s` | Add remote, fork remote, submodule |
| `g c s`, `g c n`, `g c p`, `g c f` | Checkout selected, by name, previous, or explicitly force checkout |
| `g f` | Fast-forward/update the selected branch |
| `g l` | Pull/update choices; never a one-letter Pull default |
| `g x` | Fixup choices |
| `g a` | Apply the selected stash |
| `g e …` | Editor-based commit, reword, attributes and hunk choices |
| `g b` | Bisect choices |
| `g i` | Interactive rebase choices |
| `g t` | Tracking/upstream choices |
| `t r w` | Switch to the selected existing worktree |
| `g u r` | Redo |
| `g u …` | Explicit submodule update/initialization choices |
| `g p …` | Pull-request operations |
| `g q …` | Squash/autosquash operations |
| `g r …` | Advanced rebase/base operations |
| `g s …` | Explicit stash operations |
| `g k …` | Collect/apply/clear cherry-pick selections |
| `g o` | Continue/skip/abort through the operation's review menu |
| `g m …` | Explicit commit movement operations |
| `g h …` | Custom patch and conflict-resolution operations; separate from staging |
| `s …` | Stage/unstage selected/all changes |
| `y …` | Copy identifier/path/message/content, with explicit format choices |
| `t …` | View/diff/log settings, sorting, filters, navigation and repository switching |

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
