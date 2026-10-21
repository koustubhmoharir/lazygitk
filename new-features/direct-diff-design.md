# Direct historical comparisons

Current roadmap item: 3. The agreed endpoints and edge cases remain in the
roadmap; this records implementation seams, not additional feature scope.

[TortoiseGit's log dialog](https://tortoisegit.org/docs/tortoisegit/tgit-dug-showlog.html)
offers previous-version and working-tree comparisons separately. The fork
follows that distinction, without adding a target-choice menu before either.

## Backend and lifetime

- Add `git_commands/historical_diff.go` alongside the existing sticky difftool
  builder. Resolve a captured full OID to a commit and first parent. Root commits
  have no previous action; merge comparisons use only the first parent.
- Directory/whole-commit comparisons use literal pathspecs and fixed endpoints
  through Git's configured difftool. Include both recognized rename paths.
  Do not read or mutate sticky comparison mode.
- Single historical file versus worktree uses exact NUL-delimited literal
  `ls-tree` lookup, validates the blob mode/OID, then streams `cat-file blob`
  bytes into a private temporary directory. Preserve the basename/extension,
  close the source and mark it read-only before launching the tool. Reject
  symlink/gitlink sources rather than silently change their semantics.
- Require an existing regular destination inside the current worktree. Resolve
  relative input from that root, including linked worktrees; reject symlink
  traversal, paths outside the root and Git metadata. Accept ignored/untracked
  files. Revalidate at launch, never restore/stage files to make them comparable.
- A prepared comparison owns its temporary directory and an idempotent cleanup
  method. The synchronous terminal subprocess owns its lifetime until return.
  Detached custom tools must use their normal wait option; their returning
  early cannot keep temporary inputs alive indefinitely.
- Trusted no-index difftool accepts Git 0/1 only for that prepared command.
  `CmdObj` carries an opt-in subprocess success-status list; the GUI normalizes
  immediately after execution and before the generic return/error prompt.
  Existing runners and ordinary commands retain their error policy. Tool
  failures become Git 128 and remain errors; adapters are required for tools
  whose own successful result is nonzero.

## UI and shortcuts

- Add separate bindings in the shared commit controller and commit-file
  controller, with stable IDs, single-selection guards and bottom hints for
  working-tree comparison. Preserve the existing sticky external comparison.
- Capture the full commit OID and canonical historical path before prompting
  or waiting. Reject placeholder/range selections. Missing worktree paths must
  explain the separate arbitrary-file action; deleted historical entries do
  not silently substitute parent content.
- The explicit-file action starts a prompt naming the historical source and
  defaults to its working path. Completion browses existing filesystem entries
  (not only tracked `ls-files` output); directories help navigation but cannot
  be confirmed as single-file destinations. Paths are data, not shell text.
- Reserve `w/W` for working tree and `v/V` for previous commit. The default
  overlay removes their old worktree/sticky-diff/range-selection/paste/WIP
  assignments, leaving those described commands in the palette. The complete
  keymap milestone will supply their new shortcuts. Explicit loaded config
  remains authoritative; do not rewrite user files. The arbitrary-file action
  initially has no default shortcut and remains discoverable in the palette.

## Verification

Use argument/selection unit tests, real disposable repositories and a fake
difftool that records the actual inputs. Cover normal versus trusted exit
statuses, binary/unusual names, tracked/untracked/ignored files, first-parent
merges/root commits, rename/deletion, missing/unsupported/outside targets,
cancel/failure cleanup, dirty worktree/index preservation and sticky isolation.
Run serialized headless scenarios through the normal terminal lifecycle; no
external GUI tool is required.
