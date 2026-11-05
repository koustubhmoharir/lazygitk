# Review 12: diffs-tools

## Summary

Read-only review against BASE 4e1eebe278790f257a875da6633234391a5a85b3.
Substantial implementation lives in additive fork modules; the six reviewed command-layer modules add 672 lines.
Historical blob handling, temporary cleanup, discard confirmation and pasted-hash generation checks are generally well structured.
Four correctness problems remain: pattern expansion in staged comparisons, GUI settings overriding tool choice, invalid configured tools triggering guessing, and merge failures skipping refresh.
Three were confirmed using native Git probes; the fourth follows the complete production call path.
No repository changes, implementation, design phase, builds or Go tests were performed.

## Findings

### F1. Staged comparison expands selected filenames as patterns
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/working_file_diff_actions.go:113; supporting builder pkg/commands/git_commands/diff.go:237
- Problem: The new Files d s route calls OpenDiffToolCmdObj, which passes the path after -- without enabling literal pathspecs. Shell quoting does not prevent Git pattern expansion. This contradicts the module's stated literal-pathspec behavior and can open unselected files.
- Scenario / evidence: In the probe repository, modified tracked files literal[1].txt and literal1.txt. The builder-shaped command "git difftool --no-prompt -- literal[1].txt", with a harmless configured tool printing $MERGED, invoked the tool for BOTH files. Adding --literal-pathspecs invoked only literal[1].txt. Historical d w/d c builders already use literal pathspecs. TestWorkingFileDiffFileEndpoints asserts the nonliteral argv with an ordinary filename, so misses this.
- Suggested fix: Use a literal-pathspec builder for this fork comparison, or minimally extend the shared builder with merge-guide documentation. Add a real Git regression containing both pattern-like and matching ordinary filenames.

### F2. GUI configuration overrides the chooser's explicit tool selection
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/commands/git_commands/fork_external_tool.go:54 and :138; pkg/gui/controllers/helpers/fork_external_tool.go:183 and :205
- Problem: Detection and one-launch application consider diff.tool/merge.tool only. Git's GUI defaults give diff.guitool/merge.guitool precedence. With only a GUI tool configured, the chooser opens unnecessarily, then can launch that configured GUI tool instead of the user's chosen tool. Saving globally also fails to pin the immediate launch.
- Scenario / evidence: Native Git 2.51.1 command with difftool.guiDefault=true, diff.guitool=guiprobe, diff.tool=selected and distinct harmless custom commands printed "GUI chosen", never "selected chosen". This reproduces the chooser's effective config override. Installed git-mergetool--lib confirms GUI diff precedence diff.guitool, merge.guitool, diff.tool, merge.tool; GUI merge precedence merge.guitool, merge.tool; guiDefault=auto checks DISPLAY.
- Suggested fix: Respect Git's effective GUI-default configuration when detecting a configured tool. Pin an explicit user choice using --tool=<name> or an equally authoritative override. Preserve the requested global save while ensuring the immediate launch uses the selected tool. Cover true/auto defaults in both modes.

### F3. Invalid configured tool names still trigger Git's guessing
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/helpers/fork_external_tool.go:123
- Problem: Every nonempty configured string bypasses the chooser and launches an unchanged command. Git resets unknown tool configuration to its discovery path and guesses another tool. This violates Design.md's "Tools are never guessed" rule.
- Scenario / evidence: Native probe with diff.tool=definitely-not-a-real-tool and a harmless difftool.vimdiff.cmd printed "set to unknown tool", "Resetting to default...", the guessing banner, then "GUESSED vimdiff". No editor ran. Installed get_configured_merge_tool/get_merge_tool explains the fallback, shared by merge mode. TestExternalToolConfiguredLaunchesWithoutMenuOrEnvChange records a callback without executing Git, so its arbitrary configured tool name does not expose the issue.
- Suggested fix: Pin the detected configured tool explicitly so invalid configuration returns an error instead of discovering another tool. Add an execution-level regression with an invalid configured name and detectable harmless fallback.

