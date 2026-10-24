package gui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/keymap"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

func TestForkDefaultPathsDoNotOfferUnrelatedSameKeyActions(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	bindings, _ := gui.GetInitialKeybindings()
	views := make(map[string]bool)
	for _, binding := range bindings {
		if binding.ViewName != "" && forkCommandView(binding.ViewName) {
			views[binding.ViewName] = true
		}
		if !forkCommandView(binding.ViewName) {
			continue
		}
		check := func(key gocui.Key) {
			if !forkMouseKey(key) && key.Mod()&gocui.ModCtrl != 0 {
				t.Errorf("%s in %s retains a core Ctrl default: %v", binding.CommandID, binding.ViewName, key)
			}
		}
		for _, key := range binding.Keys {
			check(key)
		}
		for _, path := range binding.CommandPaths {
			for _, key := range path {
				check(key)
			}
		}
	}
	for view := range views {
		var walk func(*keymap.CommandTree, []gocui.Key)
		walk = func(node *keymap.CommandTree, path []gocui.Key) {
			ids := make(map[string]bool)
			for _, binding := range node.Bindings {
				// Mode variants have explicit mutually exclusive live guards.
				if binding.CommandActive == nil {
					ids[binding.CommandID] = true
				}
			}
			if len(ids) > 1 {
				t.Errorf("%s: %s offers unrelated commands %v", view, types.CommandPathLabel(path), ids)
			}
			for _, edge := range node.Children {
				walk(edge.Tree, append(append([]gocui.Key(nil), path...), edge.Key))
			}
		}
		walk(keymap.BuildCommandTree(bindings, view), nil)
	}
}

func newForkKeymapTestGui(t *testing.T) *Gui {
	t.Helper()
	cfg := config.NewDummyAppConfig()
	cmn := common.NewDummyCommonWithUserConfigAndAppState(cfg.GetUserConfig(), cfg.GetAppState())
	gui, err := NewGui(cmn, cfg, &git_commands.GitVersion{Major: 2, Minor: 51}, nil, false, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	gui.GetCheatsheetKeybindings()
	return gui
}

func TestForkProjectionKeepsModeRoutesLiveWithoutReusingStageForPatch(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	id := "diff.stage-selection"
	original := &types.Binding{
		CommandID: "diff.primary-selection", ViewName: "main", Description: "Selection action",
		CommandIDFunc: func() string { return id },
		Keys:          []gocui.Key{gocui.NewKeyRune(' ')},
	}
	bindings, err := gui.projectForkBindings([]*types.Binding{original})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, bindings, 4)
	byID := map[string]*types.Binding{}
	for _, binding := range bindings {
		byID[binding.CommandID] = binding
	}
	assert.Equal(t, []string{"g h s"}, byID["diff.stage-selection"].KeyLabels())
	assert.Equal(t, []string{"g h t"}, byID["patch.toggle-selection"].KeyLabels())
	assert.Nil(t, byID["diff.stage-selection"].GetDisabledReason())
	assert.True(t, byID["patch.toggle-selection"].GetDisabledReason().Inapplicable)
	id = "patch.toggle-selection"
	assert.True(t, byID["diff.stage-selection"].GetDisabledReason().Inapplicable)
	assert.Nil(t, byID["patch.toggle-selection"].GetDisabledReason())
	assert.Equal(t, []gocui.Key{gocui.NewKeyRune(' ')}, original.Keys)
}

