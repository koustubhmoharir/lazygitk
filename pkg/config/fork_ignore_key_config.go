package config

import (
	"fmt"
	"maps"
	"os"

	"gopkg.in/yaml.v3"
)

// HonourKeyConfigEnvVar is the internal switch that makes the loader honour
// key configuration (the keybinding section, commandBindings, and custom
// command keys). Without it, the fork ignores those settings: its defaults
// always apply, and a loaded file that contains them is reported at startup
// (roadmap item 13). The integration harness sets it for every scenario;
// a value of "" or "0" turns it off again. Users have no reason to set it.
const HonourKeyConfigEnvVar = "LAZYGITK_HONOUR_KEY_CONFIG"

// The kinds of ignored key settings, as reported to the user.
const (
	ignoredKindKeybinding      = "keybinding"
	ignoredKindCommandBindings = "commandBindings"
	ignoredKindCustomCommand   = "customCommands key"
)

// IgnoredKeySetting records which key settings one loaded config file contained
// and that were ignored.
type IgnoredKeySetting struct {
	Path  string
	Kinds []string
}

// IgnoredKeySettings lists the config files whose key settings were ignored
// while loading this config. It is empty when key configuration is honoured.
func (config *UserConfig) IgnoredKeySettings() []IgnoredKeySetting {
	return config.ignoredKeySettings
}

func honourKeyConfig() bool {
	value := os.Getenv(HonourKeyConfigEnvVar)
	return value != "" && value != "0"
}

// forkKeyConfigDefaults holds the loader's starting values for the key settings.
// Config files are merged into the base in place, so the defaults are captured
// before the first file is read.
type forkKeyConfigDefaults struct {
	keybinding      KeybindingConfig
	commandBindings map[string]CommandBindingPaths
}

// captureForkKeyConfigDefaults is called by loadUserConfig before merging any
// file. It starts a fresh ignored-settings list for this load.
func captureForkKeyConfigDefaults(base *UserConfig) forkKeyConfigDefaults {
	base.ignoredKeySettings = nil
	return forkKeyConfigDefaults{
		keybinding:      base.Keybinding,
		commandBindings: maps.Clone(base.CommandBindings),
	}
}

// discardKeyConfig runs after one config file has been merged into base. When
// key configuration is not honoured, it records the key settings the file
// contained and restores the loader's starting values, so no key setting from
// any file takes effect. Doing this per file leaves the same result as
// discarding after all files, and lets later files merge onto clean defaults.
func (defaults forkKeyConfigDefaults) discardKeyConfig(base *UserConfig, path string, content []byte) error {
	if honourKeyConfig() {
		return nil
	}
	kinds, err := ignoredKeyKinds(content)
	if err != nil {
		return fmt.Errorf("The config at `%s` couldn't be inspected for ignored key settings.\n%w", path, err)
	}
	if len(kinds) > 0 {
		base.ignoredKeySettings = append(base.ignoredKeySettings, IgnoredKeySetting{Path: path, Kinds: kinds})
	}

	base.Keybinding = defaults.keybinding
	base.CommandBindings = maps.Clone(defaults.commandBindings)
	base.keybindingExplicitFields = make(map[string]struct{})
	clearCustomCommandKeys(base.CustomCommands)
	return nil
}

func clearCustomCommandKeys(commands []CustomCommand) {
	for i := range commands {
		commands[i].Key = nil
		clearCustomCommandKeys(commands[i].CommandMenu)
	}
}

// forkKeyConfigYAML decodes only the key-related parts of a config document.
// Nodes are used so that the same non-empty rules apply as to the typed loader.
type forkKeyConfigYAML struct {
	Keybinding      yaml.Node                  `yaml:"keybinding"`
	CommandBindings yaml.Node                  `yaml:"commandBindings"`
	CustomCommands  []forkCustomCommandKeyYAML `yaml:"customCommands"`
}

type forkCustomCommandKeyYAML struct {
	Key         yaml.Node                  `yaml:"key"`
	CommandMenu []forkCustomCommandKeyYAML `yaml:"commandMenu"`
}

// ignoredKeyKinds reports which of the ignored kinds a YAML document contains.
// It inspects the file itself, not its values relative to the defaults.
func ignoredKeyKinds(content []byte) ([]string, error) {
	var document forkKeyConfigYAML
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, err
	}
	var kinds []string
	if document.Keybinding.Kind == yaml.MappingNode && len(document.Keybinding.Content) > 0 {
		kinds = append(kinds, ignoredKindKeybinding)
	}
	if document.CommandBindings.Kind == yaml.MappingNode && len(document.CommandBindings.Content) > 0 {
		kinds = append(kinds, ignoredKindCommandBindings)
	}
	hasKey, err := anyCustomCommandHasKey(document.CustomCommands)
	if err != nil {
		return nil, err
	}
	if hasKey {
		kinds = append(kinds, ignoredKindCustomCommand)
	}
	return kinds, nil
}

func anyCustomCommandHasKey(commands []forkCustomCommandKeyYAML) (bool, error) {
	for _, command := range commands {
		if command.Key.Kind != 0 {
			var key Keybinding
			if err := command.Key.Decode(&key); err != nil {
				return false, err
			}
			if len(key) > 0 {
				return true, nil
			}
		}
		hasKey, err := anyCustomCommandHasKey(command.CommandMenu)
		if err != nil || hasKey {
			return hasKey, err
		}
	}
	return false, nil
}
