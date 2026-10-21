package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/snake"
)

type SnakeController struct {
	baseController
	c *ControllerCommon
}

var _ types.IController = &SnakeController{}

func NewSnakeController(
	c *ControllerCommon,
) *SnakeController {
	return &SnakeController{
		baseController: baseController{},
		c:              c,
	}
}

func (self *SnakeController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	bindings := []*types.Binding{
		{
			Keys:      opts.GetKeys(opts.Config.Universal.NextItem),
			Handler:   self.SetDirection(snake.Down),
			CommandID: "game.snake.down",
			Verb:      "play-down",
		},
		{
			Keys:      opts.GetKeys(opts.Config.Universal.PrevItem),
			Handler:   self.SetDirection(snake.Up),
			CommandID: "game.snake.up",
			Verb:      "play-up",
		},
		{
			Keys:      opts.GetKeys(opts.Config.Universal.PrevBlock),
			Handler:   self.SetDirection(snake.Left),
			CommandID: "game.snake.left",
			Verb:      "play-left",
		},
		{
			Keys:      opts.GetKeys(opts.Config.Universal.NextBlock),
			Handler:   self.SetDirection(snake.Right),
			CommandID: "game.snake.right",
			Verb:      "play-right",
		},
		{
			Keys:      opts.GetKeys(opts.Config.Universal.Return),
			Handler:   self.Escape,
			CommandID: "game.snake.cancel",
			Verb:      "cancel",
		},
	}

	return bindings
}

func (self *SnakeController) Context() types.Context {
	return self.c.Contexts().Snake
}

func (self *SnakeController) GetOnFocus() func(types.OnFocusOpts) {
	return func(types.OnFocusOpts) {
		self.c.Helpers().Snake.StartGame()
	}
}

func (self *SnakeController) GetOnFocusLost() func(types.OnFocusLostOpts) {
	return func(types.OnFocusLostOpts) {
		self.c.Helpers().Snake.ExitGame()
		self.c.Helpers().Window.MoveToTopOfWindow(self.c.Contexts().Submodules)
	}
}

func (self *SnakeController) SetDirection(direction snake.Direction) func() error {
	return func() error {
		self.c.Helpers().Snake.SetDirection(direction)
		return nil
	}
}

func (self *SnakeController) Escape() error {
	self.c.Context().Push(self.c.Contexts().Submodules, types.OnFocusOpts{})
	return nil
}
