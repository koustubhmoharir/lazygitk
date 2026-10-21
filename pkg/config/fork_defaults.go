package config

// applyForkDefaults keeps fork policy separate from upstream's default table.
// It must run before loading user or repository config, never after: these
// defaults make mutations explicit, but are not enforced prohibitions.
func applyForkDefaults(config *UserConfig) *UserConfig {
	config.Git.AutoFetch = false
	config.Git.AutoRefresh = false
	config.Git.AutoForwardBranches = "none"
	config.Git.AutoStageResolvedConflicts = false
	// Reserve provisional keys for direct comparisons. The keymap milestone
	// will rebind displaced commands; they remain available in the palette.
	// This overlay changes defaults only, so explicitly loaded bindings remain authoritative.
	config.Keybinding.Universal.DiffWithWorkingTree = Keybinding{"w", "W"}
	config.Keybinding.Universal.DiffWithPreviousCommit = Keybinding{"v", "V"}
	config.Keybinding.Universal.DiffWithWorkingFile = nil
	config.Keybinding.Universal.NewWorktree = nil
	config.Keybinding.Universal.DiffingMenu = nil
	config.Keybinding.Universal.DiffingMenuAlt = nil
	config.Keybinding.Universal.ToggleRangeSelect = nil
	config.Keybinding.Files.CommitChangesWithoutHook = nil
	config.Keybinding.Commits.PasteCommits = nil
	return config
}
