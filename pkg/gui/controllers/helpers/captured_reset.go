package helpers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type branchActionResetFlow struct {
	target   *capturedResetRepository
	snapshot git_commands.BranchActionTarget
	label    string
	started  bool
}

type capturedResetRepository struct {
	c          *HelperCommon
	git        *commands.GitCommand
	generation int
}

func (target *capturedResetRepository) checkRepository() error {
	if target.c.Git() != target.git || target.c.State().GetRepoGeneration() != target.generation {
		return errors.New(target.c.Tr.BranchSyncRepositoryChanged)
	}
	return nil
}

// Capture source identity before any menu, not a pointer into a mutable model.
// The source may be a full local/upstream ref or a full detached commit OID.
func ResetCurrentToCapturedSource(c *HelperCommon, label, ref string) error {
	target := &capturedResetRepository{c: c, git: c.Git(), generation: c.State().GetRepoGeneration()}
	var upstreamBranch string
	if ref == "@{upstream}" {
		upstreamBranch = c.Model().CheckedOutBranch
		if upstreamBranch == "" {
			return errors.New(c.Tr.BranchSyncNoCurrentLocalBranch)
		}
	}
	return c.WithWaitingStatus(c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := target.checkRepository(); err != nil {
			return err
		}
		sourceRef := ref
		var upstreamHeadRef, upstreamHeadOID string
		if sourceRef == "@{upstream}" {
			info, err := target.git.Sync.InspectBranchSync(upstreamBranch)
			if err != nil {
				return err
			}
			if info.HeadRef != info.FullRef {
				return git_commands.ErrBranchActionChanged
			}
			upstreamHeadRef, upstreamHeadOID = info.HeadRef, info.HeadOID
			sourceRef = info.UpstreamTrackingRef
			if sourceRef == "" {
				return errors.New(c.Tr.UpstreamNotSetError)
			}
		}
		oid, err := target.git.Branch.ResolveBranchActionCommit(sourceRef)
		if err != nil {
			return err
		}
		snapshot, err := target.git.Branch.PrepareBranchActionTarget(sourceRef, oid)
		if err != nil {
			return err
		}
		if upstreamBranch != "" && (snapshot.HeadRef() != upstreamHeadRef || snapshot.HeadOID() != upstreamHeadOID) {
			return git_commands.ErrBranchActionChanged
		}
		c.OnUIThread(func() error {
			flow := &branchActionResetFlow{target: target, snapshot: snapshot, label: label}
			return flow.menu()
		})
		return nil
	})
}

func (flow *branchActionResetFlow) prompt() string {
	head, oid := flow.snapshot.HeadRef(), flow.snapshot.HeadOID()
	if head == "" {
		head = flow.target.c.Tr.BranchActionDetachedHead
	}
	if oid == "" {
		oid = flow.target.c.Tr.BranchActionUnbornHead
	}
	return fmt.Sprintf(flow.target.c.Tr.BranchActionResetReviewPrompt,
		head, oid, flow.label, flow.snapshot.SourceRef(), flow.snapshot.SourceOID())
}

func (flow *branchActionResetFlow) menu() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	rows := []struct{ mode, label, tooltip string }{
		{"mixed", "Mixed reset", flow.target.c.Tr.ResetMixedTooltip},
		{"soft", "Soft reset", flow.target.c.Tr.ResetSoftTooltip},
		{"hard", "Hard reset", flow.target.c.Tr.ResetHardTooltip},
	}
	items := make([]*types.MenuItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &types.MenuItem{
			CommandID: "branch.reset." + row.mode, Verb: "reset-" + row.mode,
			Label: row.label, Tooltip: row.tooltip, Keys: menuKey(rune(row.mode[0])),
			OpensMenu: row.mode == "hard",
			OnPress: func() error {
				if err := flow.target.checkRepository(); err != nil {
					return err
				}
				if row.mode != "hard" {
					return flow.execute(row.mode)
				}
				// Always review a hard reset, including ignored/untracked collision
				// loss that a cached tracked-dirty flag cannot describe reliably.
				flow.target.c.Confirm(types.ConfirmOpts{
					Title:         flow.target.c.Tr.Actions.HardReset,
					Prompt:        flow.target.c.Tr.BranchActionResetHardWarning + "\n\n" + flow.prompt(),
					HandleConfirm: func() error { return flow.execute("hard") },
					HandleClose:   flow.target.checkRepository,
				})
				return nil
			},
		})
	}
	return flow.target.c.Menu(types.CreateMenuOptions{
		Title: fmt.Sprintf("%s %s", flow.target.c.Tr.ResetTo, flow.label), Prompt: flow.prompt(),
		Items: items, OnCancel: flow.target.checkRepository,
	})
}

func (flow *branchActionResetFlow) execute(mode string) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	if flow.started {
		return git_commands.ErrBranchActionChanged
	}
	flow.started = true // A partial native result must never replay this approval.
	flow.target.c.LogAction("Reset")
	return flow.target.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: flow.target.c.Tr.ResettingStatus}, func(gocui.Task) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		cmd, err := flow.target.git.Branch.BranchActionResetCommand(flow.snapshot, mode)
		if err == nil {
			if err = flow.target.checkRepository(); err == nil {
				err = cmd.Run()
			}
		}
		if repositoryErr := flow.target.checkRepository(); repositoryErr != nil {
			return errors.Join(err, repositoryErr)
		}
		result := err
		flow.target.c.OnUIThread(func() error {
			if err := flow.target.checkRepository(); err != nil {
				return err
			}
			flow.target.c.Contexts().LocalCommits.SetLimitCommits(true)
			selection := types.KeepCommitSelectionByHash
			if result == nil {
				selection = types.SelectHeadCommit
			}
			flow.target.c.Refresh(types.RefreshOptions{
				Scope:           []types.RefreshableView{types.FILES, types.BRANCHES, types.REFLOG, types.COMMITS},
				CommitSelection: selection, SelectTopReflogCommit: result == nil,
			})
			return nil
		})
		return result
	})
}
