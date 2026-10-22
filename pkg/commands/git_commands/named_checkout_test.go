package git_commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func namedCheckoutMustNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func namedCheckoutDirtyHistory(t *testing.T) (root, first string) {
	t.Helper()
	root = initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("one\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "carry"), []byte("base\n"), 0o644)
	gitTest(t, root, "add", "tracked", "carry")
	gitTest(t, root, "commit", "-qm", "first")
	first = gitTest(t, root, "rev-parse", "HEAD")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("two\n"), 0o644)
	gitTest(t, root, "commit", "-qam", "second")
	return root, first
}

type namedCheckoutCapturingBuilder struct {
	inner        oscommands.ICmdObjBuilder
	checkoutCmd  *oscommands.CmdObj
	checkoutCall int
}

func (builder *namedCheckoutCapturingBuilder) New(args []string) *oscommands.CmdObj {
	cmd := builder.inner.New(args)
	if len(args) > 1 && args[1] == "checkout" {
		builder.checkoutCall++
		builder.checkoutCmd = cmd
	}
	return cmd
}

func (builder *namedCheckoutCapturingBuilder) NewShell(command, functionsFile string) *oscommands.CmdObj {
	return builder.inner.NewShell(command, functionsFile)
}

func (builder *namedCheckoutCapturingBuilder) Quote(value string) string {
	return builder.inner.Quote(value)
}

func namedCheckoutCommandsWithCapture(t *testing.T, root string) (*BranchCommands, *namedCheckoutCapturingBuilder) {
	t.Helper()
	osCommand := oscommands.NewDummyOSCommand()
	builder := &namedCheckoutCapturingBuilder{inner: osCommand.Cmd}
	commands := NewBranchCommands(NewGitCommon(common.NewDummyCommon(), nil, builder, osCommand, MockRepoPaths(root), nil, nil))
	return commands, builder
}

func TestNamedCheckoutCreatesAndMovesBranch(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "create", true: "move"}[existing], func(t *testing.T) {
			root, first, second := transactionHistory(t)
			if existing {
				gitTest(t, root, "branch", "topic", second)
				gitTest(t, root, "config", "branch.topic.remote", "origin")
				gitTest(t, root, "config", "branch.topic.merge", "refs/heads/upstream-topic")
			}
			commands := transactionCommands(t, root)
			plan, err := commands.PrepareNamedCheckout("topic", first)
			namedCheckoutMustNoError(t, err)
			assert.Equal(t, "topic", plan.Name())
			assert.Equal(t, first, plan.TargetOID())
			assert.Equal(t, existing, plan.DestinationExists())
			assert.Equal(t, existing, plan.MovesBranch())
			assert.Equal(t, gitTest(t, root, "symbolic-ref", "--no-recurse", "HEAD"), plan.HeadRef())
			namedCheckoutMustNoError(t, commands.ApplyNamedCheckout(plan))
			assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/heads/topic"))
			assert.Equal(t, first, gitTest(t, root, "rev-parse", "HEAD"))
			assert.Equal(t, "refs/heads/topic", gitTest(t, root, "symbolic-ref", "--no-recurse", "HEAD"))
			if existing {
				assert.Equal(t, "origin", gitTest(t, root, "config", "branch.topic.remote"))
				assert.Equal(t, "refs/heads/upstream-topic", gitTest(t, root, "config", "branch.topic.merge"))
			}
		})
	}
}

