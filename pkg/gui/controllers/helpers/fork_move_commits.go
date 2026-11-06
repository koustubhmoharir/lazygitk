package helpers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// Fork: g m n moves the unpushed commits of the checked-out branch to a new
// branch. This replaces upstream's MoveCommitsToNewBranch and its two variants
// in refs_helper.go, which reset the branch before the destination existed,
// read live HEAD after the prompts and stashed automatically. Here the source
// is captured when the action starts (see git_commands/fork_move_commits.go),
// the name is validated before any write, tracked changes refuse rather than
// stash, and the old branch moves only after the new branch holds the commits.

// moveCommitsRefreshOptions is the refresh upstream used after this action.
func moveCommitsRefreshOptions() types.RefreshOptions {
	return types.RefreshOptions{
		BatchUIUpdates:        true,
		BranchSelection:       types.SelectCheckedOutBranch,
		CommitSelection:       types.SelectHeadCommit,
		SelectTopReflogCommit: true,
	}
}

// moveCommitsToNewBranch is the entry point of g m n. It captures the source on
// a worker, then offers the choice on the UI thread.
func (self *RefsHelper) moveCommitsToNewBranch() error {
	repository := captureActionRepository(self.c)
	mainBranches := self.c.Model().MainBranches
	return self.c.WithWaitingStatus(self.c.Tr.MoveCommitsReadingStatus, func(gocui.Task) error {
		if err := repository.checkRepository(); err != nil {
			return err
		}
		src, err := repository.git.Branch.CaptureMoveCommitsSource()
		if err != nil {
			return self.moveCommitsError(err, "", "")
		}
		currentName := ShortBranchName(src.HeadRef())
		isMainBranch := lo.Contains(self.c.UserConfig().Git.MainBranches, currentName)
		baseName := ""
		if !isMainBranch && mainBranches != nil {
			baseRef, err := repository.git.Loaders.BranchLoader.GetBaseBranch(
				&models.Branch{Name: currentName, CommitHash: src.HeadOID()}, mainBranches)
			if err != nil {
				return err
			}
			if baseRef != "" {
				src, err = repository.git.Branch.WithMoveCommitsBase(src, baseRef)
				if err != nil {
					return err
				}
				baseName = ShortBranchName(baseRef)
			}
		}
		self.c.OnUIThread(func() error {
			if err := repository.checkRepository(); err != nil {
				return err
			}
			if isMainBranch {
				self.confirmMoveCommitsFromMain(repository, src, currentName)
				return nil
			}
			self.chooseMoveCommitsVariant(repository, src, currentName, baseName)
			return nil
		})
		return nil
	})
}

// confirmMoveCommitsFromMain is the main-branch case: the only choice is the
// stacked variant, which upstream also used without a menu.
func (self *RefsHelper) confirmMoveCommitsFromMain(repository *capturedActionRepository, src git_commands.MoveCommitsSource, currentName string) {
	prompt := utils.ResolvePlaceholderString(
		self.c.Tr.MoveCommitsToNewBranchFromMainPrompt,
		map[string]string{"baseBranchName": currentName},
	)
	self.c.Confirm(types.ConfirmOpts{
		Title:  self.c.Tr.MoveCommitsToNewBranch,
		Prompt: prompt,
		HandleConfirm: func() error {
			return self.promptMoveCommitsName(repository, src, git_commands.MoveCommitsStacked, currentName)
		},
	})
}

func (self *RefsHelper) chooseMoveCommitsVariant(repository *capturedActionRepository, src git_commands.MoveCommitsSource, currentName, baseName string) {
	prompt := utils.ResolvePlaceholderString(
		self.c.Tr.MoveCommitsToNewBranchMenuPrompt,
		map[string]string{"baseBranchName": baseName},
	)
	baseItem := &types.MenuItem{
		Label:     fmt.Sprintf(self.c.Tr.MoveCommitsToNewBranchFromBaseItem, baseName),
		CommandID: "branch.move-commits-to-new-from-base",
		Verb:      "move-commits",
		OnPress: func() error {
			return self.promptMoveCommitsName(repository, src, git_commands.MoveCommitsFromBase, baseName)
		},
	}
	if baseName == "" {
		baseItem.DisabledReason = &types.DisabledReason{Text: self.c.Tr.CouldNotDetermineBaseBranch}
		baseItem.OnPress = func() error { return nil }
	}
	self.c.Menu(types.CreateMenuOptions{
		Title:  self.c.Tr.MoveCommitsToNewBranch,
		Prompt: prompt,
		Items: []*types.MenuItem{
			baseItem,
			{
				Label:     fmt.Sprintf(self.c.Tr.MoveCommitsToNewBranchStackedItem, currentName),
				CommandID: "branch.move-commits-to-new-stacked",
				Verb:      "move-commits",
				OnPress: func() error {
					return self.promptMoveCommitsName(repository, src, git_commands.MoveCommitsStacked, currentName)
				},
			},
		},
	})
}

func (self *RefsHelper) promptMoveCommitsName(repository *capturedActionRepository, src git_commands.MoveCommitsSource, variant git_commands.MoveCommitsVariant, promptBaseName string) error {
	prompt := utils.ResolvePlaceholderString(
		self.c.Tr.NewBranchNameBranchOff,
		map[string]string{"branchName": promptBaseName},
	)
	suggestedBranchName, err := self.getSuggestedBranchName()
	if err != nil {
		return err
	}
	self.c.Prompt(types.PromptOpts{
		Title:          prompt,
		InitialContent: suggestedBranchName,
		HandleConfirm: func(response string) error {
			self.c.LogAction(self.c.Tr.MoveCommitsToNewBranch)
			name := SanitizedBranchName(response)
			return self.c.WithWaitingStatus(self.c.Tr.MovingCommitsToNewBranchStatus, func(gocui.Task) error {
				return self.runMoveCommits(repository, src, variant, name)
			})
		},
	})
	return nil
}

