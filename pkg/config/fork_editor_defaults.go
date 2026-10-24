package config

import "slices"

// applyForkEditorDefaults adds conventional Ctrl+arrow word motion while
// retaining the platform's existing editor bindings.
func applyForkEditorDefaults(config *UserConfig) {
	config.Keybinding.Universal.MoveWordLeft = appendUniqueKey(config.Keybinding.Universal.MoveWordLeft, "<ctrl+left>")
	config.Keybinding.Universal.MoveWordRight = appendUniqueKey(config.Keybinding.Universal.MoveWordRight, "<ctrl+right>")
}

func appendUniqueKey(keys Keybinding, key string) Keybinding {
	if !slices.Contains(keys, key) {
		keys = append(keys, key)
	}
	return keys
}
