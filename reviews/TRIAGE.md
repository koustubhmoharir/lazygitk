# Fork review triage (2026-11-05)

Fifteen read-only reviews of the fork's divergence from upstream commit
`4e1eebe278790f257a875da6633234391a5a85b3`, one area each, by `gpt-6.1-sol`
(medium effort) through `agents/claude-coordinator-worker-flow.md`. Findings are
referenced as `NN-Fk` (review file `NN-*.md`, finding `Fk`).

The reviews were evidence-based, but treat every finding as a claim: confirm it
before fixing (a failing test first where practical).

## 1. Clear bugs to fix (grouped into subagent batches)

Order is roughly by risk. Each batch is one Haiku subagent task under
`agents/claude-subagent.md`; the coordinator reviews every diff.

| Batch | Findings | What |
|---|---|---|
| A. Small isolated fixes | **Fixed:** 03-F1, 03-F3, 01-F4, 03-F2, 09-F2, 09-F4, 12-F4, 06-F4, 11-F3, 12-F1, 13-F1 | Fork dist build must refuse a user-requested self-update; stub `forkLookPath` in the renderer-name test; guard patch-commit index -1; refresh after a failed older-commit reword; literal pathspecs for `d s` and copy-diff; refresh Files after a failed merge tool; honour `tag.forceSignAnnotated`; `w v` reachability uses the log's ref during bisect; inventory collects `SetCommandKeybinding`; show startup notices as one combined toast |
| B. Checkout | **Fixed:** 08-F1..F7; B-U (new): undo/redo of a checkout used upstream `CheckoutRef` (plain checkout, autostash), now uses the guarded flows. Undo/redo of a rebase and redo of a commit still hard-reset with autostash: moved to J |
| C. Cross-instance file safety | **Fixed:** 02-F1, 02-F2, 14-F1, 14-F3. Shared OS file lock (`pkg/utils/filelock`) for usage counts and drafts; read-modify-write under it with unique temp files; corrupt files recovered under the lock (drafts moved aside with a notice); draft save failures keep the box open (Escape, submit, editor switch, Up/Down). Quit with the box open is unreachable from inside the app, so a save failure there is only logged |
| D. Commit list state | **Fixed (D1):** 10-F1 (U2, U2a): the multi-selection set is kept by hash, counted in the subtitle; `c p`, `g c`, `w a`/`w d` act on the whole set; rewritten members get a notice, missing objects are dropped with a notice; `w v` keeps the set. **Fixed (D2):** 10-F2: the pick's selection cleanup follows the final operation state, so an auto-skipped empty pick that ends the sequence clears it. **Fixed (D3):** 11-F2: clearing a scope or filter loads the list further when the restored commit is beyond the limited list. **Fixed (D4):** 11-F1: the descendant scope follows the commit limit; a forward walk stops after the last reachable selected commit and parses only kept rows | Clear the selection when an empty pick is auto-skipped; clearing a scope restores the cursor beyond the first 300 rows; descendant scope must not retain the whole walked history (stream/cap) |
| E. External tools | **Fixed:** 12-F2, 12-F3 | Account for `diff.guitool`/`merge.guitool`; a configured tool Git does not know opens the chooser instead of letting Git guess. Fixed: the configured tool follows Git's GUI selection (`guiDefault`, `GIT_MERGETOOL_GUI`), and every launch pins `--tool=<name>`, so an unknown configured tool fails with Git's own error rather than being guessed |
| F. Sync | **Fixed (F-a):** 05-F1 (no fetched-history veto when push and fetch URLs differ; lease review notes it), 05-F2 (unusable configured pull default is crossed out in the integration menu). **Fixed (F-b):** 05-F3 (the UI handoff only rechecks HEAD and the source; full preparation stays on the worker), 05-F5 (dormant stack-push and native current-pull code removed) | Skip the fetched-history veto when a `pushurl` differs; offer explicit integration methods for unsupported/invalid pull policies; do not repeat full inspection on the UI thread; delete dormant stack-push and old current-pull code (push is one branch by design) |
| G. Reword target | **Fixed:** 09-F3. Reword captures the repository, target, HEAD ref/ID and commit list at start (refusing a stale list), verifies before any write on both submit paths, and keeps the text as a draft on refusal, including via the editor |
| H. Prefix menus | **Fixed:** 01-F1, 01-F3 | Continuation rows carry the leaf's disabled reason (Design: failed preconditions are crossed out in every menu); key labels share the dispatch traversal. **Fixed:** a single leaf projects its reason and tooltip; several leaves are crossed out only when all are disabled, with their shared reason or the generic "Nothing here is available now". Labels use the display projection's routes with static applicability, since labels are cached and read from workers |
| I. Move commits to new branch | **Fixed:** 06-F1, 06-F2, 06-F3 (U3: refuses with tracked changes, never stashes). Captured source, name validation, revalidation before the first write; the old branch moves only after the new branch holds the commits (base variant: after the cherry-picks finish) |
| J. Ignored-file protection | **Fixed (J1):** 09-F1: generic history-editing starts, squash-fixups-above and compound amend/fixup flows get the ignored-file review. **Fixed (J2):** 07-F1: rebase risks add paths that directory-rename detection can create (mirrored-path rule; exact merge-tree chain when the rule finds a collision). **Fixed (J3):** J-U: the hard-reset review (`s h`, reset menu) lists untracked and ignored files it would overwrite, fingerprinted; undo/redo of a rebase and redo of a commit use it, refuse tracked changes and never stash | Generic history-editing rebases get the same ignored-file review as selected-commit rebases; rebases that may synthesize paths through directory renames must not bypass it. Needs a probe first to choose the detection method. Also J-U (from B-U): undo/redo of a rebase and redo of a commit use the captured hard reset (no autostash; refuse with tracked changes), and the hard-reset review (also `s h`) lists the untracked and ignored paths it would overwrite |
| K. Repository switch | **Fixed:** 14-F2, 14-F4 | A late `onNewRepo` failure restores git/cwd/env with the old model; Tab completion runs bounded and off the UI thread. Fixed (F4): the `s n` Tab completion runs on a worker and is applied only while the same press is latest, the prompt is open and its input is unchanged; focus moves to the suggestions when nothing completes. Fixed (F2): the new repo's config is loaded and checked (including an unknown language) before anything is published, then config and Git are published together; a failed switch puts back cwd, environment and panes; the shell hand-off directory is recorded only for a successful switch |
| L. Merge/rebase review | **Fixed:** 07-F2, 07-F3 (U4). Every merge and rebase choice runs from its menu with no second confirmation (Design: "No redundant reviews"); staged and ignored-file reviews and the squash-commit message confirmation remain. Merge squash keys: `s` squashes and commits, `u` squashes without committing |
| M. Tags `n b` | **Fixed:** 06-F5. Tags registers the captured-ref placement hook (`branch.place-at-selected-commit`), so `n b` creates or moves a branch at the tag's commit, peeling annotated tags |
| N. Docs | **Fixed:** 01-F2 (metadata-only list: six behaviour-changing files moved out; `s n` is Status-only; the `|` key is 03-F5, not 01-F2); 03-F4 (struct comments for `commandBindings`, `keybinding` and custom-command `key`; Config.md and schema regenerated); 03-F5 (Custom_DiffRenderers.md and `fork_diff_renderers.go` name `t d n`/`t d p`); 03-F6 (Guide: per-repo files are `lazygitk.yml` in the Git directory and `.lazygitk.yml` in parent directories, not the repo root); 04-F3 (already present in MergeGuide; this batch adds the screen dependency); 06-F6 and 08-F7 (**already correct**: MergeGuide distinguishes replaced and legacy checkout and move-commits flows); 09-F5 (rebase wrapper claim corrected to `PrepareInteractiveRebaseCommand` and its listed builders); 14-F5 (custom-patch and `NoCommitIndex` comments now say the index is inert); 15-F1 (counts from a recipe in MergeGuide; `upstream` remote; `misc/initial_open` path; translations already match upstream; deleted scenarios listed; metadata list corrected); 15-F5 (`// Fork:` markers at the main-view binding replacement, both menu gates and the rebase start; the custom-patch site was already marked). Snapshots: `new-features/keymap/bindings.json` and `new-features/inventory/inventory.json` regenerated. Code changes are comments only. | Correct MergeGuide (baseline counts with a counting recipe, remote names, metadata-only list, rebase-wrapper claim, checkout and move-commits claims, missing paths, translations), Guide (repo-root `.lazygitk.yml`), Config.md/schema descriptions of ignored key settings, stale `|` key, stale CommitIndex comments; add `// Fork:` markers at the replacement sites listed in 15-F5 |

