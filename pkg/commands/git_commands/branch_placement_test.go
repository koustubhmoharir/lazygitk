package git_commands

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func branchPlacementTestCommands(t *testing.T, root string) *BranchCommands {
	t.Helper()
	osCommand := oscommands.NewDummyOSCommand()
	return NewBranchCommands(NewGitCommon(common.NewDummyCommon(), nil, osCommand.Cmd, osCommand, MockRepoPaths(root), nil, nil))
}

func branchPlacementTestCommit(t *testing.T, root, file, contents, message string) string {
	t.Helper()
	writeTestFile(t, filepath.Join(root, file), []byte(contents), 0o644)
	gitTest(t, root, "add", "--", file)
	gitTest(t, root, "commit", "-qm", message)
	return gitTest(t, root, "rev-parse", "HEAD")
}

func branchPlacementRequirePlan(t *testing.T, commands *BranchCommands, name, target string) BranchPlacementPlan {
	t.Helper()
	plan, err := commands.PrepareBranchPlacement(name, target)
	if err != nil {
		t.Fatalf("prepare branch placement: %v", err)
	}
	return plan
}

func TestBranchPlacementPrepareAndApply(t *testing.T) {
	root, first, second := transactionHistory(t)
	commands := branchPlacementTestCommands(t, root)

	plan := branchPlacementRequirePlan(t, commands, "topic", first)
	assert.Equal(t, BranchPlacementCreate, plan.Action())
	assert.Equal(t, "topic", plan.Name())
	assert.Empty(t, plan.OldOID())
	assert.Equal(t, first, plan.TargetOID())
	if err := commands.ApplyBranchPlacement(plan); err != nil {
		t.Fatalf("create branch placement: %v", err)
	}
	assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/heads/topic"))
	assert.Contains(t, gitTest(t, root, "reflog", "-1", "--format=%gs", "topic"), "lazygitk: create branch at selected commit")

	plan = branchPlacementRequirePlan(t, commands, "topic", second)
	assert.Equal(t, BranchPlacementMove, plan.Action())
	assert.Equal(t, first, plan.OldOID())
	if err := commands.ApplyBranchPlacement(plan); err != nil {
		t.Fatalf("move branch placement: %v", err)
	}
	assert.Equal(t, second, gitTest(t, root, "rev-parse", "refs/heads/topic"))
	assert.Contains(t, gitTest(t, root, "reflog", "-1", "--format=%gs", "topic"), "lazygitk: move branch to selected commit")
	for _, current := range []bool{false, true} {
		name := "topic"
		if current {
			name = gitTest(t, root, "branch", "--show-current")
			if name == "" { // This fixture is detached; make topic current explicitly.
				gitTest(t, root, "checkout", "-q", "topic")
				name = "topic"
			}
		}
		before := gitTest(t, root, "reflog", "-1", "--format=%H %gs", name)
		noop := branchPlacementRequirePlan(t, commands, name, second)
		assert.Equal(t, BranchPlacementNoop, noop.Action())
		if err := commands.ApplyBranchPlacement(noop); err != nil {
			t.Fatalf("apply same-tip no-op (current=%v): %v", current, err)
		}
		assert.Equal(t, before, gitTest(t, root, "reflog", "-1", "--format=%H %gs", name))
	}
	gitTest(t, root, "update-ref", "refs/heads/topic", first, second)
	assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/heads/topic"), "the branch reflog's old OID supports explicit manual recovery")
}

