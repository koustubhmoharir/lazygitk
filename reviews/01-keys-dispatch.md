# Review 01: command identity, key catalogue and dispatch

## Summary

Read-only review of HEAD `a9d76554252e37e8cbc8a8aa742bad9bbe3aa307` against upstream `4e1eebe278790f257a875da6633234391a5a85b3` (BASE).
The catalogue, projection, trie, prefix-flow and binding-cache implementations are predominantly additive fork modules. The gocui hooks are small: six added lines each in `gui.go` and `keybinding.go`.
The largest upstream integration edits are `pkg/gui/keybindings.go` (+220/-83), `command_log_panel.go` (+80/-51), `context/menu_context.go` (+57/-13), and `options_map.go` (+50/-40), including metadata/alignment changes.
Found a default-configuration UX defect in disabled prefix entries, a misleading merge-guide classification, an effective-label discrepancy affecting honoured test overrides, and an inventory omission.
No repository files were changed. No tests, integration runs, builds, generation, commits or PR operations were performed. The permitted read-only annotation checker was run on all 31 listed metadata-only files: 26 passed and five failed.

## Findings

### F1. Prefix continuation rows discard the leaf's disabled state

- Category: ux
- Severity: medium
- Confidence: CLEAR BUG
- Location: `pkg/gui/fork_prefix_menu.go:211`, `pkg/gui/fork_prefix_menu.go:199`, `pkg/gui/context/menu_context.go:187`
- Problem: `show` copies `GetDisabledReason` only for bindings ending at the current trie node. Ordinary continuation rows, including rows whose next key directly invokes a leaf, are constructed with no `Binding`, `DisabledReason` or leaf tooltip. Consequently a failed precondition is rendered as an enabled action. Most normal prefix actions use this continuation branch, rather than the binding branch that preserves disabled metadata.
- Scenario / evidence: Open `s` in Files in a repository with no previous checkout. `forkCheckoutPreviousDisabledReason` (`pkg/gui/controllers/fork_checkout_previous.go:47`) returns `NoPreviousBranch` without `Inapplicable`, so `forkTreeHasApplicable` deliberately retains `s p`. Its `p` row has nil `DisabledReason`, so it is not struck through. Accepting it pops the menu, then `run` invokes the binding and reports the failure. Similarly, with an explicit commit set, `d` offers enabled-looking comparisons that will refuse with “Clear the selection first (Escape)”; the selection guard deliberately returns a failed precondition, not an inapplicable reason. This contradicts Design.md's requirement that failed preconditions be crossed out in every menu. Source paths were traced through prefix construction, menu rendering/acceptance, and the final handler guard; no runtime reproduction was run.
- Suggested fix: For a continuation resolving to a single applicable leaf, project that leaf's disabled reason, tooltip and relevant menu metadata onto the accelerator row, while preserving token handling and the final live guard. Add a focused regression covering a disabled direct continuation and an enabled sibling. Do not treat a failed precondition as inapplicable or hide it.

### F2. The metadata-only merge list contains substantive behavior changes

- Category: docs
- Severity: medium
- Confidence: n/a
- Location: `docs-master/fork/MergeGuide.md:578`, `pkg/gui/controllers/bisect_controller.go:51`, `pkg/gui/controllers/tags_controller.go:43`, `pkg/gui/controllers/helpers/refresh_helper.go:569`, `pkg/gui/controllers/helpers/repos_helper.go:379`
- Problem: The section says these files “only gained CommandID/Verb metadata” and tells a merger to keep upstream's code and mechanically re-add metadata. That instruction would lose real fork behavior in several listed files.
- Scenario / evidence: Ran the documented checker against all 31 paths, using the actual BASE hash. Exit 1; 26 passed, five failed. Inspected every failing diff:
  - `bisect_controller.go`: wraps returned bindings with `guardCommitSelectionBindings`.
  - `tags_controller.go`: adds the `w b` scope and `w v` show-in-Commits bindings.
  - `helpers/refresh_helper.go`: adds the generation-guarded completion callback and commit-scope refresh plumbing; changes reflog filtering semantics.
  - `helpers/repos_helper.go`: adds loading-pane presentation, named error result and deferred restoration after a failed switch.
  - `services/custom_commands/keybinding_creator.go`: introduces a computed `commandID` local in addition to metadata fields. This is identity plumbing rather than a separate workflow change, but it still fails the section's advertised annotation-only check.
  Some substantive hooks are documented elsewhere in the guide, making the metadata-only classification contradictory rather than compensating for it. The Keys section also calls `s n` global (`MergeGuide.md:175`), although the binding is Status-only (`pkg/gui/keybindings.go:94`).
