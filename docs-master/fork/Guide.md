# lazygitk user guide

lazygitk is a fork of lazygit. This guide lists every key it uses and how its
behaviour differs from upstream lazygit. The reasons behind these choices are
in the [design principles](Design.md). The generated per-pane reference is
[Keybindings_en.md](../keybindings/Keybindings_en.md).

## How keys work

- **Letters are case-insensitive.** `p` and `P` are the same key. The one
  exception is Shift+J and Shift+K, which extend the selection. Punctuation
  is literal: `<` is not `,`.
- **Key paths.** A path such as `d w` means: press `d`, which opens a small
  menu, then press `w` in it. The first key never changes anything. Escape
  steps back one menu level; Escape at the top closes it.
- **One meaning everywhere.** A key does the same thing in every pane, on
  whatever is selected there. Where it has nothing to act on, it does nothing:
  a menu leaves out entries that do not apply to the focused pane, and a first
  key with nothing applicable opens no menu.
- **`?` lists every command** in the app with its full key path, including
  commands that have no key. The focused pane's and the global commands come
  first, then each other pane's in its own section, then the focused pane's
  navigation. Popups (menus, prompts, confirmations, search, suggestions and
  the commit message editor) list their commands only while they are focused. Within a section, single keys come first, then the `d`, `s`,
  `n`, `c`, `u`, `y`, `g`, `t` and `w` groups, then commands without a key.
  Commands that have nothing to act on here (such as another pane's, "Available
  in Commits") are faded; commands that apply but cannot run now are crossed
  out; selecting either says why. Typing filters by name. Usage counts are
  still recorded but do not order the palette; "Reset command usage counts"
  clears them. Several lazygitk instances share the counts, and a count saved
  by one is kept when another saves at the same time.
- **Keys are fixed.** The `keybinding` section, `commandBindings` and
  custom-command `key` settings are ignored, with a startup notice naming the
  file. Custom commands run from `?`.
- **Ctrl** is used only inside the commit message editor and popups: Ctrl+S or
  Ctrl+Enter (Cmd+Enter on macOS) submits the description, Ctrl+O opens the
  commit editor menu or copies a confirmation's text, and Ctrl+Left/Right move
  by word in text fields.

## Keys

### Everywhere

| Keys | Action |
| --- | --- |
| `?` | Command palette |
| `/` | Search the pane (Commits, Sub-commits, diffs) or filter it (other lists and menus); a list pane never scrolls sideways to a match |
| F3, Shift+F3 | Next, previous search result |
| `p` | Push the selected branch, or the checked-out branch outside Branches; push the selected tag in Tags |
| `f` | Fetch all remotes; in Remotes, fetch the selected remote |
| `g l` | Pull (deliberately no single key) |
| F5 | Reload all repository state and redraw every pane, drawing the commit graph again; never fetches |
| `u u`, `u r` | Undo, redo (from the reflog) |
| `s p` | Switch to the previous branch; the menu names it (not in Status, where `s` is for repositories) |
| `c p` | Cherry-pick the selected commits, after reviewing the list; a successful pick clears the selection, and one that stops on conflicts keeps it |
| `c c` | Continue the cherry-pick, rebase, merge or revert in progress, from any pane; refuses while files are unmerged |
| `:` | Run a shell command |
| `=` or `+`, `-` | Next, previous screen mode (normal, half, full) of the focused area: the side panels, or the main views |
| `q` | Quit |
| Escape | Leave the current layer, mode or menu |

### Moving around

| Keys | Action |
| --- | --- |
| `j` `k`, Down Up | Next, previous item or line |
| `h`, `l` | Previous, next pane |
| `1`–`5` | Status, Files, Branches, Commits, Stash pane |
| `0` | Focus the main view (from Status, Files, Branches, Remote branches, Tags, Commits, Reflog, Sub-commits, Commit files, Stash) |
| Tab, Shift+Tab | Next, previous tab of a pane |
| Home, End | First, last item |
| PageUp, PageDown | Previous, next page |
| Alt+Up, Alt+Down | Scroll the main view without moving the selection |
| Alt+Left, Alt+Right | Scroll left, right |
| Shift+Up, Shift+Down; Shift+K, Shift+J | Extend the selection |
| `v`, Space | Inspect or drill down (Space toggles staging in Files and the selection in Commits instead) |

Inspection opens a branch's or tag's commits, a commit's or stash entry's
files, a file's diff, a remote's branches, a worktree's commits (without
switching to it) or a submodule (Escape returns to the parent repository).

