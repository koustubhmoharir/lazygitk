# Feature plan for the fork

Status: design and implementation plan only; no application behavior changed.
Explored on 2026-10-21 at upstream commit `4e1eebe278790f257a875da6633234391a5a85b3`.

Supporting evidence: [repository map](repository-map.md) and
[Git operation investigation](git-operation-investigation.md).

## Agreed direction

- Default shortcuts express a shared verb on the selected object. Rename can
  rename a branch, commit message, or stash; refresh and rebase cannot reuse
  that key. An inapplicable shortcut does nothing.
- Disable automatic fetch, periodic file refresh, branch forwarding, and
  staging of resolved conflicts. Retain read-only external-ref polling.
- The first version supports operations on other branches that do not require
  another worktree. General off-branch merges, rebases, and cherry-picks are
  deferred; never temporarily switch the user's checkout to simulate them.
- Distinguish the selected object, the checked-out branch, and the operation's
  destination in labels and confirmations. Selection must not silently change
  while a prompt or worker is running.

## Delivery sequence

Each numbered milestone is independently reviewable. Separate preparatory
refactors from behavior changes and keep tests with the behavior they verify.
Do not implement these milestones until the design has been presented.

| Milestone | Deliverable | Depends on | Relative size |
| --- | --- | --- | --- |
| 1 | Explicit-action defaults | None | Small |
| 2 | Stable command identity and complete key/action inventory | None | Medium |
| 3 | Built-in diff target menu | 2 for command metadata | Small–medium |
| 4 | Create/move a branch without checkout | Git probe review | Medium |
| 5 | Checkout selected commit under a branch name | 4's shared prompts/validation | Medium |
| 6 | Rebase current branch onto a selected commit | 2 | Small–medium |
| 7 | Operate on selected branches without checkout | 4; explicit target design | Large |
| 8 | Consistent default shortcuts and inactive-key behavior | 2–7's action inventory | Large |
| 9 | Frequency-ordered command palette | 2; works before or after 8 | Medium |

Shortcuts are finalized after the new commands are enumerated, so new features
do not immediately force a second remapping. Stable IDs and key semantics are
related but different: two commands may share the verb `rename` while retaining
distinct IDs such as `branch.rename` and `stash.rename`.

## 1. Explicit-action defaults

Change `GetDefaultConfig`'s Git settings in `pkg/config/user_config.go` to:

```yaml
git:
  autoFetch: false
  autoRefresh: false
  autoForwardBranches: none
  autoStageResolvedConflicts: false
  autoDetectExternalChanges: true
```

These are defaults, not enforced prohibitions. Preserve explicit user/repo
overrides and existing manual fetch, refresh, fast-forward, and conflict
staging. Do not rewrite the user's config on startup. Existing files that
explicitly enable automation remain effective.

Audit startup, repository switching, configuration reload, post-fetch handling,
and return from subprocesses. Opening/switching repositories must not launch
an automatic fetch or forward branches. Manual fetch must not forward branches
with the new defaults. Normal refresh after an explicit operation still runs;
read-only external-ref polling remains enabled independently of file polling.
Document that disabling auto-staging also disables the associated automatic
continue-merge/rebase prompt, as specified by that existing config option.

Validation: default/override config tests; targeted background-routine tests;
existing fetch/auto-forward integration scenarios, with opt-in scenarios
explicitly setting their intended config. Regenerate config docs/schema.

## 2. Stable command identity and inventory

Add stable, non-translated IDs to command bindings and relevant menu leaves.
Add semantic verb metadata where needed for the keymap audit; do not use
translated descriptions, current keys, branch names, or slice position as IDs.
Retain controllers/helpers as owners of command behavior rather than replacing
them with a speculative command framework.

Inventory every default binding across files, branches, remotes, tags, local
commits, subcommits, reflog, stash, worktrees, submodules, main/secondary panes,
menus, search/filter inputs, editors, and operation-specific modes. Include
global/parent dispatch, alternate keys, hardcoded keys, and platform variants.
Record verb, target, default keys, availability, and dispatch precedence.

Unify shared invocation plumbing only where required so shortcuts and palette
execution refer to the same action. Preserve disabled reasons in menus.
Commands with no convenient default shortcut must still appear in `?`; check
that the current binding-driven palette can expose them before assigning empty
key lists. Give custom commands optional stable IDs, with a documented stable
fallback that does not persist rendered shell commands or secrets.

Validation: uniqueness/stability of built-in IDs, aliases sharing an ID, custom
commands, unbound commands in the palette, locale/keymap changes, and unchanged
handler behavior in the preparatory refactor.

## 3. Built-in diff target menu

Extend the existing Ctrl+T external-difftool action in
`BasicCommitsController`; do not implement the feature as a shipped custom
shell command. Make the action discoverable in `?` as **Diff selected commit…**
and show it in the bottom action hints when space permits.