### F4. Merge-tool failure skips refresh after partial resolution
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/helpers/working_tree_helper.go:169; supporting implementation pkg/gui/gui.go:1128
- Problem: OpenMergeTool uses RunSubprocessAndRefresh, which returns immediately on subprocess errors. Refresh happens only on success, contrary to Guide.md's "Files refreshes when the tool exits" and the design requirement to refresh actual state after rejection.
- Scenario / evidence: Select two conflicted files; the tool resolves/stages the first, then fails on the second. The worktree/index has changed despite the nonzero exit. Production path: OpenMergeTool -> RunSubprocessAndRefresh -> runSubprocessWithSuspenseAndRefresh. The error return at gui.go:1130 precedes refresh at :1134. Automatic file refresh is disabled by default, so Files can continue showing the resolved file as conflicted. Confirmed by source tracing, not a UI execution. Current merge-tool unit tests cover paths, configuration and selection, not failure refresh.
- Suggested fix: RunSubprocess, then refresh on both success and failure while preserving the subprocess error. Add a targeted regression for a failing run after partial resolution.

## Merge-ease notes

Counts are whole-file BASE-to-HEAD additions/deletions; mixed-purpose controllers include changes outside the openDiffTool-only scope.

- commands/git_commands/working_tree.go: 0/4; small removal of obsolete no-path builder. Replacement rationale lives in fork_merge_tool.go and MergeGuide.md.
- controllers/basic_commits_controller.go: 64/46 overall; reviewed difftool hook is a small commented chooser wrapper.
- controllers/commits_files_controller.go: 82/21 overall; same small commented launch hook, historical bindings delegated.
- controllers/files_controller.go: 197/81 overall; reviewed difftool hook is localized and commented. Fix F1 in the existing fork module if possible.
- controllers/helpers/diff_helper.go: 17/6 overall; constructor dependency plus a small commented OpenDiffToolForRef wrapper.
- controllers/helpers/working_tree_helper.go: 38/6 overall; constructor wiring, replacement launcher and selected-path callback are localized and documented.
- controllers/main_view_controller.go: 114/12; mostly command metadata and expanded compact navigation declarations. Dynamic identities delegate to diff_command_identity.go. Expansion increases merge surface but does not broadly rewrite behavior.
- controllers/merge_conflicts_controller.go: 31/0; command metadata, binding append and a four-line commented render correction are localized.
- controllers/working_tree_diff_actions.go: 8/15; good replacement seam delegates mandatory approval to fork_diff_discard.go with a Fork comment.
- controllers/commit_diff_actions.go: 2/2; minimal fixed-key error-label substitutions.
- controllers/helpers/diff_line_file_menu.go: 3/1; command metadata only.
- commands/oscommands/subprocess_exit_status.go is additive. Existing integration is the per-command field in cmd_obj.go and commented normalization in Gui.runSubprocess; accepted statuses remain opt-in.
- MergeGuide.md accurately enumerates chooser call sites, UI-thread requirements, cleanup ownership, selected-path merge launching, removed upstream builder and generation-guarded pasted-hash refresh. F2/F3 must be fixed for the promised tool-selection guarantees to hold.

## Investigation and coverage

