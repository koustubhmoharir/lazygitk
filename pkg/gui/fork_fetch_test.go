package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

func TestForkFetchIsGlobalAndRemotesKeepsSelectedRemote(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	bindings, err := gui.projectForkBindings([]*types.Binding{
		{CommandID: "repo.fetch", Verb: "fetch", Keys: nil},
		{CommandID: "remote.fetch", ViewName: "remotes", Verb: "fetch"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "", bindings[0].ViewName)
	assert.Equal(t, []string{"f"}, bindings[0].KeyLabels())
	assert.Equal(t, "remotes", bindings[1].ViewName)
	assert.Equal(t, []string{"f"}, bindings[1].KeyLabels())
}

func TestForkFetchDefaultProjectionOwnsFOnlyInRemotes(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	bindings, _ := gui.GetInitialKeybindings()
	gui.setForkCommandBindings(bindings, nil)

	var global, remotes []*types.Binding
	for _, binding := range bindings {
		switch binding.CommandID {
		case "repo.fetch":
			global = append(global, binding)
		case "remote.fetch":
			remotes = append(remotes, binding)
		}
	}
	assert.Len(t, global, 1, "palette should list one repository fetch")
	if assert.Len(t, global, 1) {
		assert.Equal(t, "", global[0].ViewName)
		assert.Equal(t, []string{"f"}, global[0].KeyLabels())
	}
	if assert.NotEmpty(t, remotes) {
		for _, binding := range remotes {
			assert.Equal(t, "remotes", binding.ViewName)
		}
	}
	assert.Equal(t, "f", gui.commandKeyLabel("repo.fetch", "branches"))
	assert.Equal(t, "f", gui.commandKeyLabel("remote.fetch", "remotes"))

	// No other binding may claim a bare f on a normal pane.
	for _, binding := range bindings {
		if binding.CommandID == "repo.fetch" || binding.CommandID == "remote.fetch" {
			continue
		}
		if binding.ViewName != "" && !forkCommandView(binding.ViewName) {
			continue
		}
		for _, label := range binding.KeyLabels() {
			if label == "f" {
				t.Errorf("%s in view %q also binds f", binding.CommandID, binding.ViewName)
			}
		}
	}
}
