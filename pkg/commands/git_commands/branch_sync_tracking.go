package git_commands

import (
	"fmt"
	"strings"
)

// checkBranchSyncTrackingWrite checks local storage before a scoped network
// operation. Push can update tracking refs too, through configured fetch
// mappings. Never allow these writes to move a branch, tag, special ref or
// symbolic alias. This is validation, not a reservation against other writers.
func (self *SyncCommands) checkBranchSyncTrackingWrite(ref string) error {
	if !strings.HasPrefix(ref, "refs/") {
		return fmt.Errorf("unsafe local tracking destination %q", ref)
	}
	if err := self.cmd.New(NewGitCmd("check-ref-format").Arg(ref).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
		return fmt.Errorf("invalid local tracking destination: %w", err)
	}
	for _, namespace := range []string{"refs/heads", "refs/tags", "refs/notes", "refs/replace", "refs/bisect", "refs/rewritten", "refs/worktree", "refs/namespaces", "refs/stash"} {
		if ref == namespace || strings.HasPrefix(ref, namespace+"/") {
			return fmt.Errorf("remote sync could modify %q; choose a tracking-only destination", ref)
		}
	}
	state, err := NewBranchCommands(self.GitCommon).readPlacementRef(ref)
	if err != nil {
		return err
	}
	if state.symbolic != "" {
		return fmt.Errorf("tracking destination %q is a symbolic alias; branch sync cannot update it", ref)
	}
	return nil
}
