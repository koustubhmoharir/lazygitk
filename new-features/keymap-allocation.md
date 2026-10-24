# Command/path allocation draft

This expands [keymap-design.md](keymap-design.md) against the instantiated
command IDs and the [mode matrix](keymap-mode-matrix.md). It is a design table,
not the installed defaults. The user will review the final bindings before
implementation/activation. The user-specified bindings below replace the earlier
allocation; remaining bindings are proposals for review. Uppercase ASCII letters
are aliases of the same path; punctuation and modifiers remain literal.

An ID is an action identity, not a claim that the action is available in every
pane. Multiple IDs on one row share the stated operation verb, on the selected
object or the explicitly labeled current destination. Unsupported routes consume
the path. Modal parameter choices remain within their operation's existing
review flow, with sensible defaults and explicit confirmation.

## User-specified commands

These commands apply to normal panes. The user confirmed that Enter retains
normal confirm/input behavior in dialogs, search and editors. Enter opens the
current repository's commit flow from any normal pane, using prepared index changes; it is not an inspection key.

| Path | Meaning | Action identities / integration |
| --- | --- | --- |
| `d w` | Diff selected historical commit/file with working tree | `diff.commit.worktree`, `diff.history-file.worktree` |
| `d p` | Diff selected historical commit/file with previous commit | `diff.commit.previous`, `diff.history-file.previous` |
| `d c` | Diff selected historical commit/file with a commit whose partial/full hash the user pastes | New hash-input comparison actions; capture selection before prompting |
| `p` | Push selected tag/local branch, or labeled current branch | `tag.push`, `branch.push-current`, `branch.push-selected` |
| `f` | Fetch selected remote, or explicitly repository-wide | `remote.fetch`, `repo.fetch` |
| F2 | Rename/reword selected object where supported | `branch.rename`, `stash.rename`, `commit.reword` |
| `e` | Edit selected content | `file.edit`, `file.edit-line`, `commit.edit`, `remote.edit-url`, `submodule.edit-url` |
| `o` | Open selected object externally | `file.open`, `worktree.open-editor`, `commit.open-browser` |
| F5 | Refresh repository state and re-render every pane, without fetch | `app.refresh`; replace any Files-only refresh shadowing this command |
| Delete | Delete/drop selected object | `branch.delete-menu`, `remote-branch.delete`, `remote.delete`, `tag.delete-menu`, `stash.drop`, `commit.drop`, `submodule.delete`, `worktree.remove-menu` |
| `x` | Discard changes in selected object, with confirmation | `file.discard-menu`, `commit.discard-file-changes`, `diff.discard-worktree-selection`, `commit.discard-selection` |
| `a` | Amend using selected/current commit, as labeled | `commit.amend-head`, `commit.amend-selected` |
| `r` | Rebase current onto selected source, with explanatory confirmation | `branch.rebase-current-onto-selected`, `branch.rebase-current-onto-selected-commit` |
| `m` | Merge selected source into current, with explanatory confirmation | `branch.merge-selected-into-current` |
| `s h` | Hard reset current branch to selected commit, with confirmation | Preselect hard mode in captured reset flow |
| `s m` | Mixed reset current branch to selected commit, with confirmation | Preselect mixed mode in captured reset flow |
| `s s` | Soft reset current branch to selected commit, with confirmation | Preselect soft mode in captured reset flow |
| `s c` | Switch/checkout selected branch or commit; review branch name/create/move options | Unified selected-source checkout flow; existing named-checkout backend is a foundation |
| `n b` | Create branch at selected branch/commit without checkout; name and move-non-current options | Extend `branch.place-at-selected-commit` to selected branch sources |
| Enter | Open commit flow globally, regardless of focused normal pane | Route `commit.create` through a global action |
| `i` | Ignore/exclude selected working path | `file.ignore-menu` |
| `u u` | Undo the last applicable operation | `branch.reflog-undo`, `conflict.undo` |
| `u r` | Redo | `branch.reflog-redo` |
| `q` | Quit | `app.quit` |
| `?` | Discover commands | `app.command-palette` |
| `/` | Search current view/history | `search.open`, `commit.search` |
| `:` | Enter a shell command | `app.shell-command` |
| Space in Files | Toggle staging of selected files/directory/root | `file.toggle-stage`; selecting the tree root covers all files |
| Space elsewhere | Inspect selected details; no mutation | Use inspection actions below; no staging, stash apply or checkout |
| Left in Files | Collapse expanded directory, otherwise select its visible parent | New directional tree action; never switch panes |
| Right in Files | Expand collapsed directory, otherwise select its first child | New directional tree action; file leaves have no child |
| `c y` | Mark selected commits for cherry-pick, idempotently | Split mark-only behavior from `commit.copy-for-cherry-pick` toggle |
| `c x` | Unmark only selected commits for cherry-pick | New selected-only unmark; do not reuse reset-all |
| `c p` | Cherry-pick marked or selected commits; confirm the exact ordered list | Extend `commit.cherry-pick` beyond copied-set/count-only confirmation |
| `y` | Open copy choices | Keep copy actions under the visible copy group |

