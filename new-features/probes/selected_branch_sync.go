//go:build ignore

// Run with: go run new-features/probes/selected_branch_sync.go
// This standalone probe writes only below its owned temporary directory.
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type probe struct{ root string }

func main() {
	root, err := os.MkdirTemp("", "selected-branch-sync-probe-")
	must(err)
	defer func() {
		if err := os.RemoveAll(root); err != nil {
			fmt.Fprintf(os.Stderr, "cleanup %q: %v\n", root, err)
		}
	}()
	p := &probe{root: root}
	out, code := p.git(root, "--version")
	check(code == 0, out)
	fmt.Printf("GIT=%s\n", out)
	p.metadata()
	p.explicitPush()
	p.fetch()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func check(ok bool, message string) {
	if !ok {
		panic("assertion failed: " + message)
	}
	fmt.Printf("ASSERT ok: %s\n", message)
}

func (p *probe) git(dir string, args ...string) (string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, "git", args...)
	c.Dir = dir
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GIT_") {
			c.Env = append(c.Env, entry)
		}
	}
	c.Env = append(c.Env,
		"LANG=C", "LC_ALL=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_TERMINAL_PROMPT=0", "GIT_EDITOR=true", "GIT_SEQUENCE_EDITOR=true",
		"GIT_AUTHOR_NAME=Probe", "GIT_AUTHOR_EMAIL=probe@example.invalid",
		"GIT_COMMITTER_NAME=Probe", "GIT_COMMITTER_EMAIL=probe@example.invalid",
	)
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	err := c.Run()
	status := 0
	if err != nil {
		status = 1
		if ee, ok := err.(*exec.ExitError); ok {
			status = ee.ExitCode()
		}
		if ctx.Err() != nil {
			status = 124
		}
	}
	result := strings.TrimSpace(stdout.String())
	if stderr.Len() > 0 {
		result += " | stderr=" + strings.TrimSpace(stderr.String())
	}
	return result, status
}

func (p *probe) mustGit(dir string, args ...string) string {
	out, code := p.git(dir, args...)
	if code != 0 {
		panic(fmt.Sprintf("git %q failed (%d): %s", args, code, out))
	}
	return out
}

func (p *probe) repo(name string) string {
	dir := filepath.Join(p.root, name)
	must(os.MkdirAll(dir, 0o700))
	p.mustGit(dir, "init", "-q", "-b", "main")
	p.mustGit(dir, "config", "core.hooksPath", filepath.Join(dir, "hooks"))
	must(os.MkdirAll(filepath.Join(dir, "hooks"), 0o700))
	return dir
}

func (p *probe) config(dir string, pairs ...string) {
	for i := 0; i < len(pairs); i += 2 {
		p.mustGit(dir, "config", "--replace-all", pairs[i], pairs[i+1])
	}
}

func (p *probe) metadata() {
	fmt.Println("\n== selected branch metadata and push routing ==")
	strategies := []string{"simple", "upstream", "current", "matching", "nothing"}
	for _, strategy := range strategies {
		dir := p.repo("meta-" + strategy)
		p.branchFixture(dir)
		p.config(dir, "push.default", strategy)
		p.describe(dir, strategy)
	}
	for _, setting := range []struct{ label, key, value string }{
		{"branch pushRemote", "branch.topic.pushRemote", "publish"},
		{"remote pushDefault", "remote.pushDefault", "publish"},
		{"branch remote override", "branch.topic.remote", "review"},
	} {
		dir := p.repo("route-" + strings.ReplaceAll(setting.label, " ", "-"))
		p.branchFixture(dir)
		p.config(dir, "push.default", "upstream", setting.key, setting.value)
		p.describe(dir, setting.label)
	}
	for _, spec := range []string{"refs/heads/reviewed:refs/heads/reviewed", "refs/heads/topic:refs/heads/release", "refs/heads/*:refs/heads/archive/*"} {
		dir := p.repo("remote-push-" + fmt.Sprint(len(spec)))
		p.branchFixture(dir)
		p.config(dir, "push.default", "upstream", "branch.topic.pushRemote", "review", "remote.review.push", spec)
		p.describe(dir, "remote.review.push="+spec)
	}
	multiple := p.repo("remote-push-multiple")
	p.branchFixture(multiple)
	p.config(multiple, "push.default", "upstream", "branch.topic.pushRemote", "review",
		"remote.review.push", "refs/heads/topic:refs/heads/release-one")
	p.mustGit(multiple, "config", "--add", "remote.review.push", "refs/heads/topic:refs/heads/release-two")
	p.describe(multiple, "remote.review.push: topic to release-one and release-two")
	dir := p.repo("gone")
	p.branchFixture(dir)
	p.mustGit(dir, "update-ref", "-d", "refs/remotes/origin/reviewed")
	track := p.mustGit(dir, "for-each-ref", "--format=%(upstream:track)", "refs/heads/topic")
	fmt.Printf("gone upstream:track=%q\n", track)
	check(track == "[gone]", "deleted configured upstream tracking ref reports [gone]")
	p.describe(dir, "gone upstream ref")
	dir = p.repo("local-upstream")
	p.branchFixture(dir)
	p.config(dir, "branch.topic.remote", ".", "branch.topic.merge", "refs/heads/reviewed")
	p.describe(dir, "upstream remote dot, distinct merge name")
	dir = p.repo("custom-tracking-namespace")
	p.branchFixture(dir)
	p.config(dir, "remote.origin.fetch", "+refs/heads/*:refs/cache/origin/*")
	p.mustGit(dir, "fetch", "-q", "origin")
	p.describe(dir, "custom fetch namespace")
}

