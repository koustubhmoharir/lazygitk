# Selected-branch sync: native Git probe

Probe: [standalone Go source](probes/selected_branch_sync.go). It ran on Git
2.51.1 in disposable repositories with inherited `GIT_*` variables removed,
system/global config disabled, hooks disabled, signing disabled, terminal
prompts disabled, and cleanup limited to its owned temporary directory. This
records that Git version only; repeat on the project's minimum supported Git
before relying on version-sensitive details.

## Metadata and push route findings

`describe` reads selected-branch metadata and `topic@{push}` with HEAD on
`main`, then temporarily checks out `topic` only in its disposable fixture to
observe Git's native push defaults with `git push --dry-run --porcelain` and
`git push --dry-run --porcelain origin`. The table's no-argument push means no
remote/refspec arguments, with those dry-run and porcelain flags present; it
does not mutate remote refs. These checkout steps model native default
resolution, not the app's selected-branch workflow. Only the separate explicit
push fixture asserts a real push while selected branch differs from HEAD.

For the selected `refs/heads/topic`, `for-each-ref` with five NUL-delimited
fields produced:

```text
refs/heads/topic\0origin\0refs/heads/reviewed\0origin\0
```

The third field follows `branch.topic.remote` plus the distinct
`branch.topic.merge=refs/heads/reviewed`. The fourth/fifth describe the
configured push destination, with behaviors below. Empty fields remain empty;
the format avoids parsing human-oriented `git branch -vv` output. The gone
fixture deletes `refs/remotes/origin/reviewed`, the configured upstream's
tracking ref, and asserts that this command returns exactly `[gone]` with
status 0:

```sh
git for-each-ref '--format=%(upstream:track)' refs/heads/topic
```

The five metadata fields still return the same configured remote and merge
name after that deletion. Push resolution with `push.default=simple` still
fails because the local and upstream branch names differ; that error alone
does not establish that the tracking ref is gone.

| Configuration / selected branch route | `%(push:remotename)` / `%(push:remoteref)` | `topic@{push}` / no-argument push |
| --- | --- | --- |
| `push.default=simple`, upstream `origin/reviewed` | `origin` / empty | `cannot resolve 'simple' push to a single destination`; push exits 128 because `topic` differs from `reviewed` |
| `push.default=upstream` | `origin` / empty | resolves `refs/remotes/origin/reviewed`; porcelain is `refs/heads/topic:refs/heads/reviewed` |
| `push.default=current` | `origin` / empty | resolves `refs/remotes/origin/topic`; porcelain targets same-named `refs/heads/topic` |
| `push.default=matching` | `origin` / empty | resolves `refs/remotes/origin/topic`; pushes every matching local branch (the fixture pushed `main`, `reviewed`, and `topic`) |
| `push.default=nothing` | `origin` / empty | `push has no destination`; porcelain push exits 128 with “You didn't specify any refspecs…” |
| `branch.topic.pushRemote=publish` | `publish` / empty | upstream remains `origin/reviewed`; no-argument push exits 128 because `publish` is not the upstream and no destination was specified |
| `remote.pushDefault=publish` | `publish` / empty | same triangular-workflow error; branch's upstream remains `origin/reviewed` |
| `branch.topic.remote=review` | `review` / empty | upstream `review/reviewed`; `topic@{push}` failed to resolve in this fixture; push to configured `review` succeeds to `reviewed` |
| `remote.review.push=refs/heads/reviewed:refs/heads/reviewed` with `branch.topic.pushRemote=review` | `review` / empty | selected branch is excluded: `push refspecs for 'review' do not include 'topic'`; no-argument push instead selects that configured explicit refspec and pushes local `reviewed` |
| `remote.review.push=refs/heads/topic:refs/heads/release` | `review` / `refs/heads/release` | `topic@{push}` fails to resolve in the fixture; porcelain pushes `topic:release` |
| Two `remote.review.push` values, `topic:release-one` and `topic:release-two` (all fully qualified) | `review` / `refs/heads/release-one`, status 0 | `topic@{push}` exits 128 with unknown revision; no-argument dry run exits 0 and lists both destinations |
| `remote.review.push=refs/heads/*:refs/heads/archive/*` | `review` / `refs/heads/archive/topic` | `topic@{push}` fails to resolve; porcelain uses the wildcard refspec and also pushes `main` and `reviewed` |
| `branch.topic.remote=.` and merge `refs/heads/reviewed` | `.` / empty | metadata names the local upstream correctly; push destination resolution failed because Git had no local tracking branch for the resulting destination |

`remote.pushDefault` and branch `pushRemote` select a remote; they do not
necessarily select a matching push refspec. A branch-specific push remote
overrides the global push default. `push.default=simple` rejects an upstream
whose branch name differs from the selected local branch, while `upstream`
allows that mapping. `current` and `matching` have materially different
scope; matching can update multiple destinations. An explicit configured
`remote.<name>.push` takes precedence over the ordinary default and may map a
different local branch, omit the selected branch, or wildcard-push many
branches.

For the two-destination case the exact configured values are
`refs/heads/topic:refs/heads/release-one` and
`refs/heads/topic:refs/heads/release-two`. The five-field query returned status
0 and this record (NULs shown escaped):

```text
refs/heads/topic\0origin\0refs/heads/reviewed\0review\0refs/heads/release-one
```

`git rev-parse --symbolic-full-name 'topic@{push}'` returned status 128 with
`fatal: ambiguous argument 'topic@{push}': unknown revision or path not in the
working tree.` The no-argument dry run returned status 0 with these porcelain
rows (tabs shown escaped):

