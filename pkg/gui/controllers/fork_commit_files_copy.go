package controllers

import "github.com/jesseduffield/lazygit/pkg/gui/types"

// Fork: the flat `y` copy menu's commit file actions (see the copy table in
// docs-master/fork/Guide.md, Prefix menus). Each binding runs the same item as the
// nested Copy to clipboard menu, which keeps its items for the palette.
func (self *CommitFilesController) copyBindings() []*types.Binding {
	return flatMenuItemBindings(self.copyMenuItems,
		flatMenuEntry{"clipboard.copy-file-name", self.c.Tr.CopyFileNameToClipboard},
		flatMenuEntry{"clipboard.copy-absolute-file-path", self.c.Tr.CopyAbsoluteFilePathToClipboard},
		flatMenuEntry{"clipboard.copy-history-file-diff", self.c.Tr.CopyFileDiffToClipboard},
		flatMenuEntry{"clipboard.copy-history-file-content", self.c.Tr.CopyFileContentToClipboard},
	)
}

// addFlatCopyBindings appends the flat copy actions to the upstream list.
func (self *CommitFilesController) addFlatCopyBindings(bindings []*types.Binding) []*types.Binding {
	return append(bindings, self.copyBindings()...)
}
