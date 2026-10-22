# Selected-branch operations: audit and design

Roadmap item 7, audited against this fork after item 6. The read-only foundation
is verified; executable sync flows are not implemented yet. Native experiments
are recorded separately in
[selected-branch-sync-probes.md](selected-branch-sync-probes.md).

## Existing behavior and integration seams

| Area | Existing target | Required fork treatment |
| --- | --- | --- |
| Rename/delete | Selected local branch(es) | Retain selected-ref behavior and Git's occupancy constraints; review captured repository and literal-name handling |
| Set/unset upstream | Selected branch name | Retain explicit branch, but avoid assuming tracking refs always use the standard namespace |
| Tag, branch creation, history, diff | Selected ref | Retain existing selected-ref support; milestone 4 already adds ref-only placement |
| Branch-pane merge/rebase | Selected branch is source; current branch is destination | Name both roles explicitly; these are not off-branch integration |
| Branch-pane reset | Current checkout resets to selected ref | Label as a current-branch operation; never describe soft/mixed/hard modes as moving an unoccupied branch |
| Push/pull | Globally checked-out branch | Capture the intended branch and repository before any modal or worker |
| Fast-forward/update | Selected branches, possibly in other worktrees | Do not reuse unchanged: it may reset rewritten history or alter another checkout |
| Stack sync | Checked-out branch plus branches found in current history | Keep stack-wide operations separate, explicit choices; never derive another selected branch's stack from current history |

Primary code locations:

- `controllers/sync_controller.go`: global push/pull bindings and checked-out
  target resolution. `pushAux` normally passes no explicit source refspec;
  `setCurrentBranchUpstream` and `pullWithLock` rely on current HEAD. Existing
  stack menus put the wider operation first. `LocalCommitsController` also
  receives `HandlePull` as an internal current-branch callback.
- `controllers/branches_controller.go`: selected-ref metadata, history, tags,
  source merge/rebase, reset, rename and explicit fast-forward actions. Rename
  currently sanitizes user input; selected-ref upstream setting calls
  `Branch.SetUpstream(remote, branch, selectedName)`.
- `helpers/branches_helper.go`: `PrepareFastForward` captures model worktrees,
  fetches upstreams, then may reset a rewritten upstream and update the files
  of occupied worktrees. `IsAncestor` collapses all query errors into false;
  new safety-sensitive planning must distinguish non-ancestry from query
  failure. The batch ref update already uses native expected-old `update-ref`.
- `git_commands/sync.go`: explicit `PushOpts.Refspecs` exists, but the ordinary
  push path omits it. Fetch can follow configured tag/ref mappings. Pull is
  inherently a checked-out-worktree operation, with item 6's clean-rebase policy.
- `git_commands/branch_loader.go`: display models contain push/pull destinations
  and divergence, but are cached presentation data, not execution authority.
- `context.SubCommitsContext.GetRef` retains the local branch, remote branch,
  tag or commit used to open that history. It can identify branch-bearing
  history without reading a stale selection in another pane.
- The application Git builder pins a captured repository's working directory
  and location environment. Retain its `GitCommand` pointer through the flow;
  reading `self.c.Git()` afresh inside a later worker can select another repo.
  Refresh already has repository-generation protection; operation flags and
  completion prompts need the same repository scope considered explicitly.

## Captured-source contract

The initial additive modules are `git_commands/branch_sync_info.go` and
`controllers/branch_sync_source.go`. Neither is wired into executable push/pull
yet. Native inspection captures exact source OID, actual symbolic HEAD/tip,
upstream remote/ref/tracking namespace, suggested push metadata and fresh
worktree reservations. It rejects aliases and missing or non-commit branch
tips without requiring a clean current checkout. Returned inspection data is
not an authorized mutation plan. Push metadata may name only the first of
several configured destinations; it does not prove uniqueness.

| Focused context | Intended source |
| --- | --- |
| One selected local branch | That captured branch name, including when it is current |
| History opened from a local branch | That history's stored local branch, not another pane's selection |
| Remote-branch pane or history opened from a remote/tag/commit | Refuse the branch-sync action; require a local branch |
| Other panes | Explicitly labeled current local branch; ignore remembered selections |
| Multi-selection, missing branch, detached pseudo-branch | Refuse; do not silently pick a branch |

Tag-pane push remains its existing selected-tag operation. The current-branch
fallback requires an actual `Head` marker, not merely the first model row;
native inspection and execution validation must still confirm it. A captured
name is copied rather than retaining a mutable model pointer. Existing current
pull callbacks used by local-commit operations must remain explicitly current,
not become context-dependent accidentally.

## Execution contract for the following slices

Keep additive backend and controller modules with small integration hooks.
Separate a captured local source branch from its pull upstream and push
destination. Revalidate native state before writes, under Git's normal
concurrency model; do not promise an atomic network/ref/worktree transaction.
No automatic retry of an operation that may already have changed refs.

Use a modal showing source and destination, with editable remote/branch choices
and explicit upstream-setting consent. Do not silently run `push.default`
matching behavior, additional configured push refspecs, tag following or a
mirror operation when the chosen command says one branch. Ambiguous or missing
configuration needs a visible choice, not a guessed hidden refspec. Preserve
force-push restrictions and lease protection; do not fetch merely to replace
the expected value of a force-push lease.

Off-branch pull/update is fetch plus ancestry verification and a ref-only fast
forward. It must not reset divergent or rewritten history, run integration,
modify another worktree or create a new one. Handle remote `.` and nonstandard
tracking namespaces deliberately. Occupancy and branch identity need fresh
checks, not only the branch model's `Head` marker.

The native probe confirms that fetch needs explicit ref mapping, no pruning,
no tags, no submodule recursion and no `FETCH_HEAD` write to keep its stated
scope. A push needs an explicit chosen destination; neither cached model data
nor the first native push-mapping result is sufficient authority. Snapshot
validation and a user's modal choices must precede executable commands.
Shortcut redesign stays in item 8.

## Usability and native references

[TortoiseGit's Push dialog](https://tortoisegit.org/docs/tortoisegit/tgit-dug-push.html)
exposes local source, remote branch and remote separately, preselecting tracking
information and keeping upstream-setting a visible option. Its
[reference browser](https://tortoisegit.org/docs/tortoisegit/tgit-dug-browse-ref.html)
already supports selected-ref metadata and comparisons. These guide the
distinction between a selected branch and the current checkout, rather than
requiring a checkout for a ref-only operation.

[Git push](https://git-scm.com/docs/git-push) documents explicit refspec
precedence, configured mirroring, and explicit expected-OID leases. The
[Git fetch manual](https://git-scm.com/docs/git-fetch/2.51.1) documents that an
empty `--refmap=` disables additional configured fetch mappings. Exact ref
selection alone is not sufficient evidence that tags, extra tracking refs,
submodules or prune effects are suppressed.
