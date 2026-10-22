package git_commands

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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
	ErrBranchPushMultipleURLs = errors.New("this remote has multiple configured push destinations; review and explicitly approve them first")
	ErrBranchPushLeaseChoice  = errors.New("force-with-lease needs a locally reviewed destination tip; fetch and review it or select a known reference explicitly")
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
		if err := self.checkBranchPushTrackingWrite(ref); err != nil {
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
	if opts.SetUpstream {
		plan.trackingSignature, plan.previousRemote, plan.previousMerge, err = self.branchPushTrackingSnapshot(name)
		if err != nil {
			return BranchPushPlan{}, err
		}
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

func (self *SyncCommands) checkBranchPushTrackingWrite(ref string) error {
	if !strings.HasPrefix(ref, "refs/") || ref == "refs/stash" {
		return fmt.Errorf("unsafe local tracking destination %q", ref)
	}
	if err := self.cmd.New(NewGitCmd("check-ref-format").Arg(ref).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
		return fmt.Errorf("invalid local tracking destination: %w", err)
	}
	for _, namespace := range []string{"refs/heads", "refs/tags", "refs/notes", "refs/replace", "refs/bisect", "refs/rewritten", "refs/worktree", "refs/namespaces", "refs/stash"} {
		if ref == namespace || strings.HasPrefix(ref, namespace+"/") {
			return fmt.Errorf("remote fetch mapping could make push modify %q; choose a tracking-only mapping", ref)
		}
	}
	state, err := NewBranchCommands(self.GitCommon).readPlacementRef(ref)
	if err != nil {
		return err
	}
	if state.symbolic != "" {
		return fmt.Errorf("tracking destination %q is a symbolic alias; it cannot be updated by a branch-only push", ref)
	}
	return nil
}

func (self *SyncCommands) branchPushTrackingSnapshot(name string) (signature, remote, merges string, err error) {
	entries, err := self.branchSyncConfigEntries()
	if err != nil {
		return "", "", "", err
	}
	digest := sha256.New()
	prefix := "branch." + name + "."
	mergeValues := []string{}
	for _, entry := range entries {
		key, found := strings.CutPrefix(entry.key, prefix)
		if !found || (key != "remote" && key != "merge") {
			continue
		}
		writeBranchSyncConfigEntry(digest, entry)
		if key == "remote" {
			remote = entry.value
		} else {
			mergeValues = append(mergeValues, entry.value)
		}
	}
	return hex.EncodeToString(digest.Sum(nil)), remote, strings.Join(mergeValues, "\x00"), nil
}