## 2. UX questions for the user (answered 2026-11-06; see docs-master/fork/Tracker.md "Decisions")

- **Fixed** U1 (05-F4) Fetch in Remotes. `f` fetches the primary remote and asks
  when there are several; Remotes fetches the selected remote; "Fetch all remotes"
  is in the palette; `git.fetchAll` no longer affects `f`. Refspecs that write local
  branches are confirmed before any fetch.
- **U2 (10-F1) Selected commits hidden by a filter.** Today filtering silently
  drops hidden commits from the selection, and when none remain, commands act on
  the cursor again.
- **U3 (06-F3) Move commits to new branch with local changes.** It still stashes
  and pops automatically.
- **U4 (07-F3) Squash menu keys.** `s` squashes without committing and `S`
  squashes and commits, which breaks the case-insensitive letter rule.

## 3. Not clear bugs: proposed follow-ups (need the user's go-ahead)

Merge-ease refactors (reduce future upstream conflicts, no behaviour change):

- **Fixed** 15-F2: put fork i18n fields in one marked block, with fork values set in a
  fork-owned initializer; restore the deleted `KeybindingsMenuSectionLocal/Global`
  fields so translation JSONs match upstream again.
- **Fixed** 15-F3, 02-F4, 13-F2: move wholly fork-owned methods out of
  upstream files (draft methods in `commits_helper.go`, palette body in
  `options_menu_action.go`, hint/tip helpers in `options_map.go` and
  `command_log_panel.go`, usage handler in `keybindings.go`, operation-finished
  settlement in `merge_and_rebase_helper.go`, menu prefix setters, copy-menu
  providers in the two Files controllers).
