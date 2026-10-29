package controllers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/modes/filtering"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// Fork: w a, w d and w b show the selected commits with their ancestors or with
// their descendants, or a branch, remote branch or tag with its history. They are
// bound to Commits only; Reflog and Sub-commits keep upstream's behaviour. The filter
// itself is in helpers/fork_commit_scope_filter.go. w b from Branches, Remote branches
// and Tags, and the shared ref logic, are in fork_branch_scope_filter.go and
// fork_ref_scope_filter.go.

// addCommitScopeFilterBindings is the integration point in LocalCommitsController.
func (self *LocalCommitsController) addCommitScopeFilterBindings(bindings []*types.Binding) []*types.Binding {
	return append(bindings,
		self.commitScopeFilterBinding("filter.commit-ancestors", self.c.Tr.FilterCommitAncestors, false),
		self.commitScopeFilterBinding("filter.commit-descendants", self.c.Tr.FilterCommitDescendants, true),
		self.commitRefScopeFilterBinding(),
	)
}

// commitRefScopeFilterBinding is w b on a commit that a local branch, remote branch or
// tag points at. The refs are looked up when it runs, so the menu is never stale.
func (self *LocalCommitsController) commitRefScopeFilterBinding() *types.Binding {
	return &types.Binding{
		CommandID:   "filter.ref",
		Verb:        "filter",
		Description: self.c.Tr.FilterRef,
		Handler:     self.applyRefScopeFilterFromCommit,
		GetDisabledReason: func() *types.DisabledReason {
			commit := self.context().GetSelected()
			if commit == nil || !isFullSelectedCommitOID(commit.Hash()) {
				return &types.DisabledReason{Text: self.c.Tr.CommitScopeSelectCommits, Inapplicable: true}
			}
			if !commitHasRefScope(commit, self.c.Model().Branches, self.c.Model().Remotes) {
				return &types.DisabledReason{Text: self.c.Tr.CommitScopeNoRef, Inapplicable: true}
			}
			return nil
		},
	}
}

// applyRefScopeFilterFromCommit lists the refs at the selected commit. One ref applies
// directly; several are chosen from a menu, in the order RefsAtCommit gives them.
func (self *LocalCommitsController) applyRefScopeFilterFromCommit() error {
	commit := self.context().GetSelected()
	if commit == nil {
		return nil
	}
	refs, err := self.c.Git().Commit.RefsAtCommit(commit.Hash())
	if err != nil {
		return err
	}
	switch len(refs) {
	case 0:
		return errors.New(self.c.Tr.CommitScopeNoRef)
	case 1:
		return applyRefScope(self.c, refs[0], commit.Hash())
	}
	return self.c.Menu(types.CreateMenuOptions{
		Title: self.c.Tr.FilterRef,
		Items: lo.Map(refs, func(ref string, _ int) *types.MenuItem {
			return &types.MenuItem{
				Label:   refScopeLabel(self.c, ref),
				OnPress: func() error { return applyRefScope(self.c, ref, commit.Hash()) },
			}
		}),
	})
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
