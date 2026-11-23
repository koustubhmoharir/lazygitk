# Current work: highlighted file contents

Agreed with the user on 2026-11-24. Yazi highlights in-process (syntect);
the Go equivalent is chroma, which ships Catppuccin styles.

## Behaviour

- File contents shown in the main view by the full listing (`v f`) are
  syntax highlighted by a built-in highlighter (chroma), with the Catppuccin
  style matching the app's theme (Mocha on a dark terminal, Latte on a light
  one) and line numbers. Nothing needs installing.
- `git.fileRenderers` configures the renderers, like `git.diffRenderers`:
  a list whose entries are the built-in highlighter, plain text, or a command
  that reads the content on stdin (such as `bat --color=always --paging=never
  --style=numbers --file-name={{filename}}`). Unset or empty means built-in,
  then plain. A configured list is used as given.
- `t d n` / `t d p` cycle the file renderers while the main view shows file
  contents (and the diff renderers otherwise), with the same notification.
- The language comes from the file name; an unknown type is plain text.
  Files over 1 MiB are not highlighted by the built-in highlighter (it holds
  the whole file in memory); they stream as plain text and the title says why.
  Binary files keep their note and never reach a renderer.
- A failing command shows its error in the main view, as a failing diff
  renderer does; there is no notice when no command is configured.

## Tasks

1. Config, renderer cycling, the built-in highlighter and plain text, with
   tests.
2. Command renderers (`{{filename}}`, stdin), with tests.
3. Guide, design and merge-guide updates; close this section.

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
