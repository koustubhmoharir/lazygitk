package config

// applyForkDefaults keeps fork policy separate from upstream's default table.
// It must run before loading user or repository config, never after: these
// defaults make mutations explicit, but are not enforced prohibitions.
func applyForkDefaults(config *UserConfig) *UserConfig {
	config.Git.AutoFetch = false
	config.Git.AutoRefresh = false
	config.Git.AutoForwardBranches = "none"
	config.Git.AutoStageResolvedConflicts = false
	// Include all refs in the history by default, while preserving config overrides.
	config.Git.Log.ShowWholeGraph = true
	applyForkEditorDefaults(config)
	// Search result navigation must not consume normal pane command letters.
	config.Keybinding.Universal.NextMatch = Keybinding{"<f3>"}
	config.Keybinding.Universal.PrevMatch = Keybinding{"<shift+f3>"}
	config.Keybinding.Universal.PrevBlock = Keybinding{"h"}
	config.Keybinding.Universal.NextBlock = Keybinding{"l"}
	config.Keybinding.Universal.PrevBlockAlt = nil
	config.Keybinding.Universal.NextBlockAlt = nil
	config.Keybinding.Universal.PrevBlockAlt2 = nil
	config.Keybinding.Universal.NextBlockAlt2 = nil
	config.Keybinding.Universal.PrevTab = Keybinding{"<backtab>"}
	config.Keybinding.Universal.NextTab = Keybinding{"<tab>"}
	// Legacy single-key comparison fields remain available for explicit config
	// compatibility. Normal defaults are compiled from semantic command paths.
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