func TestNamedCheckoutCarriesSafeDirtyChangesAndRejectsOverlap(t *testing.T) {
	t.Run("carries unrelated staged unstaged and untracked changes", func(t *testing.T) {
		root, first := namedCheckoutDirtyHistory(t)
		writeTestFile(t, filepath.Join(root, "carry"), []byte("staged carry\n"), 0o644)
		gitTest(t, root, "add", "carry")
		writeTestFile(t, filepath.Join(root, "carry"), []byte("unstaged carry\n"), 0o644)
		writeTestFile(t, filepath.Join(root, "untracked"), []byte("untracked\n"), 0o644)
		plan, err := transactionCommands(t, root).PrepareNamedCheckout("topic", first)
		namedCheckoutMustNoError(t, err)
		namedCheckoutMustNoError(t, transactionCommands(t, root).ApplyNamedCheckout(plan))
		assert.Contains(t, gitTest(t, root, "diff", "--cached", "--name-only"), "carry")
		assert.Equal(t, "staged carry", gitTest(t, root, "show", ":carry"))
		contents, readErr := os.ReadFile(filepath.Join(root, "carry"))
		namedCheckoutMustNoError(t, readErr)
		assert.Equal(t, "unstaged carry\n", string(contents))
		assert.Equal(t, "one", gitTest(t, root, "show", "HEAD:tracked"))
		assert.FileExists(t, filepath.Join(root, "untracked"))
	})

	t.Run("rejects staged tracked collision without moving branch", func(t *testing.T) {
		root, first, second := transactionHistory(t)
		gitTest(t, root, "branch", "topic", second)
		writeTestFile(t, filepath.Join(root, "tracked"), []byte("staged replacement\n"), 0o644)
		gitTest(t, root, "add", "tracked")
		beforeIndex := gitTest(t, root, "write-tree")
		beforeHead := gitTest(t, root, "rev-parse", "HEAD")
		beforeWorking, readErr := os.ReadFile(filepath.Join(root, "tracked"))
		namedCheckoutMustNoError(t, readErr)
		commands := transactionCommands(t, root)
		plan, err := commands.PrepareNamedCheckout("topic", first)
		namedCheckoutMustNoError(t, err)
		err = commands.ApplyNamedCheckout(plan)
		var checkoutErr *NamedCheckoutError
		if !assert.ErrorAs(t, err, &checkoutErr) {
			t.FailNow()
		}
		assert.Contains(t, err.Error(), "may already have changed")
		assert.Equal(t, second, gitTest(t, root, "rev-parse", "refs/heads/topic"))
		assert.Equal(t, beforeHead, gitTest(t, root, "rev-parse", "HEAD"))
		assert.Equal(t, beforeIndex, gitTest(t, root, "write-tree"))
		afterWorking, readErr := os.ReadFile(filepath.Join(root, "tracked"))
		namedCheckoutMustNoError(t, readErr)
		assert.Equal(t, beforeWorking, afterWorking)
	})

	t.Run("rejects unstaged tracked collision without moving branch", func(t *testing.T) {
		root, first, second := transactionHistory(t)
		gitTest(t, root, "branch", "topic", second)
		writeTestFile(t, filepath.Join(root, "tracked"), []byte("unstaged replacement\n"), 0o644)
		commands := transactionCommands(t, root)
		plan, err := commands.PrepareNamedCheckout("topic", first)
		namedCheckoutMustNoError(t, err)
		err = commands.ApplyNamedCheckout(plan)
		var checkoutErr *NamedCheckoutError
		if !assert.ErrorAs(t, err, &checkoutErr) {
			t.FailNow()
		}
		assert.Equal(t, second, gitTest(t, root, "rev-parse", "refs/heads/topic"))
	})

	for _, ignored := range []bool{false, true} {
		t.Run(map[bool]string{false: "untracked collision", true: "ignored collision"}[ignored], func(t *testing.T) {
			root, _, second := transactionHistory(t)
			writeTestFile(t, filepath.Join(root, "collision"), []byte("target\n"), 0o644)
			gitTest(t, root, "add", "collision")
			gitTest(t, root, "commit", "-qm", "add target collision")
			target := gitTest(t, root, "rev-parse", "HEAD")
			gitTest(t, root, "rm", "-q", "collision")
			if ignored {
				writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("collision\n"), 0o644)
				gitTest(t, root, "add", ".gitignore")
			}
			gitTest(t, root, "commit", "-qm", "remove collision")
			writeTestFile(t, filepath.Join(root, "collision"), []byte("keep me\n"), 0o644)
			gitTest(t, root, "branch", "topic", second)
			commands := transactionCommands(t, root)
			plan, err := commands.PrepareNamedCheckout("topic", target)
			namedCheckoutMustNoError(t, err)
			err = commands.ApplyNamedCheckout(plan)
			var checkoutErr *NamedCheckoutError
			if !assert.ErrorAs(t, err, &checkoutErr) {
				t.FailNow()
			}
			assert.Equal(t, second, gitTest(t, root, "rev-parse", "refs/heads/topic"))
			contents, readErr := os.ReadFile(filepath.Join(root, "collision"))
			namedCheckoutMustNoError(t, readErr)
			assert.Equal(t, "keep me\n", string(contents))
		})
	}
}

