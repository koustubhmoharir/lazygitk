# Review 15: merge-footprint

## Summary

Read-only whole-fork merge-ease audit of HEAD `a9d76554252e37e8cbc8a8aa742bad9bbe3aa307`, against BASE `4e1eebe278790f257a875da6633234391a5a85b3`.
The fork already separates most large Git/UI flows well; large deletion counts often represent desirable delegation.
Excluding generated keybindings/schema/test_list, 343 existing files are modified (+6,503/-3,959), and four existing integration scenarios deleted (-297): 347 upstream paths total.
This includes 129 modified Go non-test files, 189 modified integration scenarios, 12 modified upstream *_test.go files, and four deleted scenarios. The 129 includes harness Go sources.
There are 530 added files: 209 non-test Go sources, 305 test files (integration scenarios or *_test.go), and 16 other files. Definitions are explicit because the guide's source/test counts have no counting recipe.
The local upstream/master probe produces 11 conflicted files, only two handwritten sources. Biggest actionable risks are documentation drift, dispersed string edits, and integration contracts outside the conflicted files.
No implementation, tests, builds, commits, or repository writes were performed. The throwaway merge was aborted.

## Findings

### F1. The merge guide is stale about footprint, remotes, test rewrites, and metadata-only files
- Category: docs
- Severity: medium
- Confidence: n/a
- Location: docs-master/fork/MergeGuide.md:11, :13, :40, :578, :602
- Problem: Baseline says 104 modified sources / 157 tests / 380 additions (168 source / 212 test). Actual definitions and counts above are substantially larger. It names origin as upstream, while origin is the fork and upstream is jesseduffield/lazygit. It says translations are untouched, despite six modified JSON files. The metadata-only list includes refresh_helper.go and repos_helper.go, which now contain behavioral changes also described elsewhere.
- Scenario / evidence: git diff --numstat and --diff-filter=A/M/D against the full BASE hash; git remote -v. refresh_helper.go adds generation-guarded ThenIfRepoUnchanged, scope capture/passing, HasPathOrAuthorFilter; repos_helper.go adds loading/restore and named return. These cannot be resolved as metadata-only changes. The Upstream tests list omits material replacements of cherry-pick, autostash, stack sync, worktree deletion/forwarding, and the four removed scenarios.
- Suggested fix: First refresh the guide with a reproducible counting definition and current remotes. Remove refresh_helper.go/repos_helper.go from the metadata-only bucket (or expressly limit that bucket to named hunks). Add the missing test-replacement/removal categories and translation explanation; list the missing integration/harness paths below.

### F2. English additions and deleted legacy keys create avoidable i18n conflict surface
- Category: merge-ease
- Severity: medium
- Confidence: n/a
- Location: pkg/i18n/english.go:14, :304, :981, :1558, :1844, :2220, :2571; pkg/i18n/translations/ja.json:528 (and five others)
- Problem: +685/-15 makes english.go the largest production edit. Most additions are not a single isolated fork block: many insertion locations in both TranslationSet and EnglishTranslationSet, plus two Actions fields. An initial fork block exists and one comment explicitly promises separation, but later feature strings are distributed through the upstream tables. Local/Global legacy fields and values were removed, forcing translation edits. MustExitFilterModePrompt was edited in place.
- Scenario / evidence: zero-context english diff; translation diff; git log/show 47d30f89fb4f8b39c915e0706f17078c7e1f88f4. All 11 translation deletions remove KeybindingsMenuSectionLocal/Global after the palette switched to pane sections; no translation text was rewritten. The intro replacement is intentional and has a Fork: comment plus i18n.go override. Current upstream English changes happen to merge cleanly, but future additions near the scattered fields need broader review.
- Suggested fix: Consolidate new fields into one marked fork block (Go requires direct struct fields here; a separate Go file cannot extend an existing struct), and assign fork defaults in a fork-owned initializer called after the upstream literal. Keep the original upstream string keys/translation entries where useful. readLanguageFile uses json.Unmarshal without unknown-field rejection, so retaining legacy JSON entries does not require keeping their struct fields at runtime. Keep the intro override and document the in-place filtered-log prompt change. Do this as a small behavior-preserving change, not an i18n redesign.

### F3. Several wholly fork-owned methods remain in frequently edited upstream files
- Category: merge-ease
- Severity: medium
- Confidence: n/a
- Location: pkg/gui/controllers/helpers/commits_helper.go:238, :264; pkg/gui/controllers/helpers/merge_and_rebase_helper.go:193; pkg/gui/command_log_panel.go:89; pkg/gui/options_map.go:100; pkg/gui/keybindings.go:516; pkg/gui/controllers/options_menu_action.go:17
- Problem: New methods and policy assembly still enlarge the upstream footprint even though suitable fork modules already exist. Examples: draft save/cancel/recall/report methods (~60 lines), operation-finished settlement (~29), commandRoute/keyTip (~25), two hint helpers (~22), palette assembly (~60), and usage-aware handler body. Moving these methods within the same Go package needs no exported interface or behavior change.
- Scenario / evidence: BASE..HEAD diffs; these methods are absent at BASE and contain fork policy rather than upstream implementation. The captured branch/reset/merge helpers demonstrate that a short commented delegate is already the project's preferred pattern.
- Suggested fix: Prioritize draft helper methods, palette replacement body, and hint/usage helpers; leave fields and lifecycle hook calls in upstream types/functions. Add // Fork: at replacement hooks naming what changed and the destination module. Avoid wholesale moves of old upstream code or mechanical metadata churn; those would trade one merge burden for another.

### F4. Some upstream scenarios were substantially rewritten and lost original assertion detail
- Category: code-quality
- Severity: medium
- Confidence: n/a
- Location: pkg/integration/tests/branch/merge_non_fast_forward.go:25; pkg/integration/tests/sync/push_stacked_branches.go:8; pkg/integration/tests/worktree/remove_worktree_and_delete_local_and_remote_branch.go:8; docs-master/fork/MergeGuide.md:602
- Problem: These changes exceed the guide's promise of automation opt-ins, extra reviews, config filenames and Space-to-v adaptations. Some intentionally change the scenario's contract; some also reduce assertions independently of the changed contract. merge_non_fast_forward replaces exact graph/topology rows with subject-based EventuallyContent/ContainsLines and exercises different source order/modes.
- Scenario / evidence: Before, merge_non_fast_forward asserted both complete graph outcomes and fast-forward-dependent menu order; after, it checks presence of subjects/branch labels. A wrong graph/parent relationship could satisfy those new checks. push_stacked_branches now proves single-branch isolation instead of pushing the stack, while new push_selected_branch_with_related_branches also describes only the selected branch. pull_stack_review supplies explicit stack update coverage, but this audit did not establish a replacement integration scenario for the old push-all success outcome.
- Suggested fix: Keep intentional changed semantics (no autostash, occupied worktree refusal, selected branch defaults) and document them as replacements. Preserve unaffected graph/parent assertions, preferably using Git parent OIDs where UI glyphs intentionally changed. Identify and link replacement coverage for retained explicit stack-push behavior; if none, add it in a later implementation task. Do not restore obsolete copy/history/autostash flows merely to retain old scenarios.
- Question for next worker: Which new scenario is the intended successor to upstream's successful push-all-stack test? This is a coverage mapping question, not a request to redesign the product.

### F5. Replacement hooks have uneven Fork: markers
- Category: merge-ease
- Severity: low
- Confidence: n/a
- Location: pkg/gui/controllers/main_view_controller.go:99; pkg/gui/context/menu_context.go:267, :323; pkg/gui/controllers/custom_patch_options_menu_action.go:166; pkg/commands/git_commands/rebase.go:217
- Problem: MainView dynamic primary/discard binding replacements have no Fork: marker in that file. Menu prefix/accept gates and no-autostash replacements often have useful ordinary comments but lack the requested searchable marker. A file having some Fork: comments does not identify every replacement in it.
- Scenario / evidence: rg Fork: and zero-context diffs. rebase.go's updateRefs change is marked, while no-autostash and the common start wrapper are explained without the prefix. menu_context.go marks dimming but not beforeAccept/CommandActive/disabled suppression gates.
- Suggested fix: Add short markers at the actual replaced dispatch/accept/start sites with the upstream behavior and fork rationale. No blanket annotation of every metadata field is needed.

## Merge-ease notes

### Probe merge and upstream changes that merge cleanly

