package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/modes/filtering"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: w a and w d show the selected commits with their ancestors or with their
// descendants. They are bound to Commits only; Reflog and Sub-commits keep
// upstream's behaviour. The filter itself is in helpers/fork_commit_scope_filter.go.

// addCommitScopeFilterBindings is the integration point in LocalCommitsController.
func (self *LocalCommitsController) addCommitScopeFilterBindings(bindings []*types.Binding) []*types.Binding {
	return append(bindings,
		self.commitScopeFilterBinding("filter.commit-ancestors", self.c.Tr.FilterCommitAncestors, false),
		self.commitScopeFilterBinding("filter.commit-descendants", self.c.Tr.FilterCommitDescendants, true),
	)
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
	return self.c.Helpers().Mode.SetCommitScopeFilter(filtering.CommitScope{Hashes: hashes, Descendants: descendants})
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
