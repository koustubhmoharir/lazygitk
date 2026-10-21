//go:build ignore

// Run with: go run new-features/probes/named_checkout.go
// This standalone probe writes only below its owned os.MkdirTemp directory.
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type repo struct{ path string }

func main() {
	root, err := os.MkdirTemp("", "named-checkout-probe-")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := os.RemoveAll(root); err != nil {
			fmt.Fprintf(os.Stderr, "cleanup %q: %v\n", root, err)
		}
	}()
	fmt.Printf("TEMP_ROOT=%q\n", root)
	p := newProbe(root)
	p.run()
}

type probe struct {
	root string
	n    int
}

func newProbe(root string) *probe { return &probe{root: root} }
func (p *probe) command(dir string, args ...string) (string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, "git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_AUTHOR_NAME=Probe", "GIT_AUTHOR_EMAIL=probe@example.invalid", "GIT_COMMITTER_NAME=Probe", "GIT_COMMITTER_EMAIL=probe@example.invalid")
	var out, er bytes.Buffer
	c.Stdout, c.Stderr = &out, &er
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
	return strings.TrimSpace(out.String() + func() string {
		if er.Len() > 0 {
			return " | stderr=" + strings.TrimSpace(er.String())
		}
		return ""
	}()), status
}

func (p *probe) mk(label string) *repo {
	p.n++
	r := &repo{filepath.Join(p.root, fmt.Sprintf("%02d-%s", p.n, label))}
	must(os.MkdirAll(r.path, 0o700))
	p.mustGit(r, "init", "-q", "-b", "main")
	p.mustGit(r, "config", "core.hooksPath", filepath.Join(r.path, "hooks"))
	must(os.MkdirAll(filepath.Join(r.path, "hooks"), 0o700))
	return r
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func (p *probe) write(r *repo, name, data string) {
	path := filepath.Join(r.path, filepath.FromSlash(name))
	must(os.MkdirAll(filepath.Dir(path), 0o700))
	must(os.WriteFile(path, []byte(data), 0o600))
}
func (p *probe) git(r *repo, args ...string) (string, int) { return p.command(r.path, args...) }
func (p *probe) mustGit(r *repo, args ...string) string {
	out, code := p.git(r, args...)
	if code != 0 {
		panic(fmt.Sprintf("setup git %q in %s failed (%d): %s", args, r.path, code, out))
	}
	return out
}
func (p *probe) tryGit(r *repo, args ...string) (string, int) { return p.git(r, args...) }
func (p *probe) commit(r *repo, msg string) string {
	p.mustGit(r, "add", "-A")
	p.mustGit(r, "commit", "-qm", msg)
	o := p.mustGit(r, "rev-parse", "HEAD")
	return o
}

func (p *probe) state(r *repo) string {
	h, _ := p.git(r, "rev-parse", "--verify", "HEAD")
	ref, _ := p.git(r, "symbolic-ref", "-q", "HEAD")
	refs, _ := p.git(r, "for-each-ref", "--format=%(refname)=%(objectname)", "refs/heads")
	idx, _ := p.git(r, "write-tree")
	st, _ := p.git(r, "status", "--porcelain=v1", "--untracked-files=all")
	files := make([]string, 0, 5)
	for _, name := range []string{"common", "base-only", "main-only", "topic-only", "staged", "loose"} {
		b, e := os.ReadFile(filepath.Join(r.path, name))
		if e == nil {
			files = append(files, fmt.Sprintf("%s=%q", name, string(b)))
		} else {
			files = append(files, name+"=<absent>")
		}
	}
	gitDir := p.mustGit(r, "rev-parse", "--absolute-git-dir")
	for _, name := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "BISECT_START", "rebase-merge", "rebase-apply", "late-hook-fired"} {
		if info, err := os.Stat(filepath.Join(gitDir, name)); err == nil {
			files = append(files, fmt.Sprintf("git-state:%s=%s", name, info.Mode()))
		}
	}
	return fmt.Sprintf("HEAD=%s symbolic=%q refs={%s} index=%s files={%s} status=%q", h, ref, refs, idx, strings.Join(files, ","), st)
}

func (p *probe) checkout(r *repo, name, oid string) {
	before := p.state(r)
	out, code := p.tryGit(r, "checkout", "--no-overwrite-ignore", "-B", name, oid)
	fmt.Printf("CASE %s\n  command_status=%d output=%q\n  before: %s\n  after:  %s\n", filepath.Base(r.path), code, out, before, p.state(r))
}

