package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: multi-select mode in Commits (item 24). Space toggles a commit in the explicit
// set, or the whole run in progress, and Escape clears the filter, then the set. The
// state is in context/fork_commit_multi_select.go. Commits, Sub-commits and Reflog share
// Space (commitSelectionToggleBinding); Commits inspects with Enter and v as the others do
// (see fork_command_projection.go).

// forkSelectionHashes returns the hashes that a command on the selection of source acts
// on. In Commits with a set or a run, that is the whole selection, including the members
// the loaded list does not show. Otherwise it is the selected items, which is the cursor
// commit in single-select mode, as upstream has it. A nil item gives an empty hash, which
// the callers' full-ID checks refuse.
func forkSelectionHashes(source ContainsCommits) []string {
	if set, ok := source.(context.CommitSetHolder); ok && set.MultiSelectActive() {
		return set.SelectedCommitHashes()
	}
	selected, _, _ := source.GetSelectedItems()
	hashes := make([]string, len(selected))
	for i, commit := range selected {
		if commit != nil {
			hashes[i] = commit.Hash()
		}
	}
	return hashes
}

// commitSelectionScoped lists the Commits commands that act on the selected commit, the
// cursor or an index range, so they cannot act on an explicit set. While a set exists,
// guardCommitSelectionBindings disables them. Classification:
//
// Enabled with a set, because they take the set or do not use the selection: g c
// (combine the set), c p (picks the set, in ancestry order), w a and w d (the set, see
// forkSelectionHashes), Space, v (inspect the cursor commit), navigation, search, log
// options, w o, /, g i (the
// first merge-base commit, not the cursor), autosquash, open PR, c clear, g m n (moves
// commits to a new branch from the checked-out branch), t l s (moves the cursor and
// starts a run) and focus-diff (it only shows the main view).
//
// Disabled: the y copy actions of a commit, s d, s c, s h/s m/s s, n b, r, g r b, d w,
// d p, d c, the external diff key, n (new branch), w b, n t, and the single-commit and
// range edits (reword, drop, squash, fixup, edit, pick, move, amend, attributes, revert,
// fixup creation, set fixup message, mark rebase base), and the bisect options.
//
// The bisect options (g b) are guarded from BisectController, and y i, which is bound in
// gui/keybindings.go, from getCopySelectedSideContextItemToClipboardDisabledReason. v
// (navigation.open-diff-files) stays enabled: it inspects the cursor commit, which the
// gutter marks, as the main view shows it.
var commitSelectionScoped = map[string]bool{
	// LocalCommitsController
	"commit.squash-down":          true,
	"commit.mark-fixup":           true,
	"commit.set-fixup-message":    true,
	"commit.reword":               true,
	"commit.reword-with-editor":   true,
	"commit.drop":                 true,
	"commit.edit":                 true,
	"commit.mark-pick":            true,
	"commit.create-fixup-menu":    true,
	"commit.move-down":            true,
	"commit.move-up":              true,
	"commit.amend-selected":       true,
	"commit.edit-attributes-menu": true,
	"commit.revert":               true,
	"commit.mark-rebase-base":     true,
	"tag.create-at-selection":     true,
	"filter.ref":                  true,
	// BisectController
	"bisect.options-menu": true,
	// BasicCommitsController
	"commit.checkout-menu":                       true,
	"commit.checkout-detached":                   true,
	"commit.copy-attribute-menu":                 true,
	"commit.open-browser":                        true,
	"branch.create-and-checkout":                 true,
	"branch.place-at-selected-commit":            true,
	"branch.rebase-current-onto-selected-commit": true,
	"branch.reset-menu":                          true,
	"branch.reset.hard":                          true,
	"branch.reset.mixed":                         true,
	"branch.reset.soft":                          true,
	"commit.select-current-branch-history":       true,
	"worktree.create-menu":                       true,
	"diff.commit.external":                       true,
	"diff.commit.worktree":                       true,
	"diff.commit.previous":                       true,
	"diff.commit.selected-commit":                true,
	"clipboard.copy-commit-hash":                 true,
	"clipboard.copy-commit-subject":              true,
	"clipboard.copy-commit-message":              true,
	"clipboard.copy-commit-body":                 true,
	"clipboard.copy-commit-url":                  true,
	"clipboard.copy-commit-diff":                 true,
	"clipboard.copy-commit-author":               true,
	"clipboard.copy-commit-tags":                 true,
}

