package git_commands

// Fork: LogRef is the ref that the unfiltered Commits log walks, with the bisect info it
// read. The Commits refresh (helpers/refresh_helper.go refForLog) and the w v reachability
// check (IsCommitInCommitsLog) both use it, so that they cannot disagree about the log.
func (self *BisectCommands) LogRef() (string, *BisectInfo) {
	bisectInfo := self.GetInfo()

	if !bisectInfo.Started() {
		return "HEAD", bisectInfo
	}

	// need to see if our bisect's current commit is reachable from our 'new' ref.
	if bisectInfo.Bisecting() && !self.ReachableFromStart(bisectInfo) {
		return bisectInfo.GetNewHash(), bisectInfo
	}

	return bisectInfo.GetStartHash(), bisectInfo
}
