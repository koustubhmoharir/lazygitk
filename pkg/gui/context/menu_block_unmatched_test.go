package context

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

func TestPrefixNavigationDoesNotConsumeConfiguredLetterAccelerators(t *testing.T) {
	for _, prefix := range []bool{false, true} {
		menu := newMenuContextWithBase(t)
		original := config.GetDefaultConfig().Keybinding
		original.Universal.PrevItem = config.Keybinding{"r"}
		original.Universal.ConfirmMenu = config.Keybinding{"e"}
		original.Universal.Return = config.Keybinding{"x"}
		if prefix {
			menu.SetBlockUnmatchedKeys([]gocui.Key{gocui.NewKeyRune('r')})
		}
		var received config.KeybindingConfig
		menu.ListContextTrait.Context.(*SimpleContext).BaseContext.AddKeybindingsFn(func(opts types.KeybindingsOpts) []*types.Binding {
			received = opts.Config
			return nil
		})
		menu.GetKeybindings(types.KeybindingsOpts{Config: original})
		if prefix {
			assert.Equal(t, config.Keybinding{"<up>"}, received.Universal.PrevItem)
			assert.Equal(t, config.Keybinding{"<enter>"}, received.Universal.ConfirmMenu)
			assert.Equal(t, config.Keybinding{"<esc>"}, received.Universal.Return)
		} else {
			assert.Equal(t, original, received)
		}
		assert.Equal(t, config.Keybinding{"r"}, original.Universal.PrevItem)
		assert.Equal(t, config.Keybinding{"e"}, original.Universal.ConfirmMenu)
		assert.Equal(t, config.Keybinding{"x"}, original.Universal.Return)
	}
}

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
