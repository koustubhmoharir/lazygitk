# Review 06: branch-ref-actions

## Summary

Read-only review completed against BASE `4e1eebe278790f257a875da6633234391a5a85b3`.
The additive command modules generally provide careful literal-ref validation, repository-bound plans, worktree checks, and native error propagation. Placement additionally uses an expected-old-OID transaction with a safety guard under the destination lock.
The largest gap is the inherited move-commits implementation: it remains reachable through `g m n`, despite documentation describing captured flows, and can reset history without creating its destination, use changed HEAD/upstream state, and stash implicitly.
A native Git probe also confirmed an unhandled annotated-tag signing policy. Tags lack the documented branch-placement action.
Upstream-file footprint across the named controllers/helpers is 249 added / 551 deleted lines; most substantive replacements are appropriately reduced to forwarding hooks. This total includes other areas' changes in the same files, not just this review's behavior.
No repository files were changed. No Go tests, builds, generation, formatting, integration tests, or commits were run.

## Findings

### F1. Move-commits can reset the source without creating a replacement branch
- Category: bug
- Severity: high
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/helpers/refs_helper.go:357, pkg/gui/controllers/helpers/refs_helper.go:426, pkg/gui/controllers/helpers/refs_helper.go:460; supporting native constructors in pkg/commands/git_commands/branch.go:34 and :44
- Problem: The name prompt only replaces spaces. It neither validates a literal branch name nor checks destination availability before destructive work. The base-branch variant performs `reset --hard @{u}` before attempting destination creation. The stacked variant treats an option-parsing success as successful branch creation.
- Scenario / evidence: In the base variant, enter an existing branch name: the current branch is reset first, then `checkout -b <existing> <base> --no-track` fails; the unpushed commits have no new destination and are recoverable only through existing refs/reflogs. For the stacked variant, enter `--show-current`: `NewWithoutCheckout` constructs `git branch --show-current HEAD`, which succeeds while merely printing the current branch. The following hard reset succeeds, and `git checkout --show-current` fails. Confirmed the latter exact native command sequence in `/tmp/probe-move-branch-name-tenl76xe` with Git 2.51.1: original tip `7c48fb1fd092635f7c5ad0d7d3a9c01bc6850fad`, final main/upstream tip `389585876c807701cc6513b76ca80f92d7a1d9f3`, no replacement branch, checkout exit 129. This path is inherited, not newly introduced by the captured-create modules.
- Suggested fix: Validate the literal destination and its availability before any stash/reset, use option-safe arguments, and establish the destination's retention of the captured commits before moving the source. Cover existing destinations and option-looking input in focused regressions.

### F2. Move-commits retains live HEAD/upstream and stale dirty-state decisions across prompts
- Category: bug
- Severity: high
- Confidence: CLEAR BUG
- Location: pkg/gui/controllers/helpers/refs_helper.go:333, :340, :403, :426, :437, :460, :467
- Problem: This flow never creates a captured action repository or target. It reads the model again in the menu callback, calls `self.c.Git()` throughout execution, and passes live `HEAD` / `@{u}` to mutations. `mustStash` is a one-time model-derived flag before all prompts. Neither branch identity, tips, upstream configuration, nor tracked dirty state is revalidated.
- Scenario / evidence: Open `g m n` on branch A, then externally check out branch B with a different upstream while the name prompt is open. The stacked callback creates the new branch from B's HEAD and resets B to B's upstream, despite the earlier prompt naming A. Alternatively, start clean, edit a tracked file externally during the prompt, then confirm: `mustStash` stays false and the hard reset discards the new edits. End-to-end code tracing establishes both scenarios; no UI integration run was performed. `branches_controller.go:76` still installs this helper directly and `keymap/default_paths.go:123` assigns `g m n`.
- Suggested fix: Bind the source branch, original HEAD, upstream/base refs and OIDs, repository/generation, and commit list at invocation; revalidate before each destructive step and refuse changed state. Check actual tracked changes at execution instead of trusting the initial model flag. Preserve the already documented user-control policy.

