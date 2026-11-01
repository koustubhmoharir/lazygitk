# Current work

## Palette follow-ups

- Popups (menu, prompt, confirmation, search, suggestions, commit message)
  get no section in other panes' palettes; they list their commands only
  while focused.
- Palette integration tests reach commands by filtering, not by scrolling
  about 300 lines, and open the palette once where they only read it.
- The options bar reuses the bindings built by the last keybinding reset
  instead of rebuilding and projecting every binding on each frame (about
  half of each frame's CPU). Dispatch already uses those bindings.

Tasks: none remain; the ui and main_view groups are verified next.

## Next: repo switch feedback

Agreed: on a repo switch every pane clears at once and shows "Loading…" until
its own data arrives, so old content never appears under the new repo; a
toast "Opened <repo>" appears once all panes have loaded. TortoiseGit has no
in-app repo switch to follow.

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
