package controllers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// branchUpstreamFlow reviews an exact native tracking ref for one captured
// branch and repository. It deliberately makes no remote/ref mapping guesses.
type branchUpstreamFlow struct {
	target *branchSyncTarget
}

// Capture repository identity when the upstream-options menu is created, not
// when one of its later callbacks is invoked.
func newBranchUpstreamFlow(c *ControllerCommon, name string) *branchUpstreamFlow {
	return &branchUpstreamFlow{target: newBranchSyncTarget(c, branchSyncSource{name: name, selected: true})}
}

func (flow *branchUpstreamFlow) set() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.target.inspect(func(info git_commands.BranchSyncInfo) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		flow.target.c.Prompt(types.PromptOpts{
			Title:               flow.target.c.Tr.BranchUpstreamRefPrompt,
			InitialContent:      info.UpstreamTrackingRef,
			PreserveWhitespace:  true,
			AllowEditSuggestion: true,
			HandleClose:         flow.target.checkRepository,
			HandleConfirm: func(ref string) error {
				if err := flow.target.checkRepository(); err != nil {
					return err
				}
				return flow.prepare(git_commands.BranchUpstreamOptions{TrackingRef: ref}, info.Name, info.OID)
			},
		})
		return nil
	})
}

func (flow *branchUpstreamFlow) unset() error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.target.inspect(func(info git_commands.BranchSyncInfo) error {
		return flow.prepare(git_commands.BranchUpstreamOptions{Unset: true}, info.Name, info.OID)
	})
}

func (flow *branchUpstreamFlow) prepare(opts git_commands.BranchUpstreamOptions, name, sourceOID string) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.target.c.WithWaitingStatus(flow.target.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		plan, err := flow.target.git.Sync.PrepareBranchUpstream(name, sourceOID, opts)
		if err != nil {
			return err
		}
		flow.target.c.OnUIThread(func() error {
			if err := flow.target.checkRepository(); err != nil {
				return err
			}
			return flow.review(plan)
		})
		return nil
	})
}

func (flow *branchUpstreamFlow) review(plan git_commands.BranchUpstreamPlan) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	var prompt string
	previousRemote := fmt.Sprintf("%q", plan.PreviousRemote())
	if strings.ContainsRune(previousRemote, ':') {
		previousRemote = safeBranchSyncURL(plan.PreviousRemote())
	}
	if plan.Unset() {
		prompt = fmt.Sprintf(flow.target.c.Tr.BranchUpstreamUnsetReviewPrompt,
			plan.Name(), plan.SourceOID(), previousRemote, plan.PreviousMergeRefs())
	} else {
		prompt = fmt.Sprintf(flow.target.c.Tr.BranchUpstreamSetReviewPrompt,
			plan.Name(), plan.SourceOID(), plan.TrackingRef(), plan.TargetOID(), previousRemote, plan.PreviousMergeRefs())
	}
	flow.target.c.Confirm(types.ConfirmOpts{
		Title:  flow.target.c.Tr.BranchUpstreamReviewTitle,
		Prompt: prompt,
		HandleConfirm: func() error {
			if err := flow.target.checkRepository(); err != nil {
				return err
			}
			return flow.execute(plan)
		},
		HandleClose: flow.target.checkRepository,
	})
	return nil
}

func (flow *branchUpstreamFlow) execute(plan git_commands.BranchUpstreamPlan) error {
	if err := flow.target.checkRepository(); err != nil {
		return err
	}
	return flow.target.c.WithWaitingStatus(fmt.Sprintf(flow.target.c.Tr.BranchUpstreamStatus, plan.Name()), func(gocui.Task) (runErr error) {
		defer func() {
			if flow.target.c.Git() == flow.target.git && flow.target.c.State().GetRepoGeneration() == flow.target.generation {
				flow.target.c.RefreshFromWorker(types.RefreshOptions{Scope: []types.RefreshableView{types.BRANCHES, types.COMMITS}})
			} else {
				runErr = errors.Join(runErr, flow.target.checkRepository())
			}
		}()
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		cmd, err := flow.target.git.Sync.BranchUpstreamCommand(plan)
		if err != nil {
			return err
		}
		if err := flow.target.checkRepository(); err != nil {
			return err
		}
		return cmd.Run()
	})
}
