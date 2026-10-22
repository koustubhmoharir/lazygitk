package git_commands

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// PullIntegrationMethod describes an explicit choice after fetching. Default
// resolves native pull configuration; the other choices override that default.
type PullIntegrationMethod uint8

const (
	PullIntegrationDefault PullIntegrationMethod = iota
	PullIntegrationFastForwardOnly
	PullIntegrationMerge
	PullIntegrationMergeNoFF
	PullIntegrationRebase
	PullIntegrationRebaseMerges
	PullIntegrationRebaseInteractive
	// PullIntegrationConfiguredMerge is a resolved default, not an explicit
	// choice. The UI must display MergeOptions alongside this method.
	PullIntegrationConfiguredMerge
)

// CurrentPullReview is captured before fetching, while the old tracking reflog
// can still identify the branch's fork point. It authorizes no integration.
type CurrentPullReview struct {
	owner                            *GitCommon
	name, sourceOID, policySignature string
	forkPoint                        string
	prepared                         bool
}

func (review CurrentPullReview) Name() string      { return review.name }
func (review CurrentPullReview) SourceOID() string { return review.sourceOID }

// CurrentPullIntegrationPlan pins an incoming commit, never a live tracking ref.
// Integration uses native merge/rebase, not another network pull.
type CurrentPullIntegrationPlan struct {
	review                    CurrentPullReview
	targetOID, replayBaseOID  string
	mergeOptions, mergeFF     string
	requested, method         PullIntegrationMethod
	ignoredPaths              []string
	ignoredFingerprint        string
	ignoredApproved, prepared bool
}

func (plan CurrentPullIntegrationPlan) Name() string { return plan.review.name }

func (plan CurrentPullIntegrationPlan) SourceOID() string                      { return plan.review.sourceOID }
func (plan CurrentPullIntegrationPlan) TargetOID() string                      { return plan.targetOID }
func (plan CurrentPullIntegrationPlan) ReplayBaseOID() string                  { return plan.replayBaseOID }
func (plan CurrentPullIntegrationPlan) Method() PullIntegrationMethod          { return plan.method }
func (plan CurrentPullIntegrationPlan) RequestedMethod() PullIntegrationMethod { return plan.requested }
func (plan CurrentPullIntegrationPlan) MergeOptions() string                   { return plan.mergeOptions }
func (plan CurrentPullIntegrationPlan) ConfiguredMergeFF() string              { return plan.mergeFF }
func (plan CurrentPullIntegrationPlan) UsesConfiguredRebaseOptions() bool {
	return plan.requested == PullIntegrationDefault && isPullReviewRebase(plan.method)
}

func (plan CurrentPullIntegrationPlan) IgnoredFiles() []string {
	return slices.Clone(plan.ignoredPaths)
}

func (plan CurrentPullIntegrationPlan) WithIgnoredFilesApproved() CurrentPullIntegrationPlan {
	plan.ignoredApproved = true
	return plan
}

var ErrPullIgnoredApprovalRequired = errors.New("ignored files may be overwritten by this integration; review and explicitly approve them first")

var ErrPullIntegrationChoiceRequired = errors.New("no pull reconciliation policy is configured; choose fast-forward only, merge, or rebase explicitly")

// PrepareCurrentPullReview must precede the fetch. trackingRef is the explicit
// local upstream/tracking ref whose reflog may contain the previous upstream.
// An empty value means there is no known tracking history, not a guessed ref.
func (self *SyncCommands) PrepareCurrentPullReview(name, expectedOID, trackingRef string) (CurrentPullReview, error) {
	if err := self.validateCurrentPullSource(name, expectedOID); err != nil {
		return CurrentPullReview{}, err
	}
	signature, err := self.currentPullPolicySignature(name)
	if err != nil {
		return CurrentPullReview{}, err
	}
	review := CurrentPullReview{owner: self.GitCommon, name: name, sourceOID: strings.ToLower(expectedOID), policySignature: signature}
	if trackingRef != "" {
		if !strings.HasPrefix(trackingRef, "refs/") {
			return CurrentPullReview{}, fmt.Errorf("choose a full tracking ref for fork-point inspection")
		}
		if err := self.cmd.New(NewGitCmd("check-ref-format").Arg(trackingRef).ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
			return CurrentPullReview{}, err
		}
		state, err := NewBranchCommands(self.GitCommon).readPlacementRef(trackingRef)
		if err != nil {
			return CurrentPullReview{}, err
		}
		if state.symbolic != "" {
			return CurrentPullReview{}, fmt.Errorf("tracking ref is a symbolic alias; choose a direct ref")
		}
		if state.exists {
			if err := NewBranchCommands(self.GitCommon).validateLiteralCommitOID(state.oid); err != nil {
				return CurrentPullReview{}, err
			}
			point, err := self.pullReviewCommitQuery("--fork-point", trackingRef, review.sourceOID+"^{commit}")
			if err != nil {
				return CurrentPullReview{}, err
			}
			review.forkPoint = point
		}
	}
	review.prepared = true
	return review, nil
}

