# Review 03: configuration, defaults, diff renderers, startup notices, build

## Summary

Completed the requested read-only review against BASE `4e1eebe278790f257a875da6633234391a5a85b3`, following the common rules and fork documents. No implementation or repository edits were permitted or performed.
The fork largely keeps policy in additive modules: defaults overlay upstream's constructor, theme precedence preserves explicit fields, and delta command values are fixed strings rather than interpolated user input.
The most important issue is that `buildSource=forkDist` does not protect manually requested updates; the replacement path remains reachable for an unversioned distribution build.
Startup notices can obscure one another, and an upstream renderer test now depends on the review machine's PATH.
The nine main upstream Go integration files in this area total 185 insertions / 25 deletions (gui.go includes other review areas). English strings add another 685 / 15 for the whole fork; six translation JSON files lose eleven entries.
One targeted unit test passed; the scoped diff whitespace check passed. No integration tests, generation, distribution builds, or updater operations ran.

## Findings

### F1. Distribution build marker does not prevent a manually requested upstream replacement
- Category: bug
- Severity: high
- Confidence: CLEAR BUG
- Location: scripts/build_dist.sh:41,45; scripts/build_dist.ps1:49,59; pkg/updates/updates.go:142,166,238,311; pkg/gui/controllers/helpers/update_helper.go:42,85; docs-master/fork/MergeGuide.md:573
- Problem: Both scripts and the merge guide describe `forkDist` as disabling the self-updater. The only build-source guard is inside `skipUpdateCheck`, and `CheckForNewUpdate` bypasses that function when `userRequested` is true. The foreground action passes true, offers the ordinary install prompt, then calls `Update`, which has no build-source guard and renames the upstream binary over the current executable.
- Scenario / evidence: Build on Linux from a source archive without Git metadata: the shell script deliberately falls back to `version=unversioned`, with `buildSource=forkDist`. Invoke the manual check (`t a u`, also available in the palette). `currentVersion` returns `unversioned`; `majorVersionDiffers` explicitly permits that value. With a reachable upstream release asset and writable executable location, accepting the update replaces lazygitk with upstream lazygit. This was traced through status controller -> foreground update helper -> CheckForNewUpdate(..., true) -> showUpdatePrompt -> Update -> downloadAndInstall -> os.Rename. No network/update operation was invoked. Tagged checkout builds currently have an additional version-format problem (the scripts retain a leading `v`, while currentVersion adds another), but that accidental failure does not protect archive/unversioned builds.
- Suggested fix: Enforce fork build eligibility independently of background scheduling, including at the install boundary. Keep manual checking informational for fork builds or reject unsupported installation explicitly. Add a focused mocked updater regression for `forkDist` plus `userRequested=true`, including `unversioned`; do not exercise real downloading or replacement.

### F2. Startup notices expire while hidden behind newer notices
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/fork_ignored_key_config.go:22,31; pkg/gui/layout.go:337,344,348; pkg/gui/fork_diff_renderer_notice.go:11; pkg/gui/status/status_manager.go:65,85,150
- Problem: The new notice producers emit multiple independent toasts immediately. The status manager prepends each toast, displays only the first entry, and starts each entry's two-second expiration on insertion. It does not give every queued message two seconds of visible time. Ignored-key notices are marked shown before the user sees them.
- Scenario / evidence: Start with delta absent and key settings in global and per-repo configuration. The key notices are inserted together; the later delta notice can cover them. All timers run concurrently, so covered notices disappear after approximately the same two seconds, having been visible for at most a tiny scheduling interval. With delta present, the same issue affects multiple key-config files. The session deduplication then prevents these notices from being shown again. This contradicts the documented notice naming each affected file. Confirmed by tracing the producers and the status manager implementation; GUI notice unit tests cover formatting and notice keys, not delivery/visibility.
- Suggested fix: Aggregate startup/config notices into one readable notification, or arrange sequential delivery with expiration measured from actual display. Preserve session deduplication only after reliable delivery. Add a focused delivery test with two ignored-file notices plus delta fallback.