func (p *probe) run() {
	v, _ := p.command(p.root, "--version")
	fmt.Printf("GIT=%s\n", v)
	// Common graph: base, a divergent target with distinct files, and current main.
	base := p.mk("base")
	p.write(base, "common", "base\n")
	p.write(base, "base-only", "base\n")
	baseOID := p.commit(base, "base")
	p.mustGit(base, "checkout", "-qb", "topic")
	p.write(base, "topic-only", "topic\n")
	topic := p.commit(base, "topic")
	p.write(base, "common", "topic version\n")
	topic = p.commit(base, "topic changes common")
	topicOldTip := topic
	p.mustGit(base, "checkout", "-q", "main")
	p.write(base, "main-only", "main\n")
	main := p.commit(base, "main")
	fmt.Printf("GRAPH base/topic-old/topic-current-main=%s/%s/%s/%s\n", baseOID, topicOldTip, topic, main)
	// Each case gets an isolated clone of the same commits.
	fixture := func(label string) *repo {
		r := p.mk(label)
		p.mustGit(r, "fetch", "-q", base.path, "main", "topic")
		p.mustGit(r, "checkout", "-q", "-B", "main", main)
		p.mustGit(r, "branch", "topic", topic)
		return r
	}
	r := fixture("new-different-tree")
	p.checkout(r, "created", topic)
	r = fixture("existing-different-tree")
	p.mustGit(r, "config", "branch.topic.remote", "origin")
	p.mustGit(r, "config", "branch.topic.merge", "refs/heads/remote-topic")
	p.checkout(r, "topic", baseOID)
	r = fixture("current-destination")
	p.mustGit(r, "checkout", "-q", "topic")
	p.checkout(r, "topic", baseOID)
	r = fixture("dirty-unrelated")
	p.write(r, "staged", "staged\n")
	p.mustGit(r, "add", "staged")
	p.write(r, "base-only", "dirty\n")
	p.write(r, "loose", "loose\n")
	p.checkout(r, "created", topic)
	for _, kind := range []string{"unstaged", "staged"} {
		r = fixture("overlap-" + kind)
		p.write(r, "common", "dirty\n")
		if kind == "staged" {
			p.mustGit(r, "add", "common")
		}
		p.checkout(r, "created", topic)
	}
	for _, kind := range []string{"untracked", "ignored"} {
		r = fixture("collision-" + kind)
		p.write(r, "topic-only", "collision\n")
		if kind == "ignored" {
			must(os.WriteFile(filepath.Join(r.path, ".git", "info", "exclude"), []byte("topic-only\n"), 0o600))
		}
		p.checkout(r, "created", topic)
	}
	r = fixture("same-bytes-collision")
	p.write(r, "topic-only", "topic\n")
	p.checkout(r, "created", topic)
	for _, suffix := range []string{"", "^{commit}"} {
		label := "oid-name-shadow"
		if suffix != "" {
			label += "-peel"
		}
		r = fixture(label)
		p.mustGit(r, "-c", "advice.objectNameWarning=false", "branch", topic, baseOID)
		catType, catStatus := p.git(r, "cat-file", "-t", topic)
		fmt.Printf("OID_SHADOW suffix=%q cat-file-status=%d cat-file=%q selected=%s shadow-ref=%s\n", suffix, catStatus, catType, topic, p.mustGit(r, "rev-parse", "refs/heads/"+topic))
		p.checkout(r, "created", topic+suffix)
		actual := p.mustGit(r, "rev-parse", "refs/heads/created")
		fmt.Printf("OID_SHADOW_RESULT suffix=%q requested_commit=%s created_ref=%s\n", suffix, topic, actual)
	}
	r = fixture("existing-config")
	p.mustGit(r, "config", "branch.topic.remote", "origin")
	p.mustGit(r, "config", "branch.topic.merge", "refs/heads/upstream")
	p.checkout(r, "topic", baseOID)
	x := p.mustGit(r, "config", "--get-regexp", "^branch.topic")
	fmt.Printf("  config-after=%q\n", x)
	r = fixture("auto-setup-merge-always")
	p.mustGit(r, "config", "branch.autoSetupMerge", "always")
	p.checkout(r, "created", topic)
	x, _ = p.tryGit(r, "config", "--get-regexp", "^branch.created")
	fmt.Printf("  tracking-after=%q\n", x)
	r = fixture("worktree-busy")
	p.mustGit(r, "worktree", "add", "-q", filepath.Join(r.path, "linked"), "topic")
	p.checkout(r, "topic", baseOID)
	r = fixture("symbolic-destination")
	p.mustGit(r, "symbolic-ref", "refs/heads/alias", "refs/heads/topic")
	p.checkout(r, "alias", baseOID)
	x = p.mustGit(r, "symbolic-ref", "-q", "refs/heads/alias")
	fmt.Printf("  alias-after=%q\n", x)
	// Hook cases run in distinct fixtures. Hooks are executable files created only in temp repos.
	r = fixture("post-checkout-hook-failure")
	p.hook(r, "post-checkout", "#!/bin/sh\nexit 7\n")
	p.checkout(r, "created", topic)
	r = fixture("reference-transaction-reject")
	p.hook(r, "reference-transaction", `#!/bin/sh
[ "$1" = prepared ] && exit 1
exit 0
`)
	p.checkout(r, "created", topic)
	// Exercise actual active operation states with Git's own commands, bounded by command timeout.
	for _, op := range []string{"merge", "rebase", "cherry-pick", "bisect"} {
		r := fixture("active-" + op)
		p.active(r, op)
		p.checkout(r, "created", topic)
	}
	// Late movement after index/worktree transition: post-index-change hook changes topic before branch creation.
	r = fixture("late-ref-move")
	markerPath := filepath.Join(r.path, ".git", "late-hook-fired")
	p.hook(r, "post-index-change", fmt.Sprintf("#!/bin/sh\nset -e\ngit update-ref refs/heads/topic %s\nprintf fired > %s\n", main, markerPath))
	p.checkout(r, "topic", baseOID)
	marker, markerErr := os.ReadFile(markerPath)
	newTopic, _ := p.git(r, "rev-parse", "refs/heads/topic")
	fmt.Printf("LATE_HOOK marker=%q err=%v old_destination=%s final_destination=%s\n", marker, markerErr, topic, newTopic)
	r = fixture("late-alias-change")
	p.mustGit(r, "branch", "alias", topic)
	markerPath = filepath.Join(r.path, ".git", "late-hook-fired")
	p.hook(r, "post-index-change", fmt.Sprintf("#!/bin/sh\nset -e\ngit update-ref --no-deref -d refs/heads/alias\ngit symbolic-ref refs/heads/alias refs/heads/main\nprintf fired > %s\n", markerPath))
	p.checkout(r, "alias", baseOID)
	aliasTarget, aliasCode := p.git(r, "symbolic-ref", "-q", "refs/heads/alias")
	marker, markerErr = os.ReadFile(markerPath)
	fmt.Printf("LATE_ALIAS symbolic_status=%d target=%q marker=%q err=%v\n", aliasCode, aliasTarget, marker, markerErr)
}

