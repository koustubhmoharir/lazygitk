package git_commands

import "strings"

// Fork: w v shows a commit in Commits, which first needs to know whether the unfiltered
// log can contain it. The unfiltered log walks HEAD, and with git.log.showWholeGraph
// also every ref that git log --all walks (see refreshCommitsWithLimit and getLogCmd).
// The check ignores the log's limit, so a commit beyond the loaded commits is reachable
// too; the controllers then load further. The commands leave no log entry.

// IsCommitInCommitsLog reports whether the unfiltered Commits log can contain the commit
// hash: HEAD reaches it, or, when wholeGraph is set, so does some other ref. hash must
// be a full object ID.
func (self *CommitCommands) IsCommitInCommitsLog(hash string, wholeGraph bool) (bool, error) {
	if err := validateObjectID(hash); err != nil {
		return false, err
	}
	// HEAD..hash lists what hash reaches and HEAD does not. hash is in that list unless
	// HEAD reaches it, so the list is empty exactly when HEAD reaches hash.
	missing, err := self.cmd.New(NewGitCmd("rev-list").Arg("--count", "HEAD.."+hash).ToArgv()).DontLog().RunWithOutput()
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(missing) == "0" {
		return true, nil
	}
	if !wholeGraph {
		return false, nil
	}
	// --contains lists the refs whose tip reaches hash, and --count=1 stops at the first.
	// HEAD is not a ref, so it was checked above. The value must be attached to the
	// option, since --contains takes it as an optional argument.
	refs, err := self.cmd.New(NewGitCmd("for-each-ref").
		Arg("--contains="+hash, "--count=1", "--format=%(refname)").
		ToArgv()).DontLog().RunWithOutput()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(refs) != "", nil
}