Offer two explicit choices for a single selected commit:

1. **Diff with previous commit**: first parent → selected commit, matching the
   user's `SHA~1 SHA` command.
2. **Diff with working tree**: selected commit → working tree, matching the
   user's single-revision `git difftool -d SHA` command.

Use the existing argument-vector command builder, configured external difftool,
directory-diff mode, and terminal subprocess lifecycle. An explicit menu choice
sets its endpoints independently of sticky diff mode. Preserve access to the
current sticky-mode comparison where needed, with a distinct label.
Working-tree comparison follows Git's tracked-file behavior, including staged
and unstaged contents; it does not promise to include untracked files.

For merge commits, label the previous-commit choice as first parent. For a root
commit, disable that choice with an explanation; working-tree diff still works.
Do not treat rebase placeholders or empty hashes as real selectable commits.
Support local commits, subcommits, and reflog via the shared controller. In
other panes, the same diff verb operates only on a supported selected target;
it must not fall back to an unrelated current-branch operation.

Validation: exact argv for both endpoints, root/merge commits, filenames with
spaces, sticky diff mode, no selection, submenu/tooltip discovery, subprocess
exit/failure and refresh. Use a fake difftool in tests; no GUI dependency.

## 4. Create or move a branch at a selected commit, without checkout

Add **Create/move branch here…** to every supported commit-list context. Keep
**Create branch and check it out…** as a separate action so the distinction is
visible. Reuse prompts/branch-name suggestions and existing `NewWithoutCheckout`
where appropriate, while preserving their distinct checkout semantics.

Prompt for a local branch name and capture the selected commit's full OID.
Validate with Git's branch-name rules and pass arguments without shell
interpolation. A new name creates a ref only. An existing name at that OID is
a no-op. An existing name elsewhere requires a warning showing branch name,
old/new abbreviated OIDs, and that its tip will move; cancellation changes
nothing. Preserve existing upstream metadata.

Refuse moving a branch checked out in another worktree. If the requested name
is the current branch, explain that moving its tip also moves HEAD and changes
what the unchanged index/worktree are compared against. Proposed support uses
an explicit soft-reset path, with its own confirmation and in-progress-operation
guards; it must not bypass Git's protection using a raw `update-ref`. Recheck
worktree occupancy and ref state immediately before the write. If the ref
moved after confirmation, require a new confirmation rather
than overwriting unseen changes. Use expected-old-OID ref updates where safe,
without losing worktree guards or branch/reflog semantics.

For a non-current destination, success leaves HEAD, index, files, and current
context unchanged; for the current branch, only its tip/HEAD commit changes.
Refresh branch markers and retain the selected commit. Include a useful
reflog reason for a move. Do not claim the existing undo system supports this
until verified.

Validation: create, move, already-at-target, warning cancel, invalid/conflicting
names, detached HEAD, dirty index/worktree, occupied worktrees, concurrent ref
changes, upstream retention, and recovery through reflog.

## 5. Checkout any commit under a supplied branch name

Extend `RefsHelper.CreateCheckoutMenu` with **Checkout as branch…**. Prompt for
a new or existing local name. If the name already points elsewhere, warn about
moving it before attempting checkout. This is a different operation from
moving a ref without checkout.

Use Git's combined create/reset-and-checkout operation (`checkout -B` or a
verified equivalent), never `branch -f` followed by checkout. Do not use
`--force`, hard reset, an unconditional clean-tree guard, or automatic stash.
Carry unrelated tracked/staged/untracked changes when Git can preserve them;
report Git's overwrite error when it cannot. Preserve ignored files too by
using `--no-overwrite-ignore`; the Git probes confirmed it prevents overwriting
an ignored collision that normal checkout would overwrite.
Retain the user’s changes and branch position on a rejected checkout.

A branch checked out in another worktree, invalid name/revision, an unresolved
operation, locking failure, or a failed hook can also prevent completion.
“Fail only for overwritten dirty files” means no unnecessary dirty-tree
restriction; it cannot bypass these independent Git constraints. Report the
actual reason. A user hook can have side effects after checkout; do not promise
rollback of arbitrary external programs.

Permit an existing destination that is the current branch if Git can perform
the safe combined operation. Distinguish force-moving a branch name from
force-overwriting files in all wording. Resolve the selected commit once,
revalidate the destination after confirmation, and refresh HEAD/branch/files/
reflog/worktree models together after success. Keep existing checkout/autostash
flows separate unless the user explicitly asks to change their behavior.

Validation: genuinely different source and target trees, new/existing/current
destination, unrelated and overlapping staged/unstaged changes, untracked and
ignored collisions, deleted files, detached HEAD, other-worktree occupancy,
invalid input, canceled confirmation, and rejected checkout leaving destination
ref/HEAD/index/file contents unchanged. Cover sparse checkout/submodules to the
extent supported by existing checkout behavior; do not silently force them.

