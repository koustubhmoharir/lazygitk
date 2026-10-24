package types

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/stretchr/testify/assert"
)

func TestCommandPathsKeepContinuationsOutOfDirectKeys(t *testing.T) {
	paths := [][]gocui.Key{
		{gocui.NewKeyRune('v')},
		{gocui.NewKeyRune('d'), gocui.NewKeyRune('w')},
		{gocui.NewKeyRune('t'), gocui.NewKeyRune('d'), gocui.NewKeyName(gocui.KeyF3)},
	}
	binding := &Binding{}
	binding.SetCommandPaths(paths)
	assert.Equal(t, []string{"v", "d w", "t d <f3>"}, binding.KeyLabels())
	assert.Equal(t, []gocui.Key{gocui.NewKeyRune('v')}, binding.Keys)
	paths[1][0] = gocui.NewKeyRune('z')
	assert.Equal(t, "d w", binding.KeyLabels()[1])
	binding.SetCommandPaths(nil)
	assert.Empty(t, binding.Keys)
	assert.Empty(t, binding.CommandPaths)
}

func TestMenuItemKeyLabelsUsesBindingPathsOrConfiguredKeys(t *testing.T) {
	binding := &Binding{Keys: []gocui.Key{gocui.NewKeyRune('x')}}
	binding.SetCommandPaths([][]gocui.Key{{gocui.NewKeyRune('d'), gocui.NewKeyRune('w')}})
	binding.Keys = []gocui.Key{gocui.NewKeyRune('x')}
	assert.Equal(t, []string{"x", "d w"}, (&MenuItem{Binding: binding, Keys: []gocui.Key{gocui.NewKeyRune('q')}}).KeyLabels())
	assert.Equal(t, []string{"q"}, (&MenuItem{Keys: []gocui.Key{gocui.NewKeyRune('q')}}).KeyLabels())
}
