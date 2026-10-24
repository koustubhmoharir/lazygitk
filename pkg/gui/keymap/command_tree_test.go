package keymap

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

func TestBuildCommandTreePreservesGlobalRoutesAtSharedPrefixes(t *testing.T) {
	global := &types.Binding{CommandPaths: [][]gocui.Key{{key('g'), key('a')}}}
	local := &types.Binding{ViewName: "files", CommandPaths: [][]gocui.Key{{key('g'), key('b')}}}
	root := BuildCommandTree([]*types.Binding{global, local}, "files")

	g := childTree(t, root, key('g'))
	assert.Empty(t, g.Bindings)
	assert.Equal(t, []*types.Binding{global}, childTree(t, g, key('a')).Bindings)
	assert.Equal(t, []*types.Binding{local}, childTree(t, g, key('b')).Bindings)
}

func TestBuildCommandTreeLocalBindingShadowsGlobalOnlyAtExactPath(t *testing.T) {
	globalExact := &types.Binding{CommandPaths: [][]gocui.Key{{key('g'), key('a')}}}
	globalSibling := &types.Binding{CommandPaths: [][]gocui.Key{{key('g'), key('b')}}}
	globalDescendant := &types.Binding{CommandPaths: [][]gocui.Key{{key('g'), key('a'), key('c')}}}
	localExact := &types.Binding{ViewName: "files", CommandPaths: [][]gocui.Key{{key('g'), key('a')}}}
	root := BuildCommandTree([]*types.Binding{globalExact, globalSibling, globalDescendant, localExact}, "files")

	g := childTree(t, root, key('g'))
	a := childTree(t, g, key('a'))
	assert.Equal(t, []*types.Binding{localExact}, a.Bindings)
	assert.Equal(t, []*types.Binding{globalDescendant}, childTree(t, a, key('c')).Bindings)
	assert.Equal(t, []*types.Binding{globalSibling}, childTree(t, g, key('b')).Bindings)
}

func TestBuildCommandTreeRetainsSameViewCandidateOrder(t *testing.T) {
	first := &types.Binding{ViewName: "files", Keys: []gocui.Key{key('x')}}
	second := &types.Binding{ViewName: "files", Keys: []gocui.Key{key('x')}}
	root := BuildCommandTree([]*types.Binding{first, second}, "files")
	assert.Equal(t, []*types.Binding{first, second}, childTree(t, root, key('x')).Bindings)
}

func TestBuildCommandTreeCaseFoldsAndSortsChildrenByLabel(t *testing.T) {
	upper := &types.Binding{Keys: []gocui.Key{gocui.NewKeyRune('X')}}
	lower := &types.Binding{Keys: []gocui.Key{key('x')}}
	a := &types.Binding{Keys: []gocui.Key{key('a')}}
	root := BuildCommandTree([]*types.Binding{upper, lower, a}, "files")

	if len(root.Children) != 2 {
		t.Fatalf("expected 2 root children, got %d", len(root.Children))
	}
	assert.Equal(t, "a", config.LabelForKey(root.Children[0].Key))
	assert.Equal(t, "x", config.LabelForKey(root.Children[1].Key))
	assert.Equal(t, []*types.Binding{upper, lower}, root.Children[1].Tree.Bindings)
}

func TestBuildCommandTreePreservesCtrlAndFullPathsAreNotContinuations(t *testing.T) {
	ctrl := &types.Binding{Keys: []gocui.Key{gocui.NewKeyStrMod("x", gocui.ModCtrl)}}
	path := &types.Binding{CommandPaths: [][]gocui.Key{{key('x'), key('y')}}}
	root := BuildCommandTree([]*types.Binding{ctrl, path}, "files")

	assert.Equal(t, []*types.Binding{ctrl}, childTree(t, root, gocui.NewKeyStrMod("x", gocui.ModCtrl)).Bindings)
	x := childTree(t, root, key('x'))
	assert.Empty(t, x.Bindings)
	assert.Equal(t, []*types.Binding{path}, childTree(t, x, key('y')).Bindings)
}

func TestBuildCommandTreeDeduplicatesSameBindingAndPath(t *testing.T) {
	binding := &types.Binding{
		Keys:         []gocui.Key{key('a'), key('a')},
		CommandPaths: [][]gocui.Key{{key('a')}, {gocui.NewKeyRune('A')}},
	}
	root := BuildCommandTree([]*types.Binding{binding}, "files")
	assert.Equal(t, []*types.Binding{binding}, childTree(t, root, key('a')).Bindings)
}

func TestBuildCommandTreeIgnoresEmptyAndMouseRoutes(t *testing.T) {
	binding := &types.Binding{
		Keys: []gocui.Key{gocui.NewKeyName(gocui.MouseLeft), key('a')},
		CommandPaths: [][]gocui.Key{
			{},
			{key('b'), gocui.NewKeyName(gocui.MouseWheelDown)},
		},
	}
	root := BuildCommandTree([]*types.Binding{binding}, "files")
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 root child, got %d", len(root.Children))
	}
	assert.Equal(t, []*types.Binding{binding}, childTree(t, root, key('a')).Bindings)
}

