package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type JumpToSideWindowController struct {
	baseController
	c           *ControllerCommon
	nextTabFunc func() error
}

func NewJumpToSideWindowController(
	c *ControllerCommon,
	nextTabFunc func() error,
) *JumpToSideWindowController {
	return &JumpToSideWindowController{
		baseController: baseController{},
		c:              c,
		nextTabFunc:    nextTabFunc,
	}
}

func (self *JumpToSideWindowController) Context() types.Context {
	return nil
}

func (self *JumpToSideWindowController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	windows := self.c.Helpers().Window.SideWindows()
	jumpKeys := opts.Config.Universal.JumpToBlock

	// Fork: retain parameterized navigation definitions even without shortcuts,
	// rather than deriving command existence from the configured key count.
	// Keys remain positional; extra keys are unused, as upstream.
	bindings := make([]*types.Binding, 0, len(windows))
	for i := range windows {
		var keys []gocui.Key
		if i < len(jumpKeys) {
			keys = opts.GetKeys(jumpKeys[i])
		}
		bindings = append(bindings, &types.Binding{
			ViewName:  "",
			Keys:      keys,
			Handler:   opts.Guards.NoPopupPanel(self.goToSideWindow(windows[i])),
			CommandID: "navigation.focus-panel",
			Verb:      "focus-panel",
		})
	}
	return bindings
}

func (self *JumpToSideWindowController) goToSideWindow(window string) func() error {
	return func() error {
		sideWindowAlreadyActive := self.c.Helpers().Window.CurrentWindow() == window
		if sideWindowAlreadyActive && self.c.UserConfig().Gui.SwitchTabsWithPanelJumpKeys {
			return self.nextTabFunc()
		}

		context := self.c.Helpers().Window.GetContextForWindow(window)

		self.c.Context().Push(context, types.OnFocusOpts{})
		return nil
	}
}