func (p *probe) branchFixture(dir string) {
	p.mustGit(dir, "config", "user.name", "Probe")
	p.mustGit(dir, "config", "user.email", "probe@example.invalid")
	p.mustGit(dir, "config", "commit.gpgSign", "false")
	p.mustGit(dir, "config", "tag.gpgSign", "false")
	p.mustGit(dir, "config", "init.defaultBranch", "main")
	p.mustGit(dir, "remote", "add", "origin", ".")
	p.mustGit(dir, "remote", "add", "review", ".")
	p.mustGit(dir, "remote", "add", "publish", ".")
	p.mustGit(dir, "config", "branch.topic.remote", "origin")
	p.mustGit(dir, "config", "branch.topic.merge", "refs/heads/reviewed")
	p.mustGit(dir, "symbolic-ref", "HEAD", "refs/heads/main")
	p.mustGit(dir, "commit", "--allow-empty", "-qm", "main")
	p.mustGit(dir, "branch", "topic")
	p.mustGit(dir, "branch", "reviewed")
	p.mustGit(dir, "update-ref", "refs/heads/reviewed", "refs/heads/main")
	p.mustGit(dir, "update-ref", "refs/remotes/origin/topic", "refs/heads/topic")
	p.mustGit(dir, "update-ref", "refs/remotes/origin/reviewed", "refs/heads/reviewed")
}

func (p *probe) describe(dir, label string) {
	fmt.Printf("\nCASE %s\n", label)
	format := "%(refname)%00%(upstream:remotename)%00%(upstream:remoteref)%00%(push:remotename)%00%(push:remoteref)"
	out, code := p.git(dir, "for-each-ref", "--format="+format, "refs/heads/topic")
	fmt.Printf("for-each-ref status=%d raw=%q\n", code, out)
	out, code = p.git(dir, "rev-parse", "--symbolic-full-name", "topic@{push}")
	fmt.Printf("selected@{push} status=%d value=%q\n", code, out)
	// Only this disposable metadata fixture checks out topic to observe native
	// no-argument push defaults. This is not the selected-branch app workflow.
	p.mustGit(dir, "checkout", "-q", "topic")
	for _, args := range [][]string{{"push", "--dry-run", "--porcelain"}, {"push", "--dry-run", "--porcelain", "origin"}} {
		out, code = p.git(dir, args...)
		fmt.Printf("%q status=%d output=%q\n", args, code, out)
	}
	p.mustGit(dir, "symbolic-ref", "HEAD", "refs/heads/main")
}

