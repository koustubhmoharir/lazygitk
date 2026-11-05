# Review 14: drafts-repos

## Summary
Read-only source review against BASE `4e1eebe278790f257a875da6633234391a5a85b3`.
The draft state machine is mostly isolated in a fork-owned context module; controller hooks and the optional Tab/refresh callbacks are small and documented.
The largest risks are concurrent draft-file corruption and incomplete restoration after a late repository-switch failure.
Tab completion also performs unbounded filesystem work on the UI thread.
No implementation, repository edits, commits, generation, builds, or tests were performed, as required by the worker request.
The only file created is this result artifact. Findings below are established by source tracing unless marked otherwise.

## Findings

### F1. Concurrent instances can corrupt or lose the entire drafts list
- Category: bug
- Severity: high
- Confidence: CLEAR BUG
- Location: pkg/gui/context/fork_commit_draft.go:108-126; pkg/gui/context/fork_commit_draft.go:206-213
- Problem: Every writer uses the same `LAZYGITK_COMMIT_DRAFTS.tmp` pathname. The read/modify/write sequence is also uncoordinated across instances. Atomic rename alone does not make either sequence safe.
- Scenario / evidence: Two instances editing the same worktree both read list D. A prepares D+A and B prepares D+B. Even non-overlapping temporary writes can replace one another's updates. Worse: A writes the temporary file; B opens that same file with truncation; A renames it while B still has its descriptor open. B then writes directly into the inode now visible as the final drafts file. A reader can see incomplete JSON; overlapping writes of different lengths can leave malformed JSON. A subsequent load treats malformed JSON as an empty list (lines 92-95), and the next save discards all older drafts. The second rename can also fail with a missing temporary file, only logged. Separate linked worktrees use separate WorktreeGitDirPath values, so this collision concerns multiple instances in the same worktree, rather than all worktrees sharing a repository.
- Suggested fix: Use a uniquely created temporary file for each write. Separately settle and implement the cross-instance read/modify/write policy; a unique temporary file fixes inode sharing but does not prevent lost updates. See the design question below before choosing synchronization behavior.

### F2. A late switch failure leaves the old model paired with the new Git repository
- Category: bug
- Severity: high
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/helpers/repos_helper.go:389-393,425-426; pkg/gui/gui.go:401-405; pkg/gui/controllers/helpers/fork_repo_switch.go:52-61
- Problem: The new deferred pane restoration assumes an unsuccessful onNewRepo leaves the old repository intact. In fact, onNewRepo assigns gui.git before loading repository config, and switchToLocation restores cwd/environment only for Chdir/VerifyInGitRepo failures. It does not restore them when onNewRepo fails. This is an inherited early-mutation problem that the fork's new failure-restoration flow and Guide promise do not handle.
- Scenario / evidence: Start in A, open valid repository B whose `.git/lazygitk.yml` contains invalid YAML. VerifyInGitRepo succeeds, cwd changes to B, NewGitCommand succeeds and assigns gui.git=B, and ReloadUserConfigForRepo returns an error before resetState. The deferred restore renders A's contexts/model, but its STATUS refresh now reads B. Subsequent actions obtain B's Git command object while selections still describe A. The path prompt does not reopen because cwd differs from originalDir (fork_open_repo_path.go:74-81). The existing switch integration tests cover deleted paths and invalid Git directories, which fail earlier and do not exercise this case.
- Suggested fix: Make the switch initialization failure-safe: prepare the new Git/config state before publishing it, and restore cwd/environment and any published state on initialization failure. Add a narrowly scoped regression for a valid target repository with invalid repository config and verify both the displayed model and the next command still address A.