## 6. Rebase the current branch onto a selected commit

Add **Rebase current branch onto this commit…** in `BasicCommitsController`,
including subcommits and reflog so a commit outside the current history can be
selected. Reuse `MergeAndRebaseHelper` and the rebase command machinery, making
the source current branch and destination full OID explicit in the prompt.

The ordinary operation is `git rebase <selected-OID>` on the current branch.
This is distinct from editing commits after the selection and from choosing an
old base for `git rebase --onto`. Keep the latter as an explicit advanced
choice; an existing marked base must not silently change this command's
meaning. An ancestor may legitimately produce no change.

Retain continue/skip/abort and conflict recovery. Reject missing commits,
detached/unborn HEAD, and incompatible in-progress operations. Explain history
rewrites before starting. Reusing the current helper requires auditing its
`--autostash`, `--rebase-merges`, and Git `rebase.updateRefs` behavior: do not
silently stash dirt or rewrite additional branches in a command presented as
affecting one branch. Proposed default: require a clean index/worktree for this
new action and suppress updates to other branch refs; surface this choice for
review before implementation.

Validation: divergent/ancestor/descendant targets, selected commit on another
branch or in reflog, root commit, no-op, dirty state, conflicts with continue
and abort, detached HEAD, marked-base isolation, and `rebase.updateRefs=true`.

## 7. Selected-branch operations without checkout

Replace implicit-current-branch assumptions with explicit targets in the
operations that support it. Never infer the selected branch from whichever
model happens to be current when a worker finally runs.

| Operation | First-version behavior |
| --- | --- |
| Rename/delete, upstream settings, log/diff, tag/create branch | Audit and retain selected-ref support already present |
| Push | Push the selected local ref to its explicit remote/refspec; preserve upstream, force-with-lease and protection rules |
| Fast-forward/update from upstream | Reuse selected-branch forwarding; explicit fetch, ancestry check, ref update without checkout for an unoccupied branch |
| Move branch tip to selected commit | Use milestone 4; for a non-current branch there are no soft/mixed/hard worktree reset modes |
| Pull on an unoccupied branch | Fetch its upstream and allow a verified fast-forward; report divergence with an explicit unsupported/off-branch-integration explanation |
| Merge/rebase/cherry-pick into an unoccupied branch | Deferred when replay, merge commits, or conflicts require a worktree; offer an explicit checkout route, never switch silently |
| Merge/rebase using selected branch as source | Retain clearly named “Merge selected branch into current…” and “Rebase current onto selected…” actions |
| Current-branch operations | Retain the full existing Git workflow with an explicit current-branch target |

Version one may support a merge that is provably a fast-forward through the
same safe ref-update path, but must label it accurately. General off-branch
integration is a later project, not hidden within the selected-branch audit.

In a local-branches pane, push/pull act on selection. Proposed behavior outside
a branch-bearing context: use an explicitly labeled current-branch action,
rather than an old background branch selection. A remote-tracking branch is
not a writable local destination; require selecting/creating a local branch.
For a branch occupied by another worktree, allow ref-only metadata operations
where Git supports them; block tip-changing operations in the first version.
Do not reuse forwarding paths that mutate another worktree without making
that scope explicit.

Split sync helpers so remote resolution, divergence, prompts, refspecs,
upstream-setting, status, and refresh all use the same explicit branch.
Audit stacked-branch push/pull menus: changing a selected branch must not
implicitly push or update branches below it. Preserve those as explicit choices.

Validation: local fake remote; selected != current, no upstream, gone upstream,
different push/pull remotes, force-with-lease rejection, fast-forward/diverged
and rewritten upstream, occupied worktree, concurrent ref changes, and branch
stacks. Assert current HEAD/index/worktree and unrelated refs remain unchanged.

## 8. Consistent default keys and no-op behavior

Use milestone 2's complete inventory to produce and review a verb-to-key table
before changing defaults. Keep familiar keys only where they can be consistent.
Starting candidates are `?` help, `r` rename, `R` refresh, `n` new,
`d` delete, and Ctrl+T external diff; these are candidates, not a complete keymap.
Rebase, fetch, fast-forward, fixup, checkout, staging, and browser-opening need
distinct allocations. Give uncommon commands no default shortcut if needed;
keep them reachable in the palette. Audit collisions among global and local
actions, not just within one view.

Known changes include splitting `r` (files refresh / branch rebase / commit
rename), `f` (fetch / fast-forward / fixup), and `c` (commit / checkout by name /
fixup message / restore historical file). Space currently means staging,
checkout, or menu selection; that also needs an explicit semantic decision.
`n`/`N` are new/search-next/diff-file-navigation collisions requiring resolution.