func TestBranchPlacementLiteralValidationAndExactRef(t *testing.T) {
	root, first, _ := transactionHistory(t)
	commands := branchPlacementTestCommands(t, root)
	for _, name := range []string{"", "HEAD", "-leading", "@{-1}", "has space", "has\nnewline"} {
		t.Run(fmt.Sprintf("invalid name %q", name), func(t *testing.T) {
			if _, err := commands.PrepareBranchPlacement(name, first); err == nil {
				t.Fatalf("expected invalid literal branch name %q to fail", name)
			}
		})
	}
	blob := gitTest(t, root, "rev-parse", "HEAD:tracked")
	for _, oid := range []string{"not-an-object-id", blob} {
		if _, err := commands.PrepareBranchPlacement("topic", oid); err == nil {
			t.Fatalf("expected invalid/non-commit object %q to fail", oid)
		}
	}
	for _, name := range []string{"unicodé", "nested/child"} {
		plan := branchPlacementRequirePlan(t, commands, name, first)
		assert.Equal(t, name, plan.Name())
	}

	gitTest(t, root, "branch", "topic/child", first)
	childOnly := branchPlacementRequirePlan(t, commands, "topic", first)
	assert.Equal(t, BranchPlacementCreate, childOnly.Action(), "a prefix-matched child must not count as exact topic")
	assert.Error(t, commands.ApplyBranchPlacement(childOnly), "a conflicting descendant prevents creating its parent ref")
	assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/heads/topic/child"))
	assert.Error(t, exec.Command("git", "-C", root, "show-ref", "--verify", "--quiet", "refs/heads/topic").Run())
	gitTest(t, root, "pack-refs", "--all")
	packed := branchPlacementRequirePlan(t, commands, "topic/child", first)
	assert.Equal(t, BranchPlacementNoop, packed.Action())
}

func TestBranchPlacementRejectsAliasesAndCurrentMovement(t *testing.T) {
	root, first, second := transactionHistory(t)
	gitTest(t, root, "branch", "topic", first)
	gitTest(t, root, "symbolic-ref", "refs/heads/alias", "refs/heads/topic")
	commands := branchPlacementTestCommands(t, root)
	if _, err := commands.PrepareBranchPlacement("alias", second); err == nil || !strings.Contains(err.Error(), "symbolic alias") {
		t.Fatalf("expected direct symbolic alias rejection, got %v", err)
	}
	gitTest(t, root, "symbolic-ref", "refs/heads/broken", "refs/heads/missing")
	if _, err := commands.PrepareBranchPlacement("broken", second); err == nil {
		t.Fatal("expected broken symbolic alias rejection")
	}

	current := gitTest(t, root, "branch", "--show-current")
	if _, err := commands.PrepareBranchPlacement(current, first); err == nil || !strings.Contains(err.Error(), "Reset") {
		t.Fatalf("expected current branch movement reason mentioning Reset, got %v", err)
	}
}

