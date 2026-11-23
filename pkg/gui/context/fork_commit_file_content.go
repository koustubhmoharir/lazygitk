package context

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
)

// Fork: in the full listing of a commit's files, the main view shows the content of a
// selected file rather than a diff of it (item 1 of "browse a commit's files"). This
// file decides what a selection shows; the controller renders it and the commit files
// context reports to the main view whether it is a diff.

// CommitFileShow is what the main view shows for the selected entry of the commit
// files panel.
type CommitFileShow int

const (
	// CommitFileShowDiff: the commit's changes under the entry, as the changed-files
	// listing shows them. That is every entry of the changed-files listing, and a
	// directory or a submodule of the full listing.
	CommitFileShowDiff CommitFileShow = iota
	// CommitFileShowContent: the file's content at the commit.
	CommitFileShowContent
	// CommitFileShowDeletedContent: the file's content before the commit, for a file
	// the commit deleted.
	CommitFileShowDeletedContent
)

// CommitFileShowFor decides what the main view shows for the selected entry. file is
// the entry's file, or nil for a directory. hasParent says whether the commit has a
// parent to show a deleted file from; without one there is nothing before the commit,
// so the entry shows its diff.
func CommitFileShowFor(fullListing bool, file *models.CommitFile, hasParent bool) CommitFileShow {
	if !fullListing || file == nil {
		return CommitFileShowDiff
	}
	if file.Submodule {
		return CommitFileShowDiff
	}
	if file.Deleted() {
		if hasParent {
			return CommitFileShowDeletedContent
		}
		return CommitFileShowDiff
	}
	return CommitFileShowContent
}

// showsFileContent reports whether the selection is shown as file content rather than as
// a diff.
func (self *CommitFilesContext) showsFileContent() bool {
	if self.GetRef() == nil {
		return false
	}
	from, _ := self.GetFromAndToForDiff()
	return CommitFileShowFor(self.IsFullListing(), selectedFile(self.GetSelected()), CommitFileHasParent(from)) != CommitFileShowDiff
}

// selectedFile is the file of a selected node, or nil for a directory or no selection.
func selectedFile(node *filetree.CommitFileNode) *models.CommitFile {
	if node == nil {
		return nil
	}
	return node.File
}

// CommitFileHasParent reports whether the parent named by from is a commit, rather than
// the empty tree that a root commit's parent is.
func CommitFileHasParent(from string) bool {
	return from != "" && from != models.EmptyTreeCommitHash
}