- Read worker/common rules, AGENTS.md, Tracker.md, Design.md, relevant Guide.md and MergeGuide.md sections.
- Inspected scoped implementations, changes/history and unit tests; followed necessary supporting argv/environment, menu/worker and subprocess code.
- Temporary historical copies: destination and metadata validation precede allocation; symlinks/gitlinks are rejected; binary blobs stream into private files made read-only. Preparation errors clean the directory. Launch defers Close including apply/subprocess failures; cancellation uses the settled flow. No concrete cleanup bug found on traced paths.
- GIT_CONFIG_COUNT: existing entries are preserved and a new entry appended. Normal builders explicitly copy os.Environ; Go's last-duplicate behavior makes appended indexed entries effective. Tests cover preservation/malformed counts. No clobbering bug found.
- Tool-help parser matches installed Git 2.51.1's tabs/unavailable marker; tests include user-defined-only tools, dotted names, duplicates and CRLF. Other Git versions were not executed.
- Tool discovery runs on a worker, then opens menus on UI thread after repository identity/generation checks. No concrete thread misuse found.
- Pasted-hash comparison captures paths/repository generation, resolves object identity without ref shadowing and guards post-subprocess refresh. Tests cover changed generation, deleted historical files, hash validation, SHA-256 and ref shadowing. The UI mock's Refresh is a no-op, so tests do not establish conditional refresh invocation.
- Discard tests cover immediate staged unstage, mandatory confirmation despite skip-warning config and changed-generation refusal. Command identity mapping has table coverage.
- Merge tests cover conflict selection, backup configuration freshness, explicit paths and no-path refusal. Unmerged-path tests verify marker removal is insufficient until staging.
- No Go tests ran: request says none by default; source tracing and native Git probes established the findings. No builds, vet, generation, formatting, broad suite or integration tests ran.

## Out of scope observations

None requiring another finding. gui.go and diff.go were read solely to establish scoped callers' runtime behavior.

## Commands run and outcomes

- cat of worker request/common rules and fork instructions/docs: read and followed.
- cat/sed/rg over scoped sources/tests and supporting code: completed. Some guessed filenames were absent; rg located git_command_builder.go and cmd_obj_builder.go instead.
- git diff BASE HEAD -- <scoped paths>, git diff --numstat/--stat BASE HEAD -- <paths>, git log --oneline BASE..HEAD -- <paths>: inspected changes, footprint and feature history, always using the full BASE hash above.
- git --version: 2.51.1.
- git difftool --tool-help: success, expected tab-indented sections.
- Read /usr/lib/git-core/git-mergetool--lib: confirmed GUI precedence and unknown-tool guessing.
- Created isolated /tmp/probe-diffs-tools-WI6grR with mktemp, git init, throwaway commits and two files. All tool settings were per-command git -c overrides; no user/global config changed.
- Native probe commands, with custom tool commands printing their identity or $MERGED:
  - git -C <probe> -c diff.tool=probe -c 'difftool.probe.cmd=printf "tool file: %s\n" "$MERGED"' difftool --no-prompt -- 'literal[1].txt': invoked literal1.txt AND literal[1].txt.
  - Same with --literal-pathspecs: only literal[1].txt.
  - git -C <probe> -c difftool.guiDefault=true -c diff.guitool=guiprobe -c diff.tool=selected -c 'difftool.guiprobe.cmd=printf "GUI chosen\n"' -c 'difftool.selected.cmd=printf "selected chosen\n"' difftool --no-prompt -- literal1.txt: printed GUI chosen.
  - git -C <probe> -c diff.tool=definitely-not-a-real-tool -c 'difftool.vimdiff.cmd=printf "GUESSED vimdiff\n"' difftool --no-prompt -- literal1.txt: guessing diagnostics then GUESSED vimdiff.
- git status --short before and after: same existing untracked HANDOFF.md, reviews/, test/.viminfo; untouched.
- Wrote this artifact by heredoc. An initial tool-wrapper JavaScript syntax error executed no shell command; corrected invocation wrote the artifact.

## Files changed, remaining problems and recommendations

Repository files changed: none. No commits or PRs created.
Deliverable: /tmp/lazygitk-review-run-1793910387/12-worker-result.md.
Throwaway probe files remain under the /tmp/probe directory above.

F1–F4 remain unresolved because implementation is prohibited by this worker request.
Recommend narrow fixes and targeted regressions for actual Git path matching/tool selection and merge error refresh. Keep changes in existing fork modules where possible; update MergeGuide.md if shared builder/subprocess integration changes. These fixes enforce existing recorded behavior and need no new design phase.
