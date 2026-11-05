# Review 11: log-scope-filter

## Summary

Read-only review of HEAD `a9d76554252e37e8cbc8a8aa742bad9bbe3aa307` against BASE `4e1eebe278790f257a875da6633234391a5a85b3`.
Most implementation lives in fork-owned files, with small, commented hooks into the upstream loader, refresh snapshots, controllers and filtering mode. The merge guide describes these hooks accurately.
Two correctness defects were traced end to end: clearing a scope can lose an old cursor commit, and show-in-Commits reachability disagrees with the actual log during bisect.
The biggest scalability concern is deliberate unlimited descendant loading, including retaining unrelated commits until the last selected commit is reached. Whether descendants should honor the normal row limit needs clarification before changing behavior.
No repository files were changed. No Go tests, builds, generation, formatting, integration tests or commits were run. Native Git probes ran only in `/tmp/probe-lazygitk-review11-n5qhDP`.

## Findings

### F1. Descendant scopes retain an unlimited prefix of the whole repository history

- Category: code-quality
- Severity: high
- Confidence: n/a
- Location: pkg/commands/git_commands/fork_commit_scope.go:53, :108, :123, :127, :262
- Problem: The descendants path disables Git's limit even when `GetCommitsOptions.Limit` is true, accumulates every walked commit through the last selected commit in `loadCommits`, then filters the accumulated slice. `keepCommitDescendants` additionally allocates a map sized for the entire input and a boolean slice. No returned-row cap is applied either. Unlike ancestors, this path never consults `opts.Limit` after choosing the log arguments.
- Scenario / evidence: Apply `w b` to a branch at a repository's root, then `w d`. Even with the ordinary limited-list setting still true, all history reachable from every ref is parsed and retained, and every descendant is returned. On a large repository this means memory and latency proportional to the entire history on every scoped refresh, with input blocked during application. A recent selection can also retain a large unrelated prefix under topo-order. If a selected object still exists but becomes unreachable after a ref moves, `existingCommitScopeHashes` retains it and `remaining` never reaches zero; the whole walk is consumed. In the 306-commit Git probe, the old branch has one commit, while the whole graph has 306; static tracing shows a descendants scope on that root returns all 306 despite `Limit: true`. This is intentional in the comments and `TestCommitScopeLog`, so it is a scalability finding and open policy question, rather than a claim that the implementation violates an agreed descendant cap.
- Suggested fix: First resolve Q1 below. Preserve correct merge ancestry and ordering; simply adding `-300` to the input walk would produce incorrect descendant classification. Then implement the agreed bound/optimization in the fork loader, and add a targeted loader regression covering an old/root selection, a large unrelated prefix, and an existing-but-unreachable selection.

### F2. Clearing a scope loses the cursor when it is outside the initial 300 unfiltered rows

- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/helpers/mode_helper.go:217, :230, :235; pkg/gui/controllers/helpers/fork_commit_scope_filter.go:23
- Problem: `ClearFiltering` reloads with the existing limited-list setting and only tries to select the cursor hash and the pre-filter hash in that limited result. It does not load far enough to restore either hash. The Guide promises that Escape removes the scope with the cursor kept on its commit.
- Scenario / evidence: Start with 306 linear commits, an `old` branch at the root, and the default limit enabled. From Branches use `w b` on `old`: its one-commit scope shows/selects the root without disabling the limit. Press Escape. The unfiltered refresh contains only the newest 300 commits. Both selection attempts reference the root (the scope setter records it as the pre-filter hash), so both fail and the cursor cannot remain there. The native Git probe confirmed full count 306, limited count 300, old scope count 1, and no root hash in `git log HEAD --all --topo-order -300`. No TUI execution was performed; the remaining steps are directly established by the helper code.
- Suggested fix: On scope clear, ensure the unfiltered list contains the captured cursor before completing selection, using a narrowly scoped load-further continuation analogous to show-in-Commits. Preserve focus/layout and the explicit selection. Add a regression using an old branch beyond the first 300 rows.