- **Fixed** 01 merge notes: the same kind of move. `keyTip`, `commandRoute`, `localFirstForkHintBindings` and `forkHintBindingInContext` are in `fork_command_hints.go`; `focusPanelTitle` is in `controllers/fork_jump_to_side_window.go`; the prefix option overrides and blocker bindings of `menu_context.go` `GetKeybindings` are in `context/fork_menu_context.go`; the physical essential keys of `menu_panel.go` are in `fork_menu_panel.go`.
- **Fixed** 07-F4, 10-F3: keep the unused upstream merge methods and dormant paste code
  (deleting them adds conflicts) but mark them `// Fork: unreachable, replaced by …`.
- Merge probe (15): the next upstream merge conflicts in `bisect_controller.go`
  and `commit_loader.go` (upstream's `Limit` became an int); `fork_commit_scope.go`
  must be ported with it.

Other quality/perf items: 04-F1 (true-colour probe can eat keys typed during
start-up), 04-F2 (graph cache key hashes the whole history per render), 05-F3,
02-F3 (suspected popup attribution across workers), 15-F4 (restore graph
assertions weakened in some modified upstream scenarios), 13-F3 (copy tests stop
before real handlers). Found during D2: the merge/rebase helper recognises only content-conflict
messages, so a cherry-pick stopped by an add/add conflict comes back as a plain
error (selection kept; the conflict flow is not offered). Upstream behaviour.
