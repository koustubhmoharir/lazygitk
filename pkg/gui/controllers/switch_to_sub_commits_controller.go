package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

var _ types.IController = &SwitchToSubCommitsController{}

type CanSwitchToSubCommits interface {
	types.IListContext
	GetSelectedRef() models.Ref
	ShowBranchHeadsInSubCommits() bool
}

// Not using our ListControllerTrait because our 'selected' item is not a list item
// but an attribute on it i.e. the ref of an item.
type SwitchToSubCommitsController struct {
	baseController
	*ListControllerTrait[models.Ref]
	c       *ControllerCommon
	context CanSwitchToSubCommits
	// Fork: see WithoutDoubleClick.
	keyOnly bool
}

// Fork: WithoutDoubleClick leaves the context's existing double-click action
// alone. Worktrees double-click switches worktree; only v views its commits.
func (self *SwitchToSubCommitsController) WithoutDoubleClick() *SwitchToSubCommitsController {
	self.keyOnly = true
	return self
}

func NewSwitchToSubCommitsController(
	c *ControllerCommon,
	context CanSwitchToSubCommits,
) *SwitchToSubCommitsController {
	return &SwitchToSubCommitsController{
		baseController: baseController{},
		ListControllerTrait: NewListControllerTrait(
			c,
			context,
			context.GetSelectedRef,
			func() ([]models.Ref, int, int) {
				panic("Not implemented")
			},
		),
		c:       c,
		context: context,
	}
}

func (self *SwitchToSubCommitsController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	bindings := []*types.Binding{
		{
			Handler:           self.viewCommits,
			GetDisabledReason: self.require(self.singleItemSelected()),
			Keys:              opts.GetKeys(opts.Config.Universal.GoInto),
			Description:       self.c.Tr.ViewCommits,
			CommandID:         "navigation.open-commits",
			Verb:              "view-commits",
		},
	}

	return bindings
}

func (self *SwitchToSubCommitsController) GetOnDoubleClick() func() error {
	if self.keyOnly {
		return nil
	}
	return self.viewCommits
}

func (self *SwitchToSubCommitsController) viewCommits() error {
	ref := self.context.GetSelectedRef()
	if ref == nil {
		return nil
	}

	return self.c.Helpers().SubCommits.ViewSubCommits(helpers.ViewSubCommitsOpts{
		Ref:             ref,
		TitleRef:        ref.RefName(),
		Context:         self.context,
		ShowBranchHeads: self.context.ShowBranchHeadsInSubCommits(),
	})
}