### Files

| Keys | Action |
| --- | --- |
| Space | Stage or unstage the selected files or directory; select the root to stage everything |
| Enter | Commit (Enter does nothing in other panes) |
| `a` | Amend the last commit |
| `v` | Inspect the file: focus its diff, open conflict resolution, or enter a submodule |
| Left, Right | Collapse the directory or select its parent; expand it or select its first child |
| `x` | Discard changes, always with confirmation |
| `i` | Ignore or exclude |
| `e`, `o` | Edit, open |
| `d w`, `d s`, `d c` | Difftool against HEAD, the staged version, or a pasted commit hash |
| `d m` | Merge tool for the selected conflicted files |
| `g e c`, `g e n` | Commit with the Git editor; commit without hooks |
| `g s s`, `g s o` | Stash all changes; stash options |
| `g h c` | Merge conflict options |
| `g z u`, `g z w` | Upstream reset options; working-tree reset options |
| `w s` | Filter by status |
| `t t t`, `t t c`, `t t e` | Tree or flat view; collapse all; expand all |
| `t d e` | External diff tool |
| `y n`, `y p`, `y f`, `y d`, `y w` | Copy name, path, absolute path, file diff, all files' diff |

### Diff views

In the main view of a working file (staging), of a commit (patch building) or
of a stash entry:

| Keys | Action |
| --- | --- |
| Space | Stage or unstage the selected lines; while building a patch, toggle them in it |
| `x` | Discard the selected unstaged lines, with confirmation; unstage staged lines immediately; in a commit, remove them from the commit |
| `]`, `[` | Next, previous hunk |
| `}`, `{` | Next, previous file |
| `e` | Edit the file |
| `g h s`, `g h u`, `g h t` | Stage, unstage, toggle in patch (the same as Space) |
| `g e h` | Edit the hunk in the external editor |
| `g e c`, `g e n` | Commit with the Git editor; without hooks |
| `g p v` | Open the pull request at this line |
| `t n d` | Switch between the staged and unstaged diff |
| `t h s`, `t h r` | Line or hunk selection; sticky range selection |
| `t n j` | Jump to a file in the diff |
| `y d` | Copy the selected lines |
| Escape | Back to the side pane |

### Merge conflicts view

| Keys | Action |
| --- | --- |
| Up, Down | Previous, next hunk of the conflict |
| `]`, `[` | Next, previous conflict |
| Space | Pick the selected hunk |
| `g h b` | Pick both hunks |
| `g h p` | Pick the selected hunk (the same as Space) |
| `u u` | Undo the last pick |
| `e`, `o` | Edit, open the file |
| `g h c` | Conflict options |
| `d m` | Merge tool for this file |
| Escape | Back to Files |

### Commits, Reflog and Sub-commits

| Keys | Action |
| --- | --- |
| `v` (and Space outside Commits) | Show the commit's files |
| `d w`, `d p`, `d c` | Difftool against the working tree, the previous commit, or a pasted commit hash |
| `s c` | Check out the commit on a branch: a name prompt that lists the branches already at the commit first; a new name is created there, and a name that points elsewhere is moved here after a warning |
| `s d` | Check out the commit as a detached HEAD |
| `n b` | Create or move a branch here without checking out |
| `s h`, `s m`, `s s` | Hard, mixed, soft reset of the checked-out branch to this commit. A hard reset's review lists the untracked or ignored files that it would overwrite or remove, and approval is needed before it runs |
| `r` | Rebase the checked-out branch onto this commit |
| `g c` | Combine the selected commits |
| `n t`, `n w` | New tag here; new worktree |
| `o` | Open the commit in the browser |
| `t l s` | Select the current branch's commits |
| `w v` | Show the commit in Commits (Reflog, Sub-commits) |
| `y h`, `y i`, `y s`, `y m`, `y b`, `y u`, `y a`, `y t`, `y d` | Copy full hash, short hash, subject, message, body, URL, author, tags, diff |

Commits only:

| Keys | Action |
| --- | --- |
| Space | Add the commit to the selection, or remove it; on a Shift run, toggle the whole run |
| Escape | Cancel a lone Shift run; otherwise remove the log filter, then clear the selection |
| F2, `g e r` | Reword; reword in the editor |
| `e` | Edit: start an interactive rebase here, or mark for editing during one |
| Delete | Drop |
| `a` | Amend this commit with the staged changes |
| `<`, `>` | Move the commit up, down (`,` and `.` do nothing) |
| `g i` | Start an interactive rebase for the branch |
| `g r p`, `g r b` | Mark as pick; mark as rebase base |
| `g e a` | Commit attribute options (author, co-author) |
| `g v c` | Revert |
| `g m n` | Move unpushed commits to a new branch |
| `g b` | Bisect options |
| `g p v` | Open the pull request |
| `t l o` | Log options |
| `w a`, `w d` | Show only the selected commits and their ancestors, or their descendants |
| `w b` | Show branch / tag: only a branch or tag at this commit and its history (a menu asks which when several are here) |

