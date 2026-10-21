package git_commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func branchPlacementMustNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

func branchPlacementMustBeOccupied(t *testing.T, occupation *branchPlacementOccupation, err error) {
	t.Helper()
	branchPlacementMustNoError(t, err, "query branch placement occupancy")
	if occupation == nil {
		t.Fatal("expected branch placement occupation")
	}
}

func realBranchPlacementCommands(t *testing.T, root string) *BranchCommands {
	t.Helper()
	osCommand := oscommands.NewDummyOSCommand()
	paths, err := GetRepoPathsForDir(root, osCommand.Cmd)
	branchPlacementMustNoError(t, err, "get repository paths")
	return NewBranchCommands(NewGitCommon(nil, nil, osCommand.Cmd, osCommand, paths, nil, nil))
}

func TestBranchPlacementOccupancyCurrentAndFree(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "file"), []byte("content\n"), 0o644)
	gitTest(t, root, "add", "file")
	gitTest(t, root, "commit", "-qm", "initial")
	commands := realBranchPlacementCommands(t, root)
	branch := gitTest(t, root, "branch", "--show-current")

	got, err := commands.branchPlacementOccupancy("refs/heads/" + branch)
	branchPlacementMustBeOccupied(t, got, err)
	assert.Equal(t, "checkout", got.Kind)
	assert.True(t, got.Current)

	got, err = commands.branchPlacementOccupancy("refs/heads/free")
	branchPlacementMustNoError(t, err, "query unoccupied branch")
	assert.Nil(t, got)
}

func TestBranchPlacementOccupancyLinkedAndMissingWorktree(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "file"), []byte("content\n"), 0o644)
	gitTest(t, root, "add", "file")
	gitTest(t, root, "commit", "-qm", "initial")
	branch := "linked-branch"
	gitTest(t, root, "branch", branch)
	linked := filepath.Join(t.TempDir(), "linked\nworktree")
	gitTest(t, root, "worktree", "add", "-q", linked, branch)
	commands := realBranchPlacementCommands(t, root)
	got, err := commands.branchPlacementOccupancy("refs/heads/" + branch)
	branchPlacementMustBeOccupied(t, got, err)
	assert.Equal(t, "checkout", got.Kind)
	assert.False(t, got.Current)

	// Make the linked worktree current in the scanned repository context and
	// verify that its reservation is reported as current.
	linkedPaths := *commands.repoPaths
	linkedPaths.worktreePath = linked
	linkedPaths.worktreeGitDirPath = gitTest(t, linked, "rev-parse", "--path-format=absolute", "--absolute-git-dir")
	osCommand := oscommands.NewDummyOSCommand()
	linkedCommands := NewBranchCommands(NewGitCommon(nil, nil, osCommand.Cmd, osCommand, &linkedPaths, nil, nil))
	got, err = linkedCommands.branchPlacementOccupancy("refs/heads/" + branch)
	branchPlacementMustBeOccupied(t, got, err)
	assert.True(t, got.Current)
	mainRebaseDir := filepath.Join(commands.repoPaths.WorktreeGitDirPath(), "rebase-merge")
	branchPlacementMustNoError(t, os.MkdirAll(mainRebaseDir, 0o755), "create main worktree rebase metadata")
	branchPlacementMustNoError(t, os.WriteFile(filepath.Join(mainRebaseDir, "head-name"), []byte("refs/heads/"+branch+"\n"), 0o600), "write competing reservation")
	got, err = linkedCommands.branchPlacementOccupancy("refs/heads/" + branch)
	branchPlacementMustBeOccupied(t, got, err)
	assert.Equal(t, "checkout", got.Kind)
	assert.True(t, got.Current, "current worktree reservation should be preferred")
	branchPlacementMustNoError(t, os.Remove(filepath.Join(mainRebaseDir, "head-name")), "remove main worktree rebase metadata")

	// A bare common config skips only its own HEAD; linked metadata still counts.
	gitTest(t, root, "config", "core.bare", "true")
	got, err = commands.branchPlacementOccupancy("refs/heads/" + branch)
	branchPlacementMustBeOccupied(t, got, err)
	assert.Equal(t, "checkout", got.Kind)
	gitTest(t, root, "config", "core.bare", "false")

	// Git metadata remains authoritative after its physical worktree disappears.
	branchPlacementMustNoError(t, os.RemoveAll(linked), "remove linked worktree directory")
	got, err = commands.branchPlacementOccupancy("refs/heads/" + branch)
	branchPlacementMustBeOccupied(t, got, err)
	assert.Equal(t, "checkout", got.Kind)
	gitDir := gitTest(t, root, "rev-parse", "--path-format=absolute", "--git-common-dir")
	metadataEntries, err := os.ReadDir(filepath.Join(gitDir, "worktrees"))
	branchPlacementMustNoError(t, err, "read worktree metadata entries")
	if len(metadataEntries) == 0 {
		t.Fatal("expected linked worktree metadata entry")
	}
	metadataDir := filepath.Join(gitDir, "worktrees", metadataEntries[0].Name())
	branchPlacementMustNoError(t, os.Remove(filepath.Join(metadataDir, "gitdir")), "remove stale gitdir pointer")
	got, err = commands.branchPlacementOccupancy("refs/heads/free")
	branchPlacementMustNoError(t, err, "query unrelated branch after stale worktree")
	assert.Nil(t, got)
}

