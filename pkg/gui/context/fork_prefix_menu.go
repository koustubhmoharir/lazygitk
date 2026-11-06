package context

import "github.com/jesseduffield/lazygit/pkg/gocui"

// Fork: beforeAccept runs before a menu item is accepted. When it returns false the
// menu stays open. Used by prefix menus (see MenuContext.OnMenuPress).
func (self *MenuViewModel) SetBeforeAccept(beforeAccept func() bool) {
	self.beforeAccept = beforeAccept
}

// Fork: keys that a prefix menu consumes without a match, so they cannot reach the
// commands behind the menu (see MenuContext.GetKeybindings).
func (self *MenuViewModel) SetBlockUnmatchedKeys(keys []gocui.Key) {
	self.blockUnmatchedKeys = append([]gocui.Key(nil), keys...)
}
