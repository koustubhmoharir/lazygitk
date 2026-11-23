package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// Fork: commands in the full listing of a commit's files (item 1 of "browse a
// commit's files"). The full listing also holds the files the commit did not change,
// so the commands that need a changed file are crossed out on an unchanged file, and
// act only on the changed files under a folder. The changed-files listing is never
// affected: every file in it is changed.

// changedFilesReason is the disabled reason for a command that needs the changed files
// under the given nodes, or nil when there are some. The reason names a file when every
// node is one, and a folder otherwise.
func (self *CommitFilesController) changedFilesReason(nodes []*filetree.CommitFileNode) *types.DisabledReason {
	if !self.context().IsFullListing() || len(self.changedFilesUnder(nodes)) > 0 {
		return nil
	}
	if lo.EveryBy(nodes, func(node *filetree.CommitFileNode) bool { return node.IsFile() }) {
		return &types.DisabledReason{Text: self.c.Tr.CommitFileUnchanged}
	}
	return &types.DisabledReason{Text: self.c.Tr.CommitFolderUnchanged}
}

// changedFileReason is changedFilesReason for a single node.
func (self *CommitFilesController) changedFileReason(node *filetree.CommitFileNode) *types.DisabledReason {
	return self.changedFilesReason([]*filetree.CommitFileNode{node})
}

// changedRootReason is changedFilesReason for the root item, which holds the whole commit.
func (self *CommitFilesController) changedRootReason() *types.DisabledReason {
	return self.changedFilesReason([]*filetree.CommitFileNode{self.context().GetRoot()})
}

// changedFilesUnder returns the changed files under the given nodes, each once. The
// nodes should not overlap, as normalisedSelectedCommitFileNodes ensures.
func (self *CommitFilesController) changedFilesUnder(nodes []*filetree.CommitFileNode) []*models.CommitFile {
	rawNodes := lo.Map(nodes, func(node *filetree.CommitFileNode, _ int) *filetree.Node[models.CommitFile] {
		return node.Raw()
	})
	return self.context().ChangedFilesUnder(rawNodes...)
}

// fullListingDiffPaths is diffPathsForNode for the full listing: only the changed files
// take part in the diff, so no unchanged path is passed to git. The result is empty
// only for a folder with no changed file in view under a text filter; the caller
// shows an empty diff for that, never the whole commit.
func (self *CommitFilesController) fullListingDiffPaths(node *filetree.CommitFileNode) []string {
	changed := lo.Filter(self.c.Model().CommitFiles, func(file *models.CommitFile, _ int) bool {
		return self.context().IsChangedFile(file)
	})
	paths := diffPathsForNode(node.Raw(), self.context().GetRoot().Raw(), changed, self.context().IsFiltering())
	if self.context().IsFiltering() {
		// The filtered tree still holds the unchanged files, which diffPathsForNode
		// names too; keep only the names of changed files.
		changedNames := make(map[string]bool)
		for _, file := range changed {
			for _, name := range file.Names() {
				changedNames[name] = true
			}
		}
		paths = lo.Filter(paths, func(path string, _ int) bool { return changedNames[path] })
	}
	return paths
}
