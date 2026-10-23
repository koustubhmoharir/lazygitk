# Selected-source native merge probes (2026-10-24)

Git 2.51.1; isolated temporary repositories under
`/tmp/lazygitk-merge-probe-valid-94n0nmqc`. All fixture commits and file changes
were confined to these disposable repositories. The harness removed inherited
`GIT_*` variables and isolated HOME, global config and system config.

Each divergent fixture checks out `target`, commits there, checks out `main`,
and commits an independent path. Before each divergent merge, it recorded both full ref OIDs,
`git log --all --graph --oneline --decorate`, and `git merge-base`; both
`git merge-base --is-ancestor main target` and its reverse returned 1.
For example, the safe-dirty autostash case recorded:

```text
refs/heads/main   76cd95797959e425851f96ca8a3646f59381932a
refs/heads/target 2838fb39c290e630e76447c973c05a86a824a9cc
merge-base        85f4e36e45db44999ae454d2bf1559c54cb69b5f
* 76cd957 (HEAD -> main) current change
| * 2838fb3 (target) target change
|/
* 85f4e36 base
```

## 1. `--no-overwrite-ignore` and ignored collisions

In each case `target` tracked `ignored-target` (force-added despite
`.gitignore`) and the current worktree had different ignored bytes at that
path. The exact variants were:

```text
git merge --no-edit --no-overwrite-ignore refs/heads/target
git merge --no-ff --no-edit --no-overwrite-ignore refs/heads/target
git merge --squash --no-edit --no-overwrite-ignore refs/heads/target
```

| History / command | Exit and result | Final bytes, HEAD, index |
| --- | --- | --- |
| Fast-forward | 1; “untracked working tree files would be overwritten”; “Aborting” | Local bytes preserved; HEAD stays current; no staged diff |
| Divergent 3-way (`--no-ff`) | 0; `ort` merge succeeds | Target bytes overwrite ignored local bytes; HEAD advances to merge commit; index clean |
| Squash | 0; “Automatic merge went well”; “Squash commit -- not updating HEAD” | Target bytes overwrite ignored local bytes; HEAD unchanged; `ignored-target` staged as addition |

Thus the flag protects this collision in the fast-forward case observed here,
but did not prevent it in divergent 3-way or squash merges.

## 2. `--no-autostash` with `merge.autoStash=true`

Both runs used a verified divergent history: target changed tracked `shared`,
main changed independent tracked `current-only`; then `merge.autoStash=true`.
Exact command:

```text
git merge --no-autostash --no-edit refs/heads/target
```

With a dirty edit to separate tracked `dirty-safe`, exit was 0 and the merge
commit was made. The edit remained in the worktree (` M dirty-safe`), there was
no staged diff, and `git stash list` was empty.

With a dirty overlapping edit to `shared`, exit was 2 with “Your local changes
… would be overwritten by merge” and “Merge with strategy ort failed.” HEAD
stayed at the current commit, the dirty bytes remained (` M shared`), there was
no staged diff, and `git stash list` was empty.

## 3. Divergent `--ff` and `--ff-only`

Both commands ran against verified divergence and used
`refs/heads/target` explicitly:

```text
git merge --ff --no-edit refs/heads/target
git merge --ff-only --no-edit refs/heads/target
```

`--ff` exited 0 and created the ordinary `ort` merge commit; clean index and
worktree. `--ff-only` exited 128 with “Diverging branches can't be
fast-forwarded” / “Not possible to fast-forward, aborting”; HEAD, refs, index,
and worktree remained at their pre-command state.

## 4. Squash with unrelated staged content

Each case staged a new file `unrelated-staged` before:

```text
git merge --squash --no-edit refs/heads/target
```

In the fast-forward ancestry case, exit was 0. HEAD did not move; the index
contained both `A source-only` and `A unrelated-staged`, with no unstaged diff;
the staged file’s worktree bytes were unchanged. A subsequent normal commit
would include both staged paths.

