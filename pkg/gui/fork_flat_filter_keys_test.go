package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Fork: flat log filter keys. These use the real initial bindings and the same
// label helpers as the UI.
func TestForkFlatFilterDefaultRoutes(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	bindings, _ := gui.GetInitialKeybindings()
	gui.setForkCommandBindings(bindings, nil)

	t.Run("path, author and clear are w p, w u and w c from any pane", func(t *testing.T) {
		for _, view := range []string{"files", "commits", "localBranches", "status"} {
			assert.Equal(t, "w p", gui.commandKeyLabel("filter.by-path", view), view)
			assert.Equal(t, "w u", gui.commandKeyLabel("filter.by-author", view), view)
			assert.Equal(t, "w c", gui.commandKeyLabel("filter.clear", view), view)
		}
	})

	t.Run("filtering menu has no key", func(t *testing.T) {
		for _, binding := range bindings {
			if binding.CommandID == "filter.options-menu" {
				assert.Empty(t, binding.KeyLabels(), "view %q", binding.ViewName)
			}
		}
		assert.Equal(t, "", gui.commandKeyLabel("filter.options-menu", "commits"))
	})
}
