package git_commands

// CheckoutCapturedCommit detaches at a literal commit without forcing files,
// guessing a branch, or retrying a partially completed checkout. Git decides
// which local changes can be carried to the selected tree.
//
// Fork: the active-operation guard runs here, immediately before checkout, as
// it does for named checkout. Without it, a detached checkout during a merge or
// rebase would clear the operation's state (MERGE_HEAD and so on).
//
// envVars are added to the checkout command only; undo and redo pass their
// reflog action here, so the reflog entry is marked.
func (self *BranchCommands) CheckoutCapturedCommit(oid string, envVars ...string) error {
	if err := self.validateLiteralCommitOID(oid); err != nil {
		return err
	}
	if err := self.rejectWorktreeOperation("check out a detached commit"); err != nil {
		return err
	}
	return self.cmd.New(NewGitCmd("checkout").
		Arg("--detach", "--no-overwrite-ignore", "--no-guess", oid+"^{commit}").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DisableLockRetries().
		AddEnvVars("GIT_TERMINAL_PROMPT=0").AddEnvVars(envVars...).Run()
}