// PrepareLocalCurrentPullReview reads a local upstream directly, without using
// a transport named "." or allowing URL rewriting to redirect the operation.
func (self *SyncCommands) PrepareLocalCurrentPullReview(name, expectedOID, upstreamRef string) (CurrentPullReview, string, error) {
	review, err := self.PrepareCurrentPullReview(name, expectedOID, upstreamRef)
	if err != nil {
		return CurrentPullReview{}, "", err
	}
	targetOID, err := self.inspectLocalBranchSyncRef(upstreamRef)
	if err != nil {
		return CurrentPullReview{}, "", err
	}
	return review, targetOID, nil
}

func (self *SyncCommands) validateCurrentPullSource(name, expectedOID string) error {
	if err := validateObjectID(expectedOID); err != nil {
		return err
	}
	bare, err := NewBranchCommands(self.GitCommon).namedCheckoutContextIsBare()
	if err != nil {
		return err
	}
	if bare {
		return fmt.Errorf("cannot integrate a fetched commit in a bare repository")
	}
	info, err := self.InspectBranchSync(name)
	if err != nil {
		return err
	}
	if info.HeadRef != info.FullRef || info.OID != strings.ToLower(expectedOID) {
		return ErrBranchSyncChanged
	}
	if err := self.rejectWorktreeOperation("integrate a fetched commit"); err != nil {
		return err
	}
	if info.Reservation.Kind != "" {
		return fmt.Errorf("branch %q is reserved by a worktree or operation", name)
	}
	return nil
}

func (self *SyncCommands) PrepareCurrentPullIntegration(review CurrentPullReview, targetOID string, requested PullIntegrationMethod) (CurrentPullIntegrationPlan, error) {
	if !review.prepared || review.owner != self.GitCommon {
		return CurrentPullIntegrationPlan{}, fmt.Errorf("pull review belongs to a different repository instance or was not prepared")
	}
	if err := self.validateCurrentPullSource(review.name, review.sourceOID); err != nil {
		return CurrentPullIntegrationPlan{}, err
	}
	signature, err := self.currentPullPolicySignature(review.name)
	if err != nil {
		return CurrentPullIntegrationPlan{}, err
	}
	if signature != review.policySignature {
		return CurrentPullIntegrationPlan{}, ErrBranchSyncChanged
	}
	if err := NewBranchCommands(self.GitCommon).validateLiteralCommitOID(targetOID); err != nil {
		return CurrentPullIntegrationPlan{}, err
	}
	targetOID = strings.ToLower(targetOID)
	method, clean, err := self.resolvePullReviewMethod(review.name, review.sourceOID, targetOID, requested)
	if err != nil {
		return CurrentPullIntegrationPlan{}, err
	}
	if clean {
		// Review is read-only, including Git's optional stat-cache/index writes.
		if err := requireCleanRebaseStatus(self.cmd.New(cleanRebaseStatusCommand().ToArgv()).
			SetWd(self.repoPaths.WorktreePath()).AddEnvVars("GIT_OPTIONAL_LOCKS=0")); err != nil {
			return CurrentPullIntegrationPlan{}, err
		}
		canFF, err := self.branchSyncIsCommitAncestor(review.sourceOID, targetOID)
		if err != nil {
			return CurrentPullIntegrationPlan{}, err
		}
		if canFF {
			method = PullIntegrationFastForwardOnly // Native pull does not rebase a fast-forward.
		}
	}
	plan := CurrentPullIntegrationPlan{review: review, targetOID: targetOID, requested: requested, method: method, replayBaseOID: targetOID}
	if requested == PullIntegrationDefault && !isPullReviewRebase(method) {
		entries, err := self.branchSyncConfigEntries()
		if err != nil {
			return CurrentPullIntegrationPlan{}, err
		}
		for _, entry := range entries {
			if entry.key == "branch."+review.name+".mergeoptions" {
				plan.mergeOptions = entry.value
			}
		}
		if plan.mergeOptions != "" {
			plan.mergeFF, _, err = self.pullReviewConfig("pull.ff")
			if err != nil {
				return CurrentPullIntegrationPlan{}, err
			}
			if clean && method == PullIntegrationFastForwardOnly {
				plan.mergeFF = "only"
			}
			plan.method = PullIntegrationConfiguredMerge
		}
	}
	if isPullReviewRebase(method) && review.forkPoint != "" {
		base, err := self.pullReviewCommitQuery("--octopus", review.sourceOID+"^{commit}", targetOID+"^{commit}", review.forkPoint+"^{commit}")
		if err != nil {
			return CurrentPullIntegrationPlan{}, err
		}
		if base != review.forkPoint {
			plan.replayBaseOID = review.forkPoint
		}
	}
	if !isPullReviewRebase(plan.method) {
		alreadyAhead, err := self.branchSyncIsCommitAncestor(targetOID, review.sourceOID)
		if err != nil {
			return CurrentPullIntegrationPlan{}, err
		}
		if alreadyAhead {
			plan.prepared = true // Native merge is a no-op, even with --no-ff.
			return plan, nil
		}
		if plan.method == PullIntegrationFastForwardOnly || (plan.method == PullIntegrationConfiguredMerge && plan.mergeFF == "only") {
			canFF, err := self.branchSyncIsCommitAncestor(review.sourceOID, targetOID)
			if err != nil {
				return CurrentPullIntegrationPlan{}, err
			}
			if !canFF {
				return CurrentPullIntegrationPlan{}, fmt.Errorf("%w: choose merge or rebase explicitly; fast-forward-only will not integrate divergent history", ErrBranchSyncDiverged)
			}
		}
	}
	// The same conservative target/replay path scan is useful for merge and
	// fast-forward as well as rebase; share fingerprints, not an approval flag.
	rebase := &RebaseCommands{GitCommon: self.GitCommon}
	plan.ignoredPaths, plan.ignoredFingerprint, err = rebase.selectedCommitRebaseIgnoredPaths(targetOID, review.sourceOID)
	if err != nil {
		return CurrentPullIntegrationPlan{}, err
	}
	plan.prepared = true
	return plan, nil
}

