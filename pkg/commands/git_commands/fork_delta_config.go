package git_commands

// DeltaConfig returns the delta settings in the git config, one "key value"
// per line as `git config --get-regexp` prints them. The fork's default delta
// renderer leaves the styles they set to delta (config/fork_diff_renderers.go).
func (self *ConfigCommands) DeltaConfig() string {
	return self.gitConfig.GetGeneral(`--get-regexp ^delta\.`)
}