### F3. Empty-renderer name test fails on machines that have delta installed
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/config/diff_renderer_config_manager_test.go:218; pkg/config/fork_diff_renderers.go:143; pkg/config/fork_diff_renderers_test.go:16
- Problem: `TestCurrentDiffRendererNameWithoutDiffRenderers` now unconditionally expects `default`, but does not stub `forkLookPath`. The fork intentionally chooses `delta` for an empty renderer list when delta is on PATH.
- Scenario / evidence: The manager resolves an empty `UserConfig.Git.DiffRenderers` through `forkEffectiveDiffRenderers`, calls the real exec.LookPath, and names the first renderer `delta` when installed. Therefore a developer using the recommended default renderer gets a failing unit suite. The targeted command passed in this environment (`ok .../pkg/config 0.015s`), consistent with the fallback branch here; the delta-present branch is established directly by the implementation and the existing deterministic `TestForkDefaultDiffRenderersWhenDeltaIsFound`. This is an environment dependency, not a failure reproduced on this host.
- Suggested fix: Call the existing `setForkLookPath(t, false)` in the changed upstream test, or make this a two-case test with deterministic lookup outcomes.

### F4. Generated configuration reference still instructs users to change ignored key settings
- Category: docs
- Severity: low
- Confidence: n/a
- Location: pkg/config/user_config.go:35,52; docs-master/Config.md:672,695,1200; schema-master/config.json:2191,4258
- Problem: The new fixed-key documentation correctly says key settings are ignored, but the configuration field descriptions still advertise `commandBindings` as a way to bind/remove keys and explicitly instruct users to copy fields they want to override. The same misleading descriptions propagate into the schema and generated reference. They describe the internal harness capability as a supported user feature.
- Scenario / evidence: A user following the `commandBindings` example in the default reference or editor completion changes a key path, receives no binding change, and must discover the contradictory fixed-key paragraph much later in Config.md. The schema matches the struct; the defect is the source descriptions, not generation drift.
- Suggested fix: Correct struct field comments to identify ignored legacy/internal settings, then regenerate docs/schema in a separately authorized implementation step. Keep details for the internal honour-config switch out of user instructions.

### F5. Diff-renderer documentation teaches the removed cycling key
- Category: docs
- Severity: low
- Confidence: n/a
- Location: docs-master/Custom_DiffRenderers.md:5,14; pkg/config/fork_diff_renderers.go:11; pkg/gui/keymap/default_paths.go:159
- Problem: The fork-specific introduction says `|` cycles renderers, and the following general instructions repeat that key. The fixed fork keymap uses `t d n` / `t d p`, as Guide.md correctly documents. The new renderer module's leading comment also retains the stale key.
- Scenario / evidence: The user follows the fork's renderer installation/configuration page and presses `|`; it does not invoke the documented operation under the normal fixed-key profile.
- Suggested fix: Update the fork notice, the cycling instructions, and the fork-owned comment to the actual paths; preferably link to the guide where keys would otherwise be duplicated.

### F6. User guide implies repository-root dot configuration is supported
- Category: docs
- Severity: low
- Confidence: n/a
- Location: docs-master/fork/Guide.md:513; pkg/gui/gui.go:523,534; docs-master/Config.md:19
- Problem: Guide.md says `.git/lazygitk.yml` or `.lazygitk.yml` in repositories. The implementation checks the git directory plus dot files in parent directories only. It deliberately excludes the repository root, with the upstream trust-prompt TODO still present. Config.md accurately says parent directories.
- Scenario / evidence: Put `.lazygitk.yml` in the repository root as the guide suggests. `getPerRepoConfigFiles` initializes its walk with `filepath.Dir(RepoPath())`, so that file is never considered.
- Suggested fix: Align Guide.md with the current parent-directory behavior. Supporting tracked root config would require the unresolved upstream trust policy and is outside this review; do not implement that implicitly.

## Merge-ease notes