### F3. Move-commits still stashes dirty files automatically
- Category: ux
- Severity: medium
- Confidence: n/a
- Location: pkg/gui/controllers/helpers/refs_helper.go:431, :445, :461, :481
- Problem: Both variants automatically push a stash and later pop index zero whenever the initial dirty flag is true. No explicit stash choice is offered. A failure after pushing can leave a stash behind; a concurrent stash can change which entry `Pop(0)` restores.
- Scenario / evidence: Dirty tracked worktree -> `g m n` -> choose variant and enter name -> `Stash.Push` executes directly. This contradicts Design.md's "Commit first; never stash implicitly" rule. The stacked/base helpers remain upstream implementations and bypass `captured_branch_create.go` entirely.
- Suggested fix: Apply the fork's existing no-implicit-stash rule to this operation, with an actionable refusal when dirty state cannot be carried safely. Do not preserve the current implicit push/pop behavior while addressing F1/F2.

### F4. Annotated tags signed through tag.forceSignAnnotated bypass terminal signing handling
- Category: bug
- Severity: medium
- Confidence: CLEAR BUG
- Location: pkg/commands/git_commands/branch_action_tag_create.go:55, :71, :89; pkg/gui/controllers/helpers/captured_tag_create.go:151
- Problem: `Signed()` considers only `tag.gpgSign`. Git also signs annotated tags when `tag.forceSignAnnotated=true`; the generated `git tag -m ...` command honors that policy, while the GUI incorrectly routes it to background `cmd.Run()` as unsigned. Users whose signing setup requires the terminal can get signing failure or an inaccessible prompt instead of the intended subprocess path.
- Scenario / evidence: In `/tmp/probe-branch-ref-actions-oby16jbw`, configured `tag.gpgSign=false`, `tag.forceSignAnnotated=true`, and `gpg.program=/bin/false`. `git config --type=bool --get tag.gpgSign` returned `false`; the same command shape as the fork, `git tag -m annotation -- signed-by-policy HEAD^{commit}`, attempted GPG signing and exited 128 with "gpg failed to sign the data". A lightweight tag command succeeded. Thus the fork's `signed=false` classification is demonstrably inconsistent with Git. The probe confirms signing selection, not an interactive terminal hang.
- Suggested fix: Include the effective force-sign-annotated policy when determining whether this specific command will sign, and exercise the terminal route with that configuration. Preserve lightweight-tag behavior.

### F5. Tags do not offer the documented n b branch placement
- Category: ux
- Severity: medium
- Confidence: n/a
- Location: pkg/gui/controllers/tags_controller.go:40; pkg/gui/controllers/branch_placement_refs.go:12; docs-master/fork/Guide.md:202
- Problem: The shared Branches/Remote branches/Tags table promises `n b` to create/move a branch at the selected object's commit, but Tags registers no `branch.place-at-selected-commit` binding.
- Scenario / evidence: All registration sites were searched. Placement bindings exist only for BasicCommitsController, BranchesController, and RemoteBranchesController. Tags has neither the placement binding nor a branch-create binding, and command projection rewrites existing bindings rather than synthesizing this missing action. The global default path assigns `n b` to the placement command; it cannot make it available in Tags without a handler. The documented action is therefore absent there.
- Suggested fix: Add the existing captured-ref placement hook for a selected tag, including annotated-tag peeling, or explicitly correct the guide if Tags was intentionally excluded. No product choice was assumed during this review.

### F6. Merge guide incorrectly claims move-commits uses captured selections
- Category: docs
- Severity: low
- Confidence: n/a
- Location: docs-master/fork/MergeGuide.md:375; pkg/gui/controllers/helpers/refs_helper.go:333
- Problem: The helpers entry lists move-commits-to-new-branch among captured replacements and points at captured modules that never handle it.
- Scenario / evidence: The diff against BASE shows only CommandID/Verb metadata changes inside this implementation. Its live model/backend lookups, automatic stash, reset, and checkout sequence remain intact. A maintainer following this entry could preserve an unsafe upstream path while believing the captured policy is covered.
- Suggested fix: Correct this entry alongside the move-commits remediation, clearly distinguishing replaced creation/reset flows from the remaining inherited move implementation.

