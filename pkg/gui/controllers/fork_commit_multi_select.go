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
