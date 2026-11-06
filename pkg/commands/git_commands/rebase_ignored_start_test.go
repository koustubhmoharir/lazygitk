package git_commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// editFirstCommitRunEnv stops a generic start at its first replayed commit, which is
// the commit being edited. It replaces the lazygit sequence editor, which the unit
// test binary cannot run.
var editFirstCommitRunEnv = []string{"GIT_SEQUENCE_EDITOR=sed -i '0,/^pick /s/^pick /edit /'", "GIT_EDITOR=true"}

func requireIgnoredRebaseRisk(t *testing.T, err error) *IgnoredRebaseRiskError {
	t.Helper()
	var riskErr *IgnoredRebaseRiskError
	if !errors.As(err, &riskErr) {
		t.Fatalf("expected an ignored-file risk, got %v", err)
	}
	assert.ErrorIs(t, err, ErrRebaseIgnoredApprovalRequired)
	return riskErr
}

// setUpReplayedCollision makes a history whose middle commit writes a path that HEAD
// deletes and ignores. The ignored file then holds the user's bytes. This is the
// 09-F1 shape: a rewind to base writes the path again.
func setUpReplayedCollision(t *testing.T) (root, base, middle string) {
	t.Helper()
	root = initHistoricalRepo(t)
	base = selectedCommitRebaseCommit(t, root, "collision", "base contents\n", "base")
	middle = selectedCommitRebaseCommit(t, root, "collision", "historical contents\n", "middle")
	gitTest(t, root, "rm", "-q", "collision")
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("/collision\n"), 0o644)
	gitTest(t, root, "add", ".gitignore")
	gitTest(t, root, "commit", "-qm", "head")
	writeTestFile(t, filepath.Join(root, "collision"), []byte("valuable ignored contents\n"), 0o600)
	return root, base, middle
}

func TestRebaseStartIgnoredRiskReviewWithRealGit(t *testing.T) {
	root, base, _ := setUpReplayedCollision(t)
	head := gitTest(t, root, "rev-parse", "HEAD")
	collision := filepath.Join(root, "collision")
	commands := selectedCommitRebaseCommands(t, root)
	opts := PrepareInteractiveRebaseCommandOpts{baseHashOrRoot: base}

	_, err := commands.PrepareInteractiveRebaseCommand(opts)
	riskErr := requireIgnoredRebaseRisk(t, err)
	assert.Equal(t, []string{"collision"}, riskErr.Paths)
	assert.NotEmpty(t, riskErr.Fingerprint)
	assertFileContents(t, collision, "valuable ignored contents\n")
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"), "a refused start must not move HEAD")

	// Approval covers only its own scope. Once it returns, the same start is refused.
	selectedRebaseRequireNoError(t, commands.ApproveIgnoredRebaseRisks(riskErr.Fingerprint, func() error { return nil }))
	_, err = commands.PrepareInteractiveRebaseCommand(opts)
	requireIgnoredRebaseRisk(t, err)

	err = commands.ApproveIgnoredRebaseRisks(riskErr.Fingerprint, func() error {
		cmd, err := commands.PrepareInteractiveRebaseCommand(opts)
		if err != nil {
			return err
		}
		cmd.AddEnvVars(editFirstCommitRunEnv...)
		return cmd.Run()
	})
	selectedRebaseRequireNoError(t, err)
	// The approved rewind overwrote the ignored bytes, and the rebase stopped on the edit.
	assertFileContents(t, collision, "historical contents\n")
	_, statErr := os.Stat(filepath.Join(root, ".git", "rebase-merge"))
	assert.NoError(t, statErr, "the approved rebase should have started")
	gitTest(t, root, "rebase", "--abort")
}

func TestRebaseStartIgnoredRiskFreshFingerprintAfterChange(t *testing.T) {
	root, base, _ := setUpReplayedCollision(t)
	head := gitTest(t, root, "rev-parse", "HEAD")
	collision := filepath.Join(root, "collision")
	commands := selectedCommitRebaseCommands(t, root)
	opts := PrepareInteractiveRebaseCommandOpts{baseHashOrRoot: base}

	_, err := commands.PrepareInteractiveRebaseCommand(opts)
	riskErr := requireIgnoredRebaseRisk(t, err)
	writeTestFile(t, collision, []byte("changed ignored contents\n"), 0o600)

	err = commands.ApproveIgnoredRebaseRisks(riskErr.Fingerprint, func() error {
		_, err := commands.PrepareInteractiveRebaseCommand(opts)
		return err
	})
	fresh := requireIgnoredRebaseRisk(t, err)
	assert.NotEqual(t, riskErr.Fingerprint, fresh.Fingerprint, "a changed ignored file needs a fresh review")
	assert.Equal(t, []string{"collision"}, fresh.Paths)
	assertFileContents(t, collision, "changed ignored contents\n")
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
}

