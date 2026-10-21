# Git operation investigation

Scope: Git semantics for creating or resetting a branch without checkout,
checking out an arbitrary commit under a branch name while retaining local
changes, and operations on a selected non-current branch. Investigation only;
no repository tests were run.

## Findings

### Create/reset a named branch without checkout

`git branch <name> <commit>` creates a branch ref without moving `HEAD`,
changing the index, or changing the worktree. Existing
`BranchCommands.NewWithoutCheckout` in
`pkg/commands/git_commands/branch.go:44` does exactly this. It fails if the
name already exists. `git branch -f <name> <commit>` moves an existing ref
without checking it out, but refuses to move a branch checked out in any
worktree. This supports branch-ref-only operations on non-current branches;
it does not perform a checkout or reconcile dirt against the target tree.

### Checkout a commit under a new or existing branch name

`git checkout -B <name> <commit>` is Git's combined create/reset-and-checkout
operation. Git documents `-B` as creating/resetting the branch only if the
checkout succeeds. In probes with Git 2.51.1, the command switched `HEAD` to
the named branch and set it to the requested commit when successful. With an
existing branch checked out in a separate worktree, it exited 128, left the
current worktree on its original branch, and left the target ref unchanged.

The corrected fixture used genuinely different trees: commit A on `main`
contained `common=base` and `overlap=base`; commit B on `target` changed
`overlap` to `target` and added `target-only=target`. Before each operation,
`git rev-parse HEAD` differed from the frozen B SHA and `git diff HEAD <B>`
showed both tree changes. The probes showed:

- unrelated staged changes, unstaged changes, and untracked files survived a
  successful `checkout -B`; their index/worktree state remained present;
- an unstaged edit and, separately, a staged edit to `overlap` both caused
  checkout rejection because the target changed that path;
- a differing untracked `target-only` caused rejection because checkout
  would overwrite it;
- an ignored `target-only` was overwritten by default, leaving the checkout
  clean; adding `--no-overwrite-ignore` made checkout reject it;
- an existing branch checked out in another worktree caused rejection, and
  the ref remained unchanged.

The observed tracked and untracked rejection cases left the existing `dest`
branch at A, kept symbolic `HEAD` on `main`, and preserved the exact index
tree hash and dirty file bytes. The worktree rejection did the same. The
ignored-file default is a real exception to a broad “preserve user files”
promise: ordinary `checkout -B` overwrites ignored collisions. The
`--no-overwrite-ignore` probe rejected the checkout instead.

For a failed `checkout -B` caused by another worktree, the branch ref, current
`HEAD`, index, and worktree were preserved. Git's documented contract also
places the ref update after successful checkout. This is materially safer
than separately moving the ref and then checking it out.

Current-branch case: `checkout -B main <B>` succeeded and moved the current
branch and worktree to B. By contrast, `git branch -f main <B>` exited 128
with `fatal: cannot force update the branch 'main' used by worktree ...`;
`main`, attached `HEAD`, and the index hash were unchanged. Thus ref-only
`branch -f` cannot be used to reset the currently checked-out branch in this
Git version.

### Operations on a selected non-current branch

The command layer already contains branch-ref operations that do not switch
the user's checkout: `NewWithoutCheckout`, `CreateWithUpstream`, `SetUpstream`,
`UnsetUpstream`, `Rename`, `LocalDelete`, and `UpdateBranchRefs` in
`pkg/commands/git_commands/branch.go`. Queries and graph viewing also accept
the selected branch ref directly. In contrast, `RefsHelper.CheckoutRef` calls
`Branch.Checkout`, and selected-branch `press` in
`pkg/gui/controllers/branches_controller.go` routes there to check out the
branch. Upstream reset/rebase actions in the branch controller delegate to
ref helpers and must be separately understood before describing them as
checkout-free; they can mutate branch history/ref state.

Some branch operations have worktree constraints: local deletion is guarded
by `BranchesHelper.ConfirmLocalDelete` in
`pkg/gui/controllers/helpers/branches_helper.go`, which prompts about or
blocks branches checked out in another worktree; Git itself refuses force
updates of a ref checked out in another worktree.

## Probe method and exact commands

No project tests or code were run. Temporary repositories were created under
`mktemp -d`. Commands were run with `GIT_CONFIG_NOSYSTEM=1`,
`GIT_CONFIG_GLOBAL=/dev/null`, `GIT_TERMINAL_PROMPT=0`, and Git's local
`-c core.hooksPath=/dev/null -c user.name=Probe -c
user.email=probe@example.invalid` options.

Each corrected fixture used `git init -b main`, committed A with
`common=base` and `overlap=base`, created `target` from A, committed B with
`overlap=target` and `target-only=target`, returned to `main`, and created
`dest` at A. Each command used the frozen B SHA as its destination. The
precondition check was:

```sh
test "$(git rev-parse HEAD)" != "$target_sha"
git diff --stat HEAD "$target_sha"
```

