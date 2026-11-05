# Review 08: checkout

## Summary
Read-only review completed against BASE `4e1eebe278790f257a875da6633234391a5a85b3`.
The named-checkout backend carefully protects dirty/ignored files, validates literal destinations, rechecks HEAD/destination snapshots, and respects worktree reservations.
Its safeguards are not consistently applied: the default palette's typed checkout retains upstream's unsafe command, detached checkout lacks operation guards, and remote checkout creates a branch before guarded checkout begins.
Git probes confirmed ignored-file overwrite, path-based discard, merge-state removal, and ambiguous-name failures/fallbacks.
Most implementation is additive. The four scoped upstream helper/controller files total 74 additions / 144 deletions; checkout-specific changes in refs_helper.go are mostly metadata plus an additive menu entry, rather than wholesale replacement.
No repository files were changed, no commits were made, and no Go/integration tests were run, as required by this request.

## Findings

### F1. Default palette checkout by name can discard files and still offers autostash
- Category: bug
- Severity: high
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/branches_controller.go:402, pkg/gui/controllers/branches_controller.go:425, pkg/gui/controllers/helpers/refs_helper.go:34, pkg/commands/git_commands/branch.go:151
- Problem: Removing the key from `branch.checkout-by-name` does not remove its executable palette action. Guide.md explicitly advertises "Checkout by name" in `?`. Its handler still calls `CheckoutRemoteBranch` / `CheckoutRef`, ultimately running plain `git checkout <input>` with neither literal-ref classification nor `--no-overwrite-ignore`. Dirty rejection also offers upstream autostash. The default handler replacement only covers four selected-object IDs, not this ID.
- Scenario / evidence: In `/tmp/probe-checkout-ignored-xyy41acc`, plain `git checkout target` succeeded and replaced an ignored file containing `precious ignored bytes` with the target's tracked bytes. After editing that tracked file, `git checkout ignored` (a path, with no such branch) succeeded and discarded the edit. These are precisely the argv constructed by the typed-name path. No confirmation is required for either loss. This contradicts both literal/ambiguous-input refusal and checkout file-protection rules.
- Suggested fix: Route the default palette action through a fork-owned typed checkout classifier and guarded checkout flows. Refuse ambiguous names and path input before invoking checkout. Keep upstream behavior reachable only through the intended explicit legacy mechanism. Add default-palette execution tests, including ignored collisions and path names.

### F2. Detached checkout silently abandons an unfinished merge
- Category: bug
- Severity: high
- Confidence: CLEAR BUG
- Location: pkg/commands/git_commands/captured_detached_checkout.go:6, pkg/gui/controllers/helpers/fork_flat_checkout.go:161, pkg/commands/git_commands/worktree_operation_guard.go:11
- Problem: `CheckoutCapturedCommit` validates only the OID; it never calls the shared active-operation guard. Thus `s d` and the detached fallback of `s p` bypass the check applied to named checkout. Git itself permits some such transitions and clears operation metadata.
- Scenario / evidence: In `/tmp/probe-checkout-real-z88327qq`, created divergent branches, performed a real clean `git merge --no-commit side`, then ran the backend's exact command `git checkout --detach --no-overwrite-ignore --no-guess <HEAD>^{commit}`. Exit 0; MERGE_HEAD changed from present to absent; the merged file remained staged (`A  side`). A subsequent commit would lose the merge parent unless the user reconstructs the operation. An earlier synthetic MERGE_HEAD probe produced the same removal. No finish/abort decision was offered.
- Suggested fix: Apply `rejectWorktreeOperation` in the detached backend immediately before checkout, with operation-specific rejection text. Cover real pending merge plus other guarded markers, including the detached previous-checkout route.

### F3. Remote checkout writes before its safety checks and recaptures a mutable target
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/helpers/fork_flat_checkout.go:31, pkg/gui/controllers/helpers/fork_flat_checkout.go:43, pkg/gui/controllers/helpers/fork_flat_checkout.go:54
- Problem: With no matching local branch in the model, remote checkout synchronously creates a tracking branch before checking active operations or whether checkout will fail. It then refreshes and resolves the newly created local branch again rather than retaining the chosen remote commit. Failure leaves a branch and tracking configuration created by an operation presented as checkout; a ref change between creation and the refresh callback can redirect checkout without a changed-target review.
- Scenario / evidence: End-to-end code trace: `CreateWithUpstream` runs `git branch --track topic origin/topic` first; only the later `checkoutLocalBranchNamed` reaches `PrepareNamedCheckout` and `rejectNamedCheckoutOperation`. With MERGE_HEAD present, or tracked changes colliding with the remote tree, branch creation succeeds and checkout subsequently rejects. If an external process moves `topic` during the refresh, the callback resolves that new tip and prepares it as the original target. The named backend's snapshot checks cannot detect the earlier change because the snapshot is taken afterward.
- Suggested fix: Capture the remote's full-ref OID and repository first; perform operation guards before writes; use a combined guarded create/checkout flow with explicit tracking semantics. Retain and revalidate the captured target across refreshes instead of resolving the new local branch as a fresh source. Do not add assumed rollback.