### F3. Draft persistence failure silently closes the only recoverable message
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/context/fork_commit_draft.go:102-105; pkg/gui/controllers/helpers/commits_helper.go:266-274,220-225
- Problem: Draft saves return no success/error to their callers. Escape closes the panel regardless of persistence failure; asynchronous submissions start regardless of whether the safety draft was written. Only a log entry reports the failure.
- Scenario / evidence: Open a reword box (`PreserveMessage: false`), type a different message, and press Escape while the worktree Git directory cannot accept a temporary file, or while the filesystem is full. saveCommitDrafts logs the error, CancelCommitMessagePanel closes the panel, and neither the preserved-message file nor the drafts file contains the edited message. A failed hook after an unsuccessful pre-submission save has the same missing recovery text. This contradicts the documented recovery behavior and cannot be corrected by reopening and pressing Up.
- Suggested fix: Propagate persistence errors through the fork-owned draft API, retain the panel/text on a failed cancellation save, and make the failed recovery-save state visible before dispatching a submission. Treat the quit failure behavior as a separate explicit policy question rather than silently claiming persistence.

### F4. Tab completion blocks the UI on slow or large directories
- Category: ux
- Severity: medium
- Confidence: n/a
- Location: pkg/gui/controllers/helpers/fork_open_repo_path.go:89-94,155-173,184-195; pkg/gui/controllers/prompt_controller.go:51-55
- Problem: The Tab handler calls matchRepoPathDirs synchronously on the UI thread. That reads and sorts the whole directory, scans all matches, and stats matching symlinks. The 15-item suggestion limit is applied only after the full scan and does not bound Tab work. Ordinary suggestion searches use the asynchronous SuggestionsContext path, but Tab bypasses it.
- Scenario / evidence: Type a path on a slow network filesystem or into a directory with many matching symlinks, then press Tab. Input, Escape, redraw, and other UI actions wait for os.ReadDir and each os.Stat to finish. The existing tests use small local temporary trees and cannot establish responsiveness.
- Suggested fix: Dispatch Tab filesystem work through an asynchronous path, capture the prompt/input identity, and apply a completion only while that same prompt/input remains current. Preserve complete match enumeration for correct common-prefix calculation rather than applying the display cap to completion results.

### F5. Retained upstream index comments still promise removed history recall
- Category: docs
- Severity: low
- Confidence: n/a
- Location: pkg/gui/controllers/custom_patch_options_menu_action.go:181-183,220-222; pkg/gui/context/commit_message_context.go:25-28
- Problem: Both custom-patch call sites still say their CommitIndex value lets Up recall the original commit message. Up now reads cancelled drafts exclusively. The NoCommitIndex comment also describes the old existing-commit navigation model without identifying it as inert compatibility plumbing.
- Scenario / evidence: The removal of handleCommitIndexChange and GetCommitMessageFromHistory is explicit in the BASE diff. The field/option/setter comments correctly explain the compatibility trade-off, but these remaining comments contradict it. MergeGuide.md:458-461 says the retained call-site indexes drive nothing and their Fork comments explain why; these two call sites retain the opposite explanation.
- Suggested fix: Keep the inert values and parameters to avoid unnecessary merge surface, but replace these obsolete comments with concise Fork compatibility comments. Adjust the merge-guide wording if it is intended to describe only the shared declarations rather than each call site.

## Merge-ease notes

Counts below are full-file additions/deletions against BASE; mixed-purpose files include changes outside this review area. They are footprint indicators, not a claim that every changed line belongs to drafts/repos.

