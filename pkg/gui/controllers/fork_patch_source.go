package controllers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: a patch built from a commit records that commit as its source. The source-editing
// actions of the custom patch menu rewrite that commit, so they need its index in the
// Commits list. When the list does not contain the source (a filter hides it, or a refresh
// after an external rewrite removed it), getPatchCommitIndex returns -1. The actions then
// refuse with PatchSourceNotInCommits, and the menu item shows it as its disabled reason.
// The check runs in each handler too, because the list can change while the menu is open.

// patchSourceCommitIndex returns the source's index, or the refusal when it is missing.
func (self *CustomPatchOptionsMenuAction) patchSourceCommitIndex() (int, error) {
	index := self.getPatchCommitIndex()
	if index < 0 {
		return index, errors.New(self.c.Tr.PatchSourceNotInCommits)
	}
	return index, nil
}

// patchSourceDisabledReason is the reason the source-editing items are disabled when the
// source is missing, or nil when it is in the list.
func (self *CustomPatchOptionsMenuAction) patchSourceDisabledReason() *types.DisabledReason {
	if self.getPatchCommitIndex() < 0 {
		return &types.DisabledReason{Text: self.c.Tr.PatchSourceNotInCommits}
	}
	return nil
}
