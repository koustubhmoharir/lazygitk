# Review 09: history-editing

## Summary
Read-only review completed against BASE `4e1eebe278790f257a875da6633234391a5a85b3`.
The captured combine and selected-commit rebase implementations have good separation and substantial real-Git unit-test coverage on inspection.
The largest confirmed risk is ignored-file data loss in the older interactive history-editing paths, which do not use the captured ignored-file approval machinery.
Older patch/reword paths also retain unchecked indexes or mutable HEAD/selection across prompts.
The upstream-file footprint is moderate: rebase.go +64/-30, patch.go +5/-16; controller totals include unrelated binding metadata and other fork features.
No repository files were changed; no builds, unit tests, or integration tests were run. Two native-Git probes were performed under /tmp.

## Findings

### F1. Generic history-editing rebases can overwrite ignored files without approval
- Category: bug
- Severity: high
- Confidence: CLEAR BUG
- Location: pkg/commands/git_commands/rebase.go:217; pkg/commands/git_commands/rebase_start_policy.go:30; pkg/commands/git_commands/rebase.go:475
- Problem: The generic preparation path only checks tracked status before starting Git. It never computes or reviews ignored-file collisions. Reword, move, amend-to and custom-patch editing reach this path rather than `PrepareSelectedCommitRebase` or `PrepareCombinePlan`. A clean tracked status does not protect ignored files during Git's initial rewind. This contradicts Design.md's ignored-file protection rule and Guide.md's promise of approval before a rebase can overwrite ignored files.
- Scenario / evidence: Confirmed with native Git in `/tmp/probe-history-review-spwaym`. Create base commit tracking `collision` with `base contents`, middle commit changing it to `historical contents`, then HEAD deleting it and adding `/collision` to .gitignore. Recreate `collision` with `valuable ignored contents`. The exact clean-start status command returns empty output. Run `GIT_SEQUENCE_EDITOR="sed -i 's/^pick /edit /'" GIT_EDITOR=true git rebase --interactive --no-autostash --keep-empty --no-autosquash --rebase-merges HEAD~2`. Git stops successfully at middle and `cat collision` prints `historical contents`: the ignored bytes have already been overwritten. This reproduces the rewind used to reword middle; the sequence-editor change is wholly inside the temporary probe repository. No approval callback exists along the production `RewordCommit -> BeginInteractiveRebaseForCommitRange -> runInteractiveRebase` path.
- Suggested fix: Apply the existing captured collision/fingerprint/approval policy to generic history-editing starts before any mutation, including compound amend/fixup flows before creating their first commit. Add the demonstrated historical-delete/ignored-recreation regression case. This is a gap in fork safety coverage of an inherited operation, not a claim that the fork introduced Git's overwrite behavior.

### F2. Custom patch editing can panic when its source is absent from the current commit list
- Category: bug
- Severity: high
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/custom_patch_options_menu_action.go:135; pkg/gui/controllers/custom_patch_options_menu_action.go:165; pkg/commands/git_commands/rebase.go:455; pkg/commands/git_commands/rebase.go:469
- Problem: `getPatchCommitIndex` returns -1 when the patch source is absent, but all source-edit handlers pass that value to backend editing methods. The range check only checks the upper bound; an end/start of -1 reaches `commits[-1]`. `CanRebase` was captured when building the patch and does not establish that its source remains in the displayed history.
- Scenario / evidence: Build a nonempty patch from a rebasable local commit, then filter Commits so the source disappears, or externally rewrite the source away and refresh. The patch remains active: repository-wide searches found no refresh-time reset/revalidation of PatchBuilder.CanRebase. `Call` only checks Active, IsEmpty, CanRebase and operation state before offering Move patch out into index. That action obtains -1, then `MovePatchIntoIndex -> BeginInteractiveRebaseForCommit(-1) -> BeginInteractiveRebaseForCommitRange(-1,-1)` reaches the negative slice access before the clean-tree check. Other source-edit actions share the problem. This is inherited logic still exposed by the fork's changed patch flow; confirmed by end-to-end code tracing, not by a UI run.
- Suggested fix: Refuse source editing if the captured source cannot be resolved, and validate both lower and upper index bounds in backend entry points. Resolve against actual captured history rather than assuming the filtered display includes the source.