- `pkg/config/app_config.go` (+33/-12): directory replacement is small, commented, and routes through one fork constant; preserving the original loader loop with capture/discard hooks is reasonable. The test-only SaveGlobalUserConfig replacement is already mostly in a fork module, but its new reload/marshal integration is absent from MergeGuide.md's Configuration section. Document that seam and its baseline/provenance requirement. The state-directory comment does not use the `Fork:` marker, though the adjacent path replacement is understandable.
- `pkg/config/user_config.go` (+28/-4): wrapping the upstream constructor rather than copying it is the right shape for future fields. New upstream fields retain upstream defaults automatically; new automation options still need a deliberate fork-policy review. Struct additions are limited. The constructor hook explains replacement but does not literally use the `Fork:` prefix. Correct the user-facing comments called out in F4.
- `pkg/config/user_config_validation.go` (+7/-1): two short validator hooks and a custom-menu error update; reasonable integration surface. The hooks have no fork marker, though the merge guide identifies them. Key identity/binding validation internals are another review area.
- `pkg/config/theme_config.go` (+5/-1): one well-commented merge-layer insertion, with explicit precedence preserved and new theme fields continuing to receive upstream defaults. Corresponding upstream test changes keep upstream background-color assertions against themeDefaults directly.
- `pkg/config/diff_renderer_config_manager.go` (+25/-4): repeated renderer-list reads are all routed through the fork resolver; the command builder itself is additive. Good boundary, documented in the merge guide. The old empty-list nil branch becomes unreachable but is harmless compatibility scaffolding. New upstream reads must continue using the effective list.
- `pkg/gui/gui.go` (+67/-3 overall): this area's hooks are small: per-repo filenames, notice call, delta-config callback. Most edits belong to other areas. The `.lazygit.yml` TODO retains an upstream filename; if ever implemented, use the fork constant as well as settling trust behavior. Focus-triggered config reload currently does not call showForkIgnoredKeyConfigNotices: adding key settings during a session silently ignores them until a restart/repo reload. The current docs promise startup notices, so this is an improvement opportunity rather than a separate claimed contract violation.
- `pkg/gui/layout.go` (+16/-0 overall): startup hooks are additive and commented, but their interaction exposes F2. The focused-frame lines belong to another area.
- `pkg/gui/main_view_render.go` (+2/-0): minimal TrueColor render-value hook. No broader rendering review performed.
- `pkg/i18n/i18n.go` (+2/-0): one explicit post-translation reset keeps the fork's intro without rewriting translated intros. Good minimal seam.
- `pkg/i18n/english.go` (+685/-15 overall): intro text correctly names the fork and preserves donation attribution; most other strings are outside this area. Large monolithic English string additions are still a merge hotspot, but moving them requires considering the translation generator and is not proposed here.
- `pkg/i18n/translations/{ja,nl,pl,pt,zh-CN,zh-TW}.json` (eleven removed lines): only obsolete KeybindingsMenuSectionLocal/Global entries were removed; translated intro strings were not rewritten. Runtime uses ordinary json.Unmarshal, which tolerates obsolete keys, so this churn was not necessary for runtime loading. MergeGuide.md:40 says translations are untouched, which is factually false. Prefer retaining obsolete upstream JSON keys, or accurately document the cleanup and any generator requirement. No such requirement was found in the inspected runtime/tests.
- `justfile` (+6/-2): one additive dist recipe plus test timeout changes, all small; shell distribution invocation remains Unix-oriented, with the separate PowerShell entry point documented for Windows.
- `.gitignore` (+1/-0): isolated `/dist/` entry.
- `scripts/build_dist.sh` / `.ps1`: fully additive and easy to maintain. Temporary build caches are cleaned with EXIT/finally; builds disable CGO and use vendored dependencies. The scripts were reviewed statically and not executed because they write dist/. Review F1 before relying on updater isolation. The raw `git describe` version also includes a leading v unlike currentVersion's expectation; normalize metadata deliberately when fixing that path.
- `docs-master/Config.md` (+98/-112), `Custom_DiffRenderers.md` (+5/-0), `Custom_Command_Keybindings.md` (+21/-0): changed intros are mostly localized, but generated field descriptions preserve obsolete user guidance. The custom-command intro says the rest of the page applies unchanged even though its opening still teaches `e` on Status and `q` from the old menu. The statement that no usage counters are stored by the "preparatory identity change" is historical language and should be reconciled with current Guide.md, which describes usage tracking.
- `schema-master/config.json`: inspected relevant automation, graph, command-binding, identity and renderer definitions without regeneration. They match the corresponding config structs/defaults in this area; no separate structural mismatch found. Private load metadata and the private resolver callback are not exposed.

