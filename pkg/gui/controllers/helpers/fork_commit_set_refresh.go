package helpers

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: the Commits selection can hold commits that a refresh does not list. A filter or
// the commit limit hides them, or a rewrite outside the selection replaced or removed
// them. The selection keeps them (see context/fork_commit_multi_select.go). After each
// refresh, the selected commits the new list lacks are classified by Git on a worker
// (ClassifySelectedCommits). Those whose objects are gone are removed from the selection,
// with a notice. Those that no ref or HEAD reaches stay selected, with a notice the first
// time each is seen. The rest are only hidden, and get no notice.
//
// checkSelectedCommitsAfterRefresh is the integration point, called from
// refreshCommitsWithLimit once the new list is in the model.

// checkSelectedCommitsAfterRefresh starts the check for the commits the selection holds
// and commits does not. It does nothing when there are none.
func (self *RefreshHelper) checkSelectedCommitsAfterRefresh(env refreshEnv, commits []*models.Commit) {
	absent := self.c.Contexts().LocalCommits.SelectedCommitHashesNotIn(commits)
	if len(absent) == 0 {
		return
	}
	repository := captureActionRepository(self.c)
	self.onWorker(env.background, func(gocui.Task) error {
		missing, unreachable, err := repository.git.Commit.ClassifySelectedCommits(absent)
		if err != nil {
			return err
		}
		self.onUIThread(env.background, func() error {
			return self.applySelectedCommitsCheck(repository, missing, unreachable)
		})
		return nil
	})
}

// applySelectedCommitsCheck updates the selection and the notices on the UI thread. A
// repository change while the check ran drops the result: the next refresh checks again.
func (self *RefreshHelper) applySelectedCommitsCheck(repository *capturedActionRepository, missing []string, unreachable []string) error {
	if err := repository.checkRepository(); err != nil {
		return nil
	}
	commits := self.c.Contexts().LocalCommits
	if len(missing) > 0 {
		commits.RemoveFromCommitSet(missing)
		self.c.Toast(fmt.Sprintf(self.c.Tr.CommitSetMissingNotice, commitSetShortHashes(missing)))
	}
	if fresh := commits.NewUnreachableNotices(unreachable); len(fresh) > 0 {
		self.c.Toast(fmt.Sprintf(self.c.Tr.CommitSetRewrittenNotice, commitSetShortHashes(fresh)))
	}
	self.c.PostRefreshUpdate(commits)
	return nil
}

func commitSetShortHashes(hashes []string) string {
	short := make([]string, len(hashes))
	for i, hash := range hashes {
		short[i] = utils.ShortHash(hash)
	}
	return strings.Join(short, ", ")
}
