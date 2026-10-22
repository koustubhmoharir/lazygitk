package controllers

import (
	"errors"
	"slices"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// inspectStack captures native ancestry independently of the loaded history.
// The selected root is first; returned members are not mutation approvals.
func (target *branchSyncTarget) inspectStack(info git_commands.BranchSyncInfo, callback func([]git_commands.BranchStackMember) error) error {
	if err := target.checkRepository(); err != nil {
		return err
	}
	if info.Name != target.source.name {
		return git_commands.ErrBranchSyncChanged
	}
	mainBranches := target.c.Model().MainBranches
	mainNames := slices.Clone(target.c.UserConfig().Git.MainBranches)
	if mainBranches == nil {
		return errors.New("main branch information is not loaded; reopen stack review after refreshing")
	}
	return target.c.WithWaitingStatus(target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := target.checkRepository(); err != nil {
			return err
		}
		below, err := target.git.Sync.InspectBranchStack(info.Name, info.OID, slices.Clone(mainBranches.Get()))
		if err != nil {
			return err
		}
		// A local main can be ahead of its remote main ref without becoming a
		// topic-stack member. Retain upstream's configured-name exclusion.
		below = slices.DeleteFunc(below, func(member git_commands.BranchStackMember) bool {
			return slices.Contains(mainNames, member.Name)
		})
		members := append([]git_commands.BranchStackMember{{Name: info.Name, OID: info.OID}}, below...)
		target.c.OnUIThread(func() error {
			if err := target.checkRepository(); err != nil {
				return err
			}
			return callback(members)
		})
		return nil
	})
}
