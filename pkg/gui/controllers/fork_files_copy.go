package controllers

import (
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: the flat `y` copy menu's file actions (see the copy table in
// docs-master/fork/Guide.md, Prefix menus). Each binding runs the same item as the
// nested Copy to clipboard menu, which keeps its items for the palette.
func (self *FilesController) copyBindings() []*types.Binding {
	return flatMenuItemBindings(self.copyMenuItems,
		flatMenuEntry{"clipboard.copy-file-name", self.c.Tr.CopyFileNameToClipboard},
		flatMenuEntry{"clipboard.copy-absolute-file-path", self.c.Tr.CopyAbsoluteFilePathToClipboard},
		flatMenuEntry{"clipboard.copy-working-file-diff", self.c.Tr.CopyFileDiffToClipboard},
		flatMenuEntry{"clipboard.copy-working-tree-diff", self.c.Tr.CopyAllFilesDiffToClipboard},
	)
}

// addFlatCopyBindings appends the flat copy actions to the upstream list.
func (self *FilesController) addFlatCopyBindings(bindings []*types.Binding) []*types.Binding {
	return append(bindings, self.copyBindings()...)
}

// Fork: the items are shared with the flat `y` copy bindings (see fork_files_copy.go).
func (self *FilesController) copyMenuItems() []*types.MenuItem {
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
			absPath, err := filepath.Abs(node.GetPath())
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
		CommandID: "clipboard.copy-working-file-diff",
		Verb:      "copy",
		Tooltip:   self.c.Tr.CopyFileDiffTooltip,
		OnPress: func() error {
			path := self.context().GetSelectedPath()
			hasStaged := self.hasPathStagedChanges(node)
			// Fork: literal pathspecs, so a name such as a*.txt copies only that
			// file's diff (fork_literal_pathspecs.go).
			diff, err := self.c.Git().Diff.GetLiteralPathsDiff(hasStaged, path)
			if err != nil {
				return err
			}
			if err := self.c.OS().CopyToClipboard(diff); err != nil {
				return err
			}
			self.c.Toast(self.c.Tr.FileDiffCopiedToast)
			return nil
		},
		DisabledReason: self.require(self.singleItemSelected(
			func(file *filetree.FileNode) *types.DisabledReason {
				if !node.GetHasStagedOrTrackedChanges() {
					return &types.DisabledReason{Text: self.c.Tr.NoContentToCopyError}
				}
				return nil
			},
		))(),
		Keys: menuKey('s'),
	}
	copyAllDiff := &types.MenuItem{
		Label:     self.c.Tr.CopyAllFilesDiff,
		CommandID: "clipboard.copy-working-tree-diff",
		Verb:      "copy",
		Tooltip:   self.c.Tr.CopyFileDiffTooltip,
		OnPress: func() error {
			hasStaged := self.c.Helpers().WorkingTree.AnyStagedFiles()
			diff, err := self.c.Git().Diff.GetDiff(hasStaged, "--")
			if err != nil {
				return err
			}
			if err := self.c.OS().CopyToClipboard(diff); err != nil {
				return err
			}
			self.c.Toast(self.c.Tr.AllFilesDiffCopiedToast)
			return nil
		},
		DisabledReason: self.require(
			func() *types.DisabledReason {
				if !self.anyStagedOrTrackedFile() {
					return &types.DisabledReason{Text: self.c.Tr.NoContentToCopyError}
				}
				return nil
			},
		)(),
		Keys: menuKey('a'),
	}

	return []*types.MenuItem{
		copyNameItem,
		copyRelativePathItem,
		copyAbsolutePathItem,
		copyFileDiffItem,
		copyAllDiff,
	}
}
