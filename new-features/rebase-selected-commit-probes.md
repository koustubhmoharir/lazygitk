# Selected commit rebase Git probe

Probe: `go run new-features/probes/rebase_selected_commit.go` (standalone,
build-ignored Go file). It creates disposable repositories under one temporary
directory, disables system/global Git config and hooks, and removes that
directory on exit. All repository changes are confined to those disposable
repositories; the repository under development is not modified. Observed Git version:
`git version 2.51.1`.
The command runner removes inherited `GIT_*` variables before supplying its
own isolated environment, so a caller's repository/index location cannot
redirect the disposable commands. Git diagnostics use the C locale.

The tested command form was:

```text
git -c rebase.updateRefs=false rebase --no-autostash <fullOID>^{commit}
```

## Observations

- A divergent selected target rebased the checked-out `main` successfully.
  The selected target ref remained unchanged. An ancestor target and a root
  target already contained in `main` both returned success with “up to date”
  and did not move the branch. Rebasing a branch at an ancestor onto its
  descendant fast-forwarded it to that descendant.
- With `rebase.autoStash=true`, staged and unstaged tracked changes caused
  rebase to fail with “cannot rebase”/“index contains uncommitted changes”.
  HEAD, refs, index tree and raw index-file hash, and a hash over the bytes of
  all worktree files were unchanged; the stash list stayed empty.
  The explicit `--no-autostash` therefore suppressed the config opt-in.
- A harmless untracked file survived a successful rebase. An untracked file
  colliding with a path introduced by the target caused Git to reject the
  rebase before changing HEAD/refs or the file. This does not establish a
  universal outcome for all path collisions, but the agreed feature behavior
  is to allow harmless untracked files. Git rejects this colliding case without
  changing HEAD, refs, the index, or file bytes. Staged and unstaged tracked
  changes remain disallowed.
- An ignored file at a target-created tracked path was overwritten by the
  ordinary rebase command: its bytes changed from `ignored user bytes\n` to
  `topic\n`; rebase succeeded. The probe also records HEAD, refs, index tree,
  raw index hash, and all worktree file bytes before and after. Unlike the
  named-checkout path, this rebase invocation does not use
  `--no-overwrite-ignore`. This is evidence for a safety guard/design question,
  not a policy choice made by the probe.
- With `rebase.updateRefs=true`, another local branch at a rewritten commit
  moved with the rebase. A command-scoped `-c rebase.updateRefs=false` kept
  that branch at its original OID while `main` moved. A conflicting rebase
  fixture had two replay commits, with the additional branch at the first
  (interior) commit and a conflict on the later commit. After resolving and
  staging, plain `git rebase --continue` succeeded, and the additional branch
  still pointed to the original interior commit despite the repository config
  remaining `rebase.updateRefs=true`.
- Against a genuinely divergent target, default rebase completed with zero
  merge commits (`git rev-list --count --merges HEAD`); adding
  `--rebase-merges` retained at least one merge commit with a two-parent shape.
- A ref whose name was exactly the selected full 40-hex OID could be created,
  and Git warned that the name was ambiguous. The expression
  `<fullOID>^{commit}` resolved to the object OID, not the shadow ref's target;
  assertions checked both the resolved OID and that the target became an
  ancestor of the rebased HEAD. The rebase completed against the selected
  object. The ref's deliberately ambiguous name emitted Git warnings.

The probe asserts expected success/failure, branch movement, ref preservation,
conflict completion, merge counts, OID selection, and unchanged state for the
rejected dirty and untracked-collision cases. State output includes HEAD, local
refs, status, index tree, raw index hash, and a worktree-file byte hash. The
ignored collision prints its exact before/after bytes.

## Limits and unresolved points

These observations are from one Git version and one platform. They do not
establish minimum-version support, every merge/rebase backend, linked-worktree
ref behavior, hooks, alternate object formats, or every conflict/resolution
shape. The merge topology check records the default and explicit option
results; it does not recommend whether the UI should expose an optional
preserve-merges choice. That choice should follow the TortoiseGit audit and
the user-facing design review. The untracked-file behavior is settled: allow
harmless untracked files, while staged or unstaged tracked changes block the
operation. The ignored-file overwrite result remains a safety design question;
this probe records Git's behavior but does not choose the guard or user-facing
policy.
