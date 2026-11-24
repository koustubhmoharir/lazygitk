package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: the Files v menu chooses how to go into the selected entry, where Enter
// goes in by default (see fork_files_inspect.go). A conflicted file offers r
// (resolve, as Enter does), m (merge tool, as d m does) and d (focus the diff
// without resolving, as 0 does). A submodule entry offers s (enter, as Enter
// does) and d. Any other entry has no applicable route, so the prefix menu does
// nothing there. The default paths are in keymap/default_paths.go.
//
// These routes have their own command IDs so that the menu can omit the ones
// that do not apply. Their handlers are shared with the routes they mirror, and
// the mirrored routes keep their own keys everywhere.

func (self *FilesController) inspectMenuBindings() []*types.Binding {
	return []*types.Binding{
		{
			CommandID: "file.inspect.resolve", Verb: "resolve",
			Handler:           self.withItem(self.resolveConflictEntry),
			GetDisabledReason: self.require(self.singleItemSelected(self.inspectResolveDisabledReason)),
			Description:       self.c.Tr.InspectResolveConflict,
		},
		{
			CommandID: "file.inspect.submodule", Verb: "enter-submodule",
			Handler:           self.withItem(self.enterSubmoduleEntry),
			GetDisabledReason: self.require(self.singleItemSelected(self.inspectSubmoduleDisabledReason)),
			Description:       self.c.Tr.InspectEnterSubmodule,
		},
		{
			CommandID: "file.inspect.merge-tool", Verb: "open-merge-tool",
			Handler:           self.withItem(self.openMergeToolForEntry),
			GetDisabledReason: self.require(self.singleItemSelected(self.inspectResolveDisabledReason)),
			Description:       self.c.Tr.OpenMergeToolSelected,
		},
		{
			CommandID: "file.inspect.diff", Verb: "focus-diff",
			Handler:           self.withItem(self.focusDiffForEntry),
			GetDisabledReason: self.require(self.singleItemSelected(self.inspectDiffDisabledReason)),
			Description:       self.c.Tr.InspectShowDiff,
		},
	}
}

// inspectResolveDisabledReason applies to a conflicted file, the same test that
// Enter's resolution uses. Directories and unconflicted files have no route.
func (self *FilesController) inspectResolveDisabledReason(node *filetree.FileNode) *types.DisabledReason {
	if node.File == nil || !node.File.HasMergeConflicts {
		return self.inspectNotApplicable()
	}
	return nil
}

// inspectSubmoduleDisabledReason applies to a submodule entry that Enter enters.
// A conflicted submodule is resolved by Enter instead, so it is excluded.
func (self *FilesController) inspectSubmoduleDisabledReason(node *filetree.FileNode) *types.DisabledReason {
	if node.File == nil || node.File.HasMergeConflicts || !self.isSubmoduleEntry(node.File) {
		return self.inspectNotApplicable()
	}
	return nil
}

// inspectDiffDisabledReason applies to the entries that have a route in the menu:
// a conflicted file or a submodule entry.
func (self *FilesController) inspectDiffDisabledReason(node *filetree.FileNode) *types.DisabledReason {
	if node.File == nil || (!node.File.HasMergeConflicts && !self.isSubmoduleEntry(node.File)) {
		return self.inspectNotApplicable()
	}
	return nil
}

// inspectNotApplicable is inapplicable, so the v menu omits the route and a
// prefix with no applicable route does nothing at all.
func (self *FilesController) inspectNotApplicable() *types.DisabledReason {
	return &types.DisabledReason{Text: self.c.Tr.InspectNotApplicable, Inapplicable: true}
}

func (self *FilesController) resolveConflictEntry(node *filetree.FileNode) error {
	if node.File == nil {
		return nil
	}
	return self.resolveConflictedFile(node.File)
}

func (self *FilesController) enterSubmoduleEntry(node *filetree.FileNode) error {
	if node.File == nil {
		return nil
	}
	return self.enterSubmodule(node.File)
}

// openMergeToolForEntry runs the d m handler on the entry alone.
func (self *FilesController) openMergeToolForEntry(node *filetree.FileNode) error {
	return self.openMergeToolForSelected([]*filetree.FileNode{node})
}

// focusDiffForEntry is the focus-only route of 0, which only shows the diff.
func (self *FilesController) focusDiffForEntry(*filetree.FileNode) error {
	return focusMainView(self.c, self.context(), -1)
}
