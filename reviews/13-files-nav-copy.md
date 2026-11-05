# Review 13: files-nav-copy

## Summary

Read-only review of HEAD `a9d76554252e37e8cbc8a8aa742bad9bbe3aa307` against BASE `4e1eebe278790f257a875da6633234391a5a85b3`.
The navigation and inspection additions are generally small, commented integrations with shared fork helpers. No newly introduced correctness failure was established.
One definite inherited bug remains exposed by the flat copy actions: selected filenames are interpreted as Git pathspecs when copying diffs.
The largest upstream footprints are Files (+197/-81) and Commit files (+82/-21), including metadata and difftool changes outside this review's substantive scope.
Copy dispatch is shared successfully, but the copy providers still occupy upstream files and their unit tests do not exercise actual clipboard output.
No repository file was changed; no tests, builds, generation, formatting, commits, or integration scenarios were run.

## Findings

### F1. Copying one file's diff can include other files matched by its name

- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/files_controller.go:1642; pkg/gui/controllers/commits_files_controller.go:254; supporting command builders at pkg/commands/git_commands/diff.go:199 and pkg/commands/git_commands/working_tree.go:467.
- Problem: Both copy-diff handlers pass filenames after `--` to `git diff` without literal pathspec protection. `--` prevents option interpretation; it does not disable Git's wildcard or magic pathspec interpretation. This is an inherited upstream defect, not a regression introduced by the flat bindings.
- Scenario / evidence: Track files named `a*.txt` and `a2.txt`, modify both, select only `a*.txt`, then use `y d` in Files. The command ends in `-- a*.txt`, so the copied patch includes the unrelated `a2.txt` change too. The corresponding commit-file action passes the same unprotected names through `pathsForDiff` and `ShowFileDiffCmdObj`. A name beginning with `:(exclude)` can instead alter the selection using pathspec magic. End-to-end source tracing included `GitCommandBuilder.ToArgv`, `gitCmdObjBuilder.New`, and the copy handlers: no layer adds literal pathspec protection. `git show BASE:...` confirmed the Files call already existed in BASE. Spaces, quotes, shell substitutions, and newlines alone are not shell injection here: paths are argv elements, and name/path copies pass strings directly to the clipboard.
- Suggested fix: Give these copy-diff launches a command-scoped literal-pathspec mode (as the fork's historical difftool launches already do). Protect both rename endpoints and directory paths. Add a regression where a literal wildcard filename coexists with another matching changed file; assert the clipboard patch includes only the selected target. Do not globally change pathspec interpretation for commands that intentionally accept patterns.

### F2. Extracted copy providers continue to enlarge edits in upstream controllers

- Category: merge-ease
- Severity: low
- Confidence: n/a
- Location: pkg/gui/controllers/files_controller.go:1584; pkg/gui/controllers/commits_files_controller.go:276; pkg/gui/controllers/fork_files_copy.go:8; pkg/gui/controllers/fork_commit_files_copy.go:8.
- Problem: Fork-owned files contain the small flat binding declarations, but the extracted menu providers and their new command metadata remain in the two large upstream controllers. The Files copy region changes from one upstream menu method to a provider plus wrapper, with five sets of new metadata; Commit files does the same with six. Future upstream copy-menu edits must be reconciled inside these already heavily edited controllers.
- Scenario / evidence: `git diff BASE HEAD --` shows the provider extraction and every item annotation in those upstream files, while `fork_files_copy.go` and `fork_commit_files_copy.go` each contain only 20 lines. The full controller footprints are +197/-81 and +82/-21 respectively; these totals include other features and should not be attributed entirely to copying. Sharing `flatMenuItemBindings` avoids duplicating dispatch and disabled-reason logic, which is good.
- Suggested fix: In a separate mechanical cleanup, consider housing the extracted providers beside the flat bindings, leaving documented wrapper/call seams in upstream controllers. Retain meaningful Files-versus-history differences. Update MergeGuide.md with the exact provider ownership/replacement seam in the same change. This is optional merge maintenance, not a correctness blocker.

### F3. Unit coverage stops before real file-copy behavior

- Category: code-quality
- Severity: low
- Confidence: n/a
- Location: pkg/gui/controllers/fork_menu_item_bindings_test.go:136; pkg/gui/filetree/directional_navigation_test.go:52; pkg/gui/controllers/tree_navigation.go:20.
- Problem: Copy unit tests verify generic dispatch and ID membership, but do not run real Files/Commit-files copy handlers with a selected file or range. Directional-navigation tests exercise the tree function directly, not the controller callback that resets the selection after navigation.
- Scenario / evidence: `TestFlatCopyBindingIDsExistInMenuItems` constructs empty controllers, builds items, and checks IDs/descriptions; it does not call the actual bindings' disabled reasons or handlers. The enabled/disabled helper tests use synthetic menu items. Consequently these tests cannot detect F1, broken clipboard text, or a real provider accidentally accepting a range. The tree tests cover compressed parents, filtered first children, roots, empty trees, and flat-mode no-ops well, but never establish a range or invoke `treeNavigationBindings`. No claim is made here that integration coverage is absent; integration scenarios were not executed.
- Suggested fix: Add focused controller-level coverage for actual clipboard values and actual range-disabled reasons, plus a callback-level navigation test verifying that a successful collapse/parent/child action clears the range while a no-op leaves it unchanged. Preserve existing behavior rather than making an unapproved selection-policy change.

## Merge-ease notes

Raw added/deleted lines below are against BASE; totals include metadata and other features in the same files.

| Upstream file | Footprint | Assessment |
| --- | --- | --- |
| `pkg/gui/controllers/files_controller.go` | +197/-81 | Additive copy/diff/tree hooks are commented. Fetch binding and implementation are removed and documented. Copy provider ownership is the main optional extraction opportunity (F2). Conflict key-label changes are localized. |
| `pkg/gui/controllers/commits_files_controller.go` | +82/-21 | Tree and flat-copy hooks are small and commented; copy extraction is documented. Historical/difftool additions were excluded from substantive review as requested. F2 applies to the provider. |
| `pkg/gui/controllers/stash_controller.go` | +14/-1 | Hash instead of moving stash index is a one-line behavioral replacement with a Fork comment and matching MergeGuide entry; the rest is metadata. |
| `pkg/gui/controllers/switch_to_focused_main_view_controller.go` | +28/-0 | Optional inspect callback and a separate focus-main command preserve mouse focus-only behavior. Narrow, commented integration. |
| `pkg/gui/controllers/switch_to_sub_commits_controller.go` | +14/-0 | `WithoutDoubleClick` is a small, documented capability for Worktrees; default behavior remains available to other panes. |
| `pkg/gui/controllers/switch_to_diff_files_controller.go` | +2/-0 | CommandID/Verb metadata only against BASE; MergeGuide classifies this correctly. |
| `pkg/gui/controllers/jump_to_side_window_controller.go` | +45/-9 | Distinct pane identities and retained commands without shortcuts are documented. The title switch could move to a fork helper, but its present placement is a minor concern. |
| `pkg/gui/controllers/worktrees_controller.go` | +10/-0 | Metadata only against BASE; actual new inspection attachment is in controllers.go. MergeGuide's metadata-only classification is correct. |
| `pkg/gui/controllers.go` | +43/-8 | Scoped attachments are small and commented: Files inspect, Worktrees key-only history, ref copy. External-tool wiring/cache changes also contribute to this total and were not separately reviewed. |
| `pkg/gui/global_handlers.go` | +7/-0 | LocalCommits explicit-set copy guard is minimal and documented; preexisting range guard remains. |

New tree navigation and flat-copy dispatcher modules are additive and shared through generics/callbacks. Existing duplicate name/path/absolute-path item construction in the two copy providers largely predates the fork; do not introduce a broad abstraction solely to remove this modest repetition. History content, diff target resolution, absolute paths in another worktree, and working-tree staged/unstaged choices have real differences.

The integration comments and MergeGuide entries inspected match these scoped changes, including the distinct numbered jump identities. Some inspection comments still describe Enter as the conflict route and discuss explicit key overrides even though fork defaults bind Enter to commit and user key settings are ignored; this is maintenance wording, not evidence that default `v` dispatch is wrong.

## Behavior traced and test assessment

- Tree navigation checks tree mode and visible selection before mutating collapse state; handles hidden synthetic root index -1; uses the first child of the already filtered tree; walks past compressed ancestors. Collapsing retains the directory index. On success, the controller calls `SetSelection`, cancelling the range before refreshing. Flat mode and leaf/right no-ops do not refresh or clear a range. No defect established in these paths.
- File-specific name/path/absolute-path/diff/content actions reject multiple selected rows. They do not silently copy only the cursor. The explicit all-files diff action intentionally operates on the whole tree. The global relative-path binding has its own range guard. This report does not propose adding multi-file clipboard formatting or changing those choices.
- Files `v` dispatches dialog-only conflicts, submodules, inline conflicts, then normal diff focus. `0` has a separate focus-only command. Mouse focus uses the focus-only route. Default path projection was traced to confirm the intended `v`/`0` separation.
- Worktrees receives `SwitchToSubCommitsController.WithoutDoubleClick`; its own double-click switch remains. Projection chooses its history inspector for `v` and Space. Existing unit assertions verify these key labels; selected-ref construction covers branch versus detached/rebasing HEAD.
- Pane jumps preserve per-window IDs and positional keys; their commands still exist if there are fewer jump keys. Existing command-identity tests inspect target IDs and individual keys.
- New branch from stash passes the immutable entry hash into `Refs.NewBranch` rather than its shifting index. Files no longer defines fetch; the global fetch hook is documented.
- `ExpectClipboard` in `pkg/integration/components/test_driver.go:217` reads `gui.ClipboardText()` through the standard matcher. It was inspected only, not changed or executed.

## Out of scope observations

None requiring a separate finding. Shared diff builders were read only to establish the copy command's behavior; this was not a general review of rendering, historical difftools, or destructive file operations.

## Commands run and outcomes

All investigation commands were read-only. The only write was this result artifact.

- `cat /tmp/lazygitk-review-run-1793910387/13-worker-request.md` and `cat /tmp/lazygitk-review-run-1793910387/common-review-rules.md`: read the scope and constraints completely.
- `cat AGENTS.md`, `cat docs-master/fork/Tracker.md`, `cat docs-master/fork/Design.md`, and `cat`/`sed`/`rg` over relevant Guide.md/MergeGuide.md sections: read workflow, selection/key rules, behavior and merge seams. Tracker says no current work; no design phase was started.
- `git rev-parse HEAD`: returned `a9d76554252e37e8cbc8a8aa742bad9bbe3aa307`.
- `git diff --stat BASE HEAD -- <scoped paths>`, `git diff --numstat BASE HEAD -- <upstream scoped paths>`, and `git diff BASE HEAD -- <scoped controller paths>`: inspected implementation changes and quantified upstream footprint. Every invocation substituted the full BASE hash, not a ref named BASE.
- `git log --oneline BASE..HEAD -- <scoped paths>`: inspected feature provenance, especially navigation, flat copying, inspection, stash capture and global fetch.
- `git show BASE:pkg/gui/controllers/files_controller.go` and `git show BASE:pkg/gui/controllers/commits_files_controller.go`, with selected `sed` excerpts: confirmed inherited copy implementation.
- `rg --files`, `rg -n`, `cat`, and `sed -n` over scoped implementations/tests plus immediate dependencies: traced tree construction/filtering/index lookup, cursor range cancellation, copy eligibility, Git argv construction, clipboard handlers, Worktrees ref selection, key projection and controller attachments. Some initial guessed filenames were absent (e.g. `filetree/tree.go`, `context/files_context.go`, `git_commands/git_cmd.go`); actual implementations were located with `rg` and read. No investigation remained blocked by missing files.
- `cat justfile`: checked available verification recipes; the worker's narrower read-only/test restrictions took precedence.
- `git status --short` at beginning and end: identical existing untracked entries `HANDOFF.md`, `reviews/`, and `test/.viminfo`; untouched.
- Tests/builds/vet/integration: **not run**. Per request, tests are off by default. F1 is established by source tracing; the existing fake-runner test asserts ordinary argv, not literal special-character behavior, so running it would not independently reproduce F1. No test success is claimed.

## Files changed, remaining problems, and next step

- Repository files changed: **none**. Implementation and fix iteration were explicitly prohibited by the worker request.
- Artifact written: `/tmp/lazygitk-review-run-1793910387/13-worker-result.md`.
- Remaining problems: F1 remains unfixed; F2/F3 are optional maintenance and coverage improvements. Runtime reproduction was not attempted, and no validation of integration execution is claimed.
- Recommended next step: Have the coordinator triage the inherited pathspec issue and authorize a narrow literal-path copy fix with a regression proving the clipboard excludes matching sibling files. Keep the existing key paths, selection rules and modal choices. Address optional provider relocation separately so the correctness fix stays reviewable. No new design phase or pull request is needed for this handoff.
