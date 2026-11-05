# Review 07: branch-merge-rebase

## Summary

Read-only review completed against BASE `4e1eebe278790f257a875da6633234391a5a85b3`.
Captured merge/rebase plans, native squash execution, repository guards, and isolated merge preview are generally carefully implemented.
One high-severity ignored-file protection failure was confirmed with native Git: rebasing through a directory rename can overwrite an unreviewed ignored file, including on a successful rebase.
The large helper diff is mostly appropriate removal of upstream menus in favor of fork-owned modules: +77/-265 lines. Controller integration points are small and commented; whole-file controller statistics include other review areas.
There are also documented UX inconsistencies and unused upstream merge implementations left behind.
No repository file was modified; no implementation, commits, integration tests, broad tests, build, or formatting was performed.

## Findings

### F1. Rebase ignored-file review misses paths synthesized by directory renames
- Category: bug
- Severity: high
- Confidence: CLEAR BUG
- Location: pkg/commands/git_commands/branch_action_rebase.go:49,56,60; shared implementation pkg/commands/git_commands/rebase_ignored_files.go:23-38; pkg/gui/controllers/helpers/captured_rebase.go:207-208
- Problem: Both ordinary and marked-range branch rebases derive ignored-file risks from the destination's existing tree paths and the original paths of replayed commits. Git's merge backend can create a different path during replay. That path is absent from the candidate set, so it receives neither an approval prompt nor a content fingerprint. Native rebase does overwrite ignored files at such paths.
- Scenario / evidence: Base tracks `old/a`; selected destination `main` renames `old/` to `new/`; checked-out `topic`, branching from base, adds `old/b`. The worktree contains ignored `new/b` with `PRECIOUS\n`. Target tree candidates are only `new/a`; replay candidates are only `old/b`; neither collides with ignored `new/b`. The captured flow therefore treats its ignored risk list as empty and executes. With Git 2.51.1 and `merge.directoryRenames=true`, the exact rebase options used by the flow completed successfully and replaced `new/b` with `topic\n`. With the default directory-rename policy, Git stopped with `UA new/b`, but had already replaced the precious bytes. Repeating the same candidate query at execution cannot catch this: its inputs still omit `new/b`.
- Suggested fix: Extend the shared risk calculation to cover paths created by native replay/rename processing, or conservatively require approval for ignored files whose safety cannot be established. Add ordinary and marked-base regression cases, including default directory-rename conflict behavior and `merge.directoryRenames=true`; assert approval is required before the original bytes can be replaced. Apply the existing explicit-approval rule; no new design decision is needed to establish that silently losing these bytes is incorrect.

### F2. Ordinary merge/rebase choices always open a redundant confirmation
- Category: ux
- Severity: medium
- Confidence: n/a
- Location: pkg/gui/controllers/helpers/captured_merge.go:105-120,177-181; pkg/gui/controllers/helpers/captured_rebase.go:117-155,175-179
- Problem: The mode menu already names the current branch, selected source and mode/options, but choosing an ordinary merge or simple rebase always opens another confirmation repeating the same information. Actual staged/ignored risks are handled by separate later reviews. This conflicts with Design.md's explicit “No redundant reviews” rule.
- Scenario / evidence: In a clean repository, choose `m`, then an ordinary merge mode, or `r`, then simple rebase. `prepare` unconditionally calls `c.Confirm` even when there are no new risks. Controller flow tests intentionally call the extra confirmation handler, so they verify the implementation rather than compliance with that design rule. Squash-and-commit's newly entered message can legitimately warrant a distinct review and is not the basis of this finding.
- Suggested fix: Follow the recorded rule for choices that reveal no new risk, retaining explicit new-risk approvals. If the extra universal confirmation is intentional, ask the user whether the documented rule has an exception before changing behavior or documentation; do not infer that exception.

### F3. Squash menu retains opposite actions on s and S despite the case-insensitive key rule
- Category: ux
- Severity: medium
- Confidence: n/a
- Location: pkg/gui/controllers/helpers/captured_merge.go:109-110
- Problem: Lowercase `s` means squash without a commit; uppercase `S` means squash and commit. Design.md says letters are case-insensitive, with only Shift+J/Shift+K excepted. This is a meaningful behavioral difference, not just a label discrepancy.
- Scenario / evidence: These are ordinary modal item accelerators. `MenuContext.GetKeybindings` retains native item keys when `blockUnmatchedKeys` is empty, `DefaultCommand` is false, and `Gui.SetKeybinding` registers exact matching. Thus the two routes actually differ; this is not a normalized-key collision that makes an item unreachable. The merge controller tests invoke items by CommandID and do not exercise these physical keys.
- Suggested fix: Resolve the existing documentation/code inconsistency without silently choosing a new accelerator. Question for the owner: does the case-insensitive rule apply to this modal, and if so which distinct accelerator should the committed variant use? Preserve both explicit choices.