- Suggested fix: Remove behavior-changing files from the metadata-only list and point to their actual integration entries. Document the tag and bisect hooks explicitly where needed. Classify the custom-command identity helper accurately, and correct the `s n` scope. Re-run the checker on the remaining list.

### F3. Effective command-key labels retain descendants suppressed by explicit leaves

- Category: bug
- Severity: low
- Confidence: CLEAR BUG
- Location: `pkg/gui/fork_command_key_labels.go:45`, `pkg/gui/fork_command_key_labels.go:58`, `pkg/gui/fork_prefix_menu.go:127`, `pkg/gui/fork_effective_command_paths.go:53`
- Problem: `forkCommandLabelsForView` walks every child in the structural trie. Dispatch suppresses default descendants when an applicable explicit leaf owns the prefix and no explicit descendants exist. The options-bar/palette display projection implements that suppression, but `CommandKeyLabel` does not, so these paths can disagree.
- Scenario / evidence: With the internal `LAZYGITK_HONOUR_KEY_CONFIG=1` switch, set `commandBindings: {app.quit: [[t]]}` and leave the `t d +` context-increase command at its default. Dispatch handles `t` as quit immediately; `t d +` is unreachable. `forkCommandLabelsForView` nevertheless returns `t d +` for `diff.context.increase`, and context-size error hints can recommend that path. The same mismatch occurs for other explicit leaf/default descendant combinations. `TestForkEffectiveDisplayPathsRespectLeafAndDescendantChoice` already describes the expected suppression in the separate display implementation, but the key-label tests do not cover it. This finding affects honoured test overrides/direct programmatic configs; normal user key settings are discarded.
- Suggested fix: Share the effective-route traversal between display bindings and key labels, retaining any required thread-safety boundaries. Add a targeted label regression with an explicit leaf and default descendant. Avoid maintaining a third independent dispatch approximation.

### F4. Static low-level inventory omits the new command registration API

- Category: code-quality
- Severity: low
- Confidence: n/a
- Location: `new-features/inventory/generate.go:137`, `pkg/gui/fork_command_dispatch.go:60`, `pkg/gui/keybindings.go:498`
- Problem: The collector recognizes only selector calls named `SetKeybinding` and `SetEditKeybindings`. It does not recognize `SetCommandKeybinding`, which now registers all compiled normal-pane roots and prefix accelerators.
- Scenario / evidence: Source search finds the new registration calls in `fork_command_dispatch.go:60` and `keybindings.go:498`; the collector predicate excludes both. The audit therefore misses the low-level registrations that implement the feature it is intended to inventory. No generator was run, because the inventory generator writes its JSON into the repository.
- Suggested fix: Include `SetCommandKeybinding` in low-level-call collection and extend `TestCollectConditionsAndLowLevelCalls` with the new API.

## Merge-ease notes

All counts below are added/deleted lines from `git diff --numstat BASE HEAD`, not estimated semantic change size.

