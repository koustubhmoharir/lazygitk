# Compare a historical file with an explicitly selected worktree file

## Scope and current wiring

The requested behavior is to compare a file from a selected historical commit
with any explicitly selected existing file in the working directory. The
working file may be untracked, and the paths need not be recognized as a rename
pair.

`CommitFilesController.openDiffTool` currently passes one path plus commit
references to `DiffCommands.OpenDiffToolCmdObj`
([commits_files_controller.go](../pkg/gui/controllers/commits_files_controller.go#L451),
[diff.go](../pkg/commands/git_commands/diff.go#L211)). The builder invokes
`git difftool --no-prompt [commit refs] -- PATH`; this asks Git to compare a
path across repository revisions, so it cannot express two unrelated paths
with one path selector. `CommitFileNode` embeds the generic tree node and adds
no historical-blob or rename resolution behavior of its own
([commit_file_node.go](../pkg/gui/filetree/commit_file_node.go#L5)).

The normal diff subprocess lifecycle suspends the UI, runs the command with
terminal stdin/stdout/stderr, resumes the UI, and treats any nonzero command
exit as an error ([gui.go](../pkg/gui/gui.go#L1100)). The existing temporary
merge-file helper uses `os.CreateTemp`, writes bytes, and defers removal around
the consuming operation ([working_tree_helper.go](../pkg/gui/controllers/helpers/working_tree_helper.go#L326)).

## Candidate argv design

Resolve the selected historical entry to a blob object ID and mode, write the
blob bytes to a uniquely named temporary file, then invoke Git’s file-pair
mode with the temporary path first and the selected worktree path second:

```text
git -C REPO difftool --no-index --no-prompt -- TEMP_BLOB REAL_WORKTREE_PATH
```

Run from the repository root (as the probe did) so repository-local
`diff.tool` and `difftool.<tool>.cmd` configuration is found. Preserve each
argument as its own argv element; do not concatenate or shell-quote the paths
into the Git command. The user-configured difftool command itself is interpreted
by Git according to its normal configuration rules.

For blob lookup, prefer an exact path-limited tree lookup with Git's literal
pathspec mode and NUL-delimited output, for example
`git --literal-pathspecs ls-tree -rz --full-tree COMMIT -- PATH`. Take the
object ID and mode from the matching entry, then stream the object with
`git cat-file blob OBJECT_ID` into a temp file. This avoids treating a pathname
as shell text and avoids relying on `COMMIT:PATH` parsing for unusual names.
The repo uses `ls-tree` for a path-limited lookup in
`pkg/commands/git_commands/rebase.go:529`, but no existing NUL-safe exact-path
helper was found. The suggested combination and unusual-path behavior were not
probed. `CommitCommands.ShowFileContentCmdObj` uses `git show HASH:PATH`
(`pkg/commands/git_commands/commit.go:261`); its revision-plus-path syntax
should not be assumed safe for arbitrary pathnames without further checking.

The proposed order makes the historical blob `$LOCAL` and the explicitly
selected worktree file `$REMOTE` under Git’s difftool conventions. If product
orientation should instead place the current file on the left, reverse the
arguments (or request `-R`); this is a product choice, not established by the
probe.

## Probe: tool configuration, spaces, exit status, and side effects

The initial configuration/path probe used disposable repository
`/tmp/tmp.YE92PjnfVo`; the exit matrix used disposable repository
`/tmp/tmp.tuFRXJUg30`. Neither changed project files. Both configured local
`diff.tool=probe` and `difftool.probe.cmd` to invoke a fake executable with
`"$LOCAL"` and `"$REMOTE"` arguments. For the matrix, the executable appended
its two arguments and `PROBE_TOOL_EXIT` to a log, then exited with that code.

```sh
probe=$(mktemp -d)
git -C "$probe" init -q
git -C "$probe" config user.name Probe
git -C "$probe" config user.email probe@example.invalid
mkdir -p "$probe/with spaces"
printf 'blob side\n' > "$probe/blob side.txt"
printf 'work side\n' > "$probe/with spaces/real file.txt"
printf '#!/bin/sh\nprintf "LEFT=<%%s>\\nRIGHT=<%%s>\\n" "$1" "$2" > "$PROBE_LOG"\n' > "$probe/fake-diff"
chmod +x "$probe/fake-diff"
export PROBE_LOG="$probe/invocation.log"
git -C "$probe" config diff.tool probe
git -C "$probe" config difftool.probe.cmd '"$PROBE_FAKE" "$LOCAL" "$REMOTE"'
export PROBE_FAKE="$probe/fake-diff"
git -C "$probe" difftool --no-index --no-prompt "$probe/blob side.txt" "$probe/with spaces/real file.txt"
```

Observed output:

```text
status=1
LEFT=</tmp/tmp.YE92PjnfVo/blob side.txt>
RIGHT=</tmp/tmp.YE92PjnfVo/with spaces/real file.txt>
```

The probe matrix and results (Git 2.51.1):

| Pair | Fake tool exit | Git args | Git status | Tool invoked? |
|---|---:|---|---:|---|
| equal bytes | 0 | `--no-index --no-prompt` | 0 | no |
| different bytes | 0 | `--no-index --no-prompt` | 1 | yes |
| different bytes | 1 | `--no-index --no-prompt` | 1 | yes |
| different bytes | 42 | `--no-index --no-prompt` | 1 | yes |
| different bytes | 0 | `--no-index --no-prompt --trust-exit-code` | 1 | yes |
| different bytes | 1 | `--no-index --no-prompt --trust-exit-code` | 128 | yes |
| different bytes | 42 | `--no-index --no-prompt --trust-exit-code` | 128 | yes |

Thus the default mode cannot distinguish a normal differing result from a
fake-tool exit 1 or 42 using Git's process status: all unequal-file cases
returned 1. `--trust-exit-code` did not preserve nonzero tool statuses above
1; for both fake exits 1 and 42 it returned 128 and emitted `fatal: external
diff died, stopping at /tmp/tmp.tuFRXJUg30/left`. Exit 0 returned 1 for a
differing pair because Git still reports the no-index pair as different. Equal
files returned 0 without invoking the external tool. So ordinary status 1 is
distinguishable from equal inputs by Git status, but not from tool failure
status 1 in default mode. Tool exit codes greater than 1 are not preserved by
either tested mode. With trust enabled, the caller can distinguish normal
differences from a tool failure for tools whose success code is 0: accept Git
0/1 and report 128 as failure. The remaining ambiguity is a tool that uses its
own exit 1 to mean a successful comparison; such a tool needs an explicit
adapter/success-code policy rather than unconditional suppression of Git errors.

The exit-matrix setup and commands followed this shape (the `run_case`
function records the Git process status and whether the fake executable logged
an invocation):

```sh
probe=$(mktemp -d)
git -C "$probe" init -q
git -C "$probe" config user.name Probe
git -C "$probe" config user.email probe@example.invalid
printf 'same\n' > "$probe/left"
printf 'same\n' > "$probe/right"
printf '#!/bin/sh\nprintf "exit=<%%s> left=<%%s> right=<%%s>\\n" "$PROBE_TOOL_EXIT" "$1" "$2" >> "$PROBE_LOG"\nexit "$PROBE_TOOL_EXIT"\n' > "$probe/fake-diff"
chmod +x "$probe/fake-diff"
export PROBE_LOG="$probe/log" PROBE_FAKE="$probe/fake-diff"
git -C "$probe" config diff.tool probe
git -C "$probe" config difftool.probe.cmd '"$PROBE_FAKE" "$LOCAL" "$REMOTE"'

run_case() {
  label=$1
  shift
  : > "$PROBE_LOG"
  git -C "$probe" difftool --no-index --no-prompt "$@" \
    "$probe/left" "$probe/right" >/dev/null 2>&1
  status=$?
  if test -s "$PROBE_LOG"; then invoked=yes; else invoked=no; fi
  printf '%s status=%s invoked=%s\n' "$label" "$status" "$invoked"
}

export PROBE_TOOL_EXIT=0
run_case equal_exit0
printf 'different\n' > "$probe/right"
run_case different_exit0
export PROBE_TOOL_EXIT=1
run_case different_exit1_default
run_case different_exit1_trust --trust-exit-code
export PROBE_TOOL_EXIT=42
run_case different_exit42_default
run_case different_exit42_trust --trust-exit-code
```

The same generated fake tool was rerun after setting `PROBE_TOOL_EXIT` to 0,
1, and 42; before each case its log was truncated, and invocation was inferred
from whether the log became nonempty. In the initial path probe, the two
untracked files had spaces in their names, and the logged `$LOCAL` and `$REMOTE`
were intact absolute paths. After each probe, `git status --short` showed the
inputs, fake tool, and log as untracked, while `git diff --cached --exit-code`
returned 0. This establishes that Git did not stage or modify the index or
inputs in these runs. A configured external tool can itself have arbitrary
side effects; that was not tested.

`guiCommon.RunSubprocess` delegates directly to
`Gui.runSubprocessWithSuspense` (`pkg/gui/gui_common.go:53`). That function
returns `(false, cmdErr)` for any nonzero subprocess result and `(true, nil)`
only for status 0 (`pkg/gui/gui.go:1100`). `runSubprocess` also prompts to
return when the command errors (unless running an integration test), subject to
`PromptToReturnFromSubprocess` (`pkg/gui/gui.go:1122`). Consequently status 1
(normal unequal pair and fake-tool 1 under default mode) and status 128
(fake-tool errors with trust enabled) become errors to the caller; status 1
also triggers the error prompt. `--no-ext-diff` was not added to the probe:
it may interact with the configured difftool path, and no evidence here shows
that `diff.external` intercepts `git difftool --no-index`. Whether an explicit
option is needed remains unverified.

## Semantics and lifecycle questions for implementation

Coordinator disposition: the main plan fixes the historical version on the
left, uses existing files inside the worktree on the right, labels deletion
fallbacks explicitly, and leaves unsupported symlink/gitlink cases disabled
until their semantics are defined. It proposes trusted tool exit codes with
normal Git difference status handled before the generic subprocess prompt.
The points below record remaining technical coverage limits, not additional
unanswered product questions for the user.

- **Historical deletion:** a deleted entry in the selected commit has no blob
  to extract. The request says “selected historical commit file,” which may
  exclude deletions, but selection/disabled-action behavior is not resolved
  here. A missing object must not silently become an empty file.
- **Worktree existence:** the requested right side is an explicitly selected
  existing working-directory file, including untracked. Validate it at launch
  time because it may disappear between selection and invocation. A path that
  exists only in the index must not qualify as the working-directory side.
- **Symlinks:** Git stores a symlink entry as a blob containing the link target;
  materializing those bytes as an ordinary temp file compares target text, not
  a filesystem symlink. A dereferenced/real symlink comparison would be
  different semantics. The desired behavior remains unspecified.
- **Binary blobs:** writing raw blob bytes to a temp file preserves content and
  lets the chosen external tool decide how to display binary data. Avoid
  newline or text conversions. Temp file permissions may affect tools that
  expect executable bits; the historical tree mode is available from the tree
  entry if that matters. No binary-specific tool probe was run.
- **Temp lifetime:** keep the historical temp file through the complete
  synchronous `RunSubprocess` call and remove it afterward, including error
  paths. If a configured tool launches asynchronously and returns before it
  finishes reading inputs, that tool may outlive this lifetime; Git’s
  `difftool` normally waits, but this was not independently tested for detached
  custom tools. Use a unique temp file in the configured temp directory and
  close it before launching Git.
- **Path interpretation:** pass `--` before the pair so paths beginning with
  `-` are not options. Absolute paths (or paths made absolute from the repo
  root) avoid ambiguity from the child process’s working directory. The
  selected worktree path may be outside the repository if the UI supports
  such paths; that is not established by the inspected call sites.
- **No-index configuration:** the probe confirms the requested Git version and
  environment honors configured difftool selection in this mode. Behavior
  across older supported Git versions was not checked.

## Source paths inspected

- `pkg/gui/controllers/commits_files_controller.go` — current historical file
  diff-tool action.
- `pkg/commands/git_commands/diff.go` — diff-tool options and argv builder;
  temporary-file probe precedent.
- `pkg/gui/filetree/commit_file_node.go` — commit-file node wrapper.
- `pkg/gui/controllers/helpers/diff_helper.go` — neighboring diff subprocess
  call pattern.
- `pkg/gui/gui.go` — subprocess suspend/resume, terminal streams, and exit
  error behavior.
- `pkg/gui/controllers/helpers/working_tree_helper.go` — temp-file creation,
  close, and cleanup pattern.
- `pkg/commands/git_commands/patch.go` — existing temp-file helper context.
- `pkg/gui/gui_common.go` — `RunSubprocess` delegation and return values.
- `pkg/commands/git_commands/commit.go` — current `git show HASH:PATH`
  content lookup.
- `pkg/commands/git_commands/rebase.go` — path-limited `git ls-tree` use.
