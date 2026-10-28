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

	var letter byte
	switch {
	case str[0] >= 'A' && str[0] <= 'Z':
		letter = str[0] + ('a' - 'A')
	case str[0] >= 'a' && str[0] <= 'z':
		letter = str[0]
	default:
		return key
	}

	// J and K are the one exception to letter folding: Shift+J and Shift+K
	// extend the list selection exactly like Shift+Down and Shift+Up, so they
	// must stay distinct from plain j and k. A terminal may report either the
	// uppercase letter or the lowercase letter with the Shift modifier.
	shifted := key.Mod() == ModShift || (str[0] >= 'A' && str[0] <= 'Z')
	if shifted && (letter == 'j' || letter == 'k') {
		key.str = string(letter - ('a' - 'A'))
	} else {
		key.str = string(letter)
	}
	key.mod = ModNone
	return key
}
