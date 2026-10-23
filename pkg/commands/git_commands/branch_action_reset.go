package git_commands

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// BranchActionResetCommand resets current HEAD, never the selected source ref.
// Hard reset remains a destructive choice requiring explicit UI approval.
// Validation is read-only; the command runs once using the reviewed commit.
func (self *BranchCommands) BranchActionResetCommand(target BranchActionTarget, strength string) (*oscommands.CmdObj, error) {
	switch strength {
	case "soft", "mixed", "hard":
	default:
		return nil, fmt.Errorf("unknown reset mode %q", strength)
	}
	if err := self.CheckBranchActionTarget(target); err != nil {
		return nil, err
	}
	if target.HeadRef() != "" && target.HeadOID() != target.SourceOID() {
		occupation, err := self.branchOtherWorktreeOccupancy(target.HeadRef())
		if err != nil {
			return nil, err
		}
		if occupation != nil {
			return nil, fmt.Errorf("current branch %q is reserved by another worktree (%s, Git directory %q); it cannot be reset here", target.HeadRef(), occupation.Kind, occupation.GitDir)
		}
	}
	args := NewGitCmd("reset").Arg("--"+strength, target.SourceOID()+"^{commit}", "--")
	return self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()).
		AddEnvVars("GIT_TERMINAL_PROMPT=0").DisableLockRetries(), nil
}
