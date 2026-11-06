package helpers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: a hard reset reviews the untracked and ignored files it would overwrite
// or remove (ShowIgnoredRiskMenu). Undo and redo of a commit or rebase reach this
// same flow through capturedResetOptions (fork_undo_reset.go).

type branchActionResetFlow struct {
	target   *capturedActionRepository
	snapshot git_commands.BranchActionTarget
	label    string
	opts     capturedResetOptions
	started  bool
}

// capturedResetOptions adjusts the captured reset. The zero value is the reset menu
// and the plain reset keys.
type capturedResetOptions struct {
	// mode is "" for the reset menu, or "hard", "mixed" or "soft" to confirm directly.
	mode string
	// title and effect replace the hard reset confirmation's title and warning.
	title, effect string
	// envVars are set on the reset command, so undo and redo mark its reflog entry.
	envVars []string
	// preflight runs before the review and again before the reset runs. Undo and
	// redo use it to refuse tracked changes instead of stashing them.
	preflight func(branch *git_commands.BranchCommands) error
}

// Capture source identity before any menu, not a pointer into a mutable model.
// The source may be a full local/upstream ref or a full detached commit OID.
func ResetCurrentToCapturedSource(c *HelperCommon, label, ref string) error {
	return resetCurrentToCapturedSource(c, label, ref, capturedResetOptions{})
}

// ResetCurrentToCapturedSourceMode resets the current branch to a captured
// source using mode, after an explicit confirmation.
func ResetCurrentToCapturedSourceMode(c *HelperCommon, label, ref, mode string) error {
	switch mode {
	case "hard", "mixed", "soft":
	default:
		return fmt.Errorf("invalid reset mode %q", mode)
	}
	return resetCurrentToCapturedSource(c, label, ref, capturedResetOptions{mode: mode})
}

func resetCurrentToCapturedSource(c *HelperCommon, label, ref string, opts capturedResetOptions) error {
	target := captureActionRepository(c)
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
		if opts.preflight != nil {
			if err := opts.preflight(target.git.Branch); err != nil {
				return err
			}
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
			flow := &branchActionResetFlow{target: target, snapshot: snapshot, label: label, opts: opts}
			if opts.mode != "" {
				return flow.confirmMode(opts.mode)
			}
			return flow.menu()
		})
		return nil
	})
}

func (flow *branchActionResetFlow) confirmMode(mode string) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	if mode == "hard" {
		return flow.reviewHard(false)
	}
	var title, effect string
	switch mode {
	case "mixed":
		title, effect = flow.target.c.Tr.Actions.MixedReset, flow.target.c.Tr.ResetMixedTooltip
	case "soft":
		title, effect = flow.target.c.Tr.Actions.SoftReset, flow.target.c.Tr.ResetSoftTooltip
	default:
		return fmt.Errorf("invalid reset mode %q", mode)
	}
	flow.target.c.Confirm(types.ConfirmOpts{
		Title: title, Prompt: effect + "\n\n" + flow.prompt(),
		HandleConfirm: func() error { return flow.execute(mode, "") },
		HandleClose:   flow.target.checkRepository,
	})
	return nil
}

// reviewHard finds the untracked and ignored files that a hard reset would
// overwrite or remove, on a worker, and then opens the hard reset review. changed
// says that a reviewed approval was refused, so the fresh review follows a note.
func (flow *branchActionResetFlow) reviewHard(changed bool) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.target.c.WithWaitingStatus(flow.target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		paths, fingerprint, err := flow.target.git.Branch.HardResetIgnoredRisks(flow.snapshot)
		if err != nil {
			return err
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.target.checkRepository(); err != nil {
				return err
			}
			if changed {
				return flow.confirmChangedHardReview(paths, fingerprint)
			}
			return flow.presentHard(paths, fingerprint)
		})
		return nil
	})
}

