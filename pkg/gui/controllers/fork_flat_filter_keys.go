package controllers

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: w p, w u and w c are flat log filter keys from any pane. They replace
// the entries of upstream's filtering menu (filtering_menu_action.go), which is left
// unchanged.
// w p and w u open the path and author prompts of the filtering menu, prefilled
// with the selected file or commit author. w c clears the log filter and is
// offered only while one is active. The filtering menu keeps its command ID for
// the palette, but w o no longer has a key. The integration point is
// GlobalController.GetKeybindings.
func (self *GlobalController) addFlatFilterBindings(opts types.KeybindingsOpts, bindings []*types.Binding) []*types.Binding {
	return append(bindings,
		&types.Binding{
			CommandID:   "filter.by-path",
			Verb:        "filter",
			Description: self.c.Tr.FilterByPath,
			Handler: opts.Guards.NoPopupPanel(func() error {
				promptFilterPath(self.c, selectedFilterPath(self.c))
				return nil
			}),
		},
		&types.Binding{
			CommandID:   "filter.by-author",
			Verb:        "filter",
			Description: self.c.Tr.FilterByAuthor,
			Handler: opts.Guards.NoPopupPanel(func() error {
				promptFilterAuthor(self.c, selectedFilterAuthor(self.c))
				return nil
			}),
		},
		&types.Binding{
			CommandID:   "filter.clear",
			Verb:        "clear-filter",
			Description: self.c.Tr.ClearLogFilter,
			Handler: opts.Guards.NoPopupPanel(func() error {
				return self.c.Helpers().Mode.ClearFiltering()
			}),
			GetDisabledReason: func() *types.DisabledReason {
				if !self.c.Modes().Filtering.Active() {
					return &types.DisabledReason{Text: self.c.Tr.NoApplicableCommandsInThisContext, Inapplicable: true}
				}
				return nil
			},
		},
	)
}

// selectedFilterPath is the path of the file selected in Files or Commit files, or
// empty. It follows FilteringMenuAction.Call.
func selectedFilterPath(c *ControllerCommon) string {
	switch c.Context().CurrentSide() {
	case c.Contexts().Files:
		if node := c.Contexts().Files.GetSelected(); node != nil {
			return node.GetPath()
		}
	case c.Contexts().CommitFiles:
		if node := c.Contexts().CommitFiles.GetSelected(); node != nil {
			return node.GetPath()
		}
	}
	return ""
}

// selectedFilterAuthor is "Name <email>" of the commit selected in Commits, or empty.
// It follows FilteringMenuAction.Call.
func selectedFilterAuthor(c *ControllerCommon) string {
	if c.Context().CurrentSide() == c.Contexts().LocalCommits {
		if commit := c.Contexts().LocalCommits.GetSelected(); commit != nil {
			return fmt.Sprintf("%s <%s>", commit.AuthorName, commit.AuthorEmail)
		}
	}
	return ""
}

// promptFilterPath asks for the path to filter the log by, with initial as the text.
// It is the menu's path prompt with initial content.
func promptFilterPath(c *ControllerCommon, initial string) {
	c.Prompt(types.PromptOpts{
		FindSuggestionsFunc: c.Helpers().Suggestions.GetFilePathSuggestionsFunc(),
		Title:               c.Tr.EnterFileName,
		InitialContent:      initial,
		HandleConfirm: func(response string) error {
			return c.Helpers().Mode.SetFilteringPath(response)
		},
	})
}

// promptFilterAuthor asks for the author to filter the log by, with initial as the text.
// It is the menu's author prompt with initial content.
func promptFilterAuthor(c *ControllerCommon, initial string) {
	c.Prompt(types.PromptOpts{
		FindSuggestionsFunc: c.Helpers().Suggestions.GetAuthorsSuggestionsFunc(),
		Title:               c.Tr.EnterAuthor,
		InitialContent:      initial,
		HandleConfirm: func(response string) error {
			return c.Helpers().Mode.SetFilteringAuthor(response)
		},
	})
}
