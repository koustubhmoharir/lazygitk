package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

// Fork: roadmap item 22. These check the shortened routes against the real
// initial bindings, through the same label helpers the UI uses.
func TestForkShortKeysDefaultRoutes(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	bindings, _ := gui.GetInitialKeybindings()
	gui.setForkCommandBindings(bindings, nil)

	t.Run("checkout previous is one global s p binding", func(t *testing.T) {
		var found []*types.Binding
		for _, binding := range bindings {
			if binding.CommandID == "branch.checkout-previous" {
				found = append(found, binding)
			}
		}
		if assert.Len(t, found, 1) {
			assert.Equal(t, "", found[0].ViewName)
			assert.Equal(t, []string{"s p"}, found[0].KeyLabels())
		}
		assert.Equal(t, "s p", gui.commandKeyLabel("branch.checkout-previous", "localBranches"))
	})

	t.Run("checkout by name is s b in branches", func(t *testing.T) {
		assert.Equal(t, "s b", gui.commandKeyLabel("branch.checkout-by-name", "localBranches"))
	})

	t.Run("commit move keys are angle brackets", func(t *testing.T) {
		assert.Equal(t, "<", gui.commandKeyLabel("commit.move-up", "commits"))
		assert.Equal(t, ">", gui.commandKeyLabel("commit.move-down", "commits"))
	})

	t.Run("comma and period are unbound in commits and global", func(t *testing.T) {
		for _, binding := range bindings {
			if binding.ViewName != "" && binding.ViewName != "commits" {
				continue
			}
			for _, label := range binding.KeyLabels() {
				if label == "," || label == "." {
					t.Errorf("%s in view %q is bound to %q", binding.CommandID, binding.ViewName, label)
				}
			}
		}
	})

	t.Run("filter is slash in branches and files", func(t *testing.T) {
		assert.Equal(t, "/", gui.commandKeyLabel("filter.open", "localBranches"))
		assert.Equal(t, "/", gui.commandKeyLabel("filter.open", "files"))
	})

	t.Run("search is slash in commits", func(t *testing.T) {
		assert.Equal(t, "/", gui.commandKeyLabel("search.open", "commits"))
	})

	t.Run("old filter path t f t is gone", func(t *testing.T) {
		for _, binding := range bindings {
			for _, label := range binding.KeyLabels() {
				if label == "t f t" {
					t.Errorf("%s in view %q still uses t f t", binding.CommandID, binding.ViewName)
				}
			}
		}
	})

	t.Run("files pane splits v and 0 between inspect and focus-main", func(t *testing.T) {
		assert.Equal(t, "v", gui.commandKeyLabel("navigation.focus-diff", "files"))
		assert.Equal(t, "0", gui.commandKeyLabel("navigation.focus-main", "files"))
	})

	t.Run("branches focus-diff is 0", func(t *testing.T) {
		assert.Equal(t, "0", gui.commandKeyLabel("navigation.focus-diff", "localBranches"))
	})

	t.Run("worktrees open commits is v and space", func(t *testing.T) {
		var found []*types.Binding
		for _, binding := range bindings {
			if binding.CommandID == "navigation.open-commits" && binding.ViewName == "worktrees" {
				found = append(found, binding)
			}
		}
		if assert.Len(t, found, 1) {
			assert.ElementsMatch(t, []string{"v", "<space>"}, found[0].KeyLabels())
		}
	})

	t.Run("stash apply is g s a", func(t *testing.T) {
		assert.Equal(t, "g s a", gui.commandKeyLabel("stash.apply", "stash"))
	})
}
