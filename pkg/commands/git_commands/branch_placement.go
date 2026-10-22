package git_commands

import (
	"errors"
	"fmt"
	"strings"
)

type BranchPlacementAction uint8

const (
	BranchPlacementInvalid BranchPlacementAction = iota
	BranchPlacementCreate
	BranchPlacementMove
	BranchPlacementNoop
)

// BranchPlacementPlan is an immutable snapshot of the literal destination and
// the selected commit. Its old tip is the state that a move warning describes.
// ApplyBranchPlacement revalidates it; preparation never authorizes a retry.
type BranchPlacementPlan struct {
	name, ref, targetOID, oldOID string
	action                       BranchPlacementAction
}

func (plan BranchPlacementPlan) Name() string                  { return plan.name }
func (plan BranchPlacementPlan) TargetOID() string             { return plan.targetOID }
func (plan BranchPlacementPlan) OldOID() string                { return plan.oldOID }
func (plan BranchPlacementPlan) Action() BranchPlacementAction { return plan.action }

var ErrBranchPlacementChanged = errors.New("the branch changed; review its current state before proceeding")

type branchPlacementRef struct {
	exists   bool
	oid      string
	symbolic string
}

// PrepareBranchPlacement validates a literal local name, a full commit OID and
// fresh reservations. No checkout, sanitization, revision-name expansion or
// ref write occurs. A same-tip destination is a no-op, even if checked out.
func (self *BranchCommands) PrepareBranchPlacement(name, targetOID string) (BranchPlacementPlan, error) {
	if err := self.validatePlacementBranchName(name); err != nil {
		return BranchPlacementPlan{}, err
	}
	if err := self.validateLiteralCommitOID(targetOID); err != nil {
		return BranchPlacementPlan{}, err
	}
	plan := BranchPlacementPlan{name: name, ref: "refs/heads/" + name, targetOID: strings.ToLower(targetOID)}
	state, err := self.readPlacementRef(plan.ref)
	if err != nil {
		return BranchPlacementPlan{}, err
	}
	if state.symbolic != "" {
		return BranchPlacementPlan{}, fmt.Errorf("branch %q is a symbolic alias; choose a direct local branch", name)
	}
	plan.oldOID = state.oid
	switch {
	case !state.exists:
		plan.action = BranchPlacementCreate
	case state.oid == plan.targetOID:
		plan.action = BranchPlacementNoop
	default:
		plan.action = BranchPlacementMove
	}
	if plan.action != BranchPlacementNoop {
		if err := self.checkPlacementOccupation(plan); err != nil {
			return BranchPlacementPlan{}, err
		}
	}
	return plan, nil
}

// ApplyBranchPlacement applies exactly the prepared snapshot. Callers must ask
// again after ErrBranchPlacementChanged; a move must never be silently retried.
// Ref type and occupancy are checked again while Git holds the target-ref lock.
func (self *BranchCommands) ApplyBranchPlacement(plan BranchPlacementPlan) error {
	if plan.action == BranchPlacementInvalid {
		return fmt.Errorf("branch placement was not prepared")
	}
	fresh, err := self.PrepareBranchPlacement(plan.name, plan.targetOID)
	if err != nil {
		return err
	}
	if fresh != plan {
		return ErrBranchPlacementChanged
	}
	if plan.action == BranchPlacementNoop {
		return nil
	}
	expected := plan.oldOID
	if expected == "" {
		expected = strings.Repeat("0", len(plan.targetOID))
	}
	reason := "lazygitk: move branch to selected commit"
	if plan.action == BranchPlacementCreate {
		reason = "lazygitk: create branch at selected commit"
	}
	err = self.guardedBranchRefUpdate(plan.ref, plan.targetOID, expected, reason, func() error {
		state, err := self.readPlacementRef(plan.ref)
		if err != nil {
			return err
		}
		if !placementRefMatches(plan, state) {
			return ErrBranchPlacementChanged
		}
		return self.checkPlacementOccupation(plan)
	})
	if err == nil || errors.Is(err, ErrBranchPlacementChanged) {
		return err
	}
	var uncertain *branchRefCommitError
	if errors.As(err, &uncertain) {
		// Commit may already have happened. Never turn this into an automatic
		// changed-state loop that might misleadingly report "nothing changed".
		return err
	}
	// A concurrent change can reject prepare before the final guard runs.
	// Classify that as stale only after re-reading, not by matching Git text.
	if state, readErr := self.readPlacementRef(plan.ref); readErr == nil && !placementRefMatches(plan, state) {
		return errors.Join(ErrBranchPlacementChanged, err)
	}
	return err
}

