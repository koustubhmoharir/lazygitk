package git_commands

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: w b in Commits offers the local branches, remote branches and tags that
// point at the selected commit. The list is read when the command runs, so it is
// never stale.

// RefsAtCommit returns the full names of the local branches, remote branches and
// tags that point at the commit with the given full hash. The order is that of
// for-each-ref, which lists refs/heads first, then refs/remotes, then refs/tags.
// Symbolic refs such as refs/remotes/origin/HEAD are skipped, since they name
// another ref, which is listed itself. Annotated tags count when they peel to
// that commit.
func (self *CommitCommands) RefsAtCommit(hash string) ([]string, error) {
	if err := validateObjectID(hash); err != nil {
		return nil, err
	}
	output, err := self.cmd.New(NewGitCmd("for-each-ref").
		Arg("--points-at="+hash).
		Arg("--format=%(symref)%00%(refname)").
		Arg("refs/heads", "refs/remotes", "refs/tags").
		ToArgv()).
		DontLog().
		RunWithOutput()
	if err != nil {
		return nil, err
	}

	refs := []string{}
	for _, line := range utils.SplitLines(output) {
		fields := strings.Split(line, "\x00")
		if len(fields) != 2 {
			return nil, fmt.Errorf("unexpected for-each-ref line: %q", line)
		}
		symref, refName := fields[0], fields[1]
		if symref != "" {
			continue
		}
		refs = append(refs, refName)
	}
	return refs, nil
}
