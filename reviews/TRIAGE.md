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
| A. Small isolated fixes | **Fixed:** 03-F1, 03-F3, 01-F4. Open: 09-F2, 09-F4, 12-F1, 13-F1, 12-F4, 06-F4, 11-F3, 03-F2 | Fork dist build must refuse a user-requested self-update; stub `forkLookPath` in the renderer-name test; guard patch-commit index -1; refresh after a failed older-commit reword; literal pathspecs for `d s` and copy-diff; refresh Files after a failed merge tool; honour `tag.forceSignAnnotated`; `w v` reachability uses the log's ref during bisect; inventory collects `SetCommandKeybinding`; show startup notices as one combined toast |
| B. Checkout | 08-F1..F6 | Palette "Checkout by name" goes through named checkout (no autostash, no plain `git checkout`); `s d`/`s p` detached path checks the active-operation guard; remote checkout checks before creating the tracking branch and keeps the chosen commit; refuse ambiguous `@{-1}`; pass `refs/remotes/...` as start point; delete the unused `CreateSelectedRefCheckoutMenu` and its tests, add tests for the real handlers |
| C. Cross-instance file safety | 02-F1, 02-F2, 14-F1, 14-F3 | One shared lock primitive (OS file lock that releases on process exit; check `vendor/` for an existing flock dependency before adding one) for `command_usage.json` and the drafts file; unique temp names; corrupt-file recovery under the lock; a draft save failure keeps the panel open and shows the error |
| D. Commit list state | 10-F2, 11-F2, 11-F1 | Clear the selection when an empty pick is auto-skipped; clearing a scope restores the cursor beyond the first 300 rows; descendant scope must not retain the whole walked history (stream/cap) |
| E. External tools | 12-F2, 12-F3 | Account for `diff.guitool`/`merge.guitool`; a configured tool Git does not know opens the chooser instead of letting Git guess |
| F. Sync | 05-F1, 05-F2, 05-F3, 05-F5 | Skip the fetched-history veto when a `pushurl` differs; offer explicit integration methods for unsupported/invalid pull policies; do not repeat full inspection on the UI thread; delete dormant stack-push and old current-pull code (push is one branch by design) |
| G. Reword target | 09-F3 | Capture the commit and HEAD at reword start; refuse if they moved |
| H. Prefix menus | 01-F1, 01-F3 | Continuation rows carry the leaf's disabled reason (Design: failed preconditions are crossed out in every menu); key labels share the dispatch traversal |
| I. Move commits to new branch | 06-F1, 06-F2 (+06-F3 after answer U3) | Validate the name and destination before any write; capture source/upstream; revalidate before mutating; no reset before the branch exists |
| J. Ignored-file protection | 09-F1, 07-F1 | Generic history-editing rebases get the same ignored-file review as selected-commit rebases; rebases that may synthesize paths through directory renames must not bypass it. Needs a probe first to choose the detection method |
| K. Repository switch | 14-F2, 14-F4 | A late `onNewRepo` failure restores git/cwd/env with the old model; Tab completion runs bounded and off the UI thread |
| L. Merge/rebase review | 07-F2 | Drop the second confirmation after an ordinary merge/rebase choice (Design: "No redundant reviews") |
| M. Tags `n b` | 06-F5 | Add `branch.place-at-selected-commit` to Tags as the Guide promises |
| N. Docs | 01-F2, 03-F4, 03-F5, 03-F6, 04-F3 (verify; MergeGuide already mentions the clipboard writer), 06-F6, 08-F7, 09-F5, 14-F5, 15-F1, 15-F5 | Correct MergeGuide (baseline counts with a counting recipe, remote names, metadata-only list, rebase-wrapper claim, checkout and move-commits claims, missing paths, translations), Guide (repo-root `.lazygitk.yml`), Config.md/schema descriptions of ignored key settings, stale `|` key, stale CommitIndex comments; add `// Fork:` markers at the replacement sites listed in 15-F5 |

## 2. UX questions for the user (answered 2026-11-06; see docs-master/fork/Tracker.md "Decisions")

- **U1 (05-F4) Fetch in Remotes.** The Guide says `f` fetches every remote from
  any pane. In Remotes, `f` currently fetches only the selected remote, and
  `git.fetchAll: false` drops `--all` everywhere.
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
- 15-F3, 02-F4, 01 merge notes, 13-F2: move wholly fork-owned methods out of
  upstream files (draft methods in `commits_helper.go`, palette body in
  `options_menu_action.go`, hint/tip helpers in `options_map.go` and
  `command_log_panel.go`, usage handler in `keybindings.go`, operation-finished
  settlement in `merge_and_rebase_helper.go`, menu prefix setters).
- 07-F4, 10-F3: keep the unused upstream merge methods and dormant paste code
  (deleting them adds conflicts) but mark them `// Fork: unreachable, replaced by …`.
- Merge probe (15): the next upstream merge conflicts in `bisect_controller.go`
  and `commit_loader.go` (upstream's `Limit` became an int); `fork_commit_scope.go`
  must be ported with it.

Other quality/perf items: 04-F1 (true-colour probe can eat keys typed during
start-up), 04-F2 (graph cache key hashes the whole history per render), 05-F3,
02-F3 (suspected popup attribution across workers), 15-F4 (restore graph
assertions weakened in some modified upstream scenarios), 13-F3 (copy tests stop
before real handlers).
