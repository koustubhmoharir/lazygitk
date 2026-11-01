package controllers

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

type OptionsMenuAction struct {
	c *ControllerCommon
}

func (self *OptionsMenuAction) Call() error {
	ctx := self.c.Context().Current()
	bindings, _ := self.c.GetInitialKeybindingsWithCustomCommands()
	palette := commandPaletteBindings(bindings, ctx.GetViewName())

	menuItems := []*types.MenuItem{}

	paletteItem := func(binding *types.Binding, section *types.MenuSection) *types.MenuItem {
		tooltip := binding.GetTooltip()
		keyLabels := binding.KeyLabels()
		if len(keyLabels) > 1 {
			if tooltip != "" {
				tooltip += "\n\n"
			}
			tooltip += self.c.Tr.KeybindingsTooltip + strings.Join(keyLabels, ", ")
		}
		// Fork: palette items record a palette use, not a shortcut use (see fork_command_usage.go).
		return commandPaletteMenuItem(binding, section, tooltip, self.c.IGuiCommon.CallPaletteKeybindingHandler)
	}
	appendBindings := func(bindings []*types.Binding, section *types.MenuSection) {
		menuItems = append(menuItems,
			lo.Map(bindings, func(binding *types.Binding, _ int) *types.MenuItem {
				return paletteItem(binding, section)
			})...)
	}

	// Fork: sections are the focused pane with the globals, the other panes in
	// layout order, and the focused pane's navigation (see command_palette.go).
	appendBindings(palette.Focused, &types.MenuSection{Title: paletteFocusedTitle(ctx.GetViewName(), self.c.Tr), Column: 1})
	for _, pane := range palette.Panes {
		title := paletteViewTitle(pane.View, self.c.Tr)
		section := &types.MenuSection{Title: title, Column: 1}
		for _, binding := range pane.Bindings {
			item := paletteItem(binding, section)
			// The binding's callbacks were not evaluated, so its state here is unknown.
			item.DisabledReason = &types.DisabledReason{
				Text:         fmt.Sprintf(self.c.Tr.CommandPaletteAvailableIn, title),
				Inapplicable: true,
			}
			menuItems = append(menuItems, item)
		}
	}
	appendBindings(palette.Navigation, &types.MenuSection{Title: self.c.Tr.KeybindingsMenuSectionNavigation, Column: 1})

	return self.c.Menu(types.CreateMenuOptions{
		Title:                      self.c.Tr.Keybindings,
		Items:                      menuItems,
		HideCancel:                 true,
		ColumnAlignment:            []utils.Alignment{utils.AlignRight, utils.AlignLeft},
		AllowFilteringKeybindings:  true,
		KeepConflictingKeybindings: true,
		FilterAsYouType:            true,
	})
}