func TestForkProjectionContextControlsAndExplicitCommandOverride(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	gui.Config.GetUserConfig().CommandBindings = map[string]config.CommandBindingPaths{
		"diff.commit.worktree": {{"<f6>"}},
	}
	bindings, err := gui.projectForkBindings([]*types.Binding{
		{CommandID: "commit.create", ViewName: "files"},
		{CommandID: "commit.create", ViewName: "main"},
		{CommandID: "navigation.focus-diff", ViewName: "files", Tag: "global"},
		{CommandID: "file.enter", ViewName: "files"},
		{CommandID: "navigation.open-commits", ViewName: "localBranches"},
		{CommandID: "diff.commit.worktree", ViewName: "commits"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"<enter>"}, bindings[0].KeyLabels())
	assert.Empty(t, bindings[1].KeyLabels())
	assert.Equal(t, []string{"v"}, bindings[2].KeyLabels())
	assert.Empty(t, bindings[2].Tag)
	assert.Empty(t, bindings[3].KeyLabels(), "Files inspection must not invoke enterFile's submodule/dir side effects")
	assert.Equal(t, []string{"v", "<space>"}, bindings[4].KeyLabels())
	assert.Equal(t, []string{"<f6>"}, bindings[5].KeyLabels())
	assert.False(t, bindings[5].DefaultCommand)
}

func TestForkRefreshAliasSharesGlobalIdentityAndOverrides(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	for _, paths := range []config.CommandBindingPaths{nil, {{"<f6>"}}} {
		gui.Config.GetUserConfig().CommandBindings = map[string]config.CommandBindingPaths{"app.refresh": paths}
		bindings, err := gui.projectForkBindings([]*types.Binding{
			{CommandID: "app.refresh"},
			{CommandID: "file.refresh", ViewName: "files"},
		})
		if err != nil {
			t.Fatal(err)
		}
		assert.Len(t, bindings, 2)
		for _, binding := range bindings {
			assert.Equal(t, "app.refresh", binding.CommandID)
			assert.False(t, binding.DefaultCommand)
			if paths == nil {
				assert.Empty(t, binding.KeyLabels())
			} else {
				assert.Equal(t, []string{"<f6>"}, binding.KeyLabels())
			}
		}
	}
}

func TestForkSearchAliasesKeepHistoryLoadingHandlerFirst(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	history, generic := 0, 0
	for _, override := range []bool{false, true} {
		if override {
			gui.Config.GetUserConfig().CommandBindings = map[string]config.CommandBindingPaths{"search.open": {{"<f6>"}}}
		}
		bindings, err := gui.projectForkBindings([]*types.Binding{
			{CommandID: "commit.search", ViewName: "commits", Handler: func() error { history++; return nil }},
			{CommandID: "search.open", ViewName: "commits", Handler: func() error { generic++; return nil }},
		})
		if err != nil {
			t.Fatal(err)
		}
		active := forkApplicableBindings(bindings)
		assert.Len(t, active, 1)
		assert.Equal(t, "search.open", active[0].CommandID)
		if override {
			assert.Equal(t, []string{"<f6>"}, active[0].KeyLabels())
		} else {
			assert.Equal(t, []string{"/"}, active[0].KeyLabels())
		}
		assert.NoError(t, active[0].Handler())
	}
	assert.Equal(t, 2, history)
	assert.Zero(t, generic)
}

func TestForkProjectionStagedDiscardAliasKeepsImmediateUnstage(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	bindings, err := gui.projectForkBindings([]*types.Binding{{CommandID: "diff.discard-selection", ViewName: "secondary", CommandIDFunc: func() string { return "diff.unstage-selection" }}})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, b := range bindings {
		if b.CommandID == "diff.unstage-selection" {
			found = true
			assert.Equal(t, []string{"x"}, b.KeyLabels())
			assert.Nil(t, b.GetDisabledReason())
		} else {
			assert.True(t, b.GetDisabledReason().Inapplicable)
		}
	}
	assert.True(t, found)
}

func TestForkProjectionNamesInactiveActionsByTheirOwnVerb(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	bindings, err := gui.projectForkBindings([]*types.Binding{{CommandID: "diff.primary-selection", ViewName: "main", CommandIDFunc: func() string { return "diff.stage-selection" }, DescriptionFunc: func() string { return "Stage current mode" }, TooltipFunc: func() string { return "Current staging tooltip" }}})
	if err != nil {
		t.Fatal(err)
	}
	labels := map[string]string{}
	for _, b := range bindings {
		labels[b.CommandID] = b.GetDescription()
		if b.CommandID != "diff.stage-selection" {
			assert.NotEqual(t, "Current staging tooltip", b.GetTooltip())
		}
	}
	assert.Equal(t, gui.Tr.ForkUnstageSelection, labels["diff.unstage-selection"])
	assert.Equal(t, gui.Tr.ToggleSelectionForPatch, labels["patch.toggle-selection"])
	assert.Equal(t, gui.Tr.ForkRemovePatchSelection, labels["patch.remove-selection"])
}

func TestForkProjectionLegacyModeKeyStaysLiveWithSpecificOverride(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("keybinding:\n  universal:\n    select: '<f6>'\ncommandBindings:\n  patch.toggle-selection: [['g', 'p']]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := gui.Config.ReloadUserConfigForRepo([]*config.ConfigFile{{Path: path, Policy: config.ConfigFilePolicyErrorIfMissing}}); err != nil {
		t.Fatal(err)
	}
	gui.Common.SetUserConfig(gui.Config.GetUserConfig())
	id := "diff.stage-selection"
	bindings, err := gui.projectForkBindings([]*types.Binding{{CommandID: "diff.primary-selection", ViewName: "main", CommandIDFunc: func() string { return id }}})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]*types.Binding{}
	for _, b := range bindings {
		byID[b.CommandID] = b
	}
	assert.Equal(t, []string{"<f6>"}, byID["diff.stage-selection"].KeyLabels())
	assert.Equal(t, []string{"g p"}, byID["patch.toggle-selection"].KeyLabels())
	assert.Equal(t, []*types.Binding{byID["diff.stage-selection"]}, forkApplicableBindings(bindings))
	id = "patch.toggle-selection"
	assert.Equal(t, []*types.Binding{byID["patch.toggle-selection"]}, forkApplicableBindings(bindings))
	assert.False(t, byID["patch.toggle-selection"].DefaultCommand)
}
