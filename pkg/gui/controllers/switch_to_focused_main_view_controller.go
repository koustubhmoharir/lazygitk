package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// This controller is for all contexts that can focus their main view.

var _ types.IController = &SwitchToFocusedMainViewController{}

type SwitchToFocusedMainViewController struct {
	baseController
	c       *ControllerCommon
	context types.Context
	// Fork: replaces the plain focus with a context-specific inspection
	// (see WithInspect). Nil keeps upstream's focus-only behavior.
	inspect func() error
}

func NewSwitchToFocusedMainViewController(
	c *ControllerCommon,
	context types.Context,
) *SwitchToFocusedMainViewController {
	return &SwitchToFocusedMainViewController{
		baseController: baseController{},
		c:              c,
		context:        context,
	}
}

// Fork: WithInspect makes the focus-diff key run inspect instead of only focusing
// the main view. Files uses it so that its inspection key resolves a conflicted file
// the same way Enter does, rather than only showing read-only details.
func (self *SwitchToFocusedMainViewController) WithInspect(inspect func() error) *SwitchToFocusedMainViewController {
	self.inspect = inspect
	return self
}

func (self *SwitchToFocusedMainViewController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	bindings := []*types.Binding{
		{
			Keys:        opts.GetKeys(opts.Config.Universal.FocusMainView),
			Handler:     self.handleFocusMainView,
			Description: self.c.Tr.FocusMainView,
			Tag:         "global",
			CommandID:   "navigation.focus-diff",
			Verb:        "focus-diff",
		},
	}

	return bindings
}

func (self *SwitchToFocusedMainViewController) GetMouseKeybindings(opts types.KeybindingsOpts) []*gocui.ViewMouseBinding {
	return []*gocui.ViewMouseBinding{
		{
			ViewName:    "main",
			Key:         gocui.MouseLeft,
			Handler:     self.onClickMain,
			FocusedView: self.context.GetViewName(),
		},
		{
			ViewName:    "secondary",
			Key:         gocui.MouseLeft,
			Handler:     self.onClickSecondary,
			FocusedView: self.context.GetViewName(),
		},
	}
}

func (self *SwitchToFocusedMainViewController) Context() types.Context {
	return self.context
}

func (self *SwitchToFocusedMainViewController) onClickMain(opts gocui.ViewMouseBindingOpts) error {
	return self.focusMainView(self.c.Contexts().Normal, opts.Y)
}

func (self *SwitchToFocusedMainViewController) onClickSecondary(opts gocui.ViewMouseBindingOpts) error {
	return self.focusMainView(self.c.Contexts().NormalSecondary, opts.Y)
}

func (self *SwitchToFocusedMainViewController) handleFocusMainView() error {
	// Fork: see WithInspect. Mouse clicks still only focus the main view.
	if self.inspect != nil {
		return self.inspect()
	}
	return focusMainView(self.c, self.context, -1)
}

func focusMainView(c *ControllerCommon, source types.Context, clickedLineIdx int) error {
	// Usually the main pane, but the content can be in the secondary one alone: a file
	// with nothing but staged changes shows them there.
	mainViewContext := c.Contexts().Normal
	if c.State().GetRepoState().GetMainPanes() == types.SecondaryPaneOnly {
		mainViewContext = c.Contexts().NormalSecondary
	}
	return focusMainViewPane(c, source, mainViewContext, clickedLineIdx)
}

func (self *SwitchToFocusedMainViewController) focusMainView(mainViewContext *context.MainContext, clickedLineIdx int) error {
	return focusMainViewPane(self.c, self.context, mainViewContext, clickedLineIdx)
}

func focusMainViewPane(c *ControllerCommon, source types.Context, mainViewContext *context.MainContext, clickedLineIdx int) error {
	mainViewContext.ClearSearchString()
	c.Context().Push(mainViewContext, types.OnFocusOpts{})

	if _, ok := source.(types.DiffMainViewContext); !ok {
		return nil
	}

	// The diff on screen was produced for reading, and the renderer that produced it may
	// have laid it out in a way that says nothing about which line of which file each row
	// is. Now that the user wants to act on it, it is re-rendered as git's own diff — the
	// panel below decides that for itself, from the same question — and the selection
	// goes on that instead of on rows we can't place.
	if c.Helpers().DiffLine.MainViewDiffMode() == git_commands.DiffModeRaw {
		c.Helpers().DiffLine.RenderFocusedMainViewAgain(mainViewContext.GetView(), source, func() {
			c.Helpers().DiffLine.EstablishSelection(mainViewContext, clickedLineIdx)
		})
		return nil
	}

	c.Helpers().DiffLine.EstablishSelection(mainViewContext, clickedLineIdx)
	return nil
}
