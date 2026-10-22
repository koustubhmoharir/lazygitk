package git_commands

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// BranchFetchOptions makes both the remote branch and its local storage ref
// explicit. TrackingRef may use any safe namespace; configured fetch mappings
// do not select or override it.
type BranchFetchOptions struct {
	Remote, RemoteRef, TrackingRef string
}

// BranchFetchPlan is bound to the repository, selected branch, remote
// configuration, and reserved local ref captured during preparation.
type BranchFetchPlan struct {
	owner           *GitCommon
	name, sourceOID string
	options         BranchFetchOptions
	remoteSignature string
	reservation     BranchSyncReservation
	prepared        bool
}

func (plan BranchFetchPlan) Name() string                { return plan.name }
func (plan BranchFetchPlan) SourceOID() string           { return plan.sourceOID }
func (plan BranchFetchPlan) Options() BranchFetchOptions { return plan.options }

func (self *SyncCommands) PrepareBranchFetch(name, expectedOID string, opts BranchFetchOptions) (BranchFetchPlan, error) {
	if err := validateObjectID(expectedOID); err != nil {
		return BranchFetchPlan{}, err
	}
	info, err := self.InspectBranchSync(name)
	if err != nil {
		return BranchFetchPlan{}, err
	}
	expectedOID = strings.ToLower(expectedOID)
	if info.OID != expectedOID {
		return BranchFetchPlan{}, ErrBranchSyncChanged
	}
	if info.HeadRef == info.FullRef {
		return BranchFetchPlan{}, fmt.Errorf("branch %q is checked out in this worktree; branch fetch update requires an off-branch target", name)
	}
	if !strings.HasPrefix(opts.RemoteRef, "refs/heads/") || opts.RemoteRef == "refs/heads/" {
		return BranchFetchPlan{}, fmt.Errorf("choose a full remote branch ref under refs/heads/")
	}
	if err := self.cmd.New(NewGitCmd("check-ref-format").Arg(opts.RemoteRef).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
		return BranchFetchPlan{}, fmt.Errorf("invalid remote branch ref: %w", err)
	}
	if err := self.checkBranchSyncTrackingWrite(opts.TrackingRef); err != nil {
		return BranchFetchPlan{}, err
	}
	remote, err := self.InspectBranchSyncRemote(opts.Remote)
	if err != nil {
		return BranchFetchPlan{}, err
	}
	if len(remote.URLs) == 0 {
		return BranchFetchPlan{}, fmt.Errorf("remote %q has no configured fetch URL; a push-only URL is not safe for fetching", opts.Remote)
	}
	return BranchFetchPlan{
		owner: self.GitCommon, name: name, sourceOID: expectedOID, options: opts,
		remoteSignature: remote.Signature, reservation: info.Reservation, prepared: true,
	}, nil
}

// BranchFetchCommand revalidates the captured plan, then fetches exactly the
// explicitly selected source into the explicitly selected local ref. Empty
// --refmap disables configured mappings; explicit tag and pruning options
// prevent implicit tag ref updates even when remote-specific config requests them.
func (self *SyncCommands) BranchFetchCommand(plan BranchFetchPlan, task gocui.Task) (*oscommands.CmdObj, error) {
	if !plan.prepared || plan.owner != self.GitCommon {
		return nil, fmt.Errorf("fetch plan was not prepared by this repository instance")
	}
	fresh, err := self.PrepareBranchFetch(plan.name, plan.sourceOID, plan.options)
	if err != nil {
		return nil, err
	}
	if fresh != plan {
		return nil, ErrBranchSyncChanged
	}
	args := NewGitCmd("fetch").
		Arg("--refmap=", "--no-tags", "--no-prune", "--no-prune-tags", "--no-recurse-submodules", "--no-write-fetch-head", "--", plan.options.Remote,
			"+"+plan.options.RemoteRef+":"+plan.options.TrackingRef).
		ToArgv()
	return self.cmd.New(args).SetWd(self.repoPaths.WorktreePath()).
		DisableLockRetries().PromptOnCredentialRequest(task), nil
}

// PrepareFetchedBranchUpdate must be called only after BranchFetchCommand has
// succeeded. It reads the exact fetched ref and creates the separate guarded,
// ref-only fast-forward plan.
func (self *SyncCommands) PrepareFetchedBranchUpdate(plan BranchFetchPlan) (BranchFastForwardPlan, error) {
	if !plan.prepared || plan.owner != self.GitCommon {
		return BranchFastForwardPlan{}, fmt.Errorf("fetch plan was not prepared by this repository instance")
	}
	fresh, err := self.PrepareBranchFetch(plan.name, plan.sourceOID, plan.options)
	if err != nil {
		return BranchFastForwardPlan{}, err
	}
	if fresh != plan {
		return BranchFastForwardPlan{}, ErrBranchSyncChanged
	}
	state, err := NewBranchCommands(self.GitCommon).readPlacementRef(plan.options.TrackingRef)
	if err != nil {
		return BranchFastForwardPlan{}, err
	}
	if !state.exists || state.symbolic != "" {
		return BranchFastForwardPlan{}, fmt.Errorf("fetched tracking ref %q is missing or a symbolic alias", plan.options.TrackingRef)
	}
	if err := NewBranchCommands(self.GitCommon).validateLiteralCommitOID(state.oid); err != nil {
		return BranchFastForwardPlan{}, fmt.Errorf("fetched tracking ref %q does not point directly to a commit: %w", plan.options.TrackingRef, err)
	}
	return self.PrepareBranchFastForward(plan.name, plan.sourceOID, state.oid)
}
