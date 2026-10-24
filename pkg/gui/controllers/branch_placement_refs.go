package controllers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func (self *BranchesController) branchPlacementBinding() *types.Binding {
	return &types.Binding{
		CommandID: "branch.place-at-selected-commit", Verb: "place-branch",
		Handler:           self.withItem(self.beginPlacementAtLocalBranch),
		GetDisabledReason: self.require(self.singleItemSelected()),
		Description:       self.c.Tr.CreateMoveBranchHere,
		Tooltip:           self.c.Tr.PlaceBranchTooltip,
		OpensMenu:         true,
	}
}

func (self *RemoteBranchesController) branchPlacementBinding() *types.Binding {
	return &types.Binding{
		CommandID: "branch.place-at-selected-commit", Verb: "place-branch",
		Handler:           self.withItem(self.beginPlacementAtRemoteBranch),
		GetDisabledReason: self.require(self.singleItemSelected()),
		Description:       self.c.Tr.CreateMoveBranchHere,
		Tooltip:           self.c.Tr.PlaceBranchTooltip,
		OpensMenu:         true,
	}
}

func (self *BranchesController) beginPlacementAtLocalBranch(branch *models.Branch) error {
	ref := branch.FullRefName()
	if branch.DetachedHead && !isFullSelectedCommitOID(ref) {
		return errors.New(self.c.Tr.PlaceBranchSelectCommit)
	}
	return beginBranchPlacementAtRef(self.c, ref)
}

func (self *RemoteBranchesController) beginPlacementAtRemoteBranch(branch *models.RemoteBranch) error {
	return beginBranchPlacementAtRef(self.c, branch.FullRefName())
}

type branchPlacementRefTarget struct {
	c          *ControllerCommon
	git        *commands.GitCommand
	generation int
	ref        string
}

func beginBranchPlacementAtRef(c *ControllerCommon, ref string) error {
	target := &branchPlacementRefTarget{c: c, git: c.Git(), generation: c.State().GetRepoGeneration(), ref: ref}
	return target.resolve()
}

func (target *branchPlacementRefTarget) checkRepository() error {
	if target.c.Git() != target.git || target.c.State().GetRepoGeneration() != target.generation {
		return errors.New(target.c.Tr.PlaceBranchRepositoryChanged)
	}
	return nil
}

func (target *branchPlacementRefTarget) resolve() error {
	if err := target.checkRepository(); err != nil {
		return err
	}
	return target.c.WithWaitingStatus(target.c.Tr.PlaceBranchWorkingStatus, func(gocui.Task) error {
		if err := target.checkRepository(); err != nil {
			return err
		}
		oid, err := target.git.Branch.ResolveBranchActionCommit(target.ref)
		if err != nil {
			return err
		}
		target.c.OnUIThread(func() error {
			if err := target.checkRepository(); err != nil {
				return err
			}
			return (&branchPlacementFlow{c: target.c, git: target.git, oid: oid, validateRepository: target.checkRepository}).promptName()
		})
		return nil
	})
}
