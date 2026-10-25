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
| `n b` | Create/move a named branch at the selection without checkout |
| Enter | Commit in Files; no action in other normal panes |
| Space | Toggle staging in Files; inspect elsewhere |
| `v` | Inspect details |
| `i` | Ignore a working path |
| `c y`, `c x`, `c p` | Mark, unmark, review cherry-pick |
| `g c` | Review and combine selected commits; edit the combined message |
| `u u`, `u r` | Undo, redo |
| `y` | Copy menu |
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

## Configuration and migration

Use stable command IDs for new overrides:

```yaml
commandBindings:
  diff.commit.worktree: [[d, w], '<f6>']
  diff.history-file.worktree: [[d, w]]
  app.refresh: ['<f5>']
  branch.pull-current: []
  branch.pull-selected: []
```

The outer list contains alternatives; each inner list is a sequence of one to
three keys. Empty or null removes keyboard paths while keeping the action in
`?`. Unmodified Enter, Escape, Up and Down are reserved after the first key for
prefix-menu controls. Mouse events cannot be command paths.

Specific command IDs take precedence over a shared route override, explicit
legacy `keybinding` fields, then fork defaults. Explicit local routes win over
explicit global ones; explicit routes win over defaults. An explicit single-key
action displaces default continuations below that key. If both the leaf and its
continuations are explicit, a visible menu presents the choice. Custom command
roots retain native exact matching and take precedence over core paths.

The generated configuration's legacy `keybinding` values support compatibility;
they do not describe the effective normal-pane defaults. Copying the entire
legacy table makes its entries explicit overrides. Keep only intentional
overrides, and consult the [generated command tables](Keybindings_en.md) for
the effective default paths. Explicit overrides can restore upstream actions
that have no default shortcut in this fork, including squash/fixup.

| Previous upstream shortcut | Fork default |
| --- | --- |
| `P` push / `p` pull | `p` push / `g l` pull |
| `R` refresh / `r` rebase | F5 refresh / `r` rebase |
| Enter inspect | `v` or Space; Enter commits only in Files |
| Left/Right or Tab pane switching | `h`/`l` or numbers; Tab changes tabs |
| `n`/`N` search results | F3 / Shift+F3 |
| Separate squash/fixup shortcuts | `g c` combine |
| Toggle cherry-pick marking | `c y` mark / `c x` unmark; Escape clears marks |

The history graph includes all refs by default. Set
`git.log.showWholeGraph: false` to restrict it explicitly.
