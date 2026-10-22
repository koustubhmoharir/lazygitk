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

// BranchUpstreamOptions describes an explicit upstream metadata change.
type BranchUpstreamOptions struct {
	TrackingRef string
	Unset       bool
}

// BranchUpstreamPlan captures only metadata inputs; it never reserves or moves refs.
type BranchUpstreamPlan struct {
	owner                            *GitCommon
	branchName, sourceRef, sourceOID string
	trackingRef, targetOID           string
	trackingSignature, remoteProfile string
	previousRemote, previousMerge    string
	prepared                         bool
}

func (plan BranchUpstreamPlan) Name() string           { return plan.branchName }
func (plan BranchUpstreamPlan) SourceOID() string      { return plan.sourceOID }
func (plan BranchUpstreamPlan) TrackingRef() string    { return plan.trackingRef }
func (plan BranchUpstreamPlan) TargetOID() string      { return plan.targetOID }
func (plan BranchUpstreamPlan) Unset() bool            { return plan.trackingRef == "" }
func (plan BranchUpstreamPlan) PreviousRemote() string { return plan.previousRemote }
func (plan BranchUpstreamPlan) PreviousMergeRefs() []string {
	if plan.previousMerge == "" {
		return nil
	}
	return strings.Split(plan.previousMerge, "\x00")
}

var ErrBranchUpstreamAmbiguous = errors.New("the selected branch has ambiguous or inherited upstream configuration")

func (self *SyncCommands) PrepareBranchUpstream(name, expectedOID string, opts BranchUpstreamOptions) (BranchUpstreamPlan, error) {
	if opts.Unset && opts.TrackingRef != "" {
		return BranchUpstreamPlan{}, fmt.Errorf("unset upstream cannot include a tracking reference")
	}
	if !opts.Unset && opts.TrackingRef == "" {
		return BranchUpstreamPlan{}, fmt.Errorf("choose a full tracking reference")
	}
	if err := validateObjectID(expectedOID); err != nil {
		return BranchUpstreamPlan{}, err
	}
	info, err := self.InspectBranchSync(name)
	if err != nil {
		return BranchUpstreamPlan{}, err
	}
	expectedOID = strings.ToLower(expectedOID)
	if info.OID != expectedOID {
		return BranchUpstreamPlan{}, ErrBranchSyncChanged
	}
	var targetOID string
	if !opts.Unset {
		if !strings.HasPrefix(opts.TrackingRef, "refs/") || opts.TrackingRef == "refs/" {
			return BranchUpstreamPlan{}, fmt.Errorf("choose a full tracking ref")
		}
		if err := self.cmd.New(NewGitCmd("check-ref-format").Arg(opts.TrackingRef).ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
			return BranchUpstreamPlan{}, fmt.Errorf("invalid tracking ref: %w", err)
		}
		if opts.TrackingRef == info.FullRef {
			return BranchUpstreamPlan{}, fmt.Errorf("a branch cannot track itself")
		}
		state, err := NewBranchCommands(self.GitCommon).readPlacementRef(opts.TrackingRef)
		if err != nil {
			return BranchUpstreamPlan{}, err
		}
		if !state.exists || state.symbolic != "" {
			return BranchUpstreamPlan{}, fmt.Errorf("tracking ref must exist as a direct ref")
		}
		if err := NewBranchCommands(self.GitCommon).validateLiteralCommitOID(state.oid); err != nil {
			return BranchUpstreamPlan{}, fmt.Errorf("tracking ref must point directly to a commit: %w", err)
		}
		targetOID = strings.ToLower(state.oid)
	}
	trackingSignature, previousRemote, previousMerge, err := self.branchUpstreamTrackingSnapshot(name)
	if err != nil {
		return BranchUpstreamPlan{}, err
	}
	if opts.Unset && previousRemote == "" && previousMerge == "" {
		return BranchUpstreamPlan{}, fmt.Errorf("branch has no upstream configuration to unset")
	}
	profile, err := self.branchUpstreamRemoteProfile()
	if err != nil {
		return BranchUpstreamPlan{}, err
	}
	return BranchUpstreamPlan{
		owner: self.GitCommon, branchName: name, sourceRef: info.FullRef, sourceOID: info.OID,
		trackingRef: opts.TrackingRef, targetOID: targetOID, trackingSignature: trackingSignature,
		remoteProfile: profile, previousRemote: previousRemote, previousMerge: previousMerge, prepared: true,
	}, nil
}

