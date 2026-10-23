# Native branch deletion probe evidence

Git: `git version 2.51.1`. Fixtures: `/tmp/lazygitk-branch-delete-probe-ur16pm9y` (temporary directory; disposable local repos only).
Complete command/output log: `/tmp/lazygitk-branch-delete-probe-ur16pm9y/log.jsonl` (JSONL, temporary). The evidence below keeps only review-relevant commands and compact state comparisons. Observational snapshots use `git --no-optional-locks`; snapshots include HEAD, refs, status, config, raw index and worktree bytes.

## Findings for review

- A full-SHA-looking branch name is deleted literally by `git branch -D -- <name>`. The delimiter keeps names out of option parsing.
- Multi-branch `git branch -D remove-me occupied missing` reports failure after removing `remove-me`; `occupied` remains because it is checked out, and `missing` remains absent. HEAD, index and worktree snapshots are recorded before and after.
- Deleting the branch named by bare repository HEAD succeeds. A branch checked out in a linked worktree is refused. The active-rebase fixture uses diverged source/target commits and an interactive sequence editor that inserts `break`; it records the native `rebase-merge/head-name` and `onto` files. Deletion is attempted from another worktree, followed by deletion of an unrelated branch to show the reservation boundary.
- With `branch.topic.merge` set twice, `for-each-ref` exposes only the first merge value; `git config --get-all` exposes both. Native `%(upstream)` retains remapped local tracking storage, while `%(upstream:remoteref)` remains the full server ref.
- Explicit `git push --no-follow-tags --delete -- named refs/heads/exact` succeeds. With mirror=true and an extra push refspec, the same explicit delete with `-c remote.named.mirror=false` deletes only `exact`; unrelated refs and the local tag are preserved.
- A single `git push --no-follow-tags --delete -- multi refs/heads/partial` starts with that branch present at both push URLs. It exits 1 after the first local bare remote removes it and the second pre-receive hook rejects deletion, leaving the second ref intact.
- TortoiseGit’s Browse All Refs guide describes context-menu deletion/renaming and local-branch tracked-branch metadata for the selected refs; it does not claim that deletion automatically removes another worktree: [Browse All Refs](https://tortoisegit.org/docs/tortoisegit/tgit-dug-browse-ref.html).

- **State — literal before SHA-named delete:** HEAD `refs/heads/main`; refs `refs/heads/d2ba49557002a758b983aeb2479e72c265dad5fa d2ba49557002a758b983aeb2479e72c265dad5fa; refs/heads/main d2ba49557002a758b983aeb2479e72c265dad5fa`; status `clean`.
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/literal` `$ git branch -D -- d2ba49557002a758b983aeb2479e72c265dad5fa` → exit 0: `Deleted branch d2ba49557002a758b983aeb2479e72c265dad5fa (was d2ba495).`
- `ASSERT` `$ "literal SHA-named branch was deleted"` → exit 0: `PASS`
- **State — literal after SHA-named delete:** HEAD `refs/heads/main`; refs `refs/heads/main d2ba49557002a758b983aeb2479e72c265dad5fa`; status `clean`.
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/literal` `$ git branch -D d2ba49557002a758b983aeb2479e72c265dad5fa` → exit 1: `error: branch 'd2ba49557002a758b983aeb2479e72c265dad5fa' not found`
- **State — multi-delete before:** HEAD `refs/heads/occupied`; refs `refs/heads/main ba250ac65057353f1928fe70dd4a0f7c8daeca9c; refs/heads/occupied ba250ac65057353f1928fe70dd4a0f7c8daeca9c; refs/heads/remove-me ba250ac65057353f1928fe70dd4a0f7c8daeca9c`; status `1 .M N... 100644 100644 100644 df967b96a579e45a18b8251732d16804b2e56a55 df967b96a579e45a18b8251732d16804b2e56a55 base.txt
1 .A N... 000000 000000 100644 0000000000000000000000000000000000000000 0000000000000000000000000000000000000000 untracked.txt`.
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/partial` `$ git branch -D remove-me occupied missing` → exit 1: `Deleted branch remove-me (was ba250ac). ; error: cannot delete branch 'occupied' used by worktree at '/tmp/lazygitk-branch-delete-probe-ur16pm9y/partial'
error: branch 'missing' not found`
- `ASSERT` `$ "multi-delete reports native failure"` → exit 0: `PASS`
- `ASSERT` `$ "multi-delete removed earlier eligible branch and retained occupied branch"` → exit 0: `PASS`
- `ASSERT` `$ "missing branch remains absent after multi-delete"` → exit 0: `PASS`
- **State — multi-delete after remove-me occupied missing:** HEAD `refs/heads/occupied`; refs `refs/heads/main ba250ac65057353f1928fe70dd4a0f7c8daeca9c; refs/heads/occupied ba250ac65057353f1928fe70dd4a0f7c8daeca9c`; status `1 .M N... 100644 100644 100644 df967b96a579e45a18b8251732d16804b2e56a55 df967b96a579e45a18b8251732d16804b2e56a55 base.txt
1 .A N... 000000 000000 100644 0000000000000000000000000000000000000000 0000000000000000000000000000000000000000 untracked.txt`.
- `ASSERT` `$ "multi-delete preserved HEAD-bytes-hex"` → exit 0: `PASS`
- `ASSERT` `$ "multi-delete preserved index-bytes-hex"` → exit 0: `PASS`
- `ASSERT` `$ "multi-delete preserved worktree-bytes-hex"` → exit 0: `PASS`
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/bare` `$ git branch -D main` → exit 0: `Deleted branch main (was 3ba14c3).`
- `ASSERT` `$ "bare HEAD-named branch deletion succeeded"` → exit 0: `PASS`
- **State — bare after deleting branch named by bare HEAD:** HEAD `refs/heads/main`; refs ``; status `bare (no worktree)`.
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/linked` `$ git branch -D reserved` → exit 1: `error: cannot delete branch 'reserved' used by worktree at '/tmp/lazygitk-branch-delete-probe-ur16pm9y/linked-wt'`
- `ASSERT` `$ "linked-worktree branch deletion was refused"` → exit 0: `PASS`
- **State — linked after reserved delete attempt:** HEAD `refs/heads/main`; refs `refs/heads/main a4dfa5116d17c7bf42ea82d623a8de94c18afb46; refs/heads/reserved a4dfa5116d17c7bf42ea82d623a8de94c18afb46`; status `clean`.
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/active-rebase-wt` `$ git rebase -i --onto 37573213cea2bf65472fdaa8ad5b7c0e76aa1d66 78fc273cf390b8942edd5d81815a79fad3e3af53 source` → exit 0: `Rebasing (1/2)
Stopped at 3757321 (target)`
- **State — active rebase after sequence-editor break:** HEAD `(detached)`; refs `refs/heads/main 37573213cea2bf65472fdaa8ad5b7c0e76aa1d66; refs/heads/source 38f7ab8bb7ca609ae57ac741804bcca949d2a7da; refs/heads/unreserved 78fc273cf390b8942edd5d81815a79fad3e3af53`; status `clean`.
- `FIXTURE` `$ "read /tmp/lazygitk-branch-delete-probe-ur16pm9y/active-rebase/.git/worktrees/active-rebase-wt/rebase-merge/head-name"` → exit 0: `refs/heads/source`
- `ASSERT` `$ "native active rebase metadata matches .git/rebase-merge/head-name"` → exit 0: `PASS`
- `FIXTURE` `$ "read /tmp/lazygitk-branch-delete-probe-ur16pm9y/active-rebase/.git/worktrees/active-rebase-wt/rebase-merge/onto"` → exit 0: `37573213cea2bf65472fdaa8ad5b7c0e76aa1d66`
- `ASSERT` `$ "native active rebase metadata matches .git/rebase-merge/onto"` → exit 0: `PASS`
- `ASSERT` `$ "paused rebase worktree HEAD is detached"` → exit 0: `PASS`
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/active-rebase` `$ git branch -D -- source` → exit 1: `error: cannot delete branch 'source' used by worktree at '/tmp/lazygitk-branch-delete-probe-ur16pm9y/active-rebase-wt'`
- `ASSERT` `$ "branch participating in paused rebase was retained"` → exit 0: `PASS`
- **State — main worktree attempt to delete branch reserved by paused linked rebase:** HEAD `refs/heads/main`; refs `refs/heads/main 37573213cea2bf65472fdaa8ad5b7c0e76aa1d66; refs/heads/source 38f7ab8bb7ca609ae57ac741804bcca949d2a7da; refs/heads/unreserved 78fc273cf390b8942edd5d81815a79fad3e3af53`; status `clean`.
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/active-rebase` `$ git branch -D -- unreserved` → exit 0: `Deleted branch unreserved (was 78fc273).`
- `ASSERT` `$ "unrelated branch deletion succeeded during paused rebase"` → exit 0: `PASS`
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/tracking` `$ git fetch named` → exit 0: `From /tmp/lazygitk-branch-delete-probe-ur16pm9y/tracking-remote
 * [new branch]      review     -> refs/custom/cache/review`
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/tracking` `$ git for-each-ref --format=%(refname)|%(upstream)|%(upstream:short)|%(upstream:remotename)|%(upstream:remoteref) refs/heads/topic` → exit 0: `refs/heads/topic|refs/custom/cache/review|custom/cache/review|named|refs/heads/review`
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/tracking` `$ git config --get-all branch.topic.merge` → exit 0: `refs/heads/review
refs/heads/other`
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/tracking` `$ git config --get-regexp ^branch\.topic\.` → exit 0: `branch.topic.remote named
branch.topic.merge refs/heads/review
branch.topic.merge refs/heads/other`
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/tracking` `$ git --no-optional-locks for-each-ref --format=%(upstream)|%(upstream:remotename)|%(upstream:remoteref) refs/heads/topic` → exit 0: `refs/custom/cache/review|named|refs/heads/review`
- `ASSERT` `$ "named remote upstream resolves remapped tracking ref and original remote ref"` → exit 0: `PASS`
- **State — tracking metadata:** HEAD `refs/heads/main`; refs `refs/heads/main 87d1c33beee861cc0873c52b6c32c649c3338f71; refs/heads/topic 87d1c33beee861cc0873c52b6c32c649c3338f71`; status `clean`.
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/push-local` `$ git push --no-follow-tags --delete -- named refs/heads/exact` → exit 0: `To /tmp/lazygitk-branch-delete-probe-ur16pm9y/push-bare
 - [deleted]         exact`
- **State — remote before explicit mirror-suppressed delete:** HEAD `refs/heads/master`; refs `refs/heads/exact 6489f00cd38bc4179fe8de5a94d48c530f5a9511; refs/heads/unrelated 6489f00cd38bc4179fe8de5a94d48c530f5a9511`; status `bare (no worktree)`.
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/push-local` `$ git -c remote.named.mirror=false push --no-follow-tags --delete -- named refs/heads/exact` → exit 0: `To /tmp/lazygitk-branch-delete-probe-ur16pm9y/push-bare
 - [deleted]         exact`
- `ASSERT` `$ "mirror-suppressed explicit delete changed only exact branch ref"` → exit 0: `PASS`
- `ASSERT` `$ "explicit remote deletion left local tag intact"` → exit 0: `PASS`
- **State — remote after explicit mirror-suppressed delete:** HEAD `refs/heads/master`; refs `refs/heads/unrelated 6489f00cd38bc4179fe8de5a94d48c530f5a9511`; status `bare (no worktree)`.
- `ASSERT` `$ "partial branch was seeded at both push URLs"` → exit 0: `PASS`
- **State — multi-pushurl remotes before:** HEAD `refs/heads/master`; refs `refs/heads/partial e683abeee372952ed7350004a3141443f866327e`; status `bare (no worktree)`.
- **State — multi-pushurl remotes before:** HEAD `refs/heads/master`; refs `refs/heads/partial e683abeee372952ed7350004a3141443f866327e`; status `bare (no worktree)`.
- `/tmp/lazygitk-branch-delete-probe-ur16pm9y/multiurl-local` `$ git push --no-follow-tags --delete -- multi refs/heads/partial` → exit 1: `To /tmp/lazygitk-branch-delete-probe-ur16pm9y/multiurl-first
 - [deleted]         partial
remote: second-reject
To /tmp/lazygitk-branch-delete-probe-ur16pm9y/multiurl-second
 ! [remote rejected] partial (pre-receive hook declined)
error: failed to push some refs to '/tmp/lazygitk-branch-delete-probe-ur16pm9y/multiurl-second'`
- `ASSERT` `$ "multi-pushurl deletion returns native failure status 1"` → exit 0: `PASS`
- `ASSERT` `$ "multi-pushurl delete removed first ref and retained second after hook rejection"` → exit 0: `PASS`
- **State — multi-pushurl first after command:** HEAD `refs/heads/master`; refs ``; status `bare (no worktree)`.
- **State — multi-pushurl second after command:** HEAD `refs/heads/master`; refs `refs/heads/partial e683abeee372952ed7350004a3141443f866327e`; status `bare (no worktree)`.
