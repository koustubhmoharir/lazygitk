# Review 10: cherry-pick-select

## Summary
Reviewed commit multi-selection, captured cherry-pick ordering/revalidation, completion callbacks, paused sequencer detection, commit/ref copying, and dormant upstream cherry-pick state against BASE `4e1eebe278790f257a875da6633234391a5a85b3`.
The implementation mostly uses additive fork modules and small documented integration hooks. Captured OIDs, repository generation checks, native Git execution, explicit merge-parent choice and ignored-file review are sound approaches.
The largest confirmed problem is that rendering a filtered list permanently deletes hidden selections, contrary to the guide and the rule against silently narrowing a selection.
An empty pick automatically skipped to completion can also leave its selection marked on older Git versions. Dormant upstream paths are documented well in MergeGuide.md but insufficiently marked where their dangerous legacy implementation remains.
No repository files were changed. One targeted unit test was run; no integration tests, builds, generation, formatting, commits or implementation were performed, as required by this read-only request.

## Findings

### F1. Filtering permanently narrows the explicit selection
- Category: bug
- Severity: high
- Confidence: CLEAR BUG
- Location: pkg/gui/context/fork_commit_multi_select.go:158, :170, :249; docs-master/fork/Guide.md:362
- Problem: Every focus/render calls `pruneCommitSet`, which deletes every selected hash absent from the currently loaded list. Absence from a filtered or limited log does not mean the commit vanished. The comment saying the set comes back when commits return is false: deleted map entries have no restoration path. If the last selected commit disappears, `HasCommitSet` becomes false and commands resume acting on the cursor.
- Scenario / evidence: Select A and B with Space, apply a path or author filter showing A but hiding B, then clear the filter. B is permanently deselected; `c p` acts only on A. If both are hidden, selection mode ends and a command can act on the new cursor commit instead. A refresh that pushes a selected older commit beyond the loaded limit has the same issue. `TestMultiSelectSetIsPrunedToTheLoadedList` explicitly verifies deletion after replacing the list; its targeted run passed. The existing UI filter scenario uses an ancestors filter that includes both selected commits, so it does not exercise hiding selected commits.
- Suggested fix: Keep selection identity separate from loaded/visible rows; do not delete hashes merely because a filtered or limited list omits them. Ensure set-consuming commands cannot silently receive only the visible subset. Add regression coverage for filter-hide/clear and limited-list refresh. Resolve the rewritten-hash question below before choosing how genuinely replaced commits are handled.

### F2. Initial empty-pick auto-skip bypasses selection cleanup
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/fork_cherry_pick.go:198-211; pkg/gui/controllers/helpers/merge_and_rebase_helper.go:193, :295; pkg/commands/git_commands/captured_cherry_pick.go (version-dependent `--empty=keep`)
- Problem: `apply` decides whether to clear the selection using the original `cmd.Run()` error, rather than the operation outcome after `CheckMergeOrRebaseWithRefreshOptions`. That helper can successfully skip an empty pick and finish the operation. In that case `handled == nil`, `result != nil`, and `WorkingTreeState().Any() == false`; neither cleanup branch runs. The callback is only registered after the helper returns, so the recursive skip cannot use this flow's callback either.
- Scenario / evidence: On Git below 2.45, select a nonempty commit whose patch is already present at the destination. The command includes `--allow-empty` but lacks `--empty=keep`, so Git reports "The previous cherry-pick is now empty". The helper calls `genericMergeCommandImpl(skip, false, true)`. A successful final skip ends the operation, yet the explicit set/range remains marked. The same applies if that skip successfully applies the remaining picks. This is an end-to-end static path finding, not an executed old-Git reproduction.
- Suggested fix: Carry an explicit final outcome through conflict/auto-skip handling, or establish completion tracking before the recursive skip can finish. Clear only after confirmed successful completion and preserve the existing changed-selection and abort protections. Add a targeted test simulating a pre-2.45 empty pick and successful skip-to-end.

