package helpers

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// Fork: w v shows a commit in Commits (see controllers/fork_show_in_commits.go for the
// entry point and the reachability check). This file is the Commits side. Commits may
// not hold the commit yet: a limited list stops at its first commits, and a log filter
// may hide it. Then Commits is loaded further, or the filter is cleared without a
// question, and the commit is selected once the reloaded list is in place. The screen
// mode and focus land with that list, as ModeHelper's filter changes do.

// CommitNotInCommitsLogError is the refusal when the unfiltered Commits log cannot show
// the commit hash.
func CommitNotInCommitsLogError(tr *i18n.TranslationSet, hash string) error {
	return fmt.Errorf(tr.CommitNotInCommitsLog, utils.ShortHash(hash))
}

// ShowCommitInCommits makes hash the only selected commit of Commits and focuses it. The
// caller has checked that the unfiltered log can contain hash. When the list lacks it,
// the list is reloaded: first without the limit, then without a log filter, and each
// step ends by showing the commit or moving on to the next.
func (self *ModeHelper) ShowCommitInCommits(hash string) error {
	return self.showCommitInCommitsStep(hash)
}

func (self *ModeHelper) showCommitInCommitsStep(hash string) error {
	commits := self.c.Contexts().LocalCommits
	if lo.ContainsBy(commits.GetItems(), func(commit *models.Commit) bool { return commit.Hash() == hash }) {
		self.focusCommitInCommits(hash)
		return nil
	}
	if commits.GetLimitCommits() {
		commits.SetLimitCommits(false)
		return self.reloadCommits(self.c.Tr.LoadingCommitsStatus, func() error { return self.showCommitInCommitsStep(hash) })
	}
	if self.c.Modes().Filtering.Active() {
		// Unlike ClearFiltering, this does not ask, and it does not select the commit
		// that was selected before the filter: the commit to show is the one selected.
		self.c.Modes().Filtering.Reset()
		return self.reloadCommits(self.c.Tr.RemovingFilterStatus, func() error { return self.showCommitInCommitsStep(hash) })
	}
	return CommitNotInCommitsLogError(self.c.Tr, hash)
}

// reloadCommits reloads the lists that depend on the log, on a worker with a waiting
// status. Its Then runs on the UI thread once the lists are in place, and calls then.
func (self *ModeHelper) reloadCommits(message string, then func() error) error {
	return self.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: message}, func(gocui.Task) error {
		self.c.RefreshFromWorker(types.RefreshOptions{
			Scope:          ScopesToRefreshWhenFilteringModeChanges(),
			BatchUIUpdates: true,
			Then:           then,
		})
		return nil
	})
}

// selectCommitAfterClearingFilter ends a filter clear. It selects cursorHash, the commit
// that was selected in filtering mode, or else preFilterHash, the commit selected before
// filtering, in the reloaded list, and then calls land, which applies the screen mode and
// focus. The filter is already cleared, so the list is the unfiltered log. When the list
// is limited and holds neither commit, but the unfiltered log can contain one of them,
// the limit is lifted and the list reloaded first, and the selection is tried again; the
// explicit selection is not touched. Otherwise nothing is selected, as before.
func (self *ModeHelper) selectCommitAfterClearingFilter(cursorHash string, preFilterHash string, land func()) {
	commits := self.c.Contexts().LocalCommits
	candidates := lo.Filter([]string{cursorHash, preFilterHash}, func(hash string, _ int) bool { return hash != "" })
	for _, hash := range candidates {
		if commits.SelectCommitByHash(hash) {
			land()
			return
		}
	}
	if !commits.GetLimitCommits() || len(candidates) == 0 {
		land()
		return
	}

	wholeGraph := commits.GetShowWholeGitGraph()
	_ = self.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: self.c.Tr.FindingCommitStatus}, func(gocui.Task) error {
		loadable, err := self.firstCommitInCommitsLog(candidates, wholeGraph)
		self.c.OnUIThread(func() error {
			if err != nil || loadable == "" {
				land()
				return err
			}
			commits.SetLimitCommits(false)
			return self.reloadCommits(self.c.Tr.LoadingCommitsStatus, func() error {
				self.selectCommitAfterClearingFilter(cursorHash, preFilterHash, land)
				return nil
			})
		})
		return nil
	})
}

// firstCommitInCommitsLog returns the first of hashes that the unfiltered Commits log can
// contain, or "" when there is none. It runs git, so it is called on a worker.
func (self *ModeHelper) firstCommitInCommitsLog(hashes []string, wholeGraph bool) (string, error) {
	for _, hash := range hashes {
		inLog, err := self.c.Git().Commit.IsCommitInCommitsLog(hash, wholeGraph)
		if err != nil {
			return "", err
		}
		if inLog {
			return hash, nil
		}
	}
	return "", nil
}

// focusCommitInCommits moves the cursor to hash and focuses Commits. The explicit set is
// kept, as Escape keeps it: the set may hold commits that this does not show. A run in
// progress ends with the move, as a cursor move ends it; its commits join the set.
func (self *ModeHelper) focusCommitInCommits(hash string) {
	commits := self.c.Contexts().LocalCommits
	commits.SelectCommitByHash(hash)
	self.c.Context().Push(commits, types.OnFocusOpts{})
	self.enlargeSideAreaForCommits()
	self.c.PostRefreshUpdate(commits)
}

// enlargeSideAreaForCommits makes the side area half screen when Commits is shown from a
// normal screen or from enlarged main views. A side area that is already half or full
// is left as the user chose it.
func (self *ModeHelper) enlargeSideAreaForCommits() {
	repoState := self.c.State().GetRepoState()
	if repoState.GetScreenMode() != types.SCREEN_NORMAL && repoState.GetScreenModeArea() == types.SCREEN_AREA_SIDE {
		return
	}
	repoState.SetScreenModeInArea(types.SCREEN_HALF, types.SCREEN_AREA_SIDE)
	for _, context := range self.c.Context().AllList() {
		if context.NeedsRerenderOnWidthChange() == types.NEEDS_RERENDER_ON_WIDTH_CHANGE_WHEN_SCREEN_MODE_CHANGES {
			context.HandleRender()
		}
	}
}
