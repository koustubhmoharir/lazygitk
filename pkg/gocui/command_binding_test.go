package gocui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandKeybindings(t *testing.T) {
	t.Run("exact binding remains case sensitive", func(t *testing.T) {
		g := newTestGui(t)
		_, _ = g.SetView("main", 0, 0, 20, 10, 0)
		_, _ = g.SetCurrentView("main")
		called := 0
		g.SetKeybinding("main", NewKeyRune('x'), func(*Gui, *View) error { called++; return nil })
		assert.NoError(t, g.onKey(&GocuiEvent{Type: eventKey, Key: NewKeyRune('X')}))
		assert.Zero(t, called)
		assert.NoError(t, g.onKey(&GocuiEvent{Type: eventKey, Key: NewKeyRune('x')}))
		assert.Equal(t, 1, called)
	})

	t.Run("ASCII letter aliases match only command bindings", func(t *testing.T) {
		for _, key := range []Key{NewKeyRune('x'), NewKeyRune('X'), NewKeyStrMod("X", ModShift), NewKeyStrMod("x", ModShift)} {
			g := newTestGui(t)
			_, _ = g.SetView("main", 0, 0, 20, 10, 0)
			_, _ = g.SetCurrentView("main")
			command, exact := 0, 0
			g.SetCommandKeybinding("main", NewKeyRune('x'), func(*Gui, *View) error { command++; return nil })
			g.SetKeybinding("main", NewKeyRune('x'), func(*Gui, *View) error { exact++; return nil })
			assert.NoError(t, g.onKey(&GocuiEvent{Type: eventKey, Key: key}))
			assert.Equal(t, 1, command)
			assert.Equal(t, 0, exact)
		}
	})

	t.Run("nonletters and modified keys retain exact identity", func(t *testing.T) {
		for _, test := range []struct{ bound, distinct Key }{
			{NewKeyStrMod("x", ModCtrl), NewKeyStrMod("X", ModCtrl)},
			{NewKeyStrMod("x", ModAlt), NewKeyStrMod("X", ModAlt)},
			{NewKeyRune('!'), NewKeyStrMod("!", ModShift)},
			{NewKeyStrMod("!", ModShift), NewKeyRune('!')},
		} {
			g := newTestGui(t)
			_, _ = g.SetView("main", 0, 0, 20, 10, 0)
			_, _ = g.SetCurrentView("main")
			called := 0
			g.SetCommandKeybinding("main", test.bound, func(*Gui, *View) error { called++; return nil })
			assert.NoError(t, g.onKey(&GocuiEvent{Type: eventKey, Key: test.distinct}))
			assert.Zero(t, called)
			assert.NoError(t, g.onKey(&GocuiEvent{Type: eventKey, Key: test.bound}))
			assert.Equal(t, 1, called)
		}
	})
}

func TestEditableViewSuppressesCommandLettersAcrossScopes(t *testing.T) {
	for _, test := range []struct {
		name          string
		keybindOnEdit bool
		editorAccepts bool
	}{
		{name: "editor accepts", editorAccepts: true},
		{name: "editor declines"},
		{name: "keybindings enabled while editing", keybindOnEdit: true, editorAccepts: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := newTestGui(t)
			parent, child := setupParentAndChildView(t, g)
			child.Editable, child.KeybindOnEdit = true, test.keybindOnEdit
			seen := []Key{}
			child.Editor = EditorFunc(func(_ *View, k Key) bool { seen = append(seen, k); return test.editorAccepts })
			called := 0
			for _, scope := range []string{child.Name(), parent.Name(), ""} {
				g.SetCommandKeybinding(scope, NewKeyRune('x'), func(*Gui, *View) error { called++; return nil })
			}
			key := NewKeyStrMod("X", ModShift)
			assert.NoError(t, g.onKey(&GocuiEvent{Type: eventKey, Key: key}))
			if test.keybindOnEdit {
				assert.Equal(t, 1, called)
			} else {
				assert.Zero(t, called)
				assert.Equal(t, []Key{key}, seen)
			}
		})
	}
}

func TestLocalCommandBindingBlocksParentAndGlobal(t *testing.T) {
	for _, test := range []struct {
		name         string
		handlerError error
	}{
		{name: "handled"},
		{name: "declined", handlerError: ErrKeybindingNotHandled},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := newTestGui(t)
			parent, child := setupParentAndChildView(t, g)
			called := []string{}
			g.SetCommandKeybinding(child.Name(), NewKeyRune('x'), func(*Gui, *View) error { called = append(called, "local"); return test.handlerError })
			g.SetCommandKeybinding(parent.Name(), NewKeyRune('x'), func(*Gui, *View) error { called = append(called, "parent"); return nil })
			g.SetCommandKeybinding("", NewKeyRune('x'), func(*Gui, *View) error { called = append(called, "global"); return nil })
			err := g.onKey(&GocuiEvent{Type: eventKey, Key: NewKeyRune('X')})
			if test.handlerError != nil {
				assert.ErrorIs(t, err, test.handlerError)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, []string{"local"}, called)
		})
	}
}

func TestCommandBindingDoesNotInterceptPasteIntoNonEditableView(t *testing.T) {
	g := newTestGui(t)
	_, _ = g.SetView("main", 0, 0, 20, 10, 0)
	_, _ = g.SetCurrentView("main")
	g.IsPasting = true
	called := 0
	g.SetCommandKeybinding("main", NewKeyRune('x'), func(*Gui, *View) error { called++; return nil })
	assert.NoError(t, g.onKey(&GocuiEvent{Type: eventKey, Key: NewKeyRune('x')}))
	assert.Zero(t, called)
}
