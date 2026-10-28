package git_commands

import (
	"fmt"
	"strings"
)

// PreviousCheckoutCommit resolves Git's @{-1}, the checkout before the current
// one, to the commit it names. It is used for a previous checkout that is not a
// local branch, which is then checked out detached.
func (self *BranchCommands) PreviousCheckoutCommit() (string, error) {
	output, err := self.cmd.New(NewGitCmd("rev-parse").
		Arg("--verify", "--end-of-options", "@{-1}^{commit}").ToArgv()).DontLog().RunWithOutput()
	if err != nil {
		return "", err
	}
	oid := strings.TrimSpace(output)
	if err := validateObjectID(oid); err != nil {
		return "", fmt.Errorf("git returned invalid commit object ID: %w", err)
	}
	return oid, nil
}