- `pkg/gui/types/keybindings.go` (+23/-4): additive binding metadata and command-path fields; appropriate shared seam. Alignment accounts for some churn.
- `pkg/gocui/keybinding.go` (+6/-0): command flag and opt-in normalization hook; minimal and commented. Native matching remains separate.
- `pkg/gocui/gui.go` (+6/-0): small editor guard in the matching loop; minimal. It specifically skips printable command registrations when editing without `KeybindOnEdit`, rather than suppressing every possible command key.
- `pkg/gui/keybindings.go` (+220/-83): substantial textual churn, much of it binding metadata/alignment. Projection, dispatch and cache logic are extracted into fork files with comments at the hooks. An additional move of whole upstream functions would probably increase merge friction. Keep the calls to projection, cache storage, command registration and label updates together when merging.
- `pkg/gui/context/menu_context.go` (+57/-13): most sensitive upstream menu seam. Prefix controls, matching precedence, acceptance guard, disabled styling and command-path labels all touch upstream methods. New state/setters and the acceptance hook are reasonable additive extensions. The prefix-only options adjustment and blocker construction could be extracted behind a helper to shorten `GetKeybindings`; preserve the call order.
- `pkg/gui/menu_panel.go` (+23/-5): small hooks plus a fork-specific branch selecting physical essential keys. Acceptance state must be installed before menu focus/registration. Extraction of essential-key calculation is possible, but the actual integration is localized and commented.
- `pkg/gui/context.go` (+14/-0): localized invalidation, frame/layout hooks and reset after focus restoration. `Activate`'s reset is part of command correctness, not just rendering; the merge guide currently mentions invalidation/focus frames but should also name this reset explicitly.
- `pkg/gui/options_map.go` (+50/-40): real replacement of upstream hint selection with effective projected routes. Some helpers could move to `fork_command_hints.go` (`localFirstForkHintBindings`, `forkHintBindingInContext`) without changing the hook. The replacement is documented.
- `pkg/gui/command_log_panel.go` (+80/-51): extensive replacement of legacy key reads in tips, justified by the new paths. `commandRoute`/`keyTip` can be moved into a fork file to reduce additions in this upstream file, but each rewritten tip still needs deliberate preservation on merge.
- `pkg/cheatsheet/generate.go` (+3/-10): small migration to `Binding.KeyLabels`; appropriate seam.
- `pkg/gui/controllers/list_controller_trait.go` (+4/-4): minimal behavioral change adding `Inapplicable`; documented. Do not merge it as metadata-only.
- `pkg/gui/controllers/jump_to_side_window_controller.go` (+45/-9): keeps keyless jump definitions and distinct destinations. Most added logic is `focusPanelTitle`, which can move into a fork-owned controller file; the structural loop change itself must stay.
- `pkg/gui/keymap/`, `pkg/gui/fork_*.go`, identity/path helper files, three scoped config files, `pkg/gocui/command_*.go`, and the audit generators are additions relative to BASE and avoid upstream source conflicts.

Metadata-list audit covered all 31 listed paths, exceeding the requested six-file sample. The 26 successful paths were:

- Controllers: `confirmation_controller.go`, `context_lines_controller.go`, `diffing_menu_action.go`, `edit_config_action.go`, `filter_controller.go`, `filtering_menu_action.go`, `git_flow_controller.go`, `list_controller.go`, `menu_controller.go`, `remotes_controller.go`, `rename_similarity_threshold_controller.go`, `search_controller.go`, `search_prompt_controller.go`, `side_window_controller.go`, `snake_controller.go`, `status_controller.go`, `submodules_controller.go`, `suggestions_controller.go`, `switch_to_diff_files_controller.go`, `undo_controller.go`, `workspace_reset_controller.go`, `worktrees_controller.go`.
- Controller helpers: `branches_helper.go`, `diff_line_file_menu.go`, `worktree_helper.go`.
- `pkg/gui/extras_panel.go`.

The other five paths and their substantive/mechanical differences are recorded in F2.

## Investigation and cache/performance assessment

Read the request, common rules, AGENTS.md, Tracker.md, Design.md and relevant Guide/MergeGuide sections. Reviewed the scoped implementation, catalogue, resolution/provenance logic, generators and relevant tests; skimmed the generated legacy map and skipped JSON outputs as requested. Followed adjacent call sites only where needed to establish dispatch, invalidation or annotation-check evidence.

