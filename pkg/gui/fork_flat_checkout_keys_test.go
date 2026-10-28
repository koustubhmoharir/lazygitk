package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Fork: flat checkout keys under s (docs-master/fork/Tracker.md). These use the
// real initial bindings and the same label helpers as the UI.
func TestForkFlatCheckoutDefaultRoutes(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	bindings, _ := gui.GetInitialKeybindings()
	gui.setForkCommandBindings(bindings, nil)

	t.Run("checkout keys stay s c in every pane that has them", func(t *testing.T) {
		assert.Equal(t, "s c", gui.commandKeyLabel("branch.checkout", "localBranches"))
		assert.Equal(t, "s c", gui.commandKeyLabel("remote-branch.checkout", "remoteBranches"))
		assert.Equal(t, "s c", gui.commandKeyLabel("tag.checkout", "tags"))
		assert.Equal(t, "s c", gui.commandKeyLabel("commit.checkout-menu", "commits"))
		assert.Equal(t, "s c", gui.commandKeyLabel("commit.checkout-menu", "reflogCommits"))
		assert.Equal(t, "s c", gui.commandKeyLabel("commit.checkout-menu", "subCommits"))
	})

	t.Run("detached checkout is s d wherever s c is offered", func(t *testing.T) {
		for _, view := range []string{"localBranches", "remoteBranches", "tags", "commits", "reflogCommits", "subCommits"} {
			assert.Equal(t, "s d", gui.commandKeyLabel("commit.checkout-detached", view), view)
		}
	})

	t.Run("checkout by name has no key", func(t *testing.T) {
		for _, binding := range bindings {
			if binding.CommandID == "branch.checkout-by-name" {
				assert.Empty(t, binding.KeyLabels(), "view %q", binding.ViewName)
			}
		}
		assert.Equal(t, "", gui.commandKeyLabel("branch.checkout-by-name", "localBranches"))
	})

	t.Run("checkout previous keeps s p outside Status", func(t *testing.T) {
		assert.Equal(t, "s p", gui.commandKeyLabel("branch.checkout-previous", "commits"))
	})

	t.Run("repository keys are Status-only", func(t *testing.T) {
		assert.Equal(t, "s r", gui.commandKeyLabel("repo.switch-menu", "status"))
		assert.Equal(t, "s n", gui.commandKeyLabel("repo.open-path", "status"))
		assert.Equal(t, "", gui.commandKeyLabel("repo.switch-menu", "localBranches"))
		assert.Equal(t, "", gui.commandKeyLabel("repo.open-path", "localBranches"))
		assert.Equal(t, "", gui.commandKeyLabel("repo.switch-menu", ""))
		assert.Equal(t, "", gui.commandKeyLabel("repo.open-path", ""))
		for _, binding := range bindings {
			if binding.ViewName == "status" {
				continue
			}
			for _, label := range binding.KeyLabels() {
				if binding.CommandID == "repo.switch-menu" || binding.CommandID == "repo.open-path" {
					t.Errorf("%s in view %q is bound to %q", binding.CommandID, binding.ViewName, label)
				}
			}
		}
	})
}

func TestForkDetachedCheckoutBindingsCoverSelectionPanes(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	bindings, _ := gui.GetInitialKeybindings()
	found := map[string]bool{}
	for _, binding := range bindings {
		if binding.CommandID != "commit.checkout-detached" || binding.GetDisabledReason == nil {
			continue
		}
		found[binding.ViewName] = true
		assert.Equal(t, "checkout-detached", binding.GetVerb(), binding.ViewName)
		assert.Equal(t, "Checkout detached HEAD", binding.GetDescription(), binding.ViewName)
	}
	for _, view := range []string{"localBranches", "remoteBranches", "tags", "commits", "reflogCommits", "subCommits"} {
		assert.True(t, found[view], "no detached checkout binding in %s", view)
	}
}
