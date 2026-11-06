# Current work: fix the fork review findings

Source: [reviews/TRIAGE.md](../../reviews/TRIAGE.md) (batch letters and finding
ids `NN-Fk` refer to it). Confirm each finding before fixing it, with a
failing test where practical. Mark each finding fixed or declined in the
triage as its batch lands.

## Decisions (2026-11-06)

- **U1 Fetch.** Outside Remotes, `f` fetches the primary remote: `origin`, or
  the repository's only remote. With several remotes and none named `origin`,
  `f` opens a menu of the remotes plus "Fetch all remotes". In Remotes, `f`
  fetches the selected remote. "Fetch all remotes" is a palette item.
  `git.fetchAll` no longer affects `f`. Before any fetch, the remotes' fetch
  refspecs are checked; if one writes into local branches (`refs/heads/`), a
  confirmation names the refspecs and explains that Git will move those
  branches. The fetch is selected; Cancel and Escape stop it.
- **U2 Hidden selected commits.** Commits hidden by a filter or beyond the
  loaded list stay selected; selection mode does not change. The count
  includes hidden commits and commands act on the whole set.
- **U2a Rewritten selected commits** (2026-11-06). Selected commits that a
  refresh finds unreachable from every ref (rewritten outside the selection)
  stay selected and counted, and a notice names them. Copy and cherry-pick
  still use them; commands that edit the current branch's history refuse and
  name the selected commits that are no longer in it.
- **Descendant scope** (11-F1, 2026-11-06) follows the normal commit limit:
  the newest 300 descendants, loading more on scroll.
- **U3 Move commits to a new branch** refuses while tracked files have staged
  or unstaged changes; it never stashes.
- **U4 Merge menu.** `s` squash-merges and commits; `u` squash-merges without
  committing.
- **Merge-ease cleanups** in triage §3 (first group) are approved. The other
  §3 quality items stay open.

## Tasks

Each task is one subagent batch, reviewed and committed before the next.

- Batch D: commit list state.
  D2: empty-pick auto-skip cleanup (10-F2). D3: scope clear beyond 300 rows
  (11-F2). D4: limited descendant scope (11-F1).
- Batch P: fetch per U1 (05-F4).
- Batch F: sync.
- Batches E, H, K, M, and L with U4 (07-F2, 07-F3).
- O2/O3: move fork-owned methods out of upstream files; `// Fork:
  unreachable` markers (15-F3, 02-F4, 13-F2, 07-F4, 10-F3).
- Batch N: remaining doc corrections, including MergeGuide counts after O2.
- Full integration suite.

## How this tracker works

- This file holds the current piece of work: its agreed behaviour, acceptance
  criteria and remaining tasks. New work starts by writing that section here
  and reviewing it with the user. If many items are planned at once, create a
  separate roadmap for them at that point.
- Plan tasks for the current work only. Work through them in reviewable
  increments; a task may span commits. Remove each task in the commit that
  completes it, and do not keep a completed-task log: Git history records it.
- When the work is finished, move its lasting decisions into the
  [user guide](Guide.md) (keys and behaviour) and the
  [design principles](Design.md) (rules), then replace its section here with
  the next work, or say explicitly that there is none. Do not leave a stale
  section.
- Keep the tracker accurate when tasks change, record concrete blockers against
  the affected task, and present unresolved user choices in a modal flow with
  sensible defaults. Do not mark unfinished work complete.

## Verification notes

`just` is unavailable; use the equivalent commands in `justfile`. Test times
depend heavily on what else the machine is running: `branch/delete` took 7.5s
to 30s alone. Use the `-timeout` values in `justfile`, because the unit tests can
exceed Go's default of 10 minutes per package. Clear
`NO_COLOR` for color-sensitive tests. Headless invocations share
`/tmp/lazygit/test_lazygit` and fixture directories: serialize their builds and
runs. Use bounded eventual checks for background events, not immediate UI
assertions or synthetic refreshes. Use verbose named terminal runs and verify
the expected scenario names actually ran; a successful exit with "no tests to
run" is not coverage. Test names are `TestIntegration/<group>/<scenario>`.
Finish generation successfully before starting headless checks, and keep
integration-test sources frozen during the run. Each scenario variable must be
its filename converted from snake_case to PascalCase, using unqualified
`NewIntegrationTest`; compile-only checks do not validate those generator rules.
The shared integration config explicitly disables fetch/file/ref polling;
default-policy tests must restore values from the real default constructor.