Probe uses the existing local upstream/master `0002c0354770ff8ec6abe4d5ab4d025ea40e1619`; no internet fetch or claim about the latest remote tip. BASE..upstream/master changes 43 paths (+1,300/-255). Clone: /tmp/probe-merge-15, with a shared object store and explicitly fetched upstream tracking ref from the source repository.

11 conflicted paths:

- docs-master/keybindings/Keybindings_en.md
- docs-master/keybindings/Keybindings_ja.md
- docs-master/keybindings/Keybindings_ko.md
- docs-master/keybindings/Keybindings_nl.md
- docs-master/keybindings/Keybindings_pl.md
- docs-master/keybindings/Keybindings_pt.md
- docs-master/keybindings/Keybindings_ru.md
- docs-master/keybindings/Keybindings_zh-CN.md
- docs-master/keybindings/Keybindings_zh-TW.md
- pkg/commands/git_commands/commit_loader.go
- pkg/gui/controllers/bisect_controller.go

The nine generated files should be regenerated after resolving sources; reorganizing product code will not remove generated output conflicts.

bisect_controller.go has five conflict regions, largely because CommandID/Verb insertion reformats menu literals while upstream replaces them with shared mark/skip/reset item builders and generalizes commit to Ref. This is the largest handwritten conflict. Keep upstream helpers, reapply identities to their returned items and preserve the explicit-selection guard. Moving the guard alone would not avoid metadata conflicts; a fork-owned annotation step could reduce them, but would add identity mapping indirection, so this is an unresolved tradeoff rather than a recommendation to redesign metadata. Current structure deliberately trades mechanical conflicts for visible stable identities.

commit_loader.go has one conflict region: the fork takes scope.limit while upstream changes opts.Limit from bool to an integer and formats the requested count. This is unavoidable semantic overlap, already documented as scope-sensitive. It cannot be fixed by choosing one side. Fork-owned fork_commit_scope.go also assumes bool in commitScopeLogArgs.limit and newCommitAncestorFilter(opts.ScopeHashes, opts.Limit), so resolving only the conflicted line leaves incompatible code. Port the scope contract and its tests to the new integer limit, including no Git walk limit for scoped filters and count while reading ancestors. This is an integration requirement, not a bug in the current unmerged HEAD.

Cleanly merged nearby changes to inspect deliberately:

1. refresh_helper.go changes Limit to a calculated integer, stores LoadedCommitLimit, selects the next/current bisect commit, and renders inside the guarded model apply instead of a later refreshView call. The fork's captured ScopeHashes/ScopeDescendants/ScopeRef arguments survive the textual merge, as do ThenIfRepoUnchanged and HasPathOrAuthorFilter. However, the bisect selection calls and scope filtering must be checked together: the target may be outside the active scope, and the loaded-limit threshold must reflect kept scoped rows rather than silently making a scoped list behave like the unfiltered list. No bypass demonstrated; this is a required semantic merge review.
2. controllers.go removes two individual bisect attachments and attaches a generic Ref-based bisect controller to six contexts, including Branches/RemoteBranches/Tags. Existing fork ref-copy/scope hooks survive. Revisit guardCommitSelectionBindings against new contexts and give new menu item builders identities/verbs; otherwise the new routes can be unkeyed or bypass a selection guard. The documented identity test is the appropriate future verification.
3. local_commits_controller.go and sub_commits_controller.go adopt loadAllCommitsIfSelectionIsNearEnd and LoadedCommitLimit. These merge cleanly near fork explicit selections and scoped logs. Adapt scope limits, don't retain a boolean-to-300 adapter that discards upstream's bisect-driven limits.
4. gui.go changes initialContext to focus Commits when bisect started. It does not bypass the fork's onNewRepo notices/title, constructor clipboard injection, loadNewRepo toast, or controller-cache clearing in the inspected diff.
5. presentation/commits.go changes bisect status/candidate handling and removes bisectBounds. The fork commitListKey digest remains in loadPipesets. Retain full-list digest and F5 cache invalidation.
6. context/local_commits_context.go adds LoadedCommitLimit accessors. The fork's explicit selection view-model fields remain. No newly added ListCursor cancel-range entry point was found in this upstream diff.
7. i18n/english.go changes Bisect field names and adds strings. Textual merge succeeds because these are outside the fork insertion hunks. Check all fork/new command identity expectations against the replaced bisect builders.

After collecting conflicts and diffs, `git -C /tmp/probe-merge-15 merge --abort` succeeded; probe status is clean. Clone remains as an aborted throwaway directory, with no merge in progress. Source status before/after was identical: pre-existing untracked HANDOFF.md, reviews/, test/.viminfo.

### Largest 25 production source edits

Ranked by additions + deletions. Production source means existing .go files outside integration/ and excluding *_test.go. This avoids treating test fixtures or documentation as production code. The complete inventory below also includes every non-source upstream edit.

| Rank | Existing upstream file | + | - | Assessment and comments |
| --- | --- | ---: | ---: | --- |
| 1 | pkg/i18n/english.go | 685 | 15 | Not minimal: additions in many struct and initializer locations. Consolidate fork fields and values; see F2. Intro has Fork:, most string additions do not. |
| 2 | pkg/gui/controllers/branches_controller.go | 128 | 353 | Mostly minimal wrappers plus metadata; 353 removals reflect replaced menus, not 353 lines of new logic. Captured implementations already separate. checkoutByName still carries capture/name callbacks that could move. Replacement hooks have Fork: explanations. |
| 3 | pkg/gui/controllers/local_commits_controller.go | 208 | 145 | Metadata, selection guard, cherry-pick wrapper and reword/draft/fixup/amend hooks. Most logic already separate; placement/submission orchestration could be extracted narrowly. Fork: explains drag, placement, draft submission and paste replacements. |
| 4 | pkg/gui/controllers/helpers/merge_and_rebase_helper.go | 77 | 265 | Large deletions replace menus with captured helpers. Good wrappers at RebaseOntoRef/MergeRefIntoCheckedOutBranch. Move SetOnOperationFinished/settleOperationFinished methods to fork file, retaining field and two calls. Fork: at callback field and replacement delegates; settle helper has ordinary descriptive comment. |
| 5 | pkg/gui/keybindings.go | 220 | 83 | Metadata plus substantive cache/usage/registration policy. Projection and reset hooks are commented Fork:. Move callKeybindingHandlerWithUsage and cache-aware display wrapper bodies into fork modules where practical; retain minimal integration calls. |
| 6 | pkg/gui/controllers/files_controller.go | 197 | 81 | Metadata plus commented additive bindings, tree navigation, copy-menu sharing, fresh tool selection, effective hints and no implicit discard. Most policy is already in fork files. Could extract tree binding construction and tool wrapper; avoid moving upstream copy implementation just for size. Fork: at major integrations. |
| 7 | pkg/gui/controllers/helpers/refs_helper.go | 32 | 136 | Large removed reset/new-branch code replaced by small documented delegates. Named checkout has Fork:. Existing legacy autostash remains intentionally dormant for default routes. No broad extraction warranted. |
| 8 | pkg/gui/command_log_panel.go | 80 | 51 | keyTip and commandRoute are fork-owned helpers embedded in upstream file; move them. Tip table rewrites are necessary to name effective routes. Header has Fork:, helper policy lacks that marker. UI-thread log bounce is inherited from BASE, not fork debt. |
| 9 | pkg/gui/controllers/main_view_controller.go | 114 | 12 | Mostly metadata, dynamic identity references and explicit navigation bindings. Dynamic helpers already separate. No Fork: comment in this file explains changed primary/discard identity binding fields or added navigation routes; add concise integration marker. |
| 10 | pkg/gui/controllers/custom_patch_options_menu_action.go | 69 | 44 | Mostly metadata and necessary submit-result call-site hooks. No-stash replacement explains itself in ordinary comment, missing literal Fork:. Extract nothing broad; preserve two report-success integrations. |
| 11 | pkg/gui/controllers/basic_commits_controller.go | 64 | 46 | Metadata plus commented placement/reset/copy/selection additions and external-tool callback. Most logic already separate. Hooks are suitable; minor binding composition can move if simplifying, not as broad cleanup. |
| 12 | pkg/gui/controllers/helpers/commits_helper.go | 101 | 5 | Not minimal: new TakeCommitMessageSubmitReporter, Save/Cancel/Recall methods live here. Move wholly new methods to a new helpers/fork_commit_draft.go in the same package, retaining fields/options and lifecycle calls. Fork: comments are thorough. |
| 13 | pkg/gui/controllers/commits_files_controller.go | 82 | 21 | Metadata, small tree/copy/direct-diff hooks and tool callback. Good separate implementations. Copy helper refactor intentionally shares old menu; Fork: mostly present (direct-diff comment says Fork without colon). |
| 14 | pkg/gui/controllers/options_menu_action.go | 33 | 64 | Full palette assembly remains inline (~60 lines) despite fork command_palette module. Move replacement body to fork file, keep Call delegate. Fork: explains sections and usage, but top Call entry does not explicitly say upstream local/global palette is replaced. |
| 15 | pkg/commands/git_commands/rebase.go | 64 | 30 | Necessary changed preparation signature and caller propagation. runInteractiveRebase helper could live in fork file. Clean-start and no-autostash replacement comments explain rationale but lack Fork: prefix; updateRefs hook has it. |
| 16 | pkg/gui/options_map.go | 50 | 40 | Not minimal: wholly new localFirstForkHintBindings and forkHintBindingInContext belong in fork_command_hints.go. Selection/prioritization logic could become one helper call. Fork: at historical ordering and continue hint; main replacement loop has ordinary comments. |
| 17 | pkg/gui/controllers/commit_message_controller.go | 34 | 54 | Good minimal draft-method delegates plus removed obsolete history logic and metadata. Fork: at recall/quit/Escape paths. No further extraction recommended. |
| 18 | pkg/gui/controllers/sync_controller.go | 51 | 22 | Good replacement entry points and metadata. handleFocusedBranchPush/Pull could move into existing branch-sync module while retaining binding callback hooks. Fork: clearly explains captured target replacing implicit current branch. |
| 19 | pkg/gui/context/menu_context.go | 57 | 13 | Contains unavoidable changes to display, accelerator, accept and disabled dispatch. New setters can move to a new context/fork_menu_prefix.go; keep fields/hooks. Only display policy uses literal Fork:, prefix comment lacks colon and accept gates lack replacement explanation. |
| 20 | pkg/gui/gui.go | 67 | 3 | Mostly unavoidable fields, initialization/config/subprocess/theme/repo-switch hooks; extensive Fork: explanations. Group state later only if natural, not a reason to rewrite upstream Gui. |
| 21 | pkg/gui/views.go | 10 | 58 | Good reduction: border switch removed for updateFocusFrames; cursor marker/hints are minimal and Fork: documented. |
| 22 | pkg/gui/controllers/global_controller.go | 51 | 4 | Mostly metadata and a commented additive binding wrapper. Suitable minimal hooks; actual implementations separate. |
| 23 | pkg/gui/controllers/jump_to_side_window_controller.go | 45 | 9 | Metadata plus per-pane identity/verb construction. Necessary binding identity logic; focusPanelTitle could move to a fork helper but low payoff. Fork: explicitly explains retaining keyless pane commands and distinct identities. |
| 24 | pkg/gui/controllers/helpers/cherry_pick_helper.go | 0 | 53 | Only removals: mark-copy helper and resetIfNecessary. Dormant-state policy documented in guide. No reason to move or restore deleted implementation; integration comment should preserve why marks have no producer. |
| 25 | pkg/gui/controllers/helpers/tags_helper.go | 3 | 50 | Tag creation body replaced by captured delegate; Fork: explains selection/repository capture. Good minimal hook. |

