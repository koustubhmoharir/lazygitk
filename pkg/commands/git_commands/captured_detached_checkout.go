package git_commands

// CheckoutCapturedCommit detaches at a literal commit without forcing files,
// guessing a branch, or retrying a partially completed checkout. Git decides
// which local changes can be carried to the selected tree.
func (self *BranchCommands) CheckoutCapturedCommit(oid string) error {
	if err := self.validateLiteralCommitOID(oid); err != nil {
		return err
	}
	return self.cmd.New(NewGitCmd("checkout").
		Arg("--detach", "--no-overwrite-ignore", "--no-guess", oid+"^{commit}").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DisableLockRetries().
		AddEnvVars("GIT_TERMINAL_PROMPT=0").Run()
}
