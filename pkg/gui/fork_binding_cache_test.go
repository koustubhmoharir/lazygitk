package gui

import (
	"fmt"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

// forkRecoveringApplicability makes applicability callbacks safe offline, where
// the stub helpers leave some dependencies nil. A callback that panics counts as
// having no reason and as active. Both the cached and the rebuilt paths see the
// same callbacks, so their outputs can be compared.
func forkRecoveringApplicability(bindings []*types.Binding) {
	for _, binding := range bindings {
		if reason := binding.GetDisabledReason; reason != nil {
			binding.GetDisabledReason = func() (result *types.DisabledReason) {
				defer func() {
					if recover() != nil {
						result = nil
					}
				}()
				return reason()
			}
		}
		if active := binding.CommandActive; active != nil {
			binding.CommandActive = func() (result bool) {
				defer func() {
					if recover() != nil {
						result = true
					}
				}()
				return active()
			}
		}
	}
}

// forkSetStaticView makes the context for view the current static context. A
// side context stays underneath, as it does in the app, so that the side
// context is never the default one.
func forkSetStaticView(t testing.TB, gui *Gui, view string) {
	t.Helper()
	for _, ctx := range gui.State.Contexts.Flatten() {
		if ctx.GetViewName() == view {
			gui.State.ContextMgr.ContextStack = []types.Context{gui.State.Contexts.Files, ctx}
			if view == "files" {
				gui.State.ContextMgr.ContextStack = []types.Context{ctx}
			}
			return
		}
	}
	t.Fatalf("no context for view %s", view)
}

// forkBindingRoutes describes what the options bar and palette read from each
// binding: its identity, its scope, and the routes it displays.
func forkBindingRoutes(bindings []*types.Binding) []string {
	rows := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		paths := make([]string, 0, len(binding.CommandPaths))
		for _, path := range binding.CommandPaths {
			paths = append(paths, types.CommandPathLabel(path))
		}
		rows = append(rows, fmt.Sprintf("%s|%s|%v|%v|%v", binding.GetCommandID(), binding.ViewName, binding.KeyLabels(), paths, binding.DefaultCommand))
	}
	return rows
}

func TestForkBindingCacheReturnsRegisteredBindingsForNonCommandView(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	forkSetStaticView(t, gui, "options")
	bindings, custom, mouse := gui.initialBindingsWithCustomCommands()
	gui.storeForkBindingCache(bindings, custom, mouse)
	registered := append(append([]*types.Binding(nil), custom...), bindings...)

	got, gotMouse := gui.GetInitialKeybindingsWithCustomCommands()
	assert.Len(t, got, len(registered))
	for i := range registered {
		assert.Same(t, registered[i], got[i], "binding %d is not the registered object", i)
	}
	assert.Equal(t, mouse, gotMouse)
}

func TestForkBindingCacheFallsBackWhenViewChanges(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	forkSetStaticView(t, gui, "files")
	bindings, custom, mouse := gui.initialBindingsWithCustomCommands()
	gui.storeForkBindingCache(bindings, custom, mouse)

	forkSetStaticView(t, gui, "options")
	got, _ := gui.GetInitialKeybindingsWithCustomCommands()
	assert.NotEmpty(t, got)
	for _, binding := range got {
		for _, registered := range bindings {
			if binding == registered {
				t.Fatalf("view mismatch returned the cached binding %s instead of rebuilding", binding.GetCommandID())
			}
		}
	}
}

func TestForkBindingCacheClearedWithControllers(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	forkSetStaticView(t, gui, "options")
	bindings, custom, mouse := gui.initialBindingsWithCustomCommands()
	gui.storeForkBindingCache(bindings, custom, mouse)

	gui.clearForkBindingCache()

	got, _ := gui.GetInitialKeybindingsWithCustomCommands()
	assert.NotEmpty(t, got)
	assert.NotSame(t, bindings[0], got[0])
}