Other large non-test-file edits: AGENTS.md +38/-531 (intentional policy replacement; guide describes it); docs-master/Config.md +98/-112 (manual fork config/default documentation, not a minimal hook; add guide entry); test/default_test_config/config.yml +146/-0 (test fixture, assessed below). docs-master/Custom_Command_Keybindings.md +21/-0 and Custom_DiffRenderers.md +5/-0 are user-facing updates worth preserving and documenting. These are in the inventory rather than the production ranking.

### MergeGuide integration-point coverage

The list is broadly good for production behavior. Most entries accurately point to small wrappers. Specific inaccurate buckets are F1, and missing comment details are F5.

Existing non-test upstream paths whose full name or basename never occurs in the guide (lexical check, then manual qualification):

- docs-master/Config.md (+98/-112)
- docs-master/Custom_Command_Keybindings.md (+21/-0)
- docs-master/Custom_DiffRenderers.md (+5/-0)
- pkg/gui/controllers/helpers/helpers.go (+3/-0)
- pkg/gui/gui_driver.go (+10/-0)
- pkg/gui/services/custom_commands/handler_creator.go (+2/-0)
- pkg/i18n/translations/ja.json (+0/-2)
- pkg/i18n/translations/nl.json (+0/-1)
- pkg/i18n/translations/pl.json (+0/-2)
- pkg/i18n/translations/pt.json (+0/-2)
- pkg/i18n/translations/zh-CN.json (+0/-2)
- pkg/i18n/translations/zh-TW.json (+0/-2)
- pkg/integration/components/assertion_helper.go (+7/-1)
- pkg/integration/components/file_system.go (+21/-0)
- pkg/integration/components/menu_driver.go (+7/-0)
- pkg/integration/components/test_driver.go (+24/-0)
- pkg/integration/components/view_driver.go (+31/-0)
- pkg/integration/types/types.go (+9/-0)

Qualifications: gui_driver.go and integration/types/types.go are described through their GuiDriver methods elsewhere, and MenuDriver.Content, TestDriver clipboard/title and ViewDriver selection are described generically. Missing path names are documentation discoverability gaps, not all missing behavior descriptions. helpers/helpers.go adds ExternalTool field and stub construction (minimal, Fork: marked); the guide describes constructor wiring in controllers.go but not this registry. handler_creator.go only adds two comments explaining parameter choices, so no integration entry is necessary.

Upstream tests generally are not individually named in the guide. The complete inventory's Guide column lists every exact/basename absence, including all those tests, rather than implying their absence alone is a defect. The missing *categories* and specific removed scenarios are the concern.

The metadata-only bucket is otherwise plausible for the named binding-only controllers/helpers, but refresh_helper.go and repos_helper.go must be excluded from that blanket claim. A few entries have intentional harmless comments in addition to metadata (e.g. workspace_reset_controller); a metadata checker that preserves comments may flag those without implying behavior changed.

### Upstream test changes

189 modified existing integration scenarios, four removed existing scenarios, 12 modified existing unit-test files, excluding generated test_list.go.

- Minimal adapters: enabling automatic conflict staging explicitly where needed; inserting reviewed-flow approvals; repository toast expectations; lazygitk.yml filenames; v for opening commit files; legacy key/hint labels. Most small edits are in these categories, although toast expectations and the compatibility key map deserve explicit inclusion in Upstream tests.
- Replacements: branch deletion/creation/merge/reset/rename/upstream menus; cherry-pick selection instead of copying marks; sync selected-branch/explicit stack reviews; no autostash; refusal to move/delete occupied branches. They often improve safety assertions (e.g. fast_forward_branch_being_rebased_in_worktree additionally snapshots both indexes/files and verifies unrelated refs; new_branch_autostash checks no stash/ref/HEAD/file changes). Do not label all these as minimal upstream-test adaptations.
- Removed scenarios: cherry_pick/cherry_pick_across_worktrees.go (-79), cherry_pick/cherry_pick_range_after_paste.go (-106), commit/history.go (-53), commit/history_complex.go (-59). First two exercise removed copy-mark persistence; last two removed Git-history recall. Their deletion is consistent with documented replacement behavior, but guide only explicitly names the deleted unit history test. Record scenario removals and their fork successors.
- ui/mode_specific_keybinding_suggestions removes cherry-pick mark banner assertions; expected because marks have no producer. Other rebase/bisect/patch/merge assertions remain.
- merge_non_fast_forward weakens the original topology/output detail; see F4. Worktree-delete and stack default rewrites change outcomes deliberately, rather than merely weakening assertions; describe them as such.
- Unit tests: cheatsheet/generate_test.go adds command-path coverage; git_cmd_obj_runner_test.go adds runner test support and five lock-retry cases; oscommands/os_test.go adds clipboard cases; gocui/view_test.go adds explicit selection/highlight cases; helpers/refresh_helper_test.go adds repo-generation callback ordering tests; window_arrangement_helper_test.go adds four layout cases; integration/components/test_test.go supplies four driver stubs; theme/gocui_test.go adds a palette case. These additions can mostly live in fork-owned *_test.go files (the window table additions naturally belong in the existing table).
- Unit edits adapting removed/replaced behavior: commit_test.go removes TestGetCommitMessageFromHistory alongside its removed implementation; rebase_test.go adds clean-start status commands and --no-autostash without dropping success/error assertions; diff_renderer_config_manager_test.go changes the expected default name; theme_config_test.go checks themeDefaults directly for values overridden by the fork. These are small and explained; theme replacement behavior needs its separate fork test (fork_theme_palette_test.go exists). No unjustified assertion removal found in those unit edits.

