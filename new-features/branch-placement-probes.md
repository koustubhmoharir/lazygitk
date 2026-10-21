# Branch placement probes (Git 2.51.1)

Bounded probe for roadmap item 4, “Create or move a branch at a selected
commit, without checkout.” This records Git behavior only; it does not choose
the production API or claim an atomic occupancy guarantee. All repositories
were disposable fixtures under `/tmp/branch-placement-probes.TMNFNm`. Git
configuration was isolated with `GIT_CONFIG_NOSYSTEM=1`,
`GIT_CONFIG_GLOBAL=/dev/null`, and `GIT_TERMINAL_PROMPT=0`; repository commands
also set `core.hooksPath=/dev/null` and a probe identity where commits were
needed. The project repository was not used for Git writes. No project tests,
builds, or headless runs were performed.

## Literal local branch validation

Commands were `git check-ref-format --branch "$input"` and
`git check-ref-format "refs/heads/$input"`, recording stdout/stderr and the
exit status. Results:

| Input bytes (shell representation) | `--branch` status/output | Full-ref status |
| --- | --- | --- |
| `HEAD` | 128, fatal invalid branch name | 0 |
| `-leading` | 128, fatal invalid branch name | 0 |
| `@{-1}` without checkout history | 128, fatal invalid branch name | 1 |
| `has space` | 128, fatal invalid branch name | 1 |
| `has\nnewline` (actual LF byte) | 128, fatal invalid branch name; diagnostic itself spans lines | 1 |
| `unicodé` | 0; stdout exactly `unicodé` plus newline | 0 |
| `nested/child` | 0; stdout exactly the input plus newline | 0 |
| `refs/heads/prefixname` | 0; stdout exactly the input plus newline | 0 |
| `ordinary` | 0; stdout exactly the input plus newline | 0 |

After creating checkout history with `git switch --detach HEAD` followed by
`git switch -`, `git check-ref-format --branch '@{-1}'` exited 0 and printed
the expanded commit OID (`bfeb9033ddc6896cd2f50f8bfb54c077d8d588c9` in that
fixture), not the literal input. Thus `--branch` is not a literal-preserving
validator for the `@{-N}` syntax: it expands that syntax when possible. The
full-ref check tests the prefixed name and does not substitute for
`--branch` (for example, `refs/heads/HEAD` and `refs/heads/-leading` both
passed). Preserve the user's input separately if using this command, and
reject expansion syntax explicitly if the intended accepted value is a
literal branch name.

## Exact local-ref lookup, packed refs, and symbolic refs

Query used:

```sh
git for-each-ref --format='%(refname)%00%(objectname)%00%(symref)' -- refs/heads/topic
```

It exited 0 with no output when missing, and emitted one NUL-delimited record
for an exact `refs/heads/topic` when present. `git pack-refs --all` moved the
fixture ref into packed storage; the same query still emitted the record and
OID. When only `refs/heads/topic/child` existed, the query emitted that child:
the `refs/heads/topic` pattern is a prefix namespace query, not an exact-name
query. Consumers must compare the returned `%(refname)` byte-for-byte with
`refs/heads/topic`; otherwise a nested ref can be mistaken for the exact
branch. An attempt to create `topic/child` while `topic` existed failed with
status 128 (`cannot lock ref ... 'refs/heads/topic' exists`), as expected for
the file/directory ref conflict.

With `git symbolic-ref refs/heads/alias refs/heads/topic/child`, querying the
`refs/heads/topic` pattern returned the child, not the unrelated `alias`.
Querying `-- refs/heads/alias` returned a record whose `%(refname)` was
`refs/heads/alias` and whose `%(symref)` was `refs/heads/topic/child`;
`git symbolic-ref -q refs/heads/alias` also returned the target (status 0).
Therefore the formatted query can identify a symbolic ref that matches its
pattern through a nonempty `%(symref)`, but the name must still be exact
filtered. `git show-ref --verify refs/heads/alias` resolved and printed the
alias OID, so it does not by itself distinguish a symbolic ref from a direct
ref.

## `update-ref` CAS, dirty-state preservation, reflog, and alias behavior

For the successful move fixture, `topic` initially referenced commit
`f77ee96cd53b4ea6b45f9e695c3a0aaa0e3216a0`; the requested new commit was
`6f1a7771975d768f9fab80019b6110a835811cdc`. `HEAD` was detached at the old
commit. Before the operation, the index tree was
`986834d8e4c7dd5dc636da157dfcfe6879e02705`; status was `A  staged`,
` M tracked`, and `?? loose`. The command was:

