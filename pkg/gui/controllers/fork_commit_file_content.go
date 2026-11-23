package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: in the full listing of a commit's files, a selected file shows its content at the
// commit (or before it, for a file the commit deleted) rather than a diff. Binary files
// show a note instead of their bytes, which the render decides (see types.FileContentTask).
// The decision of what to show is in context.CommitFileShowFor; this file renders it.

// renderFullListingFileContent renders the selected node as file content, and reports
// whether it did. It reports false for anything shown as a diff.
func (self *CommitFilesController) renderFullListingFileContent(node *filetree.CommitFileNode) bool {
	from, to := self.context().GetFromAndToForDiff()
	var file *models.CommitFile
	if node != nil {
		file = node.File
	}
	show := context.CommitFileShowFor(self.context().IsFullListing(), file, context.CommitFileHasParent(from))

	var rev, title string
	switch show {
	case context.CommitFileShowContent:
		rev = to
		title = self.c.Tr.FileContentTitle
	case context.CommitFileShowDeletedContent:
		rev = from
		title = self.c.Tr.FileContentDeletedTitle
	default:
		return false
	}

	cmdObj := self.c.Git().Commit.ShowFileContentCmdObj(rev, node.GetPath())

	self.c.RenderToMainViews(types.RefreshMainOpts{
		Pair: self.c.MainViewPairs().Normal,
		Main: &types.ViewUpdateOpts{
			Title: title,
			Task:  types.NewFileContentTask(cmdObj.GetCmd()),
		},
		Secondary: secondaryPatchPanelUpdateOpts(self.c),
	})
	return true
}