func TestNamedCheckoutSameTipStillChecksOut(t *testing.T) {
	root, first, _ := transactionHistory(t)
	gitTest(t, root, "branch", "topic", first)
	commands := transactionCommands(t, root)
	plan, err := commands.PrepareNamedCheckout("topic", first)
	namedCheckoutMustNoError(t, err)
	assert.True(t, plan.DestinationExists())
	assert.False(t, plan.MovesBranch())
	namedCheckoutMustNoError(t, commands.ApplyNamedCheckout(plan))
	assert.Equal(t, "refs/heads/topic", gitTest(t, root, "symbolic-ref", "--no-recurse", "HEAD"))
}

func TestNamedCheckoutPeelsFullOIDWhenBranchHasOIDName(t *testing.T) {
	root, target, other := transactionHistory(t)
	gitTest(t, root, "branch", target, other)
	commands := transactionCommands(t, root)
	plan, err := commands.PrepareNamedCheckout(target, target)
	namedCheckoutMustNoError(t, err)
	namedCheckoutMustNoError(t, commands.ApplyNamedCheckout(plan))
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, target, gitTest(t, root, "rev-parse", "refs/heads/"+target))
}

func TestNamedCheckoutRunsExpectedGitArgv(t *testing.T) {
	root, target, _ := transactionHistory(t)
	commands, builder := namedCheckoutCommandsWithCapture(t, root)
	plan, err := commands.PrepareNamedCheckout("topic", target)
	namedCheckoutMustNoError(t, err)
	namedCheckoutMustNoError(t, commands.ApplyNamedCheckout(plan))
	assert.Equal(t, 1, builder.checkoutCall)
	if !assert.NotNil(t, builder.checkoutCmd) {
		t.FailNow()
	}
	assert.True(t, builder.checkoutCmd.LockRetriesDisabled())
	assert.Equal(t, []string{"git", "checkout", "--no-overwrite-ignore", "--no-guess", "-B", "topic", target + "^{commit}"}, builder.checkoutCmd.Args())
}

func TestNamedCheckoutAllowsCurrentBranchButFindsDuplicateReservation(t *testing.T) {
	root, first, _ := transactionHistory(t)
	commands := transactionCommands(t, root)
	currentRef := gitTest(t, root, "symbolic-ref", "--no-recurse", "HEAD")
	currentName := strings.TrimPrefix(currentRef, "refs/heads/")
	plan, err := commands.PrepareNamedCheckout(currentName, first)
	namedCheckoutMustNoError(t, err)

	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root, "worktree", "add", "-q", "--detach", linked, first)
	linkedGitDir := gitTest(t, linked, "rev-parse", "--path-format=absolute", "--absolute-git-dir")
	commonDir := filepath.Join(root, ".git")
	metadata := filepath.Join(commonDir, "worktrees", filepath.Base(linkedGitDir))
	if _, statErr := os.Stat(metadata); statErr != nil {
		entries, readErr := os.ReadDir(filepath.Join(commonDir, "worktrees"))
		namedCheckoutMustNoError(t, readErr)
		if !assert.Len(t, entries, 1) {
			t.FailNow()
		}
		metadata = filepath.Join(commonDir, "worktrees", entries[0].Name())
	}
	namedCheckoutMustNoError(t, os.WriteFile(filepath.Join(metadata, "HEAD"), []byte("ref: "+currentRef), 0o600))
	_, err = commands.PrepareNamedCheckout(currentName, first)
	assert.Error(t, err, "allowing this worktree must not mask another worktree reservation")
	applyErr := commands.ApplyNamedCheckout(plan)
	assert.Error(t, applyErr)
	assert.Contains(t, applyErr.Error(), "reserved by another worktree")
}

