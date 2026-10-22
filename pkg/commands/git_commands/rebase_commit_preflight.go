package git_commands

import (
	"fmt"

	goerrors "github.com/go-errors/errors"
)

// ErrRebaseWouldRetainTrackedChanges is returned before a compound commit/rebase
// when tracked worktree changes would remain outside the new commit.
var ErrRebaseWouldRetainTrackedChanges = goerrors.New("Cannot amend to commit: commit tracked changes first")

// CheckCleanAfterCommit checks a compound commit/rebase before its first write.
// Message-only amendments do not consume the index, so require it clean too.
func (self *RebaseCommands) CheckCleanAfterCommit(includeFileChanges bool) error {
	if !includeFileChanges {
		return self.requireCleanRebaseStart()
	}
	return self.requireCleanAfterIndexCommitForRebase()
}

func (self *GitCommon) requireCleanAfterIndexCommitForRebase() error {
	args := NewGitCmd("diff").Arg("--quiet", "--no-ext-diff", "--ignore-submodules=untracked", "--").ToArgv()
	cmdObj := self.cmd.New(args).DontLog()
	_, _, err := cmdObj.RunWithOutputs()
	if err == nil {
		return nil
	}
	// The application's Git runner executes a clone, so the original command's
	// ProcessState is not authoritative. Read the wrapped native exit error.
	if commandExitedWithOne(err) {
		return ErrRebaseWouldRetainTrackedChanges
	}
	return fmt.Errorf("checking tracked changes before creating rebase fixup commit: %w", err)
}