In the divergent case (independent target and current commits; target added
`source-only`), exit was 2: “Your local changes to the following files would
be overwritten by merge: unrelated-staged” / “Merge with strategy ort failed.”
HEAD stayed put; `unrelated-staged` remained staged with unchanged bytes; no
merge path was staged. This specific name was absent from target history.

## 5. Unborn HEAD and immutable target CID

Each fresh repo had a clean orphan branch `empty` and used the full CID of
`refs/heads/target` as operand. Commands:

```text
git merge --no-edit <full-target-oid>
git merge --no-ff --no-edit <full-target-oid>
git merge --squash --no-edit <full-target-oid>
```

Regular merge exited 0, updated `refs/heads/empty`/HEAD to the target CID, and
populated the index and worktree from the target tree. `--no-ff` exited 128
with `fatal: Non-fast-forward commit does not make sense into an empty head`;
`--squash` exited 128 with `fatal: Squash commit into empty head not supported
yet`. In both failures HEAD remained unborn, the index remained empty, and the
worktree remained clean. The other refs still pointed to the target CID.

## 6. Already-up-to-date squash and subsequent commit

With `target` equal to current HEAD and a separate pre-staged file:

```text
git merge --squash --no-edit refs/heads/target
git commit -m 'after ancestor squash'
```

The merge exited 0 with “Already up to date. (nothing to squash)”; the
unrelated staged addition remained in the index. The subsequent commit
succeeded and contained only `unrelated-staged` (one file in `git show --stat`).

## 7. Full-CID merge message and `--no-edit`

On verified divergence, both commands used `--no-ff` and the full target CID:

```text
git merge --no-ff --no-edit <full-target-oid>
git merge --no-ff --no-edit -m 'Selected branch label' <full-target-oid>
```

Both exited 0 and created two-parent merge commits. Without `-m`, the recorded
message was `Merge commit '<full-target-oid>'`. With `-m`, the recorded message
was `Selected branch label`; `--no-edit` did not override it. Both ended with
clean index/worktree and main advanced to the merge commit; target ref was
unchanged.

Reference material: [Git `merge` documentation](https://git-scm.com/docs/git-merge)
and [TortoiseGit's merge guide](https://tortoisegit.org/docs/tortoisegit/tgit-dug-merge.html).

## 8. Isolated object-only path preview

An independent probe in `/tmp/lazygitk-merge-preview-9cncd989` used a base
containing `old/a`, a source that renamed `old` to `new`, and current HEAD adding
`old/added`. The worktree already held ignored data at `new/added`.

`git merge-tree --write-tree --no-messages -z <head-oid> <source-oid>` ran with
`GIT_OBJECT_DIRECTORY` set to a new temporary directory and
`GIT_ALTERNATE_OBJECT_DIRECTORIES` containing the C-quoted original object path.
Exit 1 still produced a valid result tree. Listing that tree returned `new/a`
and `new/added`; the latter was absent from both input trees. HEAD/refs, raw
index, ignored bytes and the original object-store file list were unchanged.
The private store contained one new object. This confirms that conflict exit 1
is useful and static input-tree unions miss directory-rename output paths.

These are command observations only; this report makes no implementation
recommendation.

## 9. Capturing native branch merge options

Fresh repositories under `/tmp/lazygitk-merge-config-*` tested native
`branch.main.mergeOptions` before merging a single immutable descendant CID.
Two entries, `--no-ff` followed by `--log=1`, produced a fast-forward; the first
entry's no-FF setting did not survive. Supplying the command-line configuration
override `-c branch.main.mergeOptions=` also produced a fast-forward. Thus this
Git reads the final effective value, and an empty command-line override can
isolate an explicitly captured/validated argument list from implicit re-parsing.

A branch merge-options value containing only a positional `extra` branch did
not add that branch to the merge: the selected CID alone fast-forwarded current
HEAD, and `extra`'s file was absent. The fork should still reject positional
sources in an options field instead of depending on ignored configuration text.
