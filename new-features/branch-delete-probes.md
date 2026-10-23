# Native branch deletion probe evidence

Git: `git version 2.51.1`. Fixtures: `/tmp/lazygitk-branch-delete-probe-5vqq9l80` (temporary directory; disposable local repos only).
Complete command/output log: `/tmp/lazygitk-branch-delete-probe-5vqq9l80/log.jsonl` (JSONL, temporary). The evidence below keeps only review-relevant commands and compact state comparisons. Observational snapshots use `git --no-optional-locks`; snapshots include HEAD, refs, status, config, raw index and worktree bytes.

## Findings for review

- A full-SHA-looking branch name is deleted literally by `git branch -D -- <name>`. The delimiter keeps names out of option parsing.
- Multi-branch `git branch -D remove-me occupied missing` reports failure after removing `remove-me`; `occupied` remains because it is checked out, and `missing` remains absent. HEAD, index and worktree snapshots are recorded before and after.
- Deleting the branch named by bare repository HEAD succeeds. A branch checked out in a linked worktree is refused. The active-rebase fixture uses diverged source/target commits and an interactive sequence editor that inserts `break`; it records the native `rebase-merge/head-name` and `onto` files. Deletion is attempted from another worktree, followed by deletion of an unrelated branch to show the reservation boundary.
- With `branch.topic.merge` set twice, `for-each-ref` exposes only the first merge value; `git config --get-all` exposes both. Native `%(upstream)` retains remapped local tracking storage, while `%(upstream:remoteref)` remains the full server ref.
- Explicit `git push --no-follow-tags --delete -- named refs/heads/exact` succeeds. With mirror=true and an extra push refspec, the same explicit delete with `-c remote.named.mirror=false` deletes only `exact`; unrelated refs and the local tag are preserved.
- A single `git push --no-follow-tags --delete -- multi refs/heads/partial` starts with that branch present at both push URLs. It exits 1 after the first local bare remote removes it and the second pre-receive hook rejects deletion, leaving the second ref intact.
- TortoiseGit’s Browse All Refs guide describes context-menu deletion/renaming and local-branch tracked-branch metadata for the selected refs; it does not claim that deletion automatically removes another worktree: [Browse All Refs](https://tortoisegit.org/docs/tortoisegit/tgit-dug-browse-ref.html).

- **State — literal before SHA-named delete:** HEAD `refs/heads/main`; refs `refs/heads/5eaff1ec2ab0f0f1b0c5ce4e93dea9633f9b3a61 5eaff1ec2ab0f0f1b0c5ce4e93dea9633f9b3a61; refs/heads/main 5eaff1ec2ab0f0f1b0c5ce4e93dea9633f9b3a61`; status `clean`.
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/literal` `$ git branch -D -- 5eaff1ec2ab0f0f1b0c5ce4e93dea9633f9b3a61` → exit 0: `Deleted branch 5eaff1ec2ab0f0f1b0c5ce4e93dea9633f9b3a61 (was 5eaff1e).`
- `ASSERT` `$ "literal SHA-named branch was deleted"` → exit 0: `PASS`
- **State — literal after SHA-named delete:** HEAD `refs/heads/main`; refs `refs/heads/main 5eaff1ec2ab0f0f1b0c5ce4e93dea9633f9b3a61`; status `clean`.
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/literal` `$ git branch -D 5eaff1ec2ab0f0f1b0c5ce4e93dea9633f9b3a61` → exit 1: `error: branch '5eaff1ec2ab0f0f1b0c5ce4e93dea9633f9b3a61' not found`
- **State — multi-delete before:** HEAD `refs/heads/occupied`; refs `refs/heads/main fdc638c5e94ed60bade4bca596cc33eebcf15b41; refs/heads/occupied fdc638c5e94ed60bade4bca596cc33eebcf15b41; refs/heads/remove-me fdc638c5e94ed60bade4bca596cc33eebcf15b41`; status `1 .M N... 100644 100644 100644 df967b96a579e45a18b8251732d16804b2e56a55 df967b96a579e45a18b8251732d16804b2e56a55 base.txt
1 .A N... 000000 000000 100644 0000000000000000000000000000000000000000 0000000000000000000000000000000000000000 untracked.txt`.
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/partial` `$ git branch -D remove-me occupied missing` → exit 1: `Deleted branch remove-me (was fdc638c). ; error: cannot delete branch 'occupied' used by worktree at '/tmp/lazygitk-branch-delete-probe-5vqq9l80/partial'
error: branch 'missing' not found`
- `ASSERT` `$ "multi-delete reports native failure"` → exit 0: `PASS`
- `ASSERT` `$ "multi-delete removed earlier eligible branch and retained occupied branch"` → exit 0: `PASS`
- `ASSERT` `$ "missing branch remains absent after multi-delete"` → exit 0: `PASS`
- **State — multi-delete after remove-me occupied missing:** HEAD `refs/heads/occupied`; refs `refs/heads/main fdc638c5e94ed60bade4bca596cc33eebcf15b41; refs/heads/occupied fdc638c5e94ed60bade4bca596cc33eebcf15b41`; status `1 .M N... 100644 100644 100644 df967b96a579e45a18b8251732d16804b2e56a55 df967b96a579e45a18b8251732d16804b2e56a55 base.txt
1 .A N... 000000 000000 100644 0000000000000000000000000000000000000000 0000000000000000000000000000000000000000 untracked.txt`.
- `ASSERT` `$ "multi-delete preserved HEAD-bytes-hex"` → exit 0: `PASS`
- `ASSERT` `$ "multi-delete preserved index-bytes-hex"` → exit 0: `PASS`
- `ASSERT` `$ "multi-delete preserved worktree-bytes-hex"` → exit 0: `PASS`
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/bare` `$ git branch -D main` → exit 0: `Deleted branch main (was 1d88c64).`
- `ASSERT` `$ "bare HEAD-named branch deletion succeeded"` → exit 0: `PASS`
- **State — bare after deleting branch named by bare HEAD:** HEAD `refs/heads/main`; refs ``; status `bare (no worktree)`.
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/linked` `$ git branch -D reserved` → exit 1: `error: cannot delete branch 'reserved' used by worktree at '/tmp/lazygitk-branch-delete-probe-5vqq9l80/linked-wt'`
- `ASSERT` `$ "linked-worktree branch deletion was refused"` → exit 0: `PASS`
- **State — linked after reserved delete attempt:** HEAD `refs/heads/main`; refs `refs/heads/main 41b14300b5d354ecf1ca7d3ac824f63a2755f596; refs/heads/reserved 41b14300b5d354ecf1ca7d3ac824f63a2755f596`; status `clean`.
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/active-rebase-wt` `$ git rebase -i --onto b325d54d2d1f5d34ad326e4ee0024390131bfe7c 518feb5252072224bb26b2e4972a7aeec58b3d06 source` → exit 0: `Rebasing (1/2)
Stopped at b325d54 (target)`
- **State — active rebase after sequence-editor break:** HEAD `(detached)`; refs `refs/heads/main b325d54d2d1f5d34ad326e4ee0024390131bfe7c; refs/heads/source 36ab9bd8fb2e8bf276d78723997b82bc32a9acdc; refs/heads/unreserved 518feb5252072224bb26b2e4972a7aeec58b3d06`; status `clean`.
- `FIXTURE` `$ "read /tmp/lazygitk-branch-delete-probe-5vqq9l80/active-rebase/.git/worktrees/active-rebase-wt/rebase-merge/head-name"` → exit 0: `refs/heads/source`
- `ASSERT` `$ "native active rebase metadata matches .git/rebase-merge/head-name"` → exit 0: `PASS`
- `FIXTURE` `$ "read /tmp/lazygitk-branch-delete-probe-5vqq9l80/active-rebase/.git/worktrees/active-rebase-wt/rebase-merge/onto"` → exit 0: `b325d54d2d1f5d34ad326e4ee0024390131bfe7c`
- `ASSERT` `$ "native active rebase metadata matches .git/rebase-merge/onto"` → exit 0: `PASS`
- `ASSERT` `$ "paused rebase worktree HEAD is detached"` → exit 0: `PASS`
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/active-rebase` `$ git branch -D -- source` → exit 1: `error: cannot delete branch 'source' used by worktree at '/tmp/lazygitk-branch-delete-probe-5vqq9l80/active-rebase-wt'`
- `ASSERT` `$ "branch participating in paused rebase was retained"` → exit 0: `PASS`
- **State — main worktree attempt to delete branch reserved by paused linked rebase:** HEAD `refs/heads/main`; refs `refs/heads/main b325d54d2d1f5d34ad326e4ee0024390131bfe7c; refs/heads/source 36ab9bd8fb2e8bf276d78723997b82bc32a9acdc; refs/heads/unreserved 518feb5252072224bb26b2e4972a7aeec58b3d06`; status `clean`.
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/active-rebase` `$ git branch -D -- unreserved` → exit 0: `Deleted branch unreserved (was 518feb5).`
- `ASSERT` `$ "unrelated branch deletion succeeded during paused rebase"` → exit 0: `PASS`
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/tracking` `$ git fetch named` → exit 0: `From /tmp/lazygitk-branch-delete-probe-5vqq9l80/tracking-remote
 * [new branch]      review     -> refs/custom/cache/review`
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/tracking` `$ git for-each-ref --format=%(refname)|%(upstream)|%(upstream:short)|%(upstream:remotename)|%(upstream:remoteref) refs/heads/topic` → exit 0: `refs/heads/topic|refs/custom/cache/review|custom/cache/review|named|refs/heads/review`
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/tracking` `$ git config --get-all branch.topic.merge` → exit 0: `refs/heads/review
refs/heads/other`
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/tracking` `$ git config --get-regexp ^branch\.topic\.` → exit 0: `branch.topic.remote named
branch.topic.merge refs/heads/review
branch.topic.merge refs/heads/other`
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/tracking` `$ git --no-optional-locks for-each-ref --format=%(upstream)|%(upstream:remotename)|%(upstream:remoteref) refs/heads/topic` → exit 0: `refs/custom/cache/review|named|refs/heads/review`
- `ASSERT` `$ "named remote upstream resolves remapped tracking ref and original remote ref"` → exit 0: `PASS`
- **State — tracking metadata:** HEAD `refs/heads/main`; refs `refs/heads/main 518590b109dc5b0c0530172c1ca3e3b34a98185a; refs/heads/topic 518590b109dc5b0c0530172c1ca3e3b34a98185a`; status `clean`.
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/push-local` `$ git push --no-follow-tags --delete -- named refs/heads/exact` → exit 0: `To /tmp/lazygitk-branch-delete-probe-5vqq9l80/push-bare
 - [deleted]         exact`
- **State — remote before explicit mirror-suppressed delete:** HEAD `refs/heads/master`; refs `refs/heads/exact 456f8d1a6db881050d6781cdf0e72dd5d0f26214; refs/heads/unrelated 456f8d1a6db881050d6781cdf0e72dd5d0f26214`; status `bare (no worktree)`.
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/push-local` `$ git -c remote.named.mirror=false push --no-follow-tags --delete -- named refs/heads/exact` → exit 0: `To /tmp/lazygitk-branch-delete-probe-5vqq9l80/push-bare
 - [deleted]         exact`
- `ASSERT` `$ "mirror-suppressed explicit delete changed only exact branch ref"` → exit 0: `PASS`
- `ASSERT` `$ "explicit remote deletion left local tag intact"` → exit 0: `PASS`
- **State — remote after explicit mirror-suppressed delete:** HEAD `refs/heads/master`; refs `refs/heads/unrelated 456f8d1a6db881050d6781cdf0e72dd5d0f26214`; status `bare (no worktree)`.
- `ASSERT` `$ "partial branch was seeded at both push URLs"` → exit 0: `PASS`
- **State — multi-pushurl remotes before:** HEAD `refs/heads/master`; refs `refs/heads/partial 10f06c11c0ba0c4d820a95b930637f8a7a8124d8`; status `bare (no worktree)`.
- **State — multi-pushurl remotes before:** HEAD `refs/heads/master`; refs `refs/heads/partial 10f06c11c0ba0c4d820a95b930637f8a7a8124d8`; status `bare (no worktree)`.
- `/tmp/lazygitk-branch-delete-probe-5vqq9l80/multiurl-local` `$ git push --no-follow-tags --delete -- multi refs/heads/partial` → exit 1: `To /tmp/lazygitk-branch-delete-probe-5vqq9l80/multiurl-first
 - [deleted]         partial
remote: second-reject        
To /tmp/lazygitk-branch-delete-probe-5vqq9l80/multiurl-second
 ! [remote rejected] partial (pre-receive hook declined)
error: failed to push some refs to '/tmp/lazygitk-branch-delete-probe-5vqq9l80/multiurl-second'`
- `ASSERT` `$ "multi-pushurl deletion returns native failure status 1"` → exit 0: `PASS`
- `ASSERT` `$ "multi-pushurl delete removed first ref and retained second after hook rejection"` → exit 0: `PASS`
- **State — multi-pushurl first after command:** HEAD `refs/heads/master`; refs ``; status `bare (no worktree)`.
- **State — multi-pushurl second after command:** HEAD `refs/heads/master`; refs `refs/heads/partial 10f06c11c0ba0c4d820a95b930637f8a7a8124d8`; status `bare (no worktree)`.