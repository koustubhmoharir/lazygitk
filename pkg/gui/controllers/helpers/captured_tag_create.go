package helpers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

type capturedTagCreateFlow struct {
	repository  *capturedActionRepository
	overrideGpg bool
	onCreate    func()
	submitted   bool
	executing   bool
	cancelled   bool
	callbackRun bool
	target      git_commands.BranchActionTarget
	plan        git_commands.BranchActionTagCreatePlan
}

// openCapturedCreateTagPrompt captures the selected reference and repository
// before any asynchronous inspection or UI callback can run.
func (self *TagsHelper) openCapturedCreateTagPrompt(ref string, onCreate func()) error {
	flow := &capturedTagCreateFlow{
		repository:  captureActionRepository(self.c),
		overrideGpg: self.c.UserConfig().Git.OverrideGpg,
		onCreate:    onCreate,
	}
	return self.c.WithWaitingStatus(self.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.repository.checkRepository(); err != nil {
			return err
		}
		var err error
		if ref == "" {
			flow.target, err = flow.repository.git.Branch.PrepareBranchActionHeadTarget()
		} else {
			var oid string
			oid, err = flow.repository.git.Branch.ResolveBranchActionCommit(ref)
			if err == nil {
				flow.target, err = flow.repository.git.Branch.PrepareBranchActionTarget(ref, oid)
			}
		}
		if err != nil {
			return err
		}
		flow.repository.c.OnUIThread(func() error { return flow.openMessagePanel(self.commitsHelper) })
		return nil
	})
}

func (flow *capturedTagCreateFlow) checkUI() error {
	if err := flow.repository.checkRepository(); err != nil {
		return err
	}
	if flow.executing || flow.cancelled {
		return git_commands.ErrBranchActionChanged
	}
	return nil
}

func (flow *capturedTagCreateFlow) openMessagePanel(commits *CommitsHelper) error {
	if err := flow.checkUI(); err != nil {
		return err
	}
	c := flow.repository.c
	commits.OpenCommitMessagePanel(&OpenCommitMessagePanelOpts{
		CommitIndex: context.NoCommitIndex, InitialMessage: "",
		SummaryTitle: c.Tr.TagNameTitle, DescriptionTitle: c.Tr.TagMessageTitle,
		PreserveMessage: false,
		OnConfirm:       flow.submitName,
	})
	return nil
}

func (flow *capturedTagCreateFlow) submitName(name, description string) error {
	if err := flow.checkUI(); err != nil {
		return err
	}
	if flow.submitted {
		return git_commands.ErrBranchActionChanged
	}
	flow.submitted = true
	c := flow.repository.c
	return c.WithWaitingStatus(c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.repository.checkRepository(); err != nil {
			return err
		}
		if err := flow.repository.git.Branch.CheckBranchActionTarget(flow.target); err != nil {
			return err
		}
		plan, err := flow.repository.git.Tag.PrepareBranchActionTagCreate(flow.target, name, description)
		if err != nil {
			return err
		}
		flow.plan = plan
		c.OnUIThread(func() error { return flow.review() })
		return nil
	})
}

func (flow *capturedTagCreateFlow) review() error {
	if err := flow.checkUI(); err != nil {
		return err
	}
	c := flow.repository.c
	if !flow.plan.DestinationExists() {
		return flow.execute(flow.plan)
	}
	prompt := utils.ResolvePlaceholderString(c.Tr.ForceTagPrompt, map[string]string{
		"tagName":    flow.plan.Name(),
		"cancelKey":  c.UserConfig().Keybinding.Universal.Return.String(),
		"confirmKey": c.UserConfig().Keybinding.Universal.Confirm.String(),
	})
	prompt += "\n\n" + fmt.Sprintf(c.Tr.BranchActionTagCreateReview,
		flow.plan.Name(), flow.plan.OldOID(), flow.plan.Target().SourceRef(), flow.plan.Target().SourceOID())
	c.Confirm(types.ConfirmOpts{
		Title: c.Tr.ForceTag, Prompt: prompt, HandleClose: flow.cancel,
		HandleConfirm: func() error { return flow.execute(flow.plan.WithOverwriteApproved()) },
	})
	return nil
}

func (flow *capturedTagCreateFlow) execute(plan git_commands.BranchActionTagCreatePlan) error {
	if err := flow.repository.checkRepository(); err != nil {
		return err
	}
	if !flow.submitted || flow.executing || flow.cancelled {
		return git_commands.ErrBranchActionChanged
	}
	flow.executing = true
	repository := flow.repository
	c := repository.c
	return c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{Message: c.Tr.CreatingTag}, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		cmd, err := repository.git.Tag.BranchActionTagCreateCommand(plan)
		if err != nil {
			return flow.finish(err)
		}
		if plan.Annotated() {
			c.LogAction(c.Tr.Actions.CreateAnnotatedTag)
		} else {
			c.LogAction(c.Tr.Actions.CreateLightweightTag)
		}
		if plan.Signed() && !flow.overrideGpg {
			c.OnUIThread(func() error { return flow.runTerminal(plan) })
			return nil
		}
		return flow.finish(cmd.Run())
	})
}

func (flow *capturedTagCreateFlow) cancel() error {
	if err := flow.repository.checkRepository(); err != nil {
		return err
	}
	if flow.executing || flow.cancelled {
		return git_commands.ErrBranchActionChanged
	}
	flow.cancelled = true
	return nil
}

func (flow *capturedTagCreateFlow) runTerminal(plan git_commands.BranchActionTagCreatePlan) error {
	repository := flow.repository
	if err := repository.checkRepository(); err != nil {
		return err
	}
	cmd, err := repository.git.Tag.BranchActionTagCreateCommand(plan)
	if err != nil {
		return flow.finish(err)
	}
	_, result := repository.c.RunSubprocess(cmd)
	if err := repository.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	return flow.finish(result)
}

func (flow *capturedTagCreateFlow) finish(result error) error {
	repository := flow.repository
	if err := repository.checkRepository(); err != nil {
		return errors.Join(result, err)
	}
	repository.c.OnUIThread(func() error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		repository.c.Refresh(types.RefreshOptions{
			Scope: []types.RefreshableView{types.COMMITS, types.TAGS}, BatchUIUpdates: true,
			Then: func() error {
				if err := repository.checkRepository(); err != nil {
					return err
				}
				if result == nil && flow.onCreate != nil && !flow.callbackRun {
					flow.callbackRun = true
					flow.onCreate()
				}
				return nil
			},
		})
		return nil
	})
	return result
}
