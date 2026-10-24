package gocui

import "github.com/gdamore/tcell/v3"

// NormalizeCommandKey canonicalizes ASCII command letters for command matching
// and collision checks only. Never use it to normalize input events, literal
// text, or custom Ctrl combinations.
func NormalizeCommandKey(key Key) Key {
	if key.KeyName() != KeyName(tcell.KeyRune) {
		return key
	}

	str := key.Str()
	if len(str) != 1 {
		return key
	}
	if key.Mod() != ModNone && key.Mod() != ModShift {
		return key
	}

	switch {
	case str[0] >= 'A' && str[0] <= 'Z':
		key.str = string(str[0] + ('a' - 'A'))
	case str[0] >= 'a' && str[0] <= 'z':
		// Already lowercase; a terminal may still report the Shift modifier.
	default:
		return key
	}
	key.mod = ModNone
	return key
}
