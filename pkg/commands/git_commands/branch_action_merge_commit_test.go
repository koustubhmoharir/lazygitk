package git_commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func prepareSquashCommitFixture(t *testing.T, staged bool) (string, *BranchCommands, BranchActionMergePlan) {
	t.Helper()
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	mergeFixtureCommit(t, root, "source", "incoming", "incoming\n", "incoming")
	gitTest(t, root, "checkout", "-q", "master")
	if staged {
		writeTestFile(t, filepath.Join(root, "already-staged"), []byte("keep me\n"), 0o644)
		gitTest(t, root, "add", "already-staged")
	}
	commands := transactionCommands(t, root)
	plan := prepareMergePlan(t, commands, BranchActionMergeSquashAndCommit)
	if staged {
		plan = plan.WithStagedFilesApproved()
	}
	return root, commands, plan
}

func runNativeSquash(t *testing.T, commands *BranchCommands, plan BranchActionMergePlan) error {
	t.Helper()
	cmd, err := commands.BranchActionMergeCommand(plan)
	if err != nil {
		return err
	}
	return cmd.Run()
}

func TestBranchActionSquashCommitSuccessAndNoOp(t *testing.T) {
	t.Run("commit merged tree without moving source or unrelated refs", func(t *testing.T) {
		root, commands, plan := prepareSquashCommitFixture(t, false)
		gitTest(t, root, "branch", "unrelated", "HEAD")
		source := gitTest(t, root, "rev-parse", "refs/heads/source")
		unrelated := gitTest(t, root, "rev-parse", "refs/heads/unrelated")
		assert.NoError(t, runNativeSquash(t, commands, plan)) // phase two follows successful native phase one
		commitPlan, needed, err := commands.PrepareBranchActionSquashCommit(plan, BranchActionSquashCommitOptions{Message: "reviewed squash"})
		assert.NoError(t, err)
		assert.True(t, needed)
		cmd, err := commands.BranchActionSquashCommitCommand(commitPlan)
		assert.NoError(t, err)
		assert.NoError(t, cmd.Run())
		assert.Equal(t, source, gitTest(t, root, "rev-parse", "refs/heads/source"))
		assert.Equal(t, unrelated, gitTest(t, root, "rev-parse", "refs/heads/unrelated"))
		assert.Equal(t, "incoming\n", string(mustRead(t, filepath.Join(root, "incoming"))))
		assert.Equal(t, "reviewed squash", gitTest(t, root, "show", "-s", "--format=%s", "HEAD"))
		assert.Equal(t, 2, len(strings.Fields(gitTest(t, root, "rev-list", "--parents", "-n", "1", "HEAD"))))
		assert.Equal(t, gitTest(t, root, "write-tree"), gitTest(t, root, "rev-parse", "HEAD^{tree}"))
	})

	t.Run("already merged does not commit existing staged data", func(t *testing.T) {
		root, base, head := transactionHistory(t)
		gitTest(t, root, "branch", "source", base)
		gitTest(t, root, "update-ref", "refs/heads/source", head)
		writeTestFile(t, filepath.Join(root, "already-staged"), []byte("keep staged\n"), 0o644)
		gitTest(t, root, "add", "already-staged")
		commands := transactionCommands(t, root)
		plan := prepareMergePlan(t, commands, BranchActionMergeSquashAndCommit).WithStagedFilesApproved()
		assert.NoError(t, runNativeSquash(t, commands, plan))
		before := gitTest(t, root, "rev-parse", "HEAD")
		_, needed, err := commands.PrepareBranchActionSquashCommit(plan, BranchActionSquashCommitOptions{Message: "must not commit"})
		assert.NoError(t, err)
		assert.False(t, needed)
		assert.Equal(t, before, gitTest(t, root, "rev-parse", "HEAD"))
		assert.NotEmpty(t, gitTest(t, root, "diff", "--cached", "--name-only"))
	})

	t.Run("source has a different ancestor tip and adds nothing beside unrelated staged data", func(t *testing.T) {
		root, base, head := transactionHistory(t)
		gitTest(t, root, "branch", "source", base)
		gitTest(t, root, "checkout", "-q", "master")
		writeTestFile(t, filepath.Join(root, "unrelated-staged"), []byte("keep staged\n"), 0o644)
		gitTest(t, root, "add", "unrelated-staged")
		commands := transactionCommands(t, root)
		plan := prepareMergePlan(t, commands, BranchActionMergeSquashAndCommit).WithStagedFilesApproved()
		assert.NotEqual(t, base, head)
		assert.NoError(t, runNativeSquash(t, commands, plan))
		_, needed, err := commands.PrepareBranchActionSquashCommit(plan, BranchActionSquashCommitOptions{Message: "must not include unrelated staging"})
		assert.NoError(t, err)
		assert.False(t, needed)
		assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, "keep staged\n", string(mustRead(t, filepath.Join(root, "unrelated-staged"))))
		assert.Equal(t, "unrelated-staged", gitTest(t, root, "diff", "--cached", "--name-only"))
	})
}

