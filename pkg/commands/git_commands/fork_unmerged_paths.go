package git_commands

// Fork: `c c` refuses to continue while any path is unmerged. The Files model
// can be stale because automatic refresh is off, so the check asks Git directly.

// UnmergedPaths returns the paths Git reports as unmerged, including those whose
// conflict markers are gone but which are not yet staged. It never changes the index.
func (self *StatusCommands) UnmergedPaths() ([]string, error) {
	args := NewGitCmd("diff").Arg("--name-only", "--diff-filter=U", "-z").ToArgv()
	output, err := self.cmd.New(args).DontLog().RunWithOutput()
	if err != nil {
		return nil, err
	}
	return nulPaths([]byte(output)), nil
}