func TestNamedCheckoutRejectsChangedSnapshotAndOperation(t *testing.T) {
	t.Run("head changes before apply", func(t *testing.T) {
		root, first, second := transactionHistory(t)
		commands := transactionCommands(t, root)
		plan, err := commands.PrepareNamedCheckout("topic", first)
		namedCheckoutMustNoError(t, err)
		gitTest(t, root, "checkout", "--detach", second)
		assert.ErrorIs(t, commands.ApplyNamedCheckout(plan), ErrNamedCheckoutChanged)
		assert.Equal(t, second, gitTest(t, root, "rev-parse", "HEAD"))
	})
	t.Run("head ref changes without changing its oid", func(t *testing.T) {
		root, first, second := transactionHistory(t)
		commands := transactionCommands(t, root)
		plan, err := commands.PrepareNamedCheckout("topic", first)
		namedCheckoutMustNoError(t, err)
		gitTest(t, root, "branch", "same-tip", second)
		gitTest(t, root, "symbolic-ref", "HEAD", "refs/heads/same-tip")
		assert.Equal(t, second, gitTest(t, root, "rev-parse", "HEAD"))
		assert.ErrorIs(t, commands.ApplyNamedCheckout(plan), ErrNamedCheckoutChanged)
	})

	for _, change := range []string{"create", "move", "delete"} {
		t.Run("destination "+change, func(t *testing.T) {
			root, first, second := transactionHistory(t)
			commands := transactionCommands(t, root)
			if change != "create" {
				gitTest(t, root, "branch", "topic", second)
			}
			plan, err := commands.PrepareNamedCheckout("topic", first)
			namedCheckoutMustNoError(t, err)
			switch change {
			case "create":
				gitTest(t, root, "branch", "topic", second)
			case "move":
				gitTest(t, root, "update-ref", "refs/heads/topic", first, second)
			case "delete":
				gitTest(t, root, "branch", "-D", "topic")
			}
			assert.ErrorIs(t, commands.ApplyNamedCheckout(plan), ErrNamedCheckoutChanged)
		})
	}

	t.Run("active operation marker", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		commands := transactionCommands(t, root)
		plan, err := commands.PrepareNamedCheckout("topic", first)
		namedCheckoutMustNoError(t, err)
		gitDir := filepath.Join(root, ".git")
		namedCheckoutMustNoError(t, os.WriteFile(filepath.Join(gitDir, "MERGE_HEAD"), []byte(first+"\n"), 0o600))
		err = commands.ApplyNamedCheckout(plan)
		assert.Error(t, err)
		assert.False(t, errors.Is(err, ErrNamedCheckoutChanged))
		assert.Contains(t, err.Error(), "finish or abort")
	})

	for _, marker := range []string{"rebase-merge", "rebase-apply", "MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "sequencer", "BISECT_START"} {
		t.Run("active marker "+marker, func(t *testing.T) {
			root, first, _ := transactionHistory(t)
			path := filepath.Join(root, ".git", marker)
			if strings.Contains(marker, "rebase") || marker == "sequencer" {
				namedCheckoutMustNoError(t, os.Mkdir(path, 0o755))
			} else {
				namedCheckoutMustNoError(t, os.WriteFile(path, []byte(first+"\n"), 0o600))
			}
			_, err := transactionCommands(t, root).PrepareNamedCheckout("topic", first)
			assert.Error(t, err)
		})
	}
}

