package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/keymap"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

type forkPrefixMenuHarness struct {
	menus       []types.CreateMenuOptions
	currentMenu bool
	restored    bool
	valid       bool
	invoked     []*types.Binding
}

func (h *forkPrefixMenuHarness) flow() *forkPrefixFlow {
	return &forkPrefixFlow{
		valid:       func() bool { return h.valid },
		currentMenu: func() bool { return h.currentMenu },
		restored:    func() bool { return h.restored },
		openMenu: func(opts types.CreateMenuOptions) error {
			h.menus = append(h.menus, opts)
			h.currentMenu = true
			h.restored = false
			return nil
		},
		invoke: func(binding *types.Binding) error {
			h.invoked = append(h.invoked, binding)
			return nil
		},
	}
}

func forkPrefixBinding(label string) *types.Binding {
	return &types.Binding{CommandID: label, Description: label}
}

func forkPrefixAccept(h *forkPrefixMenuHarness, opts types.CreateMenuOptions, item *types.MenuItem) error {
	if !opts.BeforeAccept() {
		return nil
	}
	// Native menus remove their popup before running the accepted item's callback.
	h.currentMenu = false
	h.restored = true
	return item.OnPress()
}

func forkPrefixCancel(h *forkPrefixMenuHarness, opts types.CreateMenuOptions) error {
	if !opts.BeforeAccept() {
		return nil
	}
	// MenuContext.OnMenuPress(nil) pops after BeforeAccept, then calls OnCancel.
	h.currentMenu = false
	h.restored = true
	if opts.OnCancel != nil {
		return opts.OnCancel()
	}
	return nil
}

func TestForkPrefixFlowAcceptsOnlyAfterPopupPopAndRestoresCapturedLeaf(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	flow := h.flow()
	leaf := forkPrefixBinding("leaf")
	tree := &keymap.CommandTree{Children: []keymap.CommandEdge{{Key: gocui.NewKeyRune('x'), Tree: &keymap.CommandTree{Bindings: []*types.Binding{leaf}}}}}
	if err := flow.run(tree, nil, nil); err != nil {
		t.Fatal(err)
	}
	root := h.menus[0]
	if !root.BeforeAccept() {
		t.Fatal("active menu rejected acceptance")
	}
	if len(h.invoked) != 0 {
		t.Fatal("leaf invoked before native popup was popped")
	}
	h.currentMenu, h.restored = false, true
	if err := root.Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(h.menus) != 1 || len(h.invoked) != 1 || h.invoked[0] != leaf {
		t.Fatalf("after popup pop, continuation should invoke captured leaf once: menus=%d invoked=%v", len(h.menus), h.invoked)
	}
}

func TestForkPrefixFlowRejectsStaleCallbacksAndConsumesTokenOnce(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	flow := h.flow()
	leaf := forkPrefixBinding("leaf")
	tree := &keymap.CommandTree{Bindings: []*types.Binding{leaf}, Children: []keymap.CommandEdge{{Key: gocui.NewKeyRune('x'), Tree: &keymap.CommandTree{Bindings: []*types.Binding{forkPrefixBinding("other")}}}}}
	if err := flow.run(tree, nil, nil); err != nil {
		t.Fatal(err)
	}
	old := h.menus[0]
	if err := old.Items[0].OnPress(); err != nil { // no BeforeAccept token
		t.Fatal(err)
	}
	if len(h.invoked) != 0 {
		t.Fatal("callback ran without BeforeAccept")
	}
	if !old.BeforeAccept() {
		t.Fatal("first acceptance rejected")
	}
	// Replacing the popup invalidates the previous frame and its armed callback.
	flow.invalidate()
	if err := flow.run(tree, nil, nil); err != nil {
		t.Fatal(err)
	}
	// An armed callback from the old popup must not run while its replacement
	// is open and focus has not yet returned to the source.
	if err := old.Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(h.invoked) != 0 {
		t.Fatal("stale callback invoked after popup replacement")
	}
	newMenu := h.menus[1]
	if !newMenu.BeforeAccept() {
		t.Fatal("replacement menu rejected acceptance")
	}
	h.currentMenu = false
	h.restored = true
	if err := newMenu.Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := newMenu.Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(h.invoked) != 1 {
		t.Fatalf("one-use token invoked %d times, want 1", len(h.invoked))
	}
}