func (p *probe) explicitPush() {
	fmt.Println("\n== explicit selected-ref push, porcelain, and lease ==")
	local := p.repo("push-local")
	bare := filepath.Join(p.root, "push-remote.git")
	p.mustGit(p.root, "init", "--bare", "-q", bare)
	p.mustGit(local, "config", "user.name", "Probe")
	p.mustGit(local, "config", "user.email", "probe@example.invalid")
	p.mustGit(local, "config", "commit.gpgSign", "false")
	p.mustGit(local, "remote", "add", "origin", bare)
	p.mustGit(local, "commit", "--allow-empty", "-qm", "base")
	base := p.mustGit(local, "rev-parse", "HEAD")
	p.mustGit(local, "branch", "topic")
	p.mustGit(local, "branch", "other")
	p.mustGit(local, "checkout", "-q", "topic")
	osWrite(local, "topic.txt", "topic\n")
	p.mustGit(local, "add", "topic.txt")
	p.mustGit(local, "commit", "-qm", "topic change")
	topic := p.mustGit(local, "rev-parse", "HEAD")
	p.mustGit(local, "checkout", "-q", "main")
	osWrite(local, "main.txt", "unchanged\n")
	p.mustGit(local, "add", "main.txt")
	p.mustGit(local, "commit", "-qm", "main change")
	mainTip := p.mustGit(local, "rev-parse", "HEAD")
	p.mustGit(local, "push", "-q", "origin", "refs/heads/main:refs/heads/main", "refs/heads/other:refs/heads/other")
	p.mustGit(local, "update-ref", "refs/heads/topic", topic)
	p.mustGit(local, "update-ref", "refs/heads/other", base)
	p.mustGit(local, "update-ref", "refs/heads/main", mainTip)
	check(p.mustGit(local, "symbolic-ref", "HEAD") == "refs/heads/main", "selected topic differs from checked-out main")
	osWrite(local, "main.txt", "staged blob\n")
	p.mustGit(local, "add", "main.txt")
	osWrite(local, "main.txt", "unstaged tracked edit\n")
	osWrite(local, "untracked.txt", "untracked bytes\n")
	beforeHead := p.mustGit(local, "rev-parse", "HEAD")
	beforeTree := p.mustGit(local, "write-tree")
	beforeStatus := p.mustGit(local, "--no-optional-locks", "status", "--porcelain=v1", "--untracked-files=all")
	check(beforeStatus == "MM main.txt\n?? untracked.txt", "fixture contains staged blob, unstaged tracked edit, and untracked file")
	beforeRefs := p.mustGit(local, "for-each-ref", "--format=%(refname)=%(objectname)", "refs/heads")
	beforeFiles := snapshotFiles(local)
	out, code := p.git(local, "push", "--porcelain", "origin", "refs/heads/topic:refs/heads/topic")
	fmt.Printf("push status=%d output=%q\n", code, out)
	check(code == 0 && strings.Contains(out, "refs/heads/topic"), "explicit push porcelain names selected destination")
	check(p.mustGit(p.root, "--git-dir", bare, "rev-parse", "refs/heads/topic") == topic, "only selected destination receives selected tip")
	check(p.mustGit(p.root, "--git-dir", bare, "rev-parse", "refs/heads/main") == mainTip, "remote main remains unchanged")
	check(p.mustGit(p.root, "--git-dir", bare, "rev-parse", "refs/heads/other") == base, "remote other remains unchanged")
	check(beforeHead == p.mustGit(local, "rev-parse", "HEAD"), "HEAD unchanged")
	check(reflect.DeepEqual(beforeFiles, snapshotFiles(local)), "HEAD file, raw index bytes, and all worktree file bytes unchanged")
	check(beforeTree == p.mustGit(local, "write-tree") && beforeStatus == p.mustGit(local, "--no-optional-locks", "status", "--porcelain=v1", "--untracked-files=all"), "staged tree and dirty worktree status unchanged")
	check(beforeRefs == p.mustGit(local, "for-each-ref", "--format=%(refname)=%(objectname)", "refs/heads"), "all local branch refs unchanged")
	leaseExpected := p.mustGit(p.root, "--git-dir", bare, "rev-parse", "refs/heads/topic")
	p.mustGit(p.root, "--git-dir", bare, "update-ref", "refs/heads/topic", mainTip)
	out, code = p.git(local, "push", "--porcelain", "--force-with-lease=refs/heads/topic:"+leaseExpected, "origin", "refs/heads/topic:refs/heads/topic")
	fmt.Printf("concurrent-change leased push status=%d output=%q\n", code, out)
	check(code == 1 && strings.Contains(out, "!\trefs/heads/topic:refs/heads/topic\t[rejected] (stale info)"), "explicit expected OID lease reports porcelain stale info")
	check(p.mustGit(p.root, "--git-dir", bare, "rev-parse", "refs/heads/topic") == mainTip, "lease rejection preserves concurrent remote tip")
	// An explicit source refspec must not silently turn into a mirror push.
	p.config(local, "remote.origin.mirror", "true")
	out, code = p.git(local, "push", "--dry-run", "--porcelain", "origin", "refs/heads/topic:refs/heads/topic")
	fmt.Printf("configured mirror plus explicit refspec status=%d output=%q\n", code, out)
	check(code != 0, "configured mirror is rejected with explicit refspec rather than widening its scope")
	out, code = p.git(local, "push", "--dry-run", "--no-mirror", "--porcelain", "origin", "refs/heads/topic:refs/heads/topic")
	fmt.Printf("explicit no-mirror plus explicit refspec status=%d output=%q\n", code, out)
	check(code != 0 && strings.Contains(out, "--mirror can't be combined with refspecs"), "no-mirror does not override configured mirror mode on the tested Git")
}

