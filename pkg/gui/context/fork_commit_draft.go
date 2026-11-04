package context

import (
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// Fork: the worktree's one cancelled commit message draft. Up and Down in the
// commit message box swap it with the text being edited. It sits beside
// PreservedCommitMessageFileName, but is not that message: the preserved
// message is the pending text of a new commit, and the draft is recalled for
// any commit message box. See docs-master/fork/Guide.md.
const CommitDraftFileName = "LAZYGITK_COMMIT_DRAFT"

func (self *CommitMessageContext) GetCommitDraftPath() string {
	return filepath.Join(self.c.Git().RepoPaths.WorktreeGitDirPath(), CommitDraftFileName)
}

func (self *CommitMessageContext) getCommitDraft() (string, error) {
	buf, err := afero.ReadFile(self.c.Fs, self.GetCommitDraftPath())
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func (self *CommitMessageContext) GetCommitDraftAndLogError() string {
	draft, err := self.getCommitDraft()
	if err != nil {
		self.c.Log.Errorf("error when retrieving commit draft: %v", err)
	}
	return draft
}

// Setting an empty draft removes the file.
func (self *CommitMessageContext) setCommitDraft(message string) error {
	draftFilePath := self.GetCommitDraftPath()

	if len(message) == 0 {
		err := self.c.Fs.Remove(draftFilePath)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return afero.WriteFile(self.c.Fs, draftFilePath, []byte(message), 0o644)
}

func (self *CommitMessageContext) SetCommitDraftAndLogError(message string) {
	if err := self.setCommitDraft(message); err != nil {
		self.c.Log.Errorf("error when persisting commit draft: %v", err)
	}
}
