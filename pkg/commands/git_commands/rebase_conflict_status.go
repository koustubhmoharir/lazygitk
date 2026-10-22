package git_commands

// HasUnmergedFiles reads the index's unresolved entries independently of
// localized or uncaptured terminal diagnostics. It never refreshes or stages.
func (self *RebaseCommands) HasUnmergedFiles() (bool, error) {
	output, _, err := self.cmd.New(NewGitCmd("diff").
		Arg("--name-only", "--diff-filter=U", "-z", "--").ToArgv()).DontLog().RunWithOutputs()
	return output != "", err
}
