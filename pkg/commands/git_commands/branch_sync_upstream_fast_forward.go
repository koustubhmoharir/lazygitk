package git_commands

import (
	"fmt"
	"strings"
)

// UpstreamFastForwardState classifies a branch against its local upstream ref.
type UpstreamFastForwardState uint8

const (
	UpstreamFastForwardUpToDate           UpstreamFastForwardState = iota // equal or ahead
	UpstreamFastForwardBehind                                             // strict ancestor of its upstream
	UpstreamFastForwardDiverged                                           // neither contains the other
	UpstreamFastForwardNoUpstream                                         // no upstream configured
	UpstreamFastForwardMissingTrackingRef                                 // upstream configured, its local ref absent
)

// UpstreamFastForward is a read-only snapshot of a branch relative to the
// upstream ref exactly as it exists locally. It never fetches or mutates refs.
type UpstreamFastForward struct {
	Info        BranchSyncInfo
	UpstreamOID string // empty unless the upstream ref exists
	State       UpstreamFastForwardState
}

// Current reports whether the branch is checked out in this worktree.
func (u UpstreamFastForward) Current() bool { return u.Info.HeadRef == u.Info.FullRef }

// InspectUpstreamFastForward classifies a literal local branch against its
// configured upstream ref without fetching. Callers must revalidate before
// any mutation; this snapshot is not authorization to move the branch.
func (self *SyncCommands) InspectUpstreamFastForward(name string) (UpstreamFastForward, error) {
	info, err := self.InspectBranchSync(name)
	if err != nil {
		return UpstreamFastForward{}, err
	}
	result := UpstreamFastForward{Info: info}
	if info.UpstreamTrackingRef == "" {
		result.State = UpstreamFastForwardNoUpstream
		return result, nil
	}
	branches := NewBranchCommands(self.GitCommon)
	upstream, err := branches.readPlacementRef(info.UpstreamTrackingRef)
	if err != nil {
		return UpstreamFastForward{}, err
	}
	if upstream.symbolic != "" {
		return UpstreamFastForward{}, fmt.Errorf("upstream ref %q is a symbolic alias; configure a direct tracking ref", info.UpstreamTrackingRef)
	}
	if !upstream.exists {
		result.State = UpstreamFastForwardMissingTrackingRef
		return result, nil
	}
	if err := branches.validateLiteralCommitOID(upstream.oid); err != nil {
		return UpstreamFastForward{}, fmt.Errorf("upstream ref %q does not point to a commit: %w", info.UpstreamTrackingRef, err)
	}
	result.UpstreamOID = strings.ToLower(upstream.oid)
	if result.UpstreamOID == info.OID {
		result.State = UpstreamFastForwardUpToDate
		return result, nil
	}
	upstreamIsAncestor, err := self.branchSyncIsCommitAncestor(result.UpstreamOID, info.OID)
	if err != nil {
		return UpstreamFastForward{}, err
	}
	if upstreamIsAncestor {
		result.State = UpstreamFastForwardUpToDate
		return result, nil
	}
	branchIsAncestor, err := self.branchSyncIsCommitAncestor(info.OID, result.UpstreamOID)
	if err != nil {
		return UpstreamFastForward{}, err
	}
	if branchIsAncestor {
		result.State = UpstreamFastForwardBehind
	} else {
		result.State = UpstreamFastForwardDiverged
	}
	return result, nil
}
