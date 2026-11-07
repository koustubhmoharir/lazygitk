package gui

import "github.com/jesseduffield/lazygit/pkg/gocui"

// Fork: the essential keys of a prefix menu. They were moved here out of
// menu_panel.go to ease upstream merges.

// Fork prefix menus use physical controls independently of ordinary menu
// aliases, leaving every offered letter accelerator available.
func forkPrefixEssentialKeys() []gocui.Key {
	return []gocui.Key{gocui.NewKeyName(gocui.KeyEnter), gocui.NewKeyName(gocui.KeyEsc), gocui.NewKeyName(gocui.KeyArrowUp), gocui.NewKeyName(gocui.KeyArrowDown)}
}
