package gui

import (
	"errors"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/keymap"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Each displayed frame owns a one-use acceptance token. Native menu handling
// pops the popup before callbacks run; arm before that pop and consume after it.
// An old callback can never accept a replacement frame.
type (
	forkPrefixFrame struct{ armed bool }
	forkPrefixFlow  struct {
		active      *forkPrefixFrame
		valid       func() bool
		currentMenu func() bool
		restored    func() bool
		openMenu    func(types.CreateMenuOptions) error
		invoke      func(*types.Binding) error
	}
)

func (flow *forkPrefixFlow) invalidate() { flow.active = nil }

func freezeForkCommandTree(tree *keymap.CommandTree) *keymap.CommandTree {
	result := &keymap.CommandTree{}
	for _, binding := range tree.Bindings {
		result.Bindings = append(result.Bindings, freezeForkBinding(binding))
	}
	for _, binding := range tree.Fallbacks {
		result.Fallbacks = append(result.Fallbacks, freezeForkBinding(binding))
	}
	for _, edge := range tree.Children {
		result.Children = append(result.Children, keymap.CommandEdge{Key: edge.Key, Tree: freezeForkCommandTree(edge.Tree)})
	}
	return result
}

func freezeForkBinding(binding *types.Binding) *types.Binding {
	copy := *binding
	// Metadata is evaluated while the source pane still has focus. Guards and
	// handlers remain live and are only invoked again after restoring it.
	copy.CommandID, copy.Verb = binding.GetCommandID(), binding.GetVerb()
	copy.Description, copy.ShortDescription, copy.Tooltip = binding.GetDescription(), binding.GetShortDescription(), binding.GetTooltip()
	copy.CommandIDFunc, copy.VerbFunc = nil, nil
	copy.DescriptionFunc, copy.ShortDescriptionFunc, copy.TooltipFunc = nil, nil, nil
	return &copy
}

// invokeLeaf runs the selected leaf. Only ErrKeybindingNotHandled, the
// upstream "decline" result, reaches the node's fallbacks, in priority order.
// The first owner that handles the key, or consumes it through its own
// disabled/inapplicable result, ends dispatch. If every owner declines, the key
// is consumed here so no global or parent meaning runs outside the fork tree.
func (flow *forkPrefixFlow) invokeLeaf(binding *types.Binding, fallbacks []*types.Binding) error {
	err := flow.invoke(binding)
	if !errors.Is(err, gocui.ErrKeybindingNotHandled) {
		return err
	}
	for _, next := range fallbacks {
		// A route whose CommandActive is false is a different semantic action
		// of a shared binding, not an owner of this key in the current mode.
		if next.CommandActive != nil && !next.CommandActive() {
			continue
		}
		if err := flow.invoke(next); !errors.Is(err, gocui.ErrKeybindingNotHandled) {
			return err
		}
	}
	return nil
}

func forkApplicableBindings(bindings []*types.Binding) []*types.Binding {
	var applicable, explicit []*types.Binding
	for _, binding := range bindings {
		if binding.CommandActive != nil && !binding.CommandActive() {
			continue
		}
		if binding.GetDisabledReason != nil {
			if reason := binding.GetDisabledReason(); reason != nil && (reason.Inapplicable || reason.AllowFurtherDispatching) && binding.DefaultCommand {
				continue
			}
		}
		applicable = append(applicable, binding)
		if !binding.DefaultCommand {
			explicit = append(explicit, binding)
		}
	}
	if len(explicit) > 0 {
		applicable = explicit
	}
	// Several upstream entry points may be aliases of one semantic action.
	// Explicit routes win before coalescing so an alias cannot hide an override.
	seen := make(map[string]bool, len(applicable))
	result := applicable[:0]
	for _, binding := range applicable {
		id := binding.GetCommandID()
		if id != "" && seen[id] {
			continue
		}
		if id != "" {
			seen[id] = true
		}
		result = append(result, binding)
	}
	return result
}

func (flow *forkPrefixFlow) run(tree *keymap.CommandTree, path []gocui.Key, parent func() error) error {
	if !flow.valid() {
		return nil
	}
	bindings := forkApplicableBindings(tree.Bindings)
	// An explicit single-key action overrides default continuations. If the
	// user explicitly configured both a leaf and descendants, keep the visible
	// choice instead of guessing which configured action they intended.
	explicitLeaf := len(bindings) > 0 && !bindings[0].DefaultCommand
	if explicitLeaf && !forkExplicitDescendants(tree) {
		tree = &keymap.CommandTree{Bindings: tree.Bindings, Fallbacks: tree.Fallbacks}
	}
	if len(tree.Children) == 0 {
		if len(bindings) == 0 {
			return nil
		}
		if len(bindings) == 1 {
			return flow.invokeLeaf(bindings[0], tree.Fallbacks)
		}
	}
	return flow.show(tree, bindings, path, parent)
}

// forkTreeHasApplicable reports whether a continuation still leads to an
// action here: an applicable binding, a fallback owner, or such a descendant.
func forkTreeHasApplicable(tree *keymap.CommandTree) bool {
	if len(forkApplicableBindings(tree.Bindings)) > 0 || len(tree.Fallbacks) > 0 {
		return true
	}
	for _, edge := range tree.Children {
		if forkTreeHasApplicable(edge.Tree) {
			return true
		}
	}
	return false
}

func forkExplicitDescendants(tree *keymap.CommandTree) bool {
	for _, edge := range tree.Children {
		for _, binding := range edge.Tree.Bindings {
			if !binding.DefaultCommand {
				return true
			}
		}
		if forkExplicitDescendants(edge.Tree) {
			return true
		}
	}
	return false
}

func (flow *forkPrefixFlow) show(tree *keymap.CommandTree, bindings []*types.Binding, path []gocui.Key, parent func() error) error {
	frame := &forkPrefixFrame{}
	take := func() bool {
		if !frame.armed {
			return false
		}
		frame.armed = false
		return flow.active == nil && flow.restored() && flow.valid()
	}
	back := func() error {
		if !take() || parent == nil {
			return nil
		}
		return parent()
	}
	reopen := func() error { return flow.run(tree, path, parent) }
	items := make([]*types.MenuItem, 0, len(bindings)+len(tree.Children))
	for _, binding := range bindings {
		item := &types.MenuItem{
			Binding: binding, CommandID: binding.CommandID, Verb: binding.Verb,
			Label: binding.Description, Tooltip: binding.Tooltip,
			OnPress: func() error {
				if !take() {
					return nil
				}
				return flow.invokeLeaf(binding, tree.Fallbacks)
			},
		}
		if binding.GetDisabledReason != nil {
			item.DisabledReason = binding.GetDisabledReason()
		}
		items = append(items, item)
	}
	for _, edge := range tree.Children {
		// A continuation whose every action is inapplicable in this pane is
		// hidden, like an inapplicable leaf, so a global route cannot appear
		// where it does nothing (for example s p in Status).
		if !forkTreeHasApplicable(edge.Tree) {
			continue
		}
		nextPath := append(append([]gocui.Key(nil), path...), edge.Key)
		items = append(items, &types.MenuItem{
			CommandID: "menu.command-prefix", Verb: "discover-commands",
			Label: forkPrefixDescription(edge.Tree), Keys: []gocui.Key{edge.Key},
			OpensMenu: len(edge.Tree.Children) > 0,
			OnPress: func() error {
				if !take() {
					return nil
				}
				return flow.run(edge.Tree, nextPath, reopen)
			},
		})
	}
	opts := types.CreateMenuOptions{
		Title: types.CommandPathLabel(path), Items: items, OnCancel: back,
		BlockUnmatchedKeys: forkPrefixBlockedKeys(),
		BeforeAccept: func() bool {
			if flow.active != frame || !flow.currentMenu() {
				return false
			}
			frame.armed = true
			flow.active = nil
			return true
		},
	}
	if err := flow.openMenu(opts); err != nil {
		flow.invalidate()
		return err
	}
	flow.active = frame
	return nil
}

func forkPrefixDescription(tree *keymap.CommandTree) string {
	var descriptions []string
	var visit func(*keymap.CommandTree)
	visit = func(node *keymap.CommandTree) {
		for _, binding := range node.Bindings {
			if binding.Description != "" {
				descriptions = append(descriptions, binding.Description)
			}
		}
		for _, edge := range node.Children {
			visit(edge.Tree)
		}
	}
	visit(tree)
	return strings.Join(descriptions, " / ")
}

func forkPrefixBlockedKeys() []gocui.Key {
	var keys []gocui.Key
	// Uppercase events are included until the prefix-menu registration is wired
	// through the command matcher. Physical navigation and confirmation stay free.
	for r := rune(' '); r <= '~'; r++ {
		keys = append(keys, gocui.NewKeyRune(r))
	}
	for _, label := range []string{"<tab>", "<backtab>", "<delete>", "<f2>", "<f3>", "<shift+f3>", "<f5>"} {
		key, ok := config.KeyFromLabel(label)
		if !ok {
			panic("invalid prefix key: " + label)
		}
		keys = append(keys, key)
	}
	return keys
}
