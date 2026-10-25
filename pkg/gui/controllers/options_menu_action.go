package controllers

import (
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
	local, global, navigation := self.getBindings(ctx)

	menuItems := []*types.MenuItem{}

	appendBindings := func(bindings []*types.Binding, section *types.MenuSection) {
		menuItems = append(menuItems,
			lo.Map(bindings, func(binding *types.Binding, _ int) *types.MenuItem {
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
			})...)
	}

	appendBindings(local, &types.MenuSection{Title: self.c.Tr.KeybindingsMenuSectionLocal, Column: 1})
	appendBindings(global, &types.MenuSection{Title: self.c.Tr.KeybindingsMenuSectionGlobal, Column: 1})
	appendBindings(navigation, &types.MenuSection{Title: self.c.Tr.KeybindingsMenuSectionNavigation, Column: 1})

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

// Returns three slices of bindings: local, global, and navigation
func (self *OptionsMenuAction) getBindings(context types.Context) ([]*types.Binding, []*types.Binding, []*types.Binding) {
	bindings, _ := self.c.GetInitialKeybindingsWithCustomCommands()
	// Fork: stable operation IDs replace upstream description-based deduplication,
	// which could hide distinct commands with equal translated labels.
	return commandPaletteBindings(bindings, context.GetViewName())
}
