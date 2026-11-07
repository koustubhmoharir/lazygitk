package controllers

// Fork: focusPanelTitle gives the pane title for a numbered pane jump's
// description. It was moved here out of jump_to_side_window_controller.go to
// ease upstream merges.

func (self *JumpToSideWindowController) focusPanelTitle(window string) string {
	switch window {
	case "status":
		return self.c.Tr.StatusTitle
	case "files":
		return self.c.Tr.FilesTitle
	case "branches":
		return self.c.Tr.BranchesTitle
	case "commits":
		return self.c.Tr.CommitsTitle
	case "stash":
		return self.c.Tr.StashTitle
	case "worktrees":
		return self.c.Tr.WorktreesTitle
	case "submodules":
		return self.c.Tr.SubmodulesTitle
	case "remotes":
		return self.c.Tr.RemotesTitle
	case "tags":
		return self.c.Tr.TagsTitle
	case "reflog":
		return self.c.Tr.ReflogCommitsTitle
	}
	return window
}
