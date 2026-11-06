package git_commands

import "strings"

// Fork: w v shows a commit in Commits, which first needs to know whether the unfiltered
// log can contain it. The unfiltered log walks the ref that BisectCommands.LogRef names,
// which is HEAD outside a bisect, and with git.log.showWholeGraph also every ref that
// git log --all walks (see refreshCommitsWithLimit and getLogCmd). The check ignores the
// log's limit, so a commit beyond the loaded commits is reachable too; the controllers then
// load further. The commands leave no log entry.

// IsCommitInCommitsLog reports whether the unfiltered Commits log can contain the commit
// hash: the log's ref reaches it, or, when wholeGraph is set, so does HEAD or some other
// ref. hash must be a full object ID.
func (self *CommitCommands) IsCommitInCommitsLog(hash string, wholeGraph bool) (bool, error) {
	if err := validateObjectID(hash); err != nil {
		return false, err
	}
	logRef, _ := NewBisectCommands(self.GitCommon).LogRef()
	reached, err := self.reachesCommit(logRef, hash)
	if err != nil || reached {
		return reached, err
	}
	if !wholeGraph {
		return false, nil
	}
	// The whole graph is --all, which walks HEAD as well as the refs. HEAD is not a ref, so
	// it is checked on its own, unless it is the log's ref, which was checked above.
	if logRef != "HEAD" {
		reached, err := self.reachesCommit("HEAD", hash)
		if err != nil || reached {
			return reached, err
		}
	}
	// --contains lists the refs whose tip reaches hash, and --count=1 stops at the first.
	// The value must be attached to the option, since --contains takes it as an optional
	// argument.
	refs, err := self.cmd.New(NewGitCmd("for-each-ref").
		Arg("--contains="+hash, "--count=1", "--format=%(refname)").
		ToArgv()).DontLog().RunWithOutput()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(refs) != "", nil
}

// reachesCommit reports whether ref reaches hash. ref..hash lists what hash reaches and
// ref does not, so the list is empty exactly when ref reaches hash.
func (self *CommitCommands) reachesCommit(ref string, hash string) (bool, error) {
	missing, err := self.cmd.New(NewGitCmd("rev-list").Arg("--count", ref+".."+hash).ToArgv()).DontLog().RunWithOutput()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(missing) == "0", nil
}
