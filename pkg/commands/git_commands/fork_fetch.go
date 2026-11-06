package git_commands

import (
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// Fork: the fetch commands behind the global f, the palette's "Fetch all
// remotes", the Remotes fetch and automatic fetch (roadmap U1, 05-F4). Each
// remote's fetch refspecs are checked before a fetch, because a refspec that
// writes into refs/heads/ makes Git move local branches, including ones that
// are not checked out. The review and its flows live in
// controllers/fork_fetch.go and gui/fork_background_fetch.go.

// FetchRefspecRisk is a configured fetch refspec of a remote that writes into
// local branches.
type FetchRefspecRisk struct {
	Remote  string
	Refspec string
}

// RemoteNames returns the names of the repository's remotes, in Git's order.
func (self *SyncCommands) RemoteNames() ([]string, error) {
	cmdArgs := NewGitCmd("remote").ToArgv()
	output, err := self.cmd.New(cmdArgs).DontLog().RunWithOutput()
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(output), nil
}

// PrimaryFetchRemote returns the remote that a plain fetch uses: origin if it
// exists, otherwise the only remote. ok is false when the choice is the user's:
// there are several remotes and none is named origin.
func PrimaryFetchRemote(remotes []string) (remote string, ok bool) {
	if slices.Contains(remotes, "origin") {
		return "origin", true
	}
	if len(remotes) == 1 {
		return remotes[0], true
	}
	return "", false
}

// FetchRefspecRisks returns the fetch refspecs of the given remotes that write
// into local branches. A remote without fetch refspecs has none.
func (self *SyncCommands) FetchRefspecRisks(remotes []string) ([]FetchRefspecRisk, error) {
	var risks []FetchRefspecRisk
	for _, remote := range remotes {
		refspecs, err := self.remoteFetchRefspecs(remote)
		if err != nil {
			return nil, err
		}
		for _, refspec := range refspecs {
			if refspecWritesLocalBranch(refspec) {
				risks = append(risks, FetchRefspecRisk{Remote: remote, Refspec: refspec})
			}
		}
	}
	return risks, nil
}

// FetchBackgroundRefspecRisks returns the risky refspecs of the remotes that
// automatic fetch would fetch: all of them with git.fetchAll, otherwise the
// remote that a plain git fetch picks for the checked-out branch, which is its
// configured remote or origin.
func (self *SyncCommands) FetchBackgroundRefspecRisks() ([]FetchRefspecRisk, error) {
	var remotes []string
	if self.UserConfig().Git.FetchAll {
		all, err := self.RemoteNames()
		if err != nil {
			return nil, err
		}
		remotes = all
	} else {
		remote, err := self.plainFetchRemote()
		if err != nil {
			return nil, err
		}
		remotes = []string{remote}
	}
	return self.FetchRefspecRisks(remotes)
}

// FetchAllRemotes fetches every remote, as "Fetch all remotes" does. Like the
// other foreground fetches it does not write FETCH_HEAD, so a pull can run
// concurrently.
func (self *SyncCommands) FetchAllRemotes(task gocui.Task) error {
	return self.fetchAllRemotesCmdObj(task).Run()
}

func (self *SyncCommands) fetchAllRemotesCmdObj(task gocui.Task) *oscommands.CmdObj {
	cmdArgs := self.fetchCommandBuilder(true).ToArgv()
	return self.cmd.New(cmdArgs).PromptOnCredentialRequest(task)
}

func (self *SyncCommands) remoteFetchRefspecs(remote string) ([]string, error) {
	cmdArgs := NewGitCmd("config").Arg("--get-all", "remote."+remote+".fetch").ToArgv()
	output, err := self.cmd.New(cmdArgs).DontLog().RunWithOutput()
	if err != nil {
		// Exit code 1 means that the setting is absent.
		if commandExitedWithOne(err) {
			return nil, nil
		}
		return nil, err
	}
	return nonEmptyLines(output), nil
}

// plainFetchRemote is the remote that `git fetch` without arguments uses: the
// checked-out branch's configured remote, or origin when there is none.
func (self *SyncCommands) plainFetchRemote() (string, error) {
	// Detached HEAD makes symbolic-ref exit with 1, and it works on an unborn
	// branch too.
	cmdArgs := NewGitCmd("symbolic-ref").Arg("-q", "HEAD").ToArgv()
	ref, err := self.cmd.New(cmdArgs).DontLog().RunWithOutput()
	if err != nil {
		if !commandExitedWithOne(err) {
			return "", err
		}
		return "origin", nil
	}
	branch := strings.TrimPrefix(strings.TrimSpace(ref), "refs/heads/")

	cmdArgs = NewGitCmd("config").Arg("--get", "branch."+branch+".remote").ToArgv()
	remote, err := self.cmd.New(cmdArgs).DontLog().RunWithOutput()
	if err != nil {
		if !commandExitedWithOne(err) {
			return "", err
		}
		return "origin", nil
	}
	return strings.TrimSpace(remote), nil
}

// refspecWritesLocalBranch reports whether a fetch refspec's destination is a
// local branch. A negative refspec (^pattern) fetches nothing and writes
// nothing. A refspec without a destination writes nothing either, and a leading
// + only forces the update.
func refspecWritesLocalBranch(refspec string) bool {
	if strings.HasPrefix(refspec, "^") {
		return false
	}
	_, destination, found := strings.Cut(strings.TrimPrefix(refspec, "+"), ":")
	return found && strings.HasPrefix(destination, "refs/heads/")
}

func nonEmptyLines(output string) []string {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
