package git_commands

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// SelectedCommitRebasePlan captures the target and the checked-out branch
// identity. Its fields are private so only preparation can authorize it.
type SelectedCommitRebasePlan struct {
	targetOID, headRef, headOID string
	ignoredPaths                []string
	ignoredFingerprint          string
	ignoredApproved             bool
	preserveMerges              bool
	prepared                    bool
}

func (plan SelectedCommitRebasePlan) TargetOID() string    { return plan.targetOID }
func (plan SelectedCommitRebasePlan) HeadRef() string      { return plan.headRef }
func (plan SelectedCommitRebasePlan) HeadOID() string      { return plan.headOID }
func (plan SelectedCommitRebasePlan) PreserveMerges() bool { return plan.preserveMerges }

// IgnoredFiles returns a copy of ignored files which may collide with this rebase.
func (plan SelectedCommitRebasePlan) IgnoredFiles() []string {
	return slices.Clone(plan.ignoredPaths)
}

// WithIgnoredFilesApproved returns a copy recording explicit approval for the
// ignored-file risks found during preparation.
func (plan SelectedCommitRebasePlan) WithIgnoredFilesApproved() SelectedCommitRebasePlan {
	plan.ignoredApproved = true
	return plan
}

var (
	ErrSelectedCommitRebaseChanged   = errors.New("the rebase source changed; review it again before proceeding")
	ErrRebaseIgnoredApprovalRequired = errors.New("ignored files may be overwritten by this rebase; review and explicitly approve them first")
)

// PrepareSelectedCommitRebase captures an exact commit and current branch
// snapshot without changing repository state.
func (self *RebaseCommands) PrepareSelectedCommitRebase(targetOID string, preserveMerges bool) (SelectedCommitRebasePlan, error) {
	branches := NewBranchCommands(self.GitCommon)
	if err := branches.validateLiteralCommitOID(targetOID); err != nil {
		return SelectedCommitRebasePlan{}, err
	}
	bare, err := branches.namedCheckoutContextIsBare()
	if err != nil {
		return SelectedCommitRebasePlan{}, err
	}
	if bare {
		return SelectedCommitRebasePlan{}, fmt.Errorf("cannot rebase in a bare repository")
	}
	if err := self.rejectWorktreeOperation("start a rebase"); err != nil {
		return SelectedCommitRebasePlan{}, err
	}
	if err := self.requireCleanRebaseStart(); err != nil {
		return SelectedCommitRebasePlan{}, err
	}
	ref, oid, err := branches.namedCheckoutHead()
	if err != nil {
		return SelectedCommitRebasePlan{}, err
	}
	if ref == "" {
		return SelectedCommitRebasePlan{}, fmt.Errorf("cannot rebase with a detached HEAD")
	}
	if oid == "" {
		return SelectedCommitRebasePlan{}, fmt.Errorf("cannot rebase an unborn branch")
	}
	state, err := branches.readPlacementRef(ref)
	if err != nil {
		return SelectedCommitRebasePlan{}, err
	}
	if state.symbolic != "" {
		return SelectedCommitRebasePlan{}, fmt.Errorf("branch %q is a symbolic alias; choose a direct local branch", strings.TrimPrefix(ref, "refs/heads/"))
	}
	if !state.exists || state.oid != oid {
		return SelectedCommitRebasePlan{}, fmt.Errorf("current branch ref does not match HEAD")
	}
	if err := branches.validateLiteralCommitOID(oid); err != nil {
		return SelectedCommitRebasePlan{}, fmt.Errorf("current HEAD is not an existing commit: %w", err)
	}
	occupation, err := branches.branchOccupancy(ref, true)
	if err != nil {
		return SelectedCommitRebasePlan{}, err
	}
	if occupation != nil {
		if occupation.Current {
			return SelectedCommitRebasePlan{}, fmt.Errorf("current branch has an active %s reservation in this worktree", occupation.Kind)
		}
		return SelectedCommitRebasePlan{}, fmt.Errorf("current branch is also reserved by another worktree (%s, Git directory %q)", occupation.Kind, occupation.GitDir)
	}
	ignoredPaths, fingerprint, err := self.selectedCommitRebaseIgnoredPaths(strings.ToLower(targetOID), oid)
	if err != nil {
		return SelectedCommitRebasePlan{}, err
	}
	return SelectedCommitRebasePlan{
		targetOID: strings.ToLower(targetOID), headRef: ref, headOID: oid,
		ignoredPaths: ignoredPaths, ignoredFingerprint: fingerprint, preserveMerges: preserveMerges, prepared: true,
	}, nil
}

// SelectedCommitRebaseCommand revalidates a prepared snapshot and builds the
// explicit, non-interactive Git command. The caller controls when it runs.
func (self *RebaseCommands) SelectedCommitRebaseCommand(plan SelectedCommitRebasePlan) (*oscommands.CmdObj, error) {
	if !plan.prepared {
		return nil, fmt.Errorf("selected-commit rebase plan was not prepared")
	}
	fresh, err := self.PrepareSelectedCommitRebase(plan.targetOID, plan.preserveMerges)
	if err != nil {
		return nil, err
	}
	if fresh.targetOID != plan.targetOID || fresh.headRef != plan.headRef || fresh.headOID != plan.headOID ||
		fresh.preserveMerges != plan.preserveMerges || !slices.Equal(fresh.ignoredPaths, plan.ignoredPaths) ||
		fresh.ignoredFingerprint != plan.ignoredFingerprint {
		return nil, ErrSelectedCommitRebaseChanged
	}
	if len(fresh.ignoredPaths) > 0 && !plan.ignoredApproved {
		return nil, ErrRebaseIgnoredApprovalRequired
	}
	args := NewGitCmd("rebase").Config("rebase.updateRefs=false").Arg("--no-autostash", "--no-fork-point", "--no-autosquash")
	if plan.preserveMerges {
		args.Arg("--rebase-merges=no-rebase-cousins")
	} else {
		args.Arg("--no-rebase-merges")
	}
	args.Arg(plan.targetOID + "^{commit}")
	return self.cmd.New(args.ToArgv()).DisableLockRetries().SetWd(self.repoPaths.WorktreePath()), nil
}
