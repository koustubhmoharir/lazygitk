package controllers

import (
	"slices"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// commandPaletteBindings preserves local/global/navigation ordering. Local
// commands own aliases shared with the global section, matching view dispatch
// precedence; their handlers and disabled checks stay authoritative.
func commandPaletteBindings(bindings []*types.Binding, viewName string) ([]*types.Binding, []*types.Binding, []*types.Binding) {
	var local, global, navigation []*types.Binding
	for _, binding := range bindings {
		// Filter ineligible view bindings before evaluating callbacks. Dynamic
		// descriptions can touch state owned by another pane.
		if binding.ViewName != "" && binding.ViewName != viewName && binding.Tag != "global" {
			continue
		}
		if binding.GetDescription() == "" {
			continue
		}
		if binding.ViewName == "" || binding.Tag == "global" {
			global = append(global, binding)
		} else if binding.ViewName == viewName {
			if binding.Tag == "navigation" {
				navigation = append(navigation, binding)
			} else {
				local = append(local, binding)
			}
		}
	}

	sections := [3][]*types.Binding{local, global, navigation}
	seen := make(map[string]*types.Binding)
	for i, bindings := range sections {
		sections[i] = nil
		for _, binding := range bindings {
			id := binding.GetCommandID()
			if previous, ok := seen[id]; id != "" && ok {
				previous.Keys = appendPaletteKeys(previous.Keys, binding.Keys)
				previous.CommandPaths = appendPaletteCommandPaths(previous.CommandPaths, binding.CommandPaths)
				continue
			}
			// Palette-only copies keep alias merging from changing dispatch or hints.
			copy := *binding
			copy.Keys = appendPaletteKeys(nil, binding.Keys)
			copy.CommandPaths = appendPaletteCommandPaths(nil, binding.CommandPaths)
			sections[i] = append(sections[i], &copy)
			if id != "" {
				seen[id] = &copy
			}
			// A missing ID is not evidence of equivalence. Keep unknown commands
			// visible; the inventory/validation checks catch metadata omissions.
		}
	}
	return sections[0], sections[1], sections[2]
}

func appendPaletteCommandPaths(existing, additional [][]gocui.Key) [][]gocui.Key {
	for _, path := range additional {
		found := false
		for _, current := range existing {
			if slices.Equal(current, path) {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, append([]gocui.Key(nil), path...))
		}
	}
	return existing
}

func appendPaletteKeys(existing, additional []gocui.Key) []gocui.Key {
	seen := make(map[gocui.Key]bool, len(existing)+len(additional))
	for _, key := range existing {
		seen[key] = true
	}
	for _, key := range additional {
		if !seen[key] {
			existing = append(existing, key)
			seen[key] = true
		}
	}
	return existing
}

func commandPaletteMenuItem(binding *types.Binding, section *types.MenuSection, tooltip string, invoke func(*types.Binding) error) *types.MenuItem {
	var disabledReason *types.DisabledReason
	if binding.GetDisabledReason != nil {
		disabledReason = binding.GetDisabledReason()
	}
	return &types.MenuItem{
		Binding:        binding,
		CommandID:      binding.CommandID,
		CommandIDFunc:  binding.GetCommandID,
		Verb:           binding.Verb,
		VerbFunc:       binding.GetVerb,
		OpensMenu:      binding.OpensMenu,
		Label:          binding.GetDescription(),
		Keys:           binding.Keys,
		Tooltip:        tooltip,
		DisabledReason: disabledReason,
		Section:        section,
		OnPress: func() error {
			if binding.Handler == nil {
				return nil
			}
			return invoke(binding)
		},
	}
}
