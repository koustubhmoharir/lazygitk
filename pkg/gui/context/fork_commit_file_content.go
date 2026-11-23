package context

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
)

// Fork: in the full listing of a commit's files, the main view shows the content of a
// selected file, a listing of a selected folder, or the commit a selected submodule
// records, rather than a diff (item 1 of "browse a commit's files"). This file decides
// what a selection shows; the controller renders it and the commit files context
// reports to the main view whether it is a diff.

// CommitFileShow is what the main view shows for the selected entry of the commit
// files panel.
type CommitFileShow int

const (
	// CommitFileShowDiff: the commit's changes under the entry, as the changed-files
	// listing shows them. That is every entry of the changed-files listing, and the
	// full listing when nothing is selected.
	CommitFileShowDiff CommitFileShow = iota
	// CommitFileShowContent: the file's content at the commit.
	CommitFileShowContent
	// CommitFileShowDeletedContent: the file's content before the commit, for a file
	// the commit deleted.
	CommitFileShowDeletedContent
	// CommitFileShowFolder: the entries of a folder at the commit, or of the root.
	CommitFileShowFolder
	// CommitFileShowSubmodule: the commit a submodule records at the commit, and the
	// one before it when the commit changed the submodule.
	CommitFileShowSubmodule
)

// CommitFileShowFor decides what the main view shows for the selected node. node is nil
// when nothing is selected. hasParent says whether the commit has a parent to show a
// deleted file from; without one there is nothing before the commit, so the entry shows
// its diff.
func CommitFileShowFor(fullListing bool, node *filetree.CommitFileNode, hasParent bool) CommitFileShow {
	if !fullListing || node == nil {
		return CommitFileShowDiff
	}
	if !node.IsFile() {
		return CommitFileShowFolder
	}
	file := node.File
	if file.Submodule {
		return CommitFileShowSubmodule
	}
	if file.Deleted() {
		if hasParent {
			return CommitFileShowDeletedContent
		}
		return CommitFileShowDiff
	}
	return CommitFileShowContent
}

// Show is what the main view shows for the selection, which is CommitFileShowDiff for
// the changed-files listing and when nothing is selected.
func (self *CommitFilesContext) Show() CommitFileShow {
	if self.GetRef() == nil {
		return CommitFileShowDiff
	}
	from, _ := self.GetFromAndToForDiff()
	return CommitFileShowFor(self.IsFullListing(), self.GetSelected(), CommitFileHasParent(from))
}

// ShowsFileContent reports whether the selection is shown as file content rather than as
// a diff.
func (self *CommitFilesContext) ShowsFileContent() bool {
	show := self.Show()
	return show == CommitFileShowContent || show == CommitFileShowDeletedContent
}

// ShowsNoDiff reports whether the main view shows something other than a diff of the
// selection: file content, a folder listing or a submodule.
func (self *CommitFilesContext) ShowsNoDiff() bool {
	return self.Show() != CommitFileShowDiff
}

// CommitFileHasParent reports whether the parent named by from is a commit, rather than
// the empty tree that a root commit's parent is.
func CommitFileHasParent(from string) bool {
	return from != "" && from != models.EmptyTreeCommitHash
}
