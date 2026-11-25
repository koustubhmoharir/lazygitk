package helpers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

type SubCommitsHelper struct {
	c *HelperCommon

	refreshHelper *RefreshHelper
}

func NewSubCommitsHelper(
	c *HelperCommon,
	refreshHelper *RefreshHelper,
) *SubCommitsHelper {
	return &SubCommitsHelper{
		c:             c,
		refreshHelper: refreshHelper,
	}
}

type ViewSubCommitsOpts struct {
	Ref                     models.Ref
	RefToShowDivergenceFrom string
	TitleRef                string
	Context                 types.Context
	ShowBranchHeads         bool
}

func (self *SubCommitsHelper) ViewSubCommits(opts ViewSubCommitsOpts) error {
	commits, err := self.c.Git().Loaders.CommitLoader.GetCommits(
		git_commands.GetCommitsOptions{
			Limit:                   true,
			FilterPath:              self.c.Modes().Filtering.GetPath(),
			FilterAuthor:            self.c.Modes().Filtering.GetAuthor(),
			IncludeRebaseCommits:    false,
			RefName:                 opts.Ref.FullRefName(),
			RefForPushedStatus:      opts.Ref,
			RefToShowDivergenceFrom: opts.RefToShowDivergenceFrom,
			MainBranches:            self.c.Model().MainBranches,
			HashPool:                self.c.Model().HashPool,
		},
	)
	if err != nil {
		return err
	}

	self.c.Model().SubCommits = commits
	self.refreshHelper.RefreshAuthors(commits)

	subCommitsContext := self.c.Contexts().SubCommits
	// Fork: a set belongs to the ref it was made on, so opening another ref clears it; opening
	// the same ref keeps it, and only drops a run in progress, so that reopening adds nothing
	// to the set (see fork_commit_multi_select.go). Escape clears it when it leaves the view.
	if previous := subCommitsContext.GetRef(); previous == nil || previous.FullRefName() != opts.Ref.FullRefName() {
		subCommitsContext.ClearMultiSelection()
	} else {
		subCommitsContext.DropRun()
	}
	subCommitsContext.SetSelection(0)
	subCommitsContext.SetParentContext(opts.Context)
	subCommitsContext.SetWindowName(opts.Context.GetWindowName())
	subCommitsContext.SetTitleRef(utils.TruncateWithEllipsis(opts.TitleRef, 50))
	subCommitsContext.SetRef(opts.Ref)
	subCommitsContext.SetRefToShowDivergenceFrom(opts.RefToShowDivergenceFrom)
	subCommitsContext.SetLimitCommits(true)
	subCommitsContext.SetShowBranchHeads(opts.ShowBranchHeads)
	subCommitsContext.ClearSearchString()
	subCommitsContext.GetView().ClearSearch()
	subCommitsContext.GetView().TitlePrefix = opts.Context.GetView().TitlePrefix

	self.c.PostRefreshUpdate(self.c.Contexts().SubCommits)

	self.c.Context().Push(self.c.Contexts().SubCommits, types.OnFocusOpts{})
	return nil
}
