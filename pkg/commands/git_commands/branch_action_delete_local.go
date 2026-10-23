package git_commands

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

var ErrBranchDeleteForceApprovalRequired = errors.New("deleting unmerged branches requires explicit force approval")

// BranchActionLocalDeleteSource is native inspection data for a selected local branch.
type BranchActionLocalDeleteSource struct {
	Target BranchActionTarget
	Info   BranchSyncInfo
	Merged bool
}

type branchDeleteProof struct {
	ref    string
	oid    string
	exists bool
}

// BranchActionLocalDeletePlan is an opaque, read-only review snapshot.
type BranchActionLocalDeletePlan struct {
	owner                       *GitCommon
	targets                     []BranchActionTarget
	mainRefs, excludedProofRefs []string
	sources                     []BranchActionLocalDeleteSource
	proofs                      []branchDeleteProof
	configSignature             string
	forceApproved, prepared     bool
}

func (plan BranchActionLocalDeletePlan) Sources() []BranchActionLocalDeleteSource {
	return slices.Clone(plan.sources)
}

func (plan BranchActionLocalDeletePlan) WithForceApproved() BranchActionLocalDeletePlan {
	plan.forceApproved = true
	return plan
}

func (self *BranchCommands) PrepareBranchActionLocalDelete(targets []BranchActionTarget, mainRefs, excludedProofRefs []string) (BranchActionLocalDeletePlan, error) {
	if len(targets) == 0 {
		return BranchActionLocalDeletePlan{}, fmt.Errorf("select at least one local branch")
	}
	plan := BranchActionLocalDeletePlan{owner: self.GitCommon, targets: slices.Clone(targets), mainRefs: slices.Clone(mainRefs), excludedProofRefs: slices.Clone(excludedProofRefs), prepared: true}
	seen := map[string]bool{}
	for _, target := range targets {
		if target.owner != self.GitCommon || !strings.HasPrefix(target.sourceRef, "refs/heads/") {
			return BranchActionLocalDeletePlan{}, fmt.Errorf("local deletion requires a captured local branch from this repository")
		}
		if seen[target.sourceRef] {
			return BranchActionLocalDeletePlan{}, fmt.Errorf("duplicate local branch target")
		}
		seen[target.sourceRef] = true
		if target.headRef != targets[0].headRef || target.headOID != targets[0].headOID {
			return BranchActionLocalDeletePlan{}, ErrBranchActionChanged
		}
		if err := self.CheckBranchActionTarget(target); err != nil {
			return BranchActionLocalDeletePlan{}, err
		}
		name := strings.TrimPrefix(target.sourceRef, "refs/heads/")
		info, err := (&SyncCommands{GitCommon: self.GitCommon}).InspectBranchSync(name)
		if err != nil {
			return BranchActionLocalDeletePlan{}, err
		}
		if info.FullRef != target.sourceRef || info.OID != target.sourceOID || info.HeadRef != target.headRef || info.HeadOID != target.headOID {
			return BranchActionLocalDeletePlan{}, ErrBranchActionChanged
		}
		occupancy, err := self.branchPlacementOccupancy(target.sourceRef)
		if err != nil {
			return BranchActionLocalDeletePlan{}, err
		}
		info.Reservation = BranchSyncReservation{}
		if occupancy != nil {
			info.Reservation = BranchSyncReservation{GitDir: occupancy.GitDir, Current: occupancy.Current, Kind: occupancy.Kind}
		}
		plan.sources = append(plan.sources, BranchActionLocalDeleteSource{Target: target, Info: info})
	}
	for _, ref := range append(slices.Clone(mainRefs), excludedProofRefs...) {
		if err := self.validateDeleteProofRef(ref); err != nil {
			return BranchActionLocalDeletePlan{}, err
		}
	}
	config, err := self.deleteConfigSignature()
	if err != nil {
		return BranchActionLocalDeletePlan{}, err
	}
	plan.configSignature = config
	excluded := map[string]bool{}
	for _, source := range plan.sources {
		excluded[source.Info.FullRef] = true
	}
	for _, ref := range excludedProofRefs {
		excluded[ref] = true
	}
	proofRefs := []string{}
	add := func(ref string) {
		if ref != "" && !excluded[ref] {
			proofRefs = append(proofRefs, ref)
		}
	}
	for _, source := range plan.sources {
		add(source.Info.UpstreamTrackingRef)
	}
	for _, ref := range mainRefs {
		add(ref)
	}
	if targets[0].headRef != "" {
		add(targets[0].headRef)
	} else if targets[0].headOID != "" {
		plan.proofs = append(plan.proofs, branchDeleteProof{oid: targets[0].headOID, exists: true})
	}
	slices.Sort(proofRefs)
	proofRefs = slices.Compact(proofRefs)
	for _, ref := range proofRefs {
		proof, err := self.readDeleteProof(ref)
		if err != nil {
			return BranchActionLocalDeletePlan{}, err
		}
		plan.proofs = append(plan.proofs, proof)
	}
	proofByRef := make(map[string]branchDeleteProof, len(plan.proofs))
	for _, proof := range plan.proofs {
		if proof.ref != "" {
			proofByRef[proof.ref] = proof
		}
	}
	for i := range plan.sources {
		sourceProofs := []branchDeleteProof{}
		if targets[0].headRef == "" && targets[0].headOID != "" {
			sourceProofs = append(sourceProofs, branchDeleteProof{oid: targets[0].headOID, exists: true})
		}
		for _, ref := range mainRefs {
			if ref != targets[0].headRef && !excluded[ref] {
				sourceProofs = append(sourceProofs, proofByRef[ref])
			}
		}
		if head := targets[0].headRef; head != "" && !excluded[head] {
			sourceProofs = append(sourceProofs, proofByRef[head])
		}
		upstream := plan.sources[i].Info.UpstreamTrackingRef
		if upstream != "" && !excluded[upstream] && upstream != targets[0].headRef {
			sourceProofs = append(sourceProofs, proofByRef[upstream])
		}
		if len(sourceProofs) == 0 {
			plan.sources[i].Merged = false
			continue
		}
		args := []string{"rev-list", "--max-count=1", plan.sources[i].Info.OID + "^{commit}"}
		for _, proof := range sourceProofs {
			if proof.exists {
				args = append(args, "^"+proof.oid+"^{commit}")
			}
		}
		out, _, err := self.cmd.New(NewGitCmd(args[0]).Arg(args[1:]...).ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
		if err != nil {
			return BranchActionLocalDeletePlan{}, err
		}
		plan.sources[i].Merged = trimTerminalLineEnding(out) == ""
	}
	for _, target := range targets {
		if err := self.CheckBranchActionTarget(target); err != nil {
			return BranchActionLocalDeletePlan{}, err
		}
	}
	return plan, nil
}

func (self *BranchCommands) validateDeleteProofRef(ref string) error {
	if !strings.HasPrefix(ref, "refs/") || ref == "refs/" {
		return fmt.Errorf("proof must be a full ref name")
	}
	return self.cmd.New(NewGitCmd("check-ref-format").Arg(ref).ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().Run()
}

func (self *BranchCommands) readDeleteProof(ref string) (branchDeleteProof, error) {
	state, err := self.readPlacementRef(ref)
	if err != nil {
		return branchDeleteProof{}, err
	}
	p := branchDeleteProof{ref: ref, exists: state.exists}
	if !state.exists {
		return p, nil
	}
	output, _, err := self.cmd.New(NewGitCmd("rev-parse").Arg("--verify", ref+"^{commit}").ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return branchDeleteProof{}, err
	}
	p.oid = strings.ToLower(trimTerminalLineEnding(output))
	if err := NewBranchCommands(self.GitCommon).validateLiteralCommitOID(p.oid); err != nil {
		return branchDeleteProof{}, err
	}
	return p, nil
}

func (self *BranchCommands) deleteConfigSignature() (string, error) {
	entries, err := (&SyncCommands{GitCommon: self.GitCommon}).branchSyncConfigEntries()
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, entry := range entries {
		writeBranchSyncConfigEntry(h, entry)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// CheckBranchActionLocalDeleteReview validates inspection data without
// authorizing deletion. Occupied sources remain inspectable so the GUI can
// offer remote-only deletion while disabling local deletion.
func (self *BranchCommands) CheckBranchActionLocalDeleteReview(plan BranchActionLocalDeletePlan) error {
	fresh, err := self.refreshBranchActionLocalDeleteReview(plan)
	if err != nil {
		return err
	}
	if !plan.sameReview(fresh) {
		return ErrBranchActionChanged
	}
	return nil
}

func (plan BranchActionLocalDeletePlan) sameReview(fresh BranchActionLocalDeletePlan) bool {
	return fresh.configSignature == plan.configSignature && slices.Equal(fresh.sources, plan.sources) && slices.Equal(fresh.proofs, plan.proofs)
}

func (self *BranchCommands) refreshBranchActionLocalDeleteReview(plan BranchActionLocalDeletePlan) (BranchActionLocalDeletePlan, error) {
	if plan.owner != self.GitCommon || !plan.prepared {
		return BranchActionLocalDeletePlan{}, fmt.Errorf("local delete plan was not prepared by this repository instance")
	}
	return self.PrepareBranchActionLocalDelete(plan.targets, plan.mainRefs, plan.excludedProofRefs)
}

func (self *BranchCommands) BranchActionLocalDeleteCommand(plan BranchActionLocalDeletePlan) (*oscommands.CmdObj, error) {
	// Reprepare from captured inputs to keep upstream metadata and proof identities fresh.
	fresh, err := self.refreshBranchActionLocalDeleteReview(plan)
	if err != nil {
		return nil, err
	}
	for _, source := range fresh.sources {
		if source.Info.Reservation.Kind != "" {
			return nil, fmt.Errorf("branch %q is reserved by %s", source.Info.Name, source.Info.Reservation.Kind)
		}
	}
	if !plan.sameReview(fresh) {
		return nil, ErrBranchActionChanged
	}
	for _, source := range fresh.sources {
		if !source.Merged && !plan.forceApproved {
			return nil, ErrBranchDeleteForceApprovalRequired
		}
	}
	args := []string{"-D", "--"}
	for _, source := range fresh.sources {
		args = append(args, source.Info.Name)
	}
	return self.cmd.New(NewGitCmd("branch").Arg(args...).ToArgv()).SetWd(self.repoPaths.WorktreePath()).DisableLockRetries(), nil
}