// presentHard shows the plain hard reset confirmation when no file is at risk,
// and otherwise the review that lists the files.
func (flow *branchActionResetFlow) presentHard(paths []string, fingerprint string) error {
	if len(paths) == 0 {
		return flow.confirmHard("")
	}
	return flow.showHardRiskReview(paths, fingerprint)
}

func (flow *branchActionResetFlow) confirmHard(approved string) error {
	flow.target.c.Confirm(types.ConfirmOpts{
		Title: flow.hardTitle(), Prompt: flow.hardEffect() + "\n\n" + flow.prompt(),
		HandleConfirm: func() error { return flow.execute("hard", approved) },
		HandleClose:   flow.target.checkRepository,
	})
	return nil
}

func (flow *branchActionResetFlow) showHardRiskReview(paths []string, fingerprint string) error {
	c := flow.target.c
	return ShowIgnoredRiskMenu(c, IgnoredRiskReview{
		CommandPrefix: "reset",
		Title:         c.Tr.ResetIgnoredTitle,
		Prompt:        c.Tr.ResetIgnoredPrompt + "\n\n" + flow.prompt(),
		ApproveLabel:  c.Tr.ResetApproveIgnored,
		Paths:         paths,
		Approve:       func() error { return flow.execute("hard", fingerprint) },
		Reopen:        func() error { return flow.showHardRiskReview(paths, fingerprint) },
	})
}

// confirmChangedHardReview notes that the reviewed risks changed, then shows the
// fresh review. Previous approvals no longer apply.
func (flow *branchActionResetFlow) confirmChangedHardReview(paths []string, fingerprint string) error {
	c := flow.target.c
	c.Confirm(types.ConfirmOpts{
		Title: c.Tr.ResetChangedTitle, Prompt: c.Tr.ResetIgnoredChangedNote,
		HandleConfirm: func() error { return flow.presentHard(paths, fingerprint) },
		HandleClose:   flow.target.checkRepository,
	})
	return nil
}

func (flow *branchActionResetFlow) hardTitle() string {
	if flow.opts.title != "" {
		return flow.opts.title
	}
	return flow.target.c.Tr.Actions.HardReset
}

func (flow *branchActionResetFlow) hardEffect() string {
	if flow.opts.effect != "" {
		return flow.opts.effect
	}
	return flow.target.c.Tr.BranchActionResetHardWarning
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
					return flow.execute(row.mode, "")
				}
				return flow.reviewHard(false)
			},
		})
	}
	return flow.target.c.Menu(types.CreateMenuOptions{
		Title: fmt.Sprintf("%s %s", flow.target.c.Tr.ResetTo, flow.label), Prompt: flow.prompt(),
		Items: items, OnCancel: flow.target.checkRepository,
	})
}

// execute runs the reset once. For a hard reset, approved is the fingerprint of the
// risks that the user reviewed; the command refuses a different list.
func (flow *branchActionResetFlow) execute(mode, approved string) error {
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
		if flow.opts.preflight != nil {
			if err := flow.opts.preflight(flow.target.git.Branch); err != nil {
				return err
			}
		}
		cmd, err := flow.target.git.Branch.BranchActionResetCommand(flow.snapshot, mode, approved, flow.opts.envVars...)
		if err == nil {
			if err = flow.target.checkRepository(); err == nil {
				err = cmd.Run()
			}
		}
		if repositoryErr := flow.target.checkRepository(); repositoryErr != nil {
			return errors.Join(err, repositoryErr)
		}
		var riskErr *git_commands.ResetIgnoredRiskError
		if errors.As(err, &riskErr) {
			// Nothing ran. The risks changed after review, so the fresh review follows.
			flow.target.c.OnUIThread(func() error { return flow.reopened().reviewHard(true) })
			return nil
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

// reopened returns a flow that may execute once more. A refused approval never
// ran the reset, so its successor reviews the fresh risks under a new approval.
func (flow *branchActionResetFlow) reopened() *branchActionResetFlow {
	next := *flow
	next.started = false
	return &next
}
