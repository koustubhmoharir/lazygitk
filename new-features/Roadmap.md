# Roadmap for the fork

Status: implementation underway; see the tracker for current work.
Explored on 2026-10-21 at upstream commit `4e1eebe278790f257a875da6633234391a5a85b3`.

Current item and its remaining tasks: [Tracker.md](Tracker.md).
Upstream integration notes: [MergeGuide.md](MergeGuide.md).

Supporting evidence: [repository map](repository-map.md) and
[Git operation investigation](git-operation-investigation.md), plus
[file comparison probes](file-diff-investigation.md).

## Agreed direction

- Default shortcuts express a shared verb on the selected object. Rename can
  rename a branch, commit message, or stash; refresh and rebase cannot reuse
  that key. An inapplicable shortcut does nothing.
- Alphabetic command bindings are case-insensitive. Core defaults do not use
  Ctrl. Two- or three-letter bindings use menus on the first key, with nested
  menus if needed; no new key-chord engine is required. Text entry and menu
  navigation retain normal modal behavior.
- Previous-commit diff and working-tree diff are separate actions. The latest
  keymap uses separate leaves `d p` and `d w` in the visible diff group;
  `d c` adds comparison to a commit specified by pasted partial/full hash. Working-tree diff gets priority for both a whole commit
  and one historical file. A third action compares a historical file with an
  explicitly chosen working-directory file, regardless of rename detection.
- Disable automatic fetch, periodic file refresh, branch forwarding, and
  staging of resolved conflicts. Retain read-only external-ref polling.
- The first version supports operations on other branches that do not require
  another worktree. General off-branch merges, rebases, and cherry-picks are
  deferred; never temporarily switch the user's checkout to simulate them.
- Preserve upstream behavior outside the requested feature changes. In
  particular, cherry-picking while an interactive rebase is paused remains
  supported; do not impose a new blanket active-operation restriction.
- Distinguish the selected object, the checked-out branch, and the operation's
  destination in labels and confirmations. Selection must not silently change
  while a prompt or worker is running.
- Rebase requires no staged or unstaged tracked changes; harmless untracked
  files are allowed, with Git rejecting collisions. The user commits changes
  instead of stashing. The new workflow must not invoke or offer autostash.
- Keep existing reset for current-branch pointer movement. Named checkout must
  carry dirty changes without stashing whenever Git can avoid overwriting them;
  moving a non-current branch is only a ref change.
- Palette usage counts are shared across repositories.
- Pull is intentionally less prominent: no single-letter default binding;
  keep it reachable through a multi-key menu path and the command palette.

## Delivery sequence

Each numbered milestone is independently reviewable. Separate preparatory
refactors from behavior changes and keep tests with the behavior they verify.
The tracker contains task-level planning for the current item only; plan later
items when they become current. On completing an item, mark it complete here,
update its description to match the implementation, and replace the tracker's
tasks with the next item's plan in the same completion commit.

| Item | Deliverable | Status | Depends on | Relative size |
| --- | --- | --- | --- | --- |
| 1 | Explicit-action defaults | Complete | None | Small |
| 2 | Stable command identity and complete key/action inventory | Complete | None | Medium |
| 3 | Separate commit/file diff actions and arbitrary-file comparison | Complete | 2 for command metadata | Medium |
| 4 | Create/move a branch without checkout | Complete | Git probe review | Medium |
| 5 | Checkout selected commit under a branch name | Complete | 4's shared prompts/validation | Medium |
| 6 | Rebase current branch onto a selected commit | Complete | 2 | Small–medium |
| 7 | Operate on selected branches without checkout | Complete | 4; explicit target design | Large |
| 8 | Consistent default shortcuts and inactive-key behavior | Complete | 2–7's action inventory | Large |
| 9 | Frequency-ordered command palette | Complete | 2; works before or after 8 | Medium |
| 10 | Separate configuration and state from upstream lazygit | Complete | None | Small |
| 11 | Push without a redundant review | Complete | 7 | Small |
| 12 | One flat copy menu with consistent keys | Complete | 2, 8 | Medium |
| 13 | Fixed keybindings: key configuration is ignored | Planned | 8, 12 | Medium |

## Fork implementation principles

Follow the root [AGENTS.md](../AGENTS.md). Prefer additive fork modules and
small, explicit integration points so upstream merges remain feasible. When a
fork path replaces an upstream path, explain the replacement at the integration
point and record it in the merge guide. Correctness and maintainability take
priority over reducing the diff. This fork exists to improve usability: present
meaningful choices through modal flows with sensible defaults and leave the
user in control. Consult TortoiseGit when design or implementation questions
arise; raise unresolved choices instead of silently deciding for the user.

Shortcuts are finalized after the new commands are enumerated, so new features
do not immediately force a second remapping. Stable IDs and key semantics are
related but different: two commands may share the verb `rename` while retaining
distinct IDs such as `branch.rename` and `stash.rename`.

## 1. Explicit-action defaults

Complete (2026-10-21). The implementation uses one shared constructor hook;
existing automation handlers and manual actions remain unchanged.