func (p *probe) hook(r *repo, name, body string) {
	path := filepath.Join(r.path, "hooks", name)
	must(os.WriteFile(path, []byte(body), 0o700))
	must(os.Chmod(path, 0o700))
}

func (p *probe) active(r *repo, op string) {
	if op == "merge" {
		_, c := p.tryGit(r, "merge", "--no-commit", "--no-ff", "topic")
		_, headCode := p.git(r, "rev-parse", "--verify", "MERGE_HEAD")
		if c != 0 || headCode != 0 {
			panic(fmt.Sprintf("merge fixture inactive: merge=%d MERGE_HEAD=%d", c, headCode))
		}
		fmt.Printf("ACTIVE_SETUP merge MERGE_HEAD present\n")
		return
	}
	if op == "rebase" {
		p.write(r, "common", "main version\n")
		p.commit(r, "main changes common")
		_, c := p.tryGit(r, "rebase", "topic")
		mergePath, _ := p.git(r, "rev-parse", "--git-path", "rebase-merge")
		applyPath, _ := p.git(r, "rev-parse", "--git-path", "rebase-apply")
		_, errMerge := os.Stat(filepath.Join(r.path, mergePath))
		_, errApply := os.Stat(filepath.Join(r.path, applyPath))
		if c == 0 || (errMerge != nil && errApply != nil) {
			panic(fmt.Sprintf("rebase fixture inactive: status=%d merge=%v apply=%v", c, errMerge, errApply))
		}
		fmt.Printf("ACTIVE_SETUP rebase conflict rebase-merge=%t rebase-apply=%t\n", errMerge == nil, errApply == nil)
		return
	}
	if op == "cherry-pick" {
		p.mustGit(r, "checkout", "-qb", "side")
		p.write(r, "pick", "p\n")
		oid := p.commit(r, "pick")
		p.mustGit(r, "checkout", "-q", "main")
		p.write(r, "pick", "conflict\n")
		p.commit(r, "conflicting main")
		_, c := p.tryGit(r, "cherry-pick", oid)
		_, headCode := p.git(r, "rev-parse", "--verify", "CHERRY_PICK_HEAD")
		if c == 0 || headCode != 0 {
			panic(fmt.Sprintf("cherry-pick fixture inactive: status=%d CHERRY_PICK_HEAD=%d", c, headCode))
		}
		fmt.Printf("ACTIVE_SETUP cherry-pick conflict CHERRY_PICK_HEAD present\n")
		return
	}
	for i := 0; i < 3; i++ {
		p.mustGit(r, "commit", "--allow-empty", "-qm", fmt.Sprintf("bisect step %d", i))
	}
	p.mustGit(r, "bisect", "start", "HEAD", "HEAD~3")
	if _, err := os.Stat(filepath.Join(r.path, ".git", "BISECT_START")); err != nil {
		panic("bisect fixture inactive: BISECT_START missing")
	}
	fmt.Printf("ACTIVE_SETUP bisect BISECT_START present\n")
}
