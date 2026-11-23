package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// Fork: chooseCommitTipBranch asks which local branch, among several whose tip
// is the selected commit, to push or pull. The checked-out branch comes first
// and is the default. Each item captures its branch name and then runs the
// ordinary flow for that branch as a selected source. A branch with a push or
// pull already in progress is disabled with its own reason.
func (self *SyncController) chooseCommitTipBranch(sources []branchSyncSource, push bool) error {
	title, command, verb, run := self.c.Tr.BranchSyncPullChooseTitle, "branch.pull-choose", "pull", self.pullBranchSource
	if push {
		title, command, verb, run = self.c.Tr.BranchSyncPushChooseTitle, "branch.push-choose", "push", self.pushBranchSource
	}
	items := lo.Map(sources, func(source branchSyncSource, _ int) *types.MenuItem {
		name := source.name
		return &types.MenuItem{
			CommandID:      command,
			Verb:           verb,
			Label:          name,
			DisabledReason: self.branchSyncOperationDisabledReason(name),
			OnPress: func() error {
				return run(branchSyncSource{name: name, selected: true})
			},
		}
	})
	return self.c.Menu(types.CreateMenuOptions{
		Title:  title,
		Prompt: self.c.Tr.BranchSyncCommitTipPrompt,
		Items:  items,
	})
}
