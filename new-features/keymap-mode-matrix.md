# Effective command modes before the key migration

Use this alongside the [instantiated keys](keymap/bindings.json),
[factory inventory](inventory/inventory.json), [default field audit](keymap-default-audit.md)
and [dispatch trace](key-dispatch-audit.md). This records source-reviewed mode
differences that a fresh GUI snapshot cannot evaluate. It is an audit baseline,
not a claim that the proposed keymap is implemented or exhaustively tested.
This is the pre-migration baseline; the latest user requirements in
[keymap-allocation.md](keymap-allocation.md) supersede earlier migration
suggestions, including Enter/Space roles and the diff/reset/cherry-pick paths.

## Normal panes and selected targets

`pkg/gui/controllers.go` attaches the pane controllers, shared list/navigation
controllers, and inspection controllers. `BaseContext.GetKeybindings` reverses
attachment order; the last attached controller wins a local duplicate. Logical
context parents identify history sources; they do not inherit keyboard bindings.

| Pane | Operation target and important existing overloads | Migration constraint |
| --- | --- | --- |
| Files | Selected working paths/range; Space stages, except a single dialog-only conflict opens resolution choices; `d` discards; `r` refreshes | Separate stage, conflict choices and discard. Refresh is an application verb. The latest proposal uses global Enter for commit and `v` for inspection; Left/Right navigate the tree. |
| Local branches | Selected refs/range; Space checks out, Enter views history; rename and rebase use opposite cases | Checkout gets its own path; preserve captured selected-branch push/update/delete flows. |
| Remote branches | Selected full remote ref; Space checks out/creates a local branch, Enter views history | Keep source-ref identity; creation and checkout are distinct from inspection. |
| Remotes | Selected named remote; Enter views remote branches; fetch differs from repository-wide fetch in scope | Fetch may share a verb, with its target labeled. Remote creation has a distinct creation leaf. |
| Tags | Selected tag; Space checks out, Enter views commits; push/delete operate on the tag | No checkout on inspection/staging keys; retain tag overwrite/deletion reviews. |
| Commits | Selected commit/range in current branch history, with rebase-todo state affecting availability | Reword, edit, drop, fixup, squash, rebase, reset and checkout are different verbs. A disabled rebase `pick` must not fall through to global Pull. |
| Subcommits / reflog | Selected immutable commit from another history source | Shared history comparisons/inspection are valid. Do not manufacture current-branch-only rewrite actions. |
| Commit files | Selected historical path/range; Space toggles a custom patch, `d` rewrites a commit's file changes, Enter inspects | Patch selection gets a separate path from staging. Direct comparisons retain whole-commit versus selected-file targets. |
| Stashes | Selected stash; Space applies, `g` pops, `d` drops, Enter inspects | Apply, apply-and-drop and deletion remain distinct. Patch selection in its diff is not staging. |
| Worktrees | Selected worktree; both Space and Enter switch repository; open launches its editor | Give switching its own path; do not turn Enter/Space into an implicit checkout or switch. Explicit removal retains its separate review. |
| Submodules | Selected submodule; Space/Enter enter its repository; add/update/init/delete and URL edit are distinct | Repository navigation belongs to view/repository choices; it must not stage or inspect a different target accidentally. |
| Status | Repository-level actions; Enter opens recent repositories, while main-view focus is separate | Move repository switching into explicit navigation choices; no selected-object mutation fallback. |
| Command log / confirmations | Scrollable output, link activation and copy | Navigation remains navigation; normal operation keys must not trigger an unrelated global action. |

The factory inventory also contains modal parameter choices. They are reviewed
within the operation they configure, rather than allocated as additional root
commands. Prefix leaves open those existing reviews; they do not bypass them.

## Main/secondary diff modes

`controllers/diff_command_identity.go` resolves generic primary/discard bindings
using `diffSource()` and the main/secondary context. The source is the side pane
beneath the focused diff, not whichever side pane was last selected globally.

| Source / pane | Primary action | Existing discard action |
| --- | --- | --- |
| Working tree, main | `diff.stage-selection`: stage selected lines | `diff.discard-worktree-selection`: discard working-tree edits, with confirmation |
| Working tree, secondary | `diff.unstage-selection`: unstage selected lines | `diff.unstage-selection`: also only unstage; no working bytes are discarded |
| Commit, main | `patch.toggle-selection`: include/remove selected lines in a custom patch | `commit.discard-selection`: rewrite changes out of the commit when allowed |
| Commit, secondary | `patch.remove-selection`: remove lines from custom patch | Disabled by the existing custom-patch guard |
| No supported source / off stack | Generic metadata only; no actionable target | Generic metadata only; no actionable target |