## Remaining operation paths proposed for review

Old single-key `d` delete, `r` rename, `b` rebase, `z` reset, `c` commit and `u`
undo are removed from the proposal. `d`, `s`, `c`, `u`, `n`, `y`, `g` and `t`
are visible groups and do not mutate on the first key. The old direct `w`/`v`
diff aliases are replaced by `d w`/`d p`. There is no `s t` or stage-all shortcut;
Space on the Files tree root already covers all files. Existing stage-all and
line-staging actions can remain independently discoverable in `?`.

| Path | Meaning | Existing command IDs |
| --- | --- | --- |
| `n t` | Create tag at selected/labeled current commit | `tag.create-at-selection`, `tag.create-at-current` |
| `n w` | Create worktree | `worktree.create`, `worktree.create-menu` |
| `n r` | Add remote | `remote.create` |
| `n f` | Add fork remote | `remote.add-fork` |
| `n s` | Add submodule | `submodule.create` |
| `g c f` | Restore selected historical file version | `history-file.checkout`; keep separate from branch/commit switching |
| `g l` | Pull/update review; deliberately no single-key Pull | `branch.pull-current`, `branch.pull-selected` |
| `g f` | Fast-forward selected branches | `branch.fast-forward-selected` |
| `g t` | Configure upstream/tracking | `branch.upstream-menu`, `branch.set-upstream-to-selection` |
| `g i` | Start interactive rebase | `commit.rebase-interactive` |
| `g b` | Bisect choices | `bisect.options-menu` |
| `g v` | Revert selected commit | `commit.revert` |
| `g a` | Apply selected stash | `stash.apply` |
| `g o` | Continue/skip/abort current merge/rebase, through its operation menu | `branch.merge-rebase-menu` |
| `g y` | Git-flow choices | `branch.git-flow-menu` |
| `g x m` | Mark selected commit as fixup | `commit.mark-fixup` |
| `g x c` | Create fixup commit | `commit.create-fixup-menu` |
| `g x f` | Find fixup target | `commit.find-fixup-target` |
| `g x s` | Set fixup message | `commit.set-fixup-message` |
| `g q s` | Squash selected commits | `commit.squash-down` |
| `g q a` | Apply autosquash/fixup commits | `commit.autosquash` |
| `g r b` | Mark rebase base | `commit.mark-rebase-base` |
| `g r p` | Mark rebase todo as pick | `commit.mark-pick` |
| `c a` | Unmark all cherry-pick commits | `commit.clear-cherry-pick-selection`; distinct from selected-only `c x` |
| `g m n` | Move commits to a new branch | `branch.move-commits-to-new` |
| `g m u` | Move selected commit up | `commit.move-up` |
| `g m d` | Move selected commit down | `commit.move-down` |
| `g e c` | Commit using editor | `commit.create-with-editor` |
| `g e n` | Commit with hooks explicitly disabled | `commit.create-without-hooks` |
| `g e r` | Reword selected commit using editor | `commit.reword-with-editor` |
| `g e a` | Edit commit attributes | `commit.edit-attributes-menu` |
| `g e h` | Edit selected working-tree hunk | `diff.edit-hunk` |
| `g p c` | Create pull request | `branch.create-pull-request` |
| `g p o` | Pull-request creation options | `branch.pull-request-menu` |
| `g p v` | Open associated pull request | `branch.open-pull-request`, `commit.open-pull-request`, `diff.open-line-pull-request` |
| `g s s` | Create stash | `stash.create` |
| `g s o` | Stash creation options | `stash.create-menu` |
| `g s p` | Apply and drop selected stash | `stash.pop` |
| `g h s` / `g h u` | Stage / unstage working-tree diff lines | `diff.stage-selection`, `diff.unstage-selection`; proposed non-Space access |
| `g h t` | Toggle inclusion in custom patch | `patch.toggle-file-selection`, `patch.toggle-selection`, `patch.remove-selection` |
| `g h a` | Toggle all historical files in custom patch | `patch.toggle-all-file-selection` |
| `g h o` | Custom-patch options | `patch.options-menu` |
| `g h c` | Conflict resolution options | `conflict.options-menu` |
| `g h p` | Pick selected conflict hunk | `conflict.pick-hunk` |
| `g h b` | Pick both conflict hunks | `conflict.pick-both-hunks` |
| `g u s` | Update selected submodule | `submodule.update` |
| `g u i` | Initialize selected submodule | `submodule.initialize` |
| `g u b` | Bulk submodule operations | `submodule.bulk-menu` |
| `g z u` | Review reset against upstream | `branch.reset-upstream-menu` |
| `g z w` | Review whole-working-tree discard/reset | `worktree.discard-menu` |
| `y i` | Copy selected object's identifier | `clipboard.copy-file-path`, `clipboard.copy-branch-name`, `clipboard.copy-tag-name`, `clipboard.copy-commit-hash`, `clipboard.copy-submodule-name` |
| `y c` | Copy selected content/attributes, with format choices where offered | `file.copy-menu`, `history-file.copy-menu`, `commit.copy-attribute-menu`, `clipboard.copy-diff-selection`, `clipboard.copy-confirmation` |
| `y u` | Copy selected branch's pull-request URL | `branch.copy-pull-request-url` |