func TestNamedCheckoutDetachedAndUnbornHead(t *testing.T) {
	t.Run("detached", func(t *testing.T) {
		root, first, second := transactionHistory(t)
		gitTest(t, root, "checkout", "--detach", second)
		plan, err := transactionCommands(t, root).PrepareNamedCheckout("topic", first)
		namedCheckoutMustNoError(t, err)
		assert.Empty(t, plan.HeadRef())
		assert.Equal(t, second, plan.HeadOID())
	})
	t.Run("unborn", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		gitTest(t, root, "checkout", "--detach", first)
		gitTest(t, root, "symbolic-ref", "HEAD", "refs/heads/unborn")
		commands := transactionCommands(t, root)
		plan, err := commands.PrepareNamedCheckout("topic", first)
		namedCheckoutMustNoError(t, err)
		assert.Equal(t, "refs/heads/unborn", plan.HeadRef())
		assert.Empty(t, plan.HeadOID())
		if !assert.NoError(t, commands.ApplyNamedCheckout(plan)) {
			t.FailNow()
		}
		assert.Equal(t, first, gitTest(t, root, "rev-parse", "HEAD"))
	})
}

func TestNamedCheckoutPostCheckoutHookErrorPreservesActualFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Git shell hooks require a POSIX shell")
	}
	root, first, _ := transactionHistory(t)
	hooks := filepath.Join(root, ".git", "hooks")
	namedCheckoutMustNoError(t, os.MkdirAll(hooks, 0o755))
	hook := filepath.Join(hooks, "post-checkout")
	namedCheckoutMustNoError(t, os.WriteFile(hook, []byte("#!/bin/sh\necho hook-failed >&2\nexit 23\n"), 0o755))
	commands := transactionCommands(t, root)
	plan, err := commands.PrepareNamedCheckout("topic", first)
	namedCheckoutMustNoError(t, err)
	err = commands.ApplyNamedCheckout(plan)
	var checkoutErr *NamedCheckoutError
	if !assert.ErrorAs(t, err, &checkoutErr) {
		t.FailNow()
	}
	assert.ErrorContains(t, err, "hook-failed")
	assert.ErrorIs(t, err, checkoutErr.Cause)
	assert.Equal(t, first, gitTest(t, root, "rev-parse", "HEAD"), "Git completed checkout before reporting the hook failure")
}

func TestNamedCheckoutRejectsInvalidAndUnpreparedPlans(t *testing.T) {
	root, first, _ := transactionHistory(t)
	commands := transactionCommands(t, root)
	for _, name := range []string{"", "-option", "@{-1}", "bad..name"} {
		_, err := commands.PrepareNamedCheckout(name, first)
		assert.Error(t, err, "name %q", name)
	}
	_, err := commands.PrepareNamedCheckout("topic", "HEAD")
	assert.Error(t, err)
	assert.Error(t, commands.ApplyNamedCheckout(NamedCheckoutPlan{name: "topic", targetOID: first}))
}

func TestNamedCheckoutRejectsSymbolicDestination(t *testing.T) {
	root, first, _ := transactionHistory(t)
	gitTest(t, root, "update-ref", "refs/heads/alias", first)
	currentRef := gitTest(t, root, "symbolic-ref", "--no-recurse", "HEAD")
	gitTest(t, root, "symbolic-ref", "refs/heads/alias", currentRef)
	_, err := transactionCommands(t, root).PrepareNamedCheckout("alias", first)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symbolic alias")
}