### F4. Replaced upstream merge implementations remain as unused helper methods
- Category: code-quality
- Severity: low
- Confidence: n/a
- Location: pkg/gui/controllers/helpers/merge_and_rebase_helper.go:471-511
- Problem: `RegularMerge`, `SquashMergeUncommitted` and `SquashMergeCommitted` have no callers anywhere under `pkg`; the captured flows replace their former callers. They retain a second, live-ref/config-based implementation of merging, including the old immediate squash commit path. This obscures which safety checks are authoritative and creates unnecessary code to inspect during upstream merges.
- Scenario / evidence: `rg -n 'RegularMerge\(|SquashMergeUncommitted\(|SquashMergeCommitted\(' pkg --glob '*.go'` found only their definitions. Approximately 41 lines remain after the new commented delegation hooks.
- Suggested fix: Remove the unused implementations in an authorized cleanup, updating the helper replacement note as appropriate. Do not rewire captured flows back into these methods.

## Merge-ease notes

- `pkg/gui/controllers/helpers/merge_and_rebase_helper.go`: +77/-265 against BASE. The 223-line menu replacement hunk is largely deletion, with two small `// Fork:` delegation hooks. This is appropriate isolation rather than 342 lines of new inline fork logic. Continue/skip/abort callback hooks are only one call per execution path and are documented. The callback storage and settlement method could live in an additive helper file to trim upstream-file additions; the unused methods in F4 should not remain as alternative implementations.
- `pkg/gui/controllers/branches_controller.go`: whole-file +128/-353, much of which belongs to other areas. In this area, `merge` and `rebase` pass `FullRefName()` with explanatory fork comments; binding labels and identities change locally. These hooks are small and suitable for upstream merges.
- `pkg/gui/controllers/remote_branches_controller.go`: whole-file +45/-8, also including unrelated actions. Merge/rebase change only the source passed to `FullRefName()` plus comments and binding descriptions/identities. Good integration shape.
- `pkg/integration/components/branch_merge.go`: additive 13-line assertion component; read only. It requires the explicit confirmation currently described in F2, so a behavior fix would need coordinated assertion updates.
- Captured command, preview, execution, rebase-result, and continue-operation files are additive fork modules, limiting conflicts in upstream files. Rebase safety depends on the shared `rebase_ignored_files.go`; F1 identifies that dependency rather than proposing unrelated edits in this review.
- MergeGuide.md “Helpers (replacements)” accurately records the important `settleOperationFinished` hooks in both subprocess and worker paths, including recursive auto-skip. Its opening wording is imprecise: `CreateRebaseOptionsMenu` implements continuation options itself; it does not delegate into captured merge/rebase start flows. Correct that wording opportunistically.

## Investigation and test coverage assessment

- Preview uses `merge-tree --write-tree` with private `GIT_OBJECT_DIRECTORY` and preserved alternates. It creates neither a temporary worktree nor an index. Deferred `RemoveAll` runs on command/parse/tree-query errors as well as success; cleanup errors join the returned error. No ordinary-return cleanup leak was identified. Native driver execution is explicitly documented; arbitrary driver side effects are outside the promise of object-store isolation.
- Preview tests cover clean/conflicted trees, directory-rename result paths, dirty index/worktree preservation, quoted object paths and inherited alternates, unrelated histories, invalid strategy options, driver stdout contamination, and temporary-store removal. These are meaningful behavioral assertions; they were inspected, not executed.
- Squash plans issue native `merge --squash --ff --no-commit`; the commit is a separate phase after native success only. Checks for already-merged sources, unchanged index/no new staged result, changed captured state, and explicit inclusion of preexisting staged files address accidental commits. Conflicted squash has no MERGE_HEAD-based abort/continue operation and the UI explicitly presents native conflict state rather than automatically committing. Command and controller tests cover these distinctions, native hook/signing failures, and stale reviews.
- Branch rebase explicitly disables update-refs, passes no-autostash, shares the clean tracked-tree guard, and validates the marked replay range separately. Tests check dirty tracked files, harmless untracked files, stale refs/HEAD, metadata-bound abort, ignored-byte changes, and terminal failures. The ignored-risk test covers paths removed earlier in the marked range; it does not cover rename-generated paths, which is the gap in F1.
- Captured merge and rebase perform full command preparation again on the UI thread immediately before terminal launch (`captured_merge_execution.go:44`, `captured_rebase_execution.go:48`; squash commit similarly). This preserves a fresh check but repeats potentially expensive Git/config/worktree enumeration and ignored-file hashing after worker preflight. No latency measurement was performed; consider responsiveness when touching this boundary, without weakening state validation.
- `settleOperationFinished` retains the callback while an operation remains, clears it after abort, and invokes/clears it after a successful ending continue or skip. Both generic execution branches contain the hook. No direct test references to `settleOperationFinished` or `onOperationFinished` were found in the inspected test tree; do not infer runtime coverage merely from the merge guide's description.
- Potential continuation limitation to investigate next if needed: merge continuation chooses a terminal only for `Merging.ManualCommit`, whereas rebase continuation also checks `NeedsGpgSubprocessForCommit`. A terminal-dependent signer after resolving a merge conflict may still need the same treatment. This was not reproduced and is not counted as a confirmed finding.