### Commit files

| Keys | Action |
| --- | --- |
| `v`, Space | Show the file's diff |
| `d w`, `d p`, `d c` | Difftool against the working file, the previous commit, or a pasted commit hash |
| `t d f` | Difftool against any working file you choose |
| `g v f` | Restore this version of the file into the working tree |
| `x` | Remove this file's changes from the commit |
| `e`, `o` | Edit, open |
| `g h t`, `g h a` | Toggle the file, or all files, in the custom patch |
| Left, Right; `t t t`, `t t c`, `t t e` | Tree navigation and view, as in Files |
| `y n`, `y p`, `y f`, `y c`, `y d` | Copy name, path, absolute path, content at the commit, diff |

### Branches, Remote branches and Tags

| Keys | Action |
| --- | --- |
| `v`, Space | Show the commits |
| `s c` | Switch to the branch; for a remote branch, to its local branch, creating it to track the remote one if needed. On a tag, the name prompt as for a commit |
| `s d` | Check out its commit as a detached HEAD |
| `n b` | Create or move a branch at its commit without checking out |
| `r`, `m` | Rebase the checked-out branch onto it; merge it into the checked-out branch |
| `s h`, `s m`, `s s` | Reset the checked-out branch to it (Branches) |
| `n w` | New worktree |
| Delete | Delete, with local, remote or both choices where they apply |
| `t d e` | External diff tool |
| `y n` and the commit copy keys | Copy the name, or details of the commit it points at |
| `w b` | Show only this branch or tag and its history, in Commits; a local branch with its tracking branch |
| `w v` | Show its commit in Commits |

Branches also: F2 rename; `g f` fast-forward the
selected branches; `g t` upstream options; `n t` new tag; `g p c`, `g p o`,
`g p v` create pull request, its options, open it; `y r` copy its URL; `g m n`
move commits to a new branch; `g y` git-flow; `t s o` sort order. Remote
branches: `g t` set as upstream, `t s o` sort order. Tags: `p` push the tag,
`n t` new tag.

### Other panes

| Pane | Keys |
| --- | --- |
| Remotes | `v`/Space branches; `f` fetch it; `n r` new remote; `n f` add a fork remote; `e` edit; Delete remove |
| Stash | `v`/Space files; `g s a` apply; `g s p` pop; Delete drop; F2 rename; `n w` new worktree |
| Worktrees | `v`/Space its commits; `s w` switch to it; `n w` new; `o` open in editor; Delete remove |
| Submodules | `v`/Space enter; `n s` new; `e` edit URL; Delete remove; `g u s`, `g u i`, `g u b` update, initialize, bulk options |
| Status | `s r` recent repositories; `s n` open a repository by path; `v`/Space focus the main view; `t a c` edit config; `t a u` check for updates; `t l n`, `t l p` all-branch logs |

### Prefix menus

