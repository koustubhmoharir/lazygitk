package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/karimkhaleel/jsonschema"
)

// CommandBindingPaths contains alternate paths for one stable command ID.
// Each Keybinding is a sequential path; a scalar is a one-key path.
type CommandBindingPaths []Keybinding

// UnmarshalJSON accepts Keybinding's scalar and list forms for each path.
func (paths *CommandBindingPaths) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*paths = nil
		return nil
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	result := make(CommandBindingPaths, len(raw))
	for i, entry := range raw {
		var scalar string
		if err := json.Unmarshal(entry, &scalar); err == nil {
			result[i] = Keybinding{scalar}
			continue
		}
		if err := json.Unmarshal(entry, (*[]string)(&result[i])); err != nil {
			return fmt.Errorf("command binding path %d must be a string or list of strings: %w", i+1, err)
		}
	}
	*paths = result
	return nil
}

// JSONSchema describes each path as one key or a sequential list of up to
// three keys. Null explicitly removes the command's keyboard paths.
func (CommandBindingPaths) JSONSchema() *jsonschema.Schema {
	keybindingSchema := (Keybinding{}).JSONSchema()
	keybindingSchema.OneOf[1].MinItems = 1
	keybindingSchema.OneOf[1].MaxItems = 3
	return &jsonschema.Schema{
		Description: "Alternate keyboard command paths. Enter, Escape, Up and Down are reserved for prefix-menu control after the first key; mouse keys are not supported.",
		OneOf: []*jsonschema.Schema{
			{Type: "array", Items: keybindingSchema},
			{Type: "null"},
		},
	}
}

func (config *UserConfig) validateCommandBindings() error {
	for commandID, paths := range config.CommandBindings {
		if strings.TrimSpace(commandID) == "" || strings.TrimSpace(commandID) != commandID {
			return fmt.Errorf("commandBindings: command ID %q must be nonempty and have no surrounding whitespace", commandID)
		}
		for pathIndex, path := range paths {
			if len(path) < 1 || len(path) > 3 {
				return fmt.Errorf("commandBindings.%s: path %d must contain between 1 and 3 keys", commandID, pathIndex+1)
			}
			for step, label := range path {
				key, ok := KeyFromLabel(label)
				if !ok || !key.IsSet() {
					return fmt.Errorf("commandBindings.%s: invalid key %q", commandID, label)
				}
				if gocui.IsMouseKey(key) {
					return fmt.Errorf("commandBindings.%s: mouse key %q is not a keyboard command path", commandID, label)
				}
				if step > 0 && key.Mod() == gocui.ModNone {
					name := key.KeyName()
					if name == gocui.KeyEnter || name == gocui.KeyEsc || name == gocui.KeyArrowUp || name == gocui.KeyArrowDown {
						return fmt.Errorf("commandBindings.%s: key %q is reserved for prefix-menu navigation after the first key", commandID, label)
					}
				}
			}
		}
	}
	return nil
}
