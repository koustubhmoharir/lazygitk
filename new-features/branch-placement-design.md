# Branch placement without checkout

Roadmap item: 4. The additive implementation follows the bounded
[Git probes](branch-placement-probes.md); verification is recorded in the merge guide.

## User flow and ownership

[TortoiseGit's branch dialog](https://tortoisegit.org/docs/tortoisegit/tgit-dug-branchtag.html)
treats switching as a separate choice. This fork exposes two distinct commit-list
actions: **Create/move branch here…** (ref only) and **Create branch and check it
out…** (existing workflow). The new action starts unbound but discoverable in
`?`; the full keymap milestone assigns its final shortcut.

Capture the selected full commit OID before opening the prompt. Reject ranges,
placeholders and missing commits. Suggest existing local branch names using the
existing suggestions helper, but do not silently sanitize, expand or choose a
name. The input is a literal local branch name, not a revision expression.

Backend preparation reads and validates the target commit, local ref and all
worktree reservations. A missing name means create; the same tip means an
informative no-op; an existing different tip requires a warning naming branch,
old/new abbreviated OIDs and the fact that only its tip moves. Cancellation
does nothing. A prohibited current/occupied-branch move produces a reason,
not an offer to stash, checkout another branch or detach another worktree.

The backend rechecks its snapshot immediately before writing. A changed ref
requires preparation and fresh confirmation of the updated state; it must not
silently retry a destructive write. Source selection and repository identity
remain captured throughout. Read/write work runs on the foreground worker,
with modal decisions on the UI thread. Refresh branch/commit/reflog markers
with the existing by-name/by-hash selection defaults; do not jump panes or HEAD.

## Ref safety and recovery

- Use Git's literal branch-name validation. Reject symbolic branch aliases;
  updating them must not dereference into an unrelated branch.
- Read refs through Git, not loose-ref files, so packed/reftable storage is not
  guessed. Exact matching is required: a prefix query can also return children.
- Apply an expected-old-OID update with a descriptive branch reflog reason.
  Preserve tracking configuration. A new ref must still be absent when created.
  Reusing `NewWithoutCheckout` is appropriate only if it meets these guarantees;
  do not split a ref move and checkout to emulate another workflow.
- Use a prepared `update-ref --stdin` transaction: after Git acknowledges
  `prepare`, keep its ref lock while re-reading the committed ref type/value
  and worktree reservations. Abort if a symbolic alias appeared, even if its
  resolved OID still equals the warning's old tip. Only then send `commit`.
  Closing stdin before commit must abort. Existing reference-transaction hooks
  remain enabled; drain stdout/stderr and recognize ACKs among hook output.
- Never retry a possibly committed transaction automatically. Protocol/process
  failure after sending commit must report uncertainty and refresh state rather
  than assume rollback. Bound inherited-output lifetime after Git exits and
  keep diagnostics bounded without losing protocol acknowledgements.
- Git's ref lock makes the expected-old update atomic. Worktree occupancy is a
  freshly evaluated guard, not a repository-wide lock against arbitrary external
  checkouts/worktree creation. Do not claim stronger cross-process serialization.
- The existing Undo controller parses HEAD reflog checkout/commit/rebase actions,
  not independent branch-tip updates. This feature must not claim Undo support.
  Verify recovery through the destination branch's own reflog instead.

## Worktree reservations

Git's own [branch protection](https://raw.githubusercontent.com/git/git/v2.51.1/branch.c)
includes attached HEADs, rebase/bisect branches and ancillary refs reserved by
`rebase --update-refs`. Checking only the visible branch list is insufficient.

The existing worktree loader enriches UI models but parses path-oriented output
and resolves metadata through physical working directories. The new safety check
must also handle unusual/missing worktree paths and in-progress operation state.
Prefer Git metadata directories from the captured common repository, querying
HEAD through Git and reading operation-reservation files conservatively. Read
failures or malformed active state fail closed rather than permit a move.

Git's [ref-update contract](https://git-scm.com/docs/git-update-ref.html) and
the probe report determine the exact compare-and-swap/symbolic-ref handling.
The implementation preserves these checks at both preparation and
application, not rely on a UI model captured before the warning.

## Integration boundaries

Prefer additive backend/guard modules and one shared commit-controller hook.
Keep upstream checkout/new-branch implementations unchanged apart from the
clarifying label in commit lists. Reuse the existing prompt, local-name
suggestions, worker scheduling and refresh APIs; no new command framework or
temporary worktree is needed. Tests cover Git results and the modal sequence,
including changed-state confirmation rather than only command construction.
