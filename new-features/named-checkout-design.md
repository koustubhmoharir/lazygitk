# Named checkout

Roadmap item 5 is implemented. The user chose Git's normal concurrency model:
revalidate immediately before starting, and coordinate other ref-mutating tools
while checkout runs. The command-level lock-retry opt-out, shared literal-OID
checks, backend and modal flow are implemented. The roadmap and merge guide
record verification and limits; the tracker now owns the next item's work.

## Intended flow

[TortoiseGit's checkout dialog](https://tortoisegit.org/docs/tortoisegit/tgit-dug-checkout.html)
separates the target, branch name and working-tree overwrite option. This fork's
**Checkout as branch…** will be a distinct row in `RefsHelper.CreateCheckoutMenu`.
It will never request working-file overwrite, stash or a temporary checkout.
Existing detached/existing-at-commit choices remain distinct.

Capture repository and selected commit identity before opening the name prompt.
Accept a literal local name with existing local-name suggestions, not silent
sanitization or a guessed name. Reuse the placement validation/ref-query helpers
and the shared pure full-OID guard. A backend plan should snapshot destination
type/tip and current HEAD identity; a change before application requires a fresh
review. Warn before moving an existing different tip, including the current
branch when applicable. Creating or attaching an already matching branch still
performs checkout, unlike ref-only placement's no-op.

Check current operation metadata and all worktree reservations freshly. Do not
let an allowed current checkout mask another worktree's reservation. Refuse
active merge, rebase, cherry-pick, revert, sequencer or bisect state instead of
letting checkout silently discard/leave operation metadata. Do not require a
clean index/worktree: Git must decide whether actual paths would be overwritten.

## Verified command and failure behavior

The [reproducible Git probes](named-checkout-probes.md) establish:

- Combined `checkout --no-overwrite-ignore -B` carries tested unrelated dirty
  changes and rejects tested overwrite collisions without moving branch refs.
  Do not split branch movement and checkout into separate commands.
- A bare full OID can be redirected by a branch with exactly that name.
  The verified `OID^{commit}` form avoids checkout's branch-name shadowing.
  Preparation must also verify actual object identity/type, not just a
  `cat-file -t` combined-output string that may include warnings.
- Git permits some active-operation checkouts; the fork needs its own guards.
- A failing post-checkout hook can follow a completed checkout. A rejecting
  reference-transaction hook can leave index/files changed but refs unchanged.
  Never promise universal rollback or retry after such a failure. Refresh all
  relevant models and show Git's actual failure plus possible partial changes.
- Use the command-local lock-retry opt-out; lock text is not proof that no
  state changed. Existing Git hook behavior must remain enabled.

The command sequence and timing are corroborated by Git's
[checkout implementation](https://raw.githubusercontent.com/git/git/v2.51.1/builtin/checkout.c)
and [branch ref-update implementation](https://raw.githubusercontent.com/git/git/v2.51.1/branch.c).
Ordinary overwrite rejection and post-transition failures are different cases.

## Agreed concurrency contract

The deterministic `post-index-change` probes prove that `checkout -B` does not
perform an expected-old ref update for an existing destination. A direct tip
change during checkout is overwritten. If a direct destination becomes a
symbolic alias during that interval, Git follows it and updates another branch.
No amount of pre-launch rechecking makes that interval atomic. Unlike ref-only
placement, holding a prepared ref lock through checkout would prevent Git's own
combined operation from acquiring that lock; it is not a usable drop-in solution.

The user explicitly chose Git's native concurrency contract: fresh checks
before launch, and coordination of other ref-mutating tools while checkout is
running. Do not imply expected-old atomicity for this command or introduce a
hook override, Git patch, temporary checkout or split-operation workaround.
Preserve existing hooks and report Git's partial-transition failure behavior.

The implementation uses input-blocking status for apply. When a snapshot becomes
stale, prepare its replacement on that same worker and queue the fresh warning
before buffered input is released. Failure refresh preserves the selected
history target/reflog position while updating actual HEAD, refs, index and files;
successful checkout deliberately selects HEAD/the checked-out branch and switches
to the branch pane after the batched refresh.

Sparse checkout and the user's submodule recursion setting remain Git-owned.
Real-Git acceptance tests verify cone preservation, non-recursive checkout,
clean recursive checkout and dirty nested-file rejection.
