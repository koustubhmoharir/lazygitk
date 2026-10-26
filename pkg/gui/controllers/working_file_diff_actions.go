package controllers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// Fork: Files-panel external-difftool comparisons of the selected working-tree
// file, or of a directory as a directory diff: d w (HEAD), d s (index) and d c
// (a pasted commit). The left side is the base and the right side the working
// file. They reuse the historical difftool builders, so Git's literal-pathspec
// and tracked-file rules apply. Sticky diff mode is deliberately not consulted.

func (self *FilesController) workingFileDiffBindings() []*types.Binding {
	return []*types.Binding{
		{
			CommandID: "diff.working-file.head", Verb: "diff-head",
			Handler:           self.withItem(self.diffWorkingFileWithHead),
			GetDisabledReason: self.require(self.singleItemSelected(self.workingFileHeadDisabledReason)),
			Description:       self.c.Tr.DiffWorkingFileWithHead,
		},
		{
			CommandID: "diff.working-file.staged", Verb: "diff-staged",
			Handler:           self.withItem(self.diffWorkingFileWithStaged),
			GetDisabledReason: self.require(self.singleItemSelected(self.workingFileStagedDisabledReason)),
			Description:       self.c.Tr.DiffWorkingFileWithStaged,
		},
		{
			CommandID: "diff.working-file.selected-commit", Verb: "diff-commit",
			Handler:           self.withItem(self.promptWorkingFileCommitDiff),
			GetDisabledReason: self.require(self.singleItemSelected(self.workingFileCommitDisabledReason)),
			Description:       self.c.Tr.DiffWorkingFileWithCommit, OpensMenu: true,
		},
	}
}

// untrackedWorkingFileDisabledReason rejects an untracked file, which has no
// base version for any of the comparisons. Directories are allowed; Git leaves
// their untracked files out of the comparison.
func (self *FilesController) untrackedWorkingFileDisabledReason(node *filetree.FileNode) *types.DisabledReason {
	if node.GetIsFile() && !node.GetIsTracked() {
		return &types.DisabledReason{Text: self.c.Tr.WorkingFileUntracked}
	}
	return nil
}

func (self *FilesController) workingFileHeadDisabledReason(node *filetree.FileNode) *types.DisabledReason {
	if reason := self.untrackedWorkingFileDisabledReason(node); reason != nil {
		return reason
	}
	// The model's current branch has no commit when HEAD is unborn.
	if head, found := lo.Find(self.c.Model().Branches, func(branch *models.Branch) bool { return branch.Head }); found && head.CommitHash == "" {
		return &types.DisabledReason{Text: self.c.Tr.WorkingFileUnbornHead}
	}
	return nil
}

func (self *FilesController) workingFileStagedDisabledReason(node *filetree.FileNode) *types.DisabledReason {
	if reason := self.untrackedWorkingFileDisabledReason(node); reason != nil {
		return reason
	}
	if node.GetIsFile() && !node.GetHasUnstagedChanges() {
		return &types.DisabledReason{Text: self.c.Tr.WorkingFileNoUnstagedChanges}
	}
	return nil
}

func (self *FilesController) workingFileCommitDisabledReason(node *filetree.FileNode) *types.DisabledReason {
	return self.untrackedWorkingFileDisabledReason(node)
}

// workingFileDiffPaths returns the pathspec for a comparison with the working
// tree. Like the Files diffs, it includes the old path of a staged rename and,
// for a directory, the outside ends of renames that cross its boundary.
func (self *FilesController) workingFileDiffPaths(node *filetree.FileNode) ([]string, error) {
	paths := self.pathsForDiff(node)
	if len(paths) == 0 {
		// An empty pathspec would make Git compare the whole repository.
		return nil, errors.New(self.c.Tr.WorkingFileNoPathsMatchFilter)
	}
	return paths, nil
}

// diffWorkingFileWithHead compares HEAD (left) with the working tree (right).
func (self *FilesController) diffWorkingFileWithHead(node *filetree.FileNode) error {
	if reason := self.workingFileHeadDisabledReason(node); reason != nil {
		return disabledReasonError(reason)
	}
	head, err := self.c.Git().Diff.ResolveHeadDiffCommit()
	if err != nil {
		return err
	}
	paths, err := self.workingFileDiffPaths(node)
	if err != nil {
		return err
	}
	return runHistoricalRevisionDiff(self.c, head.OID, false, paths, !node.GetIsFile())
}

// diffWorkingFileWithStaged compares the index (left) with the working tree
// (right). It uses the plain difftool builder with no commit endpoints and does
// not read sticky diff mode.
func (self *FilesController) diffWorkingFileWithStaged(node *filetree.FileNode) error {
	if reason := self.workingFileStagedDisabledReason(node); reason != nil {
		return disabledReasonError(reason)
	}
	return self.c.RunSubprocessAndRefresh(
		self.c.Git().Diff.OpenDiffToolCmdObj(git_commands.DiffToolCmdOptions{
			Filepath:    node.GetPath(),
			IsDirectory: !node.GetIsFile(),
		}),
	)
}

// promptWorkingFileCommitDiff asks for a commit hash and compares it (left)
// with the working tree (right). The selection is captured before the prompt.
func (self *FilesController) promptWorkingFileCommitDiff(node *filetree.FileNode) error {
	if reason := self.workingFileCommitDisabledReason(node); reason != nil {
		return disabledReasonError(reason)
	}
	paths, err := self.workingFileDiffPaths(node)
	if err != nil {
		return err
	}
	directory := !node.GetIsFile()
	git := self.c.Git()
	generation := self.c.State().GetRepoGeneration()
	promptPastedHistoricalCommit(self.c, git, generation, self.c.Tr.WorkingFileCommitPrompt,
		func(left git_commands.HistoricalDiffCommit) error {
			return runHistoricalRevisionDiff(self.c, left.OID, false, paths, directory)
		})
	return nil
}
