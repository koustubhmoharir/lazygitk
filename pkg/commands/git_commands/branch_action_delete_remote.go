package git_commands

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// BranchActionRemoteDeleteSelection names an explicitly chosen server branch.
// Target is the selected local/cached ref used in the review, not a claim about
// the server's current tip. RemoteRef must be a full refs/heads/ name.
type BranchActionRemoteDeleteSelection struct {
	Target    BranchActionTarget
	Remote    string
	RemoteRef string
}

type branchActionRemoteDeleteStep struct {
	remote, signature string
	refs              []string
}

type BranchActionRemoteDeletePlan struct {
	owner            *GitCommon
	selections       []BranchActionRemoteDeleteSelection
	steps            []branchActionRemoteDeleteStep
	trackingRefs     []string
	configSignature  string
	headRef, headOID string
	approved         bool
	prepared         bool
}

var ErrBranchRemoteDeleteApprovalRequired = errors.New("remote branch deletion requires explicit approval")

func (plan BranchActionRemoteDeletePlan) Selections() []BranchActionRemoteDeleteSelection {
	return slices.Clone(plan.selections)
}

// TrackingRefs are potential native cache deletions. Compound local deletion
// must not count them as proof that history will remain reachable afterwards.
func (plan BranchActionRemoteDeletePlan) TrackingRefs() []string {
	return slices.Clone(plan.trackingRefs)
}

func (plan BranchActionRemoteDeletePlan) WithDeletionApproved() BranchActionRemoteDeletePlan {
	plan.approved = true
	return plan
}

func (self *SyncCommands) PrepareBranchActionRemoteDelete(selections []BranchActionRemoteDeleteSelection) (BranchActionRemoteDeletePlan, error) {
	if len(selections) == 0 {
		return BranchActionRemoteDeletePlan{}, fmt.Errorf("choose at least one remote branch")
	}
	branches := NewBranchCommands(self.GitCommon)
	plan := BranchActionRemoteDeletePlan{
		owner: self.GitCommon, selections: slices.Clone(selections),
		headRef: selections[0].Target.HeadRef(), headOID: selections[0].Target.HeadOID(),
	}
	stepByRemote := make(map[string]int)
	trackingSeen := make(map[string]bool)
	for _, selection := range selections {
		if err := branches.CheckBranchActionTarget(selection.Target); err != nil {
			return BranchActionRemoteDeletePlan{}, err
		}
		if selection.Target.HeadRef() != plan.headRef || selection.Target.HeadOID() != plan.headOID {
			return BranchActionRemoteDeletePlan{}, ErrBranchActionChanged
		}
		if !strings.HasPrefix(selection.RemoteRef, "refs/heads/") {
			return BranchActionRemoteDeletePlan{}, fmt.Errorf("choose a full remote branch ref under refs/heads/")
		}
		if err := self.checkBranchSyncRef(selection.RemoteRef, false); err != nil {
			return BranchActionRemoteDeletePlan{}, err
		}
		remote, err := self.InspectBranchSyncRemote(selection.Remote)
		if err != nil {
			return BranchActionRemoteDeletePlan{}, err
		}
		mapping, err := self.branchSyncTrackingRefs(remote.FetchRefspecs, selection.RemoteRef)
		if err != nil {
			return BranchActionRemoteDeletePlan{}, err
		}
		for _, ref := range mapping.refs {
			// Native push updates its configured local caches as well as the
			// server. Never let that implicitly delete a local branch, tag or
			// symbolic alias, including one reserved by another worktree.
			if err := self.checkBranchSyncTrackingWrite(ref); err != nil {
				return BranchActionRemoteDeletePlan{}, err
			}
			if !trackingSeen[ref] {
				trackingSeen[ref] = true
				plan.trackingRefs = append(plan.trackingRefs, ref)
			}
		}
		index, exists := stepByRemote[remote.Name]
		if !exists {
			index = len(plan.steps)
			stepByRemote[remote.Name] = index
			plan.steps = append(plan.steps, branchActionRemoteDeleteStep{remote: remote.Name, signature: remote.Signature})
		} else if plan.steps[index].signature != remote.Signature {
			return BranchActionRemoteDeletePlan{}, ErrBranchActionChanged
		}
		if !slices.Contains(plan.steps[index].refs, selection.RemoteRef) {
			plan.steps[index].refs = append(plan.steps[index].refs, selection.RemoteRef)
		}
	}
	slices.Sort(plan.trackingRefs)
	var err error
	plan.configSignature, err = branches.deleteConfigSignature()
	if err != nil {
		return BranchActionRemoteDeletePlan{}, err
	}
	for _, selection := range selections {
		if err := branches.CheckBranchActionTarget(selection.Target); err != nil {
			return BranchActionRemoteDeletePlan{}, err
		}
	}
	plan.prepared = true
	return plan, nil
}

