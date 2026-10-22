package git_commands

import (
	"fmt"
	"os"
	"path/filepath"
)

// rejectWorktreeOperation rejects repository states in which starting a
// worktree-level operation would overlap an existing Git operation.
func (self *GitCommon) rejectWorktreeOperation(action string) error {
	gitDir := self.repoPaths.WorktreeGitDirPath()
	markers := []struct{ name, operation string }{
		{"rebase-merge", "rebase"},
		{"rebase-apply", "rebase"},
		{"MERGE_HEAD", "merge"},
		{"CHERRY_PICK_HEAD", "cherry-pick"},
		{"REVERT_HEAD", "revert"},
		{"sequencer", "sequencer operation"},
		{"BISECT_START", "bisect"},
	}
	for _, marker := range markers {
		_, err := os.Lstat(filepath.Join(gitDir, marker.name))
		if err == nil {
			return fmt.Errorf("cannot %s while a %s is in progress; finish or abort it first", action, marker.operation)
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("inspect %s operation state: %w", marker.name, err)
		}
	}
	return nil
}