| Group | Entries |
| --- | --- |
| `d` diff and merge | `w` working tree (HEAD in Files), `p` previous commit, `s` staged version (Files), `c` pasted commit; `m` merge tool (Files, conflicts view) |
| `s` switch and reset | `c` check out, `d` detached HEAD, `h` `m` `s` hard, mixed, soft reset, `p` previous branch, `w` switch worktree (Worktrees); in Status only `r` recent repositories and `n` open a repository by path |
| `n` new | `b` branch here, `t` tag, `w` worktree, `r` remote, `f` fork remote, `s` submodule |
| `c` cherry-pick and continue | `p` cherry-pick the selection, `c` continue |
| `u` undo | `u` undo, `r` redo |
| `y` copy | `n` name, `p` path, `f` absolute path, `c` file content, `h` hash, `i` short hash, `s` subject, `m` message, `b` body, `u` URL, `a` author, `t` tags, `d` diff, `w` all files' diff, `r` pull request URL, `y` confirmation text |
| `g` Git | `c` combine, `l` pull, `f` fast-forward, `t` upstream, `i` interactive rebase, `b` bisect, `o` merge/rebase options, `y` git-flow; `v f`/`v c` restore file, revert commit; `r p`/`r b` pick, rebase base; `m n` move commits to new branch; `e c`/`e n`/`e r`/`e a`/`e h` editor commit, commit without hooks, editor reword, commit attributes, edit hunk; `p c`/`p o`/`p v` pull request; `s s`/`s o`/`s a`/`s p` stash, stash options, apply, pop; `h s`/`h u`/`h t`/`h a`/`h o` stage, unstage, patch toggle, patch all, patch options; `h c`/`h p`/`h b` conflict options, pick hunk, pick both; `u s`/`u i`/`u b` submodule update, init, bulk; `z u`/`z w` upstream reset, working-tree reset |
| `t` view and tools | `d` diff display: `e` external diff, `f` diff with a chosen file, `c` comparison options, `w` whitespace, `n`/`p` renderer, `+`/`-` context, `>`/`<` rename threshold; `t` tree: `t` toggle, `c` collapse all, `e` expand all; `l` log: `n`/`p` all-branch logs, `o` log options, `c` command log options, `s` select branch commits; `s o` sort order; `a` app: `c` config, `u` check update, `s` suspend, `q` quit without changing directory, `g` game; `n`/`p` next/previous: `h` hunk, `f` file, `c` conflict, `d` diff pane, `j` jump to file; `h s`/`h r` hunk selection, range selection |
| `w` filters | `a` ancestors, `d` descendants of the selected commits (Commits); `b` branch / tag (Commits, Branches, Remote branches, Tags); `v` show in Commits (Branches, Remote branches, Tags, Reflog, Sub-commits); `s` file status (Files); from any pane `p` filter the log by path (prefilled with the selected file), `u` by author (prefilled with the selected commit's author), `c` clear the log filter (while one is active) |

## How behaviour differs from upstream

### Nothing happens on its own

- Automatic fetch, periodic file refresh, branch forwarding and automatic
  staging of resolved conflicts are off. Opening or switching a repository
  never fetches. Turning off automatic staging also turns off the automatic
  continue-merge prompt that follows it. Configuration can turn any of these
  back on.
- The commit log shows every branch's history (`git.log.showWholeGraph:
  true`); set it to `false` to show only the current branch.

### Comparing versions

- `d w`, `d p` and `d c` open the configured Git difftool for the selected
  commit, file or directory: against the working tree, against the previous
  commit (first parent for a merge), or against a commit whose full or partial
  hash you paste. They do not use or change the sticky diff mode (`t d c`).
- With no `diff.tool` (or `merge.tool`, which the difftool falls back to)
  configured, these comparisons and `t d e` first list the tools Git reports
  as available, instead of Git's "not configured" message and a guess. Choose
  one, then `Just this time` (the default) or `Save as Git's diff.tool in your
  global config, and use it`. Escape at either step launches nothing. A
  configured tool launches at once, with no menu.
- In Files, `d w`, `d s` and `d c` compare the working file with HEAD, the
  index or a pasted commit, base on the left. Untracked files have no base, so
  they are disabled.
- `t d f` compares a historical file with any file in the worktree that you
  choose, including untracked, ignored or renamed ones. The historical version
  is a temporary read-only copy, removed when the tool exits. The difftool
  should return 0 on success; detached GUI tools must wait for their window.

### Merge tool

- `d m`, in Files and in the conflicts view, opens the merge tool (`git
  mergetool`) on the selected conflicted files. The item Open external merge
  tool in `g h c` does the same. Delete/modify and add/add conflicts count. In
  Files, selected files without a conflict are ignored, and with no conflicted
  file selected `d m` does nothing. In the conflicts view it acts on the file
  being resolved.
- Git runs the tool without its per-file prompt. Git stages a file it considers
  resolved, as `git mergetool` does; a file the tool leaves unresolved stays
  conflicted. Git's `.orig` backups are not kept unless `mergetool.keepBackup`
  is set in Git's config; an explicit setting of either value is respected.
- With no `merge.tool` configured, the tools Git reports are listed first, as
  for the difftool above, and `Just this time` (the default) or `Save as Git's
  merge.tool in your global config, and use it` follows. Escape at either step
  launches nothing.
- Files refreshes when the tool exits, whether it succeeds or fails, so a file
  the tool resolved before it failed shows its new state.

### Branches and checkout

- `n b` creates a branch at the selection, or moves an existing one there
  after a warning showing the old and new commits, without touching HEAD, the
  index or files. Branches checked out in any worktree cannot be moved this
  way; use reset for the current branch.
- `s c` checks out without a second menu. A branch is switched to directly. A
  commit or tag is always checked out on a branch you name in a prompt (new,
  existing, or the current one), since several branches may point at it; `s d`
  gives a detached HEAD instead. Every checkout, including `s p`, carries
  uncommitted changes when Git can. Files that would be overwritten, including
  ignored ones, block the checkout instead. Nothing is stashed or forced.
- `s c` on a remote branch with no local branch of that name creates a local
  branch that tracks it and checks it out, in one step: if the checkout is
  refused (a local change would be overwritten, a merge is in progress, or the
  remote branch moved since you selected it), no branch is created.
- `s d`, and the detached route of `s p`, refuse while a merge, rebase,
  cherry-pick, revert, sequencer operation or bisect is in progress, as named
  checkout does. Finish or abort the operation first, so its state is kept.
- Other new branches show the source and a tracking choice (Git's configured
  policy, none, or track the source) before they are created.
- `s p` switches to the branch checked out before the current one. It reads
  that checkout from HEAD's reflog, so a tag with the same name does not
  change the result. If the previous checkout was a commit, `s p` detaches at
  that commit. If it is neither a branch that still exists nor a commit, such
  as a deleted branch, `s p` refuses and names it. Its menu entry names the
  branch.
- `u u` and `u r` undo and redo a checkout from HEAD's reflog. The target is
  the branch or commit the reflog recorded. An existing local branch is
  checked out as `s c` does it, and a commit is detached as `s d` does. Undo
  and redo ask once, naming the target; uncommitted changes are carried when
  Git can, nothing is stashed, and ignored files block the checkout. A target
  that is neither an existing local branch nor a commit ID is refused, and the
  message names it.
- `u u` and `u r` also undo and redo a commit or a rebase with a hard reset to
  HEAD's reflog target. Undo and redo ask once, naming the target. Tracked
  files must have no staged or unstaged changes; otherwise the action is
  refused, and nothing is stashed. Untracked or ignored files that the target
  tracks at the same paths are listed for approval, as `s h` lists them.
- `?` then "Checkout by name" (no key) asks for a name and reads it literally;
  it is never treated as a path, and nothing is stashed or forced. A local
  branch is checked out as `s c` does it, and wins over a tag or remote branch
  of the same name. `origin/topic` checks out that remote branch as `s c` does
  on it. A tag, or a commit ID or other revision that names a commit, is
  checked out detached as `s d` does. `-` is the previous checkout, as `s p`.
  A name that matches more than one ref (for example a remote branch and a tag
  both called `origin/topic`) is refused; enter the full ref, such as
  `refs/tags/origin/topic`. A full ref that does not exist is refused. Any
  other name offers to create a branch with that name from the selected branch.
- Deleting a branch offers local, remote or both, refuses branches checked out
  in any worktree, and asks before deleting unmerged history. For both, the
  remote branch is deleted first; a remote failure keeps the local branch.

### Filtering the commit log

- `w a` shows only the selected commits and their ancestors; `w d` shows only
  the selected commits and their descendants, from every branch. The graph is
  still drawn, the status bar names the filter, and the cursor stays on its
  commit. A filter replaces any other log filter, survives refreshes, and is
  removed by Escape (the cursor stays on its commit), or by choosing another.
  The filtered rows keep the order of the unfiltered log, with the other rows
  removed, so commits that were close together stay close together.
- `w b` (show branch / tag) shows only a ref and every ancestor of it. A
  local branch is shown with its tracking branch, so their shared history and
  both sides of a divergence are visible; a branch without one is shown alone,
  and the status bar says so. A remote branch or a tag is shown alone. It is
  offered on the selected ref in Branches, Remote branches and Tags (Commits
  opens on its tip), and on a commit in Commits that a branch, remote branch
  or tag points at: one ref applies directly, several are chosen from a menu
  (local branches, remote branches, then tags). The refs are looked up again
  on every refresh, so fetched commits appear.
- `w p` and `w u` filter the log by path or author from any pane, with the
  prompt prefilled from the selected file or the selected commit's author;
  `w c` clears whichever log filter is active. Upstream's filtering menu has
  no key; it is still in the palette.
- `w v` shows the selected branch's, tag's or entry's commit in Commits, from
  Branches (including the detached HEAD entry, whose commit is HEAD's), Remote
  branches, Tags, Reflog and Sub-commits. The cursor moves to
  the commit, and an explicit selection is kept (Escape clears it). A log filter
  that hides the commit is removed without asking; one that shows it stays. If the side panels are in
  normal mode, or the main views are enlarged, the side panels become half
  screen. When the log cannot show the commit at all (no branch, tag, or the
  ref the log walks reaches it, as with some reflog entries), nothing changes
  and a message says so. The log walks HEAD, or during a bisect the bisect's
  start ref, or its new ref when the start is not reachable from it.
- While the log is filtered, commands that rewrite history offer to leave the
  filter first, as upstream's path filter does. Unlike upstream, applying or
  clearing any log filter, including a path or author filter, never changes
  the screen mode.

### Selecting commits

- In Commits, Space toggles the commit under the cursor in the selection, so
  commits that are not next to each other can be selected together. While a
  selection exists, the cursor moves without changing it and is marked with
  `▶` in the gutter; the selected commits are highlighted.
- Shift with a movement key always selects: the commits the cursor passes
  while Shift is held join the selection when the cursor next moves without
  Shift.
- While a selection exists, `c p`, `g c`, `w a` and `w d` act on all the
  selected commits, including those that the log does not show. Commands that
  need one commit or a range (for example reword, drop, move, reset, copy,
  `w b`) are disabled until Escape clears the selection; mouse drags do not move
  commits then.
- The selection is kept by commit, so it survives refreshes and filters. A
  selected commit that a filter or the commit limit hides stays selected. The
  Commits subtitle counts the whole selection, as `N selected`, and adds
  `M not shown` for the members the log does not show. Clearing the filter
  shows them selected again. The main view shows the cursor commit's diff, or
  the whole range while a Shift run is the only selection.
- After a refresh, a selected commit whose object no longer exists is removed
  from the selection, and a notice names it. A selected commit that no branch,
  tag or HEAD reaches any more (rewritten or dropped outside the selection)
  stays selected, and a notice names it the first time it is found. Copy and
  cherry-pick still use it. `g c` refuses it if it is not in the checked-out
  branch's history, and names it.
- A Shift run that a drop, fixup or squash starts on does not join the
  selection, since the operation takes its commits.

### Rebase, merge, reset

- `r` rebases the checked-out branch onto the selected commit or branch;
  `m` merges the selected branch into it. The review names the current branch,
  the source and the commits involved, and offers the variants (simple,
  interactive or onto the base branch; regular or squash merge).
- Rebases need no staged or unstaged tracked changes. Untracked files are fine
  and Git refuses real collisions. Nothing offers or performs autostash, even
  if Git's configuration enables it, and other branches' refs are not updated.
- If a rebase could overwrite ignored files, a separate menu lists them before
  anything runs. This covers reword (in the message box or the editor), move up
  and down, drop, squash, fixup, squash fixups above, edit, amend-to, author
  edits, custom-patch moves and removals, and discarding changes from a commit. For amend-to and
  for placing a fixup, the review comes before the commit is created, so Cancel
  creates nothing. The approval is selected; choose Cancel or press Escape to
  stop. Cancel keeps a reword's message as a draft. The rebase review also lists ignored files where Git's rename detection may place replayed files.
- `g m n` moves the unpushed commits of the checked-out branch to a new
  branch. Its menu offers two variants: a new branch stacked on the current
  branch, or a new branch off the base branch with the commits cherry-picked
  onto it. On the main branch it asks to confirm the stacked variant only.
  The name prompt suggests a name, and the flow refuses before writing if:
  the name starts with `-`, is not a valid branch name, or names an existing
  branch; tracked files have staged or unstaged changes (commit or discard
  them first; nothing is stashed); or the branch, its upstream or the base has
  moved since the action started. Untracked files are fine. Nothing is
  stashed. The source keeps its commits until the new branch has them: the
  current branch is moved back to its upstream only after the new branch holds
  the commits. With the base variant, a conflict stops the cherry-picks, and
  the current branch is moved only after the operation is continued and the
  new branch holds one commit per moved commit; a skipped pick leaves the
  current branch untouched, and an abort does too.
- `s h`, `s m`, `s s` reset the checked-out branch to the selection, with a
  confirmation explaining what the mode does to the index and files.
- Conflicts and paused rebases are continued, skipped or aborted explicitly
  from `g o`. `c c` continues the operation in progress from any pane, and
  commits a resolved pick first if it is not committed yet. While any file is
  still unmerged, `c c` refuses and names the files, including a file whose
  conflict markers are gone but which is not staged; stage it first. While an
  operation is in progress, the status line shows `c c` as "Continue
  cherry-pick", "Continue merge", "Continue rebase" or "Continue revert",
  before `g o`'s hint. Without an operation, `c c` is faded and does nothing. Escape never
  aborts. Committing the resolution of a pick in a cherry-pick or revert
  sequence leaves that operation in progress, with skip unavailable.

### History editing

- `g c` combines the selected commits into one commit at the position of the
  oldest selected one (by ancestry). It refuses commits that are not in the
  checked-out branch's history, and names them. Commits in between that are
  not selected stay. The subject comes from that oldest commit and the editable
  description contains every selected commit's full message, oldest first.
  Merges cannot be crossed. This replaces squash and fixup.
- `c p` cherry-picks the selection of the pane it is pressed in: the commits
  selected in Commits, the Shift range in Sub-commits, or in Reflog. It shows
  the exact commits and destination before cherry-picking. A successful
  cherry-pick clears the selection. One that stops on conflicts keeps it while
  the operation is in progress, so an abort leaves it for a retry; continuing
  the operation to its end (`c c`, or the continue menu) clears it. Finishing
  the last pick with Enter does not clear it: Escape does.
  There is no separate copy step.
- Amending an older commit or creating stacked fixups refuses before
  committing if other tracked edits would remain for the rebase.

### Commit message box

The box for a commit, reword, combine or tag message does not step through
earlier commits' messages. Up and Down in its summary line step through the
worktree's cancelled drafts, and nothing you type is lost by doing so:

- Escape in the summary or the description saves the text as the newest draft
  if it is not blank and differs from what the box opened with. Quitting
  lazygitk with the box open saves it too. Drafts are kept per worktree, in its
  git directory, so they survive a restart. The newest 25 are kept.
- Up shows the next older draft, and Down the next newer one, then the text you
  were typing. Neither does anything at the end of the list. The box's subtitle
  says which draft is shown, as in "Draft 2 of 3". A draft equal to the text you
  were typing is skipped.
- Edits to a shown draft are kept when you step away from it or press Escape.
  Clearing all its text deletes it.
- Escape while a draft is shown keeps the draft and the text you were typing.
- Several lazygitk instances on one worktree share the drafts. Each change
  reads the file under a lock first, so a draft another instance saved is kept.
  A drafts file that is malformed is moved aside, to a file named
  `LAZYGITK_COMMIT_DRAFTS.corrupt-` followed by a timestamp, and the drafts
  start empty. A message says where the file went.
- A draft that cannot be saved keeps the box open with its text, and the error
  is shown. This applies to Escape, to switching to the editor (which does not
  happen), and to Enter on a submission whose result is reported later, such as
  a commit, reword, amend or patch move (which is not started). Up and Down do
  not move either: the shown text stays as it is.
- Quitting is not possible while the box has focus; close the box first. If
  lazygitk is closed from outside (for example, its terminal is closed) with
  the box open, it tries to save the text as a draft, and a failure is only
  logged.
- The lock files `LAZYGITK_COMMIT_DRAFTS.lock` and `command_usage.json.lock`
  stay on disk, empty, beside the files they guard. They are safe to leave.
- The description box still moves its own cursor with Up and Down.

### Sync

- **Push** pushes the selected local branch (or the checked-out one outside
  Branches) directly from the push menu, which shows the commit, remote,
  destination and options. A separate review appears only for replacing
  different tracking and for force-with-lease. A new branch with no push
  remote uses the repository's only remote without prompting.
- When the destination's remote-tracking branch, as last fetched, has commits
  the branch lacks, **Push this branch** is crossed out and says why; Force
  with reviewed lease remains. The check reads local refs only and never
  fetches. The force-with-lease review selects the push by default.
- **Fetch** (`f`) fetches every remote from any pane and never moves branches.
- **Fast-forward** (`g f`) moves each selected branch to its upstream as
  fetched, without fetching again and without prompts. Up-to-date branches are
  left alone; diverged ones, ones without an upstream and ones checked out in
  another worktree are listed in one alert at the end.
- **Pull** (`g l`) fetches first, then shows exactly what came in and the
  integration choice before changing files.

### Copying

- `y` opens one flat copy menu; each second key copies the same thing in every
  pane. Commit keys also work on branches, remote branches and tags, for the
  commit they point at.
- Copying writes the text with the OSC 52 terminal sequence, so it reaches the
  clipboard of the terminal you are looking at, also over SSH and in
  containers. Terminals without OSC 52 support (VTE-based ones) drop it
  silently; in tmux set `set-clipboard on`. An explicit
  `os.copyToClipboardCmd` still takes precedence.

### Repositories

- In Status, `s r` lists recent repositories and `s n` opens one by path.
  The `s n` prompt starts at the current
  repository's parent directory, lists matching subdirectories as you type,
  and Tab completes their common prefix. `~` and relative paths work.
- The terminal title shows the top-level folder of the repository or worktree
  you are looking at (a submodule's own folder inside it), with your home
  folder as `~`. It changes with every repository switch, and your terminal's
  previous title comes back when lazygitk exits, where the terminal supports
  saving titles. In tmux it becomes the pane title; `set-titles on` passes it
  to the outer terminal.
- On a repository switch (recent repositories, open by path, worktrees,
  submodules), every pane shows "Loading…" at once, so the old repository's
  content never shows under the new one. The panes fill together, and a toast
  "Opened <repo>" says the switch is complete. Keys pressed meanwhile run
  after it. If the switch fails, the current repository's panes come back
  with the error.

### Appearance

- The theme defaults to Catppuccin: Mocha on a dark terminal, Latte on a
  light one, switching when the terminal reports a change. Any theme field you
  set explicitly wins. Text that upstream colours with the terminal's blue
  (commit dates, reflog hashes, remote branch counts) uses the palette's Blue,
  which stays readable on a black background. The focused pane has a heavy
  frame.
- Diffs render through delta when it is on `PATH`, with Git's own diff one
  `t d n` away. Without delta, Git's diff is used and a notice says so once.
  Notices appear one at a time, each for two seconds, in this order: unreadable
  usage counts, ignored key settings, then missing delta.
  The default delta command draws file names, hunk headers and line-number
  columns in the palette's Blue instead of the terminal's. On a dark terminal
  with true colour, changed lines get dark green and red backgrounds, a little
  brighter for the changed words, so that syntax colours such as comments stay
  readable; with only 256 colours, delta keeps its own. A delta setting in
  your git config (`[delta]` or a delta feature) wins.
- lazygitk uses true colour when `COLORTERM` says the terminal has it, or
  when the terminal says so itself at startup, as Windows Terminal does
  inside WSL, where `COLORTERM` is usually unset.
- The popup on first run only says that lazygitk is a fork of lazygit, and
  how to thank lazygit's author through the donate link.

### Configuration and installation

- lazygitk keeps its own configuration and state: `lazygitk/` under the
  config and state directories, and `.git/lazygitk.yml` or `.lazygitk.yml` in
  repositories. Upstream lazygit's files are never read or written, and nothing
  is migrated. `CONFIG_DIR` and `LG_CONFIG_FILE` work as upstream.
- `scripts/build_dist.sh` (or `just dist`) and `scripts/build_dist.ps1` build
  portable `lazygitk` binaries into `dist/`.
- lazygitk never checks for or installs updates; upstream's updater would
  replace it with lazygit. `t a u` says so. Rebuild from source to update.

## Coming from upstream lazygit

| Upstream | lazygitk |
| --- | --- |
| `P` push, `p` pull | `p` push, `g l` pull |
| `R` refresh, `r` rebase | F5 refresh, `r` rebase |
| Enter to inspect | `v` or Space; Enter commits only in Files |
| Left/Right or Tab between panes | `h`/`l` or numbers; Tab changes tabs |
| `n`/`N` search results | F3 / Shift+F3 |
| `d` delete, `R` rename, `r` reword, `c` commit | Delete, F2, F2, Enter in Files |
| Space to check out a branch | `s c` |
| `-` previous branch | `s p` |
| `c` checkout by name | "Checkout by name" in `?`; normally select the branch and `s c` |
| Squash and fixup keys | `g c` combine |
| Up and Down in the commit message box step through earlier commits' messages | Up and Down step through the cancelled drafts; see the commit message box above |
| Copy (`C`) and paste (`V`) commits for cherry-pick | `c p` picks the selection; the selection is the copy |
| `z` undo, Ctrl+Z redo | `u u`, `u r` |
| Ctrl+J / Ctrl+K move commit | `>` / `<` |
| `y` copy menus with submenus | Flat `y` menu |
| Space or Enter to switch worktree | `s w` |
| Enter or Space to enter a submodule | `v` or Space in Submodules; `v` on it in Files |
| Enter to resolve a conflicted file | `v` |
| Space / `b` / `z` in the merge view | Space / `g h b` / `u u` |
| `{` / `}` diff context | `t d -` / `t d +` |
