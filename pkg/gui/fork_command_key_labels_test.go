package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func TestForkCommandKeyLabelsPreferLocalThenGlobalAndShortest(t *testing.T) {
	local := &types.Binding{ViewName: "files", CommandID: "local.cmd", Keys: []gocui.Key{gocui.NewKeyRune('v')}}
	global := &types.Binding{CommandID: "local.cmd", CommandPaths: [][]gocui.Key{{gocui.NewKeyRune('g'), gocui.NewKeyRune('x')}}}
	onlyGlobal := &types.Binding{CommandID: "global.cmd", Keys: []gocui.Key{gocui.NewKeyRune('q')}}
	core := []*types.Binding{global, local, onlyGlobal}

	gui := &Gui{}
	gui.setForkCommandBindings(core, nil)
	if got := gui.commandKeyLabel("local.cmd", "files"); got != "v" {
		t.Fatalf("local route not preferred over global: %q", got)
	}
	if got := gui.commandKeyLabel("local.cmd", ""); got != "g x" {
		t.Fatalf("global route not used for global view: %q", got)
	}
	if got := gui.commandKeyLabel("global.cmd", "files"); got != "q" {
		t.Fatalf("global fallback not used for local view: %q", got)
	}
	if got := gui.commandKeyLabel("unknown.cmd", "files"); got != "" {
		t.Fatalf("unknown command labelled: %q", got)
	}
	if got := gui.commandKeyLabel("unknown.cmd", ""); got != "" {
		t.Fatalf("unknown global command labelled: %q", got)
	}

	shortest := &types.Binding{CommandID: "shortest.cmd", CommandPaths: [][]gocui.Key{
		{gocui.NewKeyRune('t'), gocui.NewKeyRune('a'), gocui.NewKeyRune('b')},
		{gocui.NewKeyRune('g'), gocui.NewKeyRune('v')},
	}}
	tie := &types.Binding{CommandID: "tie.cmd", CommandPaths: [][]gocui.Key{
		{gocui.NewKeyRune('h'), gocui.NewKeyRune('1')},
		{gocui.NewKeyRune('h'), gocui.NewKeyRune('2')},
	}}
	gui.setForkCommandBindings([]*types.Binding{shortest, tie}, nil)
	if got := gui.commandKeyLabel("shortest.cmd", "files"); got != "g v" {
		t.Fatalf("shortest route not chosen: %q", got)
	}
	if got := gui.commandKeyLabel("tie.cmd", "files"); got != "h 1" {
		t.Fatalf("tie not resolved by walk order: %q", got)
	}
}

func TestForkCommandKeyLabelsSkipCustomClaimedRoots(t *testing.T) {
	inspect := &types.Binding{CommandID: "inspect", CommandPaths: [][]gocui.Key{{gocui.NewKeyRune('g'), gocui.NewKeyRune('v')}}}
	other := &types.Binding{CommandID: "other", ViewName: "files", CommandPaths: [][]gocui.Key{{gocui.NewKeyRune('g'), gocui.NewKeyRune('x')}}}
	custom := &types.Binding{Keys: []gocui.Key{gocui.NewKeyRune('g')}}

	gui := &Gui{}
	gui.setForkCommandBindings([]*types.Binding{inspect, other}, []*types.Binding{custom})
	if got := gui.commandKeyLabel("inspect", "files"); got != "" {
		t.Fatalf("custom-claimed root still advertised: %q", got)
	}
	if got := gui.commandKeyLabel("other", "files"); got != "" {
		t.Fatalf("custom-claimed local root still advertised: %q", got)
	}
}

func TestForkCommandKeyLabelsModalViewsUseNativeKeys(t *testing.T) {
	confirm := &types.Binding{CommandID: "confirm.cmd", ViewName: "commitMessage", Keys: []gocui.Key{gocui.NewKeyRune('e')}}
	if got := forkCommandLabelsForView([]*types.Binding{confirm}, nil, "commitMessage")["confirm.cmd"]; got != "e" {
		t.Fatalf("modal native key label=%q", got)
	}
}

func TestForkCommandKeyLabelsRecomputeAfterRebinding(t *testing.T) {
	first := &types.Binding{CommandID: "cmd", ViewName: "files", Keys: []gocui.Key{gocui.NewKeyRune('a')}}
	second := &types.Binding{CommandID: "cmd", ViewName: "files", Keys: []gocui.Key{gocui.NewKeyRune('b')}}
	gui := &Gui{}
	gui.setForkCommandBindings([]*types.Binding{first}, nil)
	if got := gui.commandKeyLabel("cmd", "files"); got != "a" {
		t.Fatalf("initial label=%q", got)
	}
	gui.setForkCommandBindings([]*types.Binding{second}, nil)
	if got := gui.commandKeyLabel("cmd", "files"); got != "b" {
		t.Fatalf("stale label after rebinding: %q", got)
	}
}