func (self *SyncCommands) BranchUpstreamCommand(plan BranchUpstreamPlan) (*oscommands.CmdObj, error) {
	if !plan.prepared || plan.owner != self.GitCommon {
		return nil, fmt.Errorf("upstream plan was not prepared by this repository instance")
	}
	opts := BranchUpstreamOptions{TrackingRef: plan.trackingRef, Unset: plan.trackingRef == ""}
	fresh, err := self.PrepareBranchUpstream(plan.branchName, plan.sourceOID, opts)
	if err != nil {
		return nil, err
	}
	if fresh != plan {
		return nil, ErrBranchSyncChanged
	}
	cmd := NewGitCmd("branch")
	if opts.Unset {
		cmd.Arg("--unset-upstream", "--", plan.branchName)
	} else {
		// Fork: changing tracking must not also rewrite branch.rebase through
		// Git's automatic setup policy; preserve the user's existing preference.
		cmd.Config("branch.autoSetupRebase=never")
		cmd.Arg("--set-upstream-to="+plan.trackingRef, "--", plan.branchName)
	}
	return self.cmd.New(cmd.ToArgv()).SetWd(self.repoPaths.WorktreePath()).DisableLockRetries(), nil
}

func (self *SyncCommands) branchUpstreamTrackingSnapshot(name string) (signature, remote, merges string, err error) {
	entries, err := self.branchSyncConfigEntries()
	if err != nil {
		return "", "", "", err
	}
	primary, _, err := self.cmd.New(NewGitCmd("config").Arg("--local", "--no-includes", "--null", "--list").ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return "", "", "", fmt.Errorf("inspect primary local tracking configuration: %w", err)
	}
	local, err := parseBranchSyncConfig(primary)
	if err != nil {
		return "", "", "", err
	}
	prefix := "branch." + name + "."
	filter := func(source []branchSyncConfigEntry) []branchSyncConfigEntry {
		filtered := []branchSyncConfigEntry{}
		for _, entry := range source {
			key, found := strings.CutPrefix(entry.key, prefix)
			if found && (key == "remote" || key == "merge") {
				filtered = append(filtered, entry)
			}
		}
		return filtered
	}
	effectiveEntries, localTrackingEntries := filter(entries), filter(local)
	// Git's --set-upstream writes the primary local config. If effective values
	// come from another config source, it cannot safely replace them there.
	if !slices.Equal(effectiveEntries, localTrackingEntries) {
		return "", "", "", ErrBranchUpstreamAmbiguous
	}
	var remotes, mergeValues []string
	for _, entry := range effectiveEntries {
		key, found := strings.CutPrefix(entry.key, prefix)
		if !found || (key != "remote" && key != "merge") || !entry.hasValue || entry.value == "" {
			return "", "", "", ErrBranchUpstreamAmbiguous
		}
		if key == "remote" {
			remotes = append(remotes, entry.value)
		} else {
			mergeValues = append(mergeValues, entry.value)
		}
	}
	if len(remotes) > 1 || len(mergeValues) > 1 {
		return "", "", "", ErrBranchUpstreamAmbiguous
	}
	// Since effective and primary-local entries were proven equal above, this
	// fingerprint is also the primary-local tracking fingerprint. A later move
	// to an include/global/worktree source is rejected during fresh preparation.
	return branchUpstreamTrackingFingerprint(effectiveEntries), firstBranchUpstreamValue(remotes),
		strings.Join(mergeValues, "\x00"), nil
}

func branchUpstreamTrackingFingerprint(entries []branchSyncConfigEntry) string {
	digest := sha256.New()
	for _, entry := range entries {
		writeBranchSyncConfigEntry(digest, entry)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func firstBranchUpstreamValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (self *SyncCommands) branchUpstreamRemoteProfile() (string, error) {
	entries, err := self.branchSyncConfigEntries()
	if err != nil {
		return "", err
	}
	digest := sha256.New()
	for _, entry := range entries {
		if strings.HasPrefix(entry.key, "remote.") {
			writeBranchSyncConfigEntry(digest, entry)
		}
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
