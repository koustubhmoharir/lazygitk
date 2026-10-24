package git_commands

import (
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestCapturedCombineOrdersByAncestryAndRetainsMessagesAndInterveningCommits(t *testing.T) {
	root := initHistoricalRepo(t)
	base := capturedCherryPickCommit(t, root, "base", "base\n", "base", capturedCherryPickDate)
	first := capturedCherryPickCommit(t, root, "one", "one\n", "first subject\nfull body\n", "2030-01-01T00:00:00Z")
	middle := capturedCherryPickCommit(t, root, "middle", "middle\n", "middle", "2000-01-01T00:00:00Z")
	last := capturedCherryPickCommit(t, root, "last", "last\n", "third\nbody\n", capturedCherryPickDate)
	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareCombinePlan([]string{last, first})
	selectedRebaseRequireNoError(t, err)
	topology := plan.Topology()
	assert.Equal(t, first, topology.KeeperOID())
	assert.Equal(t, last, topology.HeadOID())
	assert.Equal(t, "reword "+first+"\nfixup "+last+"\npick "+middle+"\n", topology.Todo())
	assert.Contains(t, topology.Description(), "first subject\nfull body\n")
	assert.Contains(t, topology.Description(), "third\nbody\n")
	assert.Equal(t, "refs/heads/"+gitTest(t, root, "branch", "--show-current"), plan.HeadRef())
	assert.NoError(t, commands.CheckCombinePlan(plan))
	assert.Equal(t, base, gitTest(t, root, "rev-parse", first+"^"))
	mutable := plan.Topology()
	mutable.selected[0].OID = "mutated"
	mutable.selected[0].Parents[0] = "mutated"
	mutable.history[0].OID = "mutated"
	assert.Equal(t, first, plan.Topology().selected[0].OID)
	assert.Equal(t, base, plan.Topology().selected[0].Parents[0])
	assert.Equal(t, first, plan.Topology().history[0].OID)
}

func TestCapturedCombineRejectsInvalidAndDuplicateFullOIDs(t *testing.T) {
	root := initHistoricalRepo(t)
	first := capturedCherryPickCommit(t, root, "one", "1\n", "one", capturedCherryPickDate)
	second := capturedCherryPickCommit(t, root, "two", "2\n", "two", capturedCherryPickDate)
	commands := selectedCommitRebaseCommands(t, root)
	_, err := commands.PrepareCombinePlan([]string{first, first})
	assert.Error(t, err)
	_, err = commands.PrepareCombinePlan([]string{first[:12], second})
	assert.Error(t, err)
	_, err = commands.PrepareCombinePlan([]string{strings.Repeat("0", len(first)), second})
	assert.Error(t, err)
}

func TestCapturedCombineRejectsShallowBoundaryWithMissingParent(t *testing.T) {
	root := initHistoricalRepo(t)
	capturedCherryPickCommit(t, root, "one", "1\n", "one", capturedCherryPickDate)
	boundary := capturedCherryPickCommit(t, root, "two", "2\n", "two", capturedCherryPickDate)
	head := capturedCherryPickCommit(t, root, "three", "3\n", "three", capturedCherryPickDate)
	shallow := filepath.Join(t.TempDir(), "shallow")
	path := filepath.ToSlash(root)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	source := (&url.URL{Scheme: "file", Path: path}).String()
	gitTest(t, root, "-c", "protocol.file.allow=always", "clone", "--depth=2", source, shallow)
	commands := selectedCommitRebaseCommands(t, shallow)
	_, err := commands.PrepareCombinePlan([]string{boundary, head})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parent")
}

func TestCapturedCombineRetainsLiteralRawCommitMessageBytes(t *testing.T) {
	root := initHistoricalRepo(t)
	parent := capturedCherryPickCommit(t, root, "one", "1\n", "parent", capturedCherryPickDate)
	tree := gitTest(t, root, "rev-parse", parent+"^{tree}")
	literalMessage := "Subject  \n\n\nbody with trailing spaces   \n\n"
	raw := "tree " + tree + "\nparent " + parent + "\nauthor Test <test@example.invalid> 1577836800 +0000\ncommitter Test <test@example.invalid> 1577836800 +0000\n\n" + literalMessage
	cmd := exec.Command("git", "-C", root, "hash-object", "-t", "commit", "-w", "--stdin")
	cmd.Stdin = strings.NewReader(raw)
	output, err := cmd.Output()
	selectedRebaseRequireNoError(t, err)
	commit := strings.TrimSpace(string(output))
	gitTest(t, root, "update-ref", "HEAD", commit)
	plan, err := selectedCommitRebaseCommands(t, root).PrepareCombinePlan([]string{parent, commit})
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, strings.Join([]string{"parent\n", literalMessage}, "\n\n"), plan.Topology().Description())
}

