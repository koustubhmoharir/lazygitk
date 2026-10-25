package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const forkKeyConfigSample = `keybinding:
  universal:
    quit: x
commandBindings:
  app.refresh: ['<f6>']
customCommands:
  - key: Z
    command: echo hello
    context: global
`

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func writeForkKeyConfigFile(t *testing.T, name string, content string) *ConfigFile {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	requireNoError(t, os.WriteFile(path, []byte(content), 0o600))
	return &ConfigFile{Path: path, Policy: ConfigFilePolicyErrorIfMissing}
}

func TestForkIgnoreKeyConfigDiscardsKeySettingsByDefault(t *testing.T) {
	t.Setenv(HonourKeyConfigEnvVar, "")
	file := writeForkKeyConfigFile(t, "config.yml", forkKeyConfigSample)

	loaded, err := loadUserConfigWithDefaults([]*ConfigFile{file}, false)
	requireNoError(t, err)

	// Compare with a load of an empty file, which is the fork default after the
	// loader merges legacy alternate keys.
	empty := writeForkKeyConfigFile(t, "empty.yml", "")
	defaults, err := loadUserConfigWithDefaults([]*ConfigFile{empty}, false)
	requireNoError(t, err)
	assert.Equal(t, defaults.Keybinding.Universal.Quit, loaded.Keybinding.Universal.Quit)
	assert.Empty(t, loaded.CommandBindings)
	if !assert.Len(t, loaded.CustomCommands, 1) {
		t.FailNow()
	}
	assert.Equal(t, "echo hello", loaded.CustomCommands[0].Command)
	assert.Empty(t, loaded.CustomCommands[0].Key)
	assert.Equal(t, []IgnoredKeySetting{{
		Path:  file.Path,
		Kinds: []string{"keybinding", "commandBindings", "customCommands key"},
	}}, loaded.IgnoredKeySettings())
	assert.False(t, loaded.KeybindingIsExplicit("universal.quit"))
}

func TestForkIgnoreKeyConfigHonouredWithSwitch(t *testing.T) {
	t.Setenv(HonourKeyConfigEnvVar, "1")
	file := writeForkKeyConfigFile(t, "config.yml", forkKeyConfigSample)

	loaded, err := loadUserConfigWithDefaults([]*ConfigFile{file}, false)
	requireNoError(t, err)

	// The legacy quit field also keeps its ctrl+c alternate.
	assert.Equal(t, Keybinding{"x", "<ctrl+c>"}, loaded.Keybinding.Universal.Quit)
	assert.Equal(t, map[string]CommandBindingPaths{"app.refresh": {{"<f6>"}}}, loaded.CommandBindings)
	if !assert.Len(t, loaded.CustomCommands, 1) {
		t.FailNow()
	}
	assert.Equal(t, Keybinding{"Z"}, loaded.CustomCommands[0].Key)
	assert.Empty(t, loaded.IgnoredKeySettings())
	assert.True(t, loaded.KeybindingIsExplicit("universal.quit"))
}

func TestForkIgnoreKeyConfigNoKeySettingsHasNoNotice(t *testing.T) {
	t.Setenv(HonourKeyConfigEnvVar, "")
	file := writeForkKeyConfigFile(t, "config.yml", "gui:\n  scrollHeight: 5\ncustomCommands:\n  - key: ''\n    command: echo hello\n    context: global\n")

	loaded, err := loadUserConfigWithDefaults([]*ConfigFile{file}, false)
	requireNoError(t, err)

	assert.Equal(t, 5, loaded.Gui.ScrollHeight)
	assert.Empty(t, loaded.IgnoredKeySettings())
}

func TestForkIgnoreKeyConfigKindsFromYAML(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{name: "empty document", content: "", want: nil},
		{name: "empty keybinding mapping", content: "keybinding: {}\n", want: nil},
		{name: "null keybinding", content: "keybinding:\n", want: nil},
		{name: "empty commandBindings mapping", content: "commandBindings: {}\n", want: nil},
		{name: "empty key values", content: "customCommands:\n  - key: ''\n    command: a\n  - key: []\n    command: b\n", want: nil},
		{name: "disabled key only", content: "customCommands:\n  - key: '<disabled>'\n    command: a\n", want: nil},
		{name: "keybinding entry", content: "keybinding:\n  universal:\n    quit: x\n", want: []string{"keybinding"}},
		{name: "commandBindings entry", content: "commandBindings:\n  app.refresh: [r]\n", want: []string{"commandBindings"}},
		{name: "sequence key", content: "customCommands:\n  - key: [a, b]\n    command: a\n", want: []string{"customCommands key"}},
		{
			name:    "key in nested command menu",
			content: "customCommands:\n  - commandMenu:\n      - key: Z\n        command: a\n",
			want:    []string{"customCommands key"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kinds, err := ignoredKeyKinds([]byte(test.content))
			requireNoError(t, err)
			assert.Equal(t, test.want, kinds)
		})
	}
}

