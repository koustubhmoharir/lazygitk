package controllers

import (
	"errors"
	"fmt"
	"os"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

type historicalDiffSelection struct {
	oid          string
	path         string
	previousPath string
	isFile       bool
	parents      []string
	deleted      bool
}

func (self *BasicCommitsController) historicalDiffBindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			CommandID: "diff.commit.worktree", Verb: "diff-worktree",
			Keys:              opts.GetKeys(opts.Config.Universal.DiffWithWorkingTree),
			Handler:           self.diffCommitWithWorktree,
			GetDisabledReason: self.require(func() *types.DisabledReason { return self.canHistoricalDiffCommit(false) }),
			Description:       self.c.Tr.DiffWithWorkingTree, DisplayOnScreen: true,
		},
		{
			CommandID: "diff.commit.previous", Verb: "diff-previous",
			Keys:              opts.GetKeys(opts.Config.Universal.DiffWithPreviousCommit),
			Handler:           self.diffCommitWithPrevious,
			GetDisabledReason: self.require(func() *types.DisabledReason { return self.canHistoricalDiffCommit(true) }),
			Description:       self.c.Tr.DiffWithPreviousCommit,
		},
		{
			CommandID: "diff.commit.selected-commit", Verb: "diff-commit",
			Handler:           self.promptHistoricalCommitComparison,
			GetDisabledReason: self.require(func() *types.DisabledReason { return self.canHistoricalDiffCommit(false) }),
			Description:       self.c.Tr.DiffWithSelectedCommit, OpensMenu: true,
		},
	}
}

func (self *BasicCommitsController) historicalCommitSelection(previous bool) (*models.Commit, *types.DisabledReason) {
	selected, start, end := self.context.GetSelectedItems()
	if len(selected) != 1 || start != end || selected[0] == nil {
		return nil, &types.DisabledReason{Text: self.c.Tr.HistoricalDiffSingleCommitOnly}
	}
	commit := selected[0]
	if !isFullSelectedCommitOID(commit.Hash()) {
		return nil, &types.DisabledReason{Text: self.c.Tr.HistoricalDiffMissingSelection}
	}
	if previous && len(commit.Parents()) == 0 {
		return nil, &types.DisabledReason{Text: self.c.Tr.HistoricalDiffRootCommit}
	}
	return commit, nil
}

func (self *BasicCommitsController) canHistoricalDiffCommit(previous bool) *types.DisabledReason {
	_, reason := self.historicalCommitSelection(previous)
	return reason
}

func (self *BasicCommitsController) diffCommitWithWorktree() error {
	commit, reason := self.historicalCommitSelection(false)
	if reason != nil {
		return disabledReasonError(reason)
	}
	return runHistoricalRevisionDiff(self.c, commit.Hash(), false, []string{"."}, true)
}

func (self *BasicCommitsController) diffCommitWithPrevious() error {
	commit, reason := self.historicalCommitSelection(true)
	if reason != nil {
		return disabledReasonError(reason)
	}
	return runHistoricalRevisionDiff(self.c, commit.Hash(), true, []string{"."}, true)
}

func disabledReasonError(reason *types.DisabledReason) error {
	if reason == nil {
		return nil
	}
	return fmt.Errorf("%s", reason.Text)
}

func runHistoricalRevisionDiff(c *ControllerCommon, oid string, previous bool, paths []string, directory bool) error {
	resolved, err := c.Git().Diff.ResolveHistoricalDiffCommit(oid)
	if err != nil {
		return err
	}
	cmd, err := c.Git().Diff.HistoricalRevisionDiffCmdObj(resolved, previous, paths, directory)
	if err != nil {
		return err
	}
	_, runErr := c.RunSubprocess(cmd)
	c.Refresh(types.RefreshOptions{})
	return runErr
}

func (self *CommitFilesController) historicalDiffBindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			CommandID: "diff.history-file.worktree", Verb: "diff-worktree",
			Keys:              opts.GetKeys(opts.Config.Universal.DiffWithWorkingTree),
			Handler:           self.diffHistoricalFileWithWorktree,
			GetDisabledReason: self.require(func() *types.DisabledReason { return self.canHistoricalFileDiff(false, false) }),
			Description:       self.c.Tr.DiffWithWorkingTree, DisplayOnScreen: true,
		},
		{
			CommandID: "diff.history-file.previous", Verb: "diff-previous",
			Keys:              opts.GetKeys(opts.Config.Universal.DiffWithPreviousCommit),
			Handler:           self.diffHistoricalFileWithPrevious,
			GetDisabledReason: self.require(func() *types.DisabledReason { return self.canHistoricalFileDiff(true, false) }),
			Description:       self.c.Tr.DiffWithPreviousCommit,
		},
		{
			CommandID: "diff.history-file.working-file", Verb: "diff-working-file",
			Keys:              opts.GetKeys(opts.Config.Universal.DiffWithWorkingFile),
			Handler:           self.promptHistoricalFileWorkingDiff,
			GetDisabledReason: self.require(func() *types.DisabledReason { return self.canHistoricalFileDiff(false, true) }),
			Description:       self.c.Tr.DiffWithWorkingFile, OpensMenu: true,
		},
		{
			CommandID: "diff.history-file.selected-commit", Verb: "diff-commit",
			Handler:           self.promptHistoricalCommitComparison,
			GetDisabledReason: self.require(self.canHistoricalFileComparison),
			Description:       self.c.Tr.DiffWithSelectedCommit, OpensMenu: true,
		},
	}
}