func (self *SyncCommands) CurrentPullIntegrationCommand(plan CurrentPullIntegrationPlan) (*oscommands.CmdObj, error) {
	if !plan.prepared {
		return nil, fmt.Errorf("pull integration plan was not prepared")
	}
	fresh, err := self.PrepareCurrentPullIntegration(plan.review, plan.targetOID, plan.requested)
	if err != nil {
		return nil, err
	}
	if fresh.method != plan.method || fresh.replayBaseOID != plan.replayBaseOID || fresh.ignoredFingerprint != plan.ignoredFingerprint || !slices.Equal(fresh.ignoredPaths, plan.ignoredPaths) {
		return nil, ErrBranchSyncChanged
	}
	if len(plan.ignoredPaths) > 0 && !plan.ignoredApproved {
		return nil, ErrPullIgnoredApprovalRequired
	}
	var args *GitCommandBuilder
	if isPullReviewRebase(plan.method) {
		args = NewGitCmd("rebase").Config("rebase.updateRefs=false").Arg("--no-autostash", "--no-fork-point").
			ArgIf(plan.requested != PullIntegrationDefault, "--no-autosquash")
		switch plan.method {
		case PullIntegrationRebaseMerges:
			args.Arg("--rebase-merges")
		case PullIntegrationRebaseInteractive:
			args.Arg("--interactive")
		default:
			args.ArgIf(plan.requested != PullIntegrationDefault, "--no-rebase-merges")
		}
		args.Arg("--onto", plan.targetOID+"^{commit}", plan.replayBaseOID+"^{commit}")
	} else {
		args = NewGitCmd("merge").Config("merge.autostash=false").Arg("--no-edit", "--no-autostash")
		if plan.requested != PullIntegrationDefault {
			// An explicit choice replaces the branch-specific merge option bundle;
			// defaults retain that bundle for the UI to display and approve.
			args.Config("branch." + plan.Name() + ".mergeOptions=")
		}
		switch plan.method {
		case PullIntegrationConfiguredMerge:
			args.ArgIf(plan.mergeFF == "only", "--ff-only").ArgIf(plan.mergeFF == "true", "--ff").ArgIf(plan.mergeFF == "false", "--no-ff")
		case PullIntegrationFastForwardOnly:
			args.Arg("--ff-only")
		case PullIntegrationMergeNoFF:
			args.Arg("--no-ff")
		default:
			args.Arg("--ff")
		}
		args.ArgIf(plan.ignoredApproved, "--overwrite-ignore").ArgIf(!plan.ignoredApproved, "--no-overwrite-ignore").Arg(plan.targetOID + "^{commit}")
	}
	return self.cmd.New(args.Config("core.warnAmbiguousRefs=false").Config("submodule.recurse=false").ToArgv()).SetWd(self.repoPaths.WorktreePath()).DisableLockRetries(), nil
}