// guardCommitSelectionBindings is the integration point for the Commits selection scope,
// applied to the bindings of LocalCommitsController and of BasicCommitsController.
// Sub-commits and Reflog share BasicCommitsController and have no set, so they are
// returned unchanged. A disabled reason from here takes precedence over the command's
// own reasons. It is not Inapplicable, so the default key shows it as a toast.
func guardCommitSelectionBindings(commits ContainsCommits, clearFirst string, bindings []*types.Binding) []*types.Binding {
	set, ok := commits.(context.CommitSetHolder)
	if !ok {
		return bindings
	}
	for _, binding := range bindings {
		if !commitSelectionScoped[binding.CommandID] {
			continue
		}
		upstream := binding.GetDisabledReason
		binding.GetDisabledReason = func() *types.DisabledReason {
			if set.HasCommitSet() {
				return &types.DisabledReason{Text: clearFirst}
			}
			if upstream != nil {
				return upstream()
			}
			return nil
		}
	}
	return bindings
}

// commitSelectionToggleBinding is Space in the three commit lists: Commits, Sub-commits and
// Reflog. It is the integration point in BasicCommitsController, which all three share.
func (self *BasicCommitsController) commitSelectionToggleBinding() *types.Binding {
	return &types.Binding{
		CommandID:   "selection.commit.toggle",
		Verb:        "toggle-selection",
		Description: self.c.Tr.ToggleCommitSelection,
		Handler: func() error {
			if set, ok := self.context.(context.CommitSetHolder); ok {
				set.ToggleCommitSelection()
			}
			self.context.HandleRender()
			return nil
		},
	}
}

// forkCommitSelectionEscape is the Escape step for the commit lists that comes before
// upstream's (see QuitActions.Escape). It reports whether it handled the key. In Commits,
// Escape cancels a lone Shift run first; then it clears the filter, which keeps the set and
// the cursor; then it clears the set, without adding the run in progress to it. Sub-commits
// and Reflog have no filter of this kind, so their Escape cancels a lone run, then clears
// the set, and only then does upstream's behaviour apply (range, search, parent view).
func (self *QuitActions) forkCommitSelectionEscape() (bool, error) {
	commits, ok := self.forkCommitsContextIfCurrent()
	if !ok {
		return false, nil
	}
	// A lone Shift run is cancelled first, as upstream cancels a range.
	if commits.IsSelectingRange() && !commits.HasCommitSet() {
		commits.ClearMultiSelection()
		self.c.PostRefreshUpdate(commits)
		return true, nil
	}
	if self.commitFilterEscape(commits) {
		return true, self.c.Helpers().Mode.ExitFilterMode()
	}
	if commits.MultiSelectActive() {
		commits.ClearMultiSelection()
		self.c.PostRefreshUpdate(commits)
		return true, nil
	}
	return false, nil
}

// forkCommitSelectionEscapeEnabled and forkCommitSelectionEscapeDescription are the
// matching parts of EscapeEnabled and EscapeDescription.
func (self *QuitActions) forkCommitSelectionEscapeEnabled() bool {
	commits, ok := self.forkCommitsContextIfCurrent()
	return ok && (self.commitFilterEscape(commits) || commits.MultiSelectActive())
}

func (self *QuitActions) forkCommitSelectionEscapeDescription() (string, bool) {
	commits, ok := self.forkCommitsContextIfCurrent()
	if !ok {
		return "", false
	}
	if commits.IsSelectingRange() && !commits.HasCommitSet() {
		return self.c.Tr.ClearCommitSelection, true
	}
	if self.commitFilterEscape(commits) {
		return self.c.Tr.ExitFilterMode, true
	}
	if commits.MultiSelectActive() {
		return self.c.Tr.ClearCommitSelection, true
	}
	return "", false
}

// commitFilterEscape reports whether Escape exits the Commits filter. The filter is Commits'
// own (path and author), so it is not offered from the other lists.
func (self *QuitActions) commitFilterEscape(commits context.CommitSetContext) bool {
	return commits.GetKey() == context.LOCAL_COMMITS_CONTEXT_KEY && self.c.Modes().Filtering.Active()
}

// forkCommitsContextIfCurrent returns the commit list when it is the focused one and holds a
// set: Commits, Sub-commits or Reflog.
func (self *QuitActions) forkCommitsContextIfCurrent() (context.CommitSetContext, bool) {
	switch self.c.Context().Current().GetKey() {
	case context.LOCAL_COMMITS_CONTEXT_KEY:
		return self.c.Contexts().LocalCommits, true
	case context.SUB_COMMITS_CONTEXT_KEY:
		return self.c.Contexts().SubCommits, true
	case context.REFLOG_COMMITS_CONTEXT_KEY:
		return self.c.Contexts().ReflogCommits, true
	}
	return nil, false
}
