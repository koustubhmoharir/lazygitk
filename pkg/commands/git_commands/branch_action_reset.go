package git_commands

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// BranchActionResetCommand resets current HEAD, never the selected source ref.
// Hard reset remains a destructive choice requiring explicit UI approval.
// A hard reset also needs approvedFingerprint, the fingerprint of the untracked and
// ignored risks the user reviewed (HardResetIgnoredRisks). The risks are recomputed
// here, and a different list is refused with ResetIgnoredRiskError. envVars are set
// on the reset command; undo and redo use them to mark its reflog entry.
// Validation is read-only; the command runs once using the reviewed commit.
func (self *BranchCommands) BranchActionResetCommand(target BranchActionTarget, strength, approvedFingerprint string, envVars ...string) (*oscommands.CmdObj, error) {
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
	if strength == "hard" {
		paths, fingerprint, err := self.hardResetRisks(target.SourceOID())
		if err != nil {
			return nil, err
		}
		if fingerprint != approvedFingerprint {
			return nil, &ResetIgnoredRiskError{Paths: paths, Fingerprint: fingerprint}
		}
	}
	args := NewGitCmd("reset").Arg("--"+strength, target.SourceOID()+"^{commit}", "--")
	return self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()).
		AddEnvVars("GIT_TERMINAL_PROMPT=0").AddEnvVars(envVars...).DisableLockRetries(), nil
}