The fork overlay in `pkg/config/fork_defaults.go`, called from the shared
platform config constructor before loading user configuration, sets:

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

Verified: debug build, isolated-config TUI start/quit and printed defaults,
full short tests, formatting, generation and lint (zero issues). The full
690-scenario headless run found two test-fixture issues (one missing staging
opt-in and the new test inheriting the harness's disabled ref polling); both
were corrected. An 11-scenario headless rerun, including both corrected tests,
both new runtime scenarios and all seven forwarding scenarios, passed. The
full suite was not repeated after those fixture-only corrections; the
version-gated `sync/push_no_follow_tags` scenario was skipped by the harness.

## 2. Stable command identity and inventory

Complete (2026-10-21). Explicit metadata covers 312 binding and 176 menu
factories; five parameter/informational factories intentionally have no
independent action ID. See [the dispatch/semantic audit](command-identity-design.md)
and [mechanical inventory](inventory/README.md). Mode-dependent staging/patch/
conflict operations resolve identity live, without changing handlers.

Palette aliases now merge by resolved ID, preserving local precedence and
combining shortcut hints on copies. Distinct equal-label actions stay separate;
unbound described actions remain executable. Hidden positional panel-jump
definitions survive removed shortcuts, without adding panel-chooser entries.
Controllers/helpers remain behavior owners; no frequency recording is added.

Verified: debug build, full short tests, standalone inventory/checker tests,
generation, formatting, lint (zero issues) and 14 targeted headless scenarios
(including unbound branch rename through its normal prompt and Git result).
No new full-suite run was performed for this item.

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

## 3. Separate diff actions for commits and historical files

Complete (2026-10-21). The fork adds an independent historical-diff backend
and two controller hooks; existing sticky comparisons retain their handlers
under a distinct label. See [implementation design](direct-diff-design.md) and
[usage](../docs-master/Fork_Features.md).

Working-tree comparison uses `w/W`, previous-commit comparison uses `v/V`,
and explicit-file comparison is initially unbound but executable from `?`.
Colliding upstream defaults are unbound, not removed; displaced operations
remain in the palette until the full keymap milestone. Palette guidance replaces
empty paste-key hints. Working-tree comparison is prioritized in display hints
without changing dispatch order. Explicit configured bindings remain effective.

Verified: debug build, full short tests, standalone inventory/checker tests,
generation, formatting and lint (zero issues). The 99-scenario affected-operation
headless run and final 15-scenario palette/diff run passed. Real Git/fake-tool
tests cover exact bytes, literal and Unicode paths, empty/large binary blobs,
root/merge/rename/deletion behavior, captured selection, staged/index preservation,
tool failure/recovery and cleanup. Verification ran on Linux/Git 2.51.1; no new
full-suite run or external GUI-tool matrix was performed. Detached custom tools
must use their wait mode; unsupported symlink/gitlink comparisons report a reason.

### Direct previous-commit and working-tree comparisons

Extend the existing external-difftool machinery in `BasicCommitsController`
and `CommitFilesController` with distinct command IDs, `?` entries and bindings
for **Diff with previous commit** and **Diff with working tree**. Do not put a
shared target-choice prompt after the complete shortcut. The delivered
provisional defaults are `w` / `v`; the user's final-keymap proposal in item 8
replaces these with distinct `d w` / `d p` paths in commit and historical-file
lists and adds `d c` for a pasted commit hash. Uppercase invokes the same actions.
The fork overlay clears their colliding defaults. No new Ctrl binding is added;
the remaining legacy Ctrl bindings are addressed by the full keymap milestone.

| Selected object | Previous-commit action | Working-tree action |
| --- | --- | --- |
| Single commit (local, subcommits, reflog) | First parent → selected commit, directory diff | Selected commit → working tree, directory diff |
| File of a single selected commit | First-parent version → selected version | Selected historical version → same-path working file |
| Directory in a historical file tree | Same endpoints, scoped to directory | Selected commit → working tree, scoped to directory |

Use the configured external difftool, argument-vector command builder and
terminal subprocess lifecycle. These actions set endpoints independently of
sticky diff mode; retain the old sticky comparison under a distinct action.
Whole-commit comparison follows normal Git tracked-file semantics, including
staged and unstaged contents, not untracked files. Comparing must not stage,
restore, cherry-pick, or create a custom patch as a prerequisite.

For recognized renames, use old/new path metadata for the correct historical
versions. If the working path is missing, explain it and expose the arbitrary
file comparison; do not guess an unrelated destination. Explicit selection
works independently of Git rename detection. A deleted file has no blob in the
selected commit: do not silently substitute parent content; label any offered
parent-version comparison explicitly. Preserve ordinary addition/deletion diffs
in the previous-commit action. For merges, previous means first parent; disable
that action for root commits with a reason. Reject empty hashes/placeholders
and initially disable ambiguous multi-commit selections.

Validation: exact argv/endpoints, full commit versus file/directory, both case
variants, staged/unstaged changes, recognized/unrecognized renames, added/deleted
files, root/merge commits, missing paths, sticky-mode isolation, no selection,
and subprocess failure/return. Use a fake difftool; no GUI dependency.

### Compare a historical file with a chosen working-directory file

Add an independent **Diff with working file…** action for a historical file.
Prompt for an existing file within the current worktree, default to its likely
matching path, and allow browsing/completion plus typing a path. Accept tracked,
untracked and ignored files, including names Git never associates with the
historical path. Treat input as a path, never a revision or shell expression.
Resolve relative paths from the worktree root, including linked worktrees.
Show both historical `commit:path` and the chosen destination. Directories are
not single-file targets. Files outside the worktree are a separate extension.

Capture the source full commit OID and tree path before prompting. Read the
blob bytes through Git into a private temporary source file and compare with
the actual chosen file using the configured difftool. Investigate and verify
`git difftool --no-index` rather than assuming the current one-path revision
comparison can compare unrelated paths. Share this blob/file comparison helper
with the direct single-file working-tree action where needed so it compares
actual working-file contents even when that path is now untracked or ignored;
do not mistakenly show an index deletion instead. Launch from the worktree to
preserve local tool configuration. Preserve the source basename/extension so
tools recognize its type; stream binary data without text conversion. Missing
source, gitlink, or unsupported symlink cases need a clear reason; a missing destination
allows selecting another path.

Historical content is a temporary read-only source. Comparison does not itself
modify the real file/index/refs. If the external tool edits the working file,
treat that as the user's tool action and refresh afterward. Keep temporary data
until the tool finishes, including supported GUI wait modes, and clean it after
success, failure or cancellation. The probes show `--trust-exit-code` separates
successful tools returning 0 (Git returns 1 for differing files) from tool
failure (Git returns 128). Treat Git 0/1 as normal results for this action, with
trust enabled, before the generic subprocess error/prompt path runs. Do not
globally suppress exit code 1 for other commands. A tool that itself uses 1 for
successful comparison needs an explicit adapter/success-code policy; do not
mistake every tool failure for normal differences. Use restrictive temp
permissions and never execute source blobs.

Validation: fake tool records left/right bytes and paths; mismatched filenames,
untracked/ignored targets, nested/spaced/Unicode/option-like paths, binary and
empty blobs, symlinks/deletions, cancel, large-file streaming, exit codes and
cleanup. Assert unrelated dirty/index state is unchanged. Probe evidence is
recorded separately; the implementation and its integration seams are tracked
in [MergeGuide.md](MergeGuide.md).

## 4. Create or move a branch at a selected commit, without checkout

Complete (2026-10-22). A shared commit-controller hook exposes the initially
unbound action in local history, independent branch history and reflog. The
additive backend uses immutable plans and a prepared Git ref transaction, not
the upstream new-branch helper: type, expected tip and worktree reservations
are checked while Git holds the target-ref lock. See
[design/probe evidence](branch-placement-design.md) and
[usage](../docs-master/Fork_Features.md#create-or-move-a-branch-without-checkout).
Deleted/changed destinations require fresh review; symbolic aliases are refused.
Same-tip requests are informative no-ops, including the current branch. Creating
the unborn current branch is refused, but another branch can be created without
attaching HEAD. Occupancy checks are fresh, not a repository-wide lock against
arbitrary concurrent external checkout/worktree creation. Branch reflog recovery
is supported; upstream HEAD-based Undo is not claimed for this operation.

Verified: debug build, full short tests, generation, formatting, inventory tool
tests, lint (zero issues), targeted backend/controller race tests, real SHA-256
creation/movement and 127 branch/commit/reflog headless scenarios (107.551s).
The first broad headless attempt was invalidated by adding a scenario before
regenerating its list; the stable-source rerun passed. Verification ran on
Linux/Git 2.51.1, not a new full-suite or cross-platform/minimum-Git matrix.

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
is the current branch, direct the user to existing explicit reset actions;
do not add another current-branch soft-reset feature under this milestone.
Changing the checkout, even with a dirty tree, is milestone 5's separate
workflow. Recheck worktree occupancy and ref state immediately before the
write. If the ref moved after confirmation, require a new confirmation rather
than overwriting unseen changes. Use expected-old-OID ref updates where safe,
without losing worktree guards or branch/reflog semantics.

For a non-current destination, success leaves HEAD, index, files, and current
context unchanged. Refresh branch markers and retain the selected commit.
Include a useful reflog reason for a move. Do not claim the existing undo system
supports this until verified.

Validation: create, move, already-at-target, warning cancel, invalid/conflicting
names, detached HEAD, dirty index/worktree, occupied worktrees, concurrent ref
changes, upstream retention, and recovery through reflog.

## 5. Checkout any commit under a supplied branch name

Complete (2026-10-22). The shared checkout menu adds **Checkout as branch…**
with a `b` accelerator. An additive backend validates literal commit identity,
captures destination/HEAD state and invokes a single dirty-preserving checkout.
The target uses `OID^{commit}` because a bare OID can be shadowed by a same-named
branch. Current destinations are allowed without masking another worktree's
reservation. New and already matching names still perform checkout; tip moves
require a warning, and pre-launch destination/HEAD changes require fresh review.
Successful refresh selects the new checkout; failure refresh keeps the selected
history target/reflog position while showing Git's actual state. Existing
checkout/autostash helpers remain separate and unchanged.

Verified: debug build, full short tests, generation, formatting, inventory tool
tests, lint (zero issues), backend/helper race checks and all 170 selected
branch/commit/reflog/worktree/undo headless scenarios (433.042s). Additional
real-Git tests cover sparse-cone preservation, explicit submodule recursion and
dirty nested-file rejection, alongside SHA-256, alias/name validation, occupied
worktrees, pre-launch changes and hook partial failures. The first named-flow
run exposed a failure-selection bug; subsequent fixture naming/matcher issues
were corrected and the affected selection passed. Verification is Linux/Git
2.51.1, not a new full-suite, minimum-Git or cross-platform/ref-storage matrix.

The user chose Git's normal concurrency model: recheck destination/HEAD before
launch and coordinate other ref-mutating tools while checkout runs. Do not
promise expected-old atomicity during combined checkout. See the
[design and verified limitations](named-checkout-design.md). Native Git can
report failure after changing index/files or even completing checkout; refresh
the actual state and preserve that error without retry or assumed rollback.

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
reflog/worktree models together after success. Audit the existing checkout
autostash offers too: the fork's named-checkout workflow never requires or
offers stash. For overwrite conflicts explain which paths block checkout so
the user can commit instead. Reuse existing reset actions for pointer-only
current-branch changes; do not confuse reset with the dirty-preserving checkout.

Validation: genuinely different source and target trees, new/existing/current
destination, unrelated and overlapping staged/unstaged changes, untracked and
ignored collisions, deleted files, detached HEAD, other-worktree occupancy,
invalid input, canceled confirmation, and rejected checkout leaving destination
ref/HEAD/index/file contents unchanged. Cover sparse checkout/submodules to the
extent supported by existing checkout behavior; do not silently force them.

## 6. Rebase the current branch onto a selected commit

Complete (2026-10-22). The shared commit-list action is available through `?`
in local history, other-branch history and reflog, with a captured full target
OID, explicit topology choice, history confirmation and separate ignored-file
risk approval. Its default shortcut remains unbound until item 8. The new
backend is independent of marked-base mode; existing advanced rebase actions
remain separate. Source/risk changes require fresh review.

Verification: debug build, full short suite, generation, inventory tests,
formatting and scoped lint passed. The final affected-area headless run passed
all 410 selected Go subtests in 547.765s: 409 scenarios executed, and one was
intentionally skipped inside the harness. This includes clean stacked fixups,
refusal before compound amendments mutate anything, conflict recovery and
deleted-branch reflog invocation. Verification is Linux/Git 2.51.1, not a
minimum-Git or cross-platform matrix; detailed limits and intermediate failures
are recorded in [MergeGuide.md](MergeGuide.md).

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
affecting one branch. Agreed behavior: require a clean index/worktree, pass
`--no-autostash` explicitly so Git config cannot re-enable it, and suppress
updates to other branch refs for this action. Show a commit-first message when
dirty, with no stash suggestion. Apply the same no-autostash, clean-tree policy
to the fork's other rebase entry points during the audit, not just this new
binding. Do not remove manual stash commands; their removal is not requested.
The user clarified that harmless untracked files remain allowed, including
when the clean-state query would otherwise display them. Let Git reject
untracked collisions; do not impose an empty-untracked-list prerequisite.
For the new selected-commit rebase flow, warn about potential ignored-file
collisions and require explicit approval before proceeding. Cancel is the
default in the risk menu. A changed affected-file snapshot requires fresh
review and approval; the scan is conservative, not a guarantee that every
listed file will be overwritten.

Historical amendments and stacked-branch fixup/amend placement must refuse
before creating a commit if tracked edits would remain for the required rebase.
The user explicitly chose refusal, not a fixup-only fallback. A message-only
amendment does not consume staged changes, so those also prevent placement.
Ordinary fixup creation without a rebase is not subject to this restriction.

Validation: divergent/ancestor/descendant targets, selected commit on another
branch or in reflog, root commit, no-op, dirty state, conflicts with continue
and abort, detached HEAD, marked-base isolation, and `rebase.updateRefs=true`.

## 7. Selected-branch operations without checkout

Complete (2026-10-24). Captured native plans and small controller/helper hooks
now retain the selected ref, current destination and repository across review,
execution and refresh. Focused push affects one selected branch only; wider
updates remain explicit. Upstream metadata, rename, fast-forward and fetch-first
current Pull use exact native tracking refs. Rewritten upstreams are not reset
automatically, and off-branch integration needing a worktree remains deferred.

Reset, rebase and merge clearly identify current HEAD as the destination and
retain their modes, messages, signing, conflicts and explicit risk approvals.
Deletion keeps local/remote/both choices, refuses occupied local branches,
requires exact remote destinations and preserves actual partial failures. Refs
being deleted cannot prove retained history. Explicit Worktrees-pane removal
remains a separate requested operation.

Branch creation exposes configured/no-setup/direct tracking policy before
create-and-checkout, preserves literal names and refuses overwrite collisions
without stash. Native tracking policy resolves the original branch ref after
the final exact-state check under Git's normal concurrency model. Pointer-only
placement and reviewed existing-name checkout remain separate actions. Tag
creation captures annotation/signing and the exact old tag object for overwrite
approval. Historical/stash sources retain immutable commits.

Verification: full short suite, generation, debug build, formatting, scoped lint
and inventory/parser checks pass. The full branch/sync terminal group passed
with `-parallel 1` (405.161s); the existing intentionally skipped sync fixture
remains skipped. All nine affected branch/commit/stash creation cases, the tag
group, occupied-worktree refusal and explicit worktree removal/filter baselines
also pass. Two long deletion scenarios reached the harness's fixed 40-second
limit during the initial concurrent broad run; focused and serialized reruns
passed without changing timeouts or code. Validation is Linux/Git 2.51.1;
[MergeGuide.md](MergeGuide.md) records integration and verification details.
Default key policy is item 8, so its migration is not included here.

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
Push destinations are independent of saved upstream settings. Upstream setup
is a separate, default-off choice. If native setup cannot safely replace
ambiguous or inherited settings, refuse that change before pushing and offer
Cancel or a push without changing tracking (see item 11 for when a push shows
a separate review). Never repair those
settings silently. Updating a remote-tracking cache ref after a successful
push is distinct from changing the local branch's saved upstream relationship.
Push is limited to the selected branch for now: do not offer a multi-branch
push or a stack-member selection dialog. Wider-stack push is deferred, including
its previously prepared flow. Audit stacked-branch pull/update menus separately:
changing a selected branch must not implicitly update branches below it; wider
updates remain explicit choices. Explicit multi-selection Fast-forward is
unchanged by this push-only scope decision.

For the checked-out branch, Pull fetches first, then presents the exact fetched
commit and integration choices before changing files. Default the integration
choice from Git configuration, retain explicit fast-forward/merge/rebase
choices, and check ignored-file overwrite risks against the captured target.
Do not perform a second network pull that could substitute another incoming
commit after review. Fetching alone must preserve HEAD, the index and files.

Validation: local fake remote; selected != current, no upstream, gone upstream,
different push/pull remotes, force-with-lease rejection, fast-forward/diverged
and rewritten upstream, occupied worktree, concurrent ref changes, and branch
stacks. Assert current HEAD/index/worktree and unrelated refs remain unchanged.

## 8. Consistent default keys and no-op behavior

Complete. The agreed table in [keymap-allocation.md](keymap-allocation.md) is
the enabled default profile, documented with a migration table in
`docs-master/keybindings/Fork_Keybindings.md`. Modal editor fields support
Ctrl+Left/Right word movement on all platforms; these are input controls, not
core Ctrl commands. The only Ctrl defaults are modal editor controls.

Implementation notes that refine the design below:

- Escape clears cherry-pick marks in the history pane that holds them. In a
  subview, Escape first returns to the parent pane and keeps the marks so they
  can be pasted elsewhere. A successful paste clears the marks (the user
  rejected upstream's hidden re-paste buffer); a paused paste keeps them.
- Enter commits in Files, so the default `v` inspection opens a conflicted
  file's resolution view or dialog there. An explicitly configured focus-diff
  key keeps upstream's focus-only behavior.
- Hints, tips and messages name the effective command path for a command ID in
  the relevant pane, never an upstream legacy key field.
- Explicit legacy fields and stable-ID `commandBindings` override defaults in
  the documented order; real-default and override scenarios under
  `pkg/integration/tests/ui/fork_*` cover the profile.

Use milestone 2's complete inventory to produce and review a verb-to-key table
before changing defaults. All alphabetic commands are case-insensitive: `p`
and `P` cannot mean pull and push, and `r` and `R` cannot mean rename and
refresh. Normalize command matching and collision checks, including menu-item
accelerators, while leaving entered text/paths/search patterns unchanged.
Define this initially for ASCII command letters rather than applying locale
case conversion to arbitrary user text. Punctuation remains literal.

Reserve Ctrl for user custom commands: no core default Ctrl bindings, including
diff, scrolling, filtering, clipboard, editor-submit, alternate navigation, and
platform-specific defaults. Preserve conventional terminal/editor input and
special keys such as Enter/Tab, which some terminals encode as control bytes;
do not claim those byte aliases are independently bindable. Core features get
unmodified letters, menus, or dedicated non-letter navigation keys. Ctrl-based
custom commands remain supported; explicitly configured overrides may opt back
into old behavior.

Use the existing menu machinery for two-/three-letter bindings. A first key
opens a visible command group, a second invokes a leaf or opens a submenu,
and a third invokes a leaf there. Groups themselves never mutate Git state.
Prefixes and continuation letters have consistent meaning across panes;
inapplicable leaves do nothing. Use accelerator-driven menus without automatic
filter-as-you-type so continuation letters select actions instead of becoming
search text. Show complete paths in `?` and cheatsheets and expose each leaf
as a standalone palette action; do not require opening the group from `?`.

Nested menus need explicit return/cancel state and captured origin/selection;
the current generic menu does not by itself establish a back-navigation stack.
Ensure Esc, focus/repo changes, popup replacement and invalid continuation do
not invoke another command or replay letters into a different pane. Prefix
menus need no inter-key timeout or hidden chord state.

The latest user-specified key table is authoritative in
[keymap-allocation.md](keymap-allocation.md); displaced defaults must move
without case-only or prefix/leaf conflicts. Diff uses `d w` (working tree),
`d p` (previous commit) and `d c` (pasted partial/full commit hash), for both a
whole commit and a selected historical file. These replace the provisional
single-letter `w`/`v` defaults. The first `d` opens a visible group; each complete
path is a separate action. Validate pasted hashes unambiguously and preserve
captured selection and labeled endpoints through the prompt.

Use `p` push, `f` fetch, F2 rename/reword, `e` edit, `o` open, F5 full repository
refresh and every-pane re-render without fetch, Delete delete/drop, `x` discard
with mandatory confirmation (except immediate unstage in the staged diff), `a` amend, `r` rebase and `m` merge. Rebase/merge
confirmation explains current destination, selected source and consequences.
Pull must have no single-letter default; keep its proposed `g l` path and
independent palette entry.

`s h` / `s m` / `s s` select hard/mixed/soft reset of current branch to selected
commit, with confirmation explaining mode effects. `s c` switches/checks out
the selected branch/commit, with options to name a new branch or explicitly
approve moving an existing non-current branch. `n b` creates/moves a branch at
the selected branch/commit without checkout. Reuse captured target/backend
flows and native Git concurrency/dirty-file protections. Reserved/occupied
branches remain protected; current-branch relocation belongs to reset.

Enter opens the commit flow only when the Files pane is focused; it must do
nothing in other normal panes, including a focused working-tree diff. Dialogs,
search and editors retain their normal Enter behavior. The user approved `v`
for inspection, `h`/`l` for pane switching, Tab/Shift+Tab for tab switching and
F3/Shift+F3 for next/previous search matches. `/` enters search text; after
submission `p` remains Push. Do not use mode-specific `p`/`n` search navigation,
or Vim-style `n`/`N` next/previous aliases that violate case-insensitivity. Space is an agreed scoped exception: only in Files it toggles
staging (including directory/root selection); on other selections it inspects
details without mutation. Do not add `s t` or a stage-all shortcut: selecting
the tree root covers all files. Left in Files collapses an expanded directory,
otherwise selects its visible parent; Right expands a collapsed directory,
otherwise selects its first child. They must not switch panes. Number keys
remain available for pane switching. Handle filtered/compressed trees and
root/leaf/empty cases safely.

`c y` marks selected commits idempotently; `c x` unmarks only selected commits;
`c p` cherry-picks marked or selected commits after a confirmation naming the
exact ordered commits and current destination. Do not reuse upstream toggle,
reset-all or count-only confirmation as equivalents. No implicit stash. The
proposed review offers a marked/selected source choice when both exist.
`u u` / `u r` undo/redo, `i` ignores, `q` quits, `?` discovers commands, `/` searches,
`:` enters a command, and `y` opens copy choices. Escape is the general exit/cancel
mechanism; leaving cherry-pick marking clears every mark and replaces a separate
`c a` unmark-all command. Preserve normal modal/prefix cancellation and never
interpret Escape as silent Git abort or working-file discard. Navigation and less
common operation paths follow the implemented allocation table.

`g c` replaces the normal squash/fixup commands with **Combine selected
commits**. The replacement sits at the first selected commit, defined by Git
ancestry: it must be an ancestor of every other selected commit, not just the
oldest timestamp/last visible row. Do not combine into an unselected parent.
Prefill the summary from that first commit; the editable description preserves
every selected full message (subject and body), including the first, in
ancestor-first order. The user confirmed preservation of all message text.
Capture and review exact commits, destination, rewrite effects and edited text;
preserve unselected history/side refs and never silently flatten merges. Resolve
topology eligibility before implementation. No implicit checkout to another
branch or autostash. Remove competing squash/fixup default paths and relocate
`g c f` file restoration to proposed `g v f` (`g v c` reverts a commit), so `g c`
is a leaf with no conflicting descendants.

Default log scope is the full all-ref graph, rather than only current-branch
history: `git.log.showWholeGraph: true`. Preserve explicit user/repository false
and ordinary view toggles. This independent default can be delivered while
final keybindings remain paused. The setting selects Git's `--all` history; it
is distinct from drawing graph characters, pagination and fetching remote data.

For a reserved action key whose verb is inapplicable, consume the event with no
side effect. Do not fall through to an unrelated parent/global binding, show
an error toast, or increment usage. Palette entries may remain disabled with
an explanatory tooltip. Keep genuine operation errors visible after invocation.
Distinguish inapplicable commands from actionable precondition failures such
as an overwrite conflict.

Agreed exception: text entry, search/filter input, and external subprocesses
retain normal typing. Menus retain explicit navigation/confirm/cancel controls.
Search-match and operation-specific modes must not repurpose reserved action
keys to unrelated commands. Review whether navigation/selection should be
shared verbs or have pane-specific meaning before final allocation.

Explicit user key overrides/custom commands remain possible; the cross-pane
contract applies to the fork's defaults. Document overrides that recreate
collisions. The old custom Ctrl+T target menu can remain as an explicit user
override, but is no longer needed for either core diff action; do not silently
edit the user's config or ship Ctrl+T as a builtin.

Validation: a default key × pane/mode matrix, including aliases and platform
variants. Verify lowercase/uppercase parity, no core Ctrl defaults, prefix/menu
path uniqueness, nested-menu cancellation, dispatch and side effects,
disabled-key no-ops, no global fallback, menus/filtering/editors, custom Ctrl
commands/overrides, and generated cheatsheets.
Provide a migration table for changed defaults in `docs-master/`.

## 9. Frequency-ordered `?` palette

Complete, with these user decisions refining the plan below:

- Scoring follows how the command was reached: a shortcut or prefix-menu leaf
  subtracts 1 (the user has learned it) and palette execution adds 1. Counts
  may go negative, so learned commands sink below never-used ones.
- A use is pending while its handler runs. The first confirmation, prompt or
  menu opened through `PopupHandler` meanwhile commits it on acceptance and
  drops it on cancel; otherwise it commits when the handler returns. For easy
  upstream merges, popups opened later from background workers are not
  tracked, and menus whose items include their own cancel action count as
  accepted.
- Counts persist in a dedicated `command_usage.json` beside `state.yml`
  (`pkg/commandusage`): deltas flush about two seconds after a use and on exit,
  under a lock file with a temp-file rename, merging concurrent processes. An
  unreadable file is dropped with a toast saying the counts were reset.
- The palette merges local and global commands into one usage-sorted list with
  a dim `local`/`global` tag; Navigation stays a static section at the bottom.
  `gui.commandPaletteOrder: static` restores the upstream sections, and a
  palette-only command resets the counts.

Sort discoverable actions by local usage counts using stable command IDs.
Agreed scope: global per-user counts, shared across repositories. Planned
initial policy is lifetime frequency. The palette still shows actions relevant
to the current context. Sort the whole actionable list by count descending,
with original declaration order as a deterministic tie-breaker. Retain
Local/Global/Navigation as annotations or
filters rather than fixed groups that prevent global frequency ordering.
Keep the displayed order stable while the palette is open.

Record direct shortcuts and palette execution consistently and once per
accepted action. Do not count typing, navigation repeats, opening `?`, disabled
actions, or canceled confirmations. Count each direct diff action separately;
opening a prefix menu must not double-count its eventual leaf. Direct palette
execution and a menu path share the same leaf identity. Once an operation is
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

## 10. Separate configuration and state from upstream lazygit

Complete, as planned below. lazygitk must install side by side with upstream lazygit without
either reading or writing the other's files. User decisions:

- Global config and state use a `lazygitk` directory wherever upstream uses
  `lazygit`: `<XDG_CONFIG_HOME>/lazygitk/` (searched in the XDG config dirs as
  upstream does, so `%APPDATA%` still works on Windows) and
  `<XDG_STATE_HOME>/lazygitk/` for `state.yml`, `command_usage.json` and the
  pull-request cache.
- Upstream's `lazygit/` and legacy `jesseduffield/lazygit/` directories are
  never read. Nothing is copied or migrated; a fresh `config.yml` is created on
  first start, as upstream does.
- Repo-level config uses only `<repo>/.git/lazygitk.yml` and `.lazygitk.yml`
  in parent directories. Existing `lazygit.yml` files no longer apply.
- `CONFIG_DIR`, `--use-config-dir` and `LG_CONFIG_FILE` keep their names and
  behavior; the integration harness depends on them.
- The temp directory stays `lazygit-<uid>`: every process uses its own
  subdirectory, so sharing it is harmless.
- `scripts/build_dist.sh` names release binaries `lazygitk`.

Acceptance: with XDG variables pointing at temporary directories, config and
state paths resolve under `lazygitk/`; an existing `lazygit/config.yml` or
`jesseduffield/lazygit/config.yml` is ignored; `CONFIG_DIR` still wins;
per-repo scenarios load `lazygitk.yml` files; `--print-config-dir` reports
the `lazygitk` directory; user docs describe the new locations.

## 11. Push without a redundant review

Complete. The "Push without tracking" choice after an ambiguous-upstream
refusal also pushes directly, since that menu names the source and
destination. The push menu already shows source, commit, remote, destination,
upstream and submodule settings, so choosing **Push this branch** pushes after
the usual preparation, without a second "Review branch push" confirmation.
TortoiseGit's push dialog likewise pushes directly from its options. User
decisions:

- No review after choosing **Push this branch**, including when the remote or
  destination was just typed in a prompt (from the menu's choose-remote or
  choose-destination items, or because no remote was configured).
- A review is still shown when setting upstream would replace existing,
  different tracking, and for force-with-lease (unchanged).
- Preparation still pins the push to the commit shown in the menu: if the
  branch moved, the menu reopens with fresh details instead of pushing.
  Ambiguous-upstream and multiple-push-URL choices are unchanged.
- The stacked-push collection flow keeps its per-member review and final
  batch approval.

This refines item 7's "newly reviewed push" wording: the menu summary is the
review for an ordinary push.

Acceptance: an ordinary push from the menu reaches the remote with one
choice; typed remote/destination paths push without a confirmation; tracking
replacement and force-with-lease still show their reviews; a branch that moved
after the menu opened reopens the menu and does not push.

## 12. One flat copy menu with consistent keys

Complete. Implementation notes: commit files also gained `y c` for the file's
content at that commit. On branches and tags the tags action checks for tags
when pressed, since the ref's commit is resolved then. The reflog's existing
copy binding is the full hash; commits and sub-commits keep theirs as the
abbreviated hash. The agreed plan:

`y` opens one flat copy menu. The nested copy menus lose their `y c`
default (they stay reachable from `?`), and each second key means the same
thing in every panel. No default uses Shift. Anything that points at a commit
offers the commit options: commits, reflog, sub-commits, local branches,
remote branches and tags (resolved to the commit at the time of the copy).

| Keys | Copies | Panels |
| --- | --- | --- |
| `y n` | Name | local/remote branches, tags, submodules, Files, commit files |
| `y p` | Relative path | Files, commit files |
| `y f` | Absolute ("full") path | Files, commit files |
| `y c` | File content at that commit | commit files |
| `y h` | Full commit hash | commit-like panels and refs |
| `y i` | Abbreviated commit hash ("id") | commit-like panels and refs |
| `y s`, `y m`, `y b` | Subject, full message, message body | commit-like panels and refs |
| `y u` | Commit URL | commit-like panels and refs |
| `y a` | Author | commit-like panels and refs |
| `y t` | Tags on the commit | commit-like panels and refs |
| `y d` | Diff of the selection: commit, selected file, or selected diff lines | commit-like panels, refs, Files, commit files, diff views |
| `y w` | Diff of all working-tree files | Files |
| `y r` | Pull request URL | local branches |
| `y y` | Confirmation text | confirmation popups |

- A message body or tag list that turns out empty is reported when the copy
  is attempted, rather than pre-computed for every menu render.
- The full-hash and abbreviated-hash actions get distinct command IDs; today
  both use `clipboard.copy-commit-hash`.
- Stash entries are out of scope.
- Explicit legacy configuration (for example `copyToClipboard` or
  `copyCommitAttributeToClipboard`) keeps its configured keys.

Acceptance: every row above works in each listed panel from the `y` menu and
the palette; copied text matches the existing menu actions; refs copy their
resolved commit; generated keybinding docs and the keymap snapshot show the
table; no `y` default conflicts within a panel.

## 13. Fixed keybindings: key configuration is ignored

Planned. This is a personal fork, so keybindings are not configurable: the
fork's defaults always apply. Upstream's `keybinding` field names overlap and
fold onto each other in the fork's case-insensitive routes (for example
explicit `p` pull and `P` push become one key), and they leak into panels
where upstream never used them. If the fork is ever distributed more widely,
a separate, non-conflicting set of setting names should be designed instead
of reviving these. User decisions:

- Ignore the `keybinding` section, `commandBindings`, and every custom
  command's `key`, in global and per-repo config files. Custom commands stay
  runnable from `?`.
- When a loaded config file contains any of these, show a startup notice
  naming the ignored settings and the file, so copied upstream settings do
  not silently do nothing.
- The integration harness keeps honouring key configuration through an
  internal switch, so upstream's scenarios (which drive keys from the legacy
  fields) keep running unchanged. A scenario can turn the switch off to test
  the user-facing behaviour. Users have no way to turn it on.
- Item 8's override precedence (`commandBindings` over explicit legacy
  fields over defaults) now applies only inside the harness.

Acceptance: with the switch off, a config that sets `keybinding` fields,
`commandBindings` and custom-command keys runs with the fork defaults, the
custom commands appear in `?` without shortcuts, and the startup notice lists
what was ignored; a config without them shows no notice; with the switch on,
the existing suite passes unchanged; docs no longer describe key overrides as
a user feature.

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

The subsequent planning revision changes Markdown only. Formatting and diff
checks passed; unchanged application tests were not rerun. Additional disposable
file-pair probes verify local difftool config, spaced paths and exit statuses;
they do not validate a completed UI or implementation.

## Next design review artifact

The earlier questions are answered under “Agreed direction” and the relevant
milestones. Item 13 is current; new work after them starts by adding
a reviewed milestone here.

This remains an extensible backlog: add further features with target semantics,
repository evidence, dependencies, and acceptance tests. No application feature
has been implemented by these planning updates.