### F4. Previous branch with a same-named tag silently becomes detached checkout
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/helpers/fork_flat_checkout.go:97, pkg/commands/git_commands/branch.go:121, pkg/commands/git_commands/fork_previous_checkout.go:12
- Problem: The previous-branch classifier treats every successful empty `PreviousRef` result as a genuinely detached previous checkout. Git also returns successful empty stdout when `@{-1}` has an ambiguous branch/tag name, emitting its error only on stderr. The fallback then resolves a commit and detaches instead of refusing ambiguity or switching to the named branch displayed by the menu.
- Scenario / evidence: In `/tmp/probe-checkout-real-z88327qq`, switch from `master` to `side` with a tag also named `master`. `git rev-parse --symbolic-full-name '@{-1}'` exits 0 with empty stdout and an ambiguity warning/error. `git rev-parse --verify --end-of-options '@{-1}^{commit}'` exits 0 with a commit and an ambiguity warning. The fork therefore takes lines 111–116 and detaches. The label parser still displays `master`.
- Suggested fix: Distinguish an actual detached reflog source from ambiguous resolution. Resolve a captured previous branch through `refs/heads/...`, or refuse ambiguity explicitly. Add branch/tag collision tests through the real previous-checkout handler.

### F5. Selected remote checkout passes an ambiguous short ref to Git
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/helpers/fork_flat_checkout.go:43, pkg/commands/git_commands/branch.go:54
- Problem: The remote pane identifies an exact remote branch, but `remote.FullName()` is `origin/topic`, not `refs/remotes/origin/topic`. `CreateWithUpstream` passes it directly as the start point. A same-named tag makes checkout fail despite the selected source being unambiguous.
- Scenario / evidence: In `/tmp/probe-checkout-84pibuy7`, created `refs/remotes/origin/topic` and `refs/tags/origin/topic`, with a valid origin fetch configuration. Exact command `git branch --track topic origin/topic` exited 128: `fatal: ambiguous object name: 'origin/topic'`. Other fork selected-ref operations use full refs specifically to prevent this.
- Suggested fix: Use the selected remote branch's full ref, capture its literal commit, and preserve tracking separately. Test remote/tag collisions and two remotes exposing the same branch name.

### F6. Selected checkout tests exercise an unused menu instead of the default flat handlers
- Category: code-quality
- Severity: low
- Confidence: n/a
- Location: pkg/gui/controllers/helpers/selected_ref_checkout.go:15, pkg/gui/controllers/helpers/selected_ref_checkout_test.go:32, pkg/gui/fork_flat_checkout_keys_test.go:11
- Problem: `CreateSelectedRefCheckoutMenu` has no production callers (repository-wide search finds only its definition and tests). Its tests assert remote checkout offers naming instead of choosing a local branch, whereas production `CheckoutSelectedRemoteBranch` explicitly chooses the same-named local branch or creates one. Key tests check labels and binding existence, not actual execution. Consequently the real remote path, its early writes, and detached operation guard omissions are uncovered.
- Scenario / evidence: Read all selected-ref checkout tests and named-checkout test cases; searched production callers and default projection. Named-checkout tests do cover dirty/ignored collisions, changed snapshots, operation markers, linked/bare worktrees, SHA-256, hook failures, and retry suppression. Those tests validate the named backend rather than the distinct flat remote implementation. Previous-checkout tests cover only the disabled-reason helper, not reflog parsing/execution.
- Suggested fix: Retire the unused selected-ref menu if no longer intended, or explicitly document its role. Exercise actual projected handlers with real repositories, especially remote creation failures, ref changes during refresh, typed checkout, and previous-checkout classification.

### F7. Merge guide overstates checkout migration to captured flows
- Category: docs
- Severity: low
- Confidence: n/a
- Location: docs-master/fork/MergeGuide.md:299, docs-master/fork/MergeGuide.md:375
- Problem: The guide says `CheckoutRemoteBranch` remains only for explicit legacy configuration and that remote checkout / checkout menus use captured selections. The default typed-name palette path still calls that legacy remote helper, and `CreateCheckoutMenu` still contains upstream checkout actions. A separate unused captured menu does not replace them. These statements hide exactly the integration gap in F1.
- Scenario / evidence: `checkoutByName` remains visible without a key (Guide.md:532); projection replaces only branch.checkout, remote-branch.checkout, tag.checkout, and commit.checkout-menu. `forkReplacedCommand` does not hide checkout-by-name.
- Suggested fix: Correct the integration inventory alongside the F1 repair, distinguishing production default handlers, explicitly legacy handlers, and unused code.

