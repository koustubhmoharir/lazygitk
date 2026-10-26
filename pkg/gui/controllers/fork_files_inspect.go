package controllers

import "github.com/jesseduffield/lazygit/pkg/gui/keymap"

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
			return self.openConflictResolutionMenu(node.File)
		}
		submoduleConfigs := self.c.Model().Submodules
		if node.File.IsSubmodule(submoduleConfigs) {
			return self.c.Helpers().Repos.EnterSubmodule(node.File.SubmoduleConfig(submoduleConfigs))
		}
		if node.File.HasInlineMergeConflicts {
			return self.switchToMerge()
		}
	}
	return focusMainView(self.c, self.context(), -1)
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