// RunBranchActionRemoteDelete runs one native push per named remote, retaining
// configured push URLs and their native partial-success behavior. The caller
// supplies its captured UI repository/generation guard. Stop on every error;
// earlier successful server/cache deletions are neither retried nor rolled back.
//
// Source snapshots are checked before the batch starts. Native pushes may then
// intentionally delete selected cached source refs. Between steps, check HEAD,
// effective configuration and cache write safety without demanding those
// already-deleted sources reappear. Every server destination stays explicit.
func (self *SyncCommands) RunBranchActionRemoteDelete(plan BranchActionRemoteDeletePlan, task gocui.Task, checkRepository func() error) error {
	if !plan.prepared || plan.owner != self.GitCommon {
		return fmt.Errorf("remote delete plan was not prepared by this repository instance")
	}
	if !plan.approved {
		return ErrBranchRemoteDeleteApprovalRequired
	}
	guard := func() error {
		if checkRepository != nil {
			return checkRepository()
		}
		return nil
	}
	if err := guard(); err != nil {
		return err
	}
	fresh, err := self.PrepareBranchActionRemoteDelete(plan.selections)
	if err != nil {
		return err
	}
	if fresh.configSignature != plan.configSignature || !slices.Equal(fresh.trackingRefs, plan.trackingRefs) ||
		!slices.EqualFunc(fresh.steps, plan.steps, func(a, b branchActionRemoteDeleteStep) bool {
			return a.remote == b.remote && a.signature == b.signature && slices.Equal(a.refs, b.refs)
		}) {
		return ErrBranchActionChanged
	}
	branches := NewBranchCommands(self.GitCommon)
	for _, step := range plan.steps {
		if err := guard(); err != nil {
			return err
		}
		ref, oid, err := branches.namedCheckoutHead()
		if err != nil {
			return err
		}
		if ref != plan.headRef || oid != plan.headOID {
			return ErrBranchActionChanged
		}
		signature, err := branches.deleteConfigSignature()
		if err != nil {
			return err
		}
		if signature != plan.configSignature {
			return ErrBranchActionChanged
		}
		for _, ref := range plan.trackingRefs {
			if err := self.checkBranchSyncTrackingWrite(ref); err != nil {
				return err
			}
		}
		args := NewGitCmd("push").configWithLiteralKey("remote."+step.remote+".mirror", "false", literalMirrorEnv).Config("push.autoSetupRemote=false").
			Arg("--no-follow-tags", "--recurse-submodules=no", "--delete", "--", step.remote).Arg(step.refs...)
		cmd := self.cmd.New(args.ToArgv()).SetWd(self.repoPaths.WorktreePath()).AddEnvVars(literalMirrorEnv + "=false").
			PromptOnCredentialRequest(task).DisableLockRetries()
		if err := guard(); err != nil {
			return err
		}
		result := cmd.Run()
		if err := guard(); err != nil {
			return errors.Join(result, err)
		}
		if result != nil {
			return result
		}
	}
	return nil
}