## Merge-ease notes

Numbers below are whole-file `git diff --numstat BASE HEAD` counts, including bindings/other areas in those files.

| Upstream file | Added / deleted | Assessment |
| --- | ---: | --- |
| pkg/gui/controllers/branches_controller.go | 128 / 353 | Substantial deleted upstream menu bodies, but the final create/delete/rename/reset entry points are small delegations with `// Fork:` explanations. Extracting them further would mostly move trivial adapters; retaining deleted upstream implementations alongside replacements would increase dead code rather than improve the seam. `checkoutByName` is a larger inline captured-fallback flow and could move into a fork-owned module behind one forwarding method. Upstream/new menu functionality must be ported into the fork flow, not restored wholesale. |
| pkg/gui/controllers/remote_branches_controller.go | 45 / 8 | Small placement hook and captured full-ref adapters; appropriate footprint. Merge/rebase behavior was excluded from substantive review. |
| pkg/gui/controllers/helpers/refs_helper.go | 32 / 136 | NewBranch and CreateGitResetMenu are correctly replaced with small documented forwarding methods; remaining menu edits are mostly identity metadata. Move-commits is still inherited, contrary to the merge-guide entry. Keep its future remediation in an additive file behind one hook. |
| pkg/gui/controllers/helpers/tags_helper.go | 3 / 50 | Clean forwarding replacement with a precise Fork comment. |
| pkg/gui/controllers/tags_controller.go | 27 / 3 | Mostly identity/inspection metadata and reset adapter changes. Missing placement integration is F5; a single additive hook should suffice. |
| pkg/gui/controllers/stash_controller.go | 14 / 1 | Small, commented hash capture replacing the shifting stash index. Other additions are command metadata. |

Backend placement/create/delete/reset/tag/rename modules are additive. No avoidable sweeping backend edits were identified in scope.

Shared-structure opportunity: captured create/reset/tag/delete flows repeatedly implement repository capture, single-use flags, worker checks, error joining, and UI-thread refresh callbacks. `captured_action_repository.go` already provides the right small shared guard. Prefer a narrow shared completion/guard helper if useful; keep operation-specific approvals and native commands explicit. Avoid a generic modal-flow framework. Placement separately duplicates repository capture and only its ref entry point captures generation; its commit entry point checks backend identity alone. Consistently using the same capture helper would reduce drift, but I did not establish a reachable production failure from that difference: normal repo switching constructs a fresh backend and foreground work participates in busy tracking.

## Verification and test-coverage assessment

- Skimmed backend tests for creation/tracking policies, deletion proofs and reservations, remote partial results, reset modes, tag overwrites/signing, rename metadata, placement races, and transaction protocol failures.
- Placement tests inject a concurrent OID change, a symbolic alias, and a new worktree before the locked guard; transaction tests cover hook output/rejection and process failures. These test meaningful failure boundaries rather than only command spelling.
- Deletion tests cover per-source upstream proof isolation, excluded tracking refs disappearing in compound deletion, occupied and operation-reserved branches, cached remote source removal, multiple remotes, and native partial failure across push URLs.
- Reset tests cover dirty index/worktree modes, detached/unborn HEAD, active merges, index-lock errors, and other-worktree reservations. The explicit hard-reset warning describes possible ignored/untracked collision loss.
- GUI captured-create/delete/tag/reset tests exercise queued callbacks, changed targets, repository replacement/generation, and refresh behavior. Placement repository tests cover replacement before prompt/confirmation submission, not every worker/refresh boundary.
- One coverage weakness: `TestBranchActionLocalDeleteExcludedCustomUpstreamMayDisappear` constructs the post-deletion local command and inspects its arguments but never executes it; its name should not be read as full mutation coverage.
- No unit regression for the inherited MoveCommitsToNewBranch path was found in the scoped flow/helper tests. The new captured-creation tests do not cover that separate command.
- The integration components `branch_create.go` and `branch_delete.go` were read only. Their assertions check review identities, absence of automatic stash, and reserved-action tooltips; they do not establish coverage for move-commits or force-sign-annotated tags.
- No Go tests were executed, per the request's default. Two narrowly scoped native-Git probes were sufficient to confirm the suspected command semantics; no integration fixtures were touched.

