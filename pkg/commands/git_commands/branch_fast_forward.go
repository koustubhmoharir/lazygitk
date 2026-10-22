package git_commands

import (
	"errors"
	"fmt"
	"strings"
)

// ErrBranchSyncChanged asks callers to review the source branch again.
var ErrBranchSyncChanged = errors.New("the branch or sync target changed; review the current state before proceeding")

// BranchFastForwardPlan captures a validated, forward-only ref update.
type BranchFastForwardPlan struct {
	owner                   *GitCommon
	name, oldOID, targetOID string
	placement               BranchPlacementPlan
	needsUpdate, prepared   bool
}

func (plan BranchFastForwardPlan) Name() string      { return plan.name }
func (plan BranchFastForwardPlan) OldOID() string    { return plan.oldOID }
func (plan BranchFastForwardPlan) TargetOID() string { return plan.targetOID }
func (plan BranchFastForwardPlan) NeedsUpdate() bool { return plan.needsUpdate }

// PrepareBranchFastForward permits equality, an already-ahead no-op, or a strict fast-forward.
func (self *SyncCommands) PrepareBranchFastForward(name, expectedOID, targetOID string) (BranchFastForwardPlan, error) {
	if !IsFullObjectID(expectedOID) {
		return BranchFastForwardPlan{}, ErrBranchSyncChanged
	}
	if err := validateObjectID(targetOID); err != nil {
		return BranchFastForwardPlan{}, fmt.Errorf("target must be a full object ID: %w", err)
	}
	if err := NewBranchCommands(self.GitCommon).validateLiteralCommitOID(targetOID); err != nil {
		return BranchFastForwardPlan{}, fmt.Errorf("invalid fast-forward target: %w", err)
	}
	info, err := self.InspectBranchSync(name)
	if err != nil {
		return BranchFastForwardPlan{}, err
	}
	expectedOID = strings.ToLower(expectedOID)
	targetOID = strings.ToLower(targetOID)
	if info.OID != expectedOID {
		return BranchFastForwardPlan{}, ErrBranchSyncChanged
	}
	plan := BranchFastForwardPlan{owner: self.GitCommon, name: name, oldOID: expectedOID, targetOID: targetOID}
	if expectedOID == targetOID {
		plan.prepared = true
		return plan, nil
	}
	targetIsAncestor, err := self.branchSyncIsCommitAncestor(targetOID, expectedOID)
	if err != nil {
		return BranchFastForwardPlan{}, err
	}
	if targetIsAncestor {
		plan.prepared = true // Already ahead: never rewind.
		return plan, nil
	}
	oldIsAncestor, err := self.branchSyncIsCommitAncestor(expectedOID, targetOID)
	if err != nil {
		return BranchFastForwardPlan{}, err
	}
	if !oldIsAncestor {
		return BranchFastForwardPlan{}, fmt.Errorf("%w: branch %q and target have diverged; off-branch integration requires checkout", ErrBranchSyncDiverged, name)
	}
	placement, err := NewBranchCommands(self.GitCommon).PrepareBranchPlacement(name, targetOID)
	if err != nil {
		return BranchFastForwardPlan{}, err
	}
	if placement.OldOID() != expectedOID {
		return BranchFastForwardPlan{}, ErrBranchSyncChanged
	}
	plan.placement = placement
	plan.needsUpdate, plan.prepared = true, true
	return plan, nil
}

// CheckBranchFastForward revalidates a prepared plan without applying its ref
// update. It is a read-only preflight; callers must still use Apply for the
// expected-old guarded placement transaction.
func (self *SyncCommands) CheckBranchFastForward(plan BranchFastForwardPlan) error {
	if !plan.prepared {
		return fmt.Errorf("branch fast-forward plan was not prepared")
	}
	if plan.owner != self.GitCommon {
		return fmt.Errorf("branch fast-forward plan belongs to a different repository instance")
	}
	fresh, err := self.PrepareBranchFastForward(plan.name, plan.oldOID, plan.targetOID)
	if err != nil {
		return mapBranchFastForwardPlacementError(err)
	}
	if fresh != plan {
		return ErrBranchSyncChanged
	}
	return nil
}

// ApplyBranchFastForward revalidates the captured source and delegates a real
// update once to the expected-old guarded branch placement transaction.
func (self *SyncCommands) ApplyBranchFastForward(plan BranchFastForwardPlan) error {
	if err := self.CheckBranchFastForward(plan); err != nil {
		return err
	}
	if !plan.needsUpdate {
		return nil
	}
	return mapBranchFastForwardPlacementError(NewBranchCommands(self.GitCommon).ApplyBranchPlacement(plan.placement))
}

var ErrBranchSyncDiverged = errors.New("branch and target are not in a fast-forward relationship")

func mapBranchFastForwardPlacementError(err error) error {
	var uncertain *branchRefCommitError
	if errors.As(err, &uncertain) {
		return err
	}
	if errors.Is(err, ErrBranchPlacementChanged) {
		return errors.Join(ErrBranchSyncChanged, err)
	}
	return err
}

func (self *SyncCommands) branchSyncIsCommitAncestor(ancestor, descendant string) (bool, error) {
	// Exact OIDs are authoritative here. Git warns about 40/64-hex ref names
	// during revision parsing; disable that ambiguity warning for this query so
	// a legitimate exit status of 1 remains distinguishable from query failure.
	ancestorCommit, descendantCommit := ancestor+"^{commit}", descendant+"^{commit}"
	cmd := self.cmd.New(NewGitCmd("merge-base").Config("core.warnAmbiguousRefs=false").Arg("--is-ancestor", ancestorCommit, descendantCommit).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog()
	stdout, _, err := cmd.RunWithOutputs()
	if stdout != "" {
		return false, fmt.Errorf("merge-base --is-ancestor returned unexpected output %q", stdout)
	}
	if err == nil {
		return true, nil
	}
	if commandExitedWithOne(err) {
		return false, nil
	}
	return false, fmt.Errorf("query commit ancestry: %w", err)
}
