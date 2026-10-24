package controllers

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func TestCommandPaletteBindingsPartitionsAndFilters(t *testing.T) {
	bindings := []*types.Binding{
		{CommandID: "local", ViewName: "files", Description: "Local"},
		{CommandID: "global", ViewName: "", Description: "Global"},
		{CommandID: "tagged-global", ViewName: "other", Tag: "global", Description: "Tagged global"},
		{CommandID: "navigation", ViewName: "files", Tag: "navigation", Description: "Navigation"},
		{CommandID: "other-view", ViewName: "branches", Description: "Other view"},
		{CommandID: "no-description", ViewName: "files"},
		{CommandID: "other-nav", ViewName: "branches", Tag: "navigation", Description: "Other navigation"},
	}

	local, global, navigation := commandPaletteBindings(bindings, "files")
	if got, want := bindingIDs(local), []string{"local"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("local IDs = %v, want %v", got, want)
	}
	if got, want := bindingIDs(global), []string{"global", "tagged-global"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("global IDs = %v, want %v", got, want)
	}
	if got, want := bindingIDs(navigation), []string{"navigation"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("navigation IDs = %v, want %v", got, want)
	}
}

func TestCommandPaletteBindingsDeduplicatesAliasesWithoutMutatingBindings(t *testing.T) {
	sharedBacking := []gocui.Key{gocui.NewKeyRune('a'), gocui.NewKeyRune('b'), gocui.NewKeyRune('c')}
	localKeys := sharedBacking[:2]
	globalKeys := sharedBacking[1:]
	localHandlerCalls, globalHandlerCalls := 0, 0
	localDisabledCalls, globalDisabledCalls := 0, 0
	local := &types.Binding{
		CommandID: "same.action", ViewName: "files", Description: "Local alias", Keys: localKeys,
		Handler:           func() error { localHandlerCalls++; return nil },
		GetDisabledReason: func() *types.DisabledReason { localDisabledCalls++; return nil },
	}
	global := &types.Binding{
		CommandID: "same.action", Description: "Global alias", Keys: globalKeys,
		Handler:           func() error { globalHandlerCalls++; return nil },
		GetDisabledReason: func() *types.DisabledReason { globalDisabledCalls++; return nil },
	}
	originalShared := append([]gocui.Key(nil), sharedBacking...)

	localSection, globalSection, _ := commandPaletteBindings([]*types.Binding{global, local}, "files")
	if len(localSection) != 1 || len(globalSection) != 0 {
		t.Fatalf("section lengths = (%d, %d), want (1, 0)", len(localSection), len(globalSection))
	}
	merged := localSection[0]
	if merged == local {
		t.Fatal("deduplicated binding aliases the original binding")
	}
	if got, want := merged.Description, "Local alias"; got != want {
		t.Fatalf("description = %q, want local handler owner %q", got, want)
	}
	if err := merged.Handler(); err != nil {
		t.Fatalf("handler returned %v, want nil", err)
	}
	if localHandlerCalls != 1 || globalHandlerCalls != 0 {
		t.Fatalf("handler calls = (%d, %d), want local only", localHandlerCalls, globalHandlerCalls)
	}
	if merged.GetDisabledReason() != nil {
		t.Fatal("local disabled check returned an unexpected reason")
	}
	if localDisabledCalls != 1 || globalDisabledCalls != 0 {
		t.Fatalf("disabled-check calls = (%d, %d), want local only", localDisabledCalls, globalDisabledCalls)
	}
	if got, want := merged.Keys, []gocui.Key{gocui.NewKeyRune('a'), gocui.NewKeyRune('b'), gocui.NewKeyRune('c')}; !reflect.DeepEqual(got, want) {
		t.Fatalf("merged keys = %v, want %v", got, want)
	}
	merged.Keys[0] = gocui.NewKeyRune('z')
	if !reflect.DeepEqual(local.Keys, []gocui.Key{gocui.NewKeyRune('a'), gocui.NewKeyRune('b')}) || !reflect.DeepEqual(global.Keys, []gocui.Key{gocui.NewKeyRune('b'), gocui.NewKeyRune('c')}) {
		t.Fatalf("original keys changed: local %v, global %v", local.Keys, global.Keys)
	}
	if !reflect.DeepEqual(sharedBacking, originalShared) {
		t.Fatalf("shared key backing array changed: got %v, want %v", sharedBacking, originalShared)
	}
}