func isPullReviewRebase(method PullIntegrationMethod) bool {
	return method == PullIntegrationRebase || method == PullIntegrationRebaseMerges || method == PullIntegrationRebaseInteractive
}

func (self *SyncCommands) pullReviewConfig(key string) (value string, found bool, err error) {
	stdout, _, err := self.cmd.New(NewGitCmd("config").Arg("--type=bool-or-str", "--get", key).ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		if commandExitedWithOne(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return trimTerminalLineEnding(stdout), true, nil
}

func (self *SyncCommands) resolvePullReviewMethod(name, sourceOID, targetOID string, requested PullIntegrationMethod) (PullIntegrationMethod, bool, error) {
	if requested != PullIntegrationDefault {
		if requested < PullIntegrationFastForwardOnly || requested > PullIntegrationRebaseInteractive {
			return 0, false, fmt.Errorf("invalid pull integration method")
		}
		return requested, isPullReviewRebase(requested), nil
	}
	rebaseValue, rebaseFound, err := self.pullReviewConfig("branch." + name + ".rebase")
	if err != nil {
		return 0, false, err
	}
	if !rebaseFound {
		rebaseValue, rebaseFound, err = self.pullReviewConfig("pull.rebase")
		if err != nil {
			return 0, false, err
		}
	}
	rebaseMethod := PullIntegrationMerge
	switch rebaseValue {
	case "", "false":
	case "true":
		rebaseMethod = PullIntegrationRebase
	case "merges", "m":
		rebaseMethod = PullIntegrationRebaseMerges
	case "interactive", "i":
		rebaseMethod = PullIntegrationRebaseInteractive
	default:
		return 0, false, fmt.Errorf("unsupported configured pull rebase mode %q; choose a method explicitly", rebaseValue)
	}
	ff, ffFound, err := self.pullReviewConfig("pull.ff")
	if err != nil {
		return 0, false, err
	}
	if ffFound && ff != "true" && ff != "false" && ff != "only" {
		return 0, false, fmt.Errorf("invalid configured pull.ff value %q", ff)
	}
	clean := isPullReviewRebase(rebaseMethod)
	if ff == "only" {
		return PullIntegrationFastForwardOnly, clean, nil
	}
	if clean {
		return rebaseMethod, true, nil
	}
	if !ffFound && !rebaseFound {
		forward, err := self.branchSyncIsCommitAncestor(sourceOID, targetOID)
		if err != nil {
			return 0, false, err
		}
		if !forward {
			ahead, err := self.branchSyncIsCommitAncestor(targetOID, sourceOID)
			if err != nil {
				return 0, false, err
			}
			if !ahead {
				return 0, false, ErrPullIntegrationChoiceRequired
			}
		}
	}
	if !ffFound {
		ff, _, err = self.pullReviewConfig("merge.ff")
		if err != nil {
			return 0, false, err
		}
		// Git ignores unrecognized merge.ff values for forward compatibility.
	}
	switch ff {
	case "false":
		return PullIntegrationMergeNoFF, false, nil
	case "only":
		return PullIntegrationFastForwardOnly, false, nil
	default:
		return PullIntegrationMerge, false, nil
	}
}

// pullReviewCommitQuery distinguishes a legitimate missing merge/fork point
// from native query failures. Revision inputs are validated OIDs or exact refs.
func (self *SyncCommands) pullReviewCommitQuery(args ...string) (string, error) {
	stdout, _, err := self.cmd.New(NewGitCmd("merge-base").Config("core.warnAmbiguousRefs=false").Arg(args...).ToArgv()).SetWd(self.repoPaths.WorktreePath()).DontLog().RunWithOutputs()
	if err != nil {
		if commandExitedWithOne(err) && stdout == "" {
			return "", nil
		}
		return "", err
	}
	oid := trimTerminalLineEnding(stdout)
	if err := NewBranchCommands(self.GitCommon).validateLiteralCommitOID(oid); err != nil {
		return "", fmt.Errorf("invalid merge-base result: %w", err)
	}
	return strings.ToLower(oid), nil
}
