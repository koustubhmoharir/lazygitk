package context

// Fork: SetFullListingMode switches the commit files panel between the changed
// files and the full listing (item 1 of "browse a commit's files"). The mode lives
// on the view model and survives ReInit, so refreshes keep the listing; only the
// title format depends on it here.
func (self *CommitFilesContext) SetFullListingMode(full bool) {
	self.CommitFileTreeViewModel.SetFullListing(full)
	if full {
		self.formatStr = self.allFilesTitleFormat
	} else {
		self.formatStr = self.changedTitleFormat
	}
	self.GetView().Title = self.Title()
}
