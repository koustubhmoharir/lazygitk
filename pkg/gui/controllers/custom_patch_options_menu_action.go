package controllers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"

	"github.com/jesseduffield/generics/set"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

type CustomPatchOptionsMenuAction struct {
	c *ControllerCommon
}

func (self *CustomPatchOptionsMenuAction) Call() error {
	if !self.c.Git().Patch.PatchBuilder.Active() {
		return errors.New(self.c.Tr.NoPatchError)
	}

	if self.c.Git().Patch.PatchBuilder.IsEmpty() {
		return errors.New(self.c.Tr.EmptyPatchError)
	}

	menuItems := []*types.MenuItem{
		{
			CommandID: "patch.clear",
			Verb:      "clear-patch",
			Label:     self.c.Tr.ResetPatch,
			Tooltip:   self.c.Tr.ResetPatchTooltip,
			OnPress:   self.c.Helpers().CustomPatch.Reset,
			Keys:      menuKey('c'),
		},
		{
			CommandID: "patch.apply",
			Verb:      "apply-patch",
			Label:     self.c.Tr.ApplyPatch,
			Tooltip:   self.c.Tr.ApplyPatchTooltip,
			OnPress:   func() error { return self.handleApplyPatch(false) },
			Keys:      menuKey('a'),
		},
		{
			CommandID: "patch.apply-reverse",
			Verb:      "reverse-apply-patch",
			Label:     self.c.Tr.ApplyPatchInReverse,
			Tooltip:   self.c.Tr.ApplyPatchInReverseTooltip,
			OnPress:   func() error { return self.handleApplyPatch(true) },
			Keys:      menuKey('r'),
		},
	}

	if self.c.Git().Patch.PatchBuilder.CanRebase && self.c.Git().Status.WorkingTreeState().None() {
		// Fork: the source-editing items are disabled when the source is not in the list;
		// see fork_patch_source.go.
		sourceReason := self.patchSourceDisabledReason()
		menuItems = append(menuItems, []*types.MenuItem{
			{
				CommandID:      "patch.remove-from-source-commit",
				Verb:           "remove-patch-from-commit",
				Label:          fmt.Sprintf(self.c.Tr.RemovePatchFromOriginalCommit, utils.ShortHash(self.c.Git().Patch.PatchBuilder.To)),
				Tooltip:        self.c.Tr.RemovePatchFromOriginalCommitTooltip,
				OnPress:        self.handleDeletePatchFromCommit,
				Keys:           menuKey('d'),
				DisabledReason: sourceReason,
			},
			{
				CommandID:      "patch.move-to-index",
				Verb:           "move-patch",
				Label:          self.c.Tr.MovePatchOutIntoIndex,
				Tooltip:        self.c.Tr.MovePatchOutIntoIndexTooltip,
				OnPress:        self.handleMovePatchIntoWorkingTree,
				Keys:           menuKey('i'),
				DisabledReason: sourceReason,
			},
			{
				CommandID:      "patch.move-to-new-commit",
				Verb:           "move-patch",
				Label:          self.c.Tr.MovePatchIntoNewCommit,
				Tooltip:        self.c.Tr.MovePatchIntoNewCommitTooltip,
				OnPress:        self.handlePullPatchIntoNewCommit,
				Keys:           menuKey('n'),
				DisabledReason: sourceReason,
			},
			{
				CommandID:      "patch.move-to-new-commit-before",
				Verb:           "move-patch",
				Label:          self.c.Tr.MovePatchIntoNewCommitBefore,
				Tooltip:        self.c.Tr.MovePatchIntoNewCommitBeforeTooltip,
				OnPress:        self.handlePullPatchIntoNewCommitBefore,
				Keys:           menuKey('N'),
				DisabledReason: sourceReason,
			},
		}...)

		if self.c.Context().Current().GetKey() == self.c.Contexts().LocalCommits.GetKey() {
			selectedCommit := self.c.Contexts().LocalCommits.GetSelected()
			if selectedCommit != nil && self.c.Git().Patch.PatchBuilder.To != selectedCommit.Hash() {

				disabledReason := sourceReason
				if disabledReason == nil && self.c.Contexts().LocalCommits.AreMultipleItemsSelected() {
					disabledReason = &types.DisabledReason{Text: self.c.Tr.RangeSelectNotSupported}
				}

				// adding this option to index 1
				menuItems = append(
					menuItems[:1],
					append(
						[]*types.MenuItem{
							{
								CommandID:      "patch.move-to-selected-commit",
								Verb:           "move-patch",
								Label:          fmt.Sprintf(self.c.Tr.MovePatchToSelectedCommit, selectedCommit.Hash()),
								Tooltip:        self.c.Tr.MovePatchToSelectedCommitTooltip,
								OnPress:        self.handleMovePatchToSelectedCommit,
								Keys:           menuKey('m'),
								DisabledReason: disabledReason,
							},
						}, menuItems[1:]...,
					)...,
				)
			}
		}
	}

	menuItems = append(menuItems, []*types.MenuItem{
		{
			CommandID: "clipboard.copy-patch",
			Verb:      "copy",
			Label:     self.c.Tr.CopyPatchToClipboard,
			OnPress:   func() error { return self.copyPatchToClipboard() },
			Keys:      menuKey('y'),
		},
	}...)

	return self.c.Menu(types.CreateMenuOptions{Title: self.c.Tr.PatchOptionsTitle, Items: menuItems})
}