### F3. Reword can amend a different commit after the message editor opens
- Category: bug
- Severity: high
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/local_commits_controller.go:914; pkg/gui/controllers/local_commits_controller.go:935; pkg/gui/controllers/local_commits_controller.go:970; pkg/gui/controllers/local_commits_controller.go:943
- Problem: `reword(commit)` loads the selected commit's message, but its confirmation callback captures neither that commit nor HEAD/ref identity. Submission reads the current model/selection and the HEAD case invokes an unconditional amend of current native HEAD. The switch-to-editor callback has the same mutable-target behavior. This violates the fork's captured-selection and stale-repository rules.
- Scenario / evidence: Open reword on HEAD A, create another commit B in another terminal while editing, and submit before a model refresh. The model still identifies the selected row as HEAD, so `RewordLastCommit` amends B with A's edited message. Even if a refresh has happened, there is no comparison with the HEAD/ref that was present when the action started. The editor panel does not pause background refreshes; waiting-status suppression begins only on submission. No tests were run; this is a direct trace of the callback and native amend path. The mutable-target logic predates the fork but remains in its edited reword implementation.
- Suggested fix: Capture target OID and source HEAD/ref on opening the editor, retain them for both submit paths, and refuse/review when the native repository snapshot differs before amendment.

### F4. Failed older-commit reword does not refresh the stopped repository state
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/local_commits_controller.go:991; pkg/gui/controllers/fork_reword_failure.go:15; pkg/gui/controllers/fork_continue_operation.go:29
- Problem: The new error explanation checks native rebase state but returns before the only `RefreshFromWorker` call. The waiting-status helper resumes input and removes suppression without requesting a repository refresh. The model can therefore still show the pre-rebase history and operation state while Git has already moved HEAD and stopped. The new global Continue binding uses `WorkingTreeStateAtLastCommitRefresh` for availability, so it can remain inapplicable immediately after the error despite the explanation saying to continue/abort.
- Scenario / evidence: Reword an older commit with a rejecting pre-commit hook. Starting the rebase succeeds and stops at the commit; `RewordLastCommit` fails. `handleReword` returns `rewordFailure(err)` at line 994; line 998 is never reached. `AppStatusHelper.WithWaitingStatusImpl` only pauses/resumes refreshes, and `endBlockingInput` only changes suppression/input. An eventual ref poll or F5 may repair the display, but there is no guaranteed post-failure refresh. Inspected TestRewordFailureMessage only verifies text generation and does not exercise the controller failure path.
- Suggested fix: Request the appropriate worker refresh on both success and failure before releasing input, while preserving the error and draft. Add a controller test asserting a failure after the rebase starts refreshes the operation state.

### F5. Merge guide's universal rebase-wrapper claim is inaccurate
- Category: docs
- Severity: low
- Confidence: n/a
- Location: docs-master/fork/MergeGuide.md:120
- Problem: “Every start site goes through runInteractiveRebase” is not the current integration invariant. Reword-in-editor and captured branch rebases use `PrepareInteractiveRebaseCommand` directly, autosquash constructs its own command, and combine and selected-commit rebases use independent command builders.
- Scenario / evidence: Repository-wide search for `NewGitCmd("rebase")`, `PrepareInteractiveRebaseCommand`, and direct textual interactive starts found rebase.go:65, rebase.go:425, branch_action_rebase.go:100, combine_command.go:44 and selected_commit_rebase.go:126. The relevant starts do have clean/no-autostash/retry policy enforcement by inspection; this is not evidence of an autostash bypass. `GitRebaseEditTodo` is editing an existing operation and appropriately does not require a clean start.
- Suggested fix: Document preparation as the shared generic entry point and list the deliberate independent builders and their required policy checks. Preserve the separate existing-operation exemption.

## Merge-ease notes

- `pkg/commands/git_commands/rebase.go` (+64/-30): Mostly necessary adaptation from returning a command to returning command+error, with a small execution wrapper. Shared clean checks live in additive files. The opts field, config setting and retry-disable comments explain their purpose. Moving the wrapper and preflight implementation into the fork-owned policy file would marginally reduce upstream surface; wholesale extraction of this file is not justified. The autosquash command is an explicit independent start and must remain in the merge checklist.
- `pkg/commands/git_commands/patch.go` (+5/-16): The rebase-start hook and removal of stash behavior are small and understandable, with replacement comments. Preserve the no-stash signature and its callers. The untouched stash field/constructor parameter remains unused within this file; removing it would require extra integration changes and is not a priority.
- `pkg/commands/git_commands/commit.go` (+0/-15): Changes are draft-history removal, outside this request's non-draft review; no non-draft delta found.
- `pkg/gui/controllers/basic_commits_controller.go` (+64/-46): Most new behavior enters through additive binding builders; removal of the old selected-commit rebase implementation is appropriate. Other changes include copy/placement/difftool and metadata beyond this area's scope.
- `pkg/gui/controllers/local_commits_controller.go` (+208/-145 total): The total includes metadata and other features. Relevant behavioral hooks are fixup/amend preflight delegation and reword error/draft reporting. `fixup_placement.go` successfully extracts the former stacked-placement implementation; the integration comments explain preflight timing. Keep fixes to F3/F4 similarly localized.
- `pkg/gui/controllers/custom_patch_options_menu_action.go` (+69/-44): No-stash removal is localized and explained, but repeated draft-reporting blocks add upstream conflict surface. The obsolete comments at lines 181 and 220 still describe Up recalling the original commit even though CommitIndex is inert; update those when next touching the file. No need for a large refactor during this review.
- `pkg/app/daemon/daemon.go` (+2/-0): Minimal enum and dispatch registration. Existing daemon parsing assumes valid kind/instruction and argv; combine uses that existing protocol, not a newly isolated robust parser. The new payload stays in a private file, avoiding ARG_MAX; todo and persistent-message handling are fork-owned.
- `pkg/commands/models/working_tree_state.go` (+3/-0): Minimal additive state field and explanatory comment for the paused sequencer; operation dispatch still uses Effective().
- New selected_commit.go, selected_commit_rebase_actions.go, fork_combine.go, fixup_placement.go, fork_reword_failure.go and fork_continue_operation.go keep new control flow out of the upstream controllers. Backend capture/topology/ignored-risk files and daemon/combine_rebase.go are likewise additive.

