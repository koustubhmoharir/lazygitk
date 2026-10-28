# lazygitk design principles

lazygitk is a personal fork of lazygit that fixes usability problems with the
upstream interface. These principles govern its current behaviour; check every
new change against them. The keys and behaviour they produce are described in
the [user guide](Guide.md); the code seams are in the [merge guide](MergeGuide.md).

When a design question comes up, first look at how TortoiseGit handles it
(its behaviour, documentation or source). Choices that these principles do not
settle are put to the user before implementation, and the decision is then
recorded here or in the user guide.

## The user stays in control

- **Explicit actions, not automation.** Automatic fetch, periodic file
  refresh, branch forwarding and automatic staging of resolved conflicts are
  off by default. Opening or switching a repository never fetches or moves
  branches. Read-only polling of external ref changes stays on. These are
  defaults, not prohibitions: explicit configuration still turns them on.
- **Meaningful choices are presented, not made silently.** Where an operation
  has real alternatives, a menu or confirmation offers them with a sensible
  default (Cancel, for risky ones). The application never quietly picks one.
- **No redundant reviews.** When a menu already shows the source, destination
  and options, choosing its action runs it; a second confirmation is shown
  only when something new is at stake (replacing tracking, force-with-lease).
- **Labels name what will change.** Labels and confirmations distinguish the
  selected object, the checked-out branch and the operation's destination,
  and explain the consequences (history rewrite, files touched, reset mode).
- **Destructive changes always confirm.** Discarding changes asks every time,
  even when an upstream setting would skip the warning. The one exception is
  `x` in the staged diff, which only unstages.

## Operations act on what was selected

- **Selection is captured when an action starts.** The commit, branch or file
  selected at that moment is the target through every prompt and worker; a
  later selection change cannot redirect it. If the repository or target
  changed before the operation runs, the flow asks again instead of guessing.
- **The selected object, not the checked-out one.** Push, fast-forward,
  rename, delete, placement and comparisons act on the selected ref or commit.
  Actions on the current branch say so in their label.
- **Never switch the checkout to simulate an operation.** Operations that
  would need another worktree (merging or rebasing into a branch that is not
  checked out) are not offered; the user checks out explicitly.

## Git does the work, and its answer stands

- **Native Git, normal concurrency.** Operations use one native Git command
  or ref transaction where possible, recheck state just before writing, and
  rely on Git's own locks. The fork adds no locks, retries or assumed
  rollback.
- **Refuse rather than guess.** Ambiguous input, unsupported topology or an
  occupied worktree is refused with the reason. Git's own error is shown when
  it rejects an operation, and the panes refresh to the actual state.
- **Commit first; never stash implicitly.** No flow stashes on the user's
  behalf or offers autostash. Rebases require no tracked changes (harmless
  untracked files are fine, and Git rejects real collisions). Checkouts carry
  dirty files when Git can, and otherwise report the files in the way.
- **Ignored files are protected.** Where an operation could overwrite ignored
  files, the user approves that risk explicitly, with Cancel as the default.

## Keys

- **Fixed keys.** Keybindings are not configurable. Upstream's key settings
  overlap once letters are case-insensitive, so they are ignored with a
  startup notice.
- **One meaning per key path.** A key or path means the same verb in every
  pane; only the selected object differs. Unrelated operations never share a
  path. Where a pane has nothing for that verb, the key does nothing: no
  fallback to another binding, no error toast.
- **Letters are case-insensitive; punctuation is literal.** `p` and `P` are
  the same key. Shift+J and Shift+K are the one exception: they extend the
  selection like Shift+Down and Shift+Up, since `j` and `k` move it.
  Shifted punctuation is its own key, so `<` and `>` move commits while `,`
  and `.` do nothing.
- **No Ctrl for core actions.** The only Ctrl keys are editor controls inside
  modal text fields.
- **Prefix groups.** The first key of a path opens a visible menu and never
  changes anything by itself; the next key picks an item or a nested menu.
  There is no hidden chord state or timeout. Escape steps back one level.
  Menu entries appear in the keymap's declaration order. The groups are:
  `d` diff, `s` switch and reset, `n` new, `c` cherry-pick marks, `u`
  undo/redo, `y` copy, `g` less common Git operations, `w` filters, and `t`
  view, navigation and app tools (nothing under `w` or `t` changes the
  repository). `/` stays the pane's own text filter or search.
- **Short keys for frequent or repeated actions.** Actions used often, or
  pressed several times in a row, get single keys; occasional ones get two-
  or three-key paths. Deliberate friction is kept where it prevents surprises:
  pull has no single key, and moving commits needs the shifted `<` / `>`.
- **Shared roles.**
  - Space toggles the selection where the pane edits something (staging in
    Files and in the diff, patch inclusion, conflict-hunk picking) and
    inspects elsewhere.
  - `v` inspects or drills down: a branch's commits, a commit's files, a
    file's diff, a submodule.
  - Enter commits, and only in Files.
  - `/` opens the pane's text tool: search where the pane has search, filter
    where it has a filter.
  - Escape leaves the current layer or mode; it never aborts a Git operation
    or discards files.
- **Everything is discoverable.** Every action is in the `?` palette with its
  full key path, including actions without keys. The palette orders commands
  by how often they are run from it: using a shortcut lowers a command, so
  learned shortcuts sink and the rest rise.

## Appearance

- Theme colours default to Catppuccin (Mocha on dark terminals, Latte on light
  ones); explicitly configured theme fields win. The focused pane has a heavy
  frame.
- Diffs render through delta by default when it is installed, with Git's own
  diff as the next renderer.

## Coexistence with upstream

- lazygitk keeps its own configuration and state (`lazygitk/` directories,
  `lazygitk.yml` repository files) and never reads or writes upstream
  lazygit's.
- Fork code lives in additive modules with small, commented integration points
  so that upstream merges stay feasible. Where fork code replaces an upstream
  path, the integration point says what it replaces and why, and the merge
  guide records it.