func placementRefMatches(plan BranchPlacementPlan, state branchPlacementRef) bool {
	return state.symbolic == "" && state.exists == (plan.oldOID != "") && state.oid == plan.oldOID
}

func (self *BranchCommands) checkPlacementOccupation(plan BranchPlacementPlan) error {
	occupation, err := self.branchPlacementOccupancy(plan.ref)
	if err != nil || occupation == nil {
		return err
	}
	if occupation.Current {
		return fmt.Errorf("branch %q is in use by the current worktree (%s). Use explicit Reset actions for current-branch movement, after finishing any in-progress operation", plan.name, occupation.Kind)
	}
	return fmt.Errorf("branch %q is reserved by another worktree (%s, Git directory %q); it cannot be moved here", plan.name, occupation.Kind, occupation.GitDir)
}

func (self *BranchCommands) validatePlacementBranchName(name string) error {
	if name == "" || strings.HasPrefix(name, "-") || name == "HEAD" || strings.ContainsAny(name, "\x00\r\n") {
		return fmt.Errorf("invalid literal local branch name %q", name)
	}
	// The full-ref check rejects @{...} before --branch could expand @{-N}.
	if err := self.cmd.New(NewGitCmd("check-ref-format").Arg("refs/heads/" + name).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
		return fmt.Errorf("invalid local branch name %q: %w", name, err)
	}
	output, err := self.cmd.New(NewGitCmd("check-ref-format").Arg("--branch", name).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return fmt.Errorf("invalid local branch name %q: %w", name, err)
	}
	if trimTerminalLineEnding(output) != name {
		return fmt.Errorf("branch names must be literal, not expanded revision expressions")
	}
	return nil
}

func (self *BranchCommands) readPlacementRef(ref string) (branchPlacementRef, error) {
	// for-each-ref omits broken symbolic aliases, so query the exact symbolic
	// name separately too. This also detects aliases while the ref lock is held.
	symbolic, err := self.cmd.New(NewGitCmd("symbolic-ref").Arg("--quiet", ref).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err == nil {
		target := trimTerminalLineEnding(symbolic)
		if target == "" {
			return branchPlacementRef{}, fmt.Errorf("symbolic-ref returned an empty target")
		}
		return branchPlacementRef{exists: true, symbolic: target}, nil
	}
	if !commandExitedWithOne(err) {
		return branchPlacementRef{}, err
	}
	output, err := self.cmd.New(NewGitCmd("for-each-ref").
		Arg("--format=%(refname)%00%(objectname)%00%(symref)", "--", ref).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return branchPlacementRef{}, err
	}
	var state branchPlacementRef
	for _, record := range strings.Split(trimTerminalLineEnding(output), "\n") {
		if record == "" {
			continue
		}
		fields := strings.Split(record, "\x00")
		if len(fields) != 3 {
			return branchPlacementRef{}, fmt.Errorf("malformed local-ref query output")
		}
		if fields[0] != ref {
			continue // Git's pattern also matches descendants of a missing ref.
		}
		if state.exists {
			return branchPlacementRef{}, fmt.Errorf("local-ref query returned duplicate exact matches")
		}
		if err := validateObjectID(fields[1]); err != nil {
			return branchPlacementRef{}, err
		}
		state = branchPlacementRef{exists: true, oid: strings.ToLower(fields[1]), symbolic: fields[2]}
	}
	return state, nil
}
