package git_commands

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertLiteralCommitGuardLeavesRepositoryUntouched(t *testing.T, root string, validate func() error) error {
	t.Helper()
	head := gitTest(t, root, "rev-parse", "HEAD")
	headRef := gitTest(t, root, "symbolic-ref", "-q", "HEAD")
	index := gitTest(t, root, "write-tree")
	status := gitTest(t, root, "status", "--porcelain=v1", "-z")
	err := validate()
	assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, headRef, gitTest(t, root, "symbolic-ref", "-q", "HEAD"))
	assert.Equal(t, index, gitTest(t, root, "write-tree"))
	assert.Equal(t, status, gitTest(t, root, "status", "--porcelain=v1", "-z"))
	return err
}

func TestLiteralCommitAcceptsFullOIDCase(t *testing.T) {
	root, oid, _ := transactionHistory(t)
	commands := branchPlacementTestCommands(t, root)
	for _, target := range []string{oid, strings.ToUpper(oid)} {
		t.Run(fmt.Sprintf("oid %q", target), func(t *testing.T) {
			err := assertLiteralCommitGuardLeavesRepositoryUntouched(t, root, func() error {
				return commands.validateLiteralCommitOID(target)
			})
			assert.NoError(t, err)
		})
	}
}

func TestLiteralCommitRejectsInvalidIdentityAndType(t *testing.T) {
	root, commit, _ := transactionHistory(t)
	blob := gitTest(t, root, "rev-parse", "HEAD:tracked")
	gitTest(t, root, "tag", "-a", "annotated", "-m", "annotated tag", commit)
	tagObject := gitTest(t, root, "rev-parse", "refs/tags/annotated")
	wrongLength := commit[:len(commit)-1]
	bad := []struct {
		name string
		oid  string
	}{
		{"abbreviated", commit[:12]},
		{"revision", "HEAD"},
		{"nul", commit + "\x00"},
		{"wrong length", wrongLength},
		{"nonhex", "g" + commit[1:]},
		{"missing full oid", strings.Repeat("0", len(commit))},
		{"blob", blob},
		{"tag object", tagObject},
	}
	commands := branchPlacementTestCommands(t, root)
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			err := assertLiteralCommitGuardLeavesRepositoryUntouched(t, root, func() error {
				return commands.validateLiteralCommitOID(tc.oid)
			})
			if err == nil {
				t.Fatalf("expected %q to be rejected", tc.name)
			}
		})
	}
}

func TestLiteralCommitUsesActualOIDWhenBranchShadowsIt(t *testing.T) {
	t.Run("SHA-1 native hash branch", func(t *testing.T) {
		root, actual, other := transactionHistory(t)
		gitTest(t, root, "branch", actual, other)
		commands := branchPlacementTestCommands(t, root)
		plan := branchPlacementRequirePlan(t, commands, "literal-target", actual)
		assert.Equal(t, actual, plan.TargetOID())
		if err := commands.ApplyBranchPlacement(plan); err != nil {
			t.Fatalf("apply literal target: %v", err)
		}
		assert.Equal(t, actual, gitTest(t, root, "rev-parse", "refs/heads/literal-target"))
	})

	t.Run("SHA-1 rejects 64 digit branch name as nonnative OID", func(t *testing.T) {
		root, actual, other := transactionHistory(t)
		longName := strings.Repeat("a", 64)
		gitTest(t, root, "branch", longName, other)
		commands := branchPlacementTestCommands(t, root)
		err := assertLiteralCommitGuardLeavesRepositoryUntouched(t, root, func() error {
			return commands.validateLiteralCommitOID(longName)
		})
		assert.Error(t, err)
		assert.Len(t, actual, 40)
	})

	t.Run("SHA-256 rejects 40 digit branch shadow", func(t *testing.T) {
		root := t.TempDir()
		gitTest(t, root, "init", "-q", "--object-format=sha256")
		gitTest(t, root, "config", "user.name", "Test")
		gitTest(t, root, "config", "user.email", "test@example.invalid")
		actual := branchPlacementTestCommit(t, root, "tracked", "one\n", "one")
		other := branchPlacementTestCommit(t, root, "tracked", "two\n", "two")
		shadow := actual[:40]
		gitTest(t, root, "branch", shadow, other)
		commands := branchPlacementTestCommands(t, root)
		err := assertLiteralCommitGuardLeavesRepositoryUntouched(t, root, func() error {
			return commands.validateLiteralCommitOID(shadow)
		})
		assert.Error(t, err)
		assert.Len(t, actual, 64)
	})
}

func TestLiteralCommitBranchPlacementKeepsLiteralTargetUnderShadow(t *testing.T) {
	root, actual, shadowTip := transactionHistory(t)
	gitTest(t, root, "branch", actual, shadowTip)
	commands := branchPlacementTestCommands(t, root)
	plan := branchPlacementRequirePlan(t, commands, "literal-target", actual)
	assert.Equal(t, actual, plan.TargetOID())
	if err := commands.ApplyBranchPlacement(plan); err != nil {
		t.Fatalf("apply prepared literal target: %v", err)
	}
	assert.Equal(t, actual, gitTest(t, root, "rev-parse", "refs/heads/literal-target"))
	assert.Equal(t, shadowTip, gitTest(t, root, "rev-parse", "refs/heads/"+actual))
	assert.NoError(t, exec.Command("git", "-C", root, "show-ref", "--verify", "--quiet", "refs/heads/literal-target").Run())
}

func TestLiteralCommitSHA256AcceptsFullOIDAndApply(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init", "-q", "--object-format=sha256")
	gitTest(t, root, "config", "user.name", "Test")
	gitTest(t, root, "config", "user.email", "test@example.invalid")
	first := branchPlacementTestCommit(t, root, filepath.Join("nested", "file"), "one\n", "one")
	commands := branchPlacementTestCommands(t, root)
	assert.Len(t, first, 64)
	err := assertLiteralCommitGuardLeavesRepositoryUntouched(t, root, func() error {
		return commands.validateLiteralCommitOID(strings.ToUpper(first))
	})
	assert.NoError(t, err)
	plan := branchPlacementRequirePlan(t, commands, "sha256-target", first)
	assert.Equal(t, first, plan.TargetOID())
	if err := commands.ApplyBranchPlacement(plan); err != nil {
		t.Fatalf("apply SHA-256 target: %v", err)
	}
	assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/heads/sha256-target"))
}
