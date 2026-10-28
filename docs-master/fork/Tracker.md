# Current work

## 24. Multi-select and selection filters in Commits

Status: in progress.

### Agreed behaviour

**Shift+J / Shift+K.** In every list, Shift+J and Shift+K do what Shift+Down
and Shift+Up do. This is the one exception to case-insensitive letters: other
shifted letters still fold to their lowercase key.

**Selection modes in Commits.** Commits has two selection modes.

- *Single-select* (the default): the cursor commit, or the current Shift
  range, is the selection, as today.
- *Multi-select*: the selection is an explicit set of commits, and the cursor
  moves without changing it. Space toggles the cursor commit in the set, and
  on a range it selects or deselects the whole range. Space enters
  multi-select mode from single-select mode.
- Shift+movement always enters multi-select mode. The commits swept by
  consecutive Shift moves, from where the run started to the cursor, are added
  to the set; moving back within the run shrinks that range, as with a Shift
  range today. A plain move ends the run and keeps the swept commits selected.
  The sticky range toggle (`t h r`) starts a run that lasts until it is
  toggled off.
- A mouse click moves the cursor, like a plain move.
- When the set becomes empty, the pane returns to single-select mode.
- In multi-select mode, selected commits keep the selection background and
  the cursor is drawn separately: a marker at the left edge of the cursor line,
  with the inactive-selection background when the cursor commit is not
  selected.
- Space no longer inspects in Commits; `v` does. Reflog and Sub-commits keep
  Space as inspect.
- The set is stored by commit hash. It survives refreshes and filtering;
  commits that no longer exist (for example after a rebase) drop out.

**Commands with a multi-selection.** Commands that accept any set of commits
use the set: the filters below, `c y`/`c x` cherry-pick mark and unmark, `c p`,
`g c` combine and the copy keys that take several commits. Commands that need
one commit or a contiguous range are disabled with the reason "Clear the
selection first (Escape)". Nothing silently falls back to the cursor commit.

**Filter group `w`.** `w` becomes the prefix group for filters. Three new
filters hide everything but part of the graph:

| Keys | Filter |
| --- | --- |
| `w a` | The selected commits and their ancestors |
| `w d` | The selected commits and their descendants (commits reachable from any ref that have a selected commit as an ancestor) |
| `w b` | A local branch, its tracking branch, and every ancestor of either (their common history included) |

- `w a` and `w d` use the multi-selection, or the single-select selection when
  there is no set.
- `w b` is offered in Branches on a local branch and in Commits on a commit that
  a local branch points at; if several do, a menu asks which branch. A branch
  without a tracking branch is shown alone, and the filter status says so.
  From Branches, it focuses Commits with the cursor on the branch tip.
- Choosing a filter while filtered replaces the filter.
- The existing filters move into the group: `t f s` (Files status filter)
  becomes `w s`, and `t f o` (log filter options) becomes `w o`. `t` keeps
  view and tools only. `/` stays the pane's text filter or search.
- The status bar names the active filter (for example "Ancestors of 3
  selected commits"), like upstream's path filter.
- Escape in a filtered Commits pane shows all commits again and keeps the
  selection; the next Escape clears the selection; the next clears the
  cherry-pick marks. Escape still cancels a single-select Shift range first.
- The multi-selection and the cursor commit are kept when they stay visible,
  which `w a` and `w d` guarantee for the selected commits.
- The branch filter resolves the branch names on each refresh, so fetched
  commits appear. The commit filters keep their hashes.
- While filtered, history-editing commands are disabled as in upstream's path
  filter mode, since the list no longer shows the true branch.

### Tasks

- Multi-select model in the Commits list: set by hash, Shift runs, Space
  toggle, rendering of selection and cursor, Escape order, refresh retention.
- Command scoping: set-aware commands use the set; the others are disabled
  with the reason.
- `w a`/`w d` keep the multi-selection once it exists (today a Shift range
  collapses to the cursor commit when a filter is applied).
- Integration scenarios for each of the above.
- Guide, Design and MergeGuide updates.

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

`just` is unavailable; use the equivalent commands in `justfile`. Clear
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
