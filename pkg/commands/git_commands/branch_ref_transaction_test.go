package git_commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func transactionCommands(t *testing.T, root string) *BranchCommands {
	t.Helper()
	osCommand := oscommands.NewDummyOSCommand()
	return NewBranchCommands(NewGitCommon(common.NewDummyCommon(), nil, osCommand.Cmd, osCommand, MockRepoPaths(root), nil, nil))
}

func mustTransactionSetup(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func transactionHistory(t *testing.T) (string, string, string) {
	t.Helper()
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("one\n"), 0o644)
	gitTest(t, root, "add", "tracked")
	gitTest(t, root, "commit", "-qm", "one")
	first := gitTest(t, root, "rev-parse", "HEAD")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("two\n"), 0o644)
	gitTest(t, root, "commit", "-qam", "two")
	second := gitTest(t, root, "rev-parse", "HEAD")
	return root, first, second
}

func lockFiles(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(filepath.Join(root, ".git"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".lock") {
			paths = append(paths, path)
		}
		return nil
	})
	mustTransactionSetup(t, err)
	return paths
}

func TestGuardedBranchRefUpdateCreateMoveAndPreserveWorktree(t *testing.T) {
	for _, tc := range []struct {
		name   string
		create bool
	}{{"create", true}, {"move", false}} {
		t.Run(tc.name, func(t *testing.T) {
			root, old, next := transactionHistory(t)
			ref := "refs/heads/topic"
			if !tc.create {
				gitTest(t, root, "branch", "topic", old)
			}
			head := gitTest(t, root, "rev-parse", "HEAD")
			headRef := gitTest(t, root, "symbolic-ref", "-q", "HEAD")
			writeTestFile(t, filepath.Join(root, "tracked"), []byte("staged tracked bytes\n"), 0o644)
			gitTest(t, root, "add", "tracked")
			index := gitTest(t, root, "write-tree")
			writeTestFile(t, filepath.Join(root, "tracked"), []byte("unstaged tracked bytes\n"), 0o644)
			writeTestFile(t, filepath.Join(root, "dirty"), []byte("dirty bytes\x00\n"), 0o600)
			gitTest(t, root, "config", "branch.topic.remote", "origin")
			gitTest(t, root, "config", "branch.topic.merge", "refs/heads/upstream-topic")
			status := gitTest(t, root, "status", "--porcelain=v1", "-z")
			guardCalls := 0
			err := transactionCommands(t, root).guardedBranchRefUpdate(ref, next, func() string {
				if tc.create {
					return strings.Repeat("0", len(next))
				}
				return old
			}(), "lazygitk: move branch to selected commit", func() error {
				guardCalls++
				if tc.create {
					cmd := exec.Command("git", "-C", root, "rev-parse", "--verify", ref)
					assert.Error(t, cmd.Run())
				} else {
					assert.Equal(t, old, gitTest(t, root, "rev-parse", "--verify", ref))
				}
				if !tc.create {
					cmd := exec.Command("git", "-C", root, "-c", "core.filesRefLockTimeout=100", "update-ref", ref, next, old)
					if output, err := cmd.CombinedOutput(); err == nil {
						return fmt.Errorf("competing same-ref update unexpectedly succeeded: %s", output)
					}
					assert.Equal(t, old, gitTest(t, root, "rev-parse", "--verify", ref))
				}
				return nil
			})
			mustTransactionSetup(t, err)
			assert.Equal(t, 1, guardCalls)
			assert.Equal(t, next, gitTest(t, root, "rev-parse", "--verify", ref))
			assert.Contains(t, gitTest(t, root, "reflog", "-1", "--format=%gs", "topic"), "lazygitk: move branch to selected commit")
			assert.Equal(t, head, gitTest(t, root, "rev-parse", "HEAD"))
			assert.Equal(t, headRef, gitTest(t, root, "symbolic-ref", "-q", "HEAD"))
			assert.Equal(t, index, gitTest(t, root, "write-tree"))
			assert.Equal(t, status, gitTest(t, root, "status", "--porcelain=v1", "-z"))
			assert.Equal(t, "dirty bytes\x00\n", string(mustReadBranchRefFile(t, filepath.Join(root, "dirty"))))
			assert.Equal(t, "origin", gitTest(t, root, "config", "--get", "branch.topic.remote"))
			assert.Equal(t, "refs/heads/upstream-topic", gitTest(t, root, "config", "--get", "branch.topic.merge"))
			assert.Equal(t, "staged tracked bytes", gitTest(t, root, "show", ":tracked"))
			assert.Equal(t, "unstaged tracked bytes\n", string(mustReadBranchRefFile(t, filepath.Join(root, "tracked"))))
			assert.Empty(t, lockFiles(t, root))
		})
	}
}

func mustReadBranchRefFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	mustTransactionSetup(t, err)
	return data
}

func TestGuardedBranchRefUpdateRefusalMismatchAndSymbolicAlias(t *testing.T) {
	root, old, next := transactionHistory(t)
	gitTest(t, root, "branch", "topic", old)
	sentinel := errors.New("guard refused")
	calls := 0
	err := transactionCommands(t, root).guardedBranchRefUpdate("refs/heads/topic", next, old, "move", func() error { calls++; return sentinel })
	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/topic"))
	assert.Empty(t, lockFiles(t, root))
	calls = 0
	err = transactionCommands(t, root).guardedBranchRefUpdate("refs/heads/topic", next, strings.Repeat("0", len(old)), "move", func() error { calls++; return nil })
	assert.Error(t, err)
	assert.Zero(t, calls)
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/topic"))

	gitTest(t, root, "symbolic-ref", "refs/heads/alias", "refs/heads/topic")
	calls = 0
	err = transactionCommands(t, root).guardedBranchRefUpdate("refs/heads/alias", next, old, "move", func() error {
		calls++
		assert.Equal(t, "refs/heads/topic", gitTest(t, root, "symbolic-ref", "refs/heads/alias"))
		assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/alias"))
		return sentinel
	})
	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, calls)
	assert.Equal(t, "refs/heads/topic", gitTest(t, root, "symbolic-ref", "refs/heads/alias"))
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/topic"))
	assert.Empty(t, lockFiles(t, root))
}

func TestGuardedBranchRefUpdateGuardPanicReleasesLock(t *testing.T) {
	root, old, next := transactionHistory(t)
	gitTest(t, root, "branch", "topic", old)
	func() {
		defer func() { assert.NotNil(t, recover()) }()
		_ = transactionCommands(t, root).guardedBranchRefUpdate("refs/heads/topic", next, old, "move", func() error { panic("guard") })
	}()
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/topic"))
	assert.Empty(t, lockFiles(t, root))
}

func TestBranchRefProtocolOutputChunkingAndBounds(t *testing.T) {
	output := newBranchRefProtocolOutput()
	for _, part := range [][]byte{[]byte("hook text without newline"), []byte("start: ok\r\nlong"), []byte(strings.Repeat("x", 100000) + "prepare: ok\ncommit: ok\ncommit: ok\n")} {
		_, _ = output.Write(part)
	}
	assert.True(t, output.acknowledged("start"))
	assert.True(t, output.acknowledged("prepare"))
	assert.True(t, output.acknowledged("commit"))
	assert.LessOrEqual(t, len(output.line), 64)
	assert.LessOrEqual(t, len(output.diagnostic.data), 64*1024)
	tail := &branchRefDiagnosticTail{}
	_, _ = tail.Write([]byte(strings.Repeat("z", 100000)))
	assert.Len(t, tail.data, 64*1024)
}

func TestBranchRefTransactionHookOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Git reference-transaction shell hooks require a POSIX shell")
	}
	root, old, next := transactionHistory(t)
	gitTest(t, root, "branch", "topic", old)
	hook := filepath.Join(root, ".git", "hooks", "reference-transaction")
	writeTestFile(t, hook, []byte("#!/bin/sh\nprintf hook-stdout\nprintf hook-stderr >&2\n"), 0o755)
	commands := transactionCommands(t, root)
	// The transaction owns both pipes; use the same real Git process and verify
	// successful progress despite hook bytes with no terminating newline.
	err := commands.guardedBranchRefUpdate("refs/heads/topic", next, old, "hook-test", func() error { return nil })
	mustTransactionSetup(t, err)
	assert.Equal(t, next, gitTest(t, root, "rev-parse", "refs/heads/topic"))
}