### Integration harness

Existing modifications are small compared with feature code:

| File | + / - | Assessment |
| --- | ---: | --- |
| pkg/integration/components/assertion_helper.go | 7 / 1 | New assertEventuallyWithin; old default preserved. Minimal integration, missing guide path. |
| pkg/integration/components/common.go | 14 / 1 | ReviewBranchRebase plus changed discard wording. Review helper could move to additive file; documented generically. |
| pkg/integration/components/file_system.go | 21 / 0 | FileContentEventually + 5s timeout for deferred usage writes; move to fork-owned helper file. Not named in guide. |
| pkg/integration/components/menu_driver.go | 7 / 0 | Whole-menu Content matcher; tiny, documented generically. Could be additive method file. |
| pkg/integration/components/runner.go | 5 / 0 | LAZYGITK_HONOUR_KEY_CONFIG before scenario overrides; minimal Fork: marker and accurate guide entry. |
| pkg/integration/components/test_driver.go | 24 / 0 | Clipboard/title matchers; move optional methods to additive file. Fork: documented; guide describes methods without path. |
| pkg/integration/components/test_test.go | 16 / 0 | Required fake interface stubs; no weakening. |
| pkg/integration/components/view_driver.go | 31 / 0 | ExplicitSelection matcher; could be additive file; generic guide entry. |
| pkg/integration/components/views.go | 5 / 0 | Extras accessor; minimal generic guide entry. |
| pkg/integration/types/types.go | 9 / 0 | Clipboard/title/screen-mode interface additions; unavoidable shared contract, Fork: comments. Name path in guide. |
| pkg/gui/gui_driver.go | 10 / 0 | Clipboard/title simulation accessors; minimal and marked. Screen-mode methods are already in fork_screen_mode_area.go. |
| pkg/gui/test_mode.go | 4 / 1 | 40s -> 120s watchdog, clearly marked and accurately documented. |
| test/default_test_config/config.yml | 146 / 0 | Explicit compatibility bindings (+137), branch-only log and rawGit fixture. Necessary test layer, not production defaults. Key compatibility policy documented. |

Additive component files (branch_create/delete/merge, cherry_pick, eventual_content, fork_screen_mode, etc.) already demonstrate a better home for optional methods. This is low-risk, moderate/low payoff extraction, after the main production helpers.

Shared config already disabled autoFetch/autoRefresh/autoDetectExternalChanges at BASE; the guide accurately describes its current fixture, but these are not new fork modifications. showWholeGraph:false is a new fixture choice and should be included in the guide. The comment at config.yml:158 claiming legacy Local/Global/Navigation palette sections is stale: OptionsMenuAction always constructs pane sections; animateExplosion does not select a legacy palette. Fix the comment only.

No integration assertion infrastructure was globally weakened: default eventual timeout remains the same; longer per-call file-content waiting and watchdog extension are targeted. The runner deliberately exercises legacy configurable paths, so default-keymap scenarios must reset configuration (shared.UseDefaultKeymap does); retain both kinds when merging.

### i18n detail

English additions are predominantly additive but scattered; see F2. In-place upstream changes are the intro, MustExitFilterModePrompt, and removal of Local/Global section keys/values. The intro is explicitly required for the fork and forced back to English after localized data merges in i18n.go; that override is a good minimal integration. No broad reformatting of the entire English table was observed.

Translations: ja/pl/pt/zh-CN/zh-TW each lose two obsolete section entries; nl loses only Global because it did not have Local. Total six files, -11 lines. Commit 47d30f89f explains the switch from Local/Global sections to pane sections. The statement “translations are untouched” is factually false. No other translation content changes occur against BASE.

## Complete upstream-file inventory

