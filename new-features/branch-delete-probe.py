#!/usr/bin/env python3
"""Bounded native Git deletion probes; all repositories live in a fresh tempdir."""
import json
import os
import subprocess
import tempfile
from pathlib import Path

ROOT = Path(tempfile.mkdtemp(prefix="lazygitk-branch-delete-probe-"))
LOG = []


def run(cwd, *args, check=False, observe=False):
    argv = ["git", *( ["--no-optional-locks"] if observe else []), *map(str, args)]
    p = subprocess.run(argv, cwd=cwd, text=True,
                       stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    LOG.append((str(cwd), argv, p.returncode,
                p.stdout, p.stderr))
    if check and p.returncode:
        raise RuntimeError(f"{args}: {p.stderr}")
    return p


def repo(name, bare=False):
    p = ROOT / name
    p.mkdir()
    run(ROOT, "init", "--bare" if bare else "--initial-branch=main", p, check=True)
    return p


def init_repo(name):
    p = repo(name)
    run(p, "config", "user.name", "Probe", check=True)
    run(p, "config", "user.email", "probe@example.invalid", check=True)
    (p / "base.txt").write_text("base\n")
    run(p, "add", "base.txt", check=True)
    run(p, "commit", "-m", "base", check=True)
    return p


def state(p):
    fields = {}
    for args in [("symbolic-ref", "-q", "HEAD"), ("rev-parse", "HEAD"),
                 ("for-each-ref", "--format=%(refname) %(objectname)", "refs/heads"),
                 ("status", "--porcelain=v2", "--untracked-files=all"),
                 ("config", "--local", "--list", "--show-origin")]:
        q = run(p, *args, observe=True)
        fields[" ".join(args)] = (q.returncode, q.stdout, q.stderr)
    idx = run(p, "ls-files", "--stage", "--debug", observe=True)
    fields["index-entries"] = (idx.returncode, idx.stdout, idx.stderr)
    for name in ("index", "HEAD"):
        location = run(p, "rev-parse", "--path-format=absolute", "--git-path", name,
                       observe=True, check=True).stdout.strip()
        path = Path(location)
        fields[name + "-bytes-hex"] = path.read_bytes().hex() if path.exists() else None
    bare = run(p, "rev-parse", "--is-bare-repository", observe=True).stdout.strip() == "true"
    fields["bare"] = bare
    files = {} if bare else {x.relative_to(p).as_posix(): x.read_bytes().hex()
             for x in p.rglob("*") if x.is_file() and ".git" not in x.parts}
    fields["worktree-bytes-hex"] = files
    return fields


def snapshot(label, p):
    data = state(p)
    LOG.append(("SNAPSHOT", [label], 0, json.dumps(data), ""))
    return data


def refs(p):
    return set(run(p, "for-each-ref", "--format=%(refname)", observe=True).stdout.splitlines())


def assert_probe(condition, message):
    if not condition:
        raise AssertionError(message)
    LOG.append(("ASSERT", [message], 0, "PASS", ""))


def main():
    ver = subprocess.run(["git", "--version"], text=True, capture_output=True)
    LOG.append(("VERSION", ["git", "--version"], ver.returncode, ver.stdout, ver.stderr))

    # (a) name/OID ambiguity and per-argument partial branch deletion.
    p = init_repo("literal")
    sha = run(p, "rev-parse", "HEAD", check=True).stdout.strip()
    run(p, "branch", sha, check=True)
    snapshot("literal before SHA-named delete", p)
    run(p, "branch", "-D", "--", sha)
    assert_probe(f"refs/heads/{sha}" not in refs(p), "literal SHA-named branch was deleted")
    snapshot("literal after SHA-named delete", p)
    run(p, "branch", "-D", sha)
    snapshot("literal after un-delimited SHA argument", p)

    p = init_repo("partial")
    run(p, "branch", "occupied", check=True)
    run(p, "branch", "remove-me", check=True)
    run(p, "checkout", "occupied", check=True)
    (p / "base.txt").write_text("unstaged\n")
    (p / "untracked.txt").write_text("untracked\n")
    run(p, "add", "-N", "untracked.txt", check=True)
    before = snapshot("multi-delete before", p)
    multi_delete = run(p, "branch", "-D", "remove-me", "occupied", "missing")
    assert_probe(multi_delete.returncode != 0, "multi-delete reports native failure")
    assert_probe("refs/heads/remove-me" not in refs(p) and "refs/heads/occupied" in refs(p),
                 "multi-delete removed earlier eligible branch and retained occupied branch")
    assert_probe("refs/heads/missing" not in refs(p), "missing branch remains absent after multi-delete")
    after = snapshot("multi-delete after remove-me occupied missing", p)
    for key in ("HEAD-bytes-hex", "index-bytes-hex", "worktree-bytes-hex"):
        assert_probe(before[key] == after[key], "multi-delete preserved " + key)

    # (b) bare/common HEAD is not a worktree reservation; linked checkout and
    # an actually paused rebase reserve branches through Git's native state.
    p = init_repo("bare-seed")
    bare = repo("bare", bare=True)
    run(p, "push", bare, "main:main", check=True)
    run(bare, "symbolic-ref", "HEAD", "refs/heads/main", check=True)
    run(bare, "branch", "-D", "main")
    assert_probe("refs/heads/main" not in refs(bare), "bare HEAD-named branch deletion succeeded")
    snapshot("bare after deleting branch named by bare HEAD", bare)
    p = init_repo("linked")
    run(p, "branch", "reserved", check=True)
    linked = ROOT / "linked-wt"
    run(p, "worktree", "add", linked, "reserved", check=True)
    snapshot("linked before reserved delete", p)
    run(p, "branch", "-D", "reserved")
    assert_probe("refs/heads/reserved" in refs(p), "linked-worktree branch deletion was refused")
    snapshot("linked after reserved delete attempt", p)
    p = init_repo("active-rebase")
    base = run(p, "rev-parse", "HEAD", check=True).stdout.strip()
    run(p, "branch", "source", check=True)
    run(p, "branch", "unreserved", check=True)
    (p / "target.txt").write_text("target\n")
    run(p, "add", "target.txt", check=True)
    run(p, "commit", "-m", "target", check=True)
    target = run(p, "rev-parse", "HEAD", check=True).stdout.strip()
    run(p, "checkout", "source", check=True)
    (p / "source.txt").write_text("source\n")
    run(p, "add", "source.txt", check=True)
    run(p, "commit", "-m", "source", check=True)
    run(p, "checkout", "main", check=True)
    rebase_wt = ROOT / "active-rebase-wt"
    run(p, "worktree", "add", rebase_wt, "source", check=True)
    editor = ROOT / "insert-break-editor.sh"
    editor.write_text("#!/bin/sh\n{ printf 'break\\n'; cat \"$1\"; } > \"$1.new\"\nmv \"$1.new\" \"$1\"\n")
    editor.chmod(0o755)
    env = {"GIT_SEQUENCE_EDITOR": str(editor)}
    q = subprocess.run(["git", "rebase", "-i", "--onto", target, base, "source"],
                       cwd=rebase_wt, env={**os.environ, **env}, text=True,
                       stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    LOG.append((str(rebase_wt), ["git", "rebase", "-i", "--onto", target, base, "source"], q.returncode, q.stdout, q.stderr))
    snapshot("active rebase after sequence-editor break", rebase_wt)
    for rel in (".git/rebase-merge/head-name", ".git/rebase-merge/onto"):
        # In a linked worktree .git is a file, so ask Git for its private dir.
        gitdir = run(rebase_wt, "rev-parse", "--git-dir", check=True).stdout.strip()
        path = Path(gitdir) / rel.split("/", 1)[1]
        value = path.read_text() if path.exists() else "MISSING\n"
        LOG.append(("FIXTURE", [f"read {path}"], 0, value, ""))
        assert_probe(value.strip() == ("refs/heads/source" if rel.endswith("head-name") else target),
                     f"native active rebase metadata matches {rel}")
    assert_probe(run(rebase_wt, "symbolic-ref", "-q", "HEAD", observe=True).returncode != 0,
                 "paused rebase worktree HEAD is detached")
    run(p, "branch", "-D", "--", "source")
    assert_probe("refs/heads/source" in refs(p), "branch participating in paused rebase was retained")
    snapshot("main worktree attempt to delete branch reserved by paused linked rebase", p)
    run(p, "branch", "-D", "--", "unreserved")
    assert_probe("refs/heads/unreserved" not in refs(p), "unrelated branch deletion succeeded during paused rebase")
    snapshot("main worktree delete unrelated branch during paused linked rebase", p)

    # (c) named-remote fetch remapping and multiple merge values.
    p = init_repo("tracking")
    tracking_remote = repo("tracking-remote", bare=True)
    run(p, "push", tracking_remote, "main:refs/heads/review", check=True)
    run(p, "remote", "add", "named", tracking_remote, check=True)
    run(p, "config", "--unset-all", "remote.named.fetch", check=True)
    run(p, "config", "--add", "remote.named.fetch", "+refs/heads/review:refs/custom/cache/review", check=True)
    run(p, "config", "branch.topic.remote", "named", check=True)
    run(p, "config", "branch.topic.merge", "refs/heads/review", check=True)
    run(p, "config", "--add", "branch.topic.merge", "refs/heads/other", check=True)
    run(p, "update-ref", "refs/heads/topic", "HEAD", check=True)
    run(p, "fetch", "named", check=True)
    run(p, "for-each-ref", "--format=%(refname)|%(upstream)|%(upstream:short)|%(upstream:remotename)|%(upstream:remoteref)", "refs/heads/topic")
    run(p, "config", "--get-all", "branch.topic.merge")
    run(p, "config", "--get-regexp", r"^branch\.topic\.")
    upstream = run(p, "for-each-ref", "--format=%(upstream)|%(upstream:remotename)|%(upstream:remoteref)", "refs/heads/topic", observe=True).stdout.strip()
    assert_probe(upstream == "refs/custom/cache/review|named|refs/heads/review",
                 "named remote upstream resolves remapped tracking ref and original remote ref")
    snapshot("tracking metadata", p)

    # (d) explicit delete push, mirror config, and extra push refspec isolation.
    local = init_repo("push-local")
    bare = repo("push-bare", bare=True)
    run(local, "remote", "add", "named", bare, check=True)
    run(local, "push", "named", "main:refs/heads/exact", check=True)
    run(local, "push", "--no-follow-tags", "--delete", "--", "named", "refs/heads/exact")
    run(local, "push", "named", "main:refs/heads/exact", check=True)
    run(local, "push", "named", "main:refs/heads/unrelated", check=True)
    run(local, "config", "remote.named.mirror", "true", check=True)
    run(local, "config", "--add", "remote.named.push", "refs/heads/*:refs/heads/copy/*", check=True)
    run(local, "tag", "-a", "-m", "annotated probe tag", "probe-tag", check=True)
    run(local, "config", "push.followTags", "true", check=True)
    refs_before_delete = refs(bare)
    snapshot("remote before explicit mirror-suppressed delete", bare)
    run(local, "-c", "remote.named.mirror=false", "push", "--no-follow-tags", "--delete", "--", "named", "refs/heads/exact")
    assert_probe(refs(bare) == refs_before_delete - {"refs/heads/exact"},
                 "mirror-suppressed explicit delete changed only exact branch ref")
    assert_probe(run(local, "show-ref", "--verify", "--quiet", "refs/tags/probe-tag").returncode == 0,
                 "explicit remote deletion left local tag intact")
    snapshot("remote after explicit mirror-suppressed delete", bare)

    # (e) two push URLs: successful first, rejecting second; one command only.
    local = init_repo("multiurl-local")
    first, second = repo("multiurl-first", bare=True), repo("multiurl-second", bare=True)
    run(local, "push", first, "HEAD:refs/heads/partial", check=True)
    run(local, "push", second, "HEAD:refs/heads/partial", check=True)
    assert_probe("refs/heads/partial" in refs(first) and "refs/heads/partial" in refs(second),
                 "partial branch was seeded at both push URLs")
    hook = second / "hooks" / "pre-receive"
    hook.write_text("#!/bin/sh\necho second-reject >&2\nexit 1\n")
    hook.chmod(0o755)
    run(local, "remote", "add", "multi", first, check=True)
    run(local, "config", "--unset-all", "remote.multi.pushurl")
    run(local, "config", "--add", "remote.multi.pushurl", first, check=True)
    run(local, "config", "--add", "remote.multi.pushurl", second, check=True)
    snapshot("multi-pushurl remotes before", first)
    snapshot("multi-pushurl remotes before", second)
    multi_push_delete = run(local, "push", "--no-follow-tags", "--delete", "--", "multi", "refs/heads/partial")
    assert_probe(multi_push_delete.returncode == 1, "multi-pushurl deletion returns native failure status 1")
    assert_probe("refs/heads/partial" not in refs(first) and "refs/heads/partial" in refs(second),
                 "multi-pushurl delete removed first ref and retained second after hook rejection")
    snapshot("multi-pushurl first after command", first)
    snapshot("multi-pushurl second after command", second)

    rawlog = ROOT / "log.jsonl"
    with rawlog.open("w") as f:
        for cwd, argv, code, stdout, stderr in LOG:
            f.write(json.dumps({"cwd": cwd, "argv": argv, "exit": code,
                                "stdout": stdout, "stderr": stderr}) + "\n")
    out = ["# Native branch deletion probe evidence", "",
           f"Git: `{ver.stdout.strip()}`. Fixtures: `{ROOT}` (temporary directory; disposable local repos only).",
           f"Complete command/output log: `{rawlog}` (JSONL, temporary). The evidence below keeps only review-relevant commands and compact state comparisons. Observational snapshots use `git --no-optional-locks`; snapshots include HEAD, refs, status, config, raw index and worktree bytes.", "",
           "## Findings for review", "",
           "- A full-SHA-looking branch name is deleted literally by `git branch -D -- <name>`. The delimiter keeps names out of option parsing.",
           "- Multi-branch `git branch -D remove-me occupied missing` reports failure after removing `remove-me`; `occupied` remains because it is checked out, and `missing` remains absent. HEAD, index and worktree snapshots are recorded before and after.",
           "- Deleting the branch named by bare repository HEAD succeeds. A branch checked out in a linked worktree is refused. The active-rebase fixture uses diverged source/target commits and an interactive sequence editor that inserts `break`; it records the native `rebase-merge/head-name` and `onto` files. Deletion is attempted from another worktree, followed by deletion of an unrelated branch to show the reservation boundary.",
           "- With `branch.topic.merge` set twice, `for-each-ref` exposes only the first merge value; `git config --get-all` exposes both. Native `%(upstream)` retains remapped local tracking storage, while `%(upstream:remoteref)` remains the full server ref.",
           "- Explicit `git push --no-follow-tags --delete -- named refs/heads/exact` succeeds. With mirror=true and an extra push refspec, the same explicit delete with `-c remote.named.mirror=false` deletes only `exact`; unrelated refs and the local tag are preserved.",
           "- A single `git push --no-follow-tags --delete -- multi refs/heads/partial` starts with that branch present at both push URLs. It exits 1 after the first local bare remote removes it and the second pre-receive hook rejects deletion, leaving the second ref intact.",
           "- TortoiseGit’s Browse All Refs guide describes context-menu deletion/renaming and local-branch tracked-branch metadata for the selected refs; it does not claim that deletion automatically removes another worktree: [Browse All Refs](https://tortoisegit.org/docs/tortoisegit/tgit-dug-browse-ref.html).", ""]
    keep_snapshots = {"literal before SHA-named delete", "literal after SHA-named delete",
                      "multi-delete before", "multi-delete after remove-me occupied missing",
                      "bare after deleting branch named by bare HEAD",
                      "linked after reserved delete attempt",
                      "active rebase after sequence-editor break",
                      "main worktree attempt to delete branch reserved by paused linked rebase",
                      "main worktree delete unrelated branch during paused rebase",
                      "tracking metadata", "remote before explicit mirror-suppressed delete",
                      "remote after explicit mirror-suppressed delete",
                      "multi-pushurl remotes before", "multi-pushurl first after command",
                      "multi-pushurl second after command"}
    for cwd, argv, code, stdout, stderr in LOG:
        if cwd == "SNAPSHOT":
            if argv[0] not in keep_snapshots:
                continue
            data = json.loads(stdout)
            refkey = "for-each-ref --format=%(refname) %(objectname) refs/heads"
            headkey = "symbolic-ref -q HEAD"
            statuskey = "status --porcelain=v2 --untracked-files=all"
            status = "bare (no worktree)" if data["bare"] else data[statuskey][1].strip() or "clean"
            out += [f"- **State — {argv[0]}:** HEAD `{data[headkey][1].strip() or '(detached)'}`; refs `{data[refkey][1].strip().replace(chr(10), '; ')}`; status `{status}`."]
        elif cwd == "ASSERT" or cwd == "FIXTURE" or (
                "branch -D" in " ".join(argv) or
                ("push" in argv and "--delete" in argv) or
                argv[:3] == ["git", "fetch", "named"] or
                ("for-each-ref" in argv and "refs/heads/topic" in argv) or
                "config --get-all" in " ".join(argv) or
                "config --get-regexp" in " ".join(argv) or
                "rebase -i" in " ".join(argv)):
            result = " ; ".join(x.strip().replace("`", "'") for x in (stdout, stderr) if x.strip())
            out += [f"- `{cwd}` `$ {subprocess.list2cmdline(argv)}` → exit {code}" + (f": `{result}`" if result else "")]
    (Path(__file__).with_name("branch-delete-probes.md")).write_text("\n".join(out))
    print(f"wrote {Path(__file__).with_name('branch-delete-probes.md')} ; fixtures {ROOT}")


if __name__ == "__main__":
    main()
