package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func requireCommandBindingNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCommandBindingsYAMLAndValidation(t *testing.T) {
	config := *GetDefaultConfig()
	requireCommandBindingNoError(t, yaml.Unmarshal([]byte("commandBindings:\n  diff.commit.worktree: [[d, w], '<f6>']\n  disabled: []\n  nullDisabled: null\n"), &config))
	assert.Equal(t, CommandBindingPaths{Keybinding{"d", "w"}, Keybinding{"<f6>"}}, config.CommandBindings["diff.commit.worktree"])
	assert.Equal(t, CommandBindingPaths{}, config.CommandBindings["disabled"])
	assert.Nil(t, config.CommandBindings["nullDisabled"])
	requireCommandBindingNoError(t, config.Validate())

	for _, value := range []string{
		"commandBindings: { ' ': [x] }",
		"commandBindings: { ' x ': [x] }",
		"commandBindings: { x: [[]] }",
		"commandBindings: { x: [[a, b, c, d]] }",
		"commandBindings: { x: [[<disabled>]] }",
		"commandBindings: { x: [[not-a-key]] }",
	} {
		invalid := *GetDefaultConfig()
		if err := yaml.Unmarshal([]byte(value), &invalid); err != nil {
			t.Fatalf("unmarshal %q: %v", value, err)
		}
		assert.Error(t, invalid.Validate(), value)
	}
}

func TestCommandBindingsLayeringAndLegacyCompatibility(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.yml")
	second := filepath.Join(dir, "second.yml")
	third := filepath.Join(dir, "third.yml")
	requireCommandBindingNoError(t, os.WriteFile(first, []byte("commandBindings:\n  shared: [[a, b], c]\n  retained: [x]\nkeybinding:\n  universal:\n    quit: q\ncustomCommands:\n  - key: '<c-t>'\n    command: echo legacy\n"), 0o600))
	requireCommandBindingNoError(t, os.WriteFile(second, []byte("commandBindings:\n  shared: [[d, e]]\n"), 0o600))
	requireCommandBindingNoError(t, os.WriteFile(third, []byte("commandBindings:\n  shared: []\n  retained: null\n"), 0o600))
	config, err := loadUserConfig([]*ConfigFile{{Path: first, Policy: ConfigFilePolicyErrorIfMissing}, {Path: second, Policy: ConfigFilePolicyErrorIfMissing}, {Path: third, Policy: ConfigFilePolicyErrorIfMissing}}, GetDefaultConfig(), false)
	requireCommandBindingNoError(t, err)
	assert.Equal(t, CommandBindingPaths{}, config.CommandBindings["shared"])
	assert.Nil(t, config.CommandBindings["retained"])
	assert.Contains(t, config.Keybinding.Universal.Quit, "q")
	assert.Contains(t, config.CustomCommands, CustomCommand{Key: Keybinding{"<c-t>"}, Command: "echo legacy"})
}

func TestCommandBindingsRoundTrip(t *testing.T) {
	original := UserConfig{CommandBindings: map[string]CommandBindingPaths{
		"diff.commit.worktree": {Keybinding{"d", "w"}, Keybinding{"<f6>"}}, "disabled": {},
	}}
	data, err := yaml.Marshal(&original)
	requireCommandBindingNoError(t, err)
	var decoded UserConfig
	requireCommandBindingNoError(t, yaml.Unmarshal(data, &decoded))
	assert.Equal(t, original.CommandBindings, decoded.CommandBindings)

	jsonData, err := json.Marshal(original.CommandBindings)
	requireCommandBindingNoError(t, err)
	var jsonDecoded map[string]CommandBindingPaths
	requireCommandBindingNoError(t, json.Unmarshal(jsonData, &jsonDecoded))
	assert.Equal(t, original.CommandBindings, jsonDecoded)
	assert.Contains(t, string(jsonData), `"diff.commit.worktree":[["d","w"],"\u003cf6\u003e"]`)
}
