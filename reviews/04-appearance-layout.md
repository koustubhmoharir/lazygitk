# Review 04: appearance-layout

## Summary

Read-only review against BASE `4e1eebe278790f257a875da6633234391a5a85b3`.
Most behavior lives in fork-owned files with small, commented upstream hooks.
The 12 upstream source files examined have 179 additions and 77 deletions in aggregate; those totals include unrelated fork changes in shared files such as gui.go and views.go.
The principal findings are startup input consumption and hashing the complete loaded history on every graph render. The clipboard replacement is missing from the merge guide.
No repository files were changed, no implementation or design phase was started, and no tests were run, as requested by the worker-specific constraints.

## Findings

### F1. Startup true-colour probing discards user input

- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gocui/fork_true_color.go:98; pkg/gocui/fork_true_color.go:85; pkg/gocui/tcell_driver.go:69
- Problem: The probe consumes arbitrary TTY bytes into a private buffer and returns only a capability boolean. It neither separates user input from terminal replies nor forwards non-reply bytes to tcell. Bytes following the device-attributes reply within the same read are also discarded.
- Scenario / evidence: With COLORTERM unset and a probe-eligible terminal, type a navigation key while the terminal answers slowly, or during the 300 ms wait on a non-answering terminal. readUntilDeviceAttributes appends the key to input; probeTrueColor discards that buffer after matching replies or timing out. A read containing `ESC[?62c` followed by a typed key loses that key as well. This is established by the complete read-to-boolean code path, not a terminal reproduction. Upstream already flushes queued input when starting its TTY; the fork adds another active read/discard interval before that startup.
- Suggested fix: Preserve non-protocol bytes and replay them after the final underlying TTY Start, which itself flushes input. Test interleaved keys and keys trailing the DA reply, including the timeout case. A replay buffer that is discarded by the subsequent Start would not fix this.

### F2. Graph cache hits still hash the entire loaded history

- Category: code-quality
- Severity: medium
- Confidence: n/a
- Location: pkg/gui/presentation/commits.go:143; pkg/gui/presentation/commits.go:323; pkg/gui/presentation/fork_pipe_set_key.go:20
- Problem: commitListKey walks every loaded commit and hashes every hash before the cache lookup. This happens on every graph render, including selection-only renders for a small visible slice. The entire operation holds the presentation package's shared mutex.
- Scenario / evidence: GetCommitListDisplayStrings takes startIdx/endIdx, but the ordinary graph path passes commits[rebaseOffset:] to loadPipesets, which computes commitListKey before checking pipeSetCache. Consequently, once an unlimited/large history is loaded, rendering tens of visible rows still hashes the full history. Divergence rendering likewise hashes both complete sections. The additional work is proportional to approximately 41 bytes per SHA-1 commit or 65 per SHA-256 commit, plus two digest writes per commit, on every call. No latency or allocation benchmark was run; user-visible severity at particular history sizes remains unmeasured.
- Suggested fix: Compute the full-list identity when publishing a refreshed immutable commit list, or otherwise memoize it using an explicit list-generation identity, and pass that identity to presentation. Preserve the existing same-head/same-length collision protection and F5 invalidation; do not revert to the upstream key.

### F3. Merge guide omits the OSC 52 clipboard replacement hooks

- Category: docs
- Severity: low
- Confidence: n/a
- Location: docs-master/fork/MergeGuide.md: integration points / Appearance and Other; pkg/commands/oscommands/os.go:263; pkg/gui/gui.go:911
- Problem: The merge guide mentions clipboard-writer modules and test helpers, but does not document the replacement of clipboard.WriteAll in OSCommand.CopyToClipboard or the required injection from NewGui. These are upstream integration points that must survive merges.
- Scenario / evidence: oscommands.CopyToClipboard now fails without a configured command or injected writer. NewGui supplies writeClipboardOSC52 via SetClipboardWriter. Losing that injection causes every default copy action to return "no terminal clipboard writer is configured"; restoring the upstream fallback loses the terminal clipboard behavior over SSH/containers. No integration-point entry describes either requirement.
- Suggested fix: Add an entry identifying the os.go replacement, its explicit copy-command precedence, the NewGui injection, and the GUI writer's dependency on the active gocui screen.

## Merge-ease notes

Full BASE-to-HEAD source counts below include other fork work where files are shared. Appearance-specific hooks are substantially smaller than the totals in gui.go, views.go, layout.go and context.go.

