# Rebase current branch onto a selected commit

Design under review for roadmap item 6. No completed-feature claim is made here.

## Meaning and modal flow

The new action captures one full selected commit OID and the repository before
opening a modal. It operates on the checked-out local branch, not on whichever
branch contains the selected commit. Preparation snapshots that branch's full
ref and tip. The confirmation names the source branch/tip and target, explains
history rewriting, and makes cancellation available. A changed source branch
or tip requires fresh review; the selected target never follows a moving ref.

Use ordinary explicit-upstream rebase semantics, not the upstream helper's
interactive, preserve-merges engine. A marked base is unrelated and must not
be read, cleared, or silently used as an `--onto` boundary. Existing advanced
marked-base commands remain distinct. An ancestor/root target can legitimately
be a no-op; a descendant can fast-forward the branch. Do not force a rewrite.

The ordinary action explicitly disables autostash, automatic additional-ref
updates and implicit fork-point/merge-preservation configuration. Merge
preservation is a meaningful separate modal choice, not a hidden consequence
of reusing upstream's builder. The eventual UI and tests must make the chosen
topology visible before execution.

This follows [TortoiseGit's rebase dialog](https://tortoisegit.org/docs/tortoisegit/tgit-dug-rebase.html)
as a usability guide: expose the source, destination and rewrite implications.
Its branch-swapping workflow is not permission to rebase an unoccupied branch
without a worktree in this milestone.

## Shared start policy

The clean-state check belongs at rebase start, not in the generic execution
runner or continue/skip/abort handlers. Interactive editing, fixup squashing and
historical-patch operations use the same check. Commands explicitly pass
`--no-autostash`; disabling a config default alone is insufficient. Return a
commit-first error without offering stash. Remove the patch-extraction flow's
automatic push/pop and its stash confirmation, while retaining manual stash
commands and unrelated checkout behavior.

Changing the interactive builder to return `(command, error)` lets a failed
precondition stop both normal execution and editor subprocesses. A small
prepare-and-run wrapper keeps ordinary callers concise. Do not represent a
failed precondition with a fake shell command or ignore a failed status query.
Do not automatically repeat a rebase command after a lock-looking hook error;
the command may already have changed the repository.

Pull is also a rebase-start path when Git configuration enables it. Audit
`branch.<current>.rebase` precedence over `pull.rebase`, including the target
worktree context used by existing forwarding flows. Preserve merge/fast-forward
behavior when pull does not rebase. Do not use the application's cached config
to promise a fresh pre-launch policy decision. Git remains the authority for
parsing its configuration and for runtime errors.

Tracked/staged dirt must always be rejected, including hidden tracked submodule
changes. The user explicitly chose to allow harmless untracked files, leaving
collision checks to Git. Use `--untracked-files=no` and
`--ignore-submodules=untracked` for the clean-state query so an unrelated
untracked file, including one inside a submodule, does not block a rebase.
Ignored-file safety and other native Git limitations still require explicit
verification, not assumptions.

## Integration and recovery requirements

- Add a shared basic-commit binding with a stable ID, initially discoverable
  through `?`; local history, subcommits and reflog share the action.
- Reuse exact native-format commit validation and current-HEAD identity checks.
  Reject detached/unborn HEAD, symbolic branch aliases, bare context and
  incompatible operations before presenting an executable plan.
- Recheck source identity and clean state immediately before launch. Git's
  normal concurrency model applies during execution; this is not an atomic
  expected-old transaction across files and refs.
- `rebase.updateRefs=false` must survive conflict continuation without changing
  the user's stored config. Verify an interior auxiliary branch, not only one
  at the old HEAD. Prefer a compatible command-scoped override over requiring
  a newer Git flag unnecessarily.
- Signing must keep its existing terminal interaction when needed. Neither
  signing failures nor hooks justify retrying, rolling back blindly, or hiding
  actual in-progress state.
- Refresh branches, commits, files, reflog and operation state after execution
  on both success and failure. Reuse conflict recovery and allow staged
  resolutions through continue. Do not impose start-only checks on recovery.

## Evidence and remaining verification

[Disposable Git probes](rebase-selected-commit-probes.md) distinguish native
behavior from fork policy. The [Git manual](https://git-scm.com/docs/git-rebase)
describes explicit upstreams, topology options, autostash and additional-ref
updates. The pull audit also reads the
[versioned Git implementation](https://github.com/git/git/blob/v2.51.1/builtin/pull.c)
for configuration precedence and the timing of its clean-state guard.

Backend tests need captured identity, dirty-state rejection, config opt-ins,
literal OIDs including same-name refs, topology, no-op, conflict continue/abort
and failure-state preservation. UI tests need source/repository capture,
fresh review, marked-base isolation, cancel, recovery and signing routing.
Generation, affected headless regressions, debug build, full short tests and
lint remain required before this item is marked complete.

The application currently requires Git 2.32.0. Its
[versioned rebase manual](https://git-scm.com/docs/git-rebase/2.32.0) and
[option parser](https://github.com/git/git/blob/v2.32.0/builtin/rebase.c)
include explicit no-autostash, no-fork-point and merge-topology choices.
The new action uses a command-scoped config override for additional-ref updates,
not the newer `--no-update-refs` flag. Actual executable verification so far is
Git 2.51.1 on Linux, not a minimum-version or cross-platform matrix.