func (self *CustomPatchOptionsMenuAction) getPatchCommitIndex() int {
	for index, commit := range self.c.Model().Commits {
		if commit.Hash() == self.c.Git().Patch.PatchBuilder.To {
			return index
		}
	}
	return -1
}

func (self *CustomPatchOptionsMenuAction) handleDeletePatchFromCommit() error {
	commits := self.c.Model().Commits
	// Fork: refuses when the source is not in the list; see fork_patch_source.go.
	commitIndex, err := self.patchSourceCommitIndex()
	if err != nil {
		return err
	}
	return self.c.WithWaitingStatus(self.c.Tr.RebasingStatus, func(gocui.Task) error {
		self.c.LogAction(self.c.Tr.Actions.RemovePatchFromCommit)
		// Fork: the start is reviewed for ignored files; see fork_rebase_ignored_review.go.
		return runWithIgnoredRebaseReview(self.c, func(_ *git_commands.RebaseCommands) error {
			return self.c.Git().Patch.DeletePatchesFromCommit(commits, commitIndex)
		}, self.c.Helpers().MergeAndRebase.CheckMergeOrRebase)
	})
}

func (self *CustomPatchOptionsMenuAction) handleMovePatchToSelectedCommit() error {
	commits := self.c.Model().Commits
	// Fork: refuses when the source is not in the list; see fork_patch_source.go.
	commitIndex, err := self.patchSourceCommitIndex()
	if err != nil {
		return err
	}
	toCommitIndex := self.c.Contexts().LocalCommits.GetSelectedLineIdx()
	return self.c.WithWaitingStatus(self.c.Tr.RebasingStatus, func(gocui.Task) error {
		self.c.LogAction(self.c.Tr.Actions.MovePatchToSelectedCommit)
		// Fork: the start is reviewed for ignored files; see fork_rebase_ignored_review.go.
		return runWithIgnoredRebaseReview(self.c, func(_ *git_commands.RebaseCommands) error {
			return self.c.Git().Patch.MovePatchToSelectedCommit(commits, commitIndex, toCommitIndex)
		}, self.c.Helpers().MergeAndRebase.CheckMergeOrRebase)
	})
}

func (self *CustomPatchOptionsMenuAction) handleMovePatchIntoWorkingTree() error {
	commits := self.c.Model().Commits
	// Fork: refuses when the source is not in the list; see fork_patch_source.go.
	commitIndex, err := self.patchSourceCommitIndex()
	if err != nil {
		return err
	}
	// This replaces the upstream stash confirmation; the backend clean check
	// reports that changes must be committed before patch movement can rebase.
	return self.c.WithWaitingStatus(self.c.Tr.RebasingStatus, func(gocui.Task) error {
		self.c.LogAction(self.c.Tr.Actions.MovePatchIntoIndex)
		// Fork: the start is reviewed for ignored files; see fork_rebase_ignored_review.go.
		return runWithIgnoredRebaseReview(self.c, func(_ *git_commands.RebaseCommands) error {
			return self.c.Git().Patch.MovePatchIntoIndex(commits, commitIndex)
		}, self.c.Helpers().MergeAndRebase.CheckMergeOrRebase)
	})
}

func (self *CustomPatchOptionsMenuAction) handlePullPatchIntoNewCommit() error {
	// Fork: refuses when the source is not in the list; see fork_patch_source.go.
	commitIndex, err := self.patchSourceCommitIndex()
	if err != nil {
		return err
	}
	self.c.Helpers().Commits.OpenCommitMessagePanel(
		&helpers.OpenCommitMessagePanelOpts{
			// Pass a commit index of one less than the moved-from commit, so that
			// you can press up arrow once to recall the original commit message:
			CommitIndex:      commitIndex - 1,
			InitialMessage:   "",
			SummaryTitle:     self.c.Tr.CommitSummaryTitle,
			DescriptionTitle: self.c.Tr.CommitDescriptionTitle,
			PreserveMessage:  false,
			// Fork: the pull reports success; see TakeCommitMessageSubmitReporter.
			ReportsSubmitResult: true,
			OnConfirm: func(summary string, description string) error {
				// Fork: see TakeCommitMessageSubmitReporter.
				reportSuccess := self.c.Helpers().Commits.TakeCommitMessageSubmitReporter()
				commits := self.c.Model().Commits
				self.c.Helpers().Commits.CloseCommitMessagePanel()
				return self.c.WithWaitingStatus(self.c.Tr.RebasingStatus, func(gocui.Task) error {
					self.c.LogAction(self.c.Tr.Actions.MovePatchIntoNewCommit)
					// Fork: the start is reviewed for ignored files; see fork_rebase_ignored_review.go.
					return runWithIgnoredRebaseReview(self.c, func(_ *git_commands.RebaseCommands) error {
						return self.c.Git().Patch.PullPatchIntoNewCommit(commits, commitIndex, summary, description)
					}, func(err error) error {
						if err := self.c.Helpers().MergeAndRebase.CheckMergeOrRebase(err); err != nil {
							return err
						}
						// Fork: the new commit exists.
						reportSuccess()
						self.c.OnUIThread(func() error {
							self.c.Context().Push(self.c.Contexts().LocalCommits, types.OnFocusOpts{})
							return nil
						})
						return nil
					})
				})
			},
		},
	)

	return nil
}

