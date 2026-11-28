package gui

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
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

// Fork: forkAssignDigitKeys gives each item that has no key of its own the next
// free digit 1-9, so that pressing the digit runs it. Digits that another item
// uses are skipped, the Cancel item gets none, and disabled items still get one
// (pressing them shows the reason). Filtering menus take printable keys as text,
// and prefix menus reserve their keys, so neither gets digits.
func forkAssignDigitKeys(opts types.CreateMenuOptions) {
	if opts.FilterAsYouType || len(opts.BlockUnmatchedKeys) > 0 {
		return
	}

	isUsed := func(key gocui.Key, items []*types.MenuItem) bool {
		for _, item := range items {
			for _, k := range item.Keys {
				if k.Equals(key) {
					return true
				}
			}
			if item.Binding != nil {
				for _, k := range item.Binding.Keys {
					if k.Equals(key) {
						return true
					}
				}
			}
		}
		return false
	}

	var free []gocui.Key
	for r := '1'; r <= '9'; r++ {
		if key := gocui.NewKeyRune(r); !isUsed(key, opts.Items) {
			free = append(free, key)
		}
	}

	for _, item := range opts.Items {
		if len(free) == 0 {
			return
		}
		if item.CommandID == "menu.cancel" || len(item.KeyLabels()) > 0 {
			continue
		}
		// A binding-backed item with no key labels would run the digit without
		// showing it, so it is left alone. No menu that gets digits has one.
		if item.Binding != nil {
			continue
		}
		item.Keys = []gocui.Key{free[0]}
		free = free[1:]
	}
}