| Upstream file | Added / removed | Assessment |
| --- | ---: | --- |
| pkg/gocui/tcell_driver.go | 3 / 1 | Minimal commented constructor-option hook; keep probe implementation in fork_true_color.go. |
| pkg/gocui/view.go | 23 / 6 | Two view fields, search guard, selected-line branch and content-line argument propagation. The signature affects one definition and three production calls, all in this file; no broad cross-package signature churn. The extra argument is useful for wrapped logical-line selection. Hooks and signature requirement are documented. |
| pkg/config/theme_config.go | 5 / 1 | Minimal commented palette insertion in merge precedence. |
| pkg/gui/views.go | 10 / 58 | Border-switch replacement is extracted cleanly into fork_focus_frame.go. Commits cursor marker setup is small. Jump-label changes belong to the separate key-routing area. |
| pkg/gui/context/setup.go | 5 / 1 | Small result-variable conversion plus one helper call to configure list views. |
| pkg/gui/presentation/commits.go | 3 / 1 | Minimal digest hook. Existing comments above and below the cache lookup still describe the old first-hash/count key and filtered-list restriction; update them when addressing F2. |
| pkg/gui/controllers/screen_mode_actions.go | 6 / 3 | Small delegation to fork area helpers; Next and Prev retain upstream rerendering. |
| pkg/gui/controllers/helpers/window_arrangement_helper.go | 9 / 2 | Adds the area argument and changes the two relevant predicates; clear Fork comments. |
| pkg/gui/context.go | 14 / 0 | Appearance adds two commented Activate hooks. Prefix invalidation and binding rebuild changes are outside this review area. |
| pkg/gui/layout.go | 16 / 0 | Initial-frame hook is small; other added lines concern startup notices. |
| pkg/gui/gui.go | 67 / 3 | Appearance uses a state field/reset, title update, clipboard injection and palette update. Each is a small commented seam; broader additions are not appearance-specific. |
| pkg/commands/oscommands/os.go | 18 / 1 | Appropriate injected writer avoids importing GUI code into commands. Explicit copy command still wins. Merge-guide coverage is missing (F3). |

Fork-only theme, focus-frame, screen-area, selected-line, title and clipboard modules avoid scattering substantial implementations into upstream files. Mock title/clipboard helpers are additive and simulation-specific.

## Focused investigation and unresolved questions

- **Timeout / hangs:** The nominal probe wait is 300 ms. On timeout it ignores tty.Drain's error and then performs an unbounded `<-replies`. Vendored Unix devTty.Drain sets a read deadline and sets VMIN/VTIME to zero; its normal success path is intended to release the read. I did not establish a normal supported-terminal hang. If Drain fails or cannot interrupt the read on a supported platform, startup can exceed the nominal bound indefinitely. Question for follow-up: should this failure path have an independently enforced upper bound? The existing probeTty mock always unblocks immediately on Drain, so its no-answer case cannot validate the real TTY behavior or a Drain failure.
- **Windows:** shouldProbeTrueColor returns false on Windows; vendored tcell enables Windows true colour itself. The raw-query path is therefore not introduced on native Windows. WSL follows the Unix path as intended.
- **Other probe skips:** NO_COLOR, TCELL_TRUECOLOR=disable, negotiation-disabled/legacy terminals and terminals excluded by colorSchemeQueries avoid the probe. COLORTERM=truecolor/24bit avoids it. tcell also recognizes COLORTERM=direct and TERM suffixes -direct/-truecolor; the fork does not skip those known-true-colour cases, so they can incur an unnecessary probe (and F1) even though tcell needs no help.
- **Clipboard transport:** The GUI delegates to vendored tcell.SetClipboard, which takes the screen lock and emits a single base64 OSC 52 sequence. There is no fork-side tmux/zellij wrapping, payload bound, chunking or acknowledgement. The vendored writer emits plain OSC 52 and discards write errors via Printf. This establishes transport behavior, not a confirmed failure in a particular multiplexer/version. Guide.md documents silent unsupported-terminal failure and tmux's set-clipboard setting. Questions: which tmux/zellij versions/configurations are supported, and should oversized copies be reported? Their limits and forwarding behavior need real-terminal validation before prescribing a fix. No size policy or new fallback was assumed.
- **Theme precedence:** ThemeForBackground merges background-specific user overrides, generic user theme, fork palette, then upstream background defaults. The basic-blue override changes style.FgBlue while retaining ColorMap's explicit ANSI-blue meaning. The associated tests check these distinctions; no contradictory path found.
- **Selection / search:** The explicit selection is indexed by vline.linesY, so wrapped content maps to logical selection lines. Cursor-only highlighting remains distinct, and search-match/flash handling follows the selection branch. List-view search retention is applied after constructing the context tree, whose views already exist; main views retain upstream sideways search behavior. Existing tests cover wrapped lines, gutter coverage and retained horizontal origin. Tests were read, not run.
- **Screen mode across repositories:** ScreenMode and ScreenModeArea both reside in GuiRepoState. resetState restores the entire cached state for an existing worktree and creates a fresh side-area zero value for a new one. The Activate focus hook then normalizes a mode if the restored target focus requires showing a hidden area. No split/global state mismatch found. Current helper tests and read-only integration scenarios cover side/main focus transitions, but I did not find an area-specific repository-switch scenario among the scoped screen-mode tests inspected.
- **Terminal title:** onNewRepo updates the title after successful setup; the helper uses the worktree path and only abbreviates a path-component-boundary home prefix. Vendored tcell sanitizes OSC controls and owns save/restore. No confirmed title issue found.
- **F5 cache invalidation:** DropPipeSetCache takes the same presentation mutex as rendering and replaces the map. No new concurrent-map access issue found in the production paths inspected.

