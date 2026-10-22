package git_commands

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestBranchFastForwardPreservesOtherState(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	gitTest(t, root, "update-ref", "refs/tags/unrelated", old)
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("staged bytes\n"), 0o644)
	gitTest(t, root, "add", "tracked")
	index := gitTest(t, root, "write-tree")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("unstaged bytes\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "loose"), []byte("untracked\x00bytes"), 0o600)
	headBytes := mustReadFile(t, filepath.Join(root, ".git", "HEAD"))
	indexBytes := mustReadFile(t, filepath.Join(root, ".git", "index"))
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchFastForward("selected", old, target)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, plan.NeedsUpdate())
	assert.Equal(t, old, plan.OldOID())
	if err := commands.ApplyBranchFastForward(plan); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "refs/heads/selected"))
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, index, gitTest(t, root, "write-tree"))
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/tags/unrelated"))
	assert.Equal(t, []byte("unstaged bytes\n"), mustReadFile(t, filepath.Join(root, "tracked")))
	assert.Equal(t, []byte("untracked\x00bytes"), mustReadFile(t, filepath.Join(root, "loose")))
	assert.Equal(t, headBytes, mustReadFile(t, filepath.Join(root, ".git", "HEAD")))
	assert.Equal(t, indexBytes, mustReadFile(t, filepath.Join(root, ".git", "index")))
}

func TestBranchFastForwardCheckIsReadOnlyAndApplyAdvancesOnlySelectedRef(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	gitTest(t, root, "update-ref", "refs/tags/unrelated", old)
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("staged bytes\n"), 0o644)
	gitTest(t, root, "add", "tracked")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("unstaged bytes\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "loose"), []byte("untracked\x00bytes"), 0o600)
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchFastForward("selected", old, target)
	if err != nil {
		t.Fatal(err)
	}
	before := branchFastForwardTreeSnapshot(t, root)
	if err := commands.CheckBranchFastForward(plan); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, before, branchFastForwardTreeSnapshot(t, root), "preflight must not alter repository files or metadata")
	if err := commands.ApplyBranchFastForward(plan); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "refs/heads/selected"))
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/tags/unrelated"))
	assert.Equal(t, []byte("unstaged bytes\n"), mustReadFile(t, filepath.Join(root, "tracked")))
	assert.Equal(t, []byte("untracked\x00bytes"), mustReadFile(t, filepath.Join(root, "loose")))
}

func TestBranchFastForwardCheckRejectsStaleForeignAndUnpreparedPlansReadOnly(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchFastForward("selected", old, target)
	if err != nil {
		t.Fatal(err)
	}
	var unprepared BranchFastForwardPlan
	assertBranchFastForwardCheckReadOnly(t, root, func() error { return commands.CheckBranchFastForward(unprepared) }, "was not prepared")
	foreign := branchSyncTestCommands(t, root)
	assertBranchFastForwardCheckReadOnly(t, root, func() error { return foreign.CheckBranchFastForward(plan) }, "different repository instance")
	gitTest(t, root, "update-ref", "refs/heads/selected", target, old)
	assertBranchFastForwardCheckReadOnly(t, root, func() error { return commands.CheckBranchFastForward(plan) }, ErrBranchSyncChanged.Error())
}

func TestBranchFastForwardCheckRejectsWorktreeOccupiedAfterPreparationReadOnly(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchFastForward("selected", old, target)
	if err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root, "worktree", "add", "-q", linked, "selected")
	assertBranchFastForwardCheckReadOnly(t, root, func() error { return commands.CheckBranchFastForward(plan) }, "another worktree")
}

func TestBranchFastForwardCheckAllowsEqualityAndAlreadyAheadWhenOccupied(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "selected", target)
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root, "worktree", "add", "-q", linked, "selected")
	commands := branchSyncTestCommands(t, root)
	for _, test := range []struct{ expected, requested string }{{target, target}, {target, old}} {
		plan, err := commands.PrepareBranchFastForward("selected", test.expected, test.requested)
		if err != nil {
			t.Fatal(err)
		}
		before := branchFastForwardTreeSnapshot(t, root)
		if err := commands.CheckBranchFastForward(plan); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, before, branchFastForwardTreeSnapshot(t, root))
	}
}