func TestBranchPlacementUnbornCurrentAndDetachedHead(t *testing.T) {
	root := initHistoricalRepo(t)
	tree := gitTest(t, root, "mktree")
	unborn := gitTest(t, root, "symbolic-ref", "-q", "HEAD")
	commit := gitTest(t, root, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit-tree", tree, "-m", "empty")
	commands := branchPlacementTestCommands(t, root)
	currentName := strings.TrimPrefix(unborn, "refs/heads/")
	if _, err := commands.PrepareBranchPlacement(currentName, commit); err == nil || !strings.Contains(err.Error(), "current worktree") {
		t.Fatalf("expected creation of the unborn current branch to be refused, got %v", err)
	}
	assert.Equal(t, unborn, gitTest(t, root, "symbolic-ref", "-q", "HEAD"))
	assert.Error(t, exec.Command("git", "-C", root, "rev-parse", "--verify", "refs/heads/created").Run())
	plan := branchPlacementRequirePlan(t, commands, "created", commit)
	assert.Equal(t, BranchPlacementCreate, plan.Action())
	if err := commands.ApplyBranchPlacement(plan); err != nil {
		t.Fatalf("create a noncurrent branch while HEAD is unborn: %v", err)
	}
	assert.Equal(t, commit, gitTest(t, root, "rev-parse", "refs/heads/created"))
	assert.Equal(t, unborn, gitTest(t, root, "symbolic-ref", "-q", "HEAD"))

	root, first, second := transactionHistory(t)
	gitTest(t, root, "branch", "topic", first)
	gitTest(t, root, "checkout", "--detach", "-q", first)
	commands = branchPlacementTestCommands(t, root)
	plan = branchPlacementRequirePlan(t, commands, "topic", second)
	if err := commands.ApplyBranchPlacement(plan); err != nil {
		t.Fatalf("move branch while HEAD is detached: %v", err)
	}
	assert.Equal(t, first, gitTest(t, root, "rev-parse", "HEAD"))
}

func TestBranchPlacementLinkedWorktreeAndUserState(t *testing.T) {
	root, old, target := transactionHistory(t)
	gitTest(t, root, "branch", "topic", old)
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root, "worktree", "add", "-q", linked, "topic")
	commands := branchPlacementTestCommands(t, root)
	if _, err := commands.PrepareBranchPlacement("topic", target); err == nil || !strings.Contains(err.Error(), "another worktree") {
		t.Fatalf("expected linked worktree reservation rejection, got %v", err)
	}
	gitTest(t, root, "worktree", "remove", "-f", linked)

	writeTestFile(t, filepath.Join(root, "tracked"), []byte("staged bytes\n"), 0o644)
	gitTest(t, root, "add", "tracked")
	index := gitTest(t, root, "write-tree")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("unstaged bytes\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "loose"), []byte("untracked\x00bytes"), 0o600)
	gitTest(t, root, "config", "branch.topic.remote", "origin")
	gitTest(t, root, "config", "branch.topic.merge", "refs/heads/upstream-topic")
	head := gitTest(t, root, "rev-parse", "HEAD")
	headRef := gitTest(t, root, "symbolic-ref", "-q", "HEAD")
	status := gitTest(t, root, "status", "--porcelain=v1", "-z")
	plan := branchPlacementRequirePlan(t, commands, "topic", target)
	if err := commands.ApplyBranchPlacement(plan); err != nil {
		t.Fatalf("apply placement with dirty worktree: %v", err)
	}
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "refs/heads/topic"))
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, headRef, gitTest(t, root, "symbolic-ref", "-q", "HEAD"))
	assert.Equal(t, index, gitTest(t, root, "write-tree"))
	assert.Equal(t, status, gitTest(t, root, "status", "--porcelain=v1", "-z"))
	assert.Equal(t, "staged bytes", gitTest(t, root, "show", ":tracked"))
	assert.Equal(t, []byte("unstaged bytes\n"), mustReadFile(t, filepath.Join(root, "tracked")))
	assert.Equal(t, []byte("untracked\x00bytes"), mustReadFile(t, filepath.Join(root, "loose")))
	assert.Equal(t, "origin", gitTest(t, root, "config", "--get", "branch.topic.remote"))
	assert.Equal(t, "refs/heads/upstream-topic", gitTest(t, root, "config", "--get", "branch.topic.merge"))
}

func TestBranchPlacementStaleSnapshots(t *testing.T) {
	for _, change := range []string{"move", "delete", "create"} {
		t.Run(change, func(t *testing.T) {
			root, first, second := transactionHistory(t)
			commands := branchPlacementTestCommands(t, root)
			if change != "create" {
				gitTest(t, root, "branch", "topic", first)
			}
			plan := branchPlacementRequirePlan(t, commands, "topic", second)
			switch change {
			case "move":
				gitTest(t, root, "update-ref", "refs/heads/topic", second, first)
			case "delete":
				gitTest(t, root, "update-ref", "-d", "refs/heads/topic", first)
			case "create":
				gitTest(t, root, "update-ref", "refs/heads/topic", first)
			}
			beforeExists := exec.Command("git", "-C", root, "show-ref", "--verify", "--quiet", "refs/heads/topic").Run() == nil
			before := ""
			if beforeExists {
				before = gitTest(t, root, "rev-parse", "refs/heads/topic")
			}
			err := commands.ApplyBranchPlacement(plan)
			if !errors.Is(err, ErrBranchPlacementChanged) {
				t.Fatalf("expected stale plan error, got %v", err)
			}
			afterExists := exec.Command("git", "-C", root, "show-ref", "--verify", "--quiet", "refs/heads/topic").Run() == nil
			assert.Equal(t, beforeExists, afterExists)
			if beforeExists {
				assert.Equal(t, before, gitTest(t, root, "rev-parse", "refs/heads/topic"))
			}
		})
	}
}