### F3. Show in Commits can refuse a visible commit during bisect

- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/fork_show_in_commits.go:92; pkg/commands/git_commands/fork_commit_reachability.go:20, :27; pkg/gui/controllers/helpers/refresh_helper.go:1730
- Problem: `w v` checks reachability from `HEAD`, but the real Commits loader uses `refForLog`, which walks the bisect start ref or bisect new/bad ref during bisect. With whole-graph disabled, this rejects commits that the unfiltered Commits log actually shows. The check precedes the helper's existing-item fast path, so having the target already loaded does not avoid the refusal.
- Scenario / evidence: Disable whole graph, start a bisect on `master` between its tip and an old ancestor, and let Git detach HEAD at a midpoint. The log still walks `BISECT_START` (`master` in the probe), so master's tip is its first row. In Branches choose master and invoke `w v`. `HEAD..master` has commits and `wholeGraph` is false, so the action reports that the tip cannot be shown. The probe ran `git bisect start master old`: HEAD became `c151`, `HEAD..master` counted 153, BISECT_START was `master`, and `git log master --topo-order -1` returned `c304`. `refForLog` selects that start ref in this topology.
- Suggested fix: Make the reachability check use the same effective revision source as the unfiltered Commits refresh, including bisect's fallback ref. Keep that decision shared or narrowly hooked so future loader changes cannot silently diverge. Add targeted reachability coverage for bisect with whole graph disabled.

## Merge-ease notes

Numbers below are full-file BASE-to-HEAD diff counts; unrelated fork features contribute where noted.

- `pkg/commands/git_commands/commit_loader.go` (+32/-8 total): scope changes add option fields, skip synthetic rebase rows, dispatch to a separate scoped loader, and replace four log-argument decisions with `commitScopeLog`. These are cohesive, commented seams. The paused-sequencer change is separate from this review. Keeping the scope algorithm in `fork_commit_scope.go` is appropriate. New upstream revision/ordering/limit options must be reflected in the scope hook, as documented.
- `pkg/gui/modes/filtering/filtering.go` (+7/-4): one field, Active and Reset changes; minimal. The Active integration has an explicit Fork comment. Reset's field reset is evident without another comment.
- `pkg/gui/controllers/helpers/mode_helper.go` (+29/-10): label hook, clear-layout selection, layout callback extraction, removal of automatic half-screen changes. This is the largest scope-specific alteration of an upstream function, but the separate fork helper owns scope behavior and the shared callback avoids duplicating the refresh flow. Existing comments explain the upstream layout replacement. No gratuitous reformatting found.
- `pkg/gui/controllers/helpers/refresh_helper.go` (+28/-1 total): this area's changes are roughly +13/-1: switch reflog's predicate, capture three immutable scope inputs and pass them to GetCommits. The rest includes unrelated generation-guarded refresh work and command metadata. Scope handling stays within existing snapshot/load boundaries; no scope algorithm was inserted into refresh orchestration. Replacing the captured primitive fields with a scope object would couple layers without an obvious merge benefit.
- `pkg/gui/controllers/helpers/diff_helper.go` (+17/-6 total): the filtering edit only replaces the Active test with a nonempty-path test and adds a comment; minimal. External-tool additions are outside this review.
- `pkg/gui/controllers/global_controller.go` (+51/-4 total): scope of this review is a wrapper call around the returned binding list plus filtering-menu command metadata. Most additions belong to other features. `fork_flat_filter_keys.go` contains the behavior.
- `pkg/gui/controllers/filtering_menu_action.go` (+16/-6): changes are command metadata and formatting immediately needed for that metadata. Behavior remains upstream-compatible; the flat prompt code duplicates the upstream prompt declarations in a fork file. This is a small, documented duplication, but future prompt changes need checking in both places.
- `pkg/gui/controllers/filter_controller.go` (+2): command identity/verb metadata only. This is the pane text-filter prompt, separate from log filtering.
- `pkg/gui/controllers/quit_actions.go` (+15): three small hooks delegate Escape, enabled state and description to the fork helper. Comments and merge-guide entry present. The helper handles a lone Shift run first, then log filter, then explicit selection; the integration comment/merge guide abbreviate this as filter then selection.
- `pkg/gui/controllers/reflog_commits_controller.go` and `sub_commits_controller.go` (+5 each): one commented binding method each. MergeGuide explicitly warns to preserve the binding if upstream adds its own GetKeybindings.
- Branches/Remote branches/Tags integration is additive binding hooks; behavior lives in the requested fork-owned controller modules. MergeGuide documents the `w b` and `w v` hooks.