func TestBranchPlacementOccupancyDetachedWorktreeDoesNotReserveBranch(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "file"), []byte("content\n"), 0o644)
	gitTest(t, root, "add", "file")
	gitTest(t, root, "commit", "-qm", "initial")
	gitTest(t, root, "branch", "unreserved")
	linked := filepath.Join(t.TempDir(), "detached")
	gitTest(t, root, "worktree", "add", "-q", "--detach", linked, "HEAD")
	got, err := realBranchPlacementCommands(t, root).branchPlacementOccupancy("refs/heads/unreserved")
	branchPlacementMustNoError(t, err, "query branch with detached worktree")
	assert.Nil(t, got)
}

func TestBranchPlacementOccupancyOperationReservations(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "file"), []byte("content\n"), 0o644)
	gitTest(t, root, "add", "file")
	gitTest(t, root, "commit", "-qm", "initial")
	commands := realBranchPlacementCommands(t, root)
	gitDir := commands.repoPaths.WorktreeGitDirPath()
	zero := strings.Repeat("0", 40)
	valid := gitTest(t, root, "rev-parse", "HEAD")
	rebaseMerge := filepath.Join(gitDir, "rebase-merge")
	branchPlacementMustNoError(t, os.MkdirAll(rebaseMerge, 0o755), "create rebase-merge fixture")
	branchPlacementMustNoError(t, os.WriteFile(filepath.Join(rebaseMerge, "head-name"), []byte("refs/heads/rebase-target\r\n"), 0o600), "write rebase-merge head-name")
	got, err := commands.branchPlacementOccupancy("refs/heads/rebase-target")
	branchPlacementMustBeOccupied(t, got, err)
	assert.Equal(t, "rebase", got.Kind)

	branchPlacementMustNoError(t, os.Remove(filepath.Join(rebaseMerge, "head-name")), "remove rebase-merge head-name")
	rebaseApply := filepath.Join(gitDir, "rebase-apply")
	branchPlacementMustNoError(t, os.MkdirAll(rebaseApply, 0o755), "create rebase-apply fixture")
	branchPlacementMustNoError(t, os.WriteFile(filepath.Join(rebaseApply, "head-name"), []byte("refs/heads/apply-target\n"), 0o600), "write rebase-apply head-name")
	got, err = commands.branchPlacementOccupancy("refs/heads/apply-target")
	branchPlacementMustBeOccupied(t, got, err)
	assert.Equal(t, "rebase", got.Kind)
	branchPlacementMustNoError(t, os.Remove(filepath.Join(rebaseApply, "head-name")), "remove rebase-apply head-name")

	branchPlacementMustNoError(t, os.WriteFile(filepath.Join(gitDir, "BISECT_START"), []byte("bisect-target\n"), 0o600), "write BISECT_START")
	got, err = commands.branchPlacementOccupancy("refs/heads/bisect-target")
	branchPlacementMustBeOccupied(t, got, err)
	assert.Equal(t, "bisect", got.Kind)
	branchPlacementMustNoError(t, os.Remove(filepath.Join(gitDir, "BISECT_START")), "remove BISECT_START")

	triples := strings.Join([]string{"refs/heads/update-target", valid, zero, ""}, "\n")
	branchPlacementMustNoError(t, os.WriteFile(filepath.Join(rebaseMerge, "update-refs"), []byte(strings.ReplaceAll(triples, "\n", "\r\n")), 0o600), "write rebase update-refs")
	got, err = commands.branchPlacementOccupancy("refs/heads/update-target")
	branchPlacementMustBeOccupied(t, got, err)
	assert.Equal(t, "rebase update-refs", got.Kind)
}