## Out of scope observations

None pursued. Existing untracked HANDOFF.md, reviews/ and test/.viminfo were present before review and were left unchanged.

## Commands run and outcomes

- Read the worker request, common-review-rules.md, AGENTS.md and Tracker.md first. Read Design.md and the relevant Guide.md / MergeGuide.md sections. The tracker says no work is in progress; no tracker edit was made for this read-only task.
- `git status --short` before and after inspection: the same three pre-existing untracked entries; no tracked modifications reported.
- `git diff --stat 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- ...`: inspected area footprint. The initial gocui directory summary included other gocui work; the final source-file footprint uses the narrower paths listed above.
- `git diff 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- ...`: read the scoped upstream hooks in gocui view/driver, config theme, GUI views/context/layout/setup, presentation, screen actions/window arrangement, gui.go and oscommands.
- `git diff --numstat 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <12 upstream files>`: 179 added / 77 removed lines in those complete file diffs.
- `git log --oneline 4e1eebe278790f257a875da6633234391a5a85b3..HEAD -- pkg/gocui/fork_true_color.go pkg/gui/fork_clipboard.go pkg/gui/fork_screen_mode_area.go pkg/gui/presentation/fork_pipe_set_key.go`: identified relevant feature/fixup commits, including graph, focus-layout and clipboard changes.
- `cat`, `sed -n`, `rg -n` and `rg --files`: traced scoped implementation, corresponding unit tests, read-only screen-mode integration scenarios/components and vendored tcell behavior end to end.
- `go env GOMODCACHE`: returned `/home/vscode/go/pkg/mod`. A lookup there found no matching tcell source; the dependency is available under vendor and was inspected there instead.
- Several exploratory reads named nonexistent candidate files (graph/pipe_set.go, gui/fork_screen_mode_area_test.go, context/fork_list_search_scroll_test.go, types/repo_state.go and an oscommands clipboard glob). Subsequent searches found graph/graph.go, helpers/fork_screen_mode_area_test.go, gocui/fork_search_scroll_test.go, GuiRepoState in gui.go and clipboard tests in os_test.go. These lookup failures do not represent build/test failures.
- Tests: none. No new targeted runtime probe was needed to establish the reported deterministic code paths. No integration tests, broad unit tests, builds, vet, generation, formatting, commits or repository writes were performed.

## Files changed / work remaining / next step

- Repository files changed: none.
- Artifact written: `/tmp/lazygitk-review-run-1793910387/04-worker-result.md` only.
- Remaining implementation problems: F1–F3 are unmodified; actual latency impact of F2 and real multiplexer/platform behavior remain unmeasured.
- Recommended next step: triage these findings with the review coordinator. Within a separately authorized implementation task, preserve startup input with targeted tests, move graph identity computation out of the render hot path, and document clipboard merge seams. Resolve the explicitly listed terminal compatibility/size questions before selecting behavior for them. This review does not initiate a new design phase.