// runMoveCommits validates the name and the captured state, then performs the
// variant's writes. It runs on a worker.
func (self *RefsHelper) runMoveCommits(repository *capturedActionRepository, src git_commands.MoveCommitsSource, variant git_commands.MoveCommitsVariant, name string) error {
	if err := repository.checkRepository(); err != nil {
		return err
	}
	oldName := ShortBranchName(src.HeadRef())
	plan, err := repository.git.Branch.PrepareMoveCommits(variant, src, name)
	if err != nil {
		return self.moveCommitsError(err, name, oldName)
	}
	if variant == git_commands.MoveCommitsStacked {
		err := repository.git.Branch.ApplyMoveCommitsStacked(plan)
		self.c.RefreshFromWorker(moveCommitsRefreshOptions())
		return self.moveCommitsError(err, name, oldName)
	}
	return self.runMoveCommitsFromBase(repository, plan, oldName)
}

// runMoveCommitsFromBase creates the new branch, cherry-picks the captured
// commits onto it, and moves the old branch once the picks have all landed.
func (self *RefsHelper) runMoveCommitsFromBase(repository *capturedActionRepository, plan git_commands.MoveCommitsPlan, oldName string) error {
	name := plan.Name()
	if err := repository.git.Branch.ApplyMoveCommitsFromBaseCreate(plan); err != nil {
		return self.moveCommitsError(err, name, oldName)
	}
	pickErr := repository.git.Rebase.CherryPickCommits(plan.CherryPickCommits())
	handled := self.rebaseHelper.CheckMergeOrRebaseWithRefreshOptions(pickErr, moveCommitsRefreshOptions())
	if handled != nil {
		return self.moveCommitsError(handled, name, oldName)
	}
	if !repository.git.Status.WorkingTreeState().None() {
		// A pick stopped on conflicts. The old branch keeps its commits until the
		// operation ends by continue or skip; an abort leaves them where they are.
		self.c.OnUIThread(func() error {
			self.rebaseHelper.SetOnOperationFinished(func() error {
				return self.c.WithWaitingStatus(self.c.Tr.MoveCommitsFinishingStatus, func(gocui.Task) error {
					return self.finishMoveCommitsFromBase(repository, plan, oldName)
				})
			})
			return nil
		})
		return nil
	}
	return self.finishMoveCommitsFromBase(repository, plan, oldName)
}

func (self *RefsHelper) finishMoveCommitsFromBase(repository *capturedActionRepository, plan git_commands.MoveCommitsPlan, oldName string) error {
	if err := repository.checkRepository(); err != nil {
		return err
	}
	err := repository.git.Branch.ApplyMoveCommitsFromBaseFinish(plan)
	self.c.RefreshFromWorker(moveCommitsRefreshOptions())
	return self.moveCommitsError(err, plan.Name(), oldName)
}

// moveCommitsError maps the flow's refusals and step failures to fork strings.
// Other errors, such as Git's own, pass through unchanged.
func (self *RefsHelper) moveCommitsError(err error, newName, oldName string) error {
	if err == nil {
		return nil
	}
	tr := self.c.Tr
	var stepErr *git_commands.MoveCommitsStepError
	if errors.As(err, &stepErr) {
		cause := stepErr.Cause
		switch stepErr.Step {
		case git_commands.MoveCommitsStepCreate:
			return fmt.Errorf(tr.MoveCommitsCreateFailed, newName, cause)
		case git_commands.MoveCommitsStepCheckout:
			return fmt.Errorf(tr.MoveCommitsCheckoutFailed, newName, newName, oldName, cause)
		case git_commands.MoveCommitsStepCreateFromBase:
			return fmt.Errorf(tr.MoveCommitsCreateFromBaseFailed, newName, cause)
		case git_commands.MoveCommitsStepMoveOld:
			return self.moveCommitsOldBranchError(cause, newName, oldName)
		}
	}
	switch {
	case errors.Is(err, git_commands.ErrMoveCommitsTrackedChanges):
		return errors.New(tr.MoveCommitsTrackedChanges)
	case errors.Is(err, git_commands.ErrMoveCommitsChanged):
		return errors.New(tr.MoveCommitsChanged)
	case errors.Is(err, git_commands.ErrMoveCommitsNameOption):
		return errors.New(tr.MoveCommitsNameOption)
	case errors.Is(err, git_commands.ErrMoveCommitsNameExists):
		return fmt.Errorf(tr.MoveCommitsNameExists, newName)
	}
	return err
}

func (self *RefsHelper) moveCommitsOldBranchError(cause error, newName, oldName string) error {
	tr := self.c.Tr
	switch {
	case errors.Is(cause, git_commands.ErrMoveCommitsOldBranchChanged):
		return fmt.Errorf(tr.MoveCommitsOldBranchChanged, oldName, newName)
	case errors.Is(cause, git_commands.ErrMoveCommitsOldBranchInUse):
		return fmt.Errorf(tr.MoveCommitsOldBranchInUse, oldName, newName)
	case errors.Is(cause, git_commands.ErrMoveCommitsIncomplete):
		return fmt.Errorf(tr.MoveCommitsIncomplete, oldName, newName)
	}
	return fmt.Errorf(tr.MoveCommitsOldBranchNotMoved, newName, oldName, cause)
}
