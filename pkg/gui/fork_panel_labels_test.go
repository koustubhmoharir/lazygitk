package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func TestForkPanelLabelsFollowCurrentOwnedRoutes(t *testing.T) {
	files := &types.Binding{CommandID: "navigation.focus-panel.files", Keys: []gocui.Key{gocui.NewKeyRune('2')}, DefaultCommand: true}
	inspect := &types.Binding{CommandID: "navigation.focus-diff", ViewName: "files", Keys: []gocui.Key{gocui.NewKeyRune('v')}, DefaultCommand: true}
	core := []*types.Binding{files, inspect}
	got := forkJumpLabelsForView(core, nil, "files")
	if got[files.CommandID] != "2" || got[inspect.CommandID] != "v" {
		t.Fatalf("Files border labels=%v", got)
	}
	got = forkJumpLabelsForView(core, nil, "localBranches")
	if got[inspect.CommandID] != "" || got[files.CommandID] != "2" {
		t.Fatalf("branch inspection advertised Files' focus shortcut: %v", got)
	}
	if got := forkJumpLabelsForView(core, nil, "commitMessage"); len(got) != 0 {
		t.Fatalf("editor advertised normal-pane shortcuts: %v", got)
	}
	configured := &types.Binding{ViewName: "files", CommandID: "configured-action", Keys: []gocui.Key{gocui.NewKeyRune('2')}}
	got = forkJumpLabelsForView(append([]*types.Binding{configured}, core...), nil, "files")
	if got[files.CommandID] != "" {
		t.Fatalf("shadowed jump still advertised: %v", got)
	}
	if files.Keys[0] != gocui.NewKeyRune('2') {
		t.Fatal("display mutated source keys")
	}
}

func TestForkPanelLabelsPreserveExplicitPathsAndCustomScope(t *testing.T) {
	inspect := &types.Binding{CommandID: "navigation.focus-diff", ViewName: "files", CommandPaths: [][]gocui.Key{{gocui.NewKeyRune('g'), gocui.NewKeyRune('v')}}}
	leaf := &types.Binding{CommandID: "configured-leaf", Keys: []gocui.Key{gocui.NewKeyRune('g')}}
	core := []*types.Binding{inspect, leaf}
	foreign := &types.Binding{ViewName: "commits", Keys: []gocui.Key{gocui.NewKeyRune('g')}}
	if got := forkJumpLabelsForView(core, []*types.Binding{foreign}, "files"); got[inspect.CommandID] != "g v" {
		t.Fatalf("explicit descendant or foreign custom scope lost: %v", got)
	}
	global := &types.Binding{Keys: []gocui.Key{gocui.NewKeyRune('g')}}
	if got := forkJumpLabelsForView(core, []*types.Binding{global}, "files"); got[inspect.CommandID] != "" {
		t.Fatalf("custom-shadowed path advertised: %v", got)
	}
	global.Keys = []gocui.Key{gocui.NewKeyRune('G')}
	if got := forkJumpLabelsForView(core, []*types.Binding{global}, "files"); got[inspect.CommandID] != "g v" {
		t.Fatalf("exact custom key was incorrectly folded: %v", got)
	}
}