## Out of scope observations

- The same shared ignored-risk helper also serves selected-commit rebase. F1 is directly reachable through the branch flow in scope; an eventual correction should check other callers too.
- Existing untracked repository artifacts were present before this review and remained untouched: `HANDOFF.md`, `reviews/`, `test/.viminfo`.

## Commands run and outcomes

- Read worker request 07, common-review-rules.md, AGENTS.md, Tracker.md, Design.md, Guide.md “Rebase, merge, reset”, and MergeGuide.md “Helpers (replacements)”. Tracker reported no active work. The worker's read-only scope overrides the general implementation/commit workflow for this task.
- Used `git diff 4e1eebe278790f257a875da6633234391a5a85b3 HEAD -- <scoped paths>`, `git diff --stat`/`--numstat`, and `git log --oneline BASE..HEAD -- <scoped paths>`; read scoped source, tests, and small shared dependency/dispatch sections using `cat`, `sed`, `nl`, and `rg`. Some exploratory searches named nonexistent helper files; these returned read errors only and were followed by searches for the actual files.
- Inspected justfile; did not run generation, formatting, broad tests, integration tests, or build commands. No Go tests were necessary: the suspected data-loss behavior was confirmed with the permitted native-Git probes and traced to the precise candidate-set logic.
- Two throwaway repositories were created under `/tmp/probe-branch-rebase-*`, as expressly permitted. No scripts or source files were written into the project. Probe repositories were left available for inspection:
  - `/tmp/probe-branch-rebase-48oiU4`: default rename handling; rebase exited 1 with a file-location conflict, `git status --short` showed `UA new/b`, and `cat new/b` returned `topic` instead of `PRECIOUS`.
  - `/tmp/probe-branch-rebase-9w9wUW`: `merge.directoryRenames=true`; rebase exited 0, final status was clean, and `cat new/b` returned `topic` instead of `PRECIOUS`.
- `git --version`: `git version 2.51.1`.
- Initial and final `git status --short` matched: only the three preexisting untracked entries listed above.

### Reproduction of the successful data-loss probe

All writes below are in a newly allocated throwaway repository:

```sh
probe_root=$(mktemp -d /tmp/probe-branch-rebase-XXXXXX)
cd "$probe_root"
git init -q -b main
git config user.name Probe
git config user.email probe@example.invalid
git config merge.directoryRenames true
mkdir old
printf 'base\n' > old/a
git add old/a
git commit -qm base
git branch topic
git mv old new
git commit -qm rename
git checkout -q topic
printf 'topic\n' > old/b
git add old/b
git commit -qm addition
printf '/new/b\n' > .git/info/exclude
mkdir new
printf 'PRECIOUS\n' > new/b
# These are the planner's path inputs; neither includes new/b:
git ls-tree -r -z --name-only 'main^{commit}'
git log --format= --name-only --no-renames --root -m -z \
  'main^{commit}..HEAD^{commit}' --
git ls-files --others --ignored --exclude-standard
GIT_SEQUENCE_EDITOR=true git -c rebase.updateRefs=false rebase \
  --interactive --no-autostash --keep-empty --no-autosquash \
  --rebase-merges 'main^{commit}'
cat new/b  # now topic, original PRECIOUS bytes lost
```

## Work performed, files changed, remaining problems, and next step

- Work performed: detailed static review, comparison against BASE and fork docs, test coverage inspection, and two narrowly scoped native-Git reproductions. No design or implementation phase was started.
- Repository files changed: none. The only handoff artifact written is `/tmp/lazygitk-review-run-1793910387/07-worker-result.md`; throwaway Git fixtures were created under the explicitly allowed `/tmp/probe-*` exception.
- Remaining problems: F1-F4 remain unfixed by design of this read-only request. No automated project test pass is claimed.
- Recommended next step: prioritize an authorized fix and focused regression tests for F1 before relying on the advertised ignored-file guarantee. Preserve both current tracked-tree/autostash/update-refs protections and explicit ignored-risk approval. Record the modal key and redundant-confirmation questions for the owner instead of opening a new design phase here. Follow with the small unused-code/doc cleanup when authorized.