### F3. Dormant legacy paste code lacks a local warning
- Category: merge-ease
- Severity: low
- Confidence: n/a
- Location: pkg/gui/controllers/helpers/cherry_pick_helper.go:35; pkg/gui/modes/cherrypicking/cherry_picking.go; docs-master/fork/MergeGuide.md:467
- Problem: The retained `CherryPickHelper.Paste` still describes itself as the live paste entry point and contains implicit stashing. There is no `// Fork:` note at that method explaining that it is intentionally unreachable and superseded by the captured selection flow. The unchanged mode package likewise gives no indication that its setters are intentionally disconnected. A future upstream caller could restore behavior that violates fork rules without touching the well-commented replacement at `LocalCommitsController.paste`.
- Scenario / evidence: Diff confirms the mode package is unchanged. Searches found no production call that populates its state: only the mode's own update implementation and the helper reset assign `CherryPickedCommits`. MergeGuide.md accurately explains the dormancy and retention trade-off; the local legacy code does not.
- Suggested fix: Add a short local fork comment at legacy `Paste` identifying its dormant status, replacement, and prohibition on reconnecting implicit stash behavior. Keep the unchanged mode implementation if avoiding upstream conflict surface remains the priority; there is no need to delete it wholesale.

## Merge-ease notes

Full-file numstat below includes unrelated features in shared files, not just this review area's lines. Across these 11 changed upstream files it is 604 additions / 638 deletions. The mode directory has no diff. Most behavior in this area is additive.

| Upstream file | Full-file + / - | Assessment for this area |
| --- | --- | --- |
| pkg/commands/git_commands/status.go | 33 / 2 | HEAD-file fallbacks and SequencePaused integration are explained. `isSequencePaused` could live in fork_paused_sequence.go to leave smaller hooks. Preserve the existing rebase stopped-sha exclusion. |
| pkg/commands/git_commands/commit_loader.go | 32 / 8 | Sequencer change is one small, commented branch delegating to the fork helper; good seam. Other changes are scope-filter work. |
| pkg/gui/context/local_commits_context.go | 11 / 0 | Minimal state field and two cursor-diff overrides; documented. |
| pkg/gui/controllers/basic_commits_controller.go | 64 / 46 | Guard wraps the complete assembled binding list; additional copy bindings live in a fork file. The nested return expression makes the integration harder to scan but does keep the guard centralized. |
| pkg/gui/controllers/local_commits_controller.go | 208 / 145 | Relevant changes are small: remove upstream marks, delegate paste, add selection toggle/guard, and bypass commit dragging with a set. Replacement paste has a useful Fork comment. Much of the file diff belongs to other features. |
| pkg/gui/controllers/bisect_controller.go | 28 / 9 | One selection guard plus metadata. MergeGuide's later “Binding metadata only” list still includes this file despite its substantive guard; the earlier Controllers section correctly explains the guard. Remove it from the metadata-only list in follow-up. |
| pkg/gui/controllers/quit_actions.go | 15 / 0 | Three paired, commented Escape hooks; appropriate and necessary. |
| pkg/gui/controllers/helpers/cherry_pick_helper.go | 0 / 53 | Removes marking APIs while retaining dormant paste/reset. Retention reduces churn, but see F3. |
| pkg/gui/controllers/helpers/mode_helper.go | 29 / 10 | Cherry-pick banner is intentionally unchanged/dormant; changes shown by diff are filtering behavior outside this area. |
| pkg/gui/controllers/helpers/merge_and_rebase_helper.go | 77 / 265 | Callback state/setter/settlement are small and documented in MergeGuide; setter and settlement could be fork-owned methods in an additive file, leaving only field/call hooks upstream. Other large replacements are out of scope. |
| pkg/gui/global_handlers.go | 7 / 0 | Small, commented guard for the globally owned abbreviated copy action. |

Additive files reviewed: captured_cherry_pick.go, fork_paused_sequence.go, fork_ref_commit.go, fork_refs_at_commit.go, fork_commit_reachability.go, controllers/fork_cherry_pick.go, fork_commit_multi_select.go, fork_commit_copy.go, fork_ref_commit_copy_controller.go, context/fork_commit_multi_select.go and gocui/fork_selected_lines.go. The integration cherry_pick.go component was read only.

