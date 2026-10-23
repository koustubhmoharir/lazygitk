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

## Check out a commit under a branch name

Select one commit in local history, another branch's history, or reflog. Open
**Checkout** and choose **Checkout as branch…** (`b` inside that menu). Enter a
literal local branch name; suggestions include existing local branches.

A new name creates and checks out the branch. An existing different tip needs
a warning showing its old/new commits and the current checkout. An already
matching branch is checked out without a tip-movement warning. The current
branch can also be moved and checked out with this workflow. Cancel changes
nothing; destination or HEAD changes before execution require fresh review.
Existing upstream settings are preserved.

Sparse-checkout settings stay in effect. Submodule working trees are updated
only according to Git's recursion configuration; dirty nested files are not
forced over. With recursion disabled, Git may leave a submodule at its existing
commit while the parent index records the selected commit's gitlink.

There is no blanket clean-tree requirement: Git carries staged, unstaged and
untracked changes when it can preserve them. Files are never forced or stashed.
An ignored file that would be overwritten also blocks checkout. Commit changes
or move the blocking files yourself, then retry. Invalid names, active operations,
occupied worktrees, locks and hooks can independently prevent completion; finish or abort an active
merge/rebase/cherry-pick/revert/bisect first. Symbolic branch aliases are refused.

This uses Git's normal concurrency model. The destination and HEAD are checked
again immediately before starting; coordinate other ref-mutating Git tools
while checkout runs. Do not expect ref-only placement's atomic expected-old
protection during combined checkout.

Git can report failure after changing HEAD, the index or files, particularly
when a hook fails. The error is preserved and the actual state refreshed; no
automatic retry or rollback is attempted. Failure keeps the selected history
target and reflog position; review the result before retrying.
Existing detached-head checkout and creation/checkout workflows remain separate;
the new named workflow does not use their autostash offers.

## Rebase the current branch onto a selected commit

In local history, another branch's history, or reflog, select one commit and
open `?` → **Rebase current branch onto this commit…**. The action is initially
unbound; configure `keybinding.commits.rebaseOntoSelectedCommit` if desired.
Choose ordinary rebase (linear history) or preserve merge topology, then review
the current source branch/tip and selected target before confirming.

This rebases the checked-out branch, not the branch containing the selected
commit. It does not use or clear a marked base. An ancestor target can be a
no-op and a descendant can fast-forward. The selected target is captured;
changing selection or a branch name cannot silently substitute another commit.

Staged/unstaged tracked changes must be committed first. Historical amendments
and stacked fixup/amend placement refuse before creating a commit when other
tracked edits would remain for the rebase. There is no automatic stash or
fixup-only fallback; plain fixups without a rebase remain available.
Harmless untracked files remain allowed; Git rejects collisions. Neither this
action nor other fork-owned rebase starts offer or invoke autostash, even when Git config enables
it. Ordinary commits and conflict continue/skip/abort retain their workflows.
The new action suppresses automatic updates to additional branch refs.

Ignored files are different: native Git can overwrite them. If the target or
potential replay paths may collide with ignored files, a separate warning
lists the paths. **Cancel is selected by default.** Move/back up files first,
or explicitly select **Approve ignored-file overwrite risk and rebase**.
Scroll through the quoted file rows to inspect the complete list. Enter on a
file row only reopens review; it never approves. The scan is conservative, so
not every listed file necessarily changes, including in no-op cases.

Approval covers the reviewed source and affected-file snapshot. A changed
branch/tip, risk list, file type/mode or contents requires fresh confirmation
and approval. Unrelated ignored files do not require approval. This is a
pre-launch check, not a file lock: coordinate other Git writers while rebase
runs. Hooks and Git can report failure after changing state; actual state is
refreshed without automatic retry or assumed rollback. Resolve conflicts and
stage explicitly, then use normal continue/skip/abort options. Interactive
signing uses the terminal subprocess lifecycle.

## Rebase current onto a selected branch

Select a local or remote branch and open **Rebase current onto selected branch…**
in `?`. Choose simple rebase, interactive rebase, or rebase onto the detected
base branch. Interactive rebase stops at the beginning so you can edit the todo
list in the commits pane. An existing marked base defines the exclusive replay
range. Review the exact current branch, source commit and range before starting.

The selected branch is a source; the checked-out branch is the destination.
Changing its source tip, current HEAD or the marked base while reviewing
requires a new review. Native rebase runs through the terminal, including signing, then
returns to the UI. Only the current branch moves; automatic updates to other
branches are disabled. Tracked edits must be committed first. Harmless untracked
files remain allowed, and ignored-file overwrite risk requires separate approval
with Cancel first. The flow does not offer or invoke a stash.

The original marked base is consumed once Git starts. Conflicts offer inspection
or explicit abortion of that rebase. Signing failures and commits that become
empty stay paused: inspect the pending state, then use **Rebase options** to
continue, skip or abort. LazyGit does not automatically skip an empty replay in
this flow. Native failures retain their actual partial state without automatic
retry or rollback.

## Reset current HEAD to a selected source

The reset menu accepts commits, tags and local or remote branches. Soft, mixed
and hard reset affect **current HEAD**; selecting another branch does not reset
that branch. The review shows the current checkout and exact source commit.
Changing either before execution requires a new review. Annotated tags resolve
to their commit; upstream sources use Git's configured tracking namespace.

Hard reset always asks for explicit approval. It discards tracked changes and
can overwrite obstructing untracked or ignored paths, even when tracked files
are clean. Cancel preserves the repository. Mixed/hard reset can explicitly
abort a merge in this worktree. A tip-changing reset refuses a branch reserved
by another worktree. Native failures are reported and actual state refreshed
without automatic retries, stashing or rollback.

In the branches pane, **Upstream options** shows the selected branch and Git's
exact upstream tracking ref. Divergence works with custom tracking namespaces
and local `.` upstreams. Missing upstreams disable dependent actions while
set/unset remain available to repair tracking metadata. Reset and rebase are
explicitly actions on the checked-out branch; they check the reviewed source,
checkout and upstream again before opening their submenus.

## Delete selected branches

The branches pane offers local, remote and both deletion. Reviews show exact
source refs/commits and current HEAD. Local deletion refuses branches checked
out or reserved by any worktree; it never offers to detach or remove that
worktree. Explicit removal from the Worktrees pane remains a separate action.

Unmerged history needs approval. A ref being deleted, including a remote cache
removed by **both**, cannot count as proof that its history will remain
reachable. Remote deletion shows the named remote and full server branch refs;
ambiguous tracking mappings require a choice. Missing tracking metadata can be
replaced by an explicit configured remote and full branch ref.

A remote cache is not proof of the server's current tip: confirmation permits
deleting the named server branch even if it has newer commits. For **both**,
remote deletion runs first; any remote error keeps the local branch. Multiple
push URLs can partially succeed. Errors retain Git's actual state without
retrying or rolling back earlier deletions. Credential prompts remain usable.

## Command palette identity

Equivalent invocation paths share an entry with combined shortcut hints.
Different actions are not merged just because their translated labels match.
Actions with no shortcut remain executable through `?`. Usage-based ordering
is a later roadmap milestone, not yet enabled.
