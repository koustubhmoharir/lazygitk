package config

// applyForkDefaults keeps fork policy separate from upstream's default table.
// It must run before loading user or repository config, never after: these
// defaults make mutations explicit, but are not enforced prohibitions.
func applyForkDefaults(config *UserConfig) *UserConfig {
	config.Git.AutoFetch = false
	config.Git.AutoRefresh = false
	config.Git.AutoForwardBranches = "none"
	config.Git.AutoStageResolvedConflicts = false
	return config
}
