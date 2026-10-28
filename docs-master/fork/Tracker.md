# Current work

## Terminal title names the repository being viewed

lazygitk never sets the terminal title, so it keeps whatever the shell set at
launch, usually the launch directory. Agreed behaviour:

- The title is the top-level directory of the repository or worktree being
  viewed, even when lazygitk was launched from a subdirectory; inside a
  submodule it is the submodule's directory. The home directory is shown as
  `~`.
- It is set at startup and on every repository change: `s r`, `s n`,
  entering or leaving a submodule, and `s w` worktree switches.
- The terminal's title stack is saved at startup and restored on exit, so
  the shell's own title returns where the terminal supports it. Elsewhere
  the last repository path stays until the shell sets a title.

Acceptance: the title is the repository path at startup and after each kind
of repository change; it uses `~` for the home directory; exiting restores the
saved title.

Tasks:

- Design the title writer and its seams (tcell `SetTitle`, title stack save
  and restore, the repository-change hook), and present it before coding.

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
