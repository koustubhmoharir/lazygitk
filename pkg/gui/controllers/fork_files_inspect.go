package controllers

// Fork: InspectSelected is the Files inspection key's handler (see
// SwitchToFocusedMainViewController.WithInspect). It focuses the selected file's
// details like the upstream focus-diff key, but a conflicted file's details are its
// resolution flow, so it resolves exactly as Enter does. It never enters a submodule
// or toggles a directory.
func (self *FilesController) InspectSelected() error {
	node := self.context().GetSelected()
	if node != nil && node.File != nil {
		if self.conflictNeedsResolutionDialog(node.File) {
			return self.openConflictResolutionMenu(node.File)
		}
		if node.File.HasInlineMergeConflicts {
			return self.switchToMerge()
		}
	}
	return focusMainView(self.c, self.context(), -1)
}