func (self *CustomPatchOptionsMenuAction) handlePullPatchIntoNewCommitBefore() error {
	// Fork: refuses when the source is not in the list; see fork_patch_source.go.
	commitIndex, err := self.patchSourceCommitIndex()
	if err != nil {
		return err
	}
	self.c.Helpers().Commits.OpenCommitMessagePanel(
		&helpers.OpenCommitMessagePanelOpts{
			// Pass a commit index of one less than the moved-from commit, so that
			// you can press up arrow once to recall the original commit message:
			CommitIndex:      commitIndex - 1,
			InitialMessage:   "",
			SummaryTitle:     self.c.Tr.CommitSummaryTitle,
			DescriptionTitle: self.c.Tr.CommitDescriptionTitle,
			PreserveMessage:  false,
			// Fork: the pull reports success; see TakeCommitMessageSubmitReporter.
			ReportsSubmitResult: true,
			OnConfirm: func(summary string, description string) error {
				// Fork: see TakeCommitMessageSubmitReporter.
				reportSuccess := self.c.Helpers().Commits.TakeCommitMessageSubmitReporter()
				commits := self.c.Model().Commits
				self.c.Helpers().Commits.CloseCommitMessagePanel()
				return self.c.WithWaitingStatus(self.c.Tr.RebasingStatus, func(gocui.Task) error {
					self.c.LogAction(self.c.Tr.Actions.MovePatchIntoNewCommit)
					// Fork: the start is reviewed for ignored files; see fork_rebase_ignored_review.go.
					return runWithIgnoredRebaseReview(self.c, func(_ *git_commands.RebaseCommands) error {
						return self.c.Git().Patch.PullPatchIntoNewCommitBefore(commits, commitIndex, summary, description)
					}, func(err error) error {
						if err := self.c.Helpers().MergeAndRebase.CheckMergeOrRebase(err); err != nil {
							return err
						}
						// Fork: the new commit exists.
						reportSuccess()
						self.c.OnUIThread(func() error {
							self.c.Context().Push(self.c.Contexts().LocalCommits, types.OnFocusOpts{})
							return nil
						})
						return nil
					})
				})
			},
		},
	)

	return nil
}

func (self *CustomPatchOptionsMenuAction) handleApplyPatch(reverse bool) error {
	affectedUnstagedFiles := self.getAffectedUnstagedFiles()

	mustStageFiles := len(affectedUnstagedFiles) > 0
	return self.c.ConfirmIf(mustStageFiles, types.ConfirmOpts{
		Title:  self.c.Tr.MustStageFilesAffectedByPatchTitle,
		Prompt: self.c.Tr.MustStageFilesAffectedByPatchWarning,
		HandleConfirm: func() error {
			action := self.c.Tr.Actions.ApplyPatch
			if reverse {
				action = "Apply patch in reverse"
			}
			self.c.LogAction(action)

			if mustStageFiles {
				if err := self.c.Git().WorkingTree.StageFiles(affectedUnstagedFiles, nil); err != nil {
					return err
				}
			}

			if err := self.c.Git().Patch.ApplyCustomPatch(reverse, true); err != nil {
				return err
			}

			self.c.Refresh(types.RefreshOptions{})
			return nil
		},
	})
}

func (self *CustomPatchOptionsMenuAction) copyPatchToClipboard() error {
	patch := self.c.Git().Patch.PatchBuilder.RenderAggregatedPatch(true)

	self.c.LogAction(self.c.Tr.Actions.CopyPatchToClipboard)
	if err := self.c.OS().CopyToClipboard(patch); err != nil {
		return err
	}

	self.c.Toast(self.c.Tr.PatchCopiedToClipboard)

	return nil
}

// Returns a list of files that have unstaged changes and are contained in the patch.
func (self *CustomPatchOptionsMenuAction) getAffectedUnstagedFiles() []string {
	unstagedFiles := set.NewFromSlice(lo.FilterMap(self.c.Model().Files, func(f *models.File, _ int) (string, bool) {
		if f.GetHasUnstagedChanges() {
			return f.GetPath(), true
		}
		return "", false
	}))

	return lo.Filter(self.c.Git().Patch.PatchBuilder.AllFilesInPatch(), func(patchFile string, _ int) bool {
		return unstagedFiles.Includes(patchFile)
	})
}
