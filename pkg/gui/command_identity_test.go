package gui

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

var builtinCommandIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+)*$`)

func TestBuiltinCommandIdentityMetadata(t *testing.T) {
	platforms := []string{"linux", "darwin", "windows"}
	languages := []string{"en", "nl"}
	var baseline map[string]string

	for _, platform := range platforms {
		for _, language := range languages {
			t.Run(fmt.Sprintf("%s/%s", platform, language), func(t *testing.T) {
				bindings := builtinBindingsForTest(t, platform, language, false)
				assertFocusPanelBindings(t, bindings, config.GetDefaultConfigForPlatform(platform), false)
				identities := validateBuiltinBindingMetadata(t, bindings)
				reversedBindings := make([]*types.Binding, len(bindings))
				for index := range bindings {
					reversedBindings[len(bindings)-1-index] = bindings[index]
				}
				reversedIdentities := validateBuiltinBindingMetadata(t, reversedBindings)
				if !reflect.DeepEqual(reversedIdentities, identities) {
					t.Errorf("reversing binding order changed logical command metadata: %s", identityMapDifference(reversedIdentities, identities))
				}
				if baseline == nil {
					baseline = identities
				} else if !reflect.DeepEqual(identities, baseline) {
					t.Errorf("unique command ID to verb map differs by platform or locale\n got: %#v\nwant: %#v", identities, baseline)
				}
			})
		}
	}

	bindings := builtinBindingsForTest(t, "linux", "en", true)
	assertFocusPanelBindings(t, bindings, config.GetDefaultConfigForPlatform("linux"), true)
	identities := validateBuiltinBindingMetadata(t, bindings)
	if !reflect.DeepEqual(identities, baseline) {
		t.Errorf("removing configured keys changed logical command metadata: %s", identityMapDifference(identities, baseline))
	}

	var branchRenameFound bool
	for _, binding := range bindings {
		if binding.CommandID != "branch.rename" {
			continue
		}
		branchRenameFound = true
		if len(binding.Keys) != 0 {
			t.Errorf("unbound branch.rename still has keys: %v", binding.Keys)
		}
		if binding.Description == "" || binding.Handler == nil {
			t.Errorf("unbound branch.rename lost description or handler: description=%q handler nil=%t", binding.Description, binding.Handler == nil)
		}
	}
	if !branchRenameFound {
		t.Fatal("no branch.rename binding found after removing configured keys")
	}
}

func assertFocusPanelBindings(t *testing.T, bindings []*types.Binding, userConfig *config.UserConfig, keysCleared bool) {
	t.Helper()

	var focusBindings []*types.Binding
	for _, binding := range bindings {
		if binding.CommandID == "navigation.focus-panel" {
			focusBindings = append(focusBindings, binding)
		}
	}
	if want := len(userConfig.Gui.SidePanels); len(focusBindings) != want {
		t.Fatalf("navigation.focus-panel binding count = %d, want one per configured side panel (%d)", len(focusBindings), want)
	}

	for index, binding := range focusBindings {
		if binding.Handler == nil {
			t.Errorf("navigation.focus-panel factory %d lost its handler", index)
		}
		if binding.Description != "" {
			t.Errorf("navigation.focus-panel factory %d unexpectedly has description %q", index, binding.Description)
		}
		if keysCleared {
			if len(binding.Keys) != 0 {
				t.Errorf("cleared navigation.focus-panel factory %d has keys %v", index, binding.Keys)
			}
			continue
		}
		if index >= len(userConfig.Keybinding.Universal.JumpToBlock) {
			t.Errorf("navigation.focus-panel factory %d has no positional configured key", index)
			continue
		}
		wantKeys := config.GetValidatedKeyBindingKeys(userConfig.Keybinding.Universal.JumpToBlock[index])
		if !reflect.DeepEqual(binding.Keys, wantKeys) {
			t.Errorf("navigation.focus-panel factory %d keys = %v, want positional keys %v", index, binding.Keys, wantKeys)
		}
	}
}

func identityMapDifference(got, want map[string]string) string {
	var differences []string
	for id, verb := range want {
		if gotVerb, ok := got[id]; !ok {
			differences = append(differences, "missing "+id)
		} else if gotVerb != verb {
			differences = append(differences, fmt.Sprintf("%s: got %s, want %s", id, gotVerb, verb))
		}
	}
	for id := range got {
		if _, ok := want[id]; !ok {
			differences = append(differences, "unexpected "+id)
		}
	}
	sort.Strings(differences)
	return fmt.Sprint(differences)
}

func builtinBindingsForTest(t *testing.T, platform, language string, clearKeys bool) []*types.Binding {
	t.Helper()

	cfg := config.NewDummyAppConfig()
	userConfig := config.GetDefaultConfigForPlatform(platform)
	if clearKeys {
		clearConfiguredKeybindingLeaves(reflect.ValueOf(&userConfig.Keybinding))
	}
	*cfg.GetUserConfig() = *userConfig
	cmn := common.NewDummyCommonWithUserConfigAndAppState(cfg.GetUserConfig(), cfg.GetAppState())
	translations, err := i18n.NewTranslationSetFromConfig(utils.NewDummyLog(), language)
	if err != nil {
		t.Fatal(err)
	}
	cmn.Tr = translations

	gui, err := NewGui(cmn, cfg, &git_commands.GitVersion{Major: 2, Minor: 51}, nil, false, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return gui.GetCheatsheetKeybindings()
}

func clearConfiguredKeybindingLeaves(value reflect.Value) {
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if !value.IsNil() {
			clearConfiguredKeybindingLeaves(value.Elem())
		}
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		for index := range value.NumField() {
			clearConfiguredKeybindingLeaves(value.Field(index))
		}
	case reflect.Slice:
		if value.Type().Elem().Kind() == reflect.Struct {
			for index := range value.Len() {
				clearConfiguredKeybindingLeaves(value.Index(index))
			}
		} else if value.CanSet() {
			value.Set(reflect.Zero(value.Type()))
		}
	default:
		// Non-keybinding scalar fields in the config need no transformation.
	}
}

func validateBuiltinBindingMetadata(t *testing.T, bindings []*types.Binding) map[string]string {
	t.Helper()

	identities := make(map[string]string)
	for _, binding := range bindings {
		if binding.CommandID == "" {
			t.Errorf("binding %q has empty static CommandID", binding.Description)
			continue
		}
		if !builtinCommandIDPattern.MatchString(binding.CommandID) {
			t.Errorf("binding %q has invalid command ID %q", binding.Description, binding.CommandID)
		}
		if binding.Verb == "" {
			t.Errorf("binding %q (%s) has empty static Verb", binding.Description, binding.CommandID)
			continue
		}
		if previousVerb, ok := identities[binding.CommandID]; ok && previousVerb != binding.Verb {
			t.Errorf("aliases of %s have different verbs: %q and %q", binding.CommandID, previousVerb, binding.Verb)
		}
		identities[binding.CommandID] = binding.Verb
	}

	for _, expectedID := range []string{"branch.rename", "commit.reword", "stash.rename", "file.edit", "app.command-palette"} {
		if _, ok := identities[expectedID]; !ok {
			t.Errorf("expected built-in command ID %q was not returned", expectedID)
		}
	}

	return identities
}
