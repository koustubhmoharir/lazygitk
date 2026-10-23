# Captured selected-source merge (current item 7)

The rebase flow is integrated. This is the remaining merge audit's working
design, not delivered behavior or planning for a later roadmap item. Retain
ordinary/configured merge, allow-FF/no-FF choices, both squash modes, message
editing/configuration and signing. Keep selected source and current destination
explicit. Consult [TortoiseGit's merge guide](https://tortoisegit.org/docs/tortoisegit/tgit-dug-merge.html)
and the [native probes](branch-merge-probes.md).

## Established constraints

- A source is read-only even when checked out elsewhere. Current-destination
  actions must retain source/current HEAD identities and repository generation.
  Tip-changing actions refuse another worktree's reservation; no-op actions need
  not move anything. No implicit checkout, stash, retry or rollback.
- `--ff` permits a three-way merge when necessary; it is not `--ff-only`.
- `--no-overwrite-ignore` does not protect every three-way/squash collision on
  Git 2.51.1. Input-tree unions miss directory-rename output paths.
- Native `merge-tree --write-tree` supplies checkout paths, including conflicts.
  Exit 1 is a valid conflicting result. Private object storage with the original
  object store as an alternate keeps generated objects outside the repository.
  Retain inherited alternates and Git's C-quoted path-list syntax.
- Native merge previews may run configured low-level drivers. Run preview in an
  explicit merge review, not while merely rendering/selecting a branch. Cache
  its paths; final validation should not re-run drivers automatically.
- Validate the complete preview protocol: one result-tree record and only the
  documented conflict records on exit 1. Driver diagnostics on stdout must not
  be mistaken for a valid but incomplete result tree.
- Configured argument strings must not introduce unreviewed merge sources or
  dispatch abort/quit/continue. Tokenize as arguments without shell expansion;
  preserve supported options and fail clearly rather than guessing unknown ones.

## Native plan and remaining GUI boundary

Prepare an opaque source/current-HEAD plan using `BranchActionTarget`. Capture
the application's merge settings on the UI thread. Inspect effective native
settings, graph state and staged-index entries on the captured backend. Recheck
the exact inputs before preview and application. Keep command construction and
native result handling explicit rather than forwarding live GUI/Git access.
Native probes confirmed last-value semantics for `branch.<current>.mergeOptions`;
capture and validate that value, clear its implicit command-level application,
then pass the captured arguments explicitly before application-configured ones.
This prevents later or additional unreviewed merge sources and duplicate parsing.

Preview applicable paths only after the user chooses/reviews the operation.
Intersect them with native ignored paths, with case/prefix collision handling,
and fingerprint the affected bytes/modes. Present a Cancel-first risk review.
Validate the cached prediction's inputs and fresh filesystem risk snapshot at
launch; changed inputs require review, not automatic retry. The ort preview API
does not claim support for arbitrary other strategies. Preserve their arguments
and conservatively review all ignored files as potential overwrite risks when
an exact preview is unavailable for the configured strategy. Explain this in the
modal and require explicit approval; never silently substitute ort. For a true
fast-forward, the source tree supplies paths without running merge drivers. If
no ignored files exist, skip preview and require a fresh review if any appear
before launch. A failed or malformed ort preview remains an error, not permission
to proceed with an incomplete prediction.

For squash-and-commit, explicitly review any already-staged changes before they
can enter the commit. Retain an index snapshot across phases, and recheck HEAD,
reservation and staged content before the commit. If squash adds no staged
changes, do not accidentally commit unrelated existing changes. A source already
in current history likewise must not trigger an unrelated-only commit.

Handle regular merge, successful squash and conflicted squash separately.
Squash conflicts can lack `MERGE_HEAD`; do not offer an invalid native merge
abort or fall through to commit merely because a conflict prompt returned nil.
Retain native partial state and the reviewed message when completion needs
explicit conflict resolution. Terminal execution must preserve signing and
configured editor interaction, with repository checks at every queued boundary.

The native probes establish that a clean unborn destination supports an ordinary
fast-forward, while forced no-FF and squash fail. Preserve those distinctions.
The native review, merge plan and separately prepared squash-commit plan now
exist in `branch_action_merge_{review,plan,commit}.go`. They bind configuration
and staged entries, preserve configured message/signing/editor choices, and
construct immutable-source commands with autostash and lock retries disabled.
The post-squash plan rejects conflicts and unchanged staged results; its queued
commit checks the fresh staged snapshot again. A queued normal-merge abort
checks current HEAD plus `MERGE_HEAD` and `ORIG_HEAD`, rather than resolving a
possibly moved source branch again. The repository-bound GUI routing and its
acceptance checks remain necessary before this audit is complete.

Native option folding follows Git's callbacks, including their non-obvious
reset behavior: `--no-strategy` does nothing, and `--no-message` clears the
buffer while retaining the indication that a custom message was supplied.
Squash disables only the default automatic commit after option parsing; it
conflicts with an explicit `--commit` in either order. Disabling squash restores
the default commit policy unless a separate `--no-commit` remains effective.
See [Git 2.51.1's merge implementation](https://github.com/git/git/blob/v2.51.1/builtin/merge.c).
