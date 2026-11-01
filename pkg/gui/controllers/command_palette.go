package controllers

import (
	"slices"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// commandPalette is the content of the `?` palette for one focused view.
type commandPalette struct {
	// Focused holds the focused pane's local commands and the global commands,
	// merged by command ID. Local commands own aliases shared with the global
	// section, matching view dispatch precedence.
	Focused []*types.Binding
	// Panes holds the other panes that have commands, in palette order.
	Panes []palettePane
	// Navigation holds the focused pane's navigation commands.
	Navigation []*types.Binding
}

type palettePane struct {
	View     string
	Bindings []*types.Binding
}

// commandPaletteBindings partitions the bindings for the `?` palette. Every
// described command appears: the focused pane and the globals are evaluated with
// their callbacks, while other panes are described from their static fields
// only, because their callbacks may read that pane's state (see
// staticPaletteCopy). Navigation of other panes is not listed; the navigation
// section belongs to the focused pane. Popups (forkPalettePopupViews) get no
// section from another pane; when a popup is focused, its commands are in the
// focused section.
func commandPaletteBindings(bindings []*types.Binding, viewName string) commandPalette {
	var focusedLocal, global, navigation []*types.Binding
	otherPanes := make(map[string][]*types.Binding)
	for _, binding := range bindings {
		if binding.ViewName == "" || binding.Tag == "global" {
			if binding.GetDescription() != "" {
				global = append(global, binding)
			}
			continue
		}
		if binding.ViewName == viewName {
			if binding.GetDescription() == "" {
				continue
			}
			if binding.Tag == "navigation" {
				navigation = append(navigation, binding)
			} else {
				focusedLocal = append(focusedLocal, binding)
			}
			continue
		}
		if binding.Tag == "navigation" || binding.Description == "" || forkPalettePopupViews[binding.ViewName] {
			continue
		}
		otherPanes[binding.ViewName] = append(otherPanes[binding.ViewName], binding)
	}

	// The focused section is merged first; its command IDs then keep the same
	// command out of the other panes' sections.
	seen := make(map[string]*types.Binding)
	focused := mergePaletteCopies(nil, append(focusedLocal, global...), seen, false)
	focusedIDs := make(map[string]bool, len(seen))
	for id := range seen {
		focusedIDs[id] = true
	}
	navigation = mergePaletteCopies(nil, navigation, seen, false)

	var panes []palettePane
	views := make([]string, 0, len(otherPanes))
	for view := range otherPanes {
		views = append(views, view)
	}
	for _, view := range orderPaletteViews(views) {
		var candidates []*types.Binding
		for _, binding := range otherPanes[view] {
			if binding.CommandID != "" && focusedIDs[binding.CommandID] {
				continue
			}
			candidates = append(candidates, binding)
		}
		paneBindings := mergePaletteCopies(nil, candidates, make(map[string]*types.Binding), true)
		if len(paneBindings) == 0 {
			continue
		}
		panes = append(panes, palettePane{View: view, Bindings: sortPaletteBindingsByKeyGroup(paneBindings)})
	}

	return commandPalette{
		Focused:    sortPaletteBindingsByKeyGroup(focused),
		Panes:      panes,
		Navigation: sortPaletteBindingsByKeyGroup(navigation),
	}
}

// mergePaletteCopies appends palette copies of bindings to list. A binding whose
// command ID is already in seen merges its keys and paths into that copy, so
// aliases show as one command. Copies are palette-only: dispatch and hints keep
// the originals. Bindings without an ID are never merged, since a missing ID is
// not evidence of equivalence. With static, IDs come from the static field.
func mergePaletteCopies(list, bindings []*types.Binding, seen map[string]*types.Binding, static bool) []*types.Binding {
	for _, binding := range bindings {
		id := binding.CommandID
		if !static {
			id = binding.GetCommandID()
		}
		if previous, ok := seen[id]; id != "" && ok {
			previous.Keys = appendPaletteKeys(previous.Keys, binding.Keys)
			previous.CommandPaths = appendPaletteCommandPaths(previous.CommandPaths, binding.CommandPaths)
			continue
		}
		var copy *types.Binding
		if static {
			copy = staticPaletteCopy(binding)
		} else {
			clone := *binding
			copy = &clone
		}
		copy.Keys = appendPaletteKeys(nil, binding.Keys)
		copy.CommandPaths = appendPaletteCommandPaths(nil, binding.CommandPaths)
		list = append(list, copy)
		if id != "" {
			seen[id] = copy
		}
	}
	return list
}

// staticPaletteCopy copies a binding of another pane for the palette. Its
// dynamic callbacks are dropped: the palette shows only the static description,
// key and tooltip, and the command is never active or enabled from here. Its
// disabled reason is set by the palette, which explains that it is available
// in its own pane.
func staticPaletteCopy(binding *types.Binding) *types.Binding {
	clone := *binding
	clone.DescriptionFunc = nil
	clone.ShortDescriptionFunc = nil
	clone.CommandIDFunc = nil
	clone.VerbFunc = nil
	clone.TooltipFunc = nil
	clone.CommandActive = nil
	clone.GetDisabledReason = nil
	return &clone
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

// commandPaletteMenuItem builds a palette item. Its disabled reason is evaluated
// here, so callers that must not evaluate a binding's callbacks pass a copy
// without GetDisabledReason and set the reason on the item themselves.
// ExplainDisabled makes a press show the reason, even for an inapplicable default
// command, which would otherwise consume the press silently.
func commandPaletteMenuItem(binding *types.Binding, section *types.MenuSection, tooltip string, invoke func(*types.Binding) error) *types.MenuItem {
	var disabledReason *types.DisabledReason
	if binding.GetDisabledReason != nil {
		disabledReason = binding.GetDisabledReason()
	}
	return &types.MenuItem{
		Binding:         binding,
		CommandID:       binding.CommandID,
		CommandIDFunc:   binding.GetCommandID,
		Verb:            binding.Verb,
		VerbFunc:        binding.GetVerb,
		OpensMenu:       binding.OpensMenu,
		Label:           binding.GetDescription(),
		Keys:            binding.Keys,
		Tooltip:         tooltip,
		DisabledReason:  disabledReason,
		ExplainDisabled: true,
		Section:         section,
		OnPress: func() error {
			if binding.Handler == nil {
				return nil
			}
			return invoke(binding)
		},
	}
}
