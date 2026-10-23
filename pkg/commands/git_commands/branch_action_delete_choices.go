package git_commands

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var ErrBranchDeleteNoUpstream = errors.New("the selected branch has no configured upstream; choose a remote branch explicitly")

// BranchActionRemoteDeleteChoices is inspection data, not approval to delete
// its first entry. Local-branch upstreams retain every merge ref. Remote-pane
// inspection inverts the configured fetch mapping instead of stripping a
// display prefix. The GUI must explicitly resolve multiple candidates.
type BranchActionRemoteDeleteChoices struct {
	target          BranchActionTarget
	remote          string
	refs            []string
	configSignature string
	localUpstreams  bool
}

func (choices BranchActionRemoteDeleteChoices) Target() BranchActionTarget { return choices.target }
func (choices BranchActionRemoteDeleteChoices) Remote() string             { return choices.remote }
func (choices BranchActionRemoteDeleteChoices) Refs() []string             { return slices.Clone(choices.refs) }

func (self *SyncCommands) InspectBranchActionLocalDeleteUpstreams(target BranchActionTarget) (BranchActionRemoteDeleteChoices, error) {
	branches := NewBranchCommands(self.GitCommon)
	if err := branches.CheckBranchActionTarget(target); err != nil {
		return BranchActionRemoteDeleteChoices{}, err
	}
	if !strings.HasPrefix(target.SourceRef(), "refs/heads/") {
		return BranchActionRemoteDeleteChoices{}, fmt.Errorf("choose a local branch to inspect its upstreams")
	}
	name := strings.TrimPrefix(target.SourceRef(), "refs/heads/")
	info, err := self.InspectBranchSync(name)
	if err != nil {
		return BranchActionRemoteDeleteChoices{}, err
	}
	if info.OID != target.SourceOID() || info.HeadRef != target.HeadRef() || info.HeadOID != target.HeadOID() {
		return BranchActionRemoteDeleteChoices{}, ErrBranchActionChanged
	}
	entries, err := self.branchSyncConfigEntries()
	if err != nil {
		return BranchActionRemoteDeleteChoices{}, err
	}
	choices := BranchActionRemoteDeleteChoices{target: target, remote: info.UpstreamRemote, localUpstreams: true}
	for _, entry := range entries {
		if entry.key != "branch."+name+".merge" {
			continue
		}
		if !entry.hasValue || !strings.HasPrefix(entry.value, "refs/") {
			return BranchActionRemoteDeleteChoices{}, fmt.Errorf("upstream merge refs must be fully qualified")
		}
		if err := self.checkBranchSyncRef(entry.value, false); err != nil {
			return BranchActionRemoteDeleteChoices{}, err
		}
		if !slices.Contains(choices.refs, entry.value) {
			choices.refs = append(choices.refs, entry.value)
		}
	}
	if choices.remote == "" || len(choices.refs) == 0 {
		return BranchActionRemoteDeleteChoices{}, ErrBranchDeleteNoUpstream
	}
	if choices.remote != "." {
		if _, err := self.InspectBranchSyncRemote(choices.remote); err != nil {
			return BranchActionRemoteDeleteChoices{}, err
		}
	}
	choices.configSignature, err = branches.deleteConfigSignature()
	if err != nil {
		return BranchActionRemoteDeleteChoices{}, err
	}
	if err := branches.CheckBranchActionTarget(target); err != nil {
		return BranchActionRemoteDeleteChoices{}, err
	}
	return choices, nil
}

func (self *SyncCommands) InspectBranchActionRemoteDeleteChoices(target BranchActionTarget, remoteName string) (BranchActionRemoteDeleteChoices, error) {
	branches := NewBranchCommands(self.GitCommon)
	if err := branches.CheckBranchActionTarget(target); err != nil {
		return BranchActionRemoteDeleteChoices{}, err
	}
	remote, err := self.InspectBranchSyncRemote(remoteName)
	if err != nil {
		return BranchActionRemoteDeleteChoices{}, err
	}
	refs, err := self.branchSyncRemoteRefsForTracking(remote.FetchRefspecs, target.SourceRef())
	if err != nil {
		return BranchActionRemoteDeleteChoices{}, err
	}
	if len(refs) == 0 {
		return BranchActionRemoteDeleteChoices{}, fmt.Errorf("configured fetch mappings do not identify a remote source for this cached ref; choose the remote branch explicitly")
	}
	signature, err := branches.deleteConfigSignature()
	if err != nil {
		return BranchActionRemoteDeleteChoices{}, err
	}
	if err := branches.CheckBranchActionTarget(target); err != nil {
		return BranchActionRemoteDeleteChoices{}, err
	}
	return BranchActionRemoteDeleteChoices{target: target, remote: remoteName, refs: refs, configSignature: signature}, nil
}

func (self *SyncCommands) CheckBranchActionRemoteDeleteChoices(choices BranchActionRemoteDeleteChoices) error {
	var fresh BranchActionRemoteDeleteChoices
	var err error
	if choices.localUpstreams {
		fresh, err = self.InspectBranchActionLocalDeleteUpstreams(choices.target)
	} else {
		fresh, err = self.InspectBranchActionRemoteDeleteChoices(choices.target, choices.remote)
	}
	if err != nil {
		return err
	}
	if fresh.configSignature != choices.configSignature || fresh.remote != choices.remote || !slices.Equal(fresh.refs, choices.refs) {
		return ErrBranchActionChanged
	}
	return nil
}
