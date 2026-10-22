package git_commands

import (
	"errors"
	"fmt"
	"strings"
)

var errBranchRenameDestinationOccupied = errors.New("branch rename destination is occupied")

// BranchRenamePlan is an immutable approval to rename one literal local branch.
type BranchRenamePlan struct {
	owner                 *GitCommon
	oldName, newName, oid string
	needsUpdate, prepared bool
}

func (plan BranchRenamePlan) OldName() string   { return plan.oldName }
func (plan BranchRenamePlan) NewName() string   { return plan.newName }
func (plan BranchRenamePlan) SourceOID() string { return plan.oid }
func (plan BranchRenamePlan) NeedsUpdate() bool { return plan.needsUpdate }

// PrepareBranchRename captures a literal source and a currently unused target.
func (self *SyncCommands) PrepareBranchRename(oldName, expectedOID, newName string) (BranchRenamePlan, error) {
	if !IsFullObjectID(expectedOID) {
		return BranchRenamePlan{}, fmt.Errorf("source must be a full object ID")
	}
	branches := NewBranchCommands(self.GitCommon)
	if err := branches.validatePlacementBranchName(newName); err != nil {
		return BranchRenamePlan{}, err
	}
	info, err := self.InspectBranchSync(oldName)
	if err != nil {
		return BranchRenamePlan{}, err
	}
	expectedOID = strings.ToLower(expectedOID)
	if info.OID != expectedOID {
		return BranchRenamePlan{}, ErrBranchSyncChanged
	}
	plan := BranchRenamePlan{owner: self.GitCommon, oldName: oldName, newName: newName, oid: expectedOID, needsUpdate: oldName != newName, prepared: true}
	if !plan.needsUpdate {
		return plan, nil
	}
	if info.Reservation.Kind != "" && info.Reservation.Kind != "checkout" {
		return BranchRenamePlan{}, fmt.Errorf("branch %q is reserved by an in-progress operation (%s)", oldName, info.Reservation.Kind)
	}
	target, err := branches.readPlacementRef("refs/heads/" + newName)
	if err != nil {
		return BranchRenamePlan{}, err
	}
	if target.exists || target.symbolic != "" {
		return BranchRenamePlan{}, fmt.Errorf("destination branch %q already exists: %w", newName, errBranchRenameDestinationOccupied)
	}
	entries, err := self.branchSyncConfigEntries()
	if err != nil {
		return BranchRenamePlan{}, err
	}
	if branchRenameHasDestinationConfig(entries, newName) {
		return BranchRenamePlan{}, fmt.Errorf("destination branch %q already has branch configuration: %w", newName, errBranchRenameDestinationOccupied)
	}
	return plan, nil
}

func branchRenameHasDestinationConfig(entries []branchSyncConfigEntry, name string) bool {
	prefix := "branch." + name + "."
	for _, entry := range entries {
		remainder, ok := strings.CutPrefix(entry.key, prefix)
		if ok && remainder != "" && !strings.ContainsRune(remainder, '.') {
			return true
		}
	}
	return false
}

// ApplyBranchRename rechecks approval, then invokes native Git exactly once.
func (self *SyncCommands) ApplyBranchRename(plan BranchRenamePlan) error {
	if !plan.prepared {
		return fmt.Errorf("branch rename plan was not prepared")
	}
	if plan.owner != self.GitCommon {
		return fmt.Errorf("branch rename plan belongs to a different repository instance")
	}
	fresh, err := self.PrepareBranchRename(plan.oldName, plan.oid, plan.newName)
	if err != nil {
		if errors.Is(err, errBranchRenameDestinationOccupied) {
			return ErrBranchSyncChanged
		}
		return err
	}
	if fresh != plan {
		return ErrBranchSyncChanged
	}
	if !plan.needsUpdate {
		return nil
	}
	cmd := self.cmd.New(NewGitCmd("branch").Arg("--move", "--", plan.oldName, plan.newName).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DisableLockRetries()
	return cmd.Run()
}