## View and navigation choices

Inspection means branch → commits, commit → changed files, historical file →
diff, or focusing selected read-only details. It never applies a stash, checks
out a ref or switches worktrees. Space performs this outside Files. `v` is a
proposed inspect alias in all normal panes, including Files, so inspecting a
working file does not compete with staging. Directory inspection in Files must
focus its details rather than toggle staging or unexpectedly collapse the tree.

| Path | Meaning | Existing command IDs |
| --- | --- | --- |
| `v`, Space outside Files | Inspect selected details | `file.enter` (split directory behavior), `remote.view-branches`, `navigation.open-commits`, `navigation.open-diff-files`, `navigation.open-history-file-diff`, `navigation.focus-diff` |
| `t r r` | Open recent repositories | `repo.switch-menu` |
| `t r w` | Switch to selected worktree | `worktree.switch` |
| `t r s` | Enter selected submodule repository | `submodule.enter` |
| `t d e` | External diff using explicitly configured comparison mode | `diff.commit.external`, `diff.history-file.external`, `diff.ref.external`, `diff.working-file.external` |
| `t d f` | Compare historical file with explicitly chosen working file | `diff.history-file.working-file` |
| `t d c` | Comparison mode options | `diff.options-menu` |
| `t d w` | Toggle whitespace display | `diff.toggle-whitespace` |
| `t d n` / `t d p` | Next / previous diff renderer | `diff.renderer.next`, `diff.renderer.previous` |
| `t d +` / `t d -` | Increase / decrease diff context | `diff.context.increase`, `diff.context.decrease` |
| `t d >` / `t d <` | Increase / decrease rename similarity threshold | `diff.rename-threshold.increase`, `diff.rename-threshold.decrease` |
| `t f t` | Text filter | `filter.open` |
| `t f s` | Working-file status filter | `file.filter-menu` |
| `t f o` | Filter options | `filter.options-menu` |
| `t t t` | Toggle file-tree view | `ui.file-tree.toggle` |
| `t t c` / `t t e` | Collapse / expand all file-tree nodes | `ui.file-tree.collapse-all`, `ui.file-tree.expand-all` |
| `t l n` / `t l p` | Next / previous all-branch history display | `history.all-branches.next`, `history.all-branches.previous` |
| `t l o` | Commit log options | `commit.log-menu` |
| `t l c` | Command log options | `command-log.options-menu` |
| `t l f` | Focus command log | `navigation.focus-command-log` |
| `t l s` | Select current branch's history | `commit.select-current-branch-history` |
| `t s n` / `t s p` | Next / previous screen mode | `ui.screen.next-mode`, `ui.screen.previous-mode` |
| `t s o` | Branch sort order | `branch.sort-menu`, `remote-branch.sort-menu` |
| `t a c` | Edit configuration | `config.edit-menu` |
| `t a u` | Check for updates | `app.check-update` |
| `t a s` | Suspend application | `app.suspend` |
| `t a q` | Quit without changing shell directory | `app.quit-without-cd` |
| `t a g` | Start game | `game.snake` |
| `t n h` / `t p h` | Next / previous diff hunk | `navigation.diff.next-hunk`, `navigation.diff.previous-hunk` |
| `t n f` / `t p f` | Next / previous file in diff | `navigation.diff.next-file`, `navigation.diff.previous-file` |
| `t n c` / `t p c` | Next / previous conflict | `navigation.conflict.next`, `navigation.conflict.previous` |
| `t n d` | Switch diff pane | `navigation.diff.switch-pane` |
| `t n j` | Jump to diff file menu | `diff.jump-to-file-menu` |
| `t h s` | Toggle line/hunk selection | `selection.diff.toggle-hunk` |
| `t h r` | Toggle sticky range selection | `selection.diff.toggle-range`, `selection.list.toggle-range` |

