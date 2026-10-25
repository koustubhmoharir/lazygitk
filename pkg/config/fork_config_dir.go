package config

// AppDirName names the fork's config and state directories, so lazygitk never
// reads or writes upstream lazygit's files (fork: roadmap item 10).
const AppDirName = "lazygitk"

// RepoConfigFileName is the per-repo config file in the git dir; parent
// directories use the dot-prefixed form.
const RepoConfigFileName = AppDirName + ".yml"
