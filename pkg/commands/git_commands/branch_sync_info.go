package git_commands

import (
	"fmt"
	"strings"
)

// BranchSyncReservation describes a worktree or operation that currently
// reserves a branch. It is inspection data, not authorization to mutate it.
type BranchSyncReservation struct {
	GitDir  string
	Current bool
	Kind    string
}

// BranchSyncInfo is display and inspection data for a local branch. Callers
// must revalidate all values before a mutation. PushRef may be only the first
// configured push destination and is not guaranteed to be unique.
type BranchSyncInfo struct {
	Name                string
	FullRef             string
	OID                 string
	HeadRef             string
	HeadOID             string
	UpstreamRemote      string
	UpstreamRef         string
	UpstreamTrackingRef string
	PushRemote          string
	PushRef             string
	// SoleRemote is the only configured remote, as git remote reports it in
	// the same snapshot as the branch metadata above. It is empty when zero or
	// several remotes are configured. A string keeps the struct comparable.
	SoleRemote  string
	Reservation BranchSyncReservation
}

// InspectBranchSync reads sync metadata for a literal local branch name.
// The returned snapshot is not a plan or authorization for a mutating command;
// callers must freshly revalidate branch, remote configuration, and occupancy.
func (self *SyncCommands) InspectBranchSync(name string) (BranchSyncInfo, error) {
	branches := NewBranchCommands(self.GitCommon)
	if err := branches.validatePlacementBranchName(name); err != nil {
		return BranchSyncInfo{}, err
	}
	fullRef := "refs/heads/" + name
	state, err := branches.readPlacementRef(fullRef)
	if err != nil {
		return BranchSyncInfo{}, err
	}
	if state.symbolic != "" {
		return BranchSyncInfo{}, fmt.Errorf("branch %q is a symbolic alias; choose a direct local branch", name)
	}
	if !state.exists {
		return BranchSyncInfo{}, fmt.Errorf("local branch %q does not exist", name)
	}
	if err := branches.validateLiteralCommitOID(state.oid); err != nil {
		return BranchSyncInfo{}, fmt.Errorf("local branch %q does not point to a commit: %w", name, err)
	}
	headRef, headOID, err := branches.namedCheckoutHead()
	if err != nil {
		return BranchSyncInfo{}, err
	}
	const format = "%(refname)%00%(objectname)%00%(upstream)%00%(upstream:remotename)%00%(upstream:remoteref)%00%(push:remotename)%00%(push:remoteref)%00"
	stdout, _, err := self.cmd.New(NewGitCmd("for-each-ref").Arg("--format="+format, "--", fullRef).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return BranchSyncInfo{}, err
	}
	info, err := parseBranchSyncInfoRows(stdout, fullRef)
	if err != nil {
		return BranchSyncInfo{}, err
	}
	if info.OID != state.oid {
		return BranchSyncInfo{}, fmt.Errorf("branch %q changed while being inspected; retry inspection", name)
	}
	remotes, err := self.remoteNames()
	if err != nil {
		return BranchSyncInfo{}, err
	}
	if len(remotes) == 1 {
		info.SoleRemote = remotes[0]
	}
	info.Name = name
	info.HeadRef, info.HeadOID = headRef, headOID
	occupation, err := branches.branchOccupancy(fullRef, true)
	if err != nil {
		return BranchSyncInfo{}, err
	}
	if occupation != nil {
		info.Reservation = BranchSyncReservation{GitDir: occupation.GitDir, Current: occupation.Current, Kind: occupation.Kind}
	}
	return info, nil
}

// remoteNames lists configured remote names. Git forbids whitespace in remote
// names, so splitting on whitespace is exact.
func (self *SyncCommands) remoteNames() ([]string, error) {
	stdout, err := self.cmd.New(NewGitCmd("remote").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutput()
	if err != nil {
		return nil, err
	}
	return strings.Fields(stdout), nil
}

func parseBranchSyncInfoRows(stdout, exactRef string) (BranchSyncInfo, error) {
	var result BranchSyncInfo
	found := false
	for offset := 0; offset < len(stdout); {
		fields := make([]string, 0, 7)
		for len(fields) < 7 {
			relative := strings.IndexByte(stdout[offset:], '\x00')
			if relative < 0 {
				return BranchSyncInfo{}, fmt.Errorf("malformed branch sync query output")
			}
			end := offset + relative
			fields = append(fields, stdout[offset:end])
			offset = end + 1
		}
		if offset < len(stdout) && stdout[offset] == '\r' {
			offset++
		}
		if offset >= len(stdout) || stdout[offset] != '\n' {
			return BranchSyncInfo{}, fmt.Errorf("malformed branch sync query record terminator")
		}
		offset++
		if fields[0] == "" {
			return BranchSyncInfo{}, fmt.Errorf("malformed branch sync query refname")
		}
		if err := validateObjectID(fields[1]); err != nil {
			return BranchSyncInfo{}, fmt.Errorf("malformed branch sync object ID: %w", err)
		}
		if fields[0] != exactRef {
			continue
		}
		if found {
			return BranchSyncInfo{}, fmt.Errorf("branch sync query returned duplicate exact matches")
		}
		found = true
		result = BranchSyncInfo{FullRef: fields[0], OID: strings.ToLower(fields[1]), UpstreamTrackingRef: fields[2], UpstreamRemote: fields[3], UpstreamRef: fields[4], PushRemote: fields[5], PushRef: fields[6]}
	}
	if !found {
		return BranchSyncInfo{}, fmt.Errorf("local branch disappeared while being inspected; retry inspection")
	}
	return result, nil
}