func TestBranchRefTransactionHookRejectsPrepareBeforeGuard(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Git reference-transaction shell hooks require a POSIX shell")
	}
	root, old, next := transactionHistory(t)
	gitTest(t, root, "branch", "topic", old)
	hook := filepath.Join(root, ".git", "hooks", "reference-transaction")
	writeTestFile(t, hook, []byte("#!/bin/sh\n[ \"$1\" = prepared ] && exit 1\nexit 0\n"), 0o755)
	guardCalls := 0
	err := transactionCommands(t, root).guardedBranchRefUpdate("refs/heads/topic", next, old, "hook-reject", func() error { guardCalls++; return nil })
	assert.Error(t, err)
	assert.Zero(t, guardCalls)
	assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/topic"))
	assert.Empty(t, lockFiles(t, root))
}

type branchRefHelperBuilder struct {
	inner oscommands.ICmdObjBuilder
	mode  string
	calls atomic.Int32
}

func (builder *branchRefHelperBuilder) New(args []string) *oscommands.CmdObj {
	builder.calls.Add(1)
	var cmd *exec.Cmd
	if builder.mode == "missing" {
		cmd = exec.Command(filepath.Join(os.TempDir(), "branch-ref-definitely-missing-executable"))
	} else {
		cmd = exec.Command(os.Args[0], "-test.run=^TestBranchRefTransactionHelperProcess$")
		cmd.Env = append(os.Environ(), "LAZYGIT_BRANCH_REF_HELPER="+builder.mode)
	}
	return builder.inner.(*oscommands.CmdObjBuilder).NewFromCmd(cmd)
}

func (builder *branchRefHelperBuilder) NewShell(command, functionsFile string) *oscommands.CmdObj {
	return builder.inner.NewShell(command, functionsFile)
}

func (builder *branchRefHelperBuilder) Quote(value string) string { return builder.inner.Quote(value) }

func TestBranchRefTransactionHelperProcess(t *testing.T) {
	mode := os.Getenv("LAZYGIT_BRANCH_REF_HELPER")
	if mode == "" {
		return
	}
	for scanner := bufio.NewScanner(os.Stdin); scanner.Scan(); {
		line := scanner.Text()
		switch line {
		case "start":
			_, _ = fmt.Fprintln(os.Stdout, "start: ok")
		case "prepare":
			if mode == "before-prepare" {
				os.Exit(17)
			}
			_, _ = fmt.Fprintln(os.Stdout, "prepare: ok")
		case "commit":
			if mode == "after-commit" {
				os.Exit(19)
			}
		}
	}
}

func TestGuardedBranchRefUpdateProcessFailuresDoNotRetry(t *testing.T) {
	root, old, next := transactionHistory(t)
	gitTest(t, root, "branch", "topic", old)
	for _, tc := range []struct {
		mode            string
		wantCommitError bool
	}{{"missing", false}, {"before-prepare", false}, {"after-commit", true}} {
		t.Run(tc.mode, func(t *testing.T) {
			builder := &branchRefHelperBuilder{inner: oscommands.NewDummyOSCommand().Cmd, mode: tc.mode}
			commands := transactionCommands(t, root)
			commands.cmd = builder
			guardCalls := 0
			err := commands.guardedBranchRefUpdate("refs/heads/topic", next, old, "helper-test", func() error { guardCalls++; return nil })
			assert.Error(t, err)
			assert.Equal(t, int32(1), builder.calls.Load())
			if tc.mode == "before-prepare" {
				assert.Contains(t, err.Error(), "before prepare acknowledgement")
			}
			if tc.mode == "missing" {
				assert.Error(t, err)
			}
			if tc.wantCommitError {
				var commitErr *branchRefCommitError
				assert.ErrorAs(t, err, &commitErr)
			} else {
				var commitErr *branchRefCommitError
				assert.NotErrorAs(t, err, &commitErr)
			}
			if tc.mode == "after-commit" {
				assert.Equal(t, 1, guardCalls)
			} else {
				assert.Zero(t, guardCalls)
			}
			assert.Equal(t, old, gitTest(t, root, "rev-parse", "refs/heads/topic"))
			assert.Empty(t, lockFiles(t, root))
		})
	}
}
