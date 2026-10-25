package config

// CommandUsageFilePath returns the path of the per-user command usage counts
// file, kept beside state.yml (fork: per-user usage counts; see pkg/commandusage).
func CommandUsageFilePath() (string, error) {
	return stateFilePath("command_usage.json")
}