func TestNamedCheckoutInLinkedWorktreeOfBareCommonRepository(t *testing.T) {
	root, first, _ := transactionHistory(t)
	bare := filepath.Join(t.TempDir(), "bare.git")
	gitTest(t, root, "clone", "-q", "--bare", root, bare)
	linked := filepath.Join(t.TempDir(), "linked")
	currentName := strings.TrimPrefix(gitTest(t, root, "symbolic-ref", "--no-recurse", "HEAD"), "refs/heads/")
	gitTest(t, bare, "worktree", "add", "-q", linked, currentName)
	commands := realBranchPlacementCommands(t, linked)
	plan, err := commands.PrepareNamedCheckout("topic", first)
	namedCheckoutMustNoError(t, err)
	namedCheckoutMustNoError(t, commands.ApplyNamedCheckout(plan))
	assert.Equal(t, first, gitTest(t, linked, "rev-parse", "HEAD"))
	assert.Equal(t, "refs/heads/topic", gitTest(t, linked, "symbolic-ref", "--no-recurse", "HEAD"))
}

func TestNamedCheckoutSHA256RepositoryWhenSupported(t *testing.T) {
	root := t.TempDir()
	init := exec.Command("git", "-C", root, "init", "-q", "--object-format=sha256")
	if output, err := init.CombinedOutput(); err != nil {
		t.Skipf("Git does not support SHA-256 repositories: %s", output)
	}
	gitTest(t, root, "config", "user.name", "Test")
	gitTest(t, root, "config", "user.email", "test@example.invalid")
	writeTestFile(t, filepath.Join(root, "file"), []byte("sha256\n"), 0o644)
	gitTest(t, root, "add", "file")
	gitTest(t, root, "commit", "-qm", "sha256")
	oid := gitTest(t, root, "rev-parse", "HEAD")
	assert.Len(t, oid, 64)
	commands := realBranchPlacementCommands(t, root)
	plan, err := commands.PrepareNamedCheckout("topic", oid)
	namedCheckoutMustNoError(t, err)
	namedCheckoutMustNoError(t, commands.ApplyNamedCheckout(plan))
	assert.Equal(t, oid, gitTest(t, root, "rev-parse", "HEAD"))
}

func TestNamedCheckoutDisablesRetryAfterRefTransactionHookFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Git reference-transaction shell hooks require a POSIX shell")
	}
	root, first, second := transactionHistory(t)
	gitTest(t, root, "branch", "topic", second)
	beforeHead := gitTest(t, root, "rev-parse", "HEAD")
	hooks := filepath.Join(t.TempDir(), "hooks")
	namedCheckoutMustNoError(t, os.MkdirAll(hooks, 0o755))
	count := filepath.Join(t.TempDir(), "hook-count")
	hook := "#!/bin/sh\nif [ \"$1\" = prepared ]; then echo called >> \"" + count + "\"; echo 'cannot lock ref' >&2; exit 1; fi\nexit 0\n"
	namedCheckoutMustNoError(t, os.WriteFile(filepath.Join(hooks, "reference-transaction"), []byte(hook), 0o755))
	gitTest(t, root, "config", "core.hooksPath", hooks)
	commands := transactionCommands(t, root)
	plan, err := commands.PrepareNamedCheckout("topic", first)
	namedCheckoutMustNoError(t, err)
	err = commands.ApplyNamedCheckout(plan)
	var checkoutErr *NamedCheckoutError
	if !assert.ErrorAs(t, err, &checkoutErr) {
		t.FailNow()
	}
	contents, readErr := os.ReadFile(count)
	namedCheckoutMustNoError(t, readErr)
	assert.Equal(t, 1, strings.Count(string(contents), "called"), "a post-transition lock diagnostic must never trigger a whole-checkout retry")
	assert.Equal(t, second, gitTest(t, root, "rev-parse", "refs/heads/topic"), "the rejecting ref hook leaves the destination ref unchanged")
	assert.Equal(t, beforeHead, gitTest(t, root, "rev-parse", "HEAD"), "the rejected reference transaction leaves HEAD unchanged")
	assert.Equal(t, "one", gitTest(t, root, "show", ":tracked"), "checkout may already have updated the index before its ref hook rejects")
	contents, readErr = os.ReadFile(filepath.Join(root, "tracked"))
	namedCheckoutMustNoError(t, readErr)
	assert.Equal(t, "one\n", string(contents), "checkout may already have updated working files before its ref hook rejects")
}