func assertBranchFastForwardCheckReadOnly(t *testing.T, root string, check func() error, message string) {
	t.Helper()
	before := branchFastForwardTreeSnapshot(t, root)
	err := check()
	if err == nil || !strings.Contains(err.Error(), message) {
		t.Fatalf("expected error containing %q, got %v", message, err)
	}
	assert.Equal(t, before, branchFastForwardTreeSnapshot(t, root), "rejected preflight must not alter repository files or metadata")
}

func branchFastForwardTreeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		snapshot[rel] = fmt.Sprintf("%o:%x", info.Mode(), bytes)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	gitDir := gitTest(t, root, "rev-parse", "--absolute-git-dir")
	err = filepath.WalkDir(gitDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(gitDir, path)
		if err != nil {
			return err
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		snapshot[".git/"+rel] = fmt.Sprintf("%o:%x", info.Mode(), bytes)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestBranchFastForwardNoopNeverRewinds(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "selected", target)
	commands := branchSyncTestCommands(t, root)
	for _, oid := range []string{target, old} {
		plan, err := commands.PrepareBranchFastForward("selected", target, oid)
		if err != nil {
			t.Fatal(err)
		}
		assert.False(t, plan.NeedsUpdate())
		if err := commands.ApplyBranchFastForward(plan); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, target, gitTest(t, root, "rev-parse", "refs/heads/selected"))
	}
}

func TestBranchFastForwardRefusesRewrittenHistoryWithSharedBase(t *testing.T) {
	root, base, current := transactionHistory(t)
	gitTest(t, root, "branch", "selected", current)
	// A rewritten sibling can have the same tree without being a fast-forward.
	tree := gitTest(t, root, "rev-parse", current+"^{tree}")
	rewritten := gitTest(t, root, "commit-tree", tree, "-p", base, "-m", "rewritten upstream")
	_, err := branchSyncTestCommands(t, root).PrepareBranchFastForward("selected", current, rewritten)
	assert.ErrorIs(t, err, ErrBranchSyncDiverged)
	assert.Equal(t, current, gitTest(t, root, "rev-parse", "refs/heads/selected"))
}

func TestBranchFastForwardRejectsChangedAndDiverged(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	commands := branchSyncTestCommands(t, root)
	if _, err := commands.PrepareBranchFastForward("selected", target, old); !errors.Is(err, ErrBranchSyncChanged) {
		t.Fatalf("expected changed source, got %v", err)
	}
	diverged := gitTest(t, root, "commit-tree", gitTest(t, root, "rev-parse", "HEAD^{tree}"), "-m", "diverged")
	if _, err := commands.PrepareBranchFastForward("selected", old, diverged); !errors.Is(err, ErrBranchSyncDiverged) || !strings.Contains(err.Error(), "requires checkout") {
		t.Fatalf("expected contextual divergence, got %v", err)
	}
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/selected"))
	plan, err := commands.PrepareBranchFastForward("selected", old, target)
	if err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "update-ref", "refs/heads/selected", target, old)
	if err := commands.ApplyBranchFastForward(plan); !errors.Is(err, ErrBranchSyncChanged) {
		t.Fatalf("expected stale plan rejection, got %v", err)
	}
}

func TestBranchFastForwardRejectsCurrentAndReservedRealUpdates(t *testing.T) {
	root, old, target := transactionHistory(t)
	current := gitTest(t, root, "branch", "--show-current")
	currentOID := gitTest(t, root, "rev-parse", "refs/heads/"+current)
	tree := gitTest(t, root, "rev-parse", "HEAD^{tree}")
	forward := gitTest(t, root, "commit-tree", tree, "-p", currentOID, "-m", "forward")
	if _, err := branchSyncTestCommands(t, root).PrepareBranchFastForward(current, currentOID, forward); err == nil || !strings.Contains(err.Error(), "current worktree") {
		t.Fatalf("expected current branch refusal, got %v", err)
	}
	gitTest(t, root, "branch", "selected", old)
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root, "worktree", "add", "-q", linked, "selected")
	if _, err := branchSyncTestCommands(t, root).PrepareBranchFastForward("selected", old, target); err == nil || !strings.Contains(err.Error(), "another worktree") {
		t.Fatalf("expected linked worktree refusal, got %v", err)
	}
	gitTest(t, root, "worktree", "remove", "-f", linked)
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchFastForward("selected", old, target)
	if err != nil {
		t.Fatal(err)
	}
	// An operation reservation installed after preparation is caught by placement revalidation.
	gitTest(t, root, "worktree", "add", "-q", linked, "selected")
	if err := commands.ApplyBranchFastForward(plan); err == nil {
		t.Fatal("expected post-prepare reservation to block apply")
	}
}

