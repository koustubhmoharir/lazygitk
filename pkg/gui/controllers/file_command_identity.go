package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
)

// The upstream primary-file handler opens a conflict picker for a single
// dialog-only conflict instead of toggling its staging. Describe that existing
// dispatch using the same predicate, without embedding selected paths in IDs.
func filePrimaryCommandIdentity(nodes []*filetree.FileNode, needsDialog func(*models.File) bool) (string, string) {
	if len(nodes) == 1 && needsDialog(nodes[0].File) {
		return "conflict.options-menu", "conflict-options"
	}
	return "file.toggle-stage", "toggle-stage"
}

func (self *FilesController) primaryFileCommandIdentity() (string, string) {
	nodes, _, _ := self.context().GetSelectedItems()
	return filePrimaryCommandIdentity(nodes, self.conflictNeedsResolutionDialog)
}

func (self *FilesController) primaryFileCommandID() string {
	id, _ := self.primaryFileCommandIdentity()
	return id
}

func (self *FilesController) primaryFileVerb() string {
	_, verb := self.primaryFileCommandIdentity()
	return verb
}
