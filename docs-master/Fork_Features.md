# Fork features

This fork is under development. [Roadmap](../new-features/Roadmap.md) and
[current work](../new-features/Tracker.md) record what is complete.

## Explicit actions by default

Automatic fetch, periodic file refresh, branch forwarding and automatic staging
of resolved conflicts default to off. Read-only external-ref polling stays on.
Manual actions still work, and explicit user/repository configuration overrides
the defaults. Turning off automatic conflict staging also turns off its
associated automatic continue-merge/rebase prompt.

## Historical comparisons

Select one commit in local history, another branch's history, or reflog:

- `w` or `W`: **Diff with working tree**.
- `v` or `V`: **Diff with previous commit** (first parent for a merge).

The same keys work on a file or directory in that commit's file list. These
are separate actions, also discoverable in `?`, and do not depend on sticky
diff mode. The previous-commit action is unavailable for a root commit or an
unavailable first parent. Commit ranges/placeholders are not comparison targets.

For an unrelated or renamed working file, select the historical file, open `?`
and choose **Diff with working file…**. Enter a path relative to the worktree
root, or an absolute path inside it. Suggestions include filesystem files,
including ignored/untracked files; directory suggestions can be edited to
browse further. Only an existing regular file can be confirmed. The prompt
names the historical source, which stays captured while you choose a target.

The configured Git difftool is used. Whole-commit/directory comparisons follow
Git's tracked-file behavior. Single-file working comparisons use the actual
file's bytes, even when Git now considers it untracked or ignored. No staging,
restore or stash is needed. A historical deletion has no selected-commit blob
for a working-file comparison; use the previous-commit action to view deletion.
Symlink/gitlink blobs and symlink destinations are not supported by the explicit
single-file workflow; it reports a reason rather than dereferencing them.

Historical sources are read-only temporary files and are removed after the
tool returns, including failure. Configure detached tools to wait until their
window closes. If the tool edits the real working file, that is the user's tool
action; lazygit refreshes afterward. File-pair comparisons trust the tool's exit
code: the tool should return zero on success. Git's normal difference result
is accepted, but tool failure is not silently ignored. A tool that returns a
nonzero success code needs an explicit adapter.

The provisional diff keys displace upstream worktree creation, sticky-diff
menu, range-selection toggle, paste-commits and skip-hook/WIP shortcuts. Those
commands remain in `?`; the complete keymap milestone will assign their new
shortcuts. Explicit configured mappings still load. The new config fields are:

```yaml
keybinding:
  universal:
    diffWithWorkingTree: [w, W]
    diffWithPreviousCommit: [v, V]
    diffWithWorkingFile: []
```

## Create or move a branch without checkout

Select one commit in local history, another branch's history, or reflog. Open
`?` and choose **Create/move branch here…**, then enter a literal local branch
name. Existing names are suggested; no name is automatically chosen or cleaned
up. This action is initially unbound (`keybinding.commits.createMoveBranch`).

A new name creates a branch without checkout. An existing different tip needs
a warning showing its old/new commit IDs; cancel leaves it unchanged. If the
branch changes while the warning is open, review and confirm the new state
again. An already matching tip reports that nothing changed.

HEAD, staged changes, working files and the selected commit stay unchanged.
Existing upstream settings are preserved. A checked-out/reserved branch cannot
be moved: use explicit Reset for the current branch and finish any active
operation first. Another worktree's reservation is never bypassed. Symbolic
branch aliases are rejected. **Create branch and check it out…** remains a
separate action with its existing behavior.

Creation/movement records a descriptive destination-branch reflog entry. The
existing Undo action does not cover independent branch placement; inspect that
branch's reflog for explicit recovery. Ref changes are expected-old transactions;
occupancy is freshly checked, not locked against arbitrary external checkouts.
An uncertain Git transaction failure is reported, never silently retried.

## Command palette identity

Equivalent invocation paths share an entry with combined shortcut hints.
Different actions are not merged just because their translated labels match.
Actions with no shortcut remain executable through `?`. Usage-based ordering
is a later roadmap milestone, not yet enabled.