The structural binding cache stores the actual registered objects and evaluates applicability live. Focus activation resets bindings (`context.go:214`), successful config reload resets them (`gui.go:434`), and repository initialization resets controllers then bindings (`gui.go:415-416`); controller replacement clears the cache (`controllers.go:26`). No stale-binding defect was established on those paths. Cache validity by static view alone would be insufficient without those hooks, so their preservation is a merge requirement.

Per-frame work still walks the current tree, evaluates applicability and copies display bindings; the cache eliminates constructor/projection/trie rebuild work, not all allocations. Panel-label refresh rebuilds bindings, but its callers are view-property reconfiguration rather than the ordinary options-bar frame path. No profiling or benchmark was run and no quantitative performance claim is made.

Prefix acceptance uses a one-use frame token and verifies repository generation, Git-command identity, static/side context, selection IDs/ranges and diff/conflict content. Replacing a menu or changing focus invalidates the active token. No stale prefix invocation was established by source tracing. The tests are useful but largely use a synthetic flow harness; they should not be treated as end-to-end evidence for every focus/refresh interaction. The cache test named `TestForkBindingCacheClearedWithControllers` calls `clearForkBindingCache` directly rather than exercising controller replacement, so it does not protect the wiring by itself.

## Out of scope observations

No additional cross-area defect is asserted. The tag, bisect, refresh and repository-switch changes were examined only to verify the metadata-only claim, not to review their workflows.

## Commands run and outcomes

- `cat /tmp/lazygitk-review-run-1793910387/01-worker-request.md`, `cat /tmp/lazygitk-review-run-1793910387/common-review-rules.md`, and reads of AGENTS.md / fork documents: completed.
- `git status --short` before and during review: the same existing untracked `HANDOFF.md` and `test/.viminfo`; neither touched.
- `git rev-parse BASE`: failed because BASE is not a ref in this checkout. All subsequent baseline comparisons used the full hash supplied in the common rules. No ref was created.
- `git rev-parse HEAD`: `a9d76554252e37e8cbc8a8aa742bad9bbe3aa307`.
- `git log --oneline 4e1eebe278790f257a875da6633234391a5a85b3..HEAD -- <keymap, dispatch, prefix, cache, gocui, menu and generator paths>`: inspected scoped history.
- `git diff 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <upstream integration paths and failing annotation paths>`, plus `git diff --stat` / `--numstat` for scoped files: inspected changes and recorded footprint above. Added-file implementations were read directly.
- `cat`, `sed -n`, `rg --files`, and `rg -n` over the scoped source/tests and adjacent lifecycle callers: source inspection completed. A few exploratory reads used nonexistent guessed filenames; corrected by source search, with no effect on results.
- `go run new-features/inventory/check_annotations.go -base 4e1eebe278790f257a875da6633234391a5a85b3 <all 31 paths in MergeGuide.md's metadata-only list>`: exit 1, 26 annotation-only results and five non-metadata AST results, all investigated in F2. This program reads source/Git and does not generate or modify repository files.
- `git diff --check 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <keymap, keybindings, projection, dispatch, cache, gocui command key, menu context and generator paths>`: exit 0, no whitespace errors in that checked subset.
- Tests: none, per the request's default. Findings are based on traced code paths, not claimed runtime reproduction. No integration tests, full unit suite, generator, formatter, build or vet command was run.

## Files changed, remaining problems and recommended next step

Only `/tmp/lazygitk-review-run-1793910387/01-worker-result.md` was created. No implementation or narrowly scoped fix iteration was authorized by this read-only worker request; none was attempted. Existing repository changes were preserved.

F1-F4 remain unresolved. No new design phase is needed for these findings: F1 follows the recorded disabled-action display rule, F2 corrects documented facts, F3 aligns labels with implemented ownership rules, and F4 completes audit coverage. Next, have the coordinating reviewer validate/prioritize the findings, then authorize small fixes with focused regressions for F1/F3/F4 and a separate merge-guide correction for F2. Runtime validation of the new regression cases remains outstanding; this report does not substitute for it.
