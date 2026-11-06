package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

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

// Fork: the items are shared with the flat `y` copy bindings (see fork_commit_files_copy.go).
func (self *CommitFilesController) copyMenuItems() []*types.MenuItem {
	node := self.context().GetSelected()

	copyNameItem := &types.MenuItem{
		Label:     self.c.Tr.CopyFileName,
		CommandID: "clipboard.copy-file-name",
		Verb:      "copy",
		OnPress: func() error {
			if err := self.c.OS().CopyToClipboard(node.Name()); err != nil {
				return err
			}
			self.c.Toast(self.c.Tr.FileNameCopiedToast)
			return nil
		},
		DisabledReason: self.require(self.singleItemSelected())(),
		Keys:           menuKey('n'),
	}
	copyRelativePathItem := &types.MenuItem{
		Label:     self.c.Tr.CopyRelativeFilePath,
		CommandID: "clipboard.copy-file-path",
		Verb:      "copy",
		OnPress: func() error {
			if err := self.c.OS().CopyToClipboard(node.GetPath()); err != nil {
				return err
			}
			self.c.Toast(self.c.Tr.FilePathCopiedToast)
			return nil
		},
		DisabledReason: self.require(self.singleItemSelected())(),
		Keys:           menuKey('p'),
	}
	copyAbsolutePathItem := &types.MenuItem{
		Label:     self.c.Tr.CopyAbsoluteFilePath,
		CommandID: "clipboard.copy-absolute-file-path",
		Verb:      "copy",
		OnPress: func() error {
			absPath, err := self.c.Helpers().Diff.AbsolutePathOfFileOfDiff(node.GetPath())
			if err != nil {
				return err
			}
			if err := self.c.OS().CopyToClipboard(absPath); err != nil {
				return err
			}
			self.c.Toast(self.c.Tr.FilePathCopiedToast)
			return nil
		},
		DisabledReason: self.require(self.singleItemSelected())(),
		Keys:           menuKey('P'),
	}
	copyFileDiffItem := &types.MenuItem{
		Label:     self.c.Tr.CopySelectedDiff,
		CommandID: "clipboard.copy-history-file-diff",
		Verb:      "copy",
		OnPress: func() error {
			return self.copyDiffToClipboard(self.pathsForDiff(node), self.c.Tr.FileDiffCopiedToast)
		},
		DisabledReason: self.require(self.singleItemSelected())(),
		Keys:           menuKey('s'),
	}
	copyAllDiff := &types.MenuItem{
		Label:     self.c.Tr.CopyAllFilesDiff,
		CommandID: "clipboard.copy-history-diff",
		Verb:      "copy",
		OnPress: func() error {
			return self.copyDiffToClipboard([]string{"."}, self.c.Tr.AllFilesDiffCopiedToast)
		},
		DisabledReason: self.require(self.itemsSelected())(),
		Keys:           menuKey('a'),
	}
	copyFileContentItem := &types.MenuItem{
		Label:     self.c.Tr.CopyFileContent,
		CommandID: "clipboard.copy-history-file-content",
		Verb:      "copy",
		OnPress: func() error {
			if err := self.copyFileContentToClipboard(node.GetPath()); err != nil {
				return err
			}
			self.c.Toast(self.c.Tr.FileContentCopiedToast)
			return nil
		},
		DisabledReason: self.require(self.singleItemSelected(
			func(node *filetree.CommitFileNode) *types.DisabledReason {
				if !node.IsFile() {
					return &types.DisabledReason{
						Text:             self.c.Tr.ErrCannotCopyContentOfDirectory,
						ShowErrorInPanel: true,
					}
				}
				return nil
			}))(),
		Keys: menuKey('c'),
	}

	return []*types.MenuItem{
		copyNameItem,
		copyRelativePathItem,
		copyAbsolutePathItem,
		copyFileDiffItem,
		copyAllDiff,
		copyFileContentItem,
	}
}