func TestCapturedCombineRootAndDetachedHead(t *testing.T) {
	root := initHistoricalRepo(t)
	first := capturedCherryPickCommit(t, root, "one", "1\n", "one", capturedCherryPickDate)
	second := capturedCherryPickCommit(t, root, "two", "2\n", "two", capturedCherryPickDate)
	third := capturedCherryPickCommit(t, root, "three", "3\n", "three", capturedCherryPickDate)
	gitTest(t, root, "checkout", "--detach", third)
	plan, err := selectedCommitRebaseCommands(t, root).PrepareCombinePlan([]string{second, first})
	selectedRebaseRequireNoError(t, err)
	assert.Empty(t, plan.HeadRef())
	assert.Equal(t, first, plan.Topology().KeeperOID())
	assert.Equal(t, third, plan.Topology().HeadOID())
}

func TestCapturedCombineRejectsDirtyTrackedAndAcceptsUntracked(t *testing.T) {
	root := initHistoricalRepo(t)
	first := capturedCherryPickCommit(t, root, "one", "1\n", "one", capturedCherryPickDate)
	second := capturedCherryPickCommit(t, root, "two", "2\n", "two", capturedCherryPickDate)
	commands := selectedCommitRebaseCommands(t, root)
	writeTestFile(t, filepath.Join(root, "two"), []byte("dirty\n"), 0o644)
	_, err := commands.PrepareCombinePlan([]string{first, second})
	assert.ErrorIs(t, err, ErrRebaseRequiresClean)
	gitTest(t, root, "checkout", "--", "two")
	writeTestFile(t, filepath.Join(root, "two"), []byte("staged dirty\n"), 0o644)
	gitTest(t, root, "add", "two")
	_, err = commands.PrepareCombinePlan([]string{first, second})
	assert.ErrorIs(t, err, ErrRebaseRequiresClean)
	gitTest(t, root, "reset", "--", "two")
	gitTest(t, root, "checkout", "--", "two")
	writeTestFile(t, filepath.Join(root, "harmless-untracked"), []byte("ok\n"), 0o644)
	_, err = commands.PrepareCombinePlan([]string{first, second})
	assert.NoError(t, err)
}

func TestCapturedCombineCleanStatusUsesCapturedWorktree(t *testing.T) {
	root := initHistoricalRepo(t)
	first := capturedCherryPickCommit(t, root, "one", "1\n", "one", capturedCherryPickDate)
	second := capturedCherryPickCommit(t, root, "two", "2\n", "two", capturedCherryPickDate)
	writeTestFile(t, filepath.Join(root, "two"), []byte("dirty in captured worktree\n"), 0o644)
	osCommand := oscommands.NewDummyOSCommand()
	// Use the real builder: rootedTestBuilder would hide missing explicit cwd.
	commands := NewRebaseCommands(NewGitCommon(common.NewDummyCommon(), nil, osCommand.Cmd, osCommand, MockRepoPaths(root), nil, nil), nil, nil)
	otherRoot := initHistoricalRepo(t)
	t.Chdir(otherRoot)
	_, err := commands.PrepareCombinePlan([]string{first, second})
	assert.ErrorIs(t, err, ErrRebaseRequiresClean)
}

