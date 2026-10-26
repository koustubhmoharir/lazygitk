# Fork keyboard controls

Press `?` to discover commands and their complete paths for the focused pane.
A path such as `d w` means press `d`, then choose `w` in the visible menu.
Letters have the same meaning in either case. Inapplicable default commands do
nothing. Escape exits the current layer; within a nested command menu it goes
back one level. Menus use physical Up/Down, Enter and Escape for navigation.

| Keys | Action |
| --- | --- |
| `d w`, `d p`, `d c` | Compare selected history with the working tree, previous commit, or a pasted commit hash |
| `p`, `f` | Push the labeled selected/current branch or fetch |
| F2, `e`, `o` | Rename/reword, edit, open |
| F5 | Reload all repository state and redraw; no fetch |
| Delete, `x` | Delete/drop; discard with confirmation |
| `a`, `r`, `m` | Amend; review rebase; review merge |
| `s h`, `s m`, `s s` | Review hard, mixed, soft reset of the current branch |
| `s c` | Checkout selected branch/commit, with branch naming and move choices |
| `s r`, `s n`, `s w` | Switch to a recent repository; open a repository by path; switch to the selected worktree |
| `n b` | Create/move a named branch at the selection without checkout |
| Enter | Commit in Files; no action in other normal panes |
| Space | Toggle staging in Files; inspect elsewhere |
| `v` | Inspect details; enter the selected submodule in Submodules and Files |
| `i` | Ignore a working path |
| `c y`, `c x`, `c p` | Mark, unmark, review cherry-pick |
| `g c` | Review and combine selected commits; edit the combined message |
| `u u`, `u r` | Undo, redo |
| `y` | Copy menu: `y` then a letter for what to copy (see below) |
| `g l` | Deliberately open pull |
| `/`, F3, Shift+F3 | Search, next result, previous result |
| `h`, `l`, numbers | Previous/next pane, numbered pane |
| Tab, Shift+Tab | Next/previous tab |
| `:`, `q` | Shell command, quit |

Files Left collapses a directory or moves to its parent. Right expands a
directory or enters its first child. Select the tree root and press Space to
stage or unstage all files. In the staged built-in diff, `x` retains upstream's
immediate unstage behavior, with no confirmation. Text inputs keep normal
typing and editing controls, including Ctrl+Left/Right word movement. Ctrl
shortcuts are available for custom commands rather than default core actions.
The only Ctrl defaults are modal editor controls: the commit description
submits with Ctrl+S or Ctrl+Enter (Cmd+Enter on macOS), Ctrl+O opens the commit
editor menu, and Ctrl+O copies a confirmation's text.

Combine supports linear history, including nonadjacent selected commits while
preserving intervening commits. It refuses merge crossings. The first selected
ancestor supplies the subject; every selected full message appears in the
editable description, ancestor first. Git's normal continue/abort flow applies
if rewriting stops for a conflict or hook failure.

## Command palette

`?` lists local and global commands together, sorted by usage. Running a command
from the palette adds one to its count, and running it from its shortcut
subtracts one, because you already know that shortcut. Commands you use most
come first, and never-used commands keep their declaration order. Counts may go
negative, and they are shared by all repositories. The order is fixed while the palette is open. Each row ends with
a dim `local` or `global` tag. Typing filters by command name only. Navigation
commands stay at the bottom of the palette in their own section.

Choose "Reset command usage counts" to clear all counts after confirmation.
Set `gui.commandPaletteOrder: static` to show Local, Global and Navigation
sections in their fixed order instead.

## Configuration and migration

Keybindings are fixed: lazygitk ignores the `keybinding` section,
`commandBindings` and custom-command `key` values in every config file, and
shows a startup notice naming each file that contains them. Upstream's setting
names overlap once keys are matched without case (for example `p` pull and
`P` push), so they cannot be honoured safely. Custom commands stay available
from `?`.

The effective keys are in the [generated command tables](Keybindings_en.md).
Coming from upstream lazygit, these are the main differences:

| Previous upstream shortcut | Fork default |
| --- | --- |
| `P` push / `p` pull | `p` push / `g l` pull |
| `R` refresh / `r` rebase | F5 refresh / `r` rebase |
| Enter inspect | `v` or Space; Enter commits only in Files |
| Left/Right or Tab pane switching | `h`/`l` or numbers; Tab changes tabs |
| `n`/`N` search results | F3 / Shift+F3 |
| Separate squash/fixup shortcuts | `g c` combine |
| Toggle cherry-pick marking | `c y` mark / `c x` unmark; Escape in a history pane clears marks (leaving a subview keeps them) |
| Enter to resolve a conflicted file | `v` opens its resolution view or dialog |
| Enter or Space to enter a submodule | `v` or Space in Submodules; `v` on a submodule entry in Files |
| Enter or Space to switch worktree | `s w` |
| Space / `b` / `z` in the merge view | `g h p` pick hunk / `g h b` pick both / `u u` undo |

The history graph includes all refs by default. Set
`git.log.showWholeGraph: false` to restrict it explicitly.

## Copy keys

`y` opens one copy menu, and each second key means the same thing in every
panel that offers it. Commit keys work in Commits, Reflog and Sub-commits, and
on the commit a selected branch, remote branch or tag points at.

| Keys | Copies |
| --- | --- |
| `y n` | Name (branch, tag, submodule, file) |
| `y p`, `y f` | Relative, absolute file path |
| `y c` | File content at the commit (commit files) |
| `y h`, `y i` | Full, abbreviated commit hash |
| `y s`, `y m`, `y b` | Commit subject, full message, message body |
| `y u`, `y a`, `y t` | Commit URL, author, tags |
| `y d` | Diff of the selection: commit, file, or selected diff lines |
| `y w` | Diff of all working-tree files (Files) |
| `y r` | Pull request URL (local branches) |
| `y y` | Confirmation popup text |
