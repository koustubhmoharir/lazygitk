package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/modes/filtering"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// Fork: w a, w d and w b show the selected commits with their ancestors or with
// their descendants, or a branch with its upstream. They are bound to Commits only;
// Reflog and Sub-commits keep upstream's behaviour. The filter itself is in
// helpers/fork_commit_scope_filter.go. w b from Branches is in
// fork_branch_scope_filter.go.

// addCommitScopeFilterBindings is the integration point in LocalCommitsController.
func (self *LocalCommitsController) addCommitScopeFilterBindings(bindings []*types.Binding) []*types.Binding {
	return append(bindings,
		self.commitScopeFilterBinding("filter.commit-ancestors", self.c.Tr.FilterCommitAncestors, false),
		self.commitScopeFilterBinding("filter.commit-descendants", self.c.Tr.FilterCommitDescendants, true),
		self.commitBranchScopeFilterBinding(),
	)
}

// commitBranchScopeFilterBinding is w b on a commit that local branches point at.
// One branch is applied directly; several are chosen from a menu.
func (self *LocalCommitsController) commitBranchScopeFilterBinding() *types.Binding {
	return &types.Binding{
		CommandID:   "filter.branch-and-upstream",
		Verb:        "filter",
		Description: self.c.Tr.FilterBranchAndUpstream,
		Handler:     self.applyBranchScopeFilterFromCommit,
		GetDisabledReason: func() *types.DisabledReason {
			commit := self.context().GetSelected()
			if commit == nil || !isFullSelectedCommitOID(commit.Hash()) {
				return &types.DisabledReason{Text: self.c.Tr.CommitScopeSelectCommits, Inapplicable: true}
			}
			if len(localBranchNamesAt(self.c.Model().Branches, commit.Hash())) == 0 {
				return &types.DisabledReason{Text: self.c.Tr.CommitScopeNoBranch, Inapplicable: true}
			}
			return nil
		},
	}
}

func (self *LocalCommitsController) applyBranchScopeFilterFromCommit() error {
	commit := self.context().GetSelected()
	if commit == nil {
		return nil
	}
	names := localBranchNamesAt(self.c.Model().Branches, commit.Hash())
	apply := func(name string) error {
		return self.c.Helpers().Mode.SetCommitScopeFilter(filtering.CommitScope{Branch: name}, commit.Hash())
	}
	if len(names) == 1 {
		return apply(names[0])
	}
	return self.c.Menu(types.CreateMenuOptions{
		Title: self.c.Tr.FilterBranchAndUpstream,
		Items: lo.Map(names, func(name string, _ int) *types.MenuItem {
			return &types.MenuItem{Label: name, OnPress: func() error { return apply(name) }}
		}),
	})
}

// localBranchNamesAt returns the names of the local branches whose tip is hash, in
// the model's order. A detached HEAD is not a branch.
func localBranchNamesAt(branches []*models.Branch, hash string) []string {
	names := []string{}
	for _, branch := range branches {
		if !branch.DetachedHead && branch.CommitHash == hash {
			names = append(names, branch.Name)
		}
	}
	return names
}

func (self *LocalCommitsController) commitScopeFilterBinding(commandID string, description string, descendants bool) *types.Binding {
	return &types.Binding{
		CommandID:   commandID,
		Verb:        "filter",
		Description: description,
		Handler:     func() error { return self.applyCommitScopeFilter(descendants) },
		GetDisabledReason: func() *types.DisabledReason {
			commits, _, _ := self.context().GetSelectedItems()
			if _, ok := forkCommitScopeHashes(commits); !ok {
				return &types.DisabledReason{Text: self.c.Tr.CommitScopeSelectCommits, Inapplicable: true}
			}
			return nil
		},
	}
}

func (self *LocalCommitsController) applyCommitScopeFilter(descendants bool) error {
	commits, _, _ := self.context().GetSelectedItems()
	hashes, ok := forkCommitScopeHashes(commits)
	if !ok {
		return nil
	}
	return self.c.Helpers().Mode.SetCommitScopeFilter(filtering.CommitScope{Hashes: hashes, Descendants: descendants}, self.context().GetSelectedCommitHash())
}

// forkCommitScopeHashes returns the hashes of commits, or false when there are
// none or one of them is not a real commit (a rebase todo, for example).
func forkCommitScopeHashes(commits []*models.Commit) ([]string, bool) {
	if len(commits) == 0 {
		return nil, false
	}
	hashes := make([]string, 0, len(commits))
	for _, commit := range commits {
		if commit == nil || !isFullSelectedCommitOID(commit.Hash()) {
			return nil, false
		}
		hashes = append(hashes, commit.Hash())
	}
	return hashes, true
}
