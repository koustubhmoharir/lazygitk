# Current work: readable menus and summaries

Coloured prompts help, but summary prompts are still hard to scan and menu
actions look like prompt text. Agreed behaviour (2026-11-28):

- In prompts made of `name: value` lines, the name and its colon are blue
  (the options bar's colour, `promptmark.Label`). Values keep their colours
  and lose their quotes. Every such line has a colon ("Source: NAV at
  5d59a479").
- A menu item that has no key of its own gets the next free digit 1-9,
  which runs it. Digits another item uses are skipped. Filtering menus
  (palette, repo switcher, file pickers) and prefix menus get no digits.
  Cancel gets none.
- Values inside menu item labels (hashes, branches, paths, as in "Mark
  5d59a479 as bad" or "Filter by path/file.go") are coloured like prompt
  values and lose their quotes, in fork and upstream menus.
- Every menu item with a key shows a cyan `→` between the key and the label,
  upstream menus included.
- Toggles in the push menu are checkboxes (`[✓]`, `[ ]`) instead of a
  "(false)" suffix, and the prompt no longer repeats their values.
- An item that opens another prompt and already ends in "…" gets no extra
  "...".

## Tasks

3. Digit keys and the `→` column in menus.
4. Coloured values in menu item labels.
5. User guide, design principles and merge guide.

## Open follow-ups

- Untracked files saved with `git stash -u` (the stash's third parent) appear
  in neither Stash listing (`Enter`/`v d`, `v f`). This predates the work
  above. Showing them would change Stash's changed-files listing; present
  the choice to the user before implementing.

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