func (self *CommitFilesController) captureHistoricalFileSelection() (*historicalDiffSelection, *types.DisabledReason) {
	if self.context().GetRefRange() != nil {
		return nil, &types.DisabledReason{Text: self.c.Tr.HistoricalDiffSingleCommitOnly}
	}
	commit, ok := self.context().GetRef().(*models.Commit)
	if !ok || commit == nil || !isFullSelectedCommitOID(commit.Hash()) {
		return nil, &types.DisabledReason{Text: self.c.Tr.HistoricalDiffMissingSelection}
	}
	nodes, start, end := self.context().GetSelectedItems()
	if len(nodes) != 1 || start != end || nodes[0] == nil {
		return nil, &types.DisabledReason{Text: self.c.Tr.HistoricalDiffFileOnly}
	}
	node := nodes[0]
	path := node.GetPath()
	if path == "" {
		return nil, &types.DisabledReason{Text: self.c.Tr.HistoricalDiffFileOnly}
	}
	selection := &historicalDiffSelection{
		oid: commit.Hash(), path: path, isFile: node.IsFile(), parents: append([]string(nil), commit.Parents()...),
	}
	if node.GetFile() != nil {
		selection.previousPath = node.GetFile().PreviousPath
		selection.deleted = node.GetFile().Deleted()
	}
	return selection, nil
}

func (self *CommitFilesController) canHistoricalFileDiff(previous, arbitrary bool) *types.DisabledReason {
	selection, reason := self.captureHistoricalFileSelection()
	if reason != nil {
		return reason
	}
	if previous && len(selection.parents) == 0 {
		return &types.DisabledReason{Text: self.c.Tr.HistoricalDiffRootCommit}
	}
	if arbitrary && !selection.isFile {
		return &types.DisabledReason{Text: self.c.Tr.HistoricalDiffFileOnly}
	}
	if !previous && selection.isFile && selection.deleted {
		return &types.DisabledReason{Text: self.c.Tr.HistoricalDiffDeletedFile}
	}
	return nil
}

func (self *CommitFilesController) canHistoricalFileComparison() *types.DisabledReason {
	_, reason := self.captureHistoricalFileSelection()
	return reason
}

func (self *CommitFilesController) diffHistoricalFileWithWorktree() error {
	selection, reason := self.captureHistoricalFileSelection()
	if reason != nil {
		return disabledReasonError(reason)
	}
	if selection.deleted {
		return fmt.Errorf("%s", self.c.Tr.HistoricalDiffDeletedFile)
	}
	if selection.isFile {
		return self.runHistoricalFileDiff(selection.oid, selection.path, selection.path)
	}
	return runHistoricalRevisionDiff(self.c, selection.oid, false, []string{selection.path}, true)
}

func (self *CommitFilesController) diffHistoricalFileWithPrevious() error {
	selection, reason := self.captureHistoricalFileSelection()
	if reason != nil {
		return disabledReasonError(reason)
	}
	if len(selection.parents) == 0 {
		return fmt.Errorf("%s", self.c.Tr.HistoricalDiffRootCommit)
	}
	paths := []string{selection.path}
	if selection.previousPath != "" && selection.previousPath != selection.path {
		paths = append(paths, selection.previousPath)
	}
	return runHistoricalRevisionDiff(self.c, selection.oid, true, paths, !selection.isFile)
}

func (self *CommitFilesController) promptHistoricalFileWorkingDiff() error {
	selection, reason := self.captureHistoricalFileSelection()
	if reason != nil {
		return disabledReasonError(reason)
	}
	if !selection.isFile {
		return fmt.Errorf("%s", self.c.Tr.HistoricalDiffFileOnly)
	}
	if selection.deleted {
		return fmt.Errorf("%s", self.c.Tr.HistoricalDiffDeletedFile)
	}
	title := fmt.Sprintf(self.c.Tr.HistoricalDiffWorkingFilePrompt, utils.ShortHash(selection.oid), selection.path)
	paths := self.c.Git().RepoPaths
	self.c.Prompt(types.PromptOpts{
		Title: title, InitialContent: selection.path, PreserveWhitespace: true,
		AllowEditSuggestion: true,
		FindSuggestionsFunc: historicalWorkingFileSuggestions(paths.WorktreePath(), paths.WorktreeGitDirPath(), paths.RepoGitDirPath()),
		HandleConfirm: func(destination string) error {
			prepared, err := self.c.Git().Diff.PrepareHistoricalFileDiff(selection.oid, selection.path, destination)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("%s: %w", self.c.Tr.HistoricalDiffMissingWorkingFile, err)
				}
				return err
			}
			return self.runPreparedHistoricalFileDiff(prepared)
		},
	})
	return nil
}

func (self *CommitFilesController) runHistoricalFileDiff(oid, path, destination string) error {
	prepared, err := self.c.Git().Diff.PrepareHistoricalFileDiff(oid, path, destination)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s: %w", self.c.Tr.HistoricalDiffMissingWorkingFile, err)
		}
		return err
	}
	return self.runPreparedHistoricalFileDiff(prepared)
}

func (self *CommitFilesController) runPreparedHistoricalFileDiff(prepared *git_commands.HistoricalFileDiff) (resultErr error) {
	defer func() {
		resultErr = errors.Join(resultErr, prepared.Close())
		self.c.Refresh(types.RefreshOptions{})
	}()
	_, resultErr = self.c.RunSubprocess(prepared.Command)
	return resultErr
}
