package gocui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeCommandKey(t *testing.T) {
	for _, letter := range "abcdefghijklmnopqrstuvwxyz" {
		lower := NewKeyRune(letter)
		upper := NewKeyRune(letter - 'a' + 'A')
		shiftUpper := NewKeyStrMod(string(letter-'a'+'A'), ModShift)
		shiftLower := NewKeyStrMod(string(letter), ModShift)

		canonical := NormalizeCommandKey(lower)
		assert.Equal(t, canonical, NormalizeCommandKey(upper))
		assert.Equal(t, canonical, NormalizeCommandKey(shiftUpper))
		assert.Equal(t, canonical, NormalizeCommandKey(shiftLower))
		assert.Equal(t, canonical, NormalizeCommandKey(canonical))
	}
}

func TestNormalizeCommandKeyPreservesNonLettersAndModifiers(t *testing.T) {
	keys := []Key{
		NewKeyRune('!'),
		NewKeyStrMod("!", ModShift),
		NewKeyName(KeyEnter),
		NewKey(KeyEnter, "", ModShift),
		NewKeyStrMod("a", ModCtrl),
		NewKeyStrMod("A", ModAlt),
		NewKeyStrMod("a", ModMeta),
		NewKeyStrMod("A", ModShift|ModCtrl),
		NewKeyRune('界'),
		NewKeyStrMod("e\u0301", ModNone),
		NewKeyStrMod("ab", ModNone),
	}
	for _, key := range keys {
		assert.Equal(t, key, NormalizeCommandKey(key))
	}
	assert.NotEqual(t, NormalizeCommandKey(NewKeyRune('!')), NormalizeCommandKey(NewKeyStrMod("!", ModShift)))
}

func TestNormalizeCommandKeyDoesNotChangeKeyEqualityOrInputValue(t *testing.T) {
	original := NewKeyRune('A')
	normalized := NormalizeCommandKey(original)

	assert.False(t, original.Equals(NewKeyRune('a')))
	assert.Equal(t, NewKeyRune('a'), normalized)
	assert.Equal(t, NewKeyRune('A'), original)
}
