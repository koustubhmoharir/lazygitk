package git_commands

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

type BranchPushSubmodules uint8

const (
	BranchPushNoSubmoduleRecursion BranchPushSubmodules = iota
	BranchPushCheckSubmodules
)

// BranchPushOptions describes explicit choices for one local branch and one
// destination ref. Tags and submodule pushes are outside this action's scope.
type BranchPushOptions struct {
	Remote, DestinationRef, LeaseRef string
	ForceWithLease, SetUpstream      bool
	RequireCurrent                   bool
	Submodules                       BranchPushSubmodules
}

// BranchPushPlan remains bound to its original backend and reviewed inputs.
// It is not persisted. URLs may contain credentials; display must redact them.
type BranchPushPlan struct {
	owner                                            *GitCommon
	name, sourceRef, sourceOID, remoteSignature      string
	configuredURLs, leaseRef, leaseOID               string
	trackingSignature, previousRemote, previousMerge string
	reservation                                      BranchSyncReservation
	options                                          BranchPushOptions
	prepared, multipleURLsApproved                   bool
}

func (plan BranchPushPlan) Name() string                       { return plan.name }
func (plan BranchPushPlan) SourceOID() string                  { return plan.sourceOID }
func (plan BranchPushPlan) Options() BranchPushOptions         { return plan.options }
func (plan BranchPushPlan) LeaseRef() string                   { return plan.leaseRef }
func (plan BranchPushPlan) LeaseOID() string                   { return plan.leaseOID }
func (plan BranchPushPlan) PreviousRemote() string             { return plan.previousRemote }
func (plan BranchPushPlan) Reservation() BranchSyncReservation { return plan.reservation }

func (plan BranchPushPlan) ConfiguredPushURLs() []string {
	if plan.configuredURLs == "" {
		return nil
	}
	return strings.Split(plan.configuredURLs, "\x00")
}

func (plan BranchPushPlan) PreviousMergeRefs() []string {
	if plan.previousMerge == "" {
		return nil
	}
	return strings.Split(plan.previousMerge, "\x00")
}

func (plan BranchPushPlan) WithMultipleURLsApproved() BranchPushPlan {
	plan.multipleURLsApproved = true
	return plan
}

var (
	ErrBranchPushMultipleURLs      = errors.New("this remote has multiple configured push destinations; review and explicitly approve them first")
	ErrBranchPushLeaseChoice       = errors.New("force-with-lease needs a locally reviewed destination tip; fetch and review it or select a known reference explicitly")
	ErrBranchPushUpstreamAmbiguous = errors.New("the selected branch has ambiguous or non-local upstream configuration; push without changing tracking")
)

func (self *SyncCommands) PrepareBranchPush(name, expectedOID string, opts BranchPushOptions) (BranchPushPlan, error) {
	if err := validateObjectID(expectedOID); err != nil {
		return BranchPushPlan{}, err
	}
	info, err := self.InspectBranchSync(name)
	if err != nil {
		return BranchPushPlan{}, err
	}
	if info.OID != strings.ToLower(expectedOID) || (opts.RequireCurrent && info.HeadRef != info.FullRef) {
		return BranchPushPlan{}, ErrBranchSyncChanged
	}
	if !strings.HasPrefix(opts.DestinationRef, "refs/heads/") || opts.DestinationRef == "refs/heads/" {
		return BranchPushPlan{}, fmt.Errorf("choose a full destination branch ref under refs/heads/")
	}
	if err := self.cmd.New(NewGitCmd("check-ref-format").Arg(opts.DestinationRef).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
		return BranchPushPlan{}, fmt.Errorf("invalid destination branch ref: %w", err)
	}
	if opts.Submodules != BranchPushNoSubmoduleRecursion && opts.Submodules != BranchPushCheckSubmodules {
		return BranchPushPlan{}, fmt.Errorf("branch-only push cannot automatically push submodules")
	}
	if opts.ForceWithLease && self.UserConfig().Git.DisableForcePushing {
		return BranchPushPlan{}, fmt.Errorf("force pushing is disabled in lazygit configuration")
	}
	if !opts.ForceWithLease && opts.LeaseRef != "" {
		return BranchPushPlan{}, fmt.Errorf("a lease reference requires force-with-lease")
	}
	var trackingSignature, previousRemote, previousMerge string
	if opts.SetUpstream {
		trackingSignature, previousRemote, previousMerge, err = self.branchPushTrackingSnapshot(name)
		if err != nil {
			return BranchPushPlan{}, err
		}
	}
	remote, err := self.InspectBranchSyncRemote(opts.Remote)
	if err != nil {
		return BranchPushPlan{}, err
	}
	if remote.Mirror {
		return BranchPushPlan{}, fmt.Errorf("remote %q is configured as a mirror; choose a non-mirror remote for a branch-only push", opts.Remote)
	}
	mapping, err := self.branchSyncTrackingRefs(remote.FetchRefspecs, opts.DestinationRef)
	if err != nil {
		return BranchPushPlan{}, err
	}
	branches := NewBranchCommands(self.GitCommon)
	for _, ref := range mapping.refs {
		if err := self.checkBranchSyncTrackingWrite(ref); err != nil {
			return BranchPushPlan{}, err
		}
	}
	urls := remote.PushURLs
	if len(urls) == 0 {
		urls = remote.URLs
	}
	plan := BranchPushPlan{
		owner: self.GitCommon, name: name, sourceRef: info.FullRef, sourceOID: info.OID,
		remoteSignature: remote.Signature, configuredURLs: strings.Join(urls, "\x00"),
		reservation: info.Reservation, options: opts,
		trackingSignature: trackingSignature, previousRemote: previousRemote, previousMerge: previousMerge,
	}
	if opts.ForceWithLease {
		plan.leaseRef = opts.LeaseRef
		if plan.leaseRef == "" {
			if mapping.excluded || mapping.preferred == "" {
				return BranchPushPlan{}, ErrBranchPushLeaseChoice
			}
			plan.leaseRef = mapping.preferred
		}
		if !strings.HasPrefix(plan.leaseRef, "refs/") {
			return BranchPushPlan{}, fmt.Errorf("choose a full, locally reviewed lease reference")
		}
		if err := self.cmd.New(NewGitCmd("check-ref-format").Arg(plan.leaseRef).ToArgv()).
			SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
			return BranchPushPlan{}, err
		}
		state, err := branches.readPlacementRef(plan.leaseRef)
		if err != nil {
			return BranchPushPlan{}, err
		}
		if !state.exists || state.symbolic != "" {
			return BranchPushPlan{}, ErrBranchPushLeaseChoice
		}
		if err := branches.validateLiteralCommitOID(state.oid); err != nil {
			return BranchPushPlan{}, fmt.Errorf("lease reference does not point directly to a commit: %w", err)
		}
		plan.leaseOID = state.oid
	}
	plan.prepared = true
	return plan, nil
}