type branchPlacementRaceBuilder struct {
	inner    oscommands.ICmdObjBuilder
	callback func()
	calls    int
}

func (builder *branchPlacementRaceBuilder) New(args []string) *oscommands.CmdObj {
	if len(args) > 1 && args[1] == "update-ref" {
		builder.calls++
		if builder.callback != nil {
			builder.callback()
		}
	}
	return builder.inner.New(args)
}

func (builder *branchPlacementRaceBuilder) NewShell(command, functionsFile string) *oscommands.CmdObj {
	return builder.inner.NewShell(command, functionsFile)
}

func (builder *branchPlacementRaceBuilder) Quote(value string) string {
	return builder.inner.Quote(value)
}

func branchPlacementCommandsWithRace(t *testing.T, root string, callback func()) (*BranchCommands, *branchPlacementRaceBuilder) {
	t.Helper()
	osCommand := oscommands.NewDummyOSCommand()
	builder := &branchPlacementRaceBuilder{inner: osCommand.Cmd, callback: callback}
	commands := NewBranchCommands(NewGitCommon(common.NewDummyCommon(), nil, builder, osCommand, MockRepoPaths(root), nil, nil))
	return commands, builder
}

func TestBranchPlacementRaceWindows(t *testing.T) {
	t.Run("external move is stopped by CAS", func(t *testing.T) {
		root, first, second := transactionHistory(t)
		third := branchPlacementTestCommit(t, root, "tracked", "three\n", "three")
		gitTest(t, root, "branch", "topic", first)
		plan := branchPlacementRequirePlan(t, branchPlacementTestCommands(t, root), "topic", second)
		commands, builder := branchPlacementCommandsWithRace(t, root, func() { gitTest(t, root, "update-ref", "refs/heads/topic", third, first) })
		err := commands.ApplyBranchPlacement(plan)
		if !errors.Is(err, ErrBranchPlacementChanged) {
			t.Fatalf("expected external ref move to invalidate plan, got %v", err)
		}
		assert.Equal(t, 1, builder.calls)
		assert.Equal(t, third, gitTest(t, root, "rev-parse", "refs/heads/topic"))
	})

	t.Run("alias appears before locked guard", func(t *testing.T) {
		root, first, second := transactionHistory(t)
		gitTest(t, root, "branch", "topic", first)
		gitTest(t, root, "branch", "other", first)
		plan := branchPlacementRequirePlan(t, branchPlacementTestCommands(t, root), "topic", second)
		commands, builder := branchPlacementCommandsWithRace(t, root, func() {
			gitTest(t, root, "update-ref", "-d", "refs/heads/topic", first)
			gitTest(t, root, "symbolic-ref", "refs/heads/topic", "refs/heads/other")
		})
		err := commands.ApplyBranchPlacement(plan)
		if !errors.Is(err, ErrBranchPlacementChanged) {
			t.Fatalf("expected locked symbolic type guard to abort, got %v", err)
		}
		assert.Equal(t, 1, builder.calls)
		assert.Equal(t, "refs/heads/other", gitTest(t, root, "symbolic-ref", "refs/heads/topic"))
		assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/heads/other"))
	})

	t.Run("worktree reservation appears before locked guard", func(t *testing.T) {
		root, first, second := transactionHistory(t)
		gitTest(t, root, "branch", "topic", first)
		plan := branchPlacementRequirePlan(t, branchPlacementTestCommands(t, root), "topic", second)
		linked := filepath.Join(t.TempDir(), "raced-worktree")
		commands, builder := branchPlacementCommandsWithRace(t, root, func() { gitTest(t, root, "worktree", "add", "-q", linked, "topic") })
		err := commands.ApplyBranchPlacement(plan)
		if err == nil || !strings.Contains(err.Error(), "another worktree") {
			t.Fatalf("expected new worktree reservation to block the move, got %v", err)
		}
		assert.Equal(t, 1, builder.calls)
		assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/heads/topic"))
	})
}
