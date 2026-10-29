package context

import "github.com/jesseduffield/lazygit/pkg/gui/types"

// Fork: the list views keep their horizontal scroll when a search selects a result.
// Upstream scrolls a view sideways to the match, which hides a list's hash and graph
// columns, and Escape leaves them hidden. Main and other non-list views keep that.
// See KeepOriginXOnSearch in pkg/gocui/view.go.
func (self *ContextTree) keepListViewsOriginXOnSearch() {
	for _, c := range self.Flatten() {
		if _, ok := c.(types.IListContext); !ok {
			continue
		}
		if view := c.GetView(); view != nil {
			view.KeepOriginXOnSearch = true
		}
	}
}