func TestForkIgnoreKeyConfigClearsNestedCommandMenuKeys(t *testing.T) {
	t.Setenv(HonourKeyConfigEnvVar, "")
	file := writeForkKeyConfigFile(t, "config.yml", "customCommands:\n  - key: M\n    commandMenu:\n      - key: Z\n        command: a\n        context: global\n")

	loaded, err := loadUserConfigWithDefaults([]*ConfigFile{file}, false)
	requireNoError(t, err)

	if !assert.Len(t, loaded.CustomCommands, 1) {
		t.FailNow()
	}
	assert.Empty(t, loaded.CustomCommands[0].Key)
	if !assert.Len(t, loaded.CustomCommands[0].CommandMenu, 1) {
		t.FailNow()
	}
	assert.Empty(t, loaded.CustomCommands[0].CommandMenu[0].Key)
	assert.Equal(t, []IgnoredKeySetting{{Path: file.Path, Kinds: []string{"customCommands key"}}}, loaded.IgnoredKeySettings())
}

func TestForkIgnoreKeyConfigPerRepoLoadStartsFromBase(t *testing.T) {
	t.Setenv(HonourKeyConfigEnvVar, "")
	global := writeForkKeyConfigFile(t, "global.yml", "keybinding:\n  universal:\n    quit: x\ngui:\n  scrollHeight: 7\n")
	repo := writeForkKeyConfigFile(t, "repo.yml", forkKeyConfigSample)

	// The base carries a value that is not the platform default, so the test
	// shows that discarding restores the base the loader started from.
	base := GetDefaultConfig()
	base.Keybinding.Universal.Quit = Keybinding{"<f12>"}
	loaded, err := loadUserConfig([]*ConfigFile{global, repo}, base, true)
	requireNoError(t, err)

	assert.Equal(t, Keybinding{"<f12>", "<ctrl+c>"}, loaded.Keybinding.Universal.Quit)
	assert.Equal(t, 7, loaded.Gui.ScrollHeight)
	assert.Empty(t, loaded.CommandBindings)
	if !assert.Len(t, loaded.CustomCommands, 1) {
		t.FailNow()
	}
	assert.Empty(t, loaded.CustomCommands[0].Key)
	assert.Equal(t, []IgnoredKeySetting{
		{Path: global.Path, Kinds: []string{"keybinding"}},
		{Path: repo.Path, Kinds: []string{"keybinding", "commandBindings", "customCommands key"}},
	}, loaded.IgnoredKeySettings())
}

func TestForkIgnoreKeyConfigSecondLoadDoesNotAccumulate(t *testing.T) {
	t.Setenv(HonourKeyConfigEnvVar, "")
	file := writeForkKeyConfigFile(t, "config.yml", forkKeyConfigSample)

	first, err := loadUserConfigWithDefaults([]*ConfigFile{file}, false)
	requireNoError(t, err)
	second, err := loadUserConfigWithDefaults([]*ConfigFile{file}, true)
	requireNoError(t, err)

	assert.Len(t, first.IgnoredKeySettings(), 1)
	assert.Len(t, second.IgnoredKeySettings(), 1)
}

func TestForkIgnoreKeyConfigInvalidIgnoredKeyDoesNotBlockLoading(t *testing.T) {
	t.Setenv(HonourKeyConfigEnvVar, "")
	file := writeForkKeyConfigFile(t, "config.yml", "keybinding:\n  universal:\n    quit: <not-a-key>\n")
	_, err := loadUserConfigWithDefaults([]*ConfigFile{file}, false)
	requireNoError(t, err)

	t.Setenv(HonourKeyConfigEnvVar, "1")
	_, err = loadUserConfigWithDefaults([]*ConfigFile{file}, false)
	if err == nil {
		t.Fatal("expected the invalid key to fail validation when key configuration is honoured")
	}
}