Physical navigation uses the focused view's list/content. Up/Down and `k`/`j`
mean previous/next item (a line, list row or conflict hunk). Shift+Up/Down extend
the selection where supported. Home/End mean first/last item. PageUp/PageDown
mean previous/next page in the focused view; Alt+Up/Down scroll main output
without moving side selection. Left/Right must collapse/expand the Files tree,
retaining focus in Files; they must not be pane-switching defaults there.
Left on a file or collapsed directory selects its visible parent; at the root
it does nothing. Right on an expanded directory selects its first visible
child; on a file or empty tree it does nothing. Respect compressed directories,
filtering and tree-root visibility; flat views must not fall through to pane
switching. The remaining navigation table is proposed for review. `h`/`l` and Tab/Backtab are proposed pane-navigation
alternates. `[`/`]` mean previous/next tab. Alt+Left/Right scroll
horizontally. Numbered pane jumps keep their explicit numbered targets. Search
match navigation moves to F3/Shift+F3. These navigation roles cover the
`navigation.*` / `selection.*` factories not individually named above, without
reusing action letters. Mouse navigation/link activation remains physical input.

## Modal controls and generic identities

Enter confirms a modal choice or single-line prompt; Esc cancels/returns, with
child-prefix Esc returning to the parent prefix. Multiline-editor Enter inserts
a newline. F10 submits the commit editor from either field; Alt+Enter is an
alternate. Editor field switching uses Tab/Backtab, message history Alt+Up/Down,
and editor-specific options F4 (F2 is reserved for rename/reword). Search input history uses Up/Down. Suggestions
use Enter to accept, Esc to cancel, Tab to return to input, Delete/`e` to delete/edit
only when the suggestions list has focus. Menu navigation retains physical
keys; printable letters in filterable menus are text, not accelerators.

These modal roles cover `menu.*`, `confirmation.*`, `prompt.*`, `suggestion.*`,
`search.*`, and `commit-editor.*`. Escape also covers `app.cancel`,
`commit.cancel-drag`, `navigation.diff.escape`, `navigation.return-files` and
`game.snake.cancel`. Game directions remain explicit game-mode navigation.
`ui.information.activate` is mouse-only today; no keyboard operation is invented.

## Review and target behavior

`x` must always request confirmation for destructive changes, even when an
existing skip-warning setting would suppress it. Rebase/merge confirmation
names the current destination, selected source and the effect on history/files.
Reset confirmation names the selected immutable commit and explains index/
working-tree effects of the chosen hard/mixed/soft mode; it does not silently
change the mode selected by the key.

`s c` captures the selected branch/commit before options appear. Let the user
attach to an existing branch, create a named branch, or explicitly approve
moving an existing non-current branch. Refuse moving a branch occupied/reserved
by another worktree, or relocating the current branch through this flow; current
branch movement uses reset. Preserve non-overwritten dirty files using Git
normal checkout behavior, without implicit stash or force. Branch-name entry,
overwrite warnings and confirmations are modal parameters, not fourth keys.
`n b` uses the same captured source/name/move rules but changes only a ref, never
HEAD/index/working files; moving a non-current branch requires explicit warning
approval. Legacy create-and-checkout/by-name/previous/force variants are options
or palette entries, not competing `n b …` or `s c …` chord descendants.

`d c` prompts for a partial/full commit hash after capturing the selected commit
and optional historical file. Resolve the entered hash unambiguously to a
commit and label both diff endpoints. Invalid/ambiguous hashes must be explained
without launching a comparison or guessing a target. Retain file/rename handling
from the existing historical comparison flows.

`c y` adds selected commits without toggling existing marks off; `c x` removes
only selected marks. `c p` reviews exact immutable commit IDs/subjects in apply
order and names the current destination branch/HEAD before confirmation. With
no marks, use the selected commits; when both sets are available, propose a
source choice (marked or selected, marked first) in the review instead of
silently combining or switching sets. Capture the chosen list and destination
through confirmation; reject stale repository/selection/destination state using
the existing native-concurrency pattern. Use Git preconditions without implicit
stash. No new application locks, CAS or automatic retries.

F5 refreshes all repository state and re-renders every pane, preserving focus
and selection where valid. It must not fetch or trigger another Git mutation.

Static `diff.primary-selection` / `diff.discard-selection` are metadata fallbacks,
not default command routes. Resolve the dynamic IDs before assigning a path,
and split working-tree staging, patch inclusion and commit rewrite guards.
`file.toggle-stage` similarly needs its conflict mode separated. The unstage
discard alias has no `x` route: use staging to unstage. No core command gets a
Ctrl default; conventional text-editing controls retain their input roles.

## Remaining review

Check each proposed leaf against the actual handler/legacy configuration field
and all live contexts before enabling it. The table is not yet a validated
runtime catalog. Preserve explicit field overrides using the loader's recorded
presence, including disabled fields and deprecated aliases. If explicit overrides
create a collision after case folding, present the conflicting action choices
instead of silently choosing by registration order. Prefix origin, availability
and stale callback checks still apply to those choices. Custom commands keep
their established precedence; Ctrl combinations remain exact.
