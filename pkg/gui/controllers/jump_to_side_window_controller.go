package controllers

import (
	"fmt"

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
			ViewName: "",
			Keys:     keys,
			Handler:  opts.Guards.NoPopupPanel(self.goToSideWindow(windows[i])),
			// Fork: distinct target identities keep explicit overrides and palette
			// deduplication from conflating five different destinations.
			CommandID:   "navigation.focus-panel." + windows[i],
			Description: fmt.Sprintf(self.c.Tr.FocusPanel, self.focusPanelTitle(windows[i])),
			Tag:         "navigation",
			Verb:        "focus-panel",
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

func (self *JumpToSideWindowController) focusPanelTitle(window string) string {
	switch window {
	case "status":
		return self.c.Tr.StatusTitle
	case "files":
		return self.c.Tr.FilesTitle
	case "branches":
		return self.c.Tr.BranchesTitle
	case "commits":
		return self.c.Tr.CommitsTitle
	case "stash":
		return self.c.Tr.StashTitle
	case "worktrees":
		return self.c.Tr.WorktreesTitle
	case "submodules":
		return self.c.Tr.SubmodulesTitle
	case "remotes":
		return self.c.Tr.RemotesTitle
	case "tags":
		return self.c.Tr.TagsTitle
	case "reflog":
		return self.c.Tr.ReflogCommitsTitle
	}
	return window
}