## Questions / remaining limits

- Remote deletion deliberately reviews named server refs without obtaining an expected server OID, then runs native `push --delete`. Does the worker feature summary's "expected old values" intend a server-tip lease here, or only the current implementation's cached-source/config guards? The current design's normal-native-concurrency language does not settle that stronger requirement. No missing lease is asserted as a bug.
- Branch placement invoked from a ref resolves it to an immutable commit and subsequently guards the destination rather than rechecking that source ref after the name/move prompts. Is that intended captured-commit semantics, or should a source-ref move invalidate the prompt as for reset/create? Reported as a question; no redesign performed.
- Source inspection for several ref actions begins in a worker and captures the ref's then-current OID, rather than comparing against the invocation-time model tip. This review treats the later displayed native snapshot as the reviewed state; clarify only if the intended guarantee is stricter.
- Interactive GPG behavior and the full application UI were not exercised. F4's classification error and Git signing attempt are confirmed; a hang depends on the user's signing setup.

## Out of scope observations

No separate cross-area finding. Supporting command builders, key projection, repo busy tracking, MainBranches, and native constructors were read only to establish the in-scope call paths.

## Commands run and outcomes

- Read worker request, common-review-rules.md, AGENTS.md, Tracker.md, Design.md, relevant Guide.md/MergeGuide.md sections, and justfile using `cat`, `sed`, and `rg`. Tracker says no active work; read-only worker constraints take precedence over the normal implementation/commit workflow.
- Used `git diff --stat`, `git diff --numstat`, `git diff BASE HEAD -- <scoped paths>`, and `git log BASE..HEAD -- <scoped paths>` with the full BASE hash above. Reviewed scoped backend/controller/helper source and skimmed their tests with `cat`, `sed`, `nl -ba`, and `rg`.
- `git --version`: Git 2.51.1.
- Executed two Python scripts from stdin solely to invoke native Git and create the permitted throwaway `/tmp/probe-*` repositories. No script file was written. Their substantive Git commands/outcomes are recorded in F1/F4. Probe repositories are retained for inspection; their paths are listed there.
- F1 probe setup: `git init -q -b main`; set local user identity; empty base commit; `git branch upstream`; `git branch --set-upstream-to=upstream main`; empty unpushed commit. Then `git branch --show-current HEAD` exited 0; `git reset --hard @{u}` exited 0; `git checkout --show-current` exited 129; `git branch --format='%(refname) %(objectname)'` and `git reflog -2 --format='%h %gs'` confirmed the reset and lack of a new branch.
- F4 probe setup: `git init -q -b main`; set local user identity; empty base commit; configure `tag.gpgSign=false`, `tag.forceSignAnnotated=true`, `gpg.program=/bin/false`. Annotated creation exited 128 attempting signing; lightweight creation exited 0.
- Some exploratory `rg` calls included nonexistent glob paths (`pkg/gui/repo*`, `pkg/gui/state*`, `pkg/gui/status.go`, `pkg/gui/controllers/helpers/common.go`) and returned exit 2 for those operands. Subsequent searches used actual files; these were discovery misses, not validation failures.
- `git status --short` before and after investigation returned the same pre-existing untracked paths: `HANDOFF.md`, `reviews/`, `test/.viminfo`. They were not modified.
- No tests/builds/lint/generation/formatting or commits run. The only handoff file written is this result artifact; repository files changed: none.

## Recommended next step

Address F1/F2/F3 together in the existing move-commits action as a narrowly scoped correction to the already recorded policies: validate and protect the destination before resetting anything, retain the captured repository/branch/ref state, and remove implicit stash. Add focused regressions for existing/option-looking destination names, a changed HEAD/upstream during name entry, and dirty-state changes; update the inaccurate merge-guide entry in the same work. Then fix the tag-signing classification and resolve the documented Tags placement discrepancy. Leave the semantic questions above for the coordinating review; do not silently broaden mutation guarantees during a fix.
