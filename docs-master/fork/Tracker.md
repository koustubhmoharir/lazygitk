# Current work

## Flat checkout keys under `s`

`s c` opened a second menu (switch, check out under a name, detached HEAD),
and `s b` (check out by name) overlapped with its naming item. Agreed
behaviour:

| Keys | Selection | Behaviour |
| --- | --- | --- |
| `s c` | Local branch | Switch to it through the safe named-checkout path; no menu |
| `s c` | Remote branch | Switch to the local branch of that name if it exists; otherwise create it tracking the remote branch, then switch |
| `s c` | Commit, tag, reflog or sub-commit entry | The named-checkout prompt for that commit; suggestions list branches already at the commit first; a new name is created there, an existing name elsewhere gets the move warning |
| `s d` | Anything with a commit | Detached HEAD at it, in every pane that has `s c` |
| `s b` | | No key; "Checkout by name" stays in `?` |
| `s p` | | Unchanged, but through the safe checkout (never offers a stash); a detached previous checkout is checked out detached |

- In Status, `s` is only for repositories: `s r` and `s n` have keys only in
  Status, and `s p` does not appear there. Elsewhere `s` switches commits and
  branches (and `s w` worktrees, in Worktrees).
- The old second-level checkout menus and their numbered branch items have no
  key route.

Acceptance: each row above works under the real defaults with no extra menu;
`s p` with blocking changes shows Git's refusal, not a stash offer; `s b`
does nothing; in Status the `s` menu lists only `r` and `n`; in other panes
it has no `r`, `n`; generated key docs and the user guide match.

Tasks:

- Integration scenarios for the acceptance cases.
- User guide, merge guide, generated docs; close this section.

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
