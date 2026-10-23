package git_commands

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/app/daemon"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

// BranchActionRebasePlan retains the selected source, current destination and
// optional marked range base. Its approval covers the exact ignored-file risk
// snapshot, not a later invocation or another repository instance.
type BranchActionRebasePlan struct {
	target             BranchActionTarget
	baseOID            string
	insertBreak        bool
	ignoredPaths       []string
	ignoredFingerprint string
	ignoredApproved    bool
}

func (plan BranchActionRebasePlan) Target() BranchActionTarget { return plan.target }
func (plan BranchActionRebasePlan) BaseOID() string            { return plan.baseOID }
func (plan BranchActionRebasePlan) InsertsBreak() bool         { return plan.insertBreak }
func (plan BranchActionRebasePlan) IgnoredFiles() []string     { return slices.Clone(plan.ignoredPaths) }

func (plan BranchActionRebasePlan) WithIgnoredFilesApproved() BranchActionRebasePlan {
	plan.ignoredApproved = true
	return plan
}

func (self *RebaseCommands) PrepareBranchActionRebase(target BranchActionTarget, baseOID string, insertBreak bool) (BranchActionRebasePlan, error) {
	branches := NewBranchCommands(self.GitCommon)
	if err := branches.CheckBranchActionTarget(target); err != nil {
		return BranchActionRebasePlan{}, err
	}
	if baseOID != "" {
		if err := branches.validateLiteralCommitOID(baseOID); err != nil {
			return BranchActionRebasePlan{}, err
		}
		baseOID = strings.ToLower(baseOID)
	}
	// Share the selected-commit start policy: native clean-tree, operation,
	// named-HEAD and reservation checks, with no automatic stash.
	safety, err := self.PrepareSelectedCommitRebase(target.SourceOID(), true)
	if err != nil {
		return BranchActionRebasePlan{}, err
	}
	if safety.HeadRef() != target.HeadRef() || safety.HeadOID() != target.HeadOID() {
		return BranchActionRebasePlan{}, ErrBranchActionChanged
	}
	paths, fingerprint := safety.ignoredPaths, safety.ignoredFingerprint
	if baseOID != "" {
		// --onto replays the marked range, which can include paths outside
		// target..HEAD. Inspect that range, not the ordinary rebase range.
		paths, fingerprint, err = self.rebaseIgnoredPaths(target.SourceOID(), baseOID, target.HeadOID())
		if err != nil {
			return BranchActionRebasePlan{}, err
		}
	}
	if err := branches.CheckBranchActionTarget(target); err != nil {
		return BranchActionRebasePlan{}, err
	}
	return BranchActionRebasePlan{
		target: target, baseOID: baseOID, insertBreak: insertBreak,
		ignoredPaths: paths, ignoredFingerprint: fingerprint,
	}, nil
}

// BranchActionRebaseCommand retains upstream's interactive todo integration,
// including its optional initial break. It runs once against immutable commits;
// the caller owns terminal signing and repository-bound refresh/error handling.
func (self *RebaseCommands) BranchActionRebaseCommand(plan BranchActionRebasePlan) (*oscommands.CmdObj, error) {
	fresh, err := self.PrepareBranchActionRebase(plan.target, plan.baseOID, plan.insertBreak)
	if err != nil {
		return nil, err
	}
	if fresh.target != plan.target || fresh.baseOID != plan.baseOID || fresh.insertBreak != plan.insertBreak ||
		!slices.Equal(fresh.ignoredPaths, plan.ignoredPaths) || fresh.ignoredFingerprint != plan.ignoredFingerprint {
		return nil, ErrBranchActionChanged
	}
	if len(fresh.ignoredPaths) > 0 && !plan.ignoredApproved {
		return nil, ErrRebaseIgnoredApprovalRequired
	}
	opts := PrepareInteractiveRebaseCommandOpts{
		baseHashOrRoot:    plan.target.SourceOID() + "^{commit}",
		disableUpdateRefs: true,
	}
	if plan.baseOID != "" {
		opts.baseHashOrRoot = plan.baseOID + "^{commit}"
		opts.onto = plan.target.SourceOID() + "^{commit}"
	}
	if plan.insertBreak {
		opts.instruction = daemon.NewInsertBreakInstruction()
	}
	cmd, err := self.PrepareInteractiveRebaseCommand(opts)
	if err != nil {
		return nil, fmt.Errorf("prepare captured branch rebase: %w", err)
	}
	return cmd.SetWd(self.repoPaths.WorktreePath()), nil
}

// CheckBranchActionRebaseInProgress identifies the rebase started from this
// review after Git has intentionally detached/moved HEAD. It must not use the
// pre-launch HEAD guard there. Interactive/rebase-merges starts use merge-backend
// metadata; another operation must not inherit a queued conflict-abort callback.
func (self *RebaseCommands) CheckBranchActionRebaseInProgress(plan BranchActionRebasePlan) error {
	if plan.target.owner == nil || plan.target.owner != self.GitCommon {
		return fmt.Errorf("rebase plan was not prepared by this repository instance")
	}
	for _, entry := range []struct{ name, expected string }{
		{"head-name", plan.target.HeadRef()},
		{"orig-head", plan.target.HeadOID()},
		{"onto", plan.target.SourceOID()},
	} {
		contents, err := os.ReadFile(filepath.Join(self.repoPaths.WorktreeGitDirPath(), "rebase-merge", entry.name))
		if os.IsNotExist(err) {
			return ErrBranchActionChanged
		}
		if err != nil {
			return err
		}
		if trimTerminalLineEnding(string(contents)) != entry.expected {
			return ErrBranchActionChanged
		}
	}
	return nil
}
