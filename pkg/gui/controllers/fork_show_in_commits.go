package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: w v shows the selected branch's, tag's, remote branch's or commit's commit in
// Commits (see helpers/fork_show_in_commits.go for what Commits does). The commit is
// read when the key is pressed: a local branch's tip, a remote branch's or tag's tip
// resolved now, or the commit of a reflog entry or sub-commit. Whether the unfiltered
// log can contain the commit is checked on a worker first. When it cannot, nothing
// changes and the refusal is shown. The integration points are the GetKeybindings of
// BranchesController, RemoteBranchesController, TagsController, ReflogCommitsController
// and SubCommitsController.

const showInCommitsCommandID = "commits.show-selected"

// showInCommitsBinding is w v. handler reads the item when it runs, and disabled
// refuses a multi-selection.
func showInCommitsBinding(c *ControllerCommon, handler func() error, disabled func() *types.DisabledReason) *types.Binding {
	return &types.Binding{
		CommandID:         showInCommitsCommandID,
		Verb:              "show",
		Description:       c.Tr.ShowInCommits,
		Handler:           handler,
		GetDisabledReason: disabled,
	}
}

func (self *BranchesController) showInCommitsBinding() *types.Binding {
	return showInCommitsBinding(self.c,
		self.withItem(func(branch *models.Branch) error {
			return showCommitInCommits(self.c, branch.CommitHash)
		}),
		self.require(self.singleItemSelected()))
}

func (self *RemoteBranchesController) showInCommitsBinding() *types.Binding {
	return showInCommitsBinding(self.c,
		self.withItem(func(branch *models.RemoteBranch) error {
			return showRefInCommits(self.c, branch.FullRefName())
		}),
		self.require(self.singleItemSelected()))
}

func (self *TagsController) showInCommitsBinding() *types.Binding {
	return showInCommitsBinding(self.c,
		self.withItem(func(tag *models.Tag) error {
			return showRefInCommits(self.c, tag.FullRefName())
		}),
		self.require(self.singleItemSelected()))
}

func (self *ReflogCommitsController) showInCommitsBinding() *types.Binding {
	return showInCommitsBinding(self.c,
		self.withItem(func(commit *models.Commit) error {
			return showCommitInCommits(self.c, commit.Hash())
		}),
		self.require(self.singleItemSelected()))
}

func (self *SubCommitsController) showInCommitsBinding() *types.Binding {
	return showInCommitsBinding(self.c,
		self.withItem(func(commit *models.Commit) error {
			return showCommitInCommits(self.c, commit.Hash())
		}),
		self.require(self.singleItemSelected()))
}

// showRefInCommits is showCommitInCommits for a full ref name. The tip is resolved when
// the key is pressed, as applyRefScopeAtTip does.
func showRefInCommits(c *ControllerCommon, fullRefName string) error {
	hash, err := c.Git().Commit.ResolveRefCommitHash(fullRefName)
	if err != nil {
		return err
	}
	return showCommitInCommits(c, hash)
}

// showCommitInCommits checks the commit against the unfiltered log on a worker, then
// shows it on the UI thread, or refuses it there. wholeGraph is read now, as the
// unfiltered log's refresh reads it.
func showCommitInCommits(c *ControllerCommon, hash string) error {
	wholeGraph := c.Contexts().LocalCommits.GetShowWholeGitGraph()
	return c.WithWaitingStatus(c.Tr.FindingCommitStatus, func(gocui.Task) error {
		inLog, err := c.Git().Commit.IsCommitInCommitsLog(hash, wholeGraph)
		if err != nil {
			return err
		}
		c.OnUIThread(func() error {
			if !inLog {
				return helpers.CommitNotInCommitsLogError(c.Tr, hash)
			}
			return c.Helpers().Mode.ShowCommitInCommits(hash)
		})
		return nil
	})
}
