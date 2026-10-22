//go:build ignore

// Run with: go run new-features/probes/rebase_selected_commit.go
// This standalone probe writes only below its owned os.MkdirTemp directory.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type repo struct{ path string }

func main() {
	root, err := os.MkdirTemp("", "rebase-selected-probe-")
	must(err)
	defer func() {
		if err := os.RemoveAll(root); err != nil {
			fmt.Fprintf(os.Stderr, "cleanup %q: %v\n", root, err)
		}
	}()
	fmt.Printf("TEMP_ROOT=%q\n", root)
	p := &probe{root: root}
	v, code := p.command(root, "--version")
	if code != 0 {
		panic(v)
	}
	fmt.Printf("GIT=%s\n", v)
	p.run()
}

type probe struct {
	root string
	n    int
}

func (p *probe) command(dir string, args ...string) (string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, "git", args...)
	c.Dir = dir
	// A caller may have launched this from a Git subprocess. Inherited Git
	// location/index/object variables must never redirect our disposable work
	// into that caller's repository, even when cmd.Dir points at a temp repo.
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GIT_") {
			c.Env = append(c.Env, entry)
		}
	}
	c.Env = append(c.Env, "LANG=C", "LC_ALL=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_EDITOR=true", "GIT_SEQUENCE_EDITOR=true", "GIT_AUTHOR_NAME=Probe", "GIT_AUTHOR_EMAIL=probe@example.invalid", "GIT_COMMITTER_NAME=Probe", "GIT_COMMITTER_EMAIL=probe@example.invalid")
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
	result := strings.TrimSpace(out.String())
	if er.Len() > 0 {
		result += " | stderr=" + strings.TrimSpace(er.String())
	}
	return result, status
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

func (p *probe) mk(label string) *repo {
	p.n++
	r := &repo{filepath.Join(p.root, fmt.Sprintf("%02d-%s", p.n, label))}
	must(os.MkdirAll(r.path, 0o700))
	p.mustGit(r, "init", "-q", "-b", "main")
	p.mustGit(r, "config", "core.hooksPath", filepath.Join(r.path, "hooks"))
	must(os.MkdirAll(filepath.Join(r.path, "hooks"), 0o700))
	return r
}
func (p *probe) git(r *repo, args ...string) (string, int) { return p.command(r.path, args...) }
func (p *probe) mustGit(r *repo, args ...string) string {
	out, code := p.git(r, args...)
	if code != 0 {
		panic(fmt.Sprintf("setup git %q failed (%d): %s", args, code, out))
	}
	return out
}
func (p *probe) tryGit(r *repo, args ...string) (string, int) { return p.git(r, args...) }
func (p *probe) write(r *repo, name, data string) {
	path := filepath.Join(r.path, filepath.FromSlash(name))
	must(os.MkdirAll(filepath.Dir(path), 0o700))
	must(os.WriteFile(path, []byte(data), 0o600))
}

func (p *probe) commit(r *repo, message string) string {
	p.mustGit(r, "add", "-A")
	p.mustGit(r, "commit", "-qm", message)
	return p.mustGit(r, "rev-parse", "HEAD")
}

func (p *probe) fixture(label string, source *repo, refs map[string]string, head string) *repo {
	r := p.mk(label)
	p.mustGit(r, "fetch", "-q", source.path, "--", "refs/heads/main")
	// Fetch each required commit by object ID into an isolated local repository.
	for name, oid := range refs {
		p.mustGit(r, "fetch", "-q", source.path, oid)
		p.mustGit(r, "branch", name, oid)
	}
	p.mustGit(r, "checkout", "-q", "-B", "main", head)
	return r
}

func (p *probe) state(r *repo) string {
	h, _ := p.git(r, "rev-parse", "HEAD")
	refs, _ := p.git(r, "for-each-ref", "--format=%(refname)=%(objectname)", "refs/heads")
	status, _ := p.git(r, "status", "--porcelain=v1", "--untracked-files=all")
	tree, _ := p.git(r, "write-tree")
	indexBytes, _ := os.ReadFile(filepath.Join(r.path, ".git", "index"))
	worktree := sha256.New()
	must(filepath.Walk(r.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(r.path, path)
		if err != nil {
			return err
		}
		if rel == ".git" {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(worktree, "%s\\x00%x\\x00", filepath.ToSlash(rel), data)
		return err
	}))
	return fmt.Sprintf("HEAD=%s refs={%s} status=%q index_tree=%q index_sha256=%x worktree_sha256=%x", h, refs, status, strings.TrimSpace(tree), sha256.Sum256(indexBytes), worktree.Sum(nil))
}