```text
*\trefs/heads/topic:refs/heads/release-one\t[new branch]
*\trefs/heads/topic:refs/heads/release-two\t[new branch]
```

Passing `origin` explicitly instead returned status 0 with
`=\trefs/heads/topic:refs/heads/reviewed\t[up to date]`. Thus a single nonempty
`%(push:remoteref)` does not prove that the configured push has a unique
destination. Choosing how the app presents multiple destinations remains a
design decision; the probe does not silently select the first one.

The same `for-each-ref` format and `topic@{push}` are not interchangeable:
`%(push:remotename)` / `%(push:remoteref)` return empty ref fields for several
implicit-default cases, while `@{push}` can resolve the effective destination,
fail for ambiguous/missing routes, or fail because of the local-remote setup.
Treat nonzero status and stderr as meaningful results; do not infer a remote
tracking ref from the remote name. Tracking storage follows configured fetch
refspecs and may use a nonstandard namespace.

## Explicit operations and safety checks

With HEAD on `main` and `topic` selected, this command pushed only the chosen
branch and named destination in porcelain output:

```sh
git push --porcelain origin refs/heads/topic:refs/heads/topic
```

The fixture stages `main.txt` with `staged blob\n`, then changes its worktree
bytes to `unstaged tracked edit\n` and adds `untracked.txt` containing
`untracked bytes\n`. Before pushing it asserts porcelain status exactly
`MM main.txt\n?? untracked.txt`. The bare remote's `main` and `other` refs
stayed unchanged. Snapshots taken before the push assert byte equality of
`.git/HEAD`, the raw `.git/index`, and all worktree files, as well as unchanged
HEAD OID, every local branch ref, staged tree, and dirty status. No checkout or
hard reset intervenes between dirty setup and these preservation assertions.
After a
simulated concurrent remote tip update, an explicit expected-OID lease rejected
the push with the exact porcelain row
`!\trefs/heads/topic:refs/heads/topic\t[rejected] (stale info)` and exit status
1. The probe also asserts that the remote `topic` tip remains the concurrent
`mainTip` OID after rejection, rather than merely accepting any push rejection:

```sh
git push --porcelain \
  --force-with-lease=refs/heads/topic:<observed-remote-oid> \
  origin refs/heads/topic:refs/heads/topic
```

The fetch fixture also configures a second mapping for the same source:

```text
remote.origin.fetch=+refs/heads/*:refs/remotes/origin/*
remote.origin.fetch=+refs/heads/topic:refs/cache/extra/topic
```

It seeds `refs/cache/extra/topic` at the old topic tip and advances the remote
topic. It retains an unrelated stale tracking ref and enables
`remote.origin.prune=true`, `remote.origin.tagOpt=--tags`, and
`submodule.recurse=true`. The exact command under investigation is:

```sh
git fetch --refmap= --no-tags --no-recurse-submodules \
  --no-write-fetch-head --no-prune origin \
  +refs/heads/topic:refs/remotes/origin/topic
```

Git's [2.51.1 fetch documentation, `--refmap`](https://git-scm.com/docs/git-fetch/2.51.1#Documentation/git-fetch.txt---refmapltrefspecgt)
explains that configured fetch refspecs can map explicitly fetched sources to
additional local destinations. An explicit source/destination refspec alone
does not suppress these configured mappings. Empty `--refmap=` disables that
additional mapping. `--no-write-fetch-head` preserves existing `FETCH_HEAD`,
and `--no-prune` overrides configured pruning. This extends the earlier
single-mapping fixture, whose simpler command did not exercise these hazards.

Observed on Git 2.51.1: exit status 0, with only `topic -> origin/topic`
reported as updated. The probe asserts the desired tracking ref equals the
remote's new topic OID, while `refs/cache/extra/topic` remains at the old OID.
A comparison of every ref before and after allows exactly that one desired
ref change, including preservation of the unrelated stale ref despite enabled
remote pruning. The existing nonempty `.git/FETCH_HEAD` is byte-for-byte
unchanged, and the tag remains absent. All assertions passed after formatting
and one standalone probe run for this extension.

The destination namespace above is only the fixture's known configured
namespace. A real operation must resolve and retain its intended tracking
destination using the active fetch refspec; `refs/remotes/<remote>/<branch>`
is not a universal mapping. `--no-tags` suppresses configured tag following;
`--no-follow-tags` suppresses `push.followTags`. Inspect or explicitly account
for `remote.<name>.mirror`, whose mirror behavior can widen a push beyond one
selected ref.

An additional mirror-mode probe on Git 2.51.1 confirmed that combining an
explicit refspec with `remote.origin.mirror=true` exits 128 rather than silently
widening the push. Adding `--no-mirror` still exits 128 with the same
incompatibility. A branch-scoped action therefore cannot treat `--no-mirror`
as a reliable override of configured mirror mode; it must reject that route
or obtain explicit consent for a supported command-scoped configuration change.

## Design context and limits

TortoiseGit's [Push](https://tortoisegit.org/docs/tortoisegit/tgit-dug-push.html),
[Pull and Fetch](https://tortoisegit.org/docs/tortoisegit/tgit-dug-pull.html),
and [Sync](https://tortoisegit.org/docs/tortoisegit/tgit-dug-sync.html)
documentation presents a selected branch's remote-tracked branch as a
preselection and lets the user choose the remote branch; push can separately
set tracking after success. Its pull dialog also exposes tag behavior. This
supports keeping selected local branch, pull source, and push destination
explicit and visibly distinct. This probe does not settle app prompts, how to
handle missing/gone upstreams, the user's choice to set upstream after a push,
or policy for mirror remotes; those remain design decisions for the tracker
workflow.