// BranchPushCommand validates approval without refreshing its lease through a
// network operation. The normal Git concurrency model applies after launch.
func (self *SyncCommands) BranchPushCommand(plan BranchPushPlan, task gocui.Task) (*oscommands.CmdObj, error) {
	if !plan.prepared || plan.owner != self.GitCommon {
		return nil, fmt.Errorf("push plan was not prepared by this repository instance")
	}
	fresh, err := self.PrepareBranchPush(plan.name, plan.sourceOID, plan.options)
	if err != nil {
		return nil, err
	}
	approval := plan.multipleURLsApproved
	plan.multipleURLsApproved = false
	if fresh != plan {
		return nil, ErrBranchSyncChanged
	}
	if len(plan.ConfiguredPushURLs()) > 1 && !approval {
		return nil, ErrBranchPushMultipleURLs
	}
	recurse := "no"
	if plan.options.Submodules == BranchPushCheckSubmodules {
		recurse = "check"
	}
	args := NewGitCmd("push").Config("push.autoSetupRemote=false").
		Arg("--porcelain", "--no-follow-tags", "--recurse-submodules="+recurse).
		ArgIf(plan.options.ForceWithLease, "--force-with-lease="+plan.options.DestinationRef+":"+plan.leaseOID).
		ArgIf(plan.options.SetUpstream, "--set-upstream").
		Arg("--", plan.options.Remote, plan.sourceRef+":"+plan.options.DestinationRef)
	return self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()).
		DisableLockRetries().PromptOnCredentialRequest(task), nil
}

func (self *SyncCommands) branchPushTrackingSnapshot(name string) (signature, remote, merges string, err error) {
	entries, err := self.branchSyncConfigEntries()
	if err != nil {
		return "", "", "", err
	}
	primary, _, err := self.cmd.New(NewGitCmd("config").Arg("--local", "--no-includes", "--null", "--list").ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return "", "", "", fmt.Errorf("inspect primary local tracking configuration: %w", err)
	}
	localEntries, err := parseBranchSyncConfig(primary)
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
	effectiveEntries, localTrackingEntries := filter(entries), filter(localEntries)
	// Git's --set-upstream writes the primary local config. If effective values
	// come from another config source, it cannot safely replace them there.
	if !slices.Equal(effectiveEntries, localTrackingEntries) {
		return "", "", "", ErrBranchPushUpstreamAmbiguous
	}
	remoteValues, mergeValues := []string{}, []string{}
	for _, entry := range effectiveEntries {
		key, found := strings.CutPrefix(entry.key, prefix)
		if !found || (key != "remote" && key != "merge") || !entry.hasValue || entry.value == "" {
			return "", "", "", ErrBranchPushUpstreamAmbiguous
		}
		if key == "remote" {
			remoteValues = append(remoteValues, entry.value)
		} else {
			mergeValues = append(mergeValues, entry.value)
		}
	}
	if len(remoteValues) > 1 || len(mergeValues) > 1 {
		return "", "", "", ErrBranchPushUpstreamAmbiguous
	}
	// Since effective and primary-local entries were proven equal above, this
	// fingerprint is also the primary-local tracking fingerprint. A later move
	// to an include/global/worktree source is rejected during fresh preparation.
	return branchPushTrackingFingerprint(effectiveEntries), firstBranchPushValue(remoteValues),
		strings.Join(mergeValues, "\x00"), nil
}

func branchPushTrackingFingerprint(entries []branchSyncConfigEntry) string {
	digest := sha256.New()
	for _, entry := range entries {
		writeBranchSyncConfigEntry(digest, entry)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func firstBranchPushValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