func TestForkPrefixFlowInvalidationsAndFrozenMetadata(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	flow := h.flow()
	label, tooltip, id, verb := "before", "tip before", "id before", "verb before"
	binding := &types.Binding{
		CommandIDFunc: func() string { return id }, VerbFunc: func() string { return verb },
		DescriptionFunc: func() string { return label }, TooltipFunc: func() string { return tooltip },
	}
	tree := freezeForkCommandTree(&keymap.CommandTree{Bindings: []*types.Binding{binding}})
	label, tooltip, id, verb = "after", "tip after", "id after", "verb after"
	if tree.Bindings[0].Description != "before" || tree.Bindings[0].Tooltip != "tip before" || tree.Bindings[0].CommandID != "id before" || tree.Bindings[0].Verb != "verb before" {
		t.Fatalf("frozen metadata changed: %+v", tree.Bindings[0])
	}
	if tree.Bindings[0].DescriptionFunc != nil || tree.Bindings[0].TooltipFunc != nil || tree.Bindings[0].CommandIDFunc != nil || tree.Bindings[0].VerbFunc != nil {
		t.Fatal("frozen metadata retained dynamic callbacks")
	}
	if err := flow.run(&keymap.CommandTree{Bindings: []*types.Binding{forkPrefixBinding("a"), forkPrefixBinding("b")}}, nil, nil); err != nil {
		t.Fatal(err)
	}
	menu := h.menus[0]
	h.currentMenu = false
	if menu.BeforeAccept() {
		t.Fatal("acceptance succeeded after popup ceased to be current")
	}
	h.currentMenu = true
	if !menu.BeforeAccept() { // a transient failed check must not poison the live frame
		t.Fatal("active frame failed acceptance after current menu was restored")
	}
	flow.invalidate()
	if menu.BeforeAccept() {
		t.Fatal("acceptance succeeded after invalidation")
	}
}

func TestForkApplicableBindingsPreservesExplicitAndActionablePreconditions(t *testing.T) {
	defaultCollision := &types.Binding{DefaultCommand: true, GetDisabledReason: func() *types.DisabledReason { return &types.DisabledReason{Inapplicable: true} }}
	explicit := &types.Binding{}
	if got := forkApplicableBindings([]*types.Binding{defaultCollision, explicit}); len(got) != 1 || got[0] != explicit {
		t.Fatalf("explicit collision did not take precedence: %v", got)
	}
	guard := &types.Binding{DefaultCommand: true, GetDisabledReason: func() *types.DisabledReason { return &types.DisabledReason{AllowFurtherDispatching: true} }}
	if got := forkApplicableBindings([]*types.Binding{defaultCollision, guard}); len(got) != 0 {
		t.Fatalf("inapplicable defaults remained selectable: %v", got)
	}
	precondition := &types.Binding{DefaultCommand: true, GetDisabledReason: func() *types.DisabledReason { return &types.DisabledReason{Text: "needs clean tree"} }}
	if got := forkApplicableBindings([]*types.Binding{precondition}); len(got) != 1 || got[0] != precondition {
		t.Fatal("actionable precondition failure was removed")
	}
}

