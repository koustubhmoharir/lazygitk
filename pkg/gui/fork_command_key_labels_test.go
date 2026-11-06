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

func TestForkCommandKeyLabelsOmitDescendantsHiddenByExplicitLeaf(t *testing.T) {
	key := func(r rune) gocui.Key { return gocui.NewKeyRune(r) }
	quit := &types.Binding{CommandID: "app.quit", Keys: []gocui.Key{key('t')}}
	increase := &types.Binding{CommandID: "diff.context.increase", DefaultCommand: true, CommandPaths: [][]gocui.Key{{key('t'), key('d'), key('+')}}}
	gui := &Gui{}
	gui.setForkCommandBindings([]*types.Binding{quit, increase}, nil)
	if got := gui.commandKeyLabel("diff.context.increase", "files"); got != "" {
		t.Fatalf("unreachable default descendant advertised: %q", got)
	}
	if got := gui.commandKeyLabel("app.quit", "files"); got != "t" {
		t.Fatalf("explicit leaf label=%q", got)
	}

	// An explicit descendant keeps the visible choice, as in dispatch.
	explicit := &types.Binding{CommandID: "diff.context.increase", CommandPaths: [][]gocui.Key{{key('t'), key('d'), key('+')}}}
	gui.setForkCommandBindings([]*types.Binding{quit, explicit}, nil)
	if got := gui.commandKeyLabel("diff.context.increase", "files"); got != "t d +" {
		t.Fatalf("explicit descendant hidden: %q", got)
	}
}

func TestForkCommandKeyLabelsAgreeWithDisplayRoutes(t *testing.T) {
	key := func(r rune) gocui.Key { return gocui.NewKeyRune(r) }
	leaf := &types.Binding{CommandID: "explicit", ViewName: "files", Keys: []gocui.Key{key('g')}}
	// g a is hidden by the explicit leaf g, h is claimed by a custom root, and k survives.
	child := &types.Binding{CommandID: "default", ViewName: "files", DefaultCommand: true, CommandPaths: [][]gocui.Key{{key('g'), key('a')}, {key('h')}, {key('k')}}}
	custom := &types.Binding{CommandID: "custom.x", Keys: []gocui.Key{key('h')}}
	core := []*types.Binding{leaf, child}

	labels := forkCommandLabelsForView(core, []*types.Binding{custom}, "files")
	if got := labels["default"]; got != "k" {
		t.Fatalf("label=%q, want the surviving route k", got)
	}
	for _, binding := range effectiveForkDisplayBindings(append([]*types.Binding{custom}, core...), "files") {
		if binding.CommandID == "default" {
			if labels := binding.KeyLabels(); len(labels) != 1 || labels[0] != "k" {
				t.Fatalf("display shows %v, labels show k", labels)
			}
		}
	}
}