Only files present at BASE, modified/deleted at HEAD. Excludes docs-master/keybindings/*, schema-master/*, pkg/integration/tests/test_list.go per request. “Guide” is “named” if full path or basename appears anywhere in MergeGuide; “absent” otherwise. It is a lexical discoverability aid, not proof that generic test categories fail to cover a path. All added files excluded from this upstream inventory.

| Upstream file | + | - | Status | Guide |
| --- | ---: | ---: | --- | --- |
| .gitignore | 1 | 0 | modified | named |
| AGENTS.md | 38 | 531 | modified | named |
| docs-master/Config.md | 98 | 112 | modified | absent |
| docs-master/Custom_Command_Keybindings.md | 21 | 0 | modified | absent |
| docs-master/Custom_DiffRenderers.md | 5 | 0 | modified | absent |
| justfile | 6 | 2 | modified | named |
| pkg/app/daemon/daemon.go | 2 | 0 | modified | named |
| pkg/cheatsheet/generate_test.go | 13 | 0 | modified | absent |
| pkg/cheatsheet/generate.go | 3 | 10 | modified | named |
| pkg/commands/git_cmd_obj_runner_test.go | 84 | 0 | modified | absent |
| pkg/commands/git_cmd_obj_runner.go | 6 | 0 | modified | named |
| pkg/commands/git_commands/commit_loader.go | 32 | 8 | modified | named |
| pkg/commands/git_commands/commit_test.go | 0 | 35 | modified | absent |
| pkg/commands/git_commands/commit.go | 0 | 15 | modified | named |
| pkg/commands/git_commands/patch.go | 5 | 16 | modified | named |
| pkg/commands/git_commands/rebase_test.go | 8 | 4 | modified | absent |
| pkg/commands/git_commands/rebase.go | 64 | 30 | modified | named |
| pkg/commands/git_commands/status.go | 33 | 2 | modified | named |
| pkg/commands/git_commands/sync.go | 21 | 1 | modified | named |
| pkg/commands/git_commands/working_tree.go | 0 | 4 | modified | named |
| pkg/commands/models/working_tree_state.go | 3 | 0 | modified | named |
| pkg/commands/oscommands/cmd_obj.go | 6 | 0 | modified | named |
| pkg/commands/oscommands/os_test.go | 73 | 0 | modified | absent |
| pkg/commands/oscommands/os.go | 18 | 1 | modified | named |
| pkg/config/app_config.go | 33 | 12 | modified | named |
| pkg/config/diff_renderer_config_manager_test.go | 3 | 1 | modified | absent |
| pkg/config/diff_renderer_config_manager.go | 25 | 4 | modified | named |
| pkg/config/theme_config_test.go | 5 | 3 | modified | absent |
| pkg/config/theme_config.go | 5 | 1 | modified | named |
| pkg/config/user_config_validation.go | 7 | 1 | modified | named |
| pkg/config/user_config.go | 28 | 4 | modified | named |
| pkg/gocui/gui.go | 6 | 0 | modified | named |
| pkg/gocui/keybinding.go | 6 | 0 | modified | named |
| pkg/gocui/tcell_driver.go | 3 | 1 | modified | named |
| pkg/gocui/view_test.go | 169 | 0 | modified | absent |
| pkg/gocui/view.go | 23 | 6 | modified | named |
| pkg/gui/command_log_panel.go | 80 | 51 | modified | named |
| pkg/gui/context.go | 14 | 0 | modified | named |
| pkg/gui/context/commit_message_context.go | 25 | 16 | modified | named |
| pkg/gui/context/confirmation_context.go | 2 | 0 | modified | named |
| pkg/gui/context/local_commits_context.go | 11 | 0 | modified | named |
| pkg/gui/context/menu_context.go | 57 | 13 | modified | named |
| pkg/gui/context/setup.go | 5 | 1 | modified | named |
| pkg/gui/controllers.go | 43 | 8 | modified | named |
| pkg/gui/controllers/basic_commits_controller.go | 64 | 46 | modified | named |
| pkg/gui/controllers/bisect_controller.go | 28 | 9 | modified | named |
| pkg/gui/controllers/branches_controller.go | 128 | 353 | modified | named |
| pkg/gui/controllers/commit_description_controller.go | 20 | 9 | modified | named |
| pkg/gui/controllers/commit_diff_actions.go | 2 | 2 | modified | named |
| pkg/gui/controllers/commit_message_controller.go | 34 | 54 | modified | named |
| pkg/gui/controllers/commits_files_controller.go | 82 | 21 | modified | named |
| pkg/gui/controllers/confirmation_controller.go | 6 | 0 | modified | named |
| pkg/gui/controllers/context_lines_controller.go | 4 | 0 | modified | named |
| pkg/gui/controllers/custom_patch_options_menu_action.go | 69 | 44 | modified | named |
| pkg/gui/controllers/diffing_menu_action.go | 12 | 4 | modified | named |
| pkg/gui/controllers/edit_config_action.go | 3 | 1 | modified | named |
| pkg/gui/controllers/files_controller.go | 197 | 81 | modified | named |
| pkg/gui/controllers/filter_controller.go | 2 | 0 | modified | named |
| pkg/gui/controllers/filtering_menu_action.go | 16 | 6 | modified | named |
| pkg/gui/controllers/git_flow_controller.go | 25 | 13 | modified | named |
| pkg/gui/controllers/global_controller.go | 51 | 4 | modified | named |
| pkg/gui/controllers/helpers/branches_helper.go | 21 | 11 | modified | named |
| pkg/gui/controllers/helpers/cherry_pick_helper.go | 0 | 53 | modified | named |
| pkg/gui/controllers/helpers/commits_helper.go | 101 | 5 | modified | named |
| pkg/gui/controllers/helpers/confirmation_helper.go | 2 | 0 | modified | named |
| pkg/gui/controllers/helpers/diff_helper.go | 17 | 6 | modified | named |
| pkg/gui/controllers/helpers/diff_line_file_menu.go | 3 | 1 | modified | named |
| pkg/gui/controllers/helpers/gpg_helper.go | 2 | 1 | modified | named |
| pkg/gui/controllers/helpers/helpers.go | 3 | 0 | modified | absent |
| pkg/gui/controllers/helpers/merge_and_rebase_helper.go | 77 | 265 | modified | named |
| pkg/gui/controllers/helpers/mode_helper.go | 29 | 10 | modified | named |
| pkg/gui/controllers/helpers/refresh_helper_test.go | 134 | 0 | modified | absent |
| pkg/gui/controllers/helpers/refresh_helper.go | 28 | 1 | modified | named |
| pkg/gui/controllers/helpers/refs_helper.go | 32 | 136 | modified | named |
| pkg/gui/controllers/helpers/repos_helper.go | 15 | 1 | modified | named |
| pkg/gui/controllers/helpers/tags_helper.go | 3 | 50 | modified | named |
| pkg/gui/controllers/helpers/window_arrangement_helper_test.go | 134 | 0 | modified | absent |
| pkg/gui/controllers/helpers/window_arrangement_helper.go | 9 | 2 | modified | named |
| pkg/gui/controllers/helpers/working_tree_helper.go | 38 | 6 | modified | named |
| pkg/gui/controllers/helpers/worktree_helper.go | 18 | 8 | modified | named |
| pkg/gui/controllers/jump_to_side_window_controller.go | 45 | 9 | modified | named |
| pkg/gui/controllers/list_controller_trait.go | 4 | 4 | modified | named |
| pkg/gui/controllers/list_controller.go | 11 | 11 | modified | named |
| pkg/gui/controllers/local_commits_controller.go | 208 | 145 | modified | named |
| pkg/gui/controllers/main_view_controller.go | 114 | 12 | modified | named |
| pkg/gui/controllers/menu_controller.go | 15 | 7 | modified | named |
| pkg/gui/controllers/merge_conflicts_controller.go | 31 | 0 | modified | named |
| pkg/gui/controllers/options_menu_action.go | 33 | 64 | modified | named |
| pkg/gui/controllers/prompt_controller.go | 14 | 0 | modified | named |
| pkg/gui/controllers/quit_actions.go | 15 | 0 | modified | named |
| pkg/gui/controllers/reflog_commits_controller.go | 5 | 0 | modified | named |
| pkg/gui/controllers/remote_branches_controller.go | 45 | 8 | modified | named |
| pkg/gui/controllers/remotes_controller.go | 12 | 0 | modified | named |
| pkg/gui/controllers/rename_similarity_threshold_controller.go | 4 | 0 | modified | named |
| pkg/gui/controllers/screen_mode_actions.go | 6 | 3 | modified | named |
| pkg/gui/controllers/search_controller.go | 2 | 0 | modified | named |
| pkg/gui/controllers/search_prompt_controller.go | 16 | 8 | modified | named |
| pkg/gui/controllers/side_window_controller.go | 2 | 2 | modified | named |
| pkg/gui/controllers/snake_controller.go | 20 | 10 | modified | named |
| pkg/gui/controllers/stash_controller.go | 14 | 1 | modified | named |
| pkg/gui/controllers/status_controller.go | 10 | 0 | modified | named |
| pkg/gui/controllers/sub_commits_controller.go | 5 | 0 | modified | named |
| pkg/gui/controllers/submodules_controller.go | 26 | 0 | modified | named |
| pkg/gui/controllers/suggestions_controller.go | 14 | 4 | modified | named |
| pkg/gui/controllers/switch_to_diff_files_controller.go | 2 | 0 | modified | named |
| pkg/gui/controllers/switch_to_focused_main_view_controller.go | 28 | 0 | modified | named |
| pkg/gui/controllers/switch_to_sub_commits_controller.go | 14 | 0 | modified | named |
| pkg/gui/controllers/sync_controller.go | 51 | 22 | modified | named |
| pkg/gui/controllers/tags_controller.go | 27 | 3 | modified | named |
| pkg/gui/controllers/undo_controller.go | 4 | 0 | modified | named |
| pkg/gui/controllers/working_tree_diff_actions.go | 8 | 15 | modified | named |
| pkg/gui/controllers/workspace_reset_controller.go | 15 | 1 | modified | named |
| pkg/gui/controllers/worktrees_controller.go | 10 | 0 | modified | named |
| pkg/gui/extras_panel.go | 9 | 5 | modified | named |
| pkg/gui/global_handlers.go | 7 | 0 | modified | named |
| pkg/gui/gui_common.go | 16 | 0 | modified | named |
| pkg/gui/gui_driver.go | 10 | 0 | modified | absent |
| pkg/gui/gui.go | 67 | 3 | modified | named |
| pkg/gui/keybindings.go | 220 | 83 | modified | named |
| pkg/gui/layout.go | 16 | 0 | modified | named |
| pkg/gui/main_view_render.go | 2 | 0 | modified | named |
| pkg/gui/menu_panel.go | 23 | 5 | modified | named |
| pkg/gui/modes/filtering/filtering.go | 7 | 4 | modified | named |
| pkg/gui/options_map.go | 50 | 40 | modified | named |
| pkg/gui/popup/popup_handler.go | 7 | 1 | modified | named |
| pkg/gui/presentation/commits.go | 3 | 1 | modified | named |
| pkg/gui/services/custom_commands/client.go | 10 | 3 | modified | named |
| pkg/gui/services/custom_commands/handler_creator.go | 2 | 0 | modified | absent |
| pkg/gui/services/custom_commands/keybinding_creator.go | 3 | 0 | modified | named |
| pkg/gui/test_mode.go | 4 | 1 | modified | named |
| pkg/gui/types/common.go | 45 | 6 | modified | named |
| pkg/gui/types/keybindings.go | 23 | 4 | modified | named |
| pkg/gui/types/refresh.go | 8 | 0 | modified | named |
| pkg/gui/views.go | 10 | 58 | modified | named |
| pkg/i18n/english.go | 685 | 15 | modified | named |
| pkg/i18n/i18n.go | 2 | 0 | modified | named |
| pkg/i18n/translations/ja.json | 0 | 2 | modified | absent |
| pkg/i18n/translations/nl.json | 0 | 1 | modified | absent |
| pkg/i18n/translations/pl.json | 0 | 2 | modified | absent |
| pkg/i18n/translations/pt.json | 0 | 2 | modified | absent |
| pkg/i18n/translations/zh-CN.json | 0 | 2 | modified | absent |
| pkg/i18n/translations/zh-TW.json | 0 | 2 | modified | absent |
| pkg/integration/components/assertion_helper.go | 7 | 1 | modified | absent |
| pkg/integration/components/common.go | 14 | 1 | modified | named |
| pkg/integration/components/file_system.go | 21 | 0 | modified | absent |
| pkg/integration/components/menu_driver.go | 7 | 0 | modified | absent |
| pkg/integration/components/runner.go | 5 | 0 | modified | named |
| pkg/integration/components/test_driver.go | 24 | 0 | modified | absent |
| pkg/integration/components/test_test.go | 16 | 0 | modified | absent |
| pkg/integration/components/view_driver.go | 31 | 0 | modified | absent |
| pkg/integration/components/views.go | 5 | 0 | modified | named |
| pkg/integration/tests/branch/checkout_by_name.go | 2 | 0 | modified | absent |
| pkg/integration/tests/branch/delete_multiple.go | 47 | 49 | modified | absent |
| pkg/integration/tests/branch/delete_remote_branch_when_tag_with_same_name_exists.go | 6 | 5 | modified | absent |
| pkg/integration/tests/branch/delete_remote_branch_with_credential_prompt.go | 5 | 4 | modified | absent |
| pkg/integration/tests/branch/delete_remote_branch_with_different_name.go | 8 | 7 | modified | absent |
| pkg/integration/tests/branch/delete_while_filtering.go | 8 | 8 | modified | absent |
| pkg/integration/tests/branch/delete.go | 66 | 73 | modified | absent |
| pkg/integration/tests/branch/detached_head.go | 3 | 0 | modified | absent |
| pkg/integration/tests/branch/merge_fast_forward.go | 39 | 38 | modified | absent |
| pkg/integration/tests/branch/merge_non_fast_forward.go | 26 | 45 | modified | absent |
| pkg/integration/tests/branch/new_branch_autostash.go | 13 | 9 | modified | absent |
| pkg/integration/tests/branch/new_branch_from_remote_tracking_different_name.go | 3 | 0 | modified | absent |
| pkg/integration/tests/branch/new_branch_from_remote_tracking_same_name.go | 3 | 0 | modified | absent |
| pkg/integration/tests/branch/new_branch_with_prefix_using_run_command.go | 2 | 0 | modified | absent |
| pkg/integration/tests/branch/new_branch_with_prefix.go | 2 | 0 | modified | absent |
| pkg/integration/tests/branch/rebase_abort_on_conflict.go | 1 | 0 | modified | absent |
| pkg/integration/tests/branch/rebase_and_drop.go | 2 | 0 | modified | absent |
| pkg/integration/tests/branch/rebase_cancel_on_conflict.go | 1 | 0 | modified | absent |
| pkg/integration/tests/branch/rebase_conflicts_fix_build_errors_with_out_of_date_submodule.go | 10 | 4 | modified | absent |
| pkg/integration/tests/branch/rebase_conflicts_fix_build_errors.go | 2 | 0 | modified | absent |
| pkg/integration/tests/branch/rebase_copied_branch.go | 1 | 0 | modified | absent |
| pkg/integration/tests/branch/rebase_does_not_autosquash.go | 1 | 0 | modified | absent |
| pkg/integration/tests/branch/rebase_from_marked_base.go | 1 | 0 | modified | absent |
| pkg/integration/tests/branch/rebase_onto_base_branch.go | 1 | 0 | modified | absent |
| pkg/integration/tests/branch/rebase_to_upstream.go | 1 | 0 | modified | absent |
| pkg/integration/tests/branch/rebase.go | 2 | 0 | modified | named |
| pkg/integration/tests/branch/rename.go | 10 | 2 | modified | absent |
| pkg/integration/tests/branch/reset_to_duplicate_named_tag.go | 14 | 0 | modified | absent |
| pkg/integration/tests/branch/reset_to_duplicate_named_upstream.go | 18 | 2 | modified | absent |
| pkg/integration/tests/branch/reset.go | 14 | 0 | modified | named |
| pkg/integration/tests/branch/set_upstream.go | 8 | 3 | modified | absent |
| pkg/integration/tests/branch/squash_merge.go | 25 | 25 | modified | absent |
| pkg/integration/tests/branch/unset_upstream.go | 8 | 0 | modified | absent |
| pkg/integration/tests/cherry_pick/cherry_pick_across_worktrees.go | 0 | 79 | deleted | absent |
| pkg/integration/tests/cherry_pick/cherry_pick_commit_that_becomes_empty.go | 15 | 17 | modified | absent |
| pkg/integration/tests/cherry_pick/cherry_pick_conflicts_empty_commit_after_resolving.go | 11 | 16 | modified | absent |
| pkg/integration/tests/cherry_pick/cherry_pick_conflicts.go | 21 | 28 | modified | absent |
| pkg/integration/tests/cherry_pick/cherry_pick_during_rebase.go | 40 | 47 | modified | absent |
| pkg/integration/tests/cherry_pick/cherry_pick_merge.go | 11 | 18 | modified | absent |
| pkg/integration/tests/cherry_pick/cherry_pick_range_after_paste.go | 0 | 106 | deleted | absent |
| pkg/integration/tests/cherry_pick/cherry_pick_range.go | 21 | 29 | modified | absent |
| pkg/integration/tests/cherry_pick/cherry_pick.go | 30 | 46 | modified | named |
| pkg/integration/tests/commit/amend_when_there_are_conflicts_and_amend.go | 3 | 1 | modified | absent |
| pkg/integration/tests/commit/amend_when_there_are_conflicts_and_cancel.go | 3 | 1 | modified | absent |
| pkg/integration/tests/commit/amend_when_there_are_conflicts_and_continue.go | 3 | 1 | modified | absent |
| pkg/integration/tests/commit/checkout.go | 2 | 0 | modified | named |
| pkg/integration/tests/commit/find_base_commit_for_fixup_only_added_lines.go | 26 | 17 | modified | absent |
| pkg/integration/tests/commit/find_base_commit_for_fixup.go | 25 | 16 | modified | absent |
| pkg/integration/tests/commit/history_complex.go | 0 | 59 | deleted | absent |
| pkg/integration/tests/commit/history.go | 0 | 53 | deleted | absent |
| pkg/integration/tests/commit/new_branch.go | 2 | 0 | modified | absent |
| pkg/integration/tests/commit/revert_with_conflict_multiple_commits.go | 3 | 1 | modified | absent |
| pkg/integration/tests/commit/revert_with_conflict_single_commit.go | 3 | 1 | modified | absent |
| pkg/integration/tests/commit/search.go | 7 | 7 | modified | absent |
| pkg/integration/tests/commit/shared.go | 1 | 0 | modified | absent |
| pkg/integration/tests/config/custom_commands_in_per_repo_config.go | 2 | 1 | modified | absent |
| pkg/integration/tests/config/side_panels_in_per_repo_config.go | 3 | 1 | modified | absent |
| pkg/integration/tests/conflicts/conflict_marker_size_not_auto_staged.go | 3 | 1 | modified | absent |
| pkg/integration/tests/conflicts/continue_prompt_dismissed_when_resolved_externally.go | 3 | 1 | modified | absent |
| pkg/integration/tests/conflicts/merge_file_both.go | 3 | 1 | modified | absent |
| pkg/integration/tests/conflicts/merge_file_current.go | 3 | 1 | modified | absent |
| pkg/integration/tests/conflicts/merge_file_incoming.go | 3 | 1 | modified | absent |
| pkg/integration/tests/conflicts/pick_both_hunks_diff3.go | 3 | 1 | modified | absent |
| pkg/integration/tests/conflicts/resolve_externally_started_merge_no_prompt.go | 3 | 1 | modified | absent |
| pkg/integration/tests/conflicts/resolve_externally.go | 3 | 1 | modified | absent |
| pkg/integration/tests/conflicts/resolve_multiple_files.go | 3 | 1 | modified | absent |
| pkg/integration/tests/conflicts/resolve_no_auto_stage.go | 1 | 3 | modified | absent |
| pkg/integration/tests/conflicts/resolve_non_textual_conflicts.go | 3 | 1 | modified | absent |
| pkg/integration/tests/conflicts/resolve_without_trailing_lf.go | 3 | 1 | modified | absent |
| pkg/integration/tests/demo/amend_old_commit.go | 4 | 5 | modified | absent |
| pkg/integration/tests/demo/cherry_pick.go | 18 | 26 | modified | named |
| pkg/integration/tests/demo/commit_and_push.go | 2 | 0 | modified | absent |
| pkg/integration/tests/demo/interactive_rebase.go | 4 | 4 | modified | absent |
| pkg/integration/tests/demo/rebase_onto.go | 10 | 10 | modified | absent |
| pkg/integration/tests/demo/worktree_create_from_branches.go | 1 | 0 | modified | absent |
| pkg/integration/tests/file/discard_all_dir_changes.go | 1 | 0 | modified | absent |
| pkg/integration/tests/file/discard_various_changes_range_select.go | 1 | 0 | modified | absent |
| pkg/integration/tests/file/discard_various_changes.go | 1 | 0 | modified | absent |
| pkg/integration/tests/filter_and_search/filter_menu_as_you_type.go | 1 | 1 | modified | absent |
| pkg/integration/tests/filter_and_search/filter_menu_by_keybinding.go | 1 | 1 | modified | absent |
| pkg/integration/tests/filter_and_search/filter_menu_cancel_filter_with_escape.go | 1 | 1 | modified | absent |
| pkg/integration/tests/filter_and_search/filter_menu_with_no_keybindings.go | 1 | 1 | modified | absent |
| pkg/integration/tests/filter_and_search/filter_menu.go | 1 | 1 | modified | absent |
| pkg/integration/tests/filter_and_search/filter_updates_when_model_changes.go | 1 | 1 | modified | absent |
| pkg/integration/tests/interactive_rebase/advanced_interactive_rebase.go | 1 | 0 | modified | absent |
| pkg/integration/tests/interactive_rebase/amend_commit_with_conflict.go | 3 | 1 | modified | absent |
| pkg/integration/tests/interactive_rebase/interactive_rebase_with_conflict_for_edit_command.go | 1 | 0 | modified | absent |
| pkg/integration/tests/interactive_rebase/rebase_with_commit_that_becomes_empty.go | 12 | 1 | modified | absent |
| pkg/integration/tests/interactive_rebase/revert_multiple_commits_in_interactive_rebase.go | 3 | 1 | modified | absent |
| pkg/integration/tests/interactive_rebase/revert_single_commit_in_interactive_rebase.go | 3 | 1 | modified | absent |
| pkg/integration/tests/interactive_rebase/swap_in_rebase_with_conflict_and_edit.go | 3 | 1 | modified | absent |
| pkg/integration/tests/interactive_rebase/swap_in_rebase_with_conflict.go | 3 | 1 | modified | absent |
| pkg/integration/tests/interactive_rebase/swap_with_conflict.go | 3 | 1 | modified | absent |
| pkg/integration/tests/main_view/apply_custom_patch_in_reverse_with_conflict.go | 1 | 0 | modified | absent |
| pkg/integration/tests/main_view/discard_diff_lines.go | 1 | 1 | modified | absent |
| pkg/integration/tests/main_view/jump_to_a_file_of_the_diff.go | 6 | 6 | modified | absent |
| pkg/integration/tests/main_view/jump_to_a_file_only_over_a_diff.go | 15 | 14 | modified | absent |
| pkg/integration/tests/main_view/move_patch_from_added_file_to_earlier_commit.go | 1 | 0 | modified | absent |
| pkg/integration/tests/main_view/move_patch_from_added_file_to_index_with_conflict.go | 1 | 0 | modified | absent |
| pkg/integration/tests/main_view/move_patch_to_index_with_conflict.go | 1 | 0 | modified | absent |
| pkg/integration/tests/main_view/move_patch_to_index_with_modified_file.go | 10 | 5 | modified | absent |
| pkg/integration/tests/main_view/open_pull_request_only_over_a_commits_diff.go | 7 | 10 | modified | absent |
| pkg/integration/tests/main_view/selection_command_tooltips_follow_the_diff.go | 22 | 14 | modified | absent |
| pkg/integration/tests/main_view/selection_commands_only_where_they_apply.go | 13 | 6 | modified | absent |
| pkg/integration/tests/misc/direnv_approves_envrc.go | 1 | 0 | modified | absent |
| pkg/integration/tests/misc/direnv_loaded_on_repo_switch.go | 1 | 0 | modified | absent |
| pkg/integration/tests/misc/direnv_unloads_on_blocked_envrc.go | 1 | 0 | modified | absent |
| pkg/integration/tests/misc/filter_recent_repos.go | 1 | 0 | modified | absent |
| pkg/integration/tests/misc/initial_open.go | 2 | 1 | modified | absent |
| pkg/integration/tests/misc/recent_repos_column_widths.go | 1 | 0 | modified | absent |
| pkg/integration/tests/misc/recent_repos_with_long_names.go | 1 | 0 | modified | absent |
| pkg/integration/tests/reflog/cherry_pick.go | 8 | 15 | modified | named |
| pkg/integration/tests/reflog/reset.go | 11 | 0 | modified | named |
| pkg/integration/tests/stash/create_branch.go | 2 | 0 | modified | absent |
| pkg/integration/tests/submodule/enter_from_dotfile_bare_repo.go | 2 | 0 | modified | absent |
| pkg/integration/tests/submodule/enter_nested.go | 2 | 0 | modified | absent |
| pkg/integration/tests/submodule/enter.go | 2 | 0 | modified | absent |
| pkg/integration/tests/submodule/remove_nested.go | 2 | 0 | modified | absent |
| pkg/integration/tests/submodule/reset.go | 3 | 0 | modified | named |
| pkg/integration/tests/sync/fast_forward_behind_branch_checked_out.go | 17 | 2 | modified | absent |
| pkg/integration/tests/sync/fast_forward_behind_branch_without_reflogs.go | 13 | 1 | modified | absent |
| pkg/integration/tests/sync/fast_forward_behind_branch.go | 14 | 1 | modified | absent |
| pkg/integration/tests/sync/fast_forward_branch_being_rebased_in_worktree.go | 25 | 35 | modified | absent |
| pkg/integration/tests/sync/fast_forward_branches_in_worktrees_with_one_failing.go | 24 | 32 | modified | absent |
| pkg/integration/tests/sync/fast_forward_rewritten_branch_checked_out.go | 14 | 12 | modified | absent |
| pkg/integration/tests/sync/fast_forward_rewritten_branch_with_local_commits.go | 12 | 9 | modified | absent |
| pkg/integration/tests/sync/fast_forward_rewritten_branch_with_stale_submodule.go | 13 | 6 | modified | absent |
| pkg/integration/tests/sync/fast_forward_rewritten_branch_with_uncommitted_changes.go | 10 | 4 | modified | absent |
| pkg/integration/tests/sync/fast_forward_rewritten_branch_without_reflogs.go | 12 | 12 | modified | absent |
| pkg/integration/tests/sync/fast_forward_rewritten_branch.go | 19 | 18 | modified | absent |
| pkg/integration/tests/sync/fast_forward_rewritten_stack_of_branches_with_local_commits.go | 20 | 13 | modified | absent |
| pkg/integration/tests/sync/fast_forward_rewritten_stack_of_branches.go | 15 | 8 | modified | absent |
| pkg/integration/tests/sync/fetch_and_auto_forward_branches_all_branches_checked_out_in_other_worktree.go | 2 | 0 | modified | absent |
| pkg/integration/tests/sync/fetch_and_auto_forward_branches_none.go | 2 | 3 | modified | absent |
| pkg/integration/tests/sync/fetch_and_auto_forward_branches_worktree_added_after_startup.go | 1 | 0 | modified | absent |
| pkg/integration/tests/sync/fetch_when_sorted_by_date.go | 13 | 3 | modified | absent |
| pkg/integration/tests/sync/force_push_multiple_matching.go | 10 | 6 | modified | absent |
| pkg/integration/tests/sync/force_push_multiple_upstream.go | 9 | 5 | modified | absent |
| pkg/integration/tests/sync/force_push_remote_branch_not_stored_locally.go | 31 | 22 | modified | absent |
| pkg/integration/tests/sync/force_push_triangular.go | 8 | 4 | modified | absent |
| pkg/integration/tests/sync/force_push.go | 8 | 5 | modified | absent |
| pkg/integration/tests/sync/pull_and_set_upstream.go | 22 | 7 | modified | absent |
| pkg/integration/tests/sync/pull_merge_conflict.go | 8 | 7 | modified | absent |
| pkg/integration/tests/sync/pull_merge.go | 5 | 6 | modified | absent |
| pkg/integration/tests/sync/pull_rebase_conflict.go | 7 | 6 | modified | absent |
| pkg/integration/tests/sync/pull_rebase_interactive_conflict_drop.go | 11 | 6 | modified | absent |
| pkg/integration/tests/sync/pull_rebase_interactive_conflict.go | 11 | 6 | modified | absent |
| pkg/integration/tests/sync/pull_rebase.go | 4 | 5 | modified | absent |
| pkg/integration/tests/sync/pull_stacked_branches_checked_out_in_other_worktree.go | 40 | 22 | modified | absent |
| pkg/integration/tests/sync/pull_stacked_branches_only_current.go | 19 | 13 | modified | absent |
| pkg/integration/tests/sync/pull_stacked_branches_update_fails.go | 35 | 18 | modified | absent |
| pkg/integration/tests/sync/pull_stacked_branches_with_local_commits.go | 14 | 21 | modified | absent |
| pkg/integration/tests/sync/pull_stacked_branches.go | 28 | 31 | modified | absent |
| pkg/integration/tests/sync/pull.go | 18 | 2 | modified | absent |
| pkg/integration/tests/sync/push_and_auto_set_upstream.go | 21 | 3 | modified | absent |
| pkg/integration/tests/sync/push_and_set_upstream.go | 22 | 6 | modified | absent |
| pkg/integration/tests/sync/push_follow_tags.go | 6 | 1 | modified | absent |
| pkg/integration/tests/sync/push_stacked_branches_current_without_upstream.go | 30 | 31 | modified | absent |
| pkg/integration/tests/sync/push_stacked_branches_no_force.go | 19 | 17 | modified | absent |
| pkg/integration/tests/sync/push_stacked_branches_only_current.go | 25 | 19 | modified | absent |
| pkg/integration/tests/sync/push_stacked_branches.go | 21 | 52 | modified | absent |
| pkg/integration/tests/sync/push_with_credential_prompt.go | 6 | 1 | modified | absent |
| pkg/integration/tests/sync/push.go | 5 | 0 | modified | absent |
| pkg/integration/tests/sync/rename_branch_and_pull.go | 7 | 0 | modified | absent |
| pkg/integration/tests/tag/reset_to_duplicate_named_branch.go | 10 | 0 | modified | absent |
| pkg/integration/tests/tag/reset.go | 11 | 1 | modified | named |
| pkg/integration/tests/ui/keybinding_suggestions_dont_crash_on_disabled_bindings.go | 1 | 1 | modified | absent |
| pkg/integration/tests/ui/keybinding_suggestions_when_switching_repos.go | 3 | 2 | modified | absent |
| pkg/integration/tests/ui/mode_specific_keybinding_suggestions.go | 2 | 15 | modified | absent |
| pkg/integration/tests/ui/reload_side_panels.go | 2 | 2 | modified | absent |
| pkg/integration/tests/ui/switch_repo_moves_the_selection.go | 1 | 0 | modified | absent |
| pkg/integration/tests/ui/switch_tab_from_menu.go | 1 | 1 | modified | absent |
| pkg/integration/tests/worktree/add_for_existing_branch.go | 1 | 0 | modified | absent |
| pkg/integration/tests/worktree/add_from_branch_detached.go | 1 | 0 | modified | absent |
| pkg/integration/tests/worktree/add_from_branch.go | 2 | 0 | modified | absent |
| pkg/integration/tests/worktree/add_from_commit.go | 1 | 0 | modified | absent |
| pkg/integration/tests/worktree/add_from_remote_branch.go | 1 | 0 | modified | absent |
| pkg/integration/tests/worktree/add_from_stash.go | 1 | 0 | modified | absent |
| pkg/integration/tests/worktree/add_from_tag.go | 1 | 0 | modified | absent |
| pkg/integration/tests/worktree/associate_branch_bisect.go | 2 | 0 | modified | absent |
| pkg/integration/tests/worktree/associate_branch_rebase.go | 2 | 0 | modified | absent |
| pkg/integration/tests/worktree/bare_repo.go | 3 | 0 | modified | absent |
| pkg/integration/tests/worktree/crud.go | 4 | 0 | modified | absent |
| pkg/integration/tests/worktree/detach_worktree_from_branch.go | 11 | 16 | modified | absent |
| pkg/integration/tests/worktree/exclude_file_in_worktree.go | 2 | 0 | modified | absent |
| pkg/integration/tests/worktree/fast_forward_worktree_branch_should_not_pollute_current_worktree.go | 27 | 29 | modified | absent |
| pkg/integration/tests/worktree/fast_forward_worktree_branch.go | 30 | 23 | modified | absent |
| pkg/integration/tests/worktree/new_worktree_picker_remote.go | 1 | 0 | modified | absent |
| pkg/integration/tests/worktree/new_worktree_picker.go | 1 | 0 | modified | absent |
| pkg/integration/tests/worktree/remove_worktree_and_delete_local_and_remote_branch.go | 20 | 34 | modified | absent |
| pkg/integration/tests/worktree/remove_worktree_from_branch.go | 13 | 21 | modified | absent |
| pkg/integration/tests/worktree/reset_window_tabs.go | 3 | 0 | modified | absent |
| pkg/integration/tests/worktree/worktree_in_repo.go | 4 | 0 | modified | absent |
| pkg/integration/types/types.go | 9 | 0 | modified | absent |
| pkg/theme/gocui_test.go | 5 | 0 | modified | absent |
| test/default_test_config/config.yml | 146 | 0 | modified | named |

## Out of scope observations

No feature redesign or cross-area correctness claim. The probe's integer-limit incompatibility is a future merge requirement, not a current HEAD build failure. No TortoiseGit behavior question was needed: this task reviews merge structure and existing agreed policies.

## Commands run and outcomes

All repository commands were read-only; commands below summarize repeated path-specific invocations.

- Read worker request, common-review-rules.md, AGENTS.md, Tracker.md, Design.md, Guide.md, and all MergeGuide.md.
- git status --short; git rev-parse HEAD refs/remotes/upstream/master; git remote -v: recorded hashes/remotes and pre-existing untracked paths.
- git diff --numstat BASE HEAD, with --diff-filter=M/A/D and explicit excluded paths: inventory and counts above. No renamed paths or deleted production sources found; four deleted scenarios.
- git diff BASE HEAD -- <production, integration components, existing scenarios, *_test.go, docs and i18n paths>; zero-context diffs and rg/head/sed inspection: assessed hooks, metadata, comments, test/assertion changes. One rg against a nonexistent fork_commit_drafts*.go glob and a later sed of a proposed helpers/fork_commit_draft.go reported no such file; these were corrected by inspecting actual paths, and no finding assumes those files exist. Broad outputs that were truncated were followed with targeted reads for the cited findings.
- git log BASE..HEAD -- pkg/i18n/translations; git show 47d30f89f: confirmed translation removals belong to palette section change.
- git clone --shared /srv/devshare/projects/lazygitk /tmp/probe-merge-15: success.
- git -C /tmp/probe-merge-15 fetch /srv/devshare/projects/lazygitk refs/remotes/upstream/master:refs/remotes/upstream/master: success, local source only.
- git -C /tmp/probe-merge-15 -c user.name=Review -c user.email=review@example.invalid merge --no-commit --no-ff upstream/master: expected exit 1, 11 conflicted files.
- git -C /tmp/probe-merge-15 diff --name-only --diff-filter=U; git diff --cc on two source conflicts; git diff BASE upstream/master and probe diff HEAD -- relevant paths: recorded conflict and clean-merge contracts.
- git -C /tmp/probe-merge-15 merge --abort: success; probe git status --short empty. Source git status --short unchanged.
- Tests/build/generation/lint: not run, as explicitly required. No automated correctness result is claimed.
- Artifact written by quoted heredoc to /tmp/lazygitk-review-run-1793910387/15-worker-result.md.

## Worker handoff and next step

Work performed: completed the scoped read-only footprint, merge probe, guide, upstream-test/harness and i18n audit. Files changed in repository: none. Only requested handoff artifact written, plus explicitly authorized throwaway clone/merge operations. No commits or PRs.

Recommended payoff order:

1. Correct MergeGuide counts/remotes, false metadata-only entries, translation statement, and omitted test-replacement contracts. This directly prevents incorrect merge resolutions.
2. For the next actual upstream merge, adapt the integer commit-limit contract across fork_commit_scope.go, ancestor filtering and tests; reapply identities to upstream's bisect item builders and inspect the seven clean-merge neighborhoods above. Regenerate outputs only after source integration. Do not treat the two source conflicts as the complete work.
3. Consolidate English additions into marked fields and fork initializer; consider restoring obsolete translation entries (the runtime loader accepts unknown JSON fields).
4. Move wholly new draft/palette/hint/usage/settlement methods into existing fork modules, keeping short commented lifecycle/dispatch hooks. Then consider optional harness method extraction.
5. Map retained stack-push coverage and restore unaffected merge-topology assertions in a focused test change; document intentional scenario removals/replacements.

Remaining problems: findings F1-F5 are review recommendations and remain unchanged. Actual merge resolution, builds and tests are intentionally unperformed. No active probe merge remains. No new design phase was started.

