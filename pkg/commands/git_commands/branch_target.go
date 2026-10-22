package git_commands

import (
	"fmt"
	"strings"
)

// validateLiteralCommitOID verifies identity as well as type. A ref whose name
// merely looks like an OID must not substitute another commit, including when
// its length differs from this repository's native object format. Keep stderr
// separate: Git can warn about a shadowing ref while still resolving the OID.
func (self *BranchCommands) validateLiteralCommitOID(oid string) error {
	if err := validateObjectID(oid); err != nil {
		return err
	}
	output, _, err := self.cmd.New(NewGitCmd("cat-file").
		Arg("--batch-check=%(objectname) %(objecttype)").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).SetStdin(oid + "\n").DontLog().RunWithOutputs()
	if err != nil {
		return err
	}
	fields := strings.Split(trimTerminalLineEnding(output), " ")
	if len(fields) != 2 || !strings.EqualFold(fields[0], oid) || fields[1] != "commit" {
		return fmt.Errorf("the selected object must be an existing commit with that exact object ID")
	}
	return nil
}
