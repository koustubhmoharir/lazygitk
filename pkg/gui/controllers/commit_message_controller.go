package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

type CommitMessageController struct {
	baseController
	c *ControllerCommon
}

var _ types.IController = &CommitMessageController{}

func NewCommitMessageController(
	c *ControllerCommon,
) *CommitMessageController {
	return &CommitMessageController{
		baseController: baseController{},
		c:              c,
	}
}

func (self *CommitMessageController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	bindings := []*types.Binding{
		{
			Keys:        opts.GetKeys(opts.Config.Universal.SubmitEditorText),
			Handler:     self.confirm,
			Description: self.c.Tr.Confirm,
			CommandID:   "commit-editor.confirm",
			Verb:        "confirm",
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.Return),
			Handler:     self.close,
			Description: self.c.Tr.Close,
			CommandID:   "commit-editor.cancel",
			Verb:        "cancel",
		},
		// Fork: Up and Down recall the cancelled drafts, not earlier commits'
		// messages. See docs-master/fork/Guide.md.
		{
			Keys:        opts.GetKeys(opts.Config.Universal.PrevItem),
			Handler:     self.handleRecallOlderDraft,
			Description: self.c.Tr.RecallOlderCommitDraft,
			CommandID:   "commit-editor.message.previous",
			Verb:        "previous-message",
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.NextItem),
			Handler:     self.handleRecallNewerDraft,
			Description: self.c.Tr.RecallNewerCommitDraft,
			CommandID:   "commit-editor.message.next",
			Verb:        "next-message",
		},
		{
			Keys:      opts.GetKeys(opts.Config.Universal.TogglePanel),
			Handler:   self.handleTogglePanel,
			CommandID: "commit-editor.focus-description",
			Verb:      "focus-description",
		},
		{
			Keys:      opts.GetKeys(opts.Config.CommitMessage.CommitMenu),
			Handler:   self.openCommitMenu,
			CommandID: "commit-editor.menu",
			Verb:      "commit-options",
		},
	}

	return bindings
}

func (self *CommitMessageController) GetMouseKeybindings(opts types.KeybindingsOpts) []*gocui.ViewMouseBinding {
	return []*gocui.ViewMouseBinding{
		{
			ViewName:    self.Context().GetViewName(),
			FocusedView: self.c.Contexts().CommitDescription.GetViewName(),
			Key:         gocui.MouseLeft,
			Handler:     self.onClick,
		},
	}
}

func (self *CommitMessageController) GetOnFocus() func(types.OnFocusOpts) {
	return func(types.OnFocusOpts) {
		self.c.Views().CommitDescription.Footer = ""
	}
}

func (self *CommitMessageController) GetOnFocusLost() func(types.OnFocusLostOpts) {
	return func(types.OnFocusLostOpts) {
		self.context().RenderSubtitle()
	}
}

func (self *CommitMessageController) GetOnQuit() func() {
	return func() {
		self.c.Helpers().Commits.PreserveCommitMessage()
		// Fork: quitting with the box open keeps its text as the draft too.
		self.c.Helpers().Commits.SaveCommitDraftIfChanged()
	}
}

func (self *CommitMessageController) Context() types.Context {
	return self.context()
}

func (self *CommitMessageController) context() *context.CommitMessageContext {
	return self.c.Contexts().CommitMessage
}

// Fork: see CommitsHelper.RecallOlderCommitDraft.
func (self *CommitMessageController) handleRecallOlderDraft() error {
	return self.c.Helpers().Commits.RecallOlderCommitDraft()
}

// Fork: see CommitsHelper.RecallNewerCommitDraft.
func (self *CommitMessageController) handleRecallNewerDraft() error {
	return self.c.Helpers().Commits.RecallNewerCommitDraft()
}

func (self *CommitMessageController) switchToCommitDescription() error {
	self.c.Context().Replace(self.c.Contexts().CommitDescription)
	return nil
}

func (self *CommitMessageController) handleTogglePanel() error {
	// The default keybinding for this action is "<tab>", which means that we
	// also get here when pasting multi-line text that contains tabs. In that
	// case we don't want to toggle the panel, but insert the tab as a character
	// (somehow, see below).
	//
	// Only do this if the TogglePanel command is actually mapped to "<tab>"
	// (the default). If it's not, we can only hope that it's mapped to some
	// ctrl key or fn key, which is unlikely to occur in pasted text. And if
	// they mapped some *other* command to "<tab>", then we're totally out of
	// luck.
	if self.c.GocuiGui().IsPasting && lo.Contains(self.c.UserConfig().Keybinding.Universal.TogglePanel, "<tab>") {
		// It is unlikely that a pasted commit message contains a tab in the
		// subject line, so it shouldn't matter too much how we handle it.
		// Simply insert 4 spaces instead; all that matters is that we don't
		// switch to the description panel.
		view := self.context().GetView()
		for range 4 {
			view.Editor.Edit(view, gocui.NewKeyRune(' '))
		}
		return nil
	}

	return self.switchToCommitDescription()
}

func (self *CommitMessageController) confirm() error {
	// The default keybinding for this action is "<enter>", which means that we
	// also get here when pasting multi-line text that contains newlines. In
	// that case we don't want to confirm the commit, but switch to the
	// description panel instead so that the rest of the pasted text goes there.
	//
	// Only do this if the SubmitEditorText command is actually mapped to
	// "<enter>" (the default). If it's not, we can only hope that it's mapped
	// to some ctrl key or fn key, which is unlikely to occur in pasted text.
	// And if they mapped some *other* command to "<enter>", then we're totally
	// out of luck.
	if self.c.GocuiGui().IsPasting && lo.Contains(self.c.UserConfig().Keybinding.Universal.SubmitEditorText, "<enter>") {
		return self.switchToCommitDescription()
	}

	return self.c.Helpers().Commits.HandleCommitConfirm()
}

// Fork: Escape saves the draft; see CommitsHelper.CancelCommitMessagePanel.
func (self *CommitMessageController) close() error {
	self.c.Helpers().Commits.CancelCommitMessagePanel()
	return nil
}

func (self *CommitMessageController) openCommitMenu() error {
	authorSuggestion := self.c.Helpers().Suggestions.GetAuthorsSuggestionsFunc()
	return self.c.Helpers().Commits.OpenCommitMenu(authorSuggestion)
}

func (self *CommitMessageController) onClick(opts gocui.ViewMouseBindingOpts) error {
	self.c.Context().Replace(self.c.Contexts().CommitMessage)
	return nil
}