func TestBranchPlacementOccupancyMalformedStateFailsClosed(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "file"), []byte("content\n"), 0o644)
	gitTest(t, root, "add", "file")
	gitTest(t, root, "commit", "-qm", "initial")
	commands := realBranchPlacementCommands(t, root)
	rebaseDir := filepath.Join(commands.repoPaths.WorktreeGitDirPath(), "rebase-merge")
	branchPlacementMustNoError(t, os.MkdirAll(rebaseDir, 0o755), "create rebase fixture")
	updateRefs := filepath.Join(rebaseDir, "update-refs")
	branchPlacementMustNoError(t, os.WriteFile(updateRefs, []byte("refs/heads/x\nnot-an-oid\n"+strings.Repeat("0", 40)+"\n"), 0o600), "write malformed update-refs")
	got, err := commands.branchPlacementOccupancy("refs/heads/free")
	assert.Error(t, err)
	assert.Nil(t, got)

	branchPlacementMustNoError(t, os.WriteFile(updateRefs, []byte("refs/heads/x\n"+strings.Repeat("0", 40)+"\n"+strings.Repeat("0", 40)+"\n\n"), 0o600), "write extra-line update-refs")
	got, err = commands.branchPlacementOccupancy("refs/heads/free")
	assert.Error(t, err, "an extra blank line must not be discarded")
	assert.Nil(t, got)

	branchPlacementMustNoError(t, os.Remove(updateRefs), "remove update-refs fixture")
	branchPlacementMustNoError(t, os.Mkdir(filepath.Join(commands.repoPaths.WorktreeGitDirPath(), "BISECT_START"), 0o755), "create unreadable BISECT_START path")
	got, err = commands.branchPlacementOccupancy("refs/heads/free")
	assert.Error(t, err, "an unreadable active-state path must fail closed")
	assert.Nil(t, got)
}

func TestBranchPlacementOccupancyMissingCurrentMetadataFailsClosed(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "file"), []byte("content\n"), 0o644)
	gitTest(t, root, "add", "file")
	gitTest(t, root, "commit", "-qm", "initial")
	commands := realBranchPlacementCommands(t, root)
	missingRegistryPath := filepath.Join(root, ".git", "unregistered-worktree")
	branchPlacementMustNoError(t, os.MkdirAll(missingRegistryPath, 0o755), "create unregistered metadata path")
	commands.repoPaths.worktreeGitDirPath = missingRegistryPath
	got, err := commands.branchPlacementOccupancy("refs/heads/free")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestBranchPlacementOccupancyDoesNotMutateRepository(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "file"), []byte("content\n"), 0o644)
	gitTest(t, root, "add", "file")
	gitTest(t, root, "commit", "-qm", "initial")
	beforeHead := gitTest(t, root, "rev-parse", "HEAD")
	beforeIndex := gitTest(t, root, "write-tree")
	beforeStatus := gitTest(t, root, "status", "--porcelain=v1", "-z")
	_, err := realBranchPlacementCommands(t, root).branchPlacementOccupancy("refs/heads/free")
	branchPlacementMustNoError(t, err, "query unoccupied branch")
	assert.Equal(t, beforeHead, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, beforeIndex, gitTest(t, root, "write-tree"))
	assert.Equal(t, beforeStatus, gitTest(t, root, "status", "--porcelain=v1", "-z"))
}
