package controllers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type historicalCommitComparisonTarget struct {
	c          *ControllerCommon
	git        *commands.GitCommand
	generation int
	left       git_commands.HistoricalDiffCommit
	paths      []string
	directory  bool
	selected   string
}

func (target *historicalCommitComparisonTarget) checkRepository() error {
	if target.c.Git() != target.git || target.c.State().GetRepoGeneration() != target.generation {
		return errors.New(target.c.Tr.BranchSyncRepositoryChanged)
	}
	return nil
}

func newHistoricalCommitComparisonTarget(c *ControllerCommon, oid string, paths []string, directory bool) (*historicalCommitComparisonTarget, error) {
	git := c.Git()
	generation := c.State().GetRepoGeneration()
	left, err := git.Diff.ResolveHistoricalComparisonCommit(oid)
	if err != nil {
		return nil, err
	}
	target := &historicalCommitComparisonTarget{
		c: c, git: git, generation: generation, left: left,
		paths: append([]string(nil), paths...), directory: directory, selected: oid,
	}
	if err := target.checkRepository(); err != nil {
		return nil, err
	}
	return target, nil
}

func (target *historicalCommitComparisonTarget) prompt() {
	target.c.Prompt(types.PromptOpts{
		Title:          fmt.Sprintf(target.c.Tr.HistoricalDiffCommitPrompt, target.selected),
		InitialContent: "", PreserveWhitespace: true,
		HandleClose: target.checkRepository,
		HandleConfirm: func(input string) error {
			if err := target.checkRepository(); err != nil {
				return err
			}
			right, err := target.git.Diff.ResolveHistoricalComparisonCommit(input)
			if err != nil {
				return err
			}
			cmd, err := target.git.Diff.HistoricalCommitComparisonCmdObj(target.left, right, target.paths, target.directory)
			if err != nil {
				return err
			}
			_, runErr := target.c.RunSubprocess(cmd)
			if target.c.Git() == target.git && target.c.State().GetRepoGeneration() == target.generation {
				target.c.Refresh(types.RefreshOptions{})
			}
			return runErr
		},
	})
}

func (self *BasicCommitsController) promptHistoricalCommitComparison() error {
	commit, reason := self.historicalCommitSelection(false)
	if reason != nil {
		return disabledReasonError(reason)
	}
	target, err := newHistoricalCommitComparisonTarget(self.c, commit.Hash(), []string{"."}, true)
	if err != nil {
		return err
	}
	target.prompt()
	return nil
}

func (self *CommitFilesController) promptHistoricalCommitComparison() error {
	selection, reason := self.captureHistoricalFileSelection()
	if reason != nil {
		return disabledReasonError(reason)
	}
	paths := []string{selection.path}
	if selection.previousPath != "" && selection.previousPath != selection.path {
		paths = append(paths, selection.previousPath)
	}
	target, err := newHistoricalCommitComparisonTarget(self.c, selection.oid, paths, !selection.isFile)
	if err != nil {
		return err
	}
	target.prompt()
	return nil
}
