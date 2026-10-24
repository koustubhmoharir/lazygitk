package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func TestForkEffectiveDisplayPathsRespectLeafAndDescendantChoice(t *testing.T) {
	key := func(r rune) gocui.Key { return gocui.NewKeyRune(r) }
	leaf := &types.Binding{CommandID: "explicit", ViewName: "files", CommandPaths: [][]gocui.Key{{key('g')}}}
	defaultChild := &types.Binding{CommandID: "default", ViewName: "files", DefaultCommand: true, CommandPaths: [][]gocui.Key{{key('g'), key('a')}}}
	got := effectiveForkDisplayBindings([]*types.Binding{leaf, defaultChild}, "files")
	if labels := got[1].KeyLabels(); len(labels) != 0 {
		t.Fatalf("default descendant should be hidden by explicit leaf: %v", labels)
	}

	explicitChild := &types.Binding{CommandID: "child", ViewName: "files", CommandPaths: [][]gocui.Key{{key('g'), key('b')}}}
	got = effectiveForkDisplayBindings([]*types.Binding{leaf, explicitChild}, "files")
	if labels := got[0].KeyLabels(); len(labels) != 1 || labels[0] != "g" {
		t.Fatalf("explicit leaf should remain a visible choice: %v", labels)
	}
	if labels := got[1].KeyLabels(); len(labels) != 1 || labels[0] != "g b" {
		t.Fatalf("explicit descendant should remain visible: %v", labels)
	}
	if leaf.CommandPaths[0][0] != key('g') || len(leaf.Keys) != 0 {
		t.Fatal("display projection mutated its source binding")
	}
}

func TestForkEffectiveDisplayPathsLocalOwnershipAndCustomExactShadow(t *testing.T) {
	key := func(r rune) gocui.Key { return gocui.NewKeyRune(r) }
	global := &types.Binding{CommandID: "global", CommandPaths: [][]gocui.Key{{key('x')}}}
	local := &types.Binding{CommandID: "local", ViewName: "files", CommandPaths: [][]gocui.Key{{key('x')}}}
	copy := effectiveForkDisplayBindings([]*types.Binding{global, local}, "files")
	if len(copy[0].KeyLabels()) != 0 || len(copy[1].KeyLabels()) != 1 {
		t.Fatalf("local route should own duplicate global path: global=%v local=%v", copy[0].KeyLabels(), copy[1].KeyLabels())
	}
	custom := &types.Binding{CommandID: "custom.example", Keys: []gocui.Key{key('x')}}
	clearForkPathsShadowedByCustoms(copy, []*types.Binding{custom})
	if len(copy[1].KeyLabels()) != 0 || len(custom.KeyLabels()) != 1 {
		t.Fatalf("exact custom event should hide core display and retain custom: core=%v custom=%v", copy[1].KeyLabels(), custom.KeyLabels())
	}

	upper := &types.Binding{CommandID: "local.upper", ViewName: "files", CommandPaths: [][]gocui.Key{{key('X')}}}
	clearForkPathsShadowedByCustoms([]*types.Binding{upper}, []*types.Binding{custom})
	if len(upper.KeyLabels()) != 1 {
		t.Fatal("custom exact matching must not introduce case folding")
	}
}

func TestForkEffectiveDisplayRetainsDisabledMetadataAndCustomExactKeys(t *testing.T) {
	key := func(r rune) gocui.Key { return gocui.NewKeyRune(r) }
	custom := &types.Binding{CommandID: "custom.upper", Keys: []gocui.Key{key('X')}, Handler: func() error { return nil }}
	core := &types.Binding{CommandID: "core.lower", ViewName: "files", CommandPaths: [][]gocui.Key{{key('x'), key('a')}}, GetDisabledReason: func() *types.DisabledReason {
		return &types.DisabledReason{Text: "unavailable", Inapplicable: true}
	}}
	got := effectiveForkDisplayBindings([]*types.Binding{custom, core}, "files")
	if len(got[0].Keys) != 1 || got[0].Keys[0] != key('X') || got[0].Handler == nil {
		t.Fatal("custom binding metadata or exact event changed")
	}
	clearForkPathsShadowedByCustoms(got, []*types.Binding{got[0]})
	if labels := got[1].KeyLabels(); len(labels) != 1 || labels[0] != "x a" {
		t.Fatalf("uppercase custom must not shadow normalized lowercase core path: %v", labels)
	}

	lowerCustom := cloneForkBinding(got[0])
	lowerCustom.Keys[0] = key('x')
	clearForkPathsShadowedByCustoms(got, []*types.Binding{lowerCustom})
	if labels := got[1].KeyLabels(); len(labels) != 0 {
		t.Fatalf("exact lowercase custom root should shadow core path: %v", labels)
	}
	if len(core.CommandPaths) != 1 || core.CommandPaths[0][1] != key('a') || custom.Keys[0] != key('X') {
		t.Fatal("effective display handling mutated source bindings")
	}
}

