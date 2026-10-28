package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: multi-select mode in Commits (item 24). Space toggles a commit in the explicit
// set, or the whole run in progress, and Escape clears the filter, then the set. The
// state is in context/fork_commit_multi_select.go. Only Commits has these routes:
// Reflog and Sub-commits keep Space as inspect, and Space in Commits is no longer the
// inspect route (see fork_command_projection.go).

// commitSelectionScoped lists the Commits commands that act on the selected commit, the
// cursor or an index range, so they cannot act on an explicit set. While a set exists,
// guardCommitSelectionBindings disables them. Classification:
//
// Enabled with a set, because they take the set or do not use the selection: c y and
// c x (mark and unmark the set), g c (combine the set), c p (the set is the source list,
// in list order), w a and w d (GetSelectedItems), Space, v (inspect the cursor commit), navigation, search, log
// options, w o, /, g i (the first merge-base commit, not the cursor), autosquash, open
// PR, c clear, g m n (moves commits to a new branch from the checked-out branch), t l s
// (moves the cursor and starts a run) and focus-diff (it only shows the main view).
//
// Disabled: the y copy actions of a commit, s d, s c, s h/s m/s s, n b, r, g r b, d w,
// d p, d c, the external diff key, n (new branch), w b, C (copy for cherry-pick, which
// copies by index), n t, and the single-commit and range edits (reword, drop, squash,
// fixup, edit, pick, move, amend, attributes, revert, fixup creation, set fixup message,
// mark rebase base), and the bisect options.
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
	"filter.branch-and-upstream":  true,
	// BisectController
	"bisect.options-menu": true,
	// BasicCommitsController
	"commit.checkout-menu":                       true,
	"commit.checkout-detached":                   true,
	"commit.copy-attribute-menu":                 true,
	"commit.open-browser":                        true,
	"commit.copy-for-cherry-pick":                true,
	"branch.create-and-checkout":                 true,
	"branch.place-at-selected-commit":            true,
	"branch.rebase-current-onto-selected-commit": true,
	"branch.reset-menu":                          true,
	"branch.reset.hard":                          true,
	"branch.reset.mixed":                         true,
	"branch.reset.soft":                          true,
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
	local, ok := commits.(*context.LocalCommitsContext)
	if !ok {
		return bindings
	}
	for _, binding := range bindings {
		if !commitSelectionScoped[binding.CommandID] {
			continue
		}
		upstream := binding.GetDisabledReason
		binding.GetDisabledReason = func() *types.DisabledReason {
			if local.HasCommitSet() {
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

// addCommitMultiSelectBindings is the integration point in LocalCommitsController.
func (self *LocalCommitsController) addCommitMultiSelectBindings(bindings []*types.Binding) []*types.Binding {
	return append(bindings, &types.Binding{
		CommandID:   "selection.commit.toggle",
		Verb:        "toggle-selection",
		Description: self.c.Tr.ToggleCommitSelection,
		Handler:     self.toggleCommitSelection,
	})
}

func (self *LocalCommitsController) toggleCommitSelection() error {
	self.context().ToggleCommitSelection()
	self.context().HandleRender()
	return nil
}

// forkCommitSelectionEscape is the Escape step for Commits that comes before upstream's
// (see QuitActions.Escape). It reports whether it handled the key. In Commits, Escape
// cancels a lone Shift run first; then it clears the filter, which keeps the set and the
// cursor; then it clears the set, without adding the run in progress to it.
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
	if self.c.Modes().Filtering.Active() {
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
	return ok && (self.c.Modes().Filtering.Active() || commits.MultiSelectActive())
}

func (self *QuitActions) forkCommitSelectionEscapeDescription() (string, bool) {
	commits, ok := self.forkCommitsContextIfCurrent()
	if !ok {
		return "", false
	}
	if commits.IsSelectingRange() && !commits.HasCommitSet() {
		return self.c.Tr.ClearCommitSelection, true
	}
	if self.c.Modes().Filtering.Active() {
		return self.c.Tr.ExitFilterMode, true
	}
	if commits.MultiSelectActive() {
		return self.c.Tr.ClearCommitSelection, true
	}
	return "", false
}

// forkCommitsContextIfCurrent returns the Commits context when it is the focused one.
func (self *QuitActions) forkCommitsContextIfCurrent() (*context.LocalCommitsContext, bool) {
	if self.c.Context().Current().GetKey() != context.LOCAL_COMMITS_CONTEXT_KEY {
		return nil, false
	}
	return self.c.Contexts().LocalCommits, true
}
