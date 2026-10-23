package git_commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchActionMergeReviewAncestryAndReadOnly(t *testing.T) {
	for _, tc := range []struct {
		name       string
		kind       string
		ff, merged bool
	}{
		{name: "fast forward", kind: "ff", ff: true},
		{name: "divergence", kind: "diverge"},
		{name: "already merged", kind: "merged", ff: true, merged: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, base, head := transactionHistory(t)
			switch tc.kind {
			case "ff":
				gitTest(t, root, "branch", "source", head)
				mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
				gitTest(t, root, "checkout", "-q", "master")
			case "diverge":
				gitTest(t, root, "branch", "source", base)
				mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
				gitTest(t, root, "checkout", "-q", "master")
				mergeFixtureCommit(t, root, "master", "other", "other\n", "other")
			case "merged":
				gitTest(t, root, "branch", "source", base)
				gitTest(t, root, "update-ref", "refs/heads/source", head)
			}
			commands := transactionCommands(t, root)
			target, err := commands.PrepareBranchActionTarget("refs/heads/source", gitTest(t, root, "rev-parse", "refs/heads/source"))
			if !assert.NoError(t, err) {
				return
			}
			before := mergePreviewSnapshot(t, root)
			review, err := commands.PrepareBranchActionMergeReview(target, "")
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, tc.ff, review.CanFastForward())
			assert.Equal(t, tc.merged, review.AlreadyMerged())
			assert.Equal(t, before, mergePreviewSnapshot(t, root))
		})
	}
}

func TestBranchActionMergeReviewOptionsPathsAndClonedGetters(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("literal newline path fixture requires Unix")
	}
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	mergeFixtureCommit(t, root, "source", "tab\tname\nline", "source\n", "source")
	gitTest(t, root, "checkout", "-q", "master")
	gitTest(t, root, "config", "branch.master.mergeOptions", "--no-ff --strategy=recursive")
	gitTest(t, root, "config", "--add", "branch.master.mergeOptions", "--ff")
	gitTest(t, root, "config", "pull.twohead", "ort  recursive")
	gitTest(t, root, "config", "merge.ff", "future-value")
	commands := transactionCommands(t, root)
	target, err := commands.PrepareBranchActionTarget("refs/heads/source", gitTest(t, root, "rev-parse", "refs/heads/source"))
	if !assert.NoError(t, err) {
		return
	}
	before := mergePreviewSnapshot(t, root)
	review, err := commands.PrepareBranchActionMergeReview(target, `--log=3 --message="quoted words"`)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, []string{"--ff", "--log=3", "--message=quoted words"}, review.Arguments())
	assert.Equal(t, []string{"ort", "", "recursive"}, review.defaultStrategies)
	assert.Equal(t, "true", review.defaultFF)
	assert.Equal(t, []string(nil), review.StagedFiles())
	args := review.Arguments()
	args[0] = "mutated"
	paths := review.StagedFiles()
	paths = append(paths, "mutated")
	assert.Contains(t, paths, "mutated")
	assert.Equal(t, []string{"--ff", "--log=3", "--message=quoted words"}, review.Arguments())
	assert.Empty(t, review.StagedFiles())
	assert.Equal(t, before, mergePreviewSnapshot(t, root))
}

func TestBranchActionMergeReviewRejectsCapturedStateChanges(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "master")
	commands := transactionCommands(t, root)
	prepare := func() BranchActionMergeReview {
		target, err := commands.PrepareBranchActionTarget("refs/heads/source", gitTest(t, root, "rev-parse", "refs/heads/source"))
		if !assert.NoError(t, err) {
			return BranchActionMergeReview{}
		}
		review, err := commands.PrepareBranchActionMergeReview(target, "")
		if !assert.NoError(t, err) {
			return BranchActionMergeReview{}
		}
		return review
	}
	originalSource := gitTest(t, root, "rev-parse", "refs/heads/source")
	t.Run("source moved", func(t *testing.T) {
		review := prepare()
		gitTest(t, root, "update-ref", "refs/heads/source", base)
		assert.ErrorIs(t, commands.CheckBranchActionMergeReview(review), ErrBranchActionChanged)
	})
	// Restore source and independently capture config and index changes.
	gitTest(t, root, "update-ref", "refs/heads/source", originalSource)
	t.Run("config changed", func(t *testing.T) {
		review := prepare()
		gitTest(t, root, "config", "merge.ff", "false")
		assert.ErrorIs(t, commands.CheckBranchActionMergeReview(review), ErrBranchActionChanged)
		gitTest(t, root, "config", "--unset", "merge.ff")
	})
	t.Run("staged index changed", func(t *testing.T) {
		review := prepare()
		writeTestFile(t, filepath.Join(root, "staged"), []byte("x"), 0o644)
		gitTest(t, root, "add", "staged")
		assert.ErrorIs(t, commands.CheckBranchActionMergeReview(review), ErrBranchActionChanged)
	})
}

