# Selected-branch menu integration audit

Current item: Roadmap 7. This records inspected integration points, not new
scope or a plan for later roadmap items.

| Entry point | Finding | Behavior to retain |
| --- | --- | --- |
| `BranchesController.viewUpstreamOptions` → `branch_upstream_menu.go` | Captured parent menu retains the selected name, model and repository generation, resolves native full tracking refs, and revalidates before reset/rebase submenus. Those submenus use their captured execution flows too. | Selected-branch/base divergence; separate set/unset; explicitly current-destination reset/rebase. |
| `BranchesController.createResetMenu` / `RefsHelper.CreateGitResetMenu` → `helpers.ResetCurrentToCapturedSource` | Captured shared flow binds source commit, current HEAD, repository and generation; hard reset always confirms. Files/upstream resolves the captured current branch's native tracking ref; tag refs peel to commits. | Soft/mixed/hard reset of current HEAD, including detached HEAD and explicitly aborting a merge. This is not a soft/mixed/hard reset of an off-branch ref. |
| `BranchesController.rebase` / remote branch / upstream rebase → `helpers/captured_rebase*.go` | Native plans and captured GUI flow retain full source refs, current HEAD, marked base and repository identity through review, ignored-file approval, terminal launch and refresh completion. Paused errors remain errors; conflict abort checks the original operation. | Rebase current onto selected; interactive initial break/todo editing, marked-base and base-branch choices, signing and explicit empty-replay decisions. |
| Local/remote branch merge → `helpers/captured_merge*.go` | Captured native reviews retain full source/current identities, configuration and staged entries through mode/message/approval/terminal callbacks. Squash commit is a separately validated phase; conflicts and no-ops cannot trigger an unrelated commit. Abort checks the original native operation. | Configured/allow-FF/no-FF, both squash modes, messages, terminal signing/editor support and native partial errors. |
| `BranchesController.delete` / remote branch deletion → `helpers/captured_branch_delete*.go` | Native target/configuration/proof snapshots retain source/current/repository through destination choice, approval, command and actual-state refresh. Ambiguous upstreams use explicit full server refs; compound deletion excludes removed caches from retained-history proofs. | Explicit local/remote/both choices, force warnings and native partial errors. Occupied local branches are refused; remote-only remains available. No implicit worktree detach/removal. Explicit Worktrees-pane removal remains separate. |
| `BranchesController.newBranch/createTag` → `helpers/captured_{branch,tag}_create.go` | Native source/current/configuration/destination plans and repository-generation guards retain target identity across literal entry, tracking/overwrite review, workers, signing and refresh. Remote entry uses full refs; stash entry uses its immutable commit; checkout-by-name creation retains its original selected source. | Explicit create-and-checkout with configured/no-setup/direct tracking choices; native dirty-preserving checkout and ignored collision refusal without stash. Retain tag annotation/signing and exact old-object overwrite approval. |

Read-only menus need a copied literal source name and repository-generation
guard, plus Git's exact native tracking ref. Mutations additionally need the
reviewed source/current-HEAD identity revalidated before launch and commands
bound to the captured backend. Do not introduce a generic GUI forwarding proxy:
it would hide the command, worker and refresh boundaries that need protection.
Keep Git's normal concurrency model; no custom repository locks, automatic
retry, rollback, checkout or stash. Preserve errors from native inspection.

For reset, [TortoiseGit's reset guide](https://tortoisegit.org/docs/tortoisegit/tgit-dug-reset.html)
confirms that the destination is current HEAD, soft/mixed/hard have different
index/worktree effects, and mixed/hard can abort a merge. Consequently, do not
blanket-disable reset merely because a merge is in progress. Hard reset can
discard uncommitted data; its explicit warning must not rely only on a stale
cached dirty-file model.

[TortoiseGit's rebase guide](https://tortoisegit.org/docs/tortoisegit/tgit-dug-rebase.html)
distinguishes the branch being rewritten from the upstream target. Retain that
distinction in the captured branch rebase flow, including its marked-base and
interactive-break choices. Its off-branch rebase capability remains outside
this fork's agreed first-version scope.
