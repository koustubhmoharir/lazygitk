# Native branch creation probe (item 7)

This bounded probe covers branch creation and checkout behavior only; the main
implementation owns product design. The script creates fresh temporary repos,
captures each command's argv/stdout/stderr/return code, and snapshots symbolic
and resolved HEAD, refs, tracking config, index entries, raw Git `HEAD`/index/
config bytes, and worktree file bytes. It does not invoke `git status`. The
complete 92-case JSON log is retained at `/tmp/lazygitk-branch-create-probe.json`;
running the script prints only log path, Git version, and case count. Tested with
Git 2.51.1. No production files or Git history were changed.

## Findings for the design owner

`git branch <name> <full-OID>` creates the ref at the specified object and
leaves HEAD, index, and worktree unchanged. For an immutable revision captured
before a prompt, this is the native exact-target command. Native branch-ref
creation has simpler tracking semantics when the user's intended source is a
moving branch: Git resolves that ref as part of the command. A prelaunch exact
ref recheck is only an observation; with native Git concurrency and no CAS, the
ref can still move before Git uses it. Therefore the design must choose whether
the operation targets the captured OID or the native branch ref. Do not claim
either choice locks or atomically validates the source.

OID tracking behavior:

- Default `branch.autoSetupMerge` values `true`, `always`, `inherit`, `simple`,
  and `false` all created a branch at an OID with no upstream.
- `--track` and `--track=direct` with an OID returned 128 (`starting point
  '<oid>' is not a branch`) and did not create the ref.
- `--track=inherit` with an OID succeeded without upstream because the OID has
  no tracking config to inherit. It is not equivalent to direct tracking.
- `--no-track` succeeded without upstream under all settings.

With branch refs, the defaults followed Git's configuration: `true` tracks a
remote-tracking ref directly but not a local branch; `always` tracks both;
`inherit` copies source tracking configuration; `simple` tracks a remote ref
only when the created branch name matches the remote branch name; `false`
tracks neither. Explicit `--track` and `--track=direct` request direct tracking.
The probe verified `simple` with both a mismatched name (`created` from
`origin/topic`, no upstream) and matching name (`topic` from `origin/topic`,
direct upstream).
Explicit `--track=inherit` copied the configured source branch's remote and
both merge values in the multi-merge probe. On an unconfigured source branch it
succeeded without tracking and emitted a warning. `--no-track` suppresses
automatic setup.

### Explicit upstream after exact-OID creation

`git branch --set-upstream-to=origin/topic created` succeeded after exact-OID
creation, wrote `branch.created.remote=origin` and
`branch.created.merge=refs/heads/topic`, and preserved the branch OID, HEAD,
index, and worktree. With a pre-existing `.git/config.lock`, the native command
returned 1 (`could not lock config file`); the created ref remained and no
tracking keys were written. This reproduces a normal config-lock failure
without a custom backend and demonstrates Git's partial state. No rollback is
appropriate if preserving native partial-state semantics.

The config override probe ran:

```text
git -c branch.created.remote=origin -c branch.created.merge=refs/heads/topic branch created <OID>
```

It created the branch but persisted neither tracking key: the `-c` values are
process-local and do not provide native tracking inference. Earlier repository
config seeding was not a per-command override and is not relied on here.

### Checkout and obstruction behavior

`git checkout -b created <OID>` checked out the exact OID with no upstream;
using `origin/topic` as the start ref with `branch.autoSetupMerge=true` created
direct tracking. `--no-overwrite-ignore` was probed separately against three
actual collisions. A target commit changed tracked `base.txt`, added an
untracked path, or added an ignored path that was locally present with different
contents. Each command returned 1 before creating the new branch; HEAD, refs,
index and colliding worktree bytes remained unchanged. The ignored case's error
describes the path as an untracked file that would be overwritten. With no
collision, checkout succeeded. Harmless local changes can be preserved only
when they do not obstruct the target; the probe does not establish behavior for
every checkout conflict shape.

Other captured cases:

- `git branch -- bad..name <OID>` returned 128 and created no ref. `--` ends
  option parsing; Git still validates the literal branch name.
- A source branch literally named `feature=foo` was accepted as a start-point.
- From unborn HEAD, `git branch created <OID>` created the ref while leaving
  symbolic HEAD unborn; `git checkout -b created <OID>` succeeded and attached
  HEAD to that branch.
- In a bare repository, `git branch created <OID>` succeeded and left the
  repository's symbolic HEAD unchanged. The bare repo has no worktree/index for
  checkout preservation.

## Recommendation boundary

For the captured-OID path, honor no-tracking with the single `git branch`
command. For direct tracking or a user-selected explicit remote/merge target,
the tested native sequence is exact-OID creation followed by
`--set-upstream-to`; a failure leaves the ref in place. `--track=direct` cannot
be used with an OID. `--track=inherit` has no source config to copy at an OID,
so its UX meaning remains a design decision; do not silently turn it into direct
tracking or inherit from the currently checked-out branch.

For a selected branch ref, native ref-based tracking may be simpler and retains
normal Git race behavior, but may resolve a later ref value than the OID
captured before prompting. The probe establishes behavior, not which semantic
the product should promise.

## References

Git's primary [git-branch documentation](https://git-scm.com/docs/git-branch)
describes direct/inherited tracking and `branch.autoSetupMerge`; the
[git-checkout documentation](https://git-scm.com/docs/git-checkout) documents
`-b` and `--no-overwrite-ignore`. TortoiseGit's
[Branching/Tagging documentation](https://tortoisegit.org/docs/tortoisegit/tgit-dug-branchtag.html)
and [log-dialog documentation](https://tortoisegit.org/docs/tortoisegit/tgit-dug-showlog.html)
provide the user-facing context for creating a branch from a selected revision.
Those docs do not decide the captured-OID versus selected-ref contract.