func TestBranchFastForwardNoopAllowedWhenOccupiedAndRejectsUnprepared(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "selected", target)
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root, "worktree", "add", "-q", linked, "selected")
	commands := branchSyncTestCommands(t, root)
	var zero BranchFastForwardPlan
	if err := commands.ApplyBranchFastForward(zero); err == nil {
		t.Fatal("expected zero plan rejection")
	}
	plan, err := commands.PrepareBranchFastForward("selected", target, old)
	if err != nil {
		t.Fatal(err)
	}
	if err := commands.ApplyBranchFastForward(plan); err != nil {
		t.Fatal(err)
	}
}

func TestBranchFastForwardRejectsRevisionExpressionsAndBadOIDFormat(t *testing.T) {
	root, old, target := transactionHistory(t)
	commands := branchSyncTestCommands(t, root)
	for _, oid := range []string{"HEAD", old[:12], strings.Repeat("a", len(old)+1)} {
		if _, err := commands.PrepareBranchFastForward("main", old, oid); err == nil {
			t.Fatalf("expected invalid target %q rejected", oid)
		}
	}
	if _, err := commands.PrepareBranchFastForward("main", old[:12], target); err == nil {
		t.Fatal("expected abbreviated expected OID rejected")
	}
}

type fastForwardFailBuilder struct{ rootedTestBuilder }

func (builder fastForwardFailBuilder) New(args []string) *oscommands.CmdObj {
	for _, arg := range args {
		if arg == "merge-base" {
			args = []string{"definitely-not-a-git-command"}
			break
		}
	}
	return builder.rootedTestBuilder.New(args)
}

func TestBranchFastForwardDistinguishesAncestryQueryFailure(t *testing.T) {
	root, old, target := transactionHistory(t)
	osCommand := oscommands.NewDummyOSCommand()
	common := NewGitCommon(common.NewDummyCommon(), nil, fastForwardFailBuilder{rootedTestBuilder{inner: osCommand.Cmd, root: root}}, osCommand, MockRepoPaths(root), nil, nil)
	gitTest(t, root, "branch", "selected", old)
	_, err := NewSyncCommands(common).PrepareBranchFastForward("selected", old, target)
	if err == nil || errors.Is(err, ErrBranchSyncDiverged) || !strings.Contains(err.Error(), "definitely-not-a-git-command") {
		t.Fatalf("ancestry query failure was collapsed into divergence: %v (%s)", err, fmt.Sprintf("%T", err))
	}
}

func TestBranchFastForwardPreservesUncertainPlacementCommitError(t *testing.T) {
	original := fmt.Errorf("wrapped: %w", &branchRefCommitError{cause: ErrBranchPlacementChanged})
	mapped := mapBranchFastForwardPlacementError(original)
	assert.Same(t, original, mapped)
	assert.False(t, errors.Is(mapped, ErrBranchSyncChanged), "an uncertain commit must never be classified for automatic retry")
}

func TestBranchFastForwardOIDNamedRefsUseLiteralCommitAncestry(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	// Each complete object ID is also a branch name pointing at the other
	// commit. The ancestry query must follow the literal OIDs, not those refs.
	gitTest(t, root, "branch", old, target)
	gitTest(t, root, "branch", target, old)
	commands := branchSyncTestCommands(t, root)

	plan, err := commands.PrepareBranchFastForward("selected", old, target)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.NeedsUpdate() {
		t.Fatal("literal old->target ancestry should produce a fast-forward")
	}
	if err := commands.ApplyBranchFastForward(plan); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "refs/heads/selected"))

	// With selected already at target, checking whether the requested old OID
	// is its ancestor must also use object identity despite both shadow refs.
	noop, err := commands.PrepareBranchFastForward("selected", target, old)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, noop.NeedsUpdate())
	if err := commands.ApplyBranchFastForward(noop); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "refs/heads/selected"))
}

func TestBranchFastForwardPlanRemainsBoundToItsBackend(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "selected", old)
	commands := branchSyncTestCommands(t, root)
	plan, err := commands.PrepareBranchFastForward("selected", old, target)
	if !assert.NoError(t, err) {
		return
	}
	// Even equivalent data must not redirect a prepared operation through a
	// replacement backend after a repository switch/reload.
	err = branchSyncTestCommands(t, root).ApplyBranchFastForward(plan)
	assert.ErrorContains(t, err, "different repository instance")
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/selected"))
}
