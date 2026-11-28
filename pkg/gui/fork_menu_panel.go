package gui

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// Fork: the essential keys of a prefix menu. They were moved here out of
// menu_panel.go to ease upstream merges.

// Fork prefix menus use physical controls independently of ordinary menu
// aliases, leaving every offered letter accelerator available.
func forkPrefixEssentialKeys() []gocui.Key {
	return []gocui.Key{gocui.NewKeyName(gocui.KeyEnter), gocui.NewKeyName(gocui.KeyEsc), gocui.NewKeyName(gocui.KeyArrowUp), gocui.NewKeyName(gocui.KeyArrowDown)}
}

// Fork: menu labels written for this fork end in "…" rather than "...", so a
// menu item that opens another menu must not get a second ellipsis.
func forkLabelEndsWithEllipsis(label string) bool {
	return strings.HasSuffix(label, "…") || strings.HasSuffix(label, "...")
}