func TestCapturedCombineRejectsSelectionsOutsideOneFirstParentLine(t *testing.T) {
	root := initHistoricalRepo(t)
	base := capturedCherryPickCommit(t, root, "base", "base\n", "base", capturedCherryPickDate)
	mainBranch := gitTest(t, root, "branch", "--show-current")
	left := capturedCherryPickCommit(t, root, "left", "left\n", "left", capturedCherryPickDate)
	gitTest(t, root, "checkout", "-qb", "side", base)
	right := capturedCherryPickCommit(t, root, "right", "right\n", "right", capturedCherryPickDate)
	gitTest(t, root, "checkout", "-q", mainBranch)
	gitTest(t, root, "merge", "--no-ff", "-qm", "merge side", "side")
	merge := gitTest(t, root, "rev-parse", "HEAD")
	commands := selectedCommitRebaseCommands(t, root)
	_, err := commands.PrepareCombinePlan([]string{left, right})
	assert.Error(t, err)
	_, err = commands.PrepareCombinePlan([]string{base, left})
	assert.Error(t, err, "a merge in the captured suffix must be rejected")
	_, err = commands.PrepareCombinePlan([]string{right, merge})
	assert.Error(t, err, "second-parent commits are outside the first-parent suffix")
	assert.NotEmpty(t, merge)
}

func TestCapturedCombineRejectsStaleHeadAndForeignOwner(t *testing.T) {
	root := initHistoricalRepo(t)
	first := capturedCherryPickCommit(t, root, "one", "1\n", "one", capturedCherryPickDate)
	second := capturedCherryPickCommit(t, root, "two", "2\n", "two", capturedCherryPickDate)
	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareCombinePlan([]string{first, second})
	selectedRebaseRequireNoError(t, err)
	foreignRoot := initHistoricalRepo(t)
	assert.ErrorIs(t, selectedCommitRebaseCommands(t, foreignRoot).CheckCombinePlan(plan), ErrBranchActionChanged)
	third := capturedCherryPickCommit(t, root, "three", "3\n", "three", capturedCherryPickDate)
	assert.NotEmpty(t, third)
	assert.ErrorIs(t, commands.CheckCombinePlan(plan), ErrBranchActionChanged)
}

func TestCapturedCombineIgnoredApprovalAndFingerprintInvalidation(t *testing.T) {
	root := initHistoricalRepo(t)
	first := capturedCherryPickCommit(t, root, "tracked", "base\n", "one", capturedCherryPickDate)
	second := capturedCherryPickCommit(t, root, "tracked", "next\n", "two", capturedCherryPickDate)
	gitTest(t, root, "rm", "tracked")
	gitTest(t, root, "commit", "-qm", "three")
	third := gitTest(t, root, "rev-parse", "HEAD")
	ignoreFile := filepath.Join(t.TempDir(), "global-ignore")
	selectedRebaseRequireNoError(t, os.WriteFile(ignoreFile, []byte("tracked\n"), 0o644))
	gitTest(t, root, "config", "core.excludesfile", ignoreFile)
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("ignored content\n"), 0o644)
	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareCombinePlan([]string{first, second})
	selectedRebaseRequireNoError(t, err)
	assert.Contains(t, plan.IgnoredFiles(), "tracked")
	assert.ErrorIs(t, commands.CheckCombinePlan(plan), ErrRebaseIgnoredApprovalRequired)
	approved := plan.WithIgnoredFilesApproved()
	assert.NoError(t, commands.CheckCombinePlan(approved))
	selectedRebaseRequireNoError(t, os.WriteFile(filepath.Join(root, "tracked"), []byte("changed fingerprint\n"), 0o644))
	assert.ErrorIs(t, commands.CheckCombinePlan(approved), ErrBranchActionChanged)
	assert.NotEmpty(t, third)
}