func TestForkPrefixFlowNestedCancelReturnsToParentThenRootCancelEnds(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	flow := h.flow()
	leaf := forkPrefixBinding("leaf")
	child := &keymap.CommandTree{Children: []keymap.CommandEdge{{Key: gocui.NewKeyRune('y'), Tree: &keymap.CommandTree{Bindings: []*types.Binding{leaf}}}}}
	tree := &keymap.CommandTree{Children: []keymap.CommandEdge{{Key: gocui.NewKeyRune('x'), Tree: child}}}
	if err := flow.run(tree, nil, nil); err != nil {
		t.Fatal(err)
	}
	root := h.menus[0]
	if err := forkPrefixAccept(h, root, root.Items[0]); err != nil {
		t.Fatal(err)
	}
	nested := h.menus[1]
	if nested.OnCancel == nil {
		t.Fatal("nested menu lacks native cancel callback")
	}
	// Native menu cancellation dispatches through its appended Cancel item's OnPress.
	if err := forkPrefixCancel(h, nested); err != nil {
		t.Fatal(err)
	}
	if len(h.menus) != 3 {
		t.Fatalf("nested Esc opened %d menus, want parent restored as third menu", len(h.menus))
	}
	if len(h.invoked) != 0 {
		t.Fatal("cancel invoked a command")
	}
	parent := h.menus[2]
	if parent.OnCancel == nil {
		t.Fatal("restored parent lacks native cancel callback")
	}
	if err := forkPrefixCancel(h, parent); err != nil {
		t.Fatal(err)
	}
	if len(h.menus) != 3 || len(h.invoked) != 0 {
		t.Fatal("root Esc continued or invoked a command")
	}
}

func TestForkPrefixFlowInvalidSelectionAndRepositoryDoNotInvokeLeaf(t *testing.T) {
	for _, invalidation := range []struct {
		name string
		set  func(*forkPrefixMenuHarness)
	}{{"selection", func(h *forkPrefixMenuHarness) { h.valid = false }}, {"repository", func(h *forkPrefixMenuHarness) { h.valid = false }}} {
		t.Run(invalidation.name, func(t *testing.T) {
			h := &forkPrefixMenuHarness{valid: true}
			flow := h.flow()
			leaf := forkPrefixBinding("leaf")
			tree := &keymap.CommandTree{Children: []keymap.CommandEdge{{Key: gocui.NewKeyRune('x'), Tree: &keymap.CommandTree{Bindings: []*types.Binding{leaf}}}}}
			if err := flow.run(tree, nil, nil); err != nil {
				t.Fatal(err)
			}
			menu := h.menus[0]
			if !menu.BeforeAccept() {
				t.Fatal("acceptance rejected before invalidation")
			}
			h.currentMenu, h.restored = false, true
			invalidation.set(h)
			if err := menu.Items[0].OnPress(); err != nil {
				t.Fatal(err)
			}
			if len(h.invoked) != 0 {
				t.Fatal("invalid captured selection/repository invoked leaf")
			}
		})
	}
}

func TestForkPrefixFlowLastItemCancelIsHandledByNativeCancelCallback(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	flow := h.flow()
	if err := flow.run(&keymap.CommandTree{Children: []keymap.CommandEdge{{Key: gocui.NewKeyRune('x'), Tree: &keymap.CommandTree{Bindings: []*types.Binding{forkPrefixBinding("leaf")}}}}}, nil, nil); err != nil {
		t.Fatal(err)
	}
	root := h.menus[0]
	// createMenu appends its native Cancel menu item and routes it through OnCancel.
	root.Items = append(root.Items, &types.MenuItem{CommandID: "menu.cancel", OnPress: root.OnCancel})
	if err := forkPrefixCancel(h, root); err != nil {
		t.Fatal(err)
	}
	if len(h.menus) != 1 || len(h.invoked) != 0 {
		t.Fatal("root cancel should close without opening a menu or invoking a leaf")
	}
}

