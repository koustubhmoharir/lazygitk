package gui

import (
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
		copy := *binding
		// Metadata is evaluated while the source pane still has focus. Guards and
		// handlers remain live and are only invoked again after restoring it.
		copy.CommandID, copy.Verb = binding.GetCommandID(), binding.GetVerb()
		copy.Description, copy.ShortDescription, copy.Tooltip = binding.GetDescription(), binding.GetShortDescription(), binding.GetTooltip()
		copy.CommandIDFunc, copy.VerbFunc = nil, nil
		copy.DescriptionFunc, copy.ShortDescriptionFunc, copy.TooltipFunc = nil, nil, nil
		result.Bindings = append(result.Bindings, &copy)
	}
	for _, edge := range tree.Children {
		result.Children = append(result.Children, keymap.CommandEdge{Key: edge.Key, Tree: freezeForkCommandTree(edge.Tree)})
	}
	return result
}

func forkApplicableBindings(bindings []*types.Binding) []*types.Binding {
	var applicable, explicit []*types.Binding
	for _, binding := range bindings {
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
		return explicit
	}
	return applicable
}

func (flow *forkPrefixFlow) run(tree *keymap.CommandTree, path []gocui.Key, parent func() error) error {
	if !flow.valid() {
		return nil
	}
	bindings := forkApplicableBindings(tree.Bindings)
	if len(tree.Children) == 0 {
		if len(bindings) == 0 {
			return nil
		}
		if len(bindings) == 1 {
			return flow.invoke(bindings[0])
		}
	}
	return flow.show(tree, bindings, path, parent)
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
				return flow.invoke(binding)
			},
		}
		if binding.GetDisabledReason != nil {
			item.DisabledReason = binding.GetDisabledReason()
		}
		items = append(items, item)
	}
	for _, edge := range tree.Children {
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
