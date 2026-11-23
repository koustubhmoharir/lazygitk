# Current work: coloured confirmation prompts

Confirmation popups are hard to read: the whole prompt is bold and the values
in it (branch names, hashes, paths, keys) are plain text inside long
sentences. Agreed behaviour (2026-11-23):

- The prompt sentence is regular weight. Values are coloured as the panes
  colour them: local branch cyan bold (branch heads in Commits); remote and
  remote branch yellow (upstream in Branches); tag magenta bold; hash yellow;
  path bold; key in the options bar's key colour; count bold.
- Destructive or irreversible consequences ("is not fully merged", "cannot
  be undone", "force push") are bold red.
- Error alerts keep their existing red bold.

Design: strings carry role markers (Go helpers that wrap text in private-use
delimiters with a role). The popup and menu-prompt code resolves them to theme
styles just before display, before the height is measured. Upstream call sites
stay untouched; upstream English strings are marked through overrides in
`fork_english.go` and recorded in the merge guide.

## Tasks

2. Mark the fork's confirmation prompts (fork strings and fork call sites).
3. Mark the upstream confirmation prompts through `fork_english.go`
   overrides.
4. Mark the review and decision menu prompts.
5. User guide, design principles and merge guide.

## Open follow-ups

None.

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
