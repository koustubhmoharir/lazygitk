#!/usr/bin/env python3
"""Capture native branch-creation behavior in disposable repositories."""

import json
import os
import pathlib
import shutil
import subprocess
import tempfile


GIT = shutil.which("git")
LOG = pathlib.Path(tempfile.gettempdir()) / "lazygitk-branch-create-probe.json"
ENV = {
    **os.environ,
    "GIT_CONFIG_NOSYSTEM": "1",
    "GIT_CONFIG_GLOBAL": os.devnull,
    "GIT_TERMINAL_PROMPT": "0",
}


def command(repo, *args):
    proc = subprocess.run(
        [GIT, *args], cwd=repo, env=ENV, capture_output=True, text=True
    )
    return {
        "argv": ["git", *args],
        "returncode": proc.returncode,
        "stdout": proc.stdout,
        "stderr": proc.stderr,
    }


def git_path(repo, name):
    result = command(repo, "rev-parse", "--git-path", name)
    return pathlib.Path(repo, result["stdout"].strip())


def snapshot(repo, bare=False):
    head = command(repo, "rev-parse", "--verify", "HEAD")
    symbolic_head = command(repo, "symbolic-ref", "-q", "HEAD")
    refs = command(repo, "for-each-ref", "--format=%(refname) %(objectname)")
    tracking = command(repo, "config", "--get-regexp", r"^branch\..*\.(remote|merge)$")
    index_path = git_path(repo, "index")
    paths = [git_path(repo, "HEAD"), index_path, git_path(repo, "config")]
    raw_git_files = {}
    for path in paths:
        if path.is_file():
            raw_git_files[str(path)] = path.read_bytes().hex()
    worktree = {}
    if not bare:
        for path in pathlib.Path(repo).rglob("*"):
            if ".git" not in path.parts and path.is_file():
                worktree[str(path.relative_to(repo))] = path.read_bytes().hex()
    return {
        "head": head,
        "symbolic_head": symbolic_head,
        "refs": refs,
        "tracking_config": tracking,
        "index_entries": command(repo, "ls-files", "--stage"),
        "raw_git_files_hex": raw_git_files,
        "worktree_files_hex": worktree,
    }


def fresh_repo():
    repo = tempfile.mkdtemp(prefix="branch-create-probe-")
    command(repo, "init", "-q", "-b", "main")
    command(repo, "config", "user.name", "Probe")
    command(repo, "config", "user.email", "probe@example.invalid")
    pathlib.Path(repo, "base.txt").write_text("base\n")
    command(repo, "add", "base.txt")
    command(repo, "commit", "-qm", "base")
    oid = command(repo, "rev-parse", "HEAD")["stdout"].strip()
    command(repo, "remote", "add", "origin", "https://example.invalid/repo.git")
    command(repo, "update-ref", "refs/remotes/origin/topic", oid)
    command(repo, "update-ref", "refs/remotes/origin/main", oid)
    return repo, oid


RESULTS = []


def record(label, repo, actions, bare=False):
    entry = {"case": label, "git_version": command(repo, "--version")["stdout"].strip()}
    entry["before"] = snapshot(repo, bare)
    entry["commands"] = [action(repo) for action in actions]
    entry["after"] = snapshot(repo, bare)
    RESULTS.append(entry)
    shutil.rmtree(repo)


def tracking_matrix():
    modes = ["true", "always", "inherit", "simple", "false"]
    starts = ["oid", "local", "remote"]
    options = [(), ("--track",), ("--track=direct",), ("--track=inherit",), ("--no-track",)]
    for mode in modes:
        for start_kind in starts:
            for tracking_option in options:
                repo, oid = fresh_repo()
                command(repo, "config", "branch.autoSetupMerge", mode)
                start = {"oid": oid, "local": "main", "remote": "origin/topic"}[start_kind]
                args = ("branch", *tracking_option, "created", start)
                tracking_label = " ".join(tracking_option) or "default"
                record(f"matrix/{mode}/{start_kind}/{tracking_label}", repo,
                       [lambda path, args=args: command(path, *args)])

    repo, _ = fresh_repo()
    command(repo, "config", "branch.autoSetupMerge", "simple")
    record("simple/same-name-remote", repo, [lambda path: command(
        path, "branch", "topic", "origin/topic"
    )])


def target_repo(changed_tracked=False, add_ignored=False, add_untracked=False):
    repo, base_oid = fresh_repo()
    command(repo, "branch", "target", base_oid)
    command(repo, "checkout", "-q", "target")
    if changed_tracked:
        pathlib.Path(repo, "base.txt").write_text("target version\n")
    if add_ignored:
        pathlib.Path(repo, "ignored-target.txt").write_text("target ignored path\n")
    if add_untracked:
        pathlib.Path(repo, "untracked-target.txt").write_text("target untracked path\n")
    command(repo, "add", "-A")
    command(repo, "commit", "-qm", "target tree")
    target_oid = command(repo, "rev-parse", "HEAD")["stdout"].strip()
    command(repo, "checkout", "-q", "main")
    return repo, base_oid, target_oid