func TestBranchActionMergeReviewIdentityAndRepositoryGuards(t *testing.T) {
	t.Run("same head oid different branch", func(t *testing.T) {
		root, base, head := transactionHistory(t)
		gitTest(t, root, "branch", "source", base)
		commands := transactionCommands(t, root)
		target, err := commands.PrepareBranchActionTarget("refs/heads/source", base)
		if !assert.NoError(t, err) {
			return
		}
		review, err := commands.PrepareBranchActionMergeReview(target, "")
		if !assert.NoError(t, err) {
			return
		}
		gitTest(t, root, "branch", "same-tip", head)
		gitTest(t, root, "checkout", "-q", "same-tip")
		assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
		assert.ErrorIs(t, commands.CheckBranchActionMergeReview(review), ErrBranchActionChanged)
	})
	t.Run("foreign backend for same filesystem", func(t *testing.T) {
		root, base, _ := transactionHistory(t)
		gitTest(t, root, "branch", "source", base)
		owner := transactionCommands(t, root)
		target, err := owner.PrepareBranchActionTarget("refs/heads/source", base)
		if !assert.NoError(t, err) {
			return
		}
		review, err := owner.PrepareBranchActionMergeReview(target, "")
		if !assert.NoError(t, err) {
			return
		}
		assert.Error(t, transactionCommands(t, root).CheckBranchActionMergeReview(review))
	})
	t.Run("detached head", func(t *testing.T) {
		root, base, head := transactionHistory(t)
		gitTest(t, root, "branch", "source", base)
		gitTest(t, root, "checkout", "--detach", "-q", head)
		commands := transactionCommands(t, root)
		target, err := commands.PrepareBranchActionTarget("refs/heads/source", base)
		if !assert.NoError(t, err) {
			return
		}
		_, err = commands.PrepareBranchActionMergeReview(target, "")
		assert.Error(t, err)
	})
	t.Run("bare repository", func(t *testing.T) {
		sourceRoot, base, _ := transactionHistory(t)
		bare := filepath.Join(t.TempDir(), "bare.git")
		gitTest(t, filepath.Dir(bare), "init", "--bare", "-q", bare)
		// Transfer the commit object first; a bare ref cannot point into another
		// repository's object database.
		gitTest(t, bare, "fetch", "-q", sourceRoot, base)
		gitTest(t, bare, "update-ref", "refs/heads/source", base)
		commands := transactionCommands(t, bare)
		target, err := commands.PrepareBranchActionTarget("refs/heads/source", base)
		if !assert.NoError(t, err) {
			return
		}
		_, err = commands.PrepareBranchActionMergeReview(target, "")
		assert.Error(t, err)
	})
}

func TestBranchActionMergeReviewRejectsOperationAndUnmergedIndex(t *testing.T) {
	for _, mode := range []string{"merge operation", "rebase operation", "unmerged index"} {
		t.Run(mode, func(t *testing.T) {
			root, base, _ := transactionHistory(t)
			gitTest(t, root, "branch", "source", base)
			commands := transactionCommands(t, root)
			var target BranchActionTarget
			switch mode {
			case "merge operation":
				target, _ = commands.PrepareBranchActionTarget("refs/heads/source", base)
				writeTestFile(t, filepath.Join(root, ".git", "MERGE_HEAD"), []byte(base+"\n"), 0o600)
			case "rebase operation":
				target, _ = commands.PrepareBranchActionTarget("refs/heads/source", base)
				assert.NoError(t, os.Mkdir(filepath.Join(root, ".git", "rebase-merge"), 0o700))
			case "unmerged index":
				gitTest(t, root, "checkout", "-qb", "conflict", base)
				writeTestFile(t, filepath.Join(root, "tracked"), []byte("side\n"), 0o644)
				gitTest(t, root, "commit", "-qam", "side")
				gitTest(t, root, "checkout", "-q", "master")
				writeTestFile(t, filepath.Join(root, "tracked"), []byte("main\n"), 0o644)
				gitTest(t, root, "commit", "-qam", "main")
				cmd := exec.Command("git", "-C", root, "merge", "conflict")
				output, mergeErr := cmd.CombinedOutput()
				if mergeErr == nil {
					t.Fatalf("expected conflict, output %s", output)
				}
				// Remove the operation marker only, leaving the real conflict
				// stages in the index so this exercises the index guard itself.
				assert.NoError(t, os.Remove(filepath.Join(root, ".git", "MERGE_HEAD")))
				var err error
				target, err = commands.PrepareBranchActionTarget("refs/heads/source", base)
				if !assert.NoError(t, err) {
					return
				}
			}
			indexBefore, err := os.ReadFile(filepath.Join(root, ".git", "index"))
			if !assert.NoError(t, err) {
				return
			}
			workBefore, err := os.ReadFile(filepath.Join(root, "tracked"))
			if !assert.NoError(t, err) {
				return
			}
			_, err = commands.PrepareBranchActionMergeReview(target, "")
			if mode == "unmerged index" {
				assert.ErrorContains(t, err, "resolve unmerged index entries")
			} else {
				assert.Error(t, err)
			}
			assert.Equal(t, indexBefore, mustRead(t, filepath.Join(root, ".git", "index")))
			assert.Equal(t, workBefore, mustRead(t, filepath.Join(root, "tracked")))
		})
	}
}

func TestBranchActionMergeReviewConfigAliasesAndLiteralSplitting(t *testing.T) {
	for _, tc := range []struct{ value, want string }{{"false", "false"}, {"no", "false"}, {"only", "only"}, {"future", "true"}, {"yes", "true"}} {
		t.Run("merge.ff "+tc.value, func(t *testing.T) {
			root, base, _ := transactionHistory(t)
			gitTest(t, root, "branch", "source", base)
			gitTest(t, root, "config", "merge.ff", tc.value)
			gitTest(t, root, "config", "pull.twohead", "ort\trecursive  ours\nrecursive")
			commands := transactionCommands(t, root)
			target, err := commands.PrepareBranchActionTarget("refs/heads/source", base)
			if !assert.NoError(t, err) {
				return
			}
			review, err := commands.PrepareBranchActionMergeReview(target, "")
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, tc.want, review.defaultFF)
			assert.Equal(t, []string{"ort\trecursive", "", "ours\nrecursive"}, review.defaultStrategies)
		})
	}
}
