package gui

// forkGitDeltaConfig returns the current repo's delta settings for the default
// delta renderer (config/fork_diff_renderers.go), or nothing before a repo is
// open.
func (gui *Gui) forkGitDeltaConfig() string {
	if gui.git == nil {
		return ""
	}
	return gui.git.Config.DeltaConfig()
}
