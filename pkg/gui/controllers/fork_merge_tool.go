package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// Fork: `d m` opens the external merge tool for the selected conflicted files,
// in Files and in the conflicts view. It never lets Git guess a tool; see
// helpers/working_tree_helper.go OpenMergeTool and docs-master/fork/Guide.md.
// The path itself is the default key route in keymap/default_paths.go.

func (self *FilesController) mergeToolBindings() []*types.Binding {
	return []*types.Binding{
		{
			CommandID: "conflict.open-merge-tool-selected", Verb: "open-merge-tool",
			Handler:           self.withItems(self.openMergeToolForSelected),
			GetDisabledReason: self.require(self.itemsSelected(self.canOpenMergeToolForSelected)),
			Description:       self.c.Tr.OpenMergeToolSelected,
		},
	}
}

// openMergeToolForSelected opens the tool on the conflicted files among the
// selection. Files that are not conflicted are ignored.
func (self *FilesController) openMergeToolForSelected(nodes []*filetree.FileNode) error {
	return self.c.Helpers().WorkingTree.OpenMergeTool(conflictedFilePaths(nodes))
}

func (self *FilesController) canOpenMergeToolForSelected(nodes []*filetree.FileNode) *types.DisabledReason {
	if len(conflictedFilePaths(nodes)) == 0 {
		return &types.DisabledReason{Text: self.c.Tr.NoConflictedFilesSelected, Inapplicable: true}
	}
	return nil
}

// conflictedFilePaths flattens the selection to files and keeps those with
// merge conflicts of any kind, so delete/modify and add/add conflicts count.
func conflictedFilePaths(nodes []*filetree.FileNode) []string {
	return lo.FilterMap(flattenSelectedNodesToFiles(nodes), func(node *filetree.FileNode, _ int) (string, bool) {
		return node.GetPath(), node.File != nil && node.File.HasMergeConflicts
	})
}

// mergeToolBindings is shared by Files and the conflicts view. The conflicts
// view acts on the file it resolves, which the Files binding can't see.
func (self *MergeConflictsController) mergeToolBindings() []*types.Binding {
	return []*types.Binding{
		{
			CommandID: "conflict.open-merge-tool-selected", Verb: "open-merge-tool",
			Handler:     self.openMergeTool,
			Description: self.c.Tr.OpenMergeToolSelected,
		},
	}
}

func (self *MergeConflictsController) openMergeTool() error {
	state := self.context().GetState()
	if state == nil {
		return nil
	}
	return self.c.Helpers().WorkingTree.OpenMergeTool([]string{state.GetPath()})
}