## Out of scope observations

No additional cross-area bug investigation undertaken. Adjacent refresh/commit-panel/merge-helper code was read only to trace the scoped callbacks and operation dispatch.

## Commands run and outcomes

- Read the complete worker request, common review rules, AGENTS.md, Tracker.md and Design.md; inspected relevant Guide.md and MergeGuide.md sections. Tracker says no work is in progress. No new design phase was started.
- `git status --short` at start and end: identical pre-existing untracked `HANDOFF.md`, `reviews/`, `test/.viminfo`; left untouched.
- `git diff 4e1eebe278790f257a875da6633234391a5a85b3 HEAD --stat/--numstat -- <scoped upstream files>` and scoped diffs: measured footprint and reviewed changed integration points. `git log --oneline BASE..HEAD --` the rebase/patch/fixup/combine paths identified the captured-rebase, policy and combine commits.
- Read scoped backend/controller/daemon implementations with cat/sed/nl; used rg over pkg for native rebase starts, preparation callers, patch resets, CanRebase, refresh/continue and tests. Some exploratory reads referenced nonexistent `interactive_rebase.go`, `main_controller.go`, or a missing continue-test glob; those produced path errors and were replaced by actual source paths/searches. One initial combined docs output was truncated; relevant sections were subsequently read explicitly.
- Skimmed tests for combine plan topology/merge rejection, capture validation, command environment and update-refs, persistent daemon messages, conflicts and hook recovery; selected-commit rebase topology/continuation/ignored collision tests; generic clean-start policy; amend/fixup preflights; combine controller replay/draft handling; and reword failure wording. These are coverage observations, not passing-test claims. In particular, the successful fixup-placement test deliberately returns a sentinel before the rebase and does not validate actual placement.
- Native probe `/tmp/probe-history-review-RpDA5K`: initial root-rebase variant refused the collision and retained the valuable bytes. This negative case narrowed the suspected overwrite to a rewind onto an existing historical tree.
- Native probe `/tmp/probe-history-review-spwaym`: nonroot rewind with the generic production flags succeeded and overwrote ignored bytes, confirming F1. Probe repositories are disposable and were left for inspection; no git write commands were run in the reviewed repository.
- No Go tests run, per the request's default. No integration tests, generation, formatting, builds, commits or repository mutations performed.

## Work performed and files changed

Investigation and review only. The only handoff artifact written is `/tmp/lazygitk-review-run-1793910387/09-worker-result.md`; native probe files were created only under the expressly permitted `/tmp/probe-*` directories. No implementation or fixes were authorized by the worker request.

## Remaining problems and recommendations for the next step

Prioritize F1 with a narrowly scoped ignored-file approval regression and implementation using the established captured-risk policy. Then address missing patch-source/index validation (F2), captured reword targets (F3), and failure refresh (F4). These are unimplemented findings, not completed fixes. Update the merge guide's start-site inventory (F5) alongside the safety integration changes. Add targeted unit/controller tests for the concrete failure paths before broad verification; do not treat the tests skimmed here as executed.

One documentation/design question remains for the owner rather than this read-only worker: Guide.md says rebases do not update other branch refs, but generic history-editing starts deliberately inherit rebase.updateRefs, and stacked fixup placement explicitly requires it. Should that user-guide sentence be limited to captured branch/selected-commit/combine rebases? The request itself describes update-refs disabling as optional, so this review does not assume a new global policy or propose silently disabling stacked placement.
