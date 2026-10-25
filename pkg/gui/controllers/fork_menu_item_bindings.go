package controllers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: flatMenuEntry is one action that the flat `y` copy menu offers directly,
// as well as through a nested menu. The menu item is found by command ID.
type flatMenuEntry struct {
	commandID   string
	description string
}

// Fork: flatMenuItemBindings turns the items of a nested menu into keybindings
// that behave exactly like choosing the item in that menu. build is called at
// press time and whenever the binding's disabled reason is read, so it must not
// run git commands or other I/O, and it must be safe with no selection.
func flatMenuItemBindings(build func() []*types.MenuItem, entries ...flatMenuEntry) []*types.Binding {
	bindings := make([]*types.Binding, 0, len(entries))
	for _, entry := range entries {
		bindings = append(bindings, &types.Binding{
			CommandID:   entry.commandID,
			Verb:        "copy",
			Description: entry.description,
			GetDisabledReason: func() *types.DisabledReason {
				item, err := findMenuItem(build(), entry.commandID)
				if err != nil {
					return &types.DisabledReason{Text: err.Error()}
				}
				return item.DisabledReason
			},
			Handler: func() error {
				item, err := findMenuItem(build(), entry.commandID)
				if err != nil {
					return err
				}
				if item.DisabledReason != nil {
					return errors.New(item.DisabledReason.Text)
				}
				return item.OnPress()
			},
		})
	}
	return bindings
}

// Fork: findMenuItem returns the item with the given command ID. A missing item
// is a programming error, so it is reported rather than ignored.
func findMenuItem(items []*types.MenuItem, commandID string) (*types.MenuItem, error) {
	for _, item := range items {
		if item.CommandID == commandID {
			return item, nil
		}
	}
	return nil, fmt.Errorf("no copy menu item has command ID %q", commandID)
}