`WorkingTreeDiffActions` and `CommitDiffActions` implement those effects. Never
assign keys from the generic static `primary-action` identity alone. In
particular, a destructive discard key must not silently mean unstage in the
secondary working-tree pane. Line editing, hunk editing, patch selection and
file editing likewise need accurate separate routes. A diff prefix snapshot
must include its source pane/selected IDs and render/line span; list IDs alone
are insufficient.

`file_command_identity.go` has another dynamic dispatch: a single conflict
requiring a dialog changes `file.toggle-stage` into `conflict.options-menu`.
The proposed staging key needs a deliberate applicability rule here; reusing
the mixed upstream handler without checking the mode would retain the overload.
Global push/pull identities are also dynamic: selected-branch routing must be
resolved while the originating pane is focused, before a prefix/palette opens.

## Modal input and terminal paths

| Mode | Actual dispatch/input behavior | Required boundary |
| --- | --- | --- |
| Menu with accelerators | `MenuContext.GetKeybindings` prepends item keys unless basic keys take precedence; `createMenu` removes only exact primary navigation conflicts | Prefix menus need case-normalized accelerator collisions and explicit frame ownership. Do not inherit arbitrary pane/global commands. |
| Filter-as-you-type menu / palette | Item accelerators are not registered. Menu filter is editable; physical Enter/Esc/arrows/page/home/end aliases remain | Letters are literal filter input. Prefix menus must disable this mode. |
| Search prompt | Special initial-binding path registers only search-context bindings; Enter is hardcoded | Keep search text literal; no root-prefix recognition. |
| Active search in a normal view | Gocui intercepts configured match/escape keys before ordinary dispatch, currently `n`/`N`/Esc | Move match navigation away from case-only command letters; test this pre-dispatch path separately. |
| Generic prompt / suggestions | Prompt is editable; Enter confirms; Tab switches suggestions; suggestions expose accept/edit/delete/navigation | Input typing and suggestion operations are explicit modal roles, not pane actions. |
| Commit summary editor | Editable; Enter submits, Tab focuses description; previous/next bindings recall messages | Printable navigation aliases must remain typed text; history needs non-letter keys. |
| Commit description editor | Editable; Enter inserts newline; configured editor confirmation submits; Tab focuses summary unless pasting | Provide non-Ctrl submission; preserve multiline paste, literal letters and cancellation. |
| Conflict view | Separate controller; Space picks a hunk, another key picks both, navigation changes conflict/hunk, undo edits resolution | Resolution choices need their own paths; Space cannot retain an unrelated pick meaning. |
| Snake | Separate game context maps navigation to direction and Esc to exit | Treat as an explicit game mode; do not expose normal Git commands as game controls. |
| Subprocess terminal | The subprocess owns terminal input | Do not intercept or normalize its typing. |

Known non-factory input includes `gocui.SimpleEditor` Ctrl/Alt word/line editing,
GUI multiline Enter handling, paste Ctrl-J conversion, menu physical aliases,
search navigation, command-log accelerators and mouse/link activation. The
cleared-config snapshot identifies hardcoded registered mouse and Enter keys;
it cannot enumerate editor or decoder conditionals.

The vendored `tcell/v3/key.go` establishes legacy byte aliases: `KeyTab = KeyTAB`
and `KeyEnter = KeyCR`. `newEventKey` handles literal control bytes, legacy
Ctrl-letter symbols, Shift-Tab and advanced protocol events separately. The
GUI driver converts legacy Ctrl-letter symbols to rune-plus-Ctrl. Thus do not
promise that physical Ctrl-I/M are independently bindable from Tab/Enter in
every terminal. Test ordinary and enhanced event forms without changing text
or key equality globally. Input word-editing controls are not core command
defaults; editor submission, suspend, quit and filter actions are commands.

## Completion boundary

The snapshot plus this matrix establishes the existing registration/dispatch
baseline and the identified dynamic splits. Allocation remains subject to a
reviewed command-ID/path table, override projection using recorded field presence and
runtime tests across these modes. It does not establish collision freedom or
new no-op behavior. No existing default was changed by producing this audit.