For a reserved action key whose verb is inapplicable, consume the event with no
side effect. Do not fall through to an unrelated parent/global binding, show
an error toast, or increment usage. Palette entries may remain disabled with
an explanatory tooltip. Keep genuine operation errors visible after invocation.
Distinguish inapplicable commands from actionable precondition failures such
as an overwrite conflict.

Proposed exception: text entry, search/filter input, and external subprocesses
retain normal typing. Menus retain explicit navigation/confirm/cancel controls.
Search-match and operation-specific modes must not repurpose reserved action
keys to unrelated commands. Review whether navigation/selection should be
shared verbs or have pane-specific meaning before final allocation.

Explicit user key overrides/custom commands remain possible; the cross-pane
contract applies to the fork's defaults. Document overrides that recreate
collisions. Remove the old Ctrl+T custom command locally when adopting the
built-in menu, since it currently overrides the builtin in that context.

Validation: a default key × pane/mode matrix, including aliases and platform
variants. Verify both dispatch and side effects, disabled-key no-ops, no global
fallback, menus/filtering/editors, custom overrides, and generated cheatsheets.
Provide a migration table for changed defaults in `docs-master/`.

## 9. Frequency-ordered `?` palette

Sort discoverable actions by local usage counts using stable command IDs.
Proposed policy: global per-user lifetime counts, shared across repositories;
the palette still shows actions relevant to the current context. Sort the
whole actionable list by count descending, with original declaration order as
a deterministic tie-breaker. Retain Local/Global/Navigation as annotations or
filters rather than fixed groups that prevent global frequency ordering.
Keep the displayed order stable while the palette is open.

Record direct shortcuts and palette execution consistently and once per
accepted action. Do not count typing, navigation repeats, opening `?`, disabled
actions, or canceled confirmations. Count distinct diff-menu choices separately;
opening the parent menu should not double-count the leaf. Once an operation is
accepted and starts, it counts even if Git later fails, so asynchronous success
does not change the definition. Custom commands use their stable ID policy.

Persist only IDs/counts, never branch names, selected hashes, commands, or
arguments. Use the app-state facilities or a dedicated state file after
reviewing write frequency and multiple-process behavior. Batch writes and
flush on normal exit; use atomic writes and a merge/lock strategy so separate
lazygit processes do not clobber each other's counts. Corrupt/unwritable state
must not prevent startup or command execution. Include reset usage and an
option to restore static ordering; no network telemetry.

Validation: direct-key/menu parity, alias sharing, cancellations, failures,
restart persistence, deterministic ties, locale/keymap changes, custom commands,
filtering, disabled items, multiple processes, corrupt/unwritable state, and
stable selection/order while open.

## Verification and repository workflow

During implementation use narrow package tests and named integration scenarios
for each milestone. Run `just generate` whenever keybindings/config/test lists
change, and review generated `docs-master/` and `schema-master/` outputs.
Edit only English translation sources; leave `docs/` for the release process.
Run repository-required formatting, build, unit tests and lint before completed
implementation commits, and affected end-to-end tests for Git/UI changes.

The environment currently has no `just` executable. No dependency installation
is needed for this planning task. For implementation, install `just` or use the
equivalent existing recipe commands and report that choice. Clear `NO_COLOR`
when running color-sensitive tests (`env -u NO_COLOR just unit-test`); the
initial environment check found those fail when colors are globally disabled.

All mutating Git tests use disposable repositories and local fake remotes.
External diff tests use a fake executable. Do not modify the development
checkout's branches to demonstrate a proposed feature.

Planning verification: formatting check, `go build ./...`, and
`env -u NO_COLOR go test ./... -short` passed. The equivalent lint recipe was
attempted but could not download its pinned golangci-lint module because the
connection to `sum.golang.org:443` was refused. No Go sources changed and no
system dependencies were installed. End-to-end feature tests remain future
implementation work; the disposable Git probes are recorded separately.

## Remaining review questions

1. May text-entry contexts and menu confirmation/navigation follow the normal
   modal conventions described above? The full key allocation, including the
   meaning of Space, is a separate review artifact before milestone 8.
2. Should “create/move branch here” allow naming the current branch through
   the explicitly warned soft-reset path, preserving the index/worktree but
   changing their comparison base? Moving another worktree's branch is blocked.
3. For the new “rebase current onto commit” action, is the proposed clean-tree,
   current-branch-only default right, or should it offer explicit opt-ins for
   autostash and updating stacked branch refs?
4. Does the proposed global lifetime-frequency policy, flattening of palette
   groups, and counting of accepted actions match the desired ordering?

Decisions already received are recorded under “Agreed direction”; these
remaining questions need not block the first independent milestone. This is
an extensible backlog: add further features here with target semantics,
repository evidence, dependencies, and acceptance tests.
