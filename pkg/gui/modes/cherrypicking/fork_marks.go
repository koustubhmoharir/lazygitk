package cherrypicking

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// MarkCommits adds snapshots of the selected commits to the marked set.
func (self *CherryPicking) MarkCommits(commits []*models.Commit) {
	if self.DidPaste {
		self.CherryPickedCommits = make([]*models.Commit, 0, len(commits))
		self.DidPaste = false
	}
	self.ContextKey = ""

	marked := make(map[string]struct{}, len(self.CherryPickedCommits)+len(commits))
	for _, commit := range self.CherryPickedCommits {
		if commit != nil {
			marked[commit.Hash()] = struct{}{}
		}
	}
	for _, commit := range commits {
		if commit == nil {
			continue
		}
		hash := commit.Hash()
		if _, exists := marked[hash]; exists {
			continue
		}
		marked[hash] = struct{}{}
		self.CherryPickedCommits = append(self.CherryPickedCommits, snapshotCommit(commit))
	}
}

// UnmarkCommits removes only the selected commits from the marked set.
func (self *CherryPicking) UnmarkCommits(commits []*models.Commit) {
	selected := make(map[string]struct{}, len(commits))
	for _, commit := range commits {
		if commit != nil {
			selected[commit.Hash()] = struct{}{}
		}
	}
	remaining := make([]*models.Commit, 0, len(self.CherryPickedCommits))
	for _, commit := range self.CherryPickedCommits {
		if commit == nil {
			continue
		}
		if _, remove := selected[commit.Hash()]; !remove {
			remaining = append(remaining, commit)
		}
	}
	self.CherryPickedCommits = remaining
	self.ContextKey = ""
}

func snapshotCommit(commit *models.Commit) *models.Commit {
	return models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{
		Hash:          commit.Hash(),
		Name:          commit.Name,
		Status:        commit.Status,
		Action:        commit.Action,
		ActionFlag:    commit.ActionFlag,
		Tags:          append([]string(nil), commit.Tags...),
		ExtraInfo:     commit.ExtraInfo,
		AuthorName:    commit.AuthorName,
		AuthorEmail:   commit.AuthorEmail,
		UnixTimestamp: commit.UnixTimestamp,
		Divergence:    commit.Divergence,
		Parents:       append([]string(nil), commit.Parents()...),
	})
}
