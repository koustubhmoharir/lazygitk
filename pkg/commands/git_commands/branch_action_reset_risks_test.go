package git_commands

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// hardResetRiskFixture reproduces the probe fixture. The target commit tracks
// p.txt, ign.txt, d/x and a file d2. The worktree holds untracked p.txt, d (a
// file), d2/y and u.txt, and ignored ign.txt, z.log and e/f.
func hardResetRiskFixture(t *testing.T) (string, BranchActionTarget, *BranchCommands, string) {
	t.Helper()
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "a.txt"), []byte("a\n"), 0o644)
	gitTest(t, root, "add", "a.txt")
	gitTest(t, root, "commit", "-qm", "current")
	current := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-q", "-b", "target")
	for _, file := range []string{"p.txt", "ign.txt", "d/x", "d2"} {
		writeTestFile(t, filepath.Join(root, file), []byte(file+" committed\n"), 0o644)
	}
	gitTest(t, root, "add", "-A")
	gitTest(t, root, "commit", "-qm", "target")
	targetOID := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "checkout", "-q", current)

	writeTestFile(t, filepath.Join(root, ".git", "info", "exclude"), []byte("ign.txt\nz.log\ne/\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "p.txt"), []byte("untracked p\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "d"), []byte("untracked d\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "d2", "y"), []byte("untracked y\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "u.txt"), []byte("untracked u\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "ign.txt"), []byte("ignored ign\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "z.log"), []byte("ignored z\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "e", "f"), []byte("ignored ef\n"), 0o644)

	commands := transactionCommands(t, root)
	target, err := commands.PrepareBranchActionTarget("refs/heads/target", targetOID)
	resetNoError(t, err)
	return root, target, commands, targetOID
}

var hardResetRiskPaths = []string{"d", "d2/y", "ign.txt", "p.txt"}

func TestBranchActionResetRisksListUntrackedAndIgnoredCollisions(t *testing.T) {
	_, target, commands, _ := hardResetRiskFixture(t)
	paths, fingerprint, err := commands.HardResetIgnoredRisks(target)
	resetNoError(t, err)
	assert.Equal(t, hardResetRiskPaths, paths)
	assert.NotEmpty(t, fingerprint)
}

func TestBranchActionResetRisksFingerprintFollowsContent(t *testing.T) {
	root, target, commands, _ := hardResetRiskFixture(t)
	paths, before, err := commands.HardResetIgnoredRisks(target)
	resetNoError(t, err)
	assert.Equal(t, hardResetRiskPaths, paths)

	writeTestFile(t, filepath.Join(root, "ign.txt"), []byte("changed ignored bytes\n"), 0o644)
	paths, afterIgnored, err := commands.HardResetIgnoredRisks(target)
	resetNoError(t, err)
	assert.Equal(t, hardResetRiskPaths, paths)
	assert.NotEqual(t, before, afterIgnored)

	writeTestFile(t, filepath.Join(root, "p.txt"), []byte("changed untracked bytes\n"), 0o644)
	_, afterUntracked, err := commands.HardResetIgnoredRisks(target)
	resetNoError(t, err)
	assert.NotEqual(t, afterIgnored, afterUntracked)
}

func TestBranchActionResetCommandHardRequiresReviewedRisks(t *testing.T) {
	root, target, commands, targetOID := hardResetRiskFixture(t)
	current := gitTest(t, root, "rev-parse", "HEAD")
	paths, reviewed, err := commands.HardResetIgnoredRisks(target)
	resetNoError(t, err)

	_, err = commands.BranchActionResetCommand(target, "hard", "")
	var riskErr *ResetIgnoredRiskError
	if !errors.As(err, &riskErr) {
		t.Fatalf("unapproved hard reset error = %v, want ResetIgnoredRiskError", err)
	}
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	assert.Equal(t, paths, riskErr.Paths)
	assert.Equal(t, reviewed, riskErr.Fingerprint)

	writeTestFile(t, filepath.Join(root, "ign.txt"), []byte("changed after review\n"), 0o644)
	_, err = commands.BranchActionResetCommand(target, "hard", reviewed)
	assert.ErrorAs(t, err, &riskErr)
	assert.Equal(t, current, gitTest(t, root, "rev-parse", "HEAD"), "a refused reset must not move HEAD")
	assert.Equal(t, "changed after review\n", string(mustRead(t, filepath.Join(root, "ign.txt"))))

	_, fresh, err := commands.HardResetIgnoredRisks(target)
	resetNoError(t, err)
	cmd, err := commands.BranchActionResetCommand(target, "hard", fresh, "GIT_REFLOG_ACTION=[lazygit undo]")
	resetNoError(t, err)
	resetNoError(t, cmd.Run())

	assert.Equal(t, targetOID, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, "p.txt committed\n", string(mustRead(t, filepath.Join(root, "p.txt"))), "approved untracked collision is overwritten")
	assert.Equal(t, "untracked u\n", string(mustRead(t, filepath.Join(root, "u.txt"))), "non-colliding untracked file is kept")
	assert.Equal(t, "ignored z\n", string(mustRead(t, filepath.Join(root, "z.log"))), "non-colliding ignored file is kept")
	assert.True(t, strings.HasPrefix(gitTest(t, root, "reflog", "-1", "--format=%gs"), "[lazygit undo]"), "reset keeps the undo reflog marker")
}

func TestBranchActionTrackedChangesPresent(t *testing.T) {
	root, _, commands, _ := hardResetRiskFixture(t)
	dirty, err := commands.TrackedChangesPresent()
	resetNoError(t, err)
	assert.False(t, dirty, "untracked and ignored files are not tracked changes")

	writeTestFile(t, filepath.Join(root, "a.txt"), []byte("unstaged\n"), 0o644)
	dirty, err = commands.TrackedChangesPresent()
	resetNoError(t, err)
	assert.True(t, dirty)

	gitTest(t, root, "add", "a.txt")
	dirty, err = commands.TrackedChangesPresent()
	resetNoError(t, err)
	assert.True(t, dirty, "staged tracked changes count")
}
