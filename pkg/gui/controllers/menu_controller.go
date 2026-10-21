package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

type MenuController struct {
	baseController
	*ListControllerTrait[*types.MenuItem]
	c *ControllerCommon
	// for delegating navigation to, see physicalKeyBindings
	listController *ListController
}

var _ types.IController = &MenuController{}

func NewMenuController(
	c *ControllerCommon,
	listController *ListController,
) *MenuController {
	return &MenuController{
		baseController: baseController{},
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().Menu,
			c.Contexts().Menu.GetSelected,
			c.Contexts().Menu.GetSelectedItems,
		),
		c:              c,
		listController: listController,
	}
}

// NOTE: if you add a new keybinding here, you'll also need to add it to
// `essentialKeys` in `pkg/gui/menu_panel.go`, so that menu items can't shadow it
func (self *MenuController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	bindings := []*types.Binding{
		{
			Keys:              opts.GetKeys(opts.Config.Universal.Select),
			Handler:           self.withItem(self.press),
			GetDisabledReason: self.require(self.singleItemSelected()),
			CommandID:         "menu.execute",
			Verb:              "execute",
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.ConfirmMenu),
			Handler:           self.withItem(self.press),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.Execute,
			DisplayOnScreen:   true,
			CommandID:         "menu.execute",
			Verb:              "execute",
		},
		{
			Keys:            opts.GetKeys(opts.Config.Universal.Return),
			Handler:         self.close,
			Description:     self.c.Tr.CloseCancel,
			DisplayOnScreen: true,
			CommandID:       "menu.cancel",
			Verb:            "cancel",
		},
	}

	if self.context().FilterAsYouType() {
		bindings = append(bindings, self.physicalKeyBindings(opts)...)
	}

	return bindings
}

// In a menu that filters as you type, the keys configured for driving the menu
// may all be printable, and printable keys become filter text once the user
// starts typing. These keys can't, so binding them on top guarantees that the
// menu stays usable no matter how the keybindings are configured.
func (self *MenuController) physicalKeyBindings(opts types.KeybindingsOpts) []*types.Binding {
	candidates := []struct {
		key        gocui.Key
		configured config.Keybinding
		binding    *types.Binding
	}{
		{
			key:        gocui.NewKeyName(gocui.KeyEnter),
			configured: opts.Config.Universal.ConfirmMenu,
			binding: &types.Binding{
				Handler:           self.withItem(self.press),
				GetDisabledReason: self.require(self.singleItemSelected()),
				CommandID:         "menu.execute",
				Verb:              "execute",
			},
		},
		{gocui.NewKeyName(gocui.KeyEsc), opts.Config.Universal.Return, &types.Binding{Handler: self.close, CommandID: "menu.cancel", Verb: "cancel"}},
		{gocui.NewKeyName(gocui.KeyArrowUp), opts.Config.Universal.PrevItem, &types.Binding{Handler: self.listController.HandlePrevLine, CommandID: "navigation.list.previous", Verb: "previous-item"}},
		{gocui.NewKeyName(gocui.KeyArrowDown), opts.Config.Universal.NextItem, &types.Binding{Handler: self.listController.HandleNextLine, CommandID: "navigation.list.next", Verb: "next-item"}},
		{gocui.NewKeyName(gocui.KeyPgup), opts.Config.Universal.PrevPage, &types.Binding{Handler: self.listController.HandlePrevPage, CommandID: "navigation.page.previous", Verb: "previous-page"}},
		{gocui.NewKeyName(gocui.KeyPgdn), opts.Config.Universal.NextPage, &types.Binding{Handler: self.listController.HandleNextPage, CommandID: "navigation.page.next", Verb: "next-page"}},
		{gocui.NewKeyName(gocui.KeyHome), opts.Config.Universal.GotoTop, &types.Binding{Handler: self.listController.HandleGotoTop, CommandID: "navigation.list.first", Verb: "first-item"}},
		{gocui.NewKeyName(gocui.KeyEnd), opts.Config.Universal.GotoBottom, &types.Binding{Handler: self.listController.HandleGotoBottom, CommandID: "navigation.list.last", Verb: "last-item"}},
	}

	bindings := []*types.Binding{}
	for _, candidate := range candidates {
		if lo.Contains(opts.GetKeys(candidate.configured), candidate.key) {
			// this key is the configured one, so it drives the menu already
			continue
		}

		candidate.binding.Keys = []gocui.Key{candidate.key}
		bindings = append(bindings, candidate.binding)
	}

	return bindings
}

func (self *MenuController) GetOnDoubleClick() func() error {
	return self.withItemGraceful(self.press)
}

func (self *MenuController) GetOnFocus() func(types.OnFocusOpts) {
	return func(types.OnFocusOpts) {
		selectedMenuItem := self.context().GetSelected()
		if selectedMenuItem != nil {
			self.c.Views().Tooltip.SetContent(self.c.Helpers().Confirmation.TooltipForMenuItem(selectedMenuItem))
		}
	}
}

func (self *MenuController) press(selectedItem *types.MenuItem) error {
	return self.context().OnMenuPress(selectedItem)
}

func (self *MenuController) close() error {
	if self.context().FilterStarted() {
		self.stopFiltering()
		return nil
	}

	if self.context().IsFiltering() {
		self.c.Helpers().Search.Cancel()
		return nil
	}

	return self.context().OnMenuPress(nil)
}

// Hides the filter row again and puts the menu back the way it was, keeping the
// item that was selected. It takes another escape to close the menu.
func (self *MenuController) stopFiltering() {
	self.c.Views().MenuFilter.ClearTextArea()
	self.c.Views().MenuFilter.RenderTextArea()

	self.context().SetFilterStarted(false)
	self.context().ClearFilter()
	self.c.PostRefreshUpdate(self.context())
}

func (self *MenuController) context() *context.MenuContext {
	return self.c.Contexts().Menu
}