func TestCommandPaletteBindingsKeepsDistinctEqualLabelsAndMissingIDs(t *testing.T) {
	bindings := []*types.Binding{
		{CommandID: "custom.user.alpha", ViewName: "files", Description: "Same label"},
		{CommandID: "custom.user.beta", Description: "Same label"},
		{ViewName: "files", Description: "Missing identity"},
		{Description: "Missing identity"},
	}
	local, global, _ := commandPaletteBindings(bindings, "files")
	if len(local) != 2 || len(global) != 2 {
		t.Fatalf("section lengths = (%d, %d), want (2, 2)", len(local), len(global))
	}
	if local[0].GetCommandID() == global[0].GetCommandID() || local[0].Description != global[0].Description {
		t.Fatalf("distinct custom IDs with equal descriptions were not preserved: local=%v global=%v", bindingIDs(local), bindingIDs(global))
	}
	if local[1].GetCommandID() != "" || global[1].GetCommandID() != "" {
		t.Fatalf("equal descriptions or custom alias ID treated specially: local=%v global=%v", bindingIDs(local), bindingIDs(global))
	}
}

func TestCommandPaletteBindingsUsesLiveCommandIDsForDeduplication(t *testing.T) {
	firstID, secondID := "shared", "shared"
	first := &types.Binding{CommandID: "static-first", CommandIDFunc: func() string { return firstID }, ViewName: "files", Description: "First"}
	second := &types.Binding{CommandID: "static-second", CommandIDFunc: func() string { return secondID }, Description: "Second"}
	local, global, _ := commandPaletteBindings([]*types.Binding{first, second}, "files")
	if len(local) != 1 || len(global) != 0 {
		t.Fatalf("initial section lengths = (%d, %d), want (1, 0)", len(local), len(global))
	}
	secondID = "second"
	local, global, _ = commandPaletteBindings([]*types.Binding{first, second}, "files")
	if len(local) != 1 || len(global) != 1 {
		t.Fatalf("changed-ID section lengths = (%d, %d), want (1, 1)", len(local), len(global))
	}
}

func TestCommandPaletteBindingsMergesAndCopiesCommandPaths(t *testing.T) {
	path := []gocui.Key{gocui.NewKeyRune('d'), gocui.NewKeyRune('w')}
	first := &types.Binding{CommandID: "diff", ViewName: "files", Description: "Diff", CommandPaths: [][]gocui.Key{path}}
	second := &types.Binding{CommandID: "diff", Description: "Diff alias", CommandPaths: [][]gocui.Key{path, {gocui.NewKeyRune('d'), gocui.NewKeyRune('p')}}}
	local, _, _ := commandPaletteBindings([]*types.Binding{first, second}, "files")
	if len(local) != 1 || len(local[0].CommandPaths) != 2 {
		t.Fatalf("merged paths = %#v, want two unique paths", local)
	}
	local[0].CommandPaths[0][0] = gocui.NewKeyRune('z')
	if path[0] != gocui.NewKeyRune('d') || first.CommandPaths[0][0] != gocui.NewKeyRune('d') || second.CommandPaths[0][0] != gocui.NewKeyRune('d') {
		t.Fatalf("source command paths were mutated: first=%v second=%v", first.CommandPaths, second.CommandPaths)
	}
}

func TestCommandPaletteBindingsSkipsForeignDescriptionCallbacks(t *testing.T) {
	calls := 0
	foreign := &types.Binding{ViewName: "branches", DescriptionFunc: func() string { calls++; return "foreign" }}
	commandPaletteBindings([]*types.Binding{foreign}, "files")
	if calls != 0 {
		t.Fatalf("foreign description callback called %d times", calls)
	}
}