## Merge-ease notes
- `pkg/gui/controllers/helpers/refs_helper.go`: total +33/-135, including out-of-area reset/new-branch delegation. Checkout-specific edits are menu command IDs and one commented `namedCheckoutItem` insertion, a reasonable additive seam. Do not interpret this as replacement of upstream `CheckoutRemoteBranch` or `CreateCheckoutMenu`; their old paths remain. The misleading inventory is F7.
- `pkg/gui/controllers/helpers/worktree_helper.go`: +22/-4, essentially command/verb metadata and two explanatory comments. Small, maintainable footprint; no fork replacement of worktree creation/switching logic here.
- `pkg/gui/controllers/switch_to_sub_commits_controller.go`: +14/-0. `WithoutDoubleClick` adds a small opt-out with a Fork comment and documented Worktrees rationale. Reasonable hook; retain its nil double-click behavior on merges.
- `pkg/gui/controllers/worktrees_controller.go`: +10/-0, command/verb metadata only. Minimal conflict surface.
- `pkg/gui/controllers/branches_controller.go` (relevant integration inspected): total +128/-353 spans other review areas; checkout-by-name retains its upstream handler with added repository/source capture. Its existing Fork comment is helpful, but does not explain that unsafe upstream execution remains accessible through the default palette.
- All other requested implementation modules are additions relative to BASE, including named/detached/previous backends, operation guard, flat checkout/suggestions/menu helpers, selected-checkout projection, and Worktrees ref adapter. This generally follows the fork-module principle. `fork_ref_commit.go` contains read-only copy/ref lookup helpers rather than checkout mutation.

## Out of scope observations
None requiring a separate finding. Existing untracked `HANDOFF.md`, `reviews/`, and `test/.viminfo` were present initially and left untouched.

## Unresolved questions (no design phase started)
- When a selected remote branch has a same-named local branch at another commit or tracking another remote, should `s c` silently switch to that local branch? Current flat code does so; prior selected-menu tests explicitly reject such silent selection. The current Guide does not settle this remote-specific alternative. Obtain a user decision before changing this behavior.
- Guide.md says new branches show a tracking choice before creation. Named checkout from a tag/commit directly runs `checkout -B <name> <OID>^{commit}` with no tracking review; remote flat checkout unconditionally uses `--track`. Is that sentence intended to cover these checkout-created branches? Clarify the documented scope rather than inventing a policy during repair.

## Commands run and outcomes
- Read `/tmp/lazygitk-review-run-1793910387/08-worker-request.md`, `common-review-rules.md`, repository `AGENTS.md`, Tracker.md, Design.md, and checkout portions/searches of Guide.md and MergeGuide.md. Followed the specific read-only request over the generic implementation/commit workflow.
- `git status --short` initially and near completion: same three pre-existing untracked paths; no tracked changes introduced.
- `git diff 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <scoped upstream paths>`; `git diff --stat` / `--numstat` for scoped files; `git log --oneline 4e1eebe278790f257a875da6633234391a5a85b3..HEAD -- <checkout and helper paths>`: reviewed integration footprint/history, including flat checkout commit `07d445307` and named backend/modal commits and fixups.
- `cat`, `sed`, `nl -ba`, and `rg` over requested modules/tests and immediate production callers: traced default projection, typed checkout, named preparation/apply, detached checkout, previous-ref resolution, worktree occupancy, repository-generation guards, and reflog loading. Some exploratory `rg`/`cat` guesses named nonexistent files; subsequent file discovery located the actual modules. No source was written.
- Python heredoc Git probes, entirely under `/tmp/probe-*`:
  - `/tmp/probe-checkout-84pibuy7`: ambiguous remote/tag tracking creation exited 128; detached checkout with synthetic MERGE_HEAD exited 0 and removed marker.
  - `/tmp/probe-checkout-real-z88327qq`: real pending merge detached checkout exited 0 and removed MERGE_HEAD while leaving merged content staged; previous branch/tag collision produced successful empty symbolic resolution and successful commit fallback. The script subsequently stopped on an unrelated ambiguity while setting up its last ignored-file experiment; that experiment was rerun independently below.
  - `/tmp/probe-checkout-ignored-xyy41acc`: plain checkout overwrote ignored bytes; path-only checkout discarded tracked edits, both exit 0.
- No Go tests, builds, vet, generation, formatting, or integration runs: Git probes and end-to-end code traces sufficed to confirm the suspected defects, so no targeted Go test was needed under the request's restrictive test policy.
- Only handoff artifact intentionally written outside the permitted throwaway probe repositories: `/tmp/lazygitk-review-run-1793910387/08-worker-result.md`.

## Remaining problems and recommended next step
All findings remain unfixed because this worker was explicitly read-only. Prioritize F1's file-loss path and F2's operation-state loss, with targeted regression tests against actual default handlers. Then repair the remote creation/capture and name-classification gaps. Resolve the two recorded behavior questions before changing remote collision or tracking policy. Update MergeGuide with the final integration seams and replace misleading coverage from the unused menu. Do not expand into a new design phase as part of this review handoff.