func TestForkPrefixFlowDirectLeafBindingDoesNotOpenContinuationMenu(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	flow := h.flow()
	leaf := forkPrefixBinding("direct")
	if err := flow.run(&keymap.CommandTree{Bindings: []*types.Binding{leaf}}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if len(h.menus) != 0 || len(h.invoked) != 1 || h.invoked[0] != leaf {
		t.Fatalf("direct leaf opened menu or failed to invoke: menus=%d invoked=%v", len(h.menus), h.invoked)
	}
}

func TestForkApplicableBindingsChoosesLiveExplicitAliasesOnce(t *testing.T) {
	inactive := &types.Binding{CommandID: "inactive", CommandActive: func() bool { return false }}
	oldDefault := &types.Binding{CommandID: "same-action", DefaultCommand: true}
	explicit := &types.Binding{CommandID: "same-action"}
	alias := &types.Binding{CommandID: "same-action"}
	got := forkApplicableBindings([]*types.Binding{inactive, oldDefault, explicit, alias})
	if len(got) != 1 || got[0] != explicit {
		t.Fatalf("applicable aliases = %#v", got)
	}
}

func TestForkPrefixExplicitLeafOverridesDefaultContinuations(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	leaf := &types.Binding{CommandID: "legacy-action"}
	child := &types.Binding{CommandID: "default-child", DefaultCommand: true}
	tree := &keymap.CommandTree{Bindings: []*types.Binding{leaf}, Children: []keymap.CommandEdge{{Key: gocui.NewKeyRune('p'), Tree: &keymap.CommandTree{Bindings: []*types.Binding{child}}}}}
	if err := h.flow().run(tree, nil, nil); err != nil {
		t.Fatal(err)
	}
	if len(h.menus) != 0 || len(h.invoked) != 1 || h.invoked[0] != leaf {
		t.Fatal("a new default prefix changed an explicit one-key shortcut")
	}
	h = &forkPrefixMenuHarness{valid: true}
	child.DefaultCommand = false
	if err := h.flow().run(tree, nil, nil); err != nil {
		t.Fatal(err)
	}
	if len(h.menus) != 1 || len(h.invoked) != 0 {
		t.Fatal("conflicting explicit leaf and descendant routes must offer a visible choice")
	}
}

// forkFallbackFlow returns a flow whose handlers return scripted results.
func forkFallbackFlow(h *forkPrefixMenuHarness, outcomes map[*types.Binding]error) *forkPrefixFlow {
	flow := h.flow()
	flow.invoke = func(binding *types.Binding) error {
		h.invoked = append(h.invoked, binding)
		return outcomes[binding]
	}
	return flow
}

func TestForkPrefixLeafFallsThroughToNextOwnerOnNotHandled(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	leaf := forkPrefixBinding("leaf")
	declining := forkPrefixBinding("declining-global")
	handling := forkPrefixBinding("handling-global")
	unused := forkPrefixBinding("unused-global")
	flow := forkFallbackFlow(h, map[*types.Binding]error{
		leaf:      gocui.ErrKeybindingNotHandled,
		declining: gocui.ErrKeybindingNotHandled,
		handling:  nil,
	})
	tree := &keymap.CommandTree{Bindings: []*types.Binding{leaf}, Fallbacks: []*types.Binding{declining, handling, unused}}
	if err := flow.run(tree, nil, nil); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []*types.Binding{leaf, declining, handling}, h.invoked)
}

func TestForkPrefixLeafAllOwnersDeclineConsumesKey(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	leaf := forkPrefixBinding("leaf")
	global := forkPrefixBinding("global")
	flow := forkFallbackFlow(h, map[*types.Binding]error{
		leaf:   gocui.ErrKeybindingNotHandled,
		global: gocui.ErrKeybindingNotHandled,
	})
	tree := &keymap.CommandTree{Bindings: []*types.Binding{leaf}, Fallbacks: []*types.Binding{global}}
	assert.NoError(t, flow.run(tree, nil, nil))
	assert.Equal(t, []*types.Binding{leaf, global}, h.invoked)
}

func TestForkPrefixDisabledLeafDoesNotFallThrough(t *testing.T) {
	// The real handler dispatcher applies the fork's consume-on-inapplicable rule.
	var invoked []string
	gui := &Gui{}
	inapplicable := &types.Binding{
		DefaultCommand:    true,
		Handler:           func() error { invoked = append(invoked, "leaf"); return gocui.ErrKeybindingNotHandled },
		GetDisabledReason: func() *types.DisabledReason { return &types.DisabledReason{Text: "wrong pane", Inapplicable: true} },
	}
	global := &types.Binding{Handler: func() error { invoked = append(invoked, "global"); return nil }}
	flow := &forkPrefixFlow{valid: func() bool { return true }, invoke: gui.callKeybindingHandler}
	tree := &keymap.CommandTree{Bindings: []*types.Binding{inapplicable}, Fallbacks: []*types.Binding{global}}
	assert.NoError(t, flow.run(tree, nil, nil))
	assert.Empty(t, invoked)
}