func TestForkDisplayAliasDedupAndEffectivePaletteFallback(t *testing.T) {
	key := func(r rune) gocui.Key { return gocui.NewKeyRune(r) }
	active := &types.Binding{CommandID: "mode.action", CommandPaths: [][]gocui.Key{{key('g'), key('a')}}}
	inactive := &types.Binding{CommandID: "mode.action", CommandActive: func() bool { return false }, CommandPaths: [][]gocui.Key{{key('g'), key('b')}}}
	merged := deduplicateForkDisplayAliases([]*types.Binding{active, inactive})
	if len(merged) != 1 || len(merged[0].KeyLabels()) != 1 || merged[0].KeyLabels()[0] != "g a" {
		t.Fatalf("inactive variant should be hidden and aliases coalesced: %+v", merged)
	}
	palette := &types.Binding{CommandID: "app.command-palette", CommandPaths: [][]gocui.Key{{key('?')}}}
	hint := effectiveModeHint(nil, "mode.action", "Action", "Open palette", []*types.Binding{palette}, style.FgGreen)
	if hint.key != "?" || hint.description != "Open palette" {
		t.Fatalf("expected effective palette path fallback: %+v", hint)
	}
	hint = effectiveModeHint(nil, "mode.action", "Action", "Open palette", nil, style.FgGreen)
	if hint.key != "" || appendModeHint(nil, hint) != nil {
		t.Fatal("unbound action and palette should not produce an empty-key hint")
	}
}

func TestForkHintContextExcludesCompiledGlobalCoreFromModal(t *testing.T) {
	globalCore := &types.Binding{CommandID: "core.action"}
	globalCustom := &types.Binding{CommandID: "custom.action"}
	if forkHintBindingInContext(globalCore, "menu") {
		t.Fatal("compiled global core action should not appear in popup hints")
	}
	if !forkHintBindingInContext(globalCustom, "menu") {
		t.Fatal("native global custom action should remain visible in popup hints")
	}
}

func TestForkEffectiveDisplayForeignCustomDoesNotClaimCurrentViewPath(t *testing.T) {
	key := func(r rune) gocui.Key { return gocui.NewKeyRune(r) }
	foreignCustom := &types.Binding{CommandID: "custom.files", ViewName: "files", Keys: []gocui.Key{key('x')}}
	localCommit := &types.Binding{CommandID: "commit.local", ViewName: "commits", CommandPaths: [][]gocui.Key{{key('x')}}}
	global := &types.Binding{CommandID: "app.global", CommandPaths: [][]gocui.Key{{key('y')}}}
	got := effectiveForkDisplayBindings([]*types.Binding{foreignCustom, localCommit, global}, "commits")
	if labels := got[1].KeyLabels(); len(labels) != 1 || labels[0] != "x" {
		t.Fatalf("foreign custom should not hide current-view path: %v", labels)
	}
	if labels := got[2].KeyLabels(); len(labels) != 1 || labels[0] != "y" {
		t.Fatalf("unclaimed global path should remain visible: %v", labels)
	}
	if foreignCustom.Keys[0] != key('x') || localCommit.CommandPaths[0][0] != key('x') || global.CommandPaths[0][0] != key('y') {
		t.Fatal("display projection mutated source bindings")
	}
}

func TestForkHintAliasesPreferLocalBindingAndKeepLocalOrder(t *testing.T) {
	localFirst := &types.Binding{CommandID: "local.first", ViewName: "commits"}
	localAlias := &types.Binding{CommandID: "shared", ViewName: "commits"}
	globalAlias := &types.Binding{CommandID: "shared"}
	globalLast := &types.Binding{CommandID: "global.last"}
	ordered := localFirstForkHintBindings([]*types.Binding{globalAlias, localFirst, globalLast, localAlias})
	if len(ordered) != 4 || ordered[0] != localFirst || ordered[1] != localAlias || ordered[2] != globalAlias || ordered[3] != globalLast {
		t.Fatalf("expected stable local-first ordering: %#v", ordered)
	}
	deduplicated := deduplicateForkDisplayAliases(ordered)
	if len(deduplicated) != 3 || deduplicated[1].ViewName != "commits" {
		t.Fatalf("local alias should win over global alias: %#v", deduplicated)
	}
}
