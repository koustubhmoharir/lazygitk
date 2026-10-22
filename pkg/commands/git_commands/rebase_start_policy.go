package git_commands

import (
	"fmt"

	"github.com/go-errors/errors"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// ErrRebaseRequiresClean is returned when a rebase cannot start because the
// index or worktree has changes. Commit those changes before rebasing.
var ErrRebaseRequiresClean = errors.New("Cannot start rebase: commit your changes first")

func (self *GitCommon) requireCleanRebaseStart() error {
	cmdObj := self.cmd.New(cleanRebaseStatusCommand().ToArgv()).DontLog()
	return requireCleanRebaseStatus(cmdObj)
}

// cleanRebaseStatusCommand builds the explicit status query shared by rebase
// starts in this worktree and in other selected worktrees.
func cleanRebaseStatusCommand() *GitCommandBuilder {
	return NewGitCmd("status").Arg("--porcelain=v1", "-z", "--untracked-files=no", "--ignore-submodules=untracked")
}

func requireCleanRebaseStatus(cmdObj *oscommands.CmdObj) error {
	output, err := cmdObj.DontLog().RunWithOutput()
	if err != nil {
		return fmt.Errorf("checking for changes before rebase: %w", err)
	}
	if output != "" {
		return ErrRebaseRequiresClean
	}
	return nil
}