func TestForkPrefixFallbackSkipsInactiveSharedRoutes(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	leaf := forkPrefixBinding("leaf")
	inactive := forkPrefixBinding("inactive")
	inactive.CommandActive = func() bool { return false }
	active := forkPrefixBinding("active")
	flow := forkFallbackFlow(h, map[*types.Binding]error{leaf: gocui.ErrKeybindingNotHandled, active: nil})
	tree := &keymap.CommandTree{Bindings: []*types.Binding{leaf}, Fallbacks: []*types.Binding{inactive, active}}
	assert.NoError(t, flow.run(tree, nil, nil))
	assert.Equal(t, []*types.Binding{leaf, active}, h.invoked)
}

func TestForkPrefixFallbacksAreNotMenuChoicesButReachMenuLeaves(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	leaf := forkPrefixBinding("leaf")
	global := forkPrefixBinding("global")
	flow := forkFallbackFlow(h, map[*types.Binding]error{leaf: gocui.ErrKeybindingNotHandled, global: nil})
	child := &keymap.CommandTree{Bindings: []*types.Binding{forkPrefixBinding("child")}}
	tree := &keymap.CommandTree{
		Bindings:  []*types.Binding{leaf, forkPrefixBinding("second")},
		Fallbacks: []*types.Binding{global},
		Children:  []keymap.CommandEdge{{Key: gocui.NewKeyRune('x'), Tree: child}},
	}
	assert.NoError(t, flow.run(tree, nil, nil))
	for _, item := range h.menus[0].Items {
		assert.NotSame(t, global, item.Binding, "fallback offered as a menu choice")
	}
	assert.Len(t, h.menus[0].Items, 3)
	assert.NoError(t, forkPrefixAccept(h, h.menus[0], h.menus[0].Items[0]))
	assert.Equal(t, []*types.Binding{leaf, global}, h.invoked)
}

func TestForkPrefixHidesContinuationsWithOnlyInapplicableActions(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	flow := h.flow()
	applicable := &types.Binding{CommandID: "applicable", DefaultCommand: true}
	inapplicable := &types.Binding{
		CommandID: "inapplicable", DefaultCommand: true,
		GetDisabledReason: func() *types.DisabledReason { return &types.DisabledReason{Text: "not here", Inapplicable: true} },
	}
	tree := &keymap.CommandTree{Children: []keymap.CommandEdge{
		{Key: gocui.NewKeyRune('r'), Tree: &keymap.CommandTree{Bindings: []*types.Binding{applicable}}},
		{Key: gocui.NewKeyRune('p'), Tree: &keymap.CommandTree{Bindings: []*types.Binding{inapplicable}}},
	}}
	if err := flow.run(tree, []gocui.Key{gocui.NewKeyRune('s')}, nil); err != nil {
		t.Fatal(err)
	}
	if assert.Len(t, h.menus, 1) && assert.Len(t, h.menus[0].Items, 1) {
		assert.Equal(t, []gocui.Key{gocui.NewKeyRune('r')}, h.menus[0].Items[0].Keys)
	}
}

func TestForkPrefixWithNoApplicableActionOpensNoMenu(t *testing.T) {
	h := &forkPrefixMenuHarness{valid: true}
	flow := h.flow()
	inapplicable := &types.Binding{
		CommandID: "inapplicable", DefaultCommand: true,
		GetDisabledReason: func() *types.DisabledReason { return &types.DisabledReason{Text: "not here", Inapplicable: true} },
	}
	tree := &keymap.CommandTree{Children: []keymap.CommandEdge{
		{Key: gocui.NewKeyRune('p'), Tree: &keymap.CommandTree{Bindings: []*types.Binding{inapplicable}}},
	}}
	if err := flow.run(tree, []gocui.Key{gocui.NewKeyRune('c')}, nil); err != nil {
		t.Fatal(err)
	}
	assert.Empty(t, h.menus)
	assert.Empty(t, h.invoked)
}