The inequality held and the diff showed `overlap` and `target-only` for each
fixture. The clean operation was `git checkout -B dest "$target_sha"`; it
succeeded, attached `HEAD` to `dest`, and moved `dest` from A to B.

For unrelated dirt, the fixture staged a new file, appended to `common`, and
created an untracked `loose`; `git checkout -B dest "$target_sha"` succeeded.
Status showed ` M common`, `A  staged`, and `?? loose`; the staged file and
loose file contents remained. The operation also updated tracked `overlap`
and added `target-only` from B.

For tracked dirt, the fixture changed `overlap` to `local`, once unstaged and
once staged, then ran `git checkout -B dest "$target_sha"`. Both exited 1
with Git's “local changes ... overwritten by checkout” error. Before and
after snapshots matched for symbolic `HEAD` (`main`), `main`, `dest`,
`git write-tree`, status, and the `overlap` bytes (`local`).

For an untracked collision, `target-only=local` was created untracked before
`git checkout -B dest "$target_sha"`. It exited 1 with Git's “untracked
working tree files would be overwritten” error. `HEAD`, both refs, the index
tree hash, status, and file bytes were unchanged. For the ignored collision,
the path was placed in `.git/info/exclude` (so no source commit changed), then
created with `local` contents. Default checkout succeeded and replaced it
with `target`; with `git checkout --no-overwrite-ignore -B dest
"$target_sha"`, checkout exited 1 and preserved the ignored file and the
pre-operation `HEAD`, refs, index hash, and status.

For the worktree rejection, `git worktree add <tmp>/wt dest` checked out
`dest` at A elsewhere; `git checkout -B dest "$target_sha"` in the main
worktree exited 128 with `fatal: 'dest' is already used by worktree ...`.
The `dest` and `main` refs, current `HEAD`, index hash, status, and tracked
file bytes matched their pre-operation snapshots.

The failure snapshots were checked before and after each command. Exact
observed hash pairs (before = after) were:

| Rejection | symbolic `HEAD` | `main` ref | `dest` ref | `git write-tree` | Preserved bytes |
| --- | --- | --- | --- | --- | --- |
| Unstaged overlap | `main` | `18d3eea` | `18d3eea` | `1ccb4b5` | `overlap=local` |
| Staged overlap | `main` | `18d3eea` | `18d3eea` | `9a00d92` | `overlap=local` |
| Untracked collision | `main` | `18d3eea` | `18d3eea` | `1ccb4b5` | `target-only=local` |
| Ignored collision with `--no-overwrite-ignore` | `main` | `b02e97e` | `b02e97e` | `1ccb4b5` | `target-only=local` |
| Branch held in other worktree | `main` | `b02e97e` | `b02e97e` | `1ccb4b5` | `overlap=base`; `target-only` absent |

The table abbreviates object IDs to seven characters; every listed value was
identical before and after the corresponding rejected operation. In the
ignored collision row, status also remained clean because the path was
excluded by `.git/info/exclude`.

Finally, `git checkout -B main "$target_sha"` succeeded on current branch
`main`, while `git branch -f main "$target_sha"` was rejected because the
branch was checked out. `git branch -f dest "$target_sha"` remains useful
for a non-current, non-worktree branch, where it moves only the ref. Trying
to force-update `dest` while it was checked out in another worktree was
rejected.

## Coordinator disposition and probe limits

The main plan distinguishes ref-only creation/movement from named checkout.
It blocks moving branches occupied in another worktree, proposes a separately
warned soft-reset path for the current branch, uses `--no-overwrite-ignore`
for named checkout, and retains Git's ordinary sparse/submodule constraints.

These probes cover ordinary tracked/staged/untracked/ignored files and branch
occupancy. They do not establish guarantees for sparse checkout, submodules,
hooks, concurrent Git processes, or recovery after process interruption.
Implementation tests must cover the supported cases rather than extrapolate
transactional guarantees from the ordinary checkout rejection cases.

## Relevant source paths

- `pkg/commands/git_commands/branch.go`: `NewWithoutCheckout`,
  `CreateWithUpstream`, `Checkout`, `SetUpstream`, `UnsetUpstream`, `Rename`,
  `LocalDelete`, `UpdateBranchRefs`.
- `pkg/gui/controllers/helpers/refs_helper.go`: `CheckoutRef`, including
  autostash handling; `CheckoutRemoteBranch` creates upstream metadata before
  delegating to checkout.
- `pkg/gui/controllers/branches_controller.go`: selected branch `press`,
  force checkout, checkout-by-name, upstream actions.
- `pkg/gui/controllers/helpers/branches_helper.go`:
  `ConfirmLocalDelete` and worktree-aware delete behavior.
- `pkg/commands/git_commands/branch_test.go`: command argument unit tests for
  branch creation, deletion, and checkout; not run per request.
