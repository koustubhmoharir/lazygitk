package controllers

import (
	"errors"
	"slices"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// Copy names at entry: neither a later model replacement nor range-selection
// change may redirect the branch(es) whose update is being reviewed.
func (self *BranchesController) fastForwardCapturedBranches(names []string) error {
	if len(names) == 0 {
		return errors.New(self.c.Tr.BranchSyncSelectOneLocalBranch)
	}
	names = slices.Clone(names)
	target := newBranchSyncTarget(self.c, branchSyncSource{name: names[0], selected: true})
	if len(names) == 1 {
		return target.inspect(func(info git_commands.BranchSyncInfo) error {
			flow := &branchSyncUpdateFlow{
				target: target, info: info, options: branchSyncUpdateDefaults(info), currentMode: info.HeadRef == info.FullRef,
				fastForwardOnly: true,
			}
			return flow.menu()
		})
	}
	return self.c.WithWaitingStatus(self.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := target.checkRepository(); err != nil {
			return err
		}
		members := make([]git_commands.BranchStackMember, 0, len(names))
		seen := map[string]bool{}
		for _, name := range names {
			if seen[name] {
				return git_commands.ErrBranchSyncChanged
			}
			seen[name] = true
			info, err := target.git.Sync.InspectBranchSync(name)
			if err != nil {
				return err
			}
			members = append(members, git_commands.BranchStackMember{Name: info.Name, OID: info.OID})
		}
		self.c.OnUIThread(func() error {
			if err := target.checkRepository(); err != nil {
				return err
			}
			batch := &branchStackUpdateFlow{target: target, members: members, fastForwardOnly: true}
			return batch.reviewNext()
		})
		return nil
	})
}