func snapshotFiles(dir string) map[string][]byte {
	files := make(map[string][]byte)
	must(filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == filepath.Join(dir, ".git") {
			return filepath.SkipDir
		}
		if !entry.IsDir() {
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files[rel], err = os.ReadFile(path)
			return err
		}
		return nil
	}))
	for _, name := range []string{".git/HEAD", ".git/index"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		must(err)
		files[name] = data
	}
	return files
}

func osWrite(dir, name, value string) {
	must(os.WriteFile(filepath.Join(dir, name), []byte(value), 0o600))
}

func (p *probe) fetch() {
	fmt.Println("\n== exact-remote branch fetch ==")
	remote := p.repo("fetch-remote")
	p.mustGit(remote, "config", "user.name", "Probe")
	p.mustGit(remote, "config", "user.email", "probe@example.invalid")
	p.mustGit(remote, "commit", "--allow-empty", "-qm", "main")
	p.mustGit(remote, "branch", "topic")
	p.mustGit(remote, "tag", "v1")
	local := p.repo("fetch-local")
	p.mustGit(local, "remote", "add", "origin", remote)
	p.mustGit(local, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	p.mustGit(local, "fetch", "-q", "origin")
	p.mustGit(local, "tag", "-d", "v1")
	p.mustGit(remote, "checkout", "-q", "topic")
	p.mustGit(remote, "commit", "--allow-empty", "-qm", "topic advanced")
	oldMain := p.mustGit(local, "rev-parse", "refs/remotes/origin/main")
	oldTopic := p.mustGit(local, "rev-parse", "refs/remotes/origin/topic")
	newTopic := p.mustGit(remote, "rev-parse", "refs/heads/topic")
	p.mustGit(local, "update-ref", "refs/remotes/origin/unrelated", oldMain)
	p.mustGit(local, "config", "--add", "remote.origin.fetch", "+refs/heads/topic:refs/cache/extra/topic")
	p.mustGit(local, "update-ref", "refs/cache/extra/topic", oldTopic)
	p.mustGit(local, "config", "remote.origin.prune", "true")
	p.mustGit(local, "config", "remote.origin.tagOpt", "--tags")
	p.mustGit(local, "config", "submodule.recurse", "true")
	beforeRefs := p.mustGit(local, "for-each-ref", "--format=%(refname)=%(objectname)")
	beforeFetchHead, err := os.ReadFile(filepath.Join(local, ".git", "FETCH_HEAD"))
	must(err)
	check(len(beforeFetchHead) > 0 && oldTopic != newTopic, "fetch fixture has existing FETCH_HEAD and an advanced remote topic")
	command := []string{"fetch", "--refmap=", "--no-tags", "--no-recurse-submodules", "--no-write-fetch-head", "--no-prune", "origin", "+refs/heads/topic:refs/remotes/origin/topic"}
	out, code := p.git(local, command...)
	fmt.Printf("%q status=%d output=%q\n", command, code, out)
	check(code == 0, "exact branch fetch succeeds")
	check(p.mustGit(local, "rev-parse", "refs/remotes/origin/main") == oldMain, "fetch leaves other tracking branch unchanged")
	check(p.mustGit(local, "rev-parse", "refs/remotes/origin/unrelated") == oldMain, "fetch leaves unrelated local tracking ref unchanged")
	check(p.mustGit(local, "rev-parse", "refs/remotes/origin/topic") == newTopic, "fetch updates requested tracking ref to exact remote tip")
	check(p.mustGit(local, "rev-parse", "refs/cache/extra/topic") == oldTopic, "empty refmap suppresses extra configured mapping of fetched topic")
	afterRefs := p.mustGit(local, "for-each-ref", "--format=%(refname)=%(objectname)")
	expectedRefs := strings.Replace(beforeRefs, "refs/remotes/origin/topic="+oldTopic, "refs/remotes/origin/topic="+newTopic, 1)
	check(afterRefs == expectedRefs, "fetch changes only explicit desired tracking ref across all refs, despite configured pruning")
	afterFetchHead, err := os.ReadFile(filepath.Join(local, ".git", "FETCH_HEAD"))
	must(err)
	check(bytes.Equal(beforeFetchHead, afterFetchHead), "fetch preserves existing FETCH_HEAD bytes")
	_, code = p.git(local, "show-ref", "--verify", "refs/tags/v1")
	check(code != 0, "--no-tags prevents configured tag following")
	fmt.Println("HAZARDS: remote.<name>.tagOpt is overridden by --no-tags; push.followTags is overridden by --no-follow-tags; remote.<name>.mirror can cause mirror semantics and must be inspected/overridden or rejected for explicit selected-ref operations.")
}
