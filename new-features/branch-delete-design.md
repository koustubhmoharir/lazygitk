# Captured branch deletion (current item 7)

This is the remaining deletion audit, not a later roadmap item. Preserve local,
remote and both choices, multiple selection, unmerged-commit warnings and native
partial failures. [TortoiseGit's reference browser](https://tortoisegit.org/docs/tortoisegit/tgit-dug-browse-ref.html)
provides selected-ref deletion and explicit tracking metadata. Native evidence
is maintained in [branch-delete-probes.md](branch-delete-probes.md).

- Copy selected names/full refs on entry and retain the backend and repository
  generation through prompts, workers and refresh completion. Read native tips,
  upstream metadata and worktree reservations; do not trust cached model flags.
- Local deletion refuses every checkout/operation reservation. It must never
  offer to detach or remove another worktree to make deletion possible. A bare
  common repository's HEAD is not itself a checkout reservation.
- Retain the existing mergedness criterion: selected history is contained in
  current HEAD, its stored upstream or a configured main branch. Capture the
  exact commits used for that assessment; an unmerged source needs explicit
  force approval. Missing/ambiguous metadata must not silently weaken warnings.
  A ref scheduled for deletion cannot count as evidence that history remains
  reachable: exclude the selected local refs and related tracking refs removed
  by a reviewed remote deletion. This also avoids counting a branch as merged
  into itself. Bind these exclusions to the compound review before approval.
- Bind local deletion to literal branch names, using native `branch -D -- ...`
  only after the applicable approval. Recheck source/configuration/reservations
  immediately before launch. Preserve Git's concurrency and partial errors;
  add no repository/ref locks, retry or rollback.
- Local-branch upstream deletion must use native remote ref names, including
  remapped tracking storage. A remote-pane ref needs the inverse configured
  fetch mapping; its display suffix is not proof of a remote branch name.
  Collect every possible source and require explicit choice when ambiguous.
  Never silently take the first upstream merge entry or fetch mapping.
- Remote deletion names a configured remote and exact full branch refs. Review
  destinations and retain configuration across launch, suppressing unrelated
  configured push mappings, mirroring and followed tags. Cached tips are not a
  claim about the server's current state. Local-upstream `.` and non-head refs
  must not be silently converted into remote branch deletions.
- For both, delete remotely first and retain local branches if any remote step
  fails. Multiple push URLs/remotes may partially succeed; report the native
  error and refresh actual state. Do not retry earlier successful deletions or
  invent an atomic transaction. A later local failure also leaves actual state.

The native review/command APIs now implement these checks in
`branch_action_delete_{local,remote,choices,context}.go`. Live GUI routing and
its acceptance checks remain in progress; the tracker retains the deletion
subtask until those pass.