## Out of scope observations

The updater implementation and toast status manager were read only far enough to verify the build marker and notice integration contracts above. Fixes may need those adjacent modules; coordinate ownership before editing them.
No review of semantic key projection, rendering/OSC behavior, Git operation policy, or command-usage storage internals was performed.
Pre-existing untracked `HANDOFF.md`, `reviews/`, and `test/.viminfo` were present at both the start and end; they were left untouched.

## Commands run and outcomes

- Read `/tmp/lazygitk-review-run-1793910387/03-worker-request.md` and `common-review-rules.md`, repository AGENTS.md, Tracker.md, Design.md, Guide.md and relevant MergeGuide.md sections using cat/sed/rg. No new design phase started.
- `git status --short` before and after: same three pre-existing untracked paths, no tracked modifications.
- `git diff 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <scoped paths>`: inspected integration diffs, relevant docs and all translation JSON changes. `git diff --stat` and `--numstat` established footprint. The initial stat inspection also included broader doc/schema listings for orientation; substantive review stayed within scope.
- `git log --oneline 4e1eebe278790f257a875da6633234391a5a85b3..HEAD -- pkg/config/app_config.go pkg/config/fork_diff_renderers.go pkg/i18n/translations scripts/build_dist.sh`: inspected rationale/history of directory split, ignored keys, palette/delta and build changes.
- cat/sed/rg reads of the scoped fork modules and unit tests, config loader/default/validation/theme/renderer manager, schema entries, GUI startup/reload/pager command paths, Git config command builder/cache, updater and toast delivery paths. A few exploratory reads/searches reported nonexistent guessed paths (for example git_config/git_config.go and status_manager_test.go); subsequent file discovery located the actual sources. These diagnostics did not affect conclusions.
- `git describe --tags --always --dirty`: `v0.66.2-277-ga9d765542`, used to examine script metadata formatting.
- `go test ./pkg/config/ -run '^TestCurrentDiffRendererNameWithoutDiffRenderers$' -short -count=1`: PASS, `ok github.com/jesseduffield/lazygit/pkg/config 0.015s`. This was the sole targeted test run, authorized to investigate the concrete PATH-dependent test suspicion. It does not validate the delta-present host case.
- `git diff --check 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <scoped source/config/gui/i18n/build/doc/schema paths>`: PASS, exit 0, no whitespace errors.
- No full test suite, integration tests, code generation, formatting writes, build scripts, network/updater calls, commits, or PRs.

## Work performed, files changed, remaining problems, and next step

Performed detailed static investigation and one narrowly targeted unit check. Implementation and iteration were intentionally limited to review because the worker request forbids modifying repository files. The only file created is this result artifact: `/tmp/lazygitk-review-run-1793910387/03-worker-result.md`.
All six findings remain unfixed. No unresolved design choice was silently implemented. Native Windows command execution and distribution builds were not exercised; quoting was inspected through the Windows shell builder. Delta command construction uses only fixed flags/palette strings, and Git config values are used only to suppress flags, so no user-value-to-shell injection path was found in this area. Ignoring keys before typed decoding/validation was outside the requested skim of the key-discard implementation.
Recommended next step: assign a narrowly scoped implementation pass for F1, then F2 and F3, with mocked/non-destructive regressions. Correct documentation and merge-guide discrepancies in the same relevant fixup commits, using the requested repository workflow. Do not run a real self-update to reproduce F1. Preserve the already agreed configuration, theme and fixed-key design.