Ordering is deliberately ancestry-first: `rev-list --reverse --topo-order`, filtered back to the selected OIDs. Lexically sorting input OIDs stabilizes tie-breaking for unrelated tips during revalidation. It does not execute Space-toggle order or simply reverse display order. The confirmation displays the resulting exact execution order. No ordering correctness bug was identified.

The explicit-set guard covers the Basic/Local bindings inspected, with bisect and the global copy action covered separately. Autosquash's above-selected submenu uses `singleItemSelected`, which observes the overridden `AreMultipleItemsSelected`, so its absence from the outer guard map does not permit a cursor mutation. No confirmed command bypass was found. That submenu's reason is the generic range-not-supported message rather than the documented clear-selection reason.

Paused detection uses the worktree Git directory and the first pick/revert todo command; it does not treat an arbitrary sequencer directory as active. Committed-first-entry suppression compares abort-safety with HEAD and leaves the entry present on read/resolve failure. No normal-state false positive was established. Short-form/malformed/stale externally authored todo files remain less strictly validated than native Git; no bug is claimed without a realistic native state.

## Remaining questions and recommended next step

- When external rebase/amend replaces selected hashes, should selection retain the original immutable commits for retry, be explicitly invalidated with an explanation, or be cleared? Current docs promise persistence but do not specify a mapping to rewritten commits. Do not infer a remapping policy from row positions. This review makes no design decision and implements nothing.
- Should a completion callback survive an operation being finished/aborted externally or by a manual final Enter? Currently it is removed only through the helper's continue/skip/abort path. A stale callback can remain until a later operation, with exact OID equality protecting only some cases. Treat this as a lifecycle question/risk requiring a focused test, not a confirmed additional bug.
- Prioritize F1 and the already specified filter-persistence regression, then F2's final-outcome cleanup test. Add callback tests for continue-to-end, skip-to-end, abort, repository change and changed selection. The reviewed controller unit tests exercise changed selection and stale destination, but do not directly exercise the new completion callback lifecycle.
- Reconcile the misleading prune comment and the bisect metadata-only documentation as narrowly scoped follow-up. No new design phase is needed for the established filter-persistence or successful-completion requirements.

## Out of scope observations

None requiring separate investigation. Shared-file changes for captured merge/rebase, commit drafts, and scope-filter implementation were not reviewed beyond the calls needed to establish this area's behavior.

## Commands run and outcomes

- Read the worker request, common-review-rules.md, AGENTS.md, Tracker.md, Design.md and relevant Guide.md/MergeGuide.md sections with cat/sed.
- `git status --short` before and after review: identical existing untracked entries `HANDOFF.md`, `reviews/`, `test/.viminfo`; left untouched.
- `git diff 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <scoped upstream paths>`: inspected relevant hooks and removals.
- `git diff --numstat 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <11 shared upstream files and mode directory>`: footprint recorded above.
- `git diff --exit-code 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- pkg/gui/modes/cherrypicking/`: exit 0, unchanged.
- `git log --oneline 4e1eebe278790f257a875da6633234391a5a85b3..HEAD -- <scoped fork files>`: inspected feature/fixup history, including selection, paused-status and captured-pick commits.
- Read source and existing targeted unit/integration test definitions with cat/sed/rg, including selection pruning, cherry-pick controller tests, paused-sequence tests, reachability tests and selection/conflict scenarios. A few exploratory rg/sed paths did not exist; actual paths were located and read. Integration tests were not executed.
- `go test ./pkg/gui/context/ -run '^TestMultiSelectSetIsPrunedToTheLoadedList$' -short -count=1`: PASS, package reported 0.006s. This confirms the suspected destructive prune behavior is explicitly encoded in the current test; it does not establish correctness of filter persistence.
- No other tests, builds, probes or vet runs. No repository file writes. Only this handoff artifact was created.