func TestBranchActionSquashCommitStagedChangesAndValidation(t *testing.T) {
	t.Run("unapproved staged changes require approval", func(t *testing.T) {
		root, commands, plan := prepareSquashCommitFixture(t, true)
		_, err := commands.BranchActionMergeCommand(plan.WithStagedFilesApproved())
		assert.NoError(t, err)
		// A separately prepared plan without approval is rejected at phase one.
		unapproved := plan
		unapproved.stagedApproved = false
		_, err = commands.BranchActionMergeCommand(unapproved)
		assert.ErrorIs(t, err, ErrMergeStagedApprovalRequired)
		assert.FileExists(t, filepath.Join(root, "already-staged"))
	})

	t.Run("changed source head config index and foreign backend are rejected", func(t *testing.T) {
		for _, change := range []string{"source", "head", "config", "backend"} {
			t.Run(change, func(t *testing.T) {
				root, commands, plan := prepareSquashCommitFixture(t, false)
				assert.NoError(t, runNativeSquash(t, commands, plan))
				switch change {
				case "source":
					gitTest(t, root, "update-ref", "refs/heads/source", plan.review.target.HeadOID())
				case "head":
					gitTest(t, root, "branch", "other-head")
					gitTest(t, root, "checkout", "-q", "other-head")
				case "config":
					gitTest(t, root, "config", "merge.ff", "false")
				case "backend":
					commands = transactionCommands(t, root)
				}
				commitPlan, needed, err := commands.PrepareBranchActionSquashCommit(plan, BranchActionSquashCommitOptions{Message: "reviewed"})
				assert.Error(t, err)
				assert.False(t, needed)
				assert.False(t, commitPlan.prepared)
			})
		}
	})

	t.Run("post-prepare index change prevents queued commit", func(t *testing.T) {
		root, commands, plan := prepareSquashCommitFixture(t, false)
		assert.NoError(t, runNativeSquash(t, commands, plan))
		commitPlan, needed, err := commands.PrepareBranchActionSquashCommit(plan, BranchActionSquashCommitOptions{Message: "reviewed"})
		assert.NoError(t, err)
		assert.True(t, needed)
		before := gitTest(t, root, "rev-parse", "HEAD")
		writeTestFile(t, filepath.Join(root, "index-change"), []byte("staged\n"), 0o644)
		gitTest(t, root, "add", "index-change")
		_, err = commands.BranchActionSquashCommitCommand(commitPlan)
		assert.Error(t, err)
		assert.Equal(t, before, gitTest(t, root, "rev-parse", "HEAD"))
	})

	t.Run("post-prepare source ref current branch identity and backend changes prevent queued commit", func(t *testing.T) {
		for _, change := range []string{"source ref", "same oid different HEAD", "backend"} {
			t.Run(change, func(t *testing.T) {
				root, commands, plan := prepareSquashCommitFixture(t, false)
				assert.NoError(t, runNativeSquash(t, commands, plan))
				commitPlan, needed, err := commands.PrepareBranchActionSquashCommit(plan, BranchActionSquashCommitOptions{Message: "reviewed"})
				assert.NoError(t, err)
				assert.True(t, needed)
				switch change {
				case "source ref":
					gitTest(t, root, "update-ref", "refs/heads/source", plan.review.target.HeadOID())
				case "same oid different HEAD":
					gitTest(t, root, "branch", "same-oid")
					gitTest(t, root, "checkout", "-q", "same-oid")
				case "backend":
					commands = transactionCommands(t, root)
				}
				_, err = commands.BranchActionSquashCommitCommand(commitPlan)
				assert.Error(t, err)
			})
		}
	})

	t.Run("messages must be nonempty and NUL free", func(t *testing.T) {
		for _, message := range []string{"", " \n\t", "bad\x00message"} {
			root, commands, plan := prepareSquashCommitFixture(t, false)
			assert.NoError(t, runNativeSquash(t, commands, plan))
			_, needed, err := commands.PrepareBranchActionSquashCommit(plan, BranchActionSquashCommitOptions{Message: message})
			assert.Error(t, err)
			assert.False(t, needed)
			assert.NotEmpty(t, gitTest(t, root, "diff", "--cached", "--name-only"))
		}
	})
}

