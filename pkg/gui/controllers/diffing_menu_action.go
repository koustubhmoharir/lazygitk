package controllers

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/gui/modes/diffing"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type DiffingMenuAction struct {
	c *ControllerCommon
}

func (self *DiffingMenuAction) Call() error {
	names := self.c.Helpers().Diff.CurrentDiffTerminals()

	menuItems := []*types.MenuItem{}
	for _, name := range names {
		menuItems = append(menuItems, []*types.MenuItem{
			{
				Label:     fmt.Sprintf("%s %s", self.c.Tr.Diff, name),
				CommandID: "diff.set-reference",
				Verb:      "set-diff-reference",
				OnPress: func() error {
					self.c.Modes().Diffing.Ref = name
					// can scope this down based on current view but too lazy right now
					self.c.Refresh(types.RefreshOptions{})
					return nil
				},
			},
		}...)
	}

	menuItems = append(menuItems, []*types.MenuItem{
		{
			Label:     self.c.Tr.EnterRefToDiff,
			CommandID: "diff.prompt-reference",
			Verb:      "set-diff-reference",
			OnPress: func() error {
				self.c.Prompt(types.PromptOpts{
					Title:               self.c.Tr.EnterRefName,
					FindSuggestionsFunc: self.c.Helpers().Suggestions.GetRefsSuggestionsFunc(),
					HandleConfirm: func(response string) error {
						self.c.Modes().Diffing.Ref = response
						self.c.Refresh(types.RefreshOptions{})
						return nil
					},
				})

				return nil
			},
		},
	}...)

	if self.c.Modes().Diffing.Active() {
		menuItems = append(menuItems, []*types.MenuItem{
			{
				Label:     self.c.Tr.SwapDiff,
				CommandID: "diff.swap-endpoints",
				Verb:      "swap-diff",
				OnPress: func() error {
					self.c.Modes().Diffing.Reverse = !self.c.Modes().Diffing.Reverse
					self.c.Refresh(types.RefreshOptions{})
					return nil
				},
			},
			{
				Label:     self.c.Tr.ExitDiffMode,
				CommandID: "diff.exit-mode",
				Verb:      "exit-diff-mode",
				OnPress: func() error {
					self.c.Modes().Diffing = diffing.New()
					self.c.Refresh(types.RefreshOptions{})
					return nil
				},
			},
		}...)
	}

	return self.c.Menu(types.CreateMenuOptions{Title: self.c.Tr.DiffingMenuTitle, Items: menuItems})
}
