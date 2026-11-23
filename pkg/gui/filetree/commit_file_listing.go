package filetree

// Fork: the commit files tree shows either the changed files (the diff, as
// upstream does) or the full listing of the commit's tree (item 1 of "browse a
// commit's files"). Each listing keeps its own folder state, so collapsing in one
// does not change the other, and the full listing starts with its folders
// collapsed on every entry.

// IsFullListing reports whether the tree lists every file of the commit.
func (self *CommitFileTreeViewModel) IsFullListing() bool {
	return self.fullListing
}

// SetFullListing switches the listing. Entering the full listing starts it with
// a fresh folder state; the changed-files listing keeps its folder state, so
// entering it again restores what it had. The caller refreshes the files and then
// calls CollapseDirectoriesBelowRoot for a full-listing entry.
func (self *CommitFileTreeViewModel) SetFullListing(full bool) {
	self.fullListing = full
	if full {
		self.fullCollapsed = NewCollapsedPaths()
		self.ICommitFileTree.setCollapsedPaths(self.fullCollapsed)
	} else {
		self.ICommitFileTree.setCollapsedPaths(self.changedCollapsed)
	}
}

// CollapseDirectoriesBelowRoot collapses every directory except the root item, so
// that the top-level entries are visible. It acts on the tree as it is now, so it
// must run after the tree has been built from the listing's files.
func (self *CommitFileTreeViewModel) CollapseDirectoriesBelowRoot() {
	self.ICommitFileTree.collapseDirectoriesBelowRoot()
}

func (self *CommitFileTree) setCollapsedPaths(collapsedPaths *CollapsedPaths) {
	self.collapsedPaths = collapsedPaths
}

func (self *CommitFileTree) collapseDirectoriesBelowRoot() {
	for _, node := range self.GetAllItems() {
		// The root item is the directory "."; without it there is no such node.
		if !node.IsFile() && node.path != "." {
			self.collapsedPaths.Collapse(node.path)
		}
	}
}
