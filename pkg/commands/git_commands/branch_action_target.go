package git_commands

import (
	"errors"
	"fmt"
	"strings"
)

// BranchActionTarget captures the source of a selected-branch action and the
// current HEAD it may affect. It is a read-only guard, not a mutation command.
// Callers still choose the action explicitly and use the captured backend and
// SourceOID when constructing it. Git's normal concurrency model applies after
// the final check; this does not hold a repository lock or authorize a retry.
type BranchActionTarget struct {
	owner                *GitCommon
	sourceRef, sourceOID string
	headRef, headOID     string
}

func (target BranchActionTarget) SourceRef() string { return target.sourceRef }
func (target BranchActionTarget) SourceOID() string { return target.sourceOID }
func (target BranchActionTarget) HeadRef() string   { return target.headRef }
func (target BranchActionTarget) HeadOID() string   { return target.headOID }

var ErrBranchActionChanged = errors.New("the selected source or current HEAD changed; review the action again")

// ResolveBranchActionCommit accepts only a full direct ref or a full commit
// OID, never an ambiguous short name or revision expression. In particular,
// callers must use Git's native upstream ref, not reconstruct its namespace.
// Tag refs are peeled to their commit; raw object IDs must name commits.
func (self *BranchCommands) ResolveBranchActionCommit(ref string) (string, error) {
	if IsFullObjectID(ref) {
		if err := self.validateLiteralCommitOID(ref); err != nil {
			return "", err
		}
		return strings.ToLower(ref), nil
	}
	if !strings.HasPrefix(ref, "refs/") {
		return "", fmt.Errorf("choose a full direct ref or commit object ID")
	}
	if err := self.cmd.New(NewGitCmd("check-ref-format").Arg(ref).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
		return "", err
	}
	state, err := self.readPlacementRef(ref)
	if err != nil {
		return "", err
	}
	if !state.exists {
		return "", fmt.Errorf("source ref %q no longer exists: %w", ref, ErrBranchActionChanged)
	}
	if state.symbolic != "" {
		return "", fmt.Errorf("source ref %q is symbolic; choose its direct target", ref)
	}
	oid := state.oid
	if strings.HasPrefix(ref, "refs/tags/") {
		// Peel the immutable object we just read, not a second lookup of the
		// mutable tag name. A reset source may be an annotated/nested tag.
		output, err := self.cmd.New(NewGitCmd("rev-parse").Arg("--verify", oid+"^{commit}").ToArgv()).
			SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
		if err != nil {
			return "", err
		}
		oid = strings.ToLower(trimTerminalLineEnding(output))
	}
	if err := self.validateLiteralCommitOID(oid); err != nil {
		return "", err
	}
	return oid, nil
}

// PrepareBranchActionTarget retains detached and unborn HEAD identities: reset
// can legitimately operate there. Actions needing a named/non-unborn branch
// must impose that requirement themselves. Do not reject an active merge here:
// an explicit reset can intentionally abort it.
func (self *BranchCommands) PrepareBranchActionTarget(ref, expectedOID string) (BranchActionTarget, error) {
	if !IsFullObjectID(expectedOID) {
		return BranchActionTarget{}, fmt.Errorf("selected source must have a full commit object ID")
	}
	oid, err := self.ResolveBranchActionCommit(ref)
	if err != nil {
		return BranchActionTarget{}, err
	}
	if oid != strings.ToLower(expectedOID) {
		return BranchActionTarget{}, ErrBranchActionChanged
	}
	headRef, headOID, err := self.namedCheckoutHead()
	if err != nil {
		return BranchActionTarget{}, err
	}
	return BranchActionTarget{owner: self.GitCommon, sourceRef: ref, sourceOID: oid, headRef: headRef, headOID: headOID}, nil
}

func (self *BranchCommands) CheckBranchActionTarget(target BranchActionTarget) error {
	if target.owner == nil || target.owner != self.GitCommon {
		return fmt.Errorf("branch action target was not prepared by this repository instance")
	}
	fresh, err := self.PrepareBranchActionTarget(target.sourceRef, target.sourceOID)
	if err != nil {
		return err
	}
	if fresh != target {
		return ErrBranchActionChanged
	}
	return nil
}
