package git_commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// CurrentBranchPullPlan captures the current branch and native pull policy.
// It deliberately models a native pull, separate from off-branch fetch/update.
type CurrentBranchPullPlan struct {
	owner           *GitCommon
	name, sourceOID string
	options         BranchFetchOptions
	remoteSignature string
	policySignature string
	rebase          bool
	prepared        bool
}

func (plan CurrentBranchPullPlan) Name() string                { return plan.name }
func (plan CurrentBranchPullPlan) SourceOID() string           { return plan.sourceOID }
func (plan CurrentBranchPullPlan) Options() BranchFetchOptions { return plan.options }
func (plan CurrentBranchPullPlan) UsesRebase() bool            { return plan.rebase }

func (self *SyncCommands) PrepareCurrentBranchPull(name, expectedOID string, opts BranchFetchOptions) (CurrentBranchPullPlan, error) {
	if err := validateObjectID(expectedOID); err != nil {
		return CurrentBranchPullPlan{}, err
	}
	info, err := self.InspectBranchSync(name)
	if err != nil {
		return CurrentBranchPullPlan{}, err
	}
	expectedOID = strings.ToLower(expectedOID)
	if info.OID != expectedOID || info.HeadRef != info.FullRef {
		return CurrentBranchPullPlan{}, ErrBranchSyncChanged
	}
	if !strings.HasPrefix(opts.RemoteRef, "refs/heads/") || opts.RemoteRef == "refs/heads/" {
		return CurrentBranchPullPlan{}, fmt.Errorf("choose a full remote branch ref under refs/heads/")
	}
	if err := self.cmd.New(NewGitCmd("check-ref-format").Arg(opts.RemoteRef).ToArgv()).
		SetWd(self.repoPaths.WorktreePath()).DontLog().Run(); err != nil {
		return CurrentBranchPullPlan{}, fmt.Errorf("invalid remote branch ref: %w", err)
	}
	if err := self.checkBranchSyncTrackingWrite(opts.TrackingRef); err != nil {
		return CurrentBranchPullPlan{}, err
	}
	remote, err := self.InspectBranchSyncRemote(opts.Remote)
	if err != nil {
		return CurrentBranchPullPlan{}, err
	}
	if len(remote.URLs) == 0 {
		return CurrentBranchPullPlan{}, fmt.Errorf("remote %q has no configured fetch URL; a push-only URL is not safe for fetching", opts.Remote)
	}
	policy, err := self.effectivePullRebasePolicy(PullOptions{})
	if err != nil {
		return CurrentBranchPullPlan{}, err
	}
	if policy.rebase {
		if err := self.requireCleanRebaseStart(); err != nil {
			return CurrentBranchPullPlan{}, err
		}
	}
	policySignature, err := self.currentPullPolicySignature(name)
	if err != nil {
		return CurrentBranchPullPlan{}, err
	}
	return CurrentBranchPullPlan{
		owner: self.GitCommon, name: name, sourceOID: expectedOID, options: opts,
		remoteSignature: remote.Signature, policySignature: policySignature,
		rebase: policy.rebase, prepared: true,
	}, nil
}

func (self *SyncCommands) currentPullPolicySignature(name string) (string, error) {
	entries, err := self.branchSyncConfigEntries()
	if err != nil {
		return "", err
	}
	digest := sha256.New()
	branchPrefix := "branch." + name + "."
	for _, entry := range entries {
		key := entry.key
		if strings.HasPrefix(key, "pull.") || strings.HasPrefix(key, "rebase.") ||
			strings.HasPrefix(key, "merge.") || key == branchPrefix+"rebase" || key == branchPrefix+"mergeoptions" {
			writeBranchSyncConfigEntry(digest, entry)
		}
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func (self *SyncCommands) CurrentBranchPullCommand(plan CurrentBranchPullPlan, task gocui.Task) (*oscommands.CmdObj, error) {
	if !plan.prepared || plan.owner != self.GitCommon {
		return nil, fmt.Errorf("pull plan was not prepared by this repository instance")
	}
	fresh, err := self.PrepareCurrentBranchPull(plan.name, plan.sourceOID, plan.options)
	if err != nil {
		return nil, err
	}
	if fresh != plan {
		return nil, ErrBranchSyncChanged
	}
	// `pull` does not accept fetch's --no-prune-tags option. Override both
	// generic and selected-remote settings for this invocation instead.
	args := NewGitCmd("pull").Config("rebase.updateRefs=false").Config("merge.autostash=false").
		Config("fetch.pruneTags=false").Config("remote."+plan.options.Remote+".pruneTags=false").
		Arg("--no-edit", "--no-autostash", "--refmap=", "--no-tags", "--no-prune", "--no-recurse-submodules", "--", plan.options.Remote,
			"+"+plan.options.RemoteRef+":"+plan.options.TrackingRef).ToArgv()
	return self.cmd.New(args).SetWd(self.repoPaths.WorktreePath()).
		AddEnvVars("GIT_SEQUENCE_EDITOR=:").DisableLockRetries().PromptOnCredentialRequest(task), nil
}