// The structure is cached, but applicability is evaluated on every call, so a
// route follows the live state of an explicit leaf that shadows a default
// descendant.
func TestForkBindingCacheKeepsApplicabilityLive(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	forkSetStaticView(t, gui, "files")
	g, a := gocui.NewKeyRune('g'), gocui.NewKeyRune('a')
	leafActive := true
	leaf := &types.Binding{
		CommandID: "explicit.leaf", ViewName: "files", Keys: []gocui.Key{g},
		CommandActive: func() bool { return leafActive },
	}
	child := &types.Binding{
		CommandID: "default.child", ViewName: "files", DefaultCommand: true,
		CommandPaths: [][]gocui.Key{{g, a}},
	}
	reasonCalls := 0
	disabled := false
	toggle := &types.Binding{
		CommandID: "default.toggle", ViewName: "files", DefaultCommand: true,
		Keys: []gocui.Key{gocui.NewKeyRune('h')},
		GetDisabledReason: func() *types.DisabledReason {
			reasonCalls++
			if disabled {
				return &types.DisabledReason{Text: "off", Inapplicable: true}
			}
			return nil
		},
	}
	gui.storeForkBindingCache([]*types.Binding{leaf, child, toggle}, nil, nil)

	childRoutes := func() []string {
		got, _ := gui.GetInitialKeybindingsWithCustomCommands()
		for _, binding := range got {
			if binding.CommandID == "default.child" {
				var labels []string
				for _, path := range binding.CommandPaths {
					labels = append(labels, types.CommandPathLabel(path))
				}
				return labels
			}
		}
		t.Fatal("default.child missing from result")
		return nil
	}
	assert.Empty(t, childRoutes(), "an active explicit leaf hides its default descendant")

	leafActive = false
	assert.Equal(t, []string{"g a"}, childRoutes(), "the descendant is routed once the leaf is inactive, without a reset")

	got, _ := gui.GetInitialKeybindingsWithCustomCommands()
	var live *types.Binding
	for _, binding := range got {
		if binding.CommandID == "default.toggle" {
			live = binding
		}
	}
	if live == nil {
		t.FailNow()
	}
	before := reasonCalls
	assert.False(t, live.IsDisabled())
	disabled = true
	assert.True(t, live.IsDisabled(), "the cached copy reports the current disabled reason")
	assert.Equal(t, before+2, reasonCalls, "the disabled reason is evaluated on each read, not at reset")
}

// The cache keeps the command tree from reset. Its output must match a rebuild
// on the same bindings for each command view.
func TestForkBindingCacheMatchesRebuildForCommandViews(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	bindings, _ := gui.GetInitialKeybindings()
	forkRecoveringApplicability(bindings)

	for _, view := range []string{"files", "commits", "localBranches", "main"} {
		t.Run(view, func(t *testing.T) {
			forkSetStaticView(t, gui, view)
			gui.storeForkBindingCache(bindings, nil, nil)

			cached, _ := gui.GetInitialKeybindingsWithCustomCommands()
			rebuilt := effectiveForkDisplayBindings(bindings, view)
			assert.Equal(t, forkBindingRoutes(rebuilt), forkBindingRoutes(cached))
			assert.NotEmpty(t, forkBindingRoutes(cached))
		})
	}
}

// BenchmarkForkOptionsBarBindings compares what the options bar pays per frame
// in the files view: a full rebuild of the bindings and their projection, and
// the cached path. Callbacks are made recovering in both, so the rebuild's
// wrappers add a little to its cost.
func BenchmarkForkOptionsBarBindings(b *testing.B) {
	gui := newForkKeymapTestGui(b)
	forkSetStaticView(b, gui, "files")
	var sink []*types.Binding

	b.Run("rebuild", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			bindings, custom, _ := gui.initialBindingsWithCustomCommands()
			forkRecoveringApplicability(bindings)
			forkRecoveringApplicability(custom)
			all := append(custom, bindings...)
			sink = effectiveForkDisplayBindings(all, "files")
		}
	})

	b.Run("cached", func(b *testing.B) {
		bindings, custom, mouse := gui.initialBindingsWithCustomCommands()
		forkRecoveringApplicability(bindings)
		forkRecoveringApplicability(custom)
		gui.storeForkBindingCache(bindings, custom, mouse)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sink, _ = gui.GetInitialKeybindingsWithCustomCommands()
		}
	})
	_ = sink
}