func (p *probe) selectedRebase(r *repo, target string) (string, int) {
	return p.tryGit(r, "-c", "rebase.updateRefs=false", "rebase", "--no-autostash", target+"^{commit}")
}

func (p *probe) run() {
	seed := p.mk("seed")
	p.write(seed, "common", "base\n")
	p.write(seed, "base-only", "base\n")
	root := p.commit(seed, "root")
	p.write(seed, "main-only", "main\n")
	ancestor := p.commit(seed, "ancestor")
	p.mustGit(seed, "checkout", "-qb", "topic", root)
	p.write(seed, "topic-only", "topic\n")
	divergent := p.commit(seed, "divergent target")
	p.mustGit(seed, "checkout", "-q", "main")
	p.write(seed, "main-only", "descendant\n")
	descendant := p.commit(seed, "descendant")
	refs := map[string]string{"target": divergent}
	fmt.Printf("GRAPH root=%s ancestor=%s divergent=%s descendant=%s\n", root, ancestor, divergent, descendant)
	for _, tc := range []struct{ name, head, target, want string }{{"divergent", descendant, divergent, "rebased"}, {"ancestor", descendant, ancestor, "unchanged"}, {"descendant", ancestor, descendant, "target"}, {"root", descendant, root, "unchanged"}} {
		r := p.fixture("target-"+tc.name, seed, refs, tc.head)
		before := p.state(r)
		out, code := p.selectedRebase(r, tc.target)
		after := p.state(r)
		got := p.mustGit(r, "rev-parse", "HEAD")
		check(code == 0, "target "+tc.name+" command succeeds")
		switch tc.want {
		case "unchanged":
			check(got == tc.head, "target "+tc.name+" leaves HEAD unchanged")
		case "target":
			check(got == tc.target, "descendant target advances HEAD to target")
		case "rebased":
			isAncestor, ancestorCode := p.git(r, "merge-base", "--is-ancestor", tc.target, "HEAD")
			check(ancestorCode == 0, "divergent target is ancestor of rebased HEAD: "+isAncestor)
		}
		fmt.Printf("TARGET %s status=%d output=%q\n  before: %s\n  after:  %s\n", tc.name, code, out, before, after)
	}
	// Configured autostash with tracked staged and unstaged changes. A rejected
	// start must leave both worktree/index and stash list untouched.
	r := p.fixture("autostash-dirty", seed, refs, descendant)
	p.mustGit(r, "config", "rebase.autoStash", "true")
	p.write(r, "staged-dirt", "staged\n")
	p.mustGit(r, "add", "staged-dirt")
	p.write(r, "main-only", "unstaged dirt\n")
	stashBefore := p.mustGit(r, "stash", "list")
	before := p.state(r)
	out, code := p.selectedRebase(r, divergent)
	stashAfter := p.mustGit(r, "stash", "list")
	check(code != 0, "configured autostash does not accept staged and unstaged tracked dirt")
	check(before == p.state(r), "rejected dirty start preserves HEAD, refs, index tree/index bytes, and worktree file bytes")
	check(stashBefore == stashAfter, "rejected dirty start leaves stash list unchanged")
	fmt.Printf("AUTOSTASH status=%d output=%q stash_before=%q stash_after=%q\n  before: %s\n  after:  %s\n", code, out, stashBefore, stashAfter, before, p.state(r))
	// Untracked input: record acceptance when harmless and behavior on collision.
	for _, collision := range []bool{false, true} {
		label := "untracked-harmless"
		if collision {
			label = "untracked-collision"
		}
		r = p.fixture(label, seed, refs, descendant)
		if collision {
			p.write(r, "topic-only", "untracked collision\n")
		} else {
			p.write(r, "loose", "untracked\n")
		}
		before = p.state(r)
		out, code = p.selectedRebase(r, divergent)
		if collision {
			check(code != 0 && before == p.state(r), "untracked collision is rejected without changing repository state")
		} else {
			check(code == 0, "harmless untracked file is accepted by Git rebase")
			check(strings.Contains(p.state(r), "?? loose"), "harmless untracked file survives rebase")
		}
		fmt.Printf("UNTRACKED collision=%t status=%d output=%q\n  before: %s\n  after:  %s\n", collision, code, out, before, p.state(r))
	}
	// Unlike an untracked file, an ignored collision may be overwritten by
	// checkout machinery. Capture the exact bytes plus HEAD/index state.
	r = p.fixture("ignored-collision", seed, refs, descendant)
	ignoredPath := filepath.Join(r.path, "topic-only")
	must(os.WriteFile(ignoredPath, []byte("ignored user bytes\n"), 0o600))
	must(os.WriteFile(filepath.Join(r.path, ".git", "info", "exclude"), []byte("topic-only\n"), 0o600))
	ignoredBefore, err := os.ReadFile(ignoredPath)
	must(err)
	before = p.state(r)
	out, code = p.selectedRebase(r, divergent)
	ignoredAfter, err := os.ReadFile(ignoredPath)
	if err != nil {
		ignoredAfter = []byte("<absent>")
	}
	check(code == 0 && bytes.Equal(ignoredAfter, []byte("topic\n")), "successful rebase overwrites ignored colliding file with target bytes")
	fmt.Printf("IGNORED_COLLISION status=%d output=%q bytes_before=%q bytes_after=%q\n  before: %s\n  after:  %s\n", code, out, ignoredBefore, ignoredAfter, before, p.state(r))
	// updateRefs: duplicate a branch at the commit to be rewritten. Include a
	// conflict fixture and continue with plain `git rebase --continue`.
	for _, scopedFalse := range []bool{false, true} {
		label := "update-refs-config-true"
		if scopedFalse {
			label = "update-refs-command-false"
		}
		r = p.fixture(label, seed, refs, descendant)
		p.mustGit(r, "branch", "other", ancestor)
		p.mustGit(r, "config", "rebase.updateRefs", "true")
		args := []string{"rebase", "--no-autostash", divergent + "^{commit}"}
		if scopedFalse {
			args = append([]string{"-c", "rebase.updateRefs=false"}, args...)
		}
		before = p.state(r)
		out, code = p.tryGit(r, args...)
		otherAfter := p.mustGit(r, "rev-parse", "refs/heads/other")
		check(code == 0, "updateRefs case completes")
		if scopedFalse {
			check(otherAfter == ancestor, "command-scoped false preserves other branch")
		} else {
			check(otherAfter != ancestor, "configured true moves other branch at rewritten commit")
		}
		fmt.Printf("UPDATE_REFS scoped_false=%t status=%d output=%q\n  before: %s\n  after:  %s\n", scopedFalse, code, out, before, p.state(r))
	}
	r = p.fixture("update-refs-conflict", seed, refs, ancestor)
	// Put another branch at the first of two replay commits; the second conflicts.
	p.mustGit(r, "checkout", "-qb", "target-branch", ancestor)
	p.write(r, "common", "target\n")
	conflictTarget := p.commit(r, "target common")
	p.mustGit(r, "checkout", "-q", "main")
	p.write(r, "interior", "interior\n")
	interior := p.commit(r, "interior replay")
	p.mustGit(r, "branch", "other", interior)
	p.write(r, "common", "current\n")
	current := p.commit(r, "conflicting replay")
	p.mustGit(r, "config", "rebase.updateRefs", "true")
	before = p.state(r)
	out, code = p.selectedRebase(r, conflictTarget)
	check(code != 0 && strings.Contains(p.state(r), "UU common"), "conflict occurs on later replay commit")
	fmt.Printf("CONFLICT_START status=%d output=%q\n  before: %s\n  after:  %s\n", code, out, before, p.state(r))
	p.write(r, "common", "resolved\n")
	p.mustGit(r, "add", "common")
	contOut, contCode := p.tryGit(r, "rebase", "--continue")
	otherAfter := p.mustGit(r, "rev-parse", "refs/heads/other")
	check(contCode == 0, "plain git rebase --continue completes conflict")
	check(otherAfter == interior, "command-scoped false still prevents interior ref update after plain continue despite config true")
	check(current != interior, "conflict fixture contains multiple replay commits")
	fmt.Printf("CONFLICT_CONTINUE_PLAIN status=%d output=%q\n  after: %s\n", contCode, contOut, p.state(r))
	// Merge topology: compare default flattening with explicit --rebase-merges.
	mergeSeed := p.mk("merge-seed")
	p.write(mergeSeed, "base", "base\n")
	mroot := p.commit(mergeSeed, "root")
	p.write(mergeSeed, "main", "main\n")
	mmain := p.commit(mergeSeed, "main")
	p.mustGit(mergeSeed, "checkout", "-qb", "side", mroot)
	p.write(mergeSeed, "side", "side\n")
	side := p.commit(mergeSeed, "side")
	p.mustGit(mergeSeed, "checkout", "-q", "main")
	p.mustGit(mergeSeed, "merge", "--no-ff", "-qm", "merge side", "side")
	mergeTip := p.mustGit(mergeSeed, "rev-parse", "HEAD")
	p.write(mergeSeed, "after", "after\n")
	afterMerge := p.commit(mergeSeed, "after merge")
	p.mustGit(mergeSeed, "checkout", "-qb", "divergent-target", mroot)
	p.write(mergeSeed, "target", "target\n")
	topologyTarget := p.commit(mergeSeed, "divergent target")
	for _, preserve := range []bool{false, true} {
		label := "merge-topology-default"
		if preserve {
			label = "merge-topology-rebase-merges"
		}
		r = p.fixture(label, mergeSeed, map[string]string{"merge-target": topologyTarget}, afterMerge)
		args := []string{"-c", "rebase.updateRefs=false", "rebase", "--no-autostash"}
		if preserve {
			args = append(args, "--rebase-merges")
		}
		args = append(args, topologyTarget+"^{commit}")
		out, code = p.tryGit(r, args...)
		graph, _ := p.git(r, "log", "--graph", "--oneline", "--decorate", "--all")
		mergeCountText := p.mustGit(r, "rev-list", "--count", "--merges", "HEAD")
		mergeCount := 0
		_, err := fmt.Sscanf(mergeCountText, "%d", &mergeCount)
		must(err)
		check(code == 0, "merge topology rebase completes")
		if preserve {
			check(mergeCount >= 1, "--rebase-merges preserves at least one merge")
		} else {
			check(mergeCount == 0, "default rebase flattens all merges")
		}
		parents, _ := p.git(r, "rev-list", "--parents", "-n", "1", "HEAD")
		fmt.Printf("MERGE_TOPOLOGY rebase_merges=%t status=%d output=%q source_merge=%s source_side=%s source_main=%s\n  HEAD_PARENTS=%q\n  graph:\n%s\n", preserve, code, out, mergeTip, side, mmain, parents, graph)
	}
	// A branch name equal to the full OID must not shadow a commit expression.
	r = p.fixture("full-oid-ref-shadow", seed, refs, descendant)
	p.mustGit(r, "config", "advice.objectNameWarning", "false")
	p.mustGit(r, "branch", divergent, ancestor)
	resolved, resolvedCode := p.git(r, "-c", "advice.objectNameWarning=false", "rev-parse", "--verify", divergent+"^{commit}")
	p.mustGit(r, "checkout", "-qb", "shadow-test", descendant)
	out, code = p.tryGit(r, "-c", "advice.objectNameWarning=false", "-c", "rebase.updateRefs=false", "rebase", "--no-autostash", divergent+"^{commit}")
	resolvedOID := strings.TrimSpace(strings.SplitN(resolved, " | stderr=", 2)[0])
	check(resolvedCode == 0 && resolvedOID == divergent, "fullOID^{commit} resolves to selected commit despite same-name ref")
	_, ancestorCode := p.git(r, "merge-base", "--is-ancestor", divergent, "HEAD")
	check(code == 0 && ancestorCode == 0, "rebase uses peeled full OID, not shadow ref target")
	fmt.Printf("OID_SHADOW status=%d output=%q oid=%s shadow_ref=%s peeled_expression=%s\n  after: %s\n", code, out, divergent, p.mustGit(r, "rev-parse", "refs/heads/"+divergent), resolved, p.state(r))
}
