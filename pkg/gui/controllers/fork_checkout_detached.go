package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: s d detaches HEAD at the commit of the selected item. It is the same
// captured detached checkout as the detached item of the old s c menu, and it is
// offered in every pane that has s c: branches, remote branches, tags, and the
// commit panes.
func checkoutDetachedBinding(c *ControllerCommon, disabled func() *types.DisabledReason, handler func() error) *types.Binding {
	return &types.Binding{
		CommandID:         "commit.checkout-detached",
		Verb:              "checkout-detached",
		Handler:           handler,
		GetDisabledReason: disabled,
		Description:       c.Tr.CheckoutDetached,
	}
}

func (self *BasicCommitsController) addSelectedCheckoutBindings(bindings []*types.Binding) []*types.Binding {
	detached := checkoutDetachedBinding(self.c, self.require(self.singleItemSelected()), self.withItem(func(commit *models.Commit) error {
		return self.c.Helpers().Refs.CheckoutCommitDetached(commit.Hash())
	}))
	return append([]*types.Binding{detached}, bindings...)
}