```sh
git update-ref --no-deref --create-reflog \
  -m 'lazygitk: move branch to selected commit' \
  refs/heads/topic NEW_OID EXPECTED_OLD_OID
```

It exited 0. Afterwards detached `HEAD`, the index tree hash, status, staged
file, modified tracked file bytes, and untracked file bytes were unchanged;
`topic` pointed to the new OID. The branch configuration remained exactly
`branch.topic.remote=origin` and
`branch.topic.merge=refs/heads/upstream-topic`. The new reflog entry's subject
was exactly `lazygitk: move branch to selected commit` and its new OID was the
requested OID.

Using the now-stale expected old OID for another update exited 128 with
`cannot lock ref ... is at <current> but expected <old>`; the ref stayed at
the current OID. Creating an absent `refs/heads/created` with an all-zero
40-character expected old OID exited 0. Repeating the same zero-OID create
against the now-existing ref exited 128 (`reference already exists`) and left
its OID unchanged.

High-priority symbolic-ref CAS probe: `refs/heads/alias` was a symbolic ref to
`refs/heads/occupied`, and that target's OID exactly equaled the supplied
expected old OID. Running
`git update-ref --no-deref --create-reflog -m sym-cas refs/heads/alias NEW
EXPECTED_OLD` exited 0. `alias` became a direct ref containing `NEW`
(`git symbolic-ref -q refs/heads/alias` then exited 1); `occupied` remained
at `EXPECTED_OLD`. It did not reject the symbolic-ref type and did not follow
the link. A production caller must reject a matching symbolic ref before this
write if symbolic names are prohibited; otherwise this command can replace
the alias itself. A second attempted update using a stale expected target OID
exited 128 and did not alter the alias target, but that mismatch does not
establish symbolic-type protection.

## Worktree occupancy and shared metadata

