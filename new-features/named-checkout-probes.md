# Named checkout probe (Git 2.51.1)

Empirical probe for roadmap item 5. Run only with:

```sh
go run new-features/probes/named_checkout.go
```

The executable prints the exact commit IDs and, before and after each case,
HEAD, symbolic HEAD, all local branch refs, index tree, selected fixture-file
bytes, operation-state marker files and porcelain status. Each Git command has a 15-second context. Git
configuration is isolated from system/global config. Repositories are created
under one `os.MkdirTemp` root and cleanup removes only that root. The project
repository receives no Git writes. No project tests/builds were run.

The tested command was:

```sh
git checkout --no-overwrite-ignore -B NAME FULL_OID
```

No `--no-track` was needed to reproduce the tested behavior. Even with
`branch.autoSetupMerge=always`, creating a branch from the explicit OID did
not configure upstream tracking. Existing `branch.topic.remote` and
`branch.topic.merge` values survived resetting `topic`.

## Results

| Case | Observed result |
| --- | --- |
| New name, genuinely different target tree | Succeeded; HEAD became the new branch, branch ref became the supplied OID, index/worktree matched target. |
| Existing non-current name, different tree | Succeeded; moved `topic` from its old tip to the base commit, checked it out, and updated the index/worktree to the base tree. |
| Existing current name | Succeeded; moved its ref from the topic tip to the base commit and updated the index/worktree. |
| Unrelated staged, unstaged and untracked changes | Succeeded when staged addition, unstaged modification to unchanged `base-only`, and untracked `loose` were unrelated to target changes. All three were preserved with the original status; the index tree reflected the new target plus staged addition. |
| Untracked target-path collision | Rejected (status 1); HEAD, refs, index, file and status stayed unchanged. |
| Ignored target-path collision | With `topic-only` in `.git/info/exclude`, rejected (status 1); HEAD, refs, index and file stayed unchanged. Status omitted the ignored file. |
| Same-bytes untracked target-path collision | Rejected (status 1) even though the untracked bytes matched the target blob; state stayed unchanged. |
| Staged overlap (`common`) | Rejected (status 1); before/after HEAD, branch refs, index tree, file bytes and `M  common` status were identical. |
| Unstaged overlap (`common`) | Rejected (status 1); before/after HEAD, branch refs, index tree, file bytes and ` M common` status were identical. |
| Branch name exactly equals selected full OID | Setup creates that 40-hex branch pointing to the base commit, with `advice.objectNameWarning=false`. Bare `FULL_OID` succeeded but created `created` at the other commit reached through the same-named branch (`shadow-ref`), not at the selected OID. |
| Same OID with `^{commit}` suffix | Succeeded with `created` at the selected OID, despite the colliding branch name. |
| `git cat-file -t FULL_OID` with same-named branch | Exit status 0; stdout was `commit`; stderr warned that the refname was ambiguous. This command alone did not expose which of the two same-type commits its output described. |
| Existing branch upstream config | Preserved the exact `remote` and `merge` values after `-B`. |
| `branch.autoSetupMerge=always` with explicit target OID | New branch had no upstream config afterwards. |
| Branch occupied by another linked worktree | Rejected (status 128) with “already used by worktree”; current HEAD, refs, index and files remained unchanged. |
| Symbolic destination alias | Succeeded; HEAD became symbolic to `refs/heads/topic`, and that target ref moved. The alias itself remained symbolic. Do not treat a symbolic destination as an independent direct branch based on this command. |
| Failing `post-checkout` hook | Git returned status 1 after branch/ref, HEAD, index and worktree had changed to the requested target. |
| `reference-transaction` hook rejects `prepared` | Git returned 128; HEAD and refs stayed unchanged, but index/worktree had already transitioned to the target (status showed staged changes/deletions/additions). This is a partial operation despite the failed command. |
| Active clean merge (`--no-commit --no-ff`) | Setup asserted `MERGE_HEAD` existed. Checkout succeeded to `created` at the requested OID. Before state included `MERGE_HEAD`, staged merge result (`M  common`, `A  topic-only`) and its index tree; after-state had target files/index and no `MERGE_HEAD`. |
| Active conflicting rebase | Setup asserted conflict state and `rebase-merge/`. Checkout rejected with status 1 (“you need to resolve your current index first”); detached HEAD, unmerged `common` index/worktree and `rebase-merge/` remained. |
| Cherry-pick conflict | Setup asserted `CHERRY_PICK_HEAD` and an unmerged index. Checkout rejected with status 1 (“you need to resolve your current index first”); HEAD, refs, unmerged index/worktree and `CHERRY_PICK_HEAD` remained. |
| Active bisect | Setup asserted `BISECT_START` and a detached intermediate HEAD. Checkout succeeded to `created` at the requested OID; `BISECT_START` remained after checkout. |
| Late direct-ref movement from `post-index-change` | Hook marker confirmed execution after the index/worktree transition. The hook moved existing `topic` to `main`; checkout then succeeded and final `topic` pointed to the requested base OID, replacing the intervening movement. |
| Late alias appearance from `post-index-change` | Destination began as a direct `alias` branch. Hook deleted that direct ref with `update-ref --no-deref -d` and made `alias` symbolic to `main`; marker confirmed both hook commands completed. Checkout succeeded, final `HEAD` was symbolic to `main`, and the target update affected `main`. |

## Reproduction details and limits

The fixtures are built inline in `new-features/probes/named_checkout.go`:
they initialize `main`, commit `common` and `base-only`, create a divergent
`topic` history that adds `topic-only` and changes `common`, then add a
`main-only` commit. Each case fetches those commits into a fresh repository.
The source contains exact inline hook bodies and operation setups; it now
fails immediately if required setup commands or state assertions fail. Outputs
include fixture-specific OIDs and state; they are not stable across runs.

The deterministic hook run demonstrates that `checkout -B` does not compare
the destination ref with the tip observed before the index transition: an
intervening direct-ref move is overwritten. It also demonstrates that a
destination changed into a symbolic ref during that interval can redirect the
final symbolic `HEAD` and ref update. The bare full OID is likewise unsafe as
an unambiguous target expression in the presence of a same-named branch;
Git warned that `cat-file -t FULL_OID` was ambiguous, bare `FULL_OID` selected
the shadow branch's other commit, and `FULL_OID^{commit}` selected the
requested object. These results bear directly on backend argv construction
and destination revalidation.

The merge, rebase, cherry-pick and bisect fixtures now assert their operation
markers before the checkout attempt. Evidence is still limited to Git 2.51.1
on this Linux environment and the operation states described above; it does
not establish behavior for every merge conflict form, rebase backend, Git
version or ref storage backend. The `reference-transaction` rejection also
continues to show that a failed command can leave index/worktree changes
behind while HEAD and refs remain unchanged.