func TestBuildCommandTreeDoesNotAliasOrMutateInputs(t *testing.T) {
	originalKey := gocui.NewKeyRune('A')
	pathKey := key('b')
	binding := &types.Binding{
		Keys:         []gocui.Key{originalKey},
		CommandPaths: [][]gocui.Key{{pathKey}},
	}
	bindings := []*types.Binding{binding}
	root := BuildCommandTree(bindings, "files")

	assert.Equal(t, originalKey, binding.Keys[0])
	assert.Equal(t, pathKey, binding.CommandPaths[0][0])
	bindings[0] = nil
	binding.Keys[0] = key('z')
	binding.CommandPaths[0][0] = key('z')
	assert.Equal(t, []*types.Binding{binding}, childTree(t, root, key('a')).Bindings)
	assert.Equal(t, []*types.Binding{binding}, childTree(t, root, key('b')).Bindings)
	assert.NotSame(t, root.Children[0].Tree, root.Children[1].Tree)
}

func key(r rune) gocui.Key { return gocui.NewKeyRune(r) }

func childTree(t *testing.T, tree *CommandTree, key gocui.Key) *CommandTree {
	t.Helper()
	for _, edge := range tree.Children {
		if edge.Key.Equals(gocui.NormalizeCommandKey(key)) {
			return edge.Tree
		}
	}
	t.Fatalf("missing child %q", config.LabelForKey(key))
	return nil
}

func TestBuildCommandTreeExplicitOverridesPrecedeNewLocalDefaults(t *testing.T) {
	global := &types.Binding{CommandID: "configured-global", Keys: []gocui.Key{key('x')}}
	localDefault := &types.Binding{CommandID: "new-local-default", ViewName: "files", DefaultCommand: true, Keys: []gocui.Key{key('x')}}
	localSibling := &types.Binding{CommandID: "local-sibling", ViewName: "files", DefaultCommand: true, CommandPaths: [][]gocui.Key{{key('x'), key('s')}}}
	for _, bindings := range [][]*types.Binding{{global, localDefault, localSibling}, {localSibling, localDefault, global}} {
		root := BuildCommandTree(bindings, "files")
		node := childTree(t, root, key('x'))
		assert.Equal(t, []*types.Binding{global}, node.Bindings)
		assert.Equal(t, []*types.Binding{localSibling}, childTree(t, node, key('s')).Bindings, "a complete-path override must not delete sibling continuations")
	}
	localDefault.DefaultCommand = false
	root := BuildCommandTree([]*types.Binding{global, localDefault}, "files")
	assert.Equal(t, []*types.Binding{localDefault}, childTree(t, root, key('x')).Bindings, "explicit local overrides retain native precedence over explicit globals")
}

func TestBuildCommandTreeExplicitSameViewRouteDisplacesDefault(t *testing.T) {
	first := &types.Binding{ViewName: "files", DefaultCommand: true, Keys: []gocui.Key{key('a')}}
	second := &types.Binding{ViewName: "files", Keys: []gocui.Key{key('a')}}
	root := BuildCommandTree([]*types.Binding{first, second}, "files")
	assert.Equal(t, []*types.Binding{second}, childTree(t, root, key('a')).Bindings)
	assert.True(t, first.DefaultCommand)
	assert.Equal(t, []gocui.Key{key('a')}, first.Keys, "ownership must not mutate source definitions")
}

func TestBuildCommandTreeKeepsLowerPriorityOwnersAsOrderedFallbacks(t *testing.T) {
	defaultGlobal := &types.Binding{DefaultCommand: true, Keys: []gocui.Key{key('e')}}
	defaultLocal := &types.Binding{DefaultCommand: true, ViewName: "files", Keys: []gocui.Key{key('e')}}
	explicitGlobal := &types.Binding{Keys: []gocui.Key{key('e')}}
	root := BuildCommandTree([]*types.Binding{defaultGlobal, defaultLocal, explicitGlobal}, "files")

	node := childTree(t, root, key('e'))
	assert.Equal(t, []*types.Binding{explicitGlobal}, node.Bindings)
	assert.Equal(t, []*types.Binding{defaultLocal, defaultGlobal}, node.Fallbacks)
}

func TestBuildCommandTreeFallbacksKeepDefaultsOrderForEqualPriority(t *testing.T) {
	first := &types.Binding{DefaultCommand: true, ViewName: "files", Keys: []gocui.Key{key('e')}}
	second := &types.Binding{DefaultCommand: true, ViewName: "files", Keys: []gocui.Key{key('e')}}
	global := &types.Binding{DefaultCommand: true, Keys: []gocui.Key{key('e')}}
	root := BuildCommandTree([]*types.Binding{first, global, second}, "files")

	node := childTree(t, root, key('e'))
	assert.Equal(t, []*types.Binding{first, second}, node.Bindings)
	assert.Equal(t, []*types.Binding{global}, node.Fallbacks)
}