func TestRebaseStartIgnoredRiskRootReplaysAllCommits(t *testing.T) {
	root := initHistoricalRepo(t)
	selectedCommitRebaseCommit(t, root, "collision", "root contents\n", "root")
	gitTest(t, root, "rm", "-q", "collision")
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("/collision\n"), 0o644)
	gitTest(t, root, "add", ".gitignore")
	gitTest(t, root, "commit", "-qm", "head")
	writeTestFile(t, filepath.Join(root, "collision"), []byte("valuable ignored contents\n"), 0o600)
	commands := selectedCommitRebaseCommands(t, root)

	_, err := commands.PrepareInteractiveRebaseCommand(PrepareInteractiveRebaseCommandOpts{baseHashOrRoot: "--root"})
	riskErr := requireIgnoredRebaseRisk(t, err)
	assert.Equal(t, []string{"collision"}, riskErr.Paths)
	assertFileContents(t, filepath.Join(root, "collision"), "valuable ignored contents\n")
}

func TestAmendToIgnoredRiskBeforeFixupCreation(t *testing.T) {
	root, base, _ := setUpReplayedCollision(t)
	writeTestFile(t, filepath.Join(root, "x"), []byte("middle\n"), 0o644)
	gitTest(t, root, "add", "x")
	gitTest(t, root, "commit", "-qm", "add x")
	middleX := gitTest(t, root, "rev-parse", "HEAD")
	head := gitTest(t, root, "rev-parse", "HEAD")
	writeTestFile(t, filepath.Join(root, "x"), []byte("fix\n"), 0o644)
	gitTest(t, root, "add", "x")
	commands := selectedCommitRebaseCommands(t, root)
	hashPool := &utils.StringPool{}
	commits := []*models.Commit{
		models.NewCommit(hashPool, models.NewCommitOpts{Hash: head}),
		models.NewCommit(hashPool, models.NewCommitOpts{Hash: middleX}),
		models.NewCommit(hashPool, models.NewCommitOpts{Hash: base}),
	}

	err := commands.AmendTo(commits, 1)
	riskErr := requireIgnoredRebaseRisk(t, err)
	assert.Equal(t, []string{"collision"}, riskErr.Paths)
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"), "a refused amend must not create a fixup commit")
	assert.Equal(t, "fix", gitTest(t, root, "show", ":x"), "the staged change must stay staged")

	err = commands.ApproveIgnoredRebaseRisks(riskErr.Fingerprint, func() error {
		return commands.CheckCompoundRebaseIgnoredRisks(commits, 1, true)
	})
	selectedRebaseRequireNoError(t, err)
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, want, string(got))
}

// SquashAllAboveFixupCommits builds its own command, so the review runs in the
// entry point itself. The approved run goes through the lazygit editor, which this
// test binary cannot run; the integration test covers it.
func TestSquashAllAboveIgnoredRiskBeforeRebaseWithRealGit(t *testing.T) {
	root := initHistoricalRepo(t)
	base := selectedCommitRebaseCommit(t, root, "collision", "base contents\n", "base")
	middle := selectedCommitRebaseCommit(t, root, "middle-file", "middle\n", "middle")
	gitTest(t, root, "rm", "-q", "collision")
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("/collision\n"), 0o644)
	gitTest(t, root, "add", ".gitignore")
	gitTest(t, root, "commit", "-qm", "head")
	writeTestFile(t, filepath.Join(root, "collision"), []byte("valuable ignored contents\n"), 0o600)
	selectedCommitRebaseCommit(t, root, "fixup-file", "fixup\n", "fixup! middle")
	head := gitTest(t, root, "rev-parse", "HEAD")
	commands := selectedCommitRebaseCommands(t, root)
	commit := models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{Hash: middle, Parents: []string{base}})

	err := commands.SquashAllAboveFixupCommits(commit)
	riskErr := requireIgnoredRebaseRisk(t, err)
	assert.Equal(t, []string{"collision"}, riskErr.Paths)
	assertFileContents(t, filepath.Join(root, "collision"), "valuable ignored contents\n")
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"), "a refused squash must not move HEAD")

	err = commands.ApproveIgnoredRebaseRisks(riskErr.Fingerprint, func() error {
		return commands.requireIgnoredRebaseSquashAbove(base)
	})
	selectedRebaseRequireNoError(t, err)
	_, err = os.Stat(filepath.Join(root, ".git", "rebase-merge"))
	assert.True(t, os.IsNotExist(err), "the guard must not start a rebase")
}