No additional extraction or broad upstream-file rewrite is recommended for this area.

## Other checks and test coverage

- Ancestors: the wanted-parent frontier correctly includes both parents of merges, de-duplicates selected ancestors and stops when complete or after 300 retained commits. `loadCommits` finalizes the previous row before the stop callback; `RunAndProcessLines` closes stdout on stop, so the ancestor stop really ends the process rather than draining the rest of its output.
- Descendants: the reverse pass correctly implements the union of descendants across merges when the input is children-before-parents. The supported scope orders supply that property; `default` is changed to `date-order`. This means the Guide's exact unfiltered-subsequence statement has an exception with `git.log.order: default` and skewed commit timestamps: the unfiltered default order can differ from scoped date-order. The code explains this exception, the Guide does not. Clarify the documentation if that configuration is supported as described.
- Path/author interaction: UI setters reset other filter kinds before applying a new one. Thus scope parent traversal does not normally operate on author/path-pruned ancestry. Reset clears all three kinds. Ref scopes re-resolve ref/upstream on each load; annotated tags are peeled.
- Rebase/sequencer rows: scoped GetCommits skips MergeRebasingCommits, so synthetic todos do not enter the scope algorithm. The real-commit filtering bindings reject malformed/non-full IDs. No new defect established here.
- Reflog/sub-commits: scoped filtering does not apply to those lists. Reflog uses HasPathOrAuthorFilter, sub-commits passes only path/author, and DiffArgs only appends a nonempty path. Graph visibility reads the filter state captured with the loaded model, so scope itself does not suppress graph rendering.
- Audited all direct `Filtering.Active()` callers found under pkg: startup/default side context; rewrite-action guard; filter-menu replacement tooltip and exit item; flat clear disabled state; status/changeFiltering; show-in-Commits clear step; and the three Commits Escape helpers. Their broad meaning is appropriate. No remaining caller incorrectly treating scope as a path/author filter was found. Path-sensitive diff/stash/commit-files callers use GetPath.
- Tests skimmed, not executed: `TestKeepCommitAncestors`, `TestCommitAncestorFilterDone`, `TestKeepCommitAncestorsLimit` test synthetic graph membership and ancestor limits; `TestKeepCommitDescendants` tests merge/union membership but no descendant limit or actual streaming loader; `TestCommitScopeLog` explicitly expects no Git limit for descendants even with Limit true. `TestCommitHasRefScope` / `TestCommitDecorationNames` cover decoration/model detection. `TestForkFlatFilterDefaultRoutes` covers key labels, not prompt execution. Reachability unit tests cover HEAD, branches, tags/custom refs, dangling/reflog-only objects and validation errors, but not bisect. Existing UI test sources cover small scope/filter/selection graphs and `w v` beyond 300 rows; that larger fixture is not used to test `w b` followed by Escape. Integration tests were not run.
- No new design phase, code implementation or iteration was started, per this request's read-only constraints.

## Open questions / remaining problems / next step

