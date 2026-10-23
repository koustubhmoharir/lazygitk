# Remaining selected-branch menu audit

Current item: Roadmap 7. This records inspected integration points, not new
scope or a plan for later roadmap items.

| Entry point | Finding | Behavior to retain |
| --- | --- | --- |
| `BranchesController.viewUpstreamOptions` | Divergence callbacks read the selection again. `FullUpstreamRefName` reconstructs `refs/remotes/<remote>/<branch>`, which is wrong for local `.` upstreams and custom tracking namespaces. | Selected-branch/base divergence; separate set/unset; explicitly current-destination reset/rebase. |
| `BranchesController.createResetMenu` → `RefsHelper.CreateGitResetMenu` | The source name is captured, but its tip, current HEAD, repository and later reset worker are not pinned together. | Soft/mixed/hard reset of current HEAD, including detached HEAD and explicitly aborting a merge. This is not a soft/mixed/hard reset of an off-branch ref. |
| `BranchesController.merge/rebase` → `MergeAndRebaseHelper` | Nested workers use live `c.Git()`; rebase also reads marked-base state at a later callback. | Merge selected source into current; rebase current onto selected. Retain regular/no-FF, both squash modes, interactive todo editing, marked-base and base-branch rebase choices. |
| `BranchesController.delete` → `BranchesHelper` | Nested menus retain mutable model pointers and live repository lookups. Single occupied-branch deletion can offer detaching/removing its worktree. | Explicit local/remote/both choices, force warnings and native partial-failure reporting; enforce the roadmap's occupied-branch policy without silently altering another worktree. |
| `BranchesController.newBranch/createTag` → shared helpers | Initial selected-ref support exists, but later prompts/workers use live Git objects or resolve the source ref later. | Existing create-and-checkout, tracking choices, tag annotation/signing and overwrite warnings; do not silently change these capabilities while fixing target capture. |

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
