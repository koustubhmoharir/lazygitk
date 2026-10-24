package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func selectedResetBindings(c *ControllerCommon, disabled func() *types.DisabledReason, reset func(string) error) []*types.Binding {
	return []*types.Binding{
		{CommandID: "branch.reset.hard", Verb: "reset-hard", Description: c.Tr.Actions.HardReset, OpensMenu: true, GetDisabledReason: disabled, Handler: func() error { return reset("hard") }},
		{CommandID: "branch.reset.mixed", Verb: "reset-mixed", Description: c.Tr.Actions.MixedReset, OpensMenu: true, GetDisabledReason: disabled, Handler: func() error { return reset("mixed") }},
		{CommandID: "branch.reset.soft", Verb: "reset-soft", Description: c.Tr.Actions.SoftReset, OpensMenu: true, GetDisabledReason: disabled, Handler: func() error { return reset("soft") }},
	}
}

func (self *BasicCommitsController) addSelectedResetBindings(bindings []*types.Binding) []*types.Binding {
	reset := func(mode string) error {
		return self.withItem(func(commit *models.Commit) error {
			return helpers.ResetCurrentToCapturedSourceMode(self.c.HelperCommon, commit.Hash(), commit.Hash(), mode)
		})()
	}
	return append(selectedResetBindings(self.c, self.require(self.singleItemSelected()), reset), bindings...)
}

func (self *BranchesController) addSelectedResetBindings(bindings []*types.Binding) []*types.Binding {
	reset := func(mode string) error {
		return self.withItem(func(branch *models.Branch) error {
			return helpers.ResetCurrentToCapturedSourceMode(self.c.HelperCommon, branch.Name, branch.FullRefName(), mode)
		})()
	}
	return append(selectedResetBindings(self.c, self.require(self.singleItemSelected()), reset), bindings...)
}