1. Q1: Is ignoring the normal limit for descendants intentional user-visible policy, or should descendant scopes respect a bounded returned list? The code explicitly chooses an unlimited walk, but Guide/Design do not state unlimited descendant loading. Resolve this specific question before a performance fix changes result size. Even if unlimited results are required, avoid retaining unrelated commit models where possible and assess worst-case refresh memory/latency.
2. Fix F2 and F3 in separate, small, verified commits with relevant guide/merge-guide updates as needed. Each has a concrete reproduction and can be addressed without reopening the feature design.
3. Add targeted loader/helper regressions for the two confirmed edges and the agreed descendant limit policy. Do not interpret existing key-label or small synthetic graph tests as coverage for those paths.
4. Optional documentation correction: qualify exact-order claims for `git.log.order: default` with skewed dates.

## Out of scope observations

No separate out-of-scope defect is reported. Reachability support, busy-task/repo-switch guards, graph state and integration-test sources were read only as necessary to validate this area's behavior. In particular, `w v` runs as foreground work and repo switching checks Busy; a cross-repository race was not established and is not reported.

## Commands run and outcomes

- Read `/tmp/lazygitk-review-run-1793910387/11-worker-request.md` and `common-review-rules.md`, repository AGENTS.md, Tracker.md, Design.md, relevant Guide/MergeGuide sections and justfile. Tracker says no work is in progress; task-specific read-only restrictions took precedence over the usual implementation/commit workflow.
- `git status --short` before and after investigation: unchanged pre-existing untracked entries `HANDOFF.md`, `reviews/`, `test/.viminfo`. No tracked edits made.
- `git rev-parse HEAD`: `a9d76554252e37e8cbc8a8aa742bad9bbe3aa307`.
- `git diff 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <scoped upstream paths>` and `git diff --numstat ...`: inspected integration changes and footprint recorded above.
- `git log --oneline BASE..HEAD -- pkg/commands/git_commands/fork_commit_scope.go pkg/gui/controllers/helpers/fork_show_in_commits.go pkg/gui/controllers/fork_show_in_commits.go`: reviewed feature history, including scope order, ref scope, show-in-Commits and detached-HEAD follow-up.
- `cat`, `sed -n`, `rg -n` and `rg --files`: read scoped fork modules, callers, tests and dependencies; audited Filtering.Active and path-sensitive callers. Some exploratory reads referenced nonexistent guessed filenames (`load_commits.go`, helper guard/graph files, basic_commits_context.go, a generic/reflog show-in-Commits UI test); those commands reported missing paths, then rg located the actual implementations/test files. These were search/read failures, not test failures.
- Native Git probe (no repository-under-review mutations): `mktemp -d /tmp/probe-lazygitk-review11-XXXXXX`; `git init -q -b master`; `git mktree`; 306 `git -c user.name=Review -c user.email=review@example.com commit-tree` calls; `git update-ref` for master and old. All succeeded. The empty-tree commits avoid working-tree changes.
- Probe `git rev-list --count HEAD --all` -> 306; `git rev-list --count --max-count=300 HEAD --all` -> 300; `git rev-list --count old` -> 1. Matching the root hash against `git log HEAD --all --topo-order -300 --format=%H` produced no match (expected), confirming F2's visibility premise.
- Probe `git bisect start master old` -> selected c151, 152 revisions left; `git rev-list --count HEAD..master` -> 153; reading `.git/BISECT_START` -> master; `git log master --topo-order -1 --format=%s` -> c304. Confirms F3's revision-source mismatch. Probe left under `/tmp` for reproduction.
- Go tests: none, as requested by default. No integration tests, full test suite, builds, vet, generation, formatting, Git writes in the reviewed repository, or external network calls were performed. Findings are supported by code-path tracing and the explicitly permitted native Git probe, not a claim of TUI/test execution.

## Files changed / artifacts

- Repository: none.
- Result artifact: `/tmp/lazygitk-review-run-1793910387/11-worker-result.md` (this file).
- Throwaway native Git repository: `/tmp/probe-lazygitk-review11-n5qhDP`.
