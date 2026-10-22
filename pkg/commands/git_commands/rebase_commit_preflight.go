package git_commands

import (
	"fmt"

	goerrors "github.com/go-errors/errors"
)

// ErrRebaseWouldRetainTrackedChanges is returned before creating an amend-to
// fixup commit when tracked worktree changes would remain outside that commit.
var ErrRebaseWouldRetainTrackedChanges = goerrors.New("Cannot amend to commit: commit tracked changes first")

func (self *GitCommon) requireCleanAfterIndexCommitForRebase() error {
	args := NewGitCmd("diff").Arg("--quiet", "--no-ext-diff", "--ignore-submodules=untracked", "--").ToArgv()
	cmdObj := self.cmd.New(args).DontLog()
	_, _, err := cmdObj.RunWithOutputs()
	if err == nil {
		return nil
	}
	if processState := cmdObj.GetCmd().ProcessState; processState != nil && processState.ExitCode() == 1 {
		return ErrRebaseWouldRetainTrackedChanges
	}
	return fmt.Errorf("checking tracked changes before creating rebase fixup commit: %w", err)
}
