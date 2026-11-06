package git_commands

import (
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
	// leaseFromUnmatchedTracking records that the default lease is the fetched
	// tracking ref while the push URLs differ from the fetch URLs.
	leaseFromUnmatchedTracking bool
}

func (plan BranchPushPlan) Name() string                       { return plan.name }
func (plan BranchPushPlan) SourceOID() string                  { return plan.sourceOID }
func (plan BranchPushPlan) Options() BranchPushOptions         { return plan.options }
func (plan BranchPushPlan) LeaseRef() string                   { return plan.leaseRef }
func (plan BranchPushPlan) LeaseOID() string                   { return plan.leaseOID }
func (plan BranchPushPlan) PreviousRemote() string             { return plan.previousRemote }
func (plan BranchPushPlan) Reservation() BranchSyncReservation { return plan.reservation }

// LeaseFromUnmatchedTracking reports whether the lease was defaulted from the
// fetched tracking ref although the push destination is not the fetched one.
func (plan BranchPushPlan) LeaseFromUnmatchedTracking() bool { return plan.leaseFromUnmatchedTracking }

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
	ErrBranchPushUpstreamAmbiguous = ErrBranchUpstreamAmbiguous
	// ErrBranchPushDiverged means the destination's local tracking ref has
	// commits the source lacks, so a plain push would be rejected. Only a
	// force-with-lease push, reviewed against that tip, can replace them.
	ErrBranchPushDiverged = errors.New("the destination's local tracking ref has commits this branch does not contain; a plain push would be rejected")
)

// BranchPushDestinationDiverged reports whether the preferred local tracking
// ref for remote and destinationRef holds commits that sourceOID does not
// contain. It reads local refs only and never fetches. It returns the tracking
// ref it checked when diverged is true. When the push URLs differ from the
// fetch URLs the tracking ref does not describe the destination, so it never
// reports divergence.
func (self *SyncCommands) BranchPushDestinationDiverged(sourceOID, remote, destinationRef string) (trackingRef string, diverged bool, err error) {
	if err := validateObjectID(sourceOID); err != nil {
		return "", false, err
	}
	if remote == "" {
		return "", false, nil
	}
	remoteInfo, err := self.InspectBranchSyncRemote(remote)
	if err != nil {
		return "", false, err
	}
	mapping, err := self.branchSyncTrackingRefs(remoteInfo.FetchRefspecs, destinationRef)
	if err != nil {
		return "", false, err
	}
	fetched, err := self.branchPushDestinationFetched(remote)
	if err != nil {
		return "", false, err
	}
	if !fetched {
		return "", false, nil
	}
	return self.branchPushDivergedFromMapping(sourceOID, mapping)
}

// branchPushDestinationFetched reports whether the remote pushes to the same
// effective URLs that it fetches from, after insteadOf and pushInsteadOf
// rewriting. Only then does a fetched tracking ref describe the push
// destination. Otherwise Git's own fast-forward check is authoritative.
// It reads configuration only and never contacts a remote.
func (self *SyncCommands) branchPushDestinationFetched(remote string) (bool, error) {
	fetchURLs, err := self.branchSyncEffectiveURLs(remote, false)
	if err != nil {
		return false, err
	}
	pushURLs, err := self.branchSyncEffectiveURLs(remote, true)
	if err != nil {
		return false, err
	}
	return slices.Equal(fetchURLs, pushURLs), nil
}

func (self *SyncCommands) branchSyncEffectiveURLs(remote string, push bool) ([]string, error) {
	// Credential-bearing URLs appear in this output, so the query is not logged.
	stdout, _, err := self.cmd.New(NewGitCmd("remote").Arg("get-url").ArgIf(push, "--push").Arg("--all", remote).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		return nil, fmt.Errorf("inspect remote URLs: %w", err)
	}
	stdout = trimTerminalLineEnding(stdout)
	if stdout == "" {
		return nil, nil
	}
	return strings.Split(stdout, "\n"), nil
}

func (self *SyncCommands) branchPushDivergedFromMapping(sourceOID string, mapping branchSyncTrackingMapping) (string, bool, error) {
	if mapping.excluded || mapping.preferred == "" {
		return "", false, nil
	}
	branches := NewBranchCommands(self.GitCommon)
	state, err := branches.readPlacementRef(mapping.preferred)
	if err != nil {
		return "", false, err
	}
	if !state.exists || state.symbolic != "" {
		return "", false, nil
	}
	if err := branches.validateLiteralCommitOID(state.oid); err != nil {
		return "", false, fmt.Errorf("local tracking ref %q does not point directly to a commit: %w", mapping.preferred, err)
	}
	ancestor, err := self.branchSyncIsCommitAncestor(state.oid, sourceOID)
	if err != nil {
		return "", false, err
	}
	return mapping.preferred, !ancestor, nil
}

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
		trackingSignature, previousRemote, previousMerge, err = self.branchUpstreamTrackingSnapshot(name)
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
	fetched, err := self.branchPushDestinationFetched(opts.Remote)
	if err != nil {
		return BranchPushPlan{}, err
	}
	if !opts.ForceWithLease && fetched {
		trackingRef, diverged, err := self.branchPushDivergedFromMapping(info.OID, mapping)
		if err != nil {
			return BranchPushPlan{}, err
		}
		if diverged {
			return BranchPushPlan{}, fmt.Errorf("%s: %w", trackingRef, ErrBranchPushDiverged)
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
			plan.leaseFromUnmatchedTracking = !fetched
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
	if plan.options.SetUpstream {
		// Fork: explicit tracking consent does not authorize Git's automatic
		// setup policy to overwrite this branch's existing rebase preference.
		args.Config("branch.autoSetupRebase=never")
	}
	return self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()).
		DisableLockRetries().PromptOnCredentialRequest(task), nil
}
