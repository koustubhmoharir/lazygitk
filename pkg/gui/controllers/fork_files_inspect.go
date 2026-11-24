package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/keymap"
)

// Fork: InspectSelected is the Files inspection key's handler (see
// SwitchToFocusedMainViewController.WithInspect). It focuses the selected file's
// details like the upstream focus-diff key, but a conflicted file's details are its
// resolution flow, so it resolves exactly as Enter does. A submodule entry enters
// that submodule, as Enter does. It never toggles a directory. An explicitly
// configured focus-diff key keeps upstream's focus-only handler.
func (self *FilesController) InspectSelected() error {
	node := self.context().GetSelected()
	if node != nil && node.File != nil && !self.focusDiffIsExplicit() {
		if self.conflictNeedsResolutionDialog(node.File) {
			return self.resolveConflictedFile(node.File)
		}
		if self.isSubmoduleEntry(node.File) {
			return self.enterSubmodule(node.File)
		}
		if node.File.HasInlineMergeConflicts {
			return self.resolveConflictedFile(node.File)
		}
	}
	return focusMainView(self.c, self.context(), -1)
}

// resolveConflictedFile is Enter's resolution of a conflicted file, also the v
// menu's resolve route (fork_files_inspect_menu.go): the resolution menu where
// the conflict needs a dialog, otherwise the merge conflicts view.
func (self *FilesController) resolveConflictedFile(file *models.File) error {
	if self.conflictNeedsResolutionDialog(file) {
		return self.openConflictResolutionMenu(file)
	}
	return self.c.Helpers().MergeConflicts.SwitchToMerge(file)
}

// isSubmoduleEntry reports whether a file entry is a submodule.
func (self *FilesController) isSubmoduleEntry(file *models.File) bool {
	return file.IsSubmodule(self.c.Model().Submodules)
}

// enterSubmodule enters a submodule entry, also the v menu's submodule route.
func (self *FilesController) enterSubmodule(file *models.File) error {
	submoduleConfigs := self.c.Model().Submodules
	return self.c.Helpers().Repos.EnterSubmodule(file.SubmoduleConfig(submoduleConfigs))
}

func (self *FilesController) focusDiffIsExplicit() bool {
	_, explicit, _, err := keymap.ResolvePaths(self.c.UserConfig(), "navigation.focus-diff", "navigation.focus-diff", "files")
	return err == nil && explicit
}

// resolveKeyLabel names the effective key that resolves the selected conflict:
// the default inspection key, or Enter's file.enter route when focus-diff is
// explicitly configured and therefore only focuses the diff.
func (self *FilesController) resolveKeyLabel() string {
	if self.focusDiffIsExplicit() {
		return self.c.CommandKeyLabel("file.enter", "files")
	}
	return self.c.CommandKeyLabel("navigation.focus-diff", "files")
}