| Upstream file | Full-file + / - | Assessment for this area |
| --- | --- | --- |
| pkg/gui/context/commit_message_context.go | 25 / 16 | Session storage/reset and subtitle hook are reasonable integration points. The disk/state-machine implementation stays in fork_commit_draft.go. Most inert index comments are useful; see F5. |
| pkg/gui/controllers/commit_message_controller.go | 34 / 54 | Replaces history handlers with draft delegates and removes the old implementation. Fork comments explain cancellation and recall; the replacement is documented. |
| pkg/gui/controllers/commit_description_controller.go | 20 / 9 | Small cancellation/quit delegates; other churn is command metadata. |
| pkg/gui/controllers/helpers/commits_helper.go | 101 / 5 | Largest draft integration surface: report ownership, lifecycle hooks, recall wrappers. Most persistence/session behavior is correctly isolated. The TakeCommitMessageSubmitReporter contract and every asynchronous call site must remain aligned on merges. Further extraction could move the fork-only helper methods into a fork file without changing the public seam. |
| pkg/gui/controllers/helpers/working_tree_helper.go | 38 / 6 | Draft changes are the opt-in, reporter capture, and success call. Other changes concern tools. No large draft replacement is needed here. |
| pkg/gui/controllers/local_commits_controller.go | mixed, not attributed | Reviewed reword/amend draft portions only: opt-in/report calls and rewordFailure delegate are localized and commented. Avoid expanding this already heavily modified controller for fixes. |
| pkg/gui/controllers/custom_patch_options_menu_action.go | mixed, not attributed | Reporter hooks are small; obsolete retained-index comments need correction (F5). |
| pkg/gui/controllers/helpers/repos_helper.go | 15 / 1 | Loading/restoration is fork-owned; upstream edit is a named result, call, and deferred hook. Sensible seam, but failure invariant is incomplete (F2). |
| pkg/gui/controllers/prompt_controller.go | 14 / 0 | Optional Tab hook before existing focus behavior is minimal and commented. |
| pkg/gui/controllers/helpers/confirmation_helper.go | 2 / 0 | Small assignment/reset hooks, both commented. |
| pkg/gui/context/confirmation_context.go | 2 / 0 | One documented callback field. |
| pkg/gui/types/common.go | 45 / 6 | Area-specific changes are only HandleTab option fields/comments; most full-file churn belongs to other work. |
| pkg/gui/controllers/helpers/refresh_helper.go | 28 / 1 | ThenIfRepoUnchanged contributes a 13-line inline guarded callback; other edits are filters/metadata. Its ordering relative to the batch, Then, and input unblock is clear. |
| pkg/gui/types/refresh.go | 8 / 0 | Callback contract clearly documented. |
| pkg/gui/gui.go | 67 / 3 | This area's loadNewRepo integration is two lines; other file changes are unrelated. The failure behavior of onNewRepo matters despite predating this fork (F2). |
| pkg/commands/git_commands/commit.go | 0 / 15 | Deletes unused history lookup and its error. MergeGuide records the deletion; restoring it during a merge would be dead code. |

Additive files fork_commit_draft.go, helpers/fork_open_repo_path.go, helpers/fork_repo_switch.go, gui/fork_repo_switch.go, and gui/fork_full_refresh.go keep the bulk of behavior out of upstream functions.

Retaining inert CommitIndex/SetPanelState/selectedindex plumbing is a reasonable merge trade-off: removing it would touch many unrelated panel callers. The remaining comments should consistently state that it has no effect.

The MergeGuide covers draft reporting call sites, prompt HandleTab pass-throughs, refresh callback generation protection, loading feedback, completion toast, and F5's graph-cache drop. No missing major integration entry was found in this area.

## Additional checks and unresolved questions

- Draft files use WorktreeGitDirPath rather than the shared RepoGitDirPath, and sit beside the preserved pending message as documented. Source-level placement is correct for linked worktrees. The placement unit test uses MockRepoPaths for an ordinary repository, so it does not independently prove a linked-worktree end-to-end flow.
- ThenIfRepoUnchanged is queued after the batch apply and Then and before EndBlockingEvents, and checks the captured generation on the UI thread. No direct ordering bug was established. Its tests cover STATUS updates, changed generations, and DontBlockRepoSwitch, but do not exercise BatchUIUpdates plus input blocking together or nested layout work. Do not read those three unit tests as coverage of every documented ordering detail.
- F5 drops the Git config cache and graph pipe-set cache, explicitly includes selected-ref history and dependent scopes, and guards its redraw callback. No clear F5 defect was established in this review.
- Draft data is persisted on cancellation, navigation away from an edited recalled draft, and asynchronous submission, rather than on each keystroke. An ungraceful process kill can lose the currently typed text; rename without fsync also does not promise durability across a machine crash. No documented autosave/power-loss guarantee was found, so this is a limitation rather than a new bug finding.
- **Question — concurrent instances:** Design.md says the fork adds no locks/retries, in the native-Git operations rule. What consistency should application-owned draft storage guarantee across concurrent instances of the same worktree, and does that rule constrain its storage mechanism? F1's shared-temp collision is unambiguous, but the desired read/modify/write consistency needs clarification before implementing a broader policy. No policy was assumed or implemented.
- **Question — unchanged prefilled submissions:** ReportsSubmitResult uses KeepCommitDraftsOnCancel, which deliberately does not add text equal to the opened message. Thus a never-edited prefilled reword/amend/template message may have no submission draft even when a hook rejects it. Guide.md says a submitted message stays a draft until success, while its cancellation rule exempts unchanged text. Is submission intended to override that exemption? Existing rejection scenarios modify the prefill. This was not treated as user-authored text loss or used to start a design phase.
- Switching to an editor writes the visible text into a separate message file and keeps displaced typed text as a draft. Edits to a recalled draft are not additionally flushed back to that draft at editor handoff; the original draft remains. The normal editor-failure path therefore retains the visible text in the handoff file, but the Guide does not specify automatic recovery/import of later external-editor edits.