func TestBranchActionSquashCommitConflictAndNativeFailure(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	gitTest(t, root, "branch", "current", base)
	mergeFixtureCommit(t, root, "source", "shared", "source\n", "source")
	mergeFixtureCommit(t, root, "current", "shared", "current\n", "current")
	gitTest(t, root, "checkout", "-q", "current")
	commands := transactionCommands(t, root)
	plan := prepareMergePlan(t, commands, BranchActionMergeSquashAndCommit)
	raw := []byte("precious\x00raw\n")
	if err := os.WriteFile(filepath.Join(root, "untracked-preserve"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	beforeHead := gitTest(t, root, "rev-parse", "HEAD")
	assert.Error(t, runNativeSquash(t, commands, plan))
	assert.NotEmpty(t, gitTest(t, root, "ls-files", "-u"))
	assert.Equal(t, raw, mustRead(t, filepath.Join(root, "untracked-preserve")))
	_, needed, err := commands.PrepareBranchActionSquashCommit(plan, BranchActionSquashCommitOptions{Message: "never"})
	assert.Error(t, err)
	assert.False(t, needed)
	assert.Equal(t, beforeHead, gitTest(t, root, "rev-parse", "HEAD"))
}

func TestBranchActionSquashCommitCommandCapturesOptionsAndNativeFailureState(t *testing.T) {
	t.Run("message signoff hooks and review arguments", func(t *testing.T) {
		root, commands, _ := prepareSquashCommitFixture(t, false)
		gitTest(t, root, "config", "commit.gpgSign", "false")
		// Review-only arguments are parsed and retained from the captured merge options.
		target, err := commands.PrepareBranchActionTarget("refs/heads/source", gitTest(t, root, "rev-parse", "refs/heads/source"))
		assert.NoError(t, err)
		review, err := commands.PrepareBranchActionMergeReview(target, "--gpg-sign=KEY --cleanup=verbatim --edit --no-cleanup")
		assert.NoError(t, err)
		plan, err := commands.PrepareBranchActionMerge(review, BranchActionMergeSquashAndCommit, "")
		assert.NoError(t, err)
		assert.NoError(t, runNativeSquash(t, commands, plan))
		commitPlan, needed, err := commands.PrepareBranchActionSquashCommit(plan, BranchActionSquashCommitOptions{Message: "captured msg", SignOff: true, SkipHooks: true})
		assert.NoError(t, err)
		assert.True(t, needed)
		cmd, err := commands.BranchActionSquashCommitCommand(commitPlan)
		assert.NoError(t, err)
		argv := strings.Join(cmd.GetCmd().Args, " ")
		for _, want := range []string{"--message=captured msg", "--signoff", "--no-verify", "--gpg-sign=KEY", "--cleanup=verbatim", "--edit", "--no-cleanup"} {
			assert.Contains(t, argv, want)
		}
		assert.NotContains(t, argv, "--no-gpg-sign")
	})

	t.Run("explicit no signing and no edit are preserved", func(t *testing.T) {
		root, commands, _ := prepareSquashCommitFixture(t, false)
		target, err := commands.PrepareBranchActionTarget("refs/heads/source", gitTest(t, root, "rev-parse", "refs/heads/source"))
		assert.NoError(t, err)
		review, err := commands.PrepareBranchActionMergeReview(target, "--no-gpg-sign --no-edit")
		assert.NoError(t, err)
		plan, err := commands.PrepareBranchActionMerge(review, BranchActionMergeSquashAndCommit, "")
		assert.NoError(t, err)
		assert.NoError(t, runNativeSquash(t, commands, plan))
		commitPlan, needed, err := commands.PrepareBranchActionSquashCommit(plan, BranchActionSquashCommitOptions{Message: "unsigned"})
		assert.NoError(t, err)
		assert.True(t, needed)
		cmd, err := commands.BranchActionSquashCommitCommand(commitPlan)
		assert.NoError(t, err)
		argv := strings.Join(cmd.GetCmd().Args, " ")
		assert.Contains(t, argv, "--no-gpg-sign")
		assert.Contains(t, argv, "--no-edit")
	})

	t.Run("commit hook failure preserves actual partial state and is not retried", func(t *testing.T) {
		root, commands, _ := prepareSquashCommitFixture(t, false)
		hook := filepath.Join(root, "reject-commit")
		logPath := filepath.Join(root, "hook-log")
		writeTestFile(t, hook, []byte("#!/bin/sh\necho hook-ran >> '"+logPath+"'\nexit 1\n"), 0o755)
		gitTest(t, root, "config", "core.hooksPath", root)
		if err := os.Rename(hook, filepath.Join(root, "pre-commit")); err != nil {
			t.Fatal(err)
		}
		plan := prepareMergePlan(t, commands, BranchActionMergeSquashAndCommit)
		assert.NoError(t, runNativeSquash(t, commands, plan))
		commitPlan, needed, err := commands.PrepareBranchActionSquashCommit(plan, BranchActionSquashCommitOptions{Message: "fails"})
		assert.NoError(t, err)
		assert.True(t, needed)
		cmd, err := commands.BranchActionSquashCommitCommand(commitPlan)
		assert.NoError(t, err)
		before := gitTest(t, root, "rev-parse", "HEAD")
		assert.Error(t, cmd.Run())
		assert.Equal(t, before, gitTest(t, root, "rev-parse", "HEAD"))
		assert.NotEmpty(t, gitTest(t, root, "diff", "--cached", "--name-only"))
		assert.Equal(t, "hook-ran\n", string(mustRead(t, logPath)))
	})

	t.Run("signing failure runs once and preserves staged index and HEAD", func(t *testing.T) {
		root, commands, _ := prepareSquashCommitFixture(t, false)
		signer := filepath.Join(root, "failing-signer")
		logPath := filepath.Join(root, "signer-log")
		writeTestFile(t, signer, []byte("#!/bin/sh\necho called >> '"+logPath+"'\nexit 1\n"), 0o755)
		gitTest(t, root, "config", "gpg.program", signer)
		gitTest(t, root, "config", "commit.gpgSign", "true")
		plan := prepareMergePlan(t, commands, BranchActionMergeSquashAndCommit)
		assert.NoError(t, runNativeSquash(t, commands, plan))
		commitPlan, needed, err := commands.PrepareBranchActionSquashCommit(plan, BranchActionSquashCommitOptions{Message: "signed"})
		assert.NoError(t, err)
		assert.True(t, needed)
		cmd, err := commands.BranchActionSquashCommitCommand(commitPlan)
		assert.NoError(t, err)
		beforeHead := gitTest(t, root, "rev-parse", "HEAD")
		beforeIndex := gitTest(t, root, "write-tree")
		assert.Error(t, cmd.Run())
		assert.Equal(t, beforeHead, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, beforeIndex, gitTest(t, root, "write-tree"))
		assert.Equal(t, "called\n", string(mustRead(t, logPath)))
	})
}

func TestBranchActionMergeAbortGuard(t *testing.T) {
	root, base, _ := transactionHistory(t)
	gitTest(t, root, "branch", "source", base)
	gitTest(t, root, "branch", "current", base)
	mergeFixtureCommit(t, root, "source", "shared", "source\n", "source")
	mergeFixtureCommit(t, root, "current", "shared", "current\n", "current")
	gitTest(t, root, "checkout", "-q", "current")
	commands := transactionCommands(t, root)
	plan := prepareMergePlan(t, commands, BranchActionMergeNoFF)
	assert.Error(t, runNativeSquash(t, commands, plan))
	assert.NoError(t, commands.CheckBranchActionMergeInProgress(plan))

	t.Run("foreign backend rejected", func(t *testing.T) {
		assert.Error(t, transactionCommands(t, root).CheckBranchActionMergeInProgress(plan))
	})
	t.Run("replaced merge metadata rejected", func(t *testing.T) {
		for _, name := range []string{"MERGE_HEAD", "ORIG_HEAD"} {
			path := filepath.Join(root, ".git", name)
			contents := mustRead(t, path)
			replacement := plan.review.target.SourceOID()
			if name == "MERGE_HEAD" {
				replacement = plan.review.target.HeadOID()
			}
			assert.NoError(t, os.WriteFile(path, []byte(replacement+"\n"), 0o644))
			assert.Error(t, commands.CheckBranchActionMergeInProgress(plan), name)
			assert.NoError(t, os.WriteFile(path, contents, 0o644))
		}
	})
	t.Run("same oid on another checked out branch rejected", func(t *testing.T) {
		gitTest(t, root, "branch", "same-oid")
		gitTest(t, root, "symbolic-ref", "HEAD", "refs/heads/same-oid")
		assert.Error(t, commands.CheckBranchActionMergeInProgress(plan))
		gitTest(t, root, "symbolic-ref", "HEAD", "refs/heads/current")
	})
	t.Run("squash has no abort metadata and is rejected", func(t *testing.T) {
		separate, base, _ := transactionHistory(t)
		gitTest(t, separate, "branch", "source", base)
		mergeFixtureCommit(t, separate, "source", "incoming", "incoming\n", "incoming")
		gitTest(t, separate, "checkout", "-q", "master")
		separateCommands := transactionCommands(t, separate)
		squash := prepareMergePlan(t, separateCommands, BranchActionMergeSquash)
		assert.NoError(t, runNativeSquash(t, separateCommands, squash))
		assert.Error(t, separateCommands.CheckBranchActionMergeInProgress(squash))
	})
}