def checkout_obstructions():
    cases = ["dirty-tracked", "untracked-collision", "ignored-collision", "no-collision"]
    for case in cases:
        repo, _, target_oid = target_repo(
            changed_tracked=case == "dirty-tracked",
            add_ignored=case == "ignored-collision",
            add_untracked=case == "untracked-collision",
        )
        if case == "dirty-tracked":
            pathlib.Path(repo, "base.txt").write_text("local dirty\n")
        elif case == "untracked-collision":
            pathlib.Path(repo, "untracked-target.txt").write_text("local untracked\n")
        elif case == "ignored-collision":
            (pathlib.Path(repo, ".git/info/exclude")).write_text("ignored-target.txt\n")
            pathlib.Path(repo, "ignored-target.txt").write_text("local ignored\n")
        actions = [lambda path, oid=target_oid: command(
            path, "checkout", "-b", "created", oid, "--no-overwrite-ignore"
        )]
        record(f"checkout/{case}/no-overwrite-ignore", repo, actions)


def inherited_tracking():
    cases = ["seeded-upstream", "unseeded"]
    for case in cases:
        repo, oid = fresh_repo()
        if case == "seeded-upstream":
            command(repo, "config", "branch.main.remote", "origin")
            command(repo, "config", "--add", "branch.main.merge", "refs/heads/topic")
            command(repo, "config", "--add", "branch.main.merge", "refs/heads/other")
        record(f"inherit/{case}", repo, [lambda path: command(
            path, "branch", "--track=inherit", "created", "main"
        )])


def config_override():
    repo, oid = fresh_repo()
    record("oid/-c-tracking-config", repo, [lambda path: command(
        path, "-c", "branch.created.remote=origin", "-c",
        "branch.created.merge=refs/heads/topic", "branch", "created", oid
    )])


def upstream_phase_and_config_lock():
    for failure in [False, True]:
        repo, oid = fresh_repo()
        def create(path, oid=oid):
            return command(path, "branch", "created", oid)
        def set_upstream(path, failure=failure):
            if failure:
                config = git_path(path, "config")
                lock = pathlib.Path(str(config) + ".lock")
                lock.write_text("hold native config lock\n")
            return command(path, "branch", "--set-upstream-to=origin/topic", "created")
        record(f"set-upstream/config-lock={failure}", repo, [create, set_upstream])


def checkout_and_edge_cases():
    repo, oid = fresh_repo()
    record("checkout/oid", repo, [lambda path: command(path, "checkout", "-b", "created", oid)])
    repo, oid = fresh_repo()
    record("checkout/remote-ref", repo, [lambda path: command(path, "checkout", "-b", "created", "origin/topic")])
    repo, oid = fresh_repo()
    record("invalid-literal-branch", repo, [lambda path: command(path, "branch", "--", "bad..name", oid)])
    repo, oid = fresh_repo()
    command(repo, "branch", "feature=foo", oid)
    record("source-branch-equals", repo, [lambda path: command(path, "branch", "created", "feature=foo")])

    repo, oid = fresh_repo()
    command(repo, "checkout", "--orphan", "orphan")
    command(repo, "rm", "-q", "--", "base.txt")
    record("unborn-head-pointer-oid", repo, [lambda path: command(path, "branch", "created", oid)])
    repo, oid = fresh_repo()
    command(repo, "checkout", "--orphan", "orphan")
    command(repo, "rm", "-q", "--", "base.txt")
    record("unborn-head-checkout-oid", repo, [lambda path: command(path, "checkout", "-b", "created", oid)])


def bare_repository():
    parent = tempfile.mkdtemp(prefix="branch-create-bare-probe-")
    source = pathlib.Path(parent, "source")
    bare = pathlib.Path(parent, "bare.git")
    command(parent, "init", "-q", "-b", "main", str(source))
    command(source, "config", "user.name", "Probe")
    command(source, "config", "user.email", "probe@example.invalid")
    pathlib.Path(source, "base.txt").write_text("base\n")
    command(source, "add", "base.txt")
    command(source, "commit", "-qm", "base")
    oid = command(source, "rev-parse", "HEAD")["stdout"].strip()
    command(parent, "init", "--bare", "-q", str(bare))
    command(source, "push", str(bare), "main")
    repo = str(bare)
    record("bare/branch-oid", repo, [lambda path: command(path, "branch", "created", oid)], bare=True)
    shutil.rmtree(parent)


def main():
    if not GIT:
        raise SystemExit("git executable not found")
    tracking_matrix()
    inherited_tracking()
    config_override()
    upstream_phase_and_config_lock()
    checkout_obstructions()
    checkout_and_edge_cases()
    bare_repository()
    document = {
        "git_version": subprocess.run([GIT, "--version"], capture_output=True, text=True).stdout.strip(),
        "results": RESULTS,
    }
    LOG.write_text(json.dumps(document, indent=2) + "\n")
    print(json.dumps({"log": str(LOG), "git_version": document["git_version"], "cases": len(RESULTS)}))


if __name__ == "__main__":
    main()