func TestCommandPaletteMenuItemCarriesBindingAndDispatchesGateway(t *testing.T) {
	id, verb := "dynamic.id", "dynamic-verb"
	disabledCalls, handlerCalls, gatewayCalls := 0, 0, 0
	disabled := &types.DisabledReason{Text: "not available"}
	keys := []gocui.Key{gocui.NewKeyRune('x'), gocui.NewKeyRune('X')}
	binding := &types.Binding{
		CommandID: "static.id", CommandIDFunc: func() string { return id },
		Verb: "static-verb", VerbFunc: func() string { return verb },
		ViewName: "files", Description: "Run", Keys: keys, OpensMenu: true, Tooltip: "Details",
		Handler:           func() error { handlerCalls++; return nil },
		GetDisabledReason: func() *types.DisabledReason { disabledCalls++; return disabled },
	}
	section := &types.MenuSection{Title: "Local"}
	dispatchErr := errors.New("gateway error")
	item := commandPaletteMenuItem(binding, section, "Palette tooltip", func(got *types.Binding) error {
		gatewayCalls++
		if got != binding {
			t.Fatalf("gateway received binding %p, want %p", got, binding)
		}
		return dispatchErr
	})
	if item.Binding != binding || item.CommandID != binding.CommandID || item.CommandIDFunc == nil || item.Verb != binding.Verb || item.VerbFunc == nil {
		t.Fatal("menu item did not preserve binding identity and verb metadata")
	}
	if item.GetCommandID() != id || item.GetVerb() != verb {
		t.Fatalf("live metadata = (%q, %q), want (%q, %q)", item.GetCommandID(), item.GetVerb(), id, verb)
	}
	id, verb = "changed.id", "changed-verb"
	if item.GetCommandID() != id || item.GetVerb() != verb {
		t.Fatalf("metadata callbacks not live: got (%q, %q)", item.GetCommandID(), item.GetVerb())
	}
	if !item.OpensMenu || item.Section != section || item.Tooltip != "Palette tooltip" || !reflect.DeepEqual(item.Keys, keys) {
		t.Fatalf("menu metadata not carried: %+v", item)
	}
	if item.DisabledReason != disabled || disabledCalls != 1 {
		t.Fatalf("disabled reason/calls = (%p, %d), want (%p, 1)", item.DisabledReason, disabledCalls, disabled)
	}
	if err := item.OnPress(); !errors.Is(err, dispatchErr) {
		t.Fatalf("OnPress error = %v, want gateway error %v", err, dispatchErr)
	}
	if gatewayCalls != 1 || handlerCalls != 0 {
		t.Fatalf("gateway/handler calls = (%d, %d), want (1, 0)", gatewayCalls, handlerCalls)
	}
}

func TestCommandPaletteMenuItemWithoutHandlerDoesNotDispatch(t *testing.T) {
	gatewayCalls := 0
	item := commandPaletteMenuItem(&types.Binding{Description: "No handler"}, nil, "", func(*types.Binding) error {
		gatewayCalls++
		return nil
	})
	if err := item.OnPress(); err != nil {
		t.Fatalf("OnPress returned %v, want nil", err)
	}
	if gatewayCalls != 0 {
		t.Fatalf("gateway called %d times, want zero", gatewayCalls)
	}
}

func TestCommandPaletteMenuItemAllowsUnboundDescribedAction(t *testing.T) {
	handlerCalls, gatewayCalls := 0, 0
	binding := &types.Binding{
		CommandID: "unbound.action", Description: "Available without a shortcut",
		Handler: func() error { handlerCalls++; return nil },
	}
	item := commandPaletteMenuItem(binding, nil, "", func(got *types.Binding) error {
		gatewayCalls++
		if got != binding {
			t.Fatalf("gateway received binding %p, want %p", got, binding)
		}
		return got.Handler()
	})
	if item.Label != binding.Description || len(item.Keys) != 0 {
		t.Fatalf("unbound menu item = label %q, keys %v", item.Label, item.Keys)
	}
	if err := item.OnPress(); err != nil {
		t.Fatalf("OnPress returned %v", err)
	}
	if gatewayCalls != 1 || handlerCalls != 1 {
		t.Fatalf("gateway/handler calls = (%d, %d), want (1, 1)", gatewayCalls, handlerCalls)
	}
}

func bindingIDs(bindings []*types.Binding) []string {
	ids := make([]string, len(bindings))
	for i, binding := range bindings {
		ids[i] = binding.GetCommandID()
	}
	return ids
}
