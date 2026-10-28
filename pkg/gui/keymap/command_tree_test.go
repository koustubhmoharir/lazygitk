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

// h and l are not declared by defaults, so these children sort by label. J and
// K are not used because they do not fold to j and k.
func TestBuildCommandTreeCaseFoldsAndSortsUndeclaredChildrenByLabel(t *testing.T) {
	upper := &types.Binding{Keys: []gocui.Key{gocui.NewKeyRune('L')}}
	lower := &types.Binding{Keys: []gocui.Key{key('l')}}
	h := &types.Binding{Keys: []gocui.Key{key('h')}}
	root := BuildCommandTree([]*types.Binding{upper, lower, h}, "files")

	if len(root.Children) != 2 {
		t.Fatalf("expected 2 root children, got %d", len(root.Children))
	}
	assert.Equal(t, "h", config.LabelForKey(root.Children[0].Key))
	assert.Equal(t, "l", config.LabelForKey(root.Children[1].Key))
	assert.Equal(t, []*types.Binding{upper, lower}, root.Children[1].Tree.Bindings)
}

// childLabels returns the child labels of tree in order.
func childLabels(tree *CommandTree) []string {
	labels := make([]string, 0, len(tree.Children))
	for _, edge := range tree.Children {
		labels = append(labels, config.LabelForKey(edge.Key))
	}
	return labels
}

func TestBuildCommandTreeOrdersDiffPrefixByDeclaration(t *testing.T) {
	ids := []string{"diff.commit.worktree", "diff.commit.previous", "diff.commit.selected-commit"}
	bindings := make([]*types.Binding, 0, len(ids))
	for _, id := range ids {
		paths, ok := DefaultPaths(id)
		if !ok {
			t.Fatalf("missing default paths for %s", id)
		}
		bindings = append(bindings, &types.Binding{CommandID: id, CommandPaths: paths})
	}
	root := BuildCommandTree(bindings, "commits")

	assert.Equal(t, []string{"w", "p", "c"}, childLabels(childTree(t, root, key('d'))))
}

func TestBuildCommandTreeOrdersFilesDiffPrefixByDeclaration(t *testing.T) {
	// Listed out of declaration order on purpose; the menu follows the keymap.
	ids := []string{"diff.working-file.selected-commit", "diff.working-file.staged", "diff.working-file.head"}
	bindings := make([]*types.Binding, 0, len(ids))
	for _, id := range ids {
		paths, ok := DefaultPaths(id)
		if !ok {
			t.Fatalf("missing default paths for %s", id)
		}
		bindings = append(bindings, &types.Binding{CommandID: id, CommandPaths: paths})
	}
	root := BuildCommandTree(bindings, "files")

	assert.Equal(t, []string{"w", "s", "c"}, childLabels(childTree(t, root, key('d'))))
}

func TestBuildCommandTreeOrdersTPrefixByDeclaration(t *testing.T) {
	ids := []string{
		"diff.commit.external", "diff.history-file.working-file", "diff.options-menu", "diff.toggle-whitespace", "diff.renderer.next", "diff.context.increase",
		"filter.open", "file.filter-menu", "ui.file-tree.toggle", "branch.sort-menu",
		"navigation.diff.next-hunk", "selection.diff.toggle-hunk",
	}
	bindings := make([]*types.Binding, 0, len(ids))
	for _, id := range ids {
		paths, ok := DefaultPaths(id)
		if !ok {
			t.Fatalf("missing default paths for %s", id)
		}
		bindings = append(bindings, &types.Binding{CommandID: id, CommandPaths: paths})
	}
	root := BuildCommandTree(bindings, "")

	// Each group keeps the rank of its first declaration: t d, t t, t s, t n, t h.
	tPrefix := childTree(t, root, key('t'))
	assert.Equal(t, []string{"d", "t", "s", "n", "h"}, childLabels(tPrefix))
	assert.Equal(t, []string{"e", "f", "c", "w", "n", "+"}, childLabels(childTree(t, tPrefix, key('d'))))
}

func TestBuildCommandTreeOrdersUserOnlyRoutesAfterDeclaredOnes(t *testing.T) {
	defaults := &types.Binding{CommandID: "diff.commit.worktree", CommandPaths: mustDefaultPaths(t, "diff.commit.worktree")}
	userZ := &types.Binding{CommandID: "user-z", CommandPaths: [][]gocui.Key{{key('d'), key('z')}}}
	userA := &types.Binding{CommandID: "user-a", CommandPaths: [][]gocui.Key{{key('d'), key('a')}}}
	userK := &types.Binding{CommandID: "user-k", CommandPaths: [][]gocui.Key{{key('d'), key('k')}}}
	root := BuildCommandTree([]*types.Binding{userZ, defaults, userA, userK}, "")

	assert.Equal(t, []string{"w", "a", "k", "z"}, childLabels(childTree(t, root, key('d'))))
}

func mustDefaultPaths(t *testing.T, id string) [][]gocui.Key {
	t.Helper()
	paths, ok := DefaultPaths(id)
	if !ok {
		t.Fatalf("missing default paths for %s", id)
	}
	return paths
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
