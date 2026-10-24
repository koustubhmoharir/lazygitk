package context

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

func newMenuContextWithBase(t *testing.T) *MenuContext {
	t.Helper()
	menu, _ := newMenuBeforeAcceptContext()
	menu.ListContextTrait = &ListContextTrait{
		Context: NewSimpleContext(NewBaseContext(NewBaseContextOpts{})),
	}
	return menu
}

func TestMenuBlockedKeysFollowAcceleratorsAndPrecedeInheritedBindings(t *testing.T) {
	menu := newMenuContextWithBase(t)
	acceleratorKey := gocui.NewKeyRune('x')
	blockedKey := gocui.NewKeyRune('j')
	menu.SetMenuItems([]*types.MenuItem{{Keys: []gocui.Key{acceleratorKey}}}, nil)
	menu.SetKeybindingsTakePrecedence(true)
	menu.SetBlockUnmatchedKeys([]gocui.Key{acceleratorKey, blockedKey})
	menu.ListContextTrait.Context.(*SimpleContext).BaseContext.AddKeybindingsFn(func(types.KeybindingsOpts) []*types.Binding {
		return []*types.Binding{{Keys: []gocui.Key{blockedKey}}}
	})

	bindings := menu.GetKeybindings(types.KeybindingsOpts{})
	assert.Equal(t, []gocui.Key{acceleratorKey}, bindings[0].Keys)
	assert.Equal(t, []gocui.Key{acceleratorKey}, bindings[1].Keys)
	assert.Equal(t, []gocui.Key{blockedKey}, bindings[2].Keys)
	assert.Equal(t, []gocui.Key{blockedKey}, bindings[3].Keys)
	assert.NoError(t, bindings[2].Handler())
}

func TestMenuBlockedKeySetterCopiesAndResets(t *testing.T) {
	menu := newMenuContextWithBase(t)
	keys := []gocui.Key{gocui.NewKeyRune('j')}
	menu.SetBlockUnmatchedKeys(keys)
	keys[0] = gocui.NewKeyRune('k')
	assert.Equal(t, []gocui.Key{gocui.NewKeyRune('j')}, menu.blockUnmatchedKeys)

	menu.SetBlockUnmatchedKeys(nil)
	assert.Nil(t, menu.blockUnmatchedKeys)
}

func TestRegularMenuKeybindingsUnchangedWithoutBlockedKeys(t *testing.T) {
	menu := newMenuContextWithBase(t)
	acceleratorKey := gocui.NewKeyRune('x')
	inheritedKey := gocui.NewKeyRune('j')
	menu.SetMenuItems([]*types.MenuItem{{Keys: []gocui.Key{acceleratorKey}}}, nil)
	menu.SetKeybindingsTakePrecedence(true)
	menu.ListContextTrait.Context.(*SimpleContext).BaseContext.AddKeybindingsFn(func(types.KeybindingsOpts) []*types.Binding {
		return []*types.Binding{{Keys: []gocui.Key{inheritedKey}}}
	})

	bindings := menu.GetKeybindings(types.KeybindingsOpts{})
	assert.Len(t, bindings, 2)
	assert.Equal(t, []gocui.Key{acceleratorKey}, bindings[0].Keys)
	assert.Equal(t, []gocui.Key{inheritedKey}, bindings[1].Keys)
}
