package git_commands

import (
	"fmt"
	"strings"
)

// PrepareLocalBranchUpdate captures the selected branch and the exact commit
// held by its chosen local upstream. It creates no fetch or checkout plan.
func (self *SyncCommands) PrepareLocalBranchUpdate(name, expectedOID, upstreamRef string) (BranchFastForwardPlan, error) {
	if err := validateObjectID(expectedOID); err != nil {
		return BranchFastForwardPlan{}, err
	}
	info, err := self.InspectBranchSync(name)
	if err != nil {
		return BranchFastForwardPlan{}, err
	}
	if info.OID != strings.ToLower(expectedOID) {
		return BranchFastForwardPlan{}, ErrBranchSyncChanged
	}
	if info.HeadRef == info.FullRef {
		return BranchFastForwardPlan{}, fmt.Errorf("branch %q is checked out in this worktree; local branch update requires an off-branch target", name)
	}
	upstreamOID, err := self.inspectLocalBranchSyncRef(upstreamRef)
	if err != nil {
		return BranchFastForwardPlan{}, err
	}
	return self.PrepareBranchFastForward(name, info.OID, upstreamOID)
}

// inspectLocalBranchSyncRef resolves only an exact direct local branch ref.
// It deliberately avoids revision parsing, DWIM lookup, and symbolic aliases.
func (self *SyncCommands) inspectLocalBranchSyncRef(fullRef string) (string, error) {
	if !strings.HasPrefix(fullRef, "refs/heads/") || fullRef == "refs/heads/" {
		return "", fmt.Errorf("choose a full local branch ref under refs/heads/")
	}
	if err := self.cmd.New(NewGitCmd("check-ref-format").Arg(fullRef).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
		return "", fmt.Errorf("invalid local upstream ref: %w", err)
	}
	state, err := NewBranchCommands(self.GitCommon).readPlacementRef(fullRef)
	if err != nil {
		return "", err
	}
	if state.symbolic != "" {
		return "", fmt.Errorf("local upstream ref %q is a symbolic alias", fullRef)
	}
	if !state.exists {
		return "", fmt.Errorf("local upstream ref %q does not exist", fullRef)
	}
	if err := NewBranchCommands(self.GitCommon).validateLiteralCommitOID(state.oid); err != nil {
		return "", fmt.Errorf("local upstream ref %q does not point directly to a commit: %w", fullRef, err)
	}
	return strings.ToLower(state.oid), nil
}