## Out of scope observations

No additional cross-area finding is proposed. The early state publication in F2 predates the fork, but is included because the requested repository-switch failure-restoration behavior directly depends on it.

## Commands run and outcomes

- `cat /tmp/lazygitk-review-run-1793910387/14-worker-request.md` and `cat /tmp/lazygitk-review-run-1793910387/common-review-rules.md`: read the scope, hard read-only constraints, permitted commands, and result format.
- `cat AGENTS.md docs-master/fork/Design.md`, `cat docs-master/fork/Tracker.md`, and targeted `rg`/`sed` reads of Guide.md and MergeGuide.md: established current behavior and merge requirements; tracker says no current implementation work.
- `git status --short` at start and near completion: same pre-existing untracked entries, `HANDOFF.md`, `reviews/`, and `test/.viminfo`. None was opened for editing, removed, or committed.
- `git diff --stat 4e1eebe278790f257a875da6633234391a5a85b3 -- ...`: initial footprint inventory; the controllers directory argument included unrelated files, which were not expanded into review work.
- `git diff 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <scoped paths>` and `git diff --numstat ...`: examined the fork integrations/deletions and collected the per-file footprint above.
- `git log --oneline 4e1eebe278790f257a875da6633234391a5a85b3..HEAD -- <fork draft/repository/full-refresh paths>`: traced feature commits including a1dcb6277, b809bcce0, e43e5d1b2, and 03583409d.
- Targeted `cat`, `sed`, `nl -ba`, and `rg` reads: traced the draft context/helper, summary and description controllers, worker reporters in working-tree/reword/amend/patch paths, GPG success callbacks, repository path completion/switch/loading helpers, refresh callback finalization, and minimal dependent config/patch/merge result handling.
- Read draft context tests, repository path/switch helper tests, refresh callback tests, both fork repository integration scenarios, and failed-reword draft integration scenarios. Integration tests were inspected only, never run.
- A few exploratory `rg` calls named nonexistent guessed paths/globs and returned exit 2; existing files were located via broader rg searches. This did not affect the findings.
- Tests: none. Existing targeted tests do not confirm the concurrent write/late-init-failure/UI-blocking scenarios. No test-only source was created and no broad or integration tests were run.
- No build, vet, formatter, generator, or network command was run. No TortoiseGit lookup/design consultation was initiated because no new design choice was implemented; unresolved policy questions are reported above.

## Work performed, remaining problems, and next step

Work performed: completed the scoped read-only investigation and wrote this handoff. Repository files changed: none. New artifact: `/tmp/lazygitk-review-run-1793910387/14-worker-result.md` only.

All findings remain unfixed, as required. Source tracing establishes the concrete failure paths, but there is no runtime reproduction in this worker run. Review coverage should not be mistaken for a test pass.

Recommended next step: triage F1 and F2 first, then authorize narrowly scoped fixes with regressions for a shared temporary-file interleaving and a valid target repository with invalid config. Resolve only the explicit storage-policy question needed for F1 before changing cross-instance semantics. F3 and F4 can follow as separate small changes; F5 is a comment/documentation correction. Preserve the existing fork-owned seams, update the merge/user guides with behavior changes, and do not open a new general design phase or a pull request.