A linked worktree was created at a path containing an actual newline. Git
stored a common metadata directory at
`.git/worktrees/wt-line-break/`; its `HEAD` contained
`ref: refs/heads/topic\n`, `commondir` contained `../..\n`, and its `gitdir`
contained the linked worktree `.git` path including the newline. The common
repository's `for-each-ref --format='%(refname)%00%(worktreepath)' --
refs/heads` reported `refs/heads/topic` and the complete newline-containing
physical worktree path. NUL-delimited parsing preserves that path; line-based
parsing does not.

With `topic` checked out in that linked worktree, `git branch -f topic HEAD`
exited 128 and said the branch was already used by the worktree. In a separate
fixture, after renaming the linked metadata `gitdir` file out of the way,
`%(worktreepath)` became empty and `git branch -f held HEAD` exited 0. This
demonstrates that Git's occupancy result relies on shared worktree metadata;
missing/stale metadata can make an actually present directory invisible to
that guard. A detached linked worktree had an OID in its per-worktree `HEAD`,
not a branch ref, and consequently did not mark an unrelated branch as
occupied.

## Rebase `--update-refs` and bisect reservations

An interactive rebase was paused in a disposable repository using
`GIT_SEQUENCE_EDITOR="sed -i '1a exec false'"`,
`GIT_EDITOR=/bin/true`, and `git rebase -i --update-refs HEAD~2`. It exited 1
at the deliberately failing `exec`, leaving `.git/rebase-merge/` active.
`head-name` contained `refs/heads/main\n`; `update-refs` contained newline
separated triples: `refs/heads/side2`, its old OID, and an all-zero OID
placeholder. `for-each-ref ... %(worktreepath)` showed an empty worktree path
for `side2` because the operation was running with detached `HEAD`. While
paused, `git branch -f main HEAD` exited 128 (current branch), `git branch -f
side2 HEAD` exited 128 (reserved for the pending update-refs operation), and
`git branch -f side HEAD` exited 0 (not reserved). This confirms that checking
only current worktree `HEAD` / `%(worktreepath)` misses the ancillary
`--update-refs` branch reservation; inspect the active operation metadata or
use an equivalent Git occupancy guard.

A bisect was started from `main`, marked bad at `HEAD` and good at `HEAD~2`.
It detached `HEAD` at an intermediate commit while `.git/BISECT_START`
contained `main\n`. During that state, `git branch -f main HEAD` exited 128,
while `git branch -f other HEAD` exited 0. Thus the operation's original
branch remains reserved even though `HEAD` is detached. In this run
`.git/BISECT_TERMS` contained `bad\ngood\n`; `BISECT_LOG` recorded the
commands. No `.git/rebase-apply/head-name` was present for the observed
interactive rebase backend; its active data was under `rebase-merge`.

## Guard boundary and limits

`git update-ref --no-deref` is a ref transaction with expected-old-OID CAS and
reflog support, but it does not apply `git branch -f`'s worktree protection:
while `occupied` was checked out in a linked worktree, `git branch -f
occupied NEW` exited 128 and left the ref unchanged; the direct
`git update-ref --no-deref -m occupied-cas refs/heads/occupied NEW OLD`
exited 0 and moved it. Consequently CAS prevents overwriting a concurrently
changed ref value, but it neither enforces branch occupancy nor atomically
combines occupancy checking with the ref update. A separate occupancy check
can race with worktree creation. These probes make no stronger concurrency
claim.

## Prepared `update-ref --stdin` transaction

Git 2.51.1 accepted a long-lived `git update-ref --stdin --no-deref
--create-reflog -m 'lazygitk: move branch to selected commit'` process with
`start`, `update`, `prepare`, then either `abort` or `commit`. The probe kept
stdin open after `prepare` and received `start: ok` and `prepare: ok` on
stdout. At that point the target's loose `<ref>.lock` existed, while normal
readonly commands still read the committed pre-transaction state.

For the alias case, `refs/heads/alias` pointed to `refs/heads/topic`, and
`topic` equaled the update's expected-old OID. The prepared commands were:

```text
start
update refs/heads/alias NEW_OID EXPECTED_OLD_OID
prepare
```

While prepared, `.git/refs/heads/alias.lock` existed;
`git symbolic-ref -q refs/heads/alias` still returned
`refs/heads/topic`; and
`git for-each-ref --format='%(refname)%00%(objectname)%00%(symref)' --
refs/heads/alias` returned the alias with nonempty `%(symref)` and the old
resolved OID. `git worktree list --porcelain` also completed and showed both
worktrees and their current branch states. The probe then sent `abort`; Git
replied `abort: ok`, exited 0 after stdin closed, and the alias remained
symbolic. This is an inspection point to identify and abort on a symbolic
alias before commit, without briefly replacing it.

A second prepared transaction targeted a direct `refs/heads/topic` ref. While
its lock existed, `for-each-ref` returned the old OID and
`git symbolic-ref -q refs/heads/topic` exited 1. A competing
`git -c core.filesRefLockTimeout=100 update-ref refs/heads/topic OTHER_OID
EXPECTED_OLD_OID` exited 128 and left the old OID visible. A competing
`git -c core.filesRefLockTimeout=100 symbolic-ref -m concurrent
refs/heads/alias refs/heads/main` during the alias transaction exited 1 and
left the alias target unchanged. After sending `commit`, Git replied
`commit: ok`, exited 0, moved the direct ref, and recorded the requested
reflog reason. These observations cover same-ref conflicts for the tested Git
commands; they do not prove that checkout or worktree creation serializes
against a ref transaction.

The transaction create form `create refs/heads/created NEW_OID` followed by
`prepare` created a `.lock` file. Closing stdin without `commit` caused Git
to exit 0 and left the ref absent: the open transaction rolled back. In the
one-shot form, the earlier explicit zero-OID expected-old create successfully
created an absent ref and failed without changing an existing one.

A temporary local `reference-transaction` hook printed `hook-stdout` and
`hook-stderr` on each phase. With
`git update-ref --stdin --no-deref -m hook-test`, stdout was interleaved as
`start: ok`, hook text, `prepare: ok`, hook text, and `commit: ok`; hook stderr
appeared separately. A caller using the interactive protocol must drain both
streams and recognize protocol acknowledgements among hook output rather than
treating every line as an ACK or waiting on only one stream. The hook existed
only in the temporary repository.

The official Git 2.27.0 `git-update-ref` manual already documents the `--stdin`
`start`, `prepare`, `commit`, and `abort` commands and their lock-file behavior
([manual](https://git-scm.com/docs/git-update-ref/2.27.0)). This establishes
the protocol as documented by Git 2.27.0; Git 2.32 itself was not executed, so
runtime compatibility on that exact version was not verified.

This report intentionally leaves API and design decisions to the main agent.
Probe limitations include one runtime Git version (2.51.1), ordinary
filesystem refs, the tested rebase-merge backend, and no checkout/worktree
creation race test.
