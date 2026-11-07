package git_commands

import "fmt"

// Fork: `r s` in the Branches rebase menu rebases the checked-out branch onto
// its remote tracking branch. The candidates are read when the menu is built;
// the caller captures and revalidates each one before it is used.

// TrackingRebaseCandidate is a remote-tracking ref that the checked-out branch
// may be rebased onto. Upstream is true for the configured upstream.
type TrackingRebaseCandidate struct {
	Ref      string
	Upstream bool
}

// TrackingRebaseCandidates lists the existing remote-tracking refs for the
// local branch name: its configured upstream first, then
// refs/remotes/<remote>/<name> for each remote in git's order. Refs that do not
// exist are omitted, and each ref appears once.
func (self *SyncCommands) TrackingRebaseCandidates(name string) ([]TrackingRebaseCandidate, error) {
	info, err := self.InspectBranchSync(name)
	if err != nil {
		return nil, err
	}
	remotes, err := self.remoteNames()
	if err != nil {
		return nil, err
	}
	branches := NewBranchCommands(self.GitCommon)
	candidates := []TrackingRebaseCandidate{}
	seen := map[string]bool{}
	add := func(ref string, upstream bool) error {
		if ref == "" || seen[ref] {
			return nil
		}
		seen[ref] = true
		state, err := branches.readPlacementRef(ref)
		if err != nil {
			return err
		}
		// A symbolic remote ref is not a direct commit, and a rebase cannot use it.
		if !state.exists || state.symbolic != "" {
			return nil
		}
		candidates = append(candidates, TrackingRebaseCandidate{Ref: ref, Upstream: upstream})
		return nil
	}
	if err := add(info.UpstreamTrackingRef, true); err != nil {
		return nil, err
	}
	for _, remote := range remotes {
		if err := add(fmt.Sprintf("refs/remotes/%s/%s", remote, name), false); err != nil {
			return nil, err
		}
	}
	return candidates, nil
}
