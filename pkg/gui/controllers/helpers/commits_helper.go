package helpers

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

type CommitsHelper struct {
	c *HelperCommon

	getCommitSummary              func() string
	setCommitSummary              func(string)
	getCommitDescription          func() string
	getUnwrappedCommitDescription func() string
	setCommitDescription          func(string)
}

func NewCommitsHelper(
	c *HelperCommon,
	getCommitSummary func() string,
	setCommitSummary func(string),
	getCommitDescription func() string,
	getUnwrappedCommitDescription func() string,
	setCommitDescription func(string),
) *CommitsHelper {
	return &CommitsHelper{
		c:                             c,
		getCommitSummary:              getCommitSummary,
		setCommitSummary:              setCommitSummary,
		getCommitDescription:          getCommitDescription,
		getUnwrappedCommitDescription: getUnwrappedCommitDescription,
		setCommitDescription:          setCommitDescription,
	}
}

// SplitCommitMessageAndDescription splits a message in git's canonical format
// (summary and body separated by a blank line) into summary and description.
func (self *CommitsHelper) SplitCommitMessageAndDescription(message string) (string, string) {
	summary, description, _ := strings.Cut(message, "\n")
	description = strings.TrimPrefix(description, "\n")
	return summary, description
}

// SplitPreservedCommitMessage splits a message in our preservation format
// (summary and description joined by a single "\n") into summary and description.
// It is lossless: round-tripping through JoinCommitMessageAndUnwrappedDescription
// preserves the exact content.
func (self *CommitsHelper) SplitPreservedCommitMessage(message string) (string, string) {
	summary, description, _ := strings.Cut(message, "\n")
	return summary, description
}

func (self *CommitsHelper) SetMessageAndDescriptionInView(message string) {
	summary, description := self.SplitCommitMessageAndDescription(message)
	self.setSummaryAndDescriptionInView(summary, description)
}

func (self *CommitsHelper) SetPreservedMessageInView(message string) {
	summary, description := self.SplitPreservedCommitMessage(message)
	self.setSummaryAndDescriptionInView(summary, description)
}

func (self *CommitsHelper) setSummaryAndDescriptionInView(summary, description string) {
	self.setCommitSummary(summary)
	self.setCommitDescription(description)
	self.c.Contexts().CommitMessage.RenderSubtitle()
}

func (self *CommitsHelper) JoinCommitMessageAndUnwrappedDescription() string {
	if len(self.getUnwrappedCommitDescription()) == 0 {
		return self.getCommitSummary()
	}
	return self.getCommitSummary() + "\n" + self.getUnwrappedCommitDescription()
}

func TryRemoveHardLineBreaks(message string, autoWrapWidth int) string {
	lastHardLineStart := 0
	result := message
	for i, b := range message {
		if b == '\n' {
			// Try to make this a soft linebreak by turning it into a space, and
			// checking whether it still wraps to the same result then.
			str := message[lastHardLineStart:i] + " " + message[i+1:]
			softLineBreakIndices := gocui.AutoWrapContent(str, autoWrapWidth)

			// See if auto-wrapping inserted a soft line break:
			if len(softLineBreakIndices) > 0 && softLineBreakIndices[0] == i-lastHardLineStart+1 {
				// It did, so change it to a space in the result.
				result = result[:i] + " " + result[i+1:]
			}
			lastHardLineStart = i + 1
		}
	}

	return result
}

func (self *CommitsHelper) SwitchToEditor() error {
	message := lo.Ternary(len(self.getCommitDescription()) == 0,
		self.getCommitSummary(),
		self.getCommitSummary()+"\n\n"+self.getCommitDescription())
	filepath := filepath.Join(self.c.OS().GetTempDir(), self.c.Git().RepoPaths.RepoName(), time.Now().Format("Jan _2 15.04.05.000000000")+".msg")
	err := self.c.OS().CreateFileWithContent(filepath, message)
	if err != nil {
		return err
	}

	self.CloseCommitMessagePanel()

	return self.c.Contexts().CommitMessage.SwitchToEditor(filepath)
}

type OpenCommitMessagePanelOpts struct {
	// Fork: no longer drives anything. Up and Down recall the worktree's
	// cancelled draft, not earlier commits' messages; see fork_commit_draft.go.
	// Kept so that upstream call sites are unchanged.
	CommitIndex      int
	SummaryTitle     string
	DescriptionTitle string
	PreserveMessage  bool
	OnConfirm        func(summary string, description string) error
	OnSwitchToEditor func(string) error
	InitialMessage   string

	// The following two fields are only for the display of the "(hooks
	// disabled)" display in the commit message panel. They have no effect on
	// the actual behavior; make sure what you are passing in matches that.
	// Leave unassigned if the concept of skipping hooks doesn't make sense for
	// what you are doing, e.g. when creating a tag.
	ForceSkipHooks  bool
	SkipHooksPrefix string
}

func (self *CommitsHelper) OpenCommitMessagePanel(opts *OpenCommitMessagePanelOpts) {
	onConfirm := func(summary string, description string) error {
		self.CloseCommitMessagePanel()

		return opts.OnConfirm(summary, description)
	}

	// When there's no explicit initial message but we're in a preservation
	// context, fall back to any previously preserved message. This is stored as
	// the "initial" value so the unchanged-message check on close still works
	// correctly (in particular, clearing the panel then escaping will notice
	// the difference and delete the preserved file).
	initialMessage := opts.InitialMessage
	initialMessageIsPreserved := false
	if opts.PreserveMessage && initialMessage == "" {
		initialMessage = self.c.Contexts().CommitMessage.GetPreservedMessageAndLogError()
		initialMessageIsPreserved = true
	}

	self.c.Contexts().CommitMessage.SetPanelState(
		opts.CommitIndex,
		opts.SummaryTitle,
		opts.DescriptionTitle,
		opts.PreserveMessage,
		initialMessage,
		onConfirm,
		opts.OnSwitchToEditor,
		opts.ForceSkipHooks,
		opts.SkipHooksPrefix,
	)

	if initialMessageIsPreserved {
		self.SetPreservedMessageInView(initialMessage)
	} else {
		self.SetMessageAndDescriptionInView(initialMessage)
	}

	self.c.Context().Push(self.c.Contexts().CommitMessage, types.OnFocusOpts{})
}

func (self *CommitsHelper) ClearPreservedCommitMessage() {
	self.c.Contexts().CommitMessage.SetPreservedMessageAndLogError("")
}

func (self *CommitsHelper) HandleCommitConfirm() error {
	summary, description := self.getCommitSummary(), self.getCommitDescription()

	if strings.TrimSpace(summary) == "" {
		return errors.New(self.c.Tr.CommitWithoutMessageErr)
	}

	// Fork: OnConfirm closes the panel, which resets showingDraft, so read the
	// draft state before it runs.
	commitMessage := self.c.Contexts().CommitMessage
	joined := self.JoinCommitMessageAndUnwrappedDescription()
	showingDraft := commitMessage.GetShowingDraft()
	draftUsed := showingDraft || joined == commitMessage.GetCommitDraftAndLogError()

	err := commitMessage.OnConfirm(summary, description)
	if err != nil {
		return err
	}

	// Fork: submitting the draft uses it up. Submitting anything else keeps it.
	if draftUsed {
		commitMessage.SetCommitDraftAndLogError("")
	}

	return nil
}

func (self *CommitsHelper) PreserveCommitMessage() {
	if self.c.Contexts().CommitMessage.GetPreserveMessage() {
		message := self.JoinCommitMessageAndUnwrappedDescription()
		if message != self.c.Contexts().CommitMessage.GetInitialMessage() {
			self.c.Contexts().CommitMessage.SetPreservedMessageAndLogError(message)
		}
	}
}

// Fork: Escape and quit. The text becomes the worktree's draft if it is not
// blank and differs from what the panel opened with. Confirming and switching
// to the editor do not save it; see docs-master/fork/Guide.md.
func (self *CommitsHelper) SaveCommitDraftIfChanged() {
	commitMessage := self.c.Contexts().CommitMessage
	message := self.JoinCommitMessageAndUnwrappedDescription()
	if strings.TrimSpace(message) != "" && message != commitMessage.GetInitialMessage() {
		commitMessage.SetCommitDraftAndLogError(message)
	}
}

// Fork: Escape in either box. Unlike CloseCommitMessagePanel, which confirm
// also uses, this saves the draft first.
func (self *CommitsHelper) CancelCommitMessagePanel() {
	self.SaveCommitDraftIfChanged()
	self.CloseCommitMessagePanel()
}

// Fork: Up in the summary box. Shows the draft in place of the text being
// edited, keeping that text so that Down can put it back. Does nothing if the
// draft is already shown, or there is no draft, or it equals the text.
func (self *CommitsHelper) RecallCommitDraft() error {
	commitMessage := self.c.Contexts().CommitMessage
	if commitMessage.GetShowingDraft() {
		return nil
	}
	draft := commitMessage.GetCommitDraftAndLogError()
	current := self.JoinCommitMessageAndUnwrappedDescription()
	if strings.TrimSpace(draft) == "" || draft == current {
		return nil
	}
	commitMessage.SetHistoryMessage(current)
	self.SetPreservedMessageInView(draft)
	commitMessage.SetShowingDraft(true)
	return nil
}

// Fork: Down in the summary box. Writes the text on screen back to the draft,
// so edits to the draft are kept for the next Up, then puts back the text that
// Up replaced. Does nothing if the draft is not shown.
func (self *CommitsHelper) ReturnFromCommitDraft() error {
	commitMessage := self.c.Contexts().CommitMessage
	if !commitMessage.GetShowingDraft() {
		return nil
	}
	commitMessage.SetCommitDraftAndLogError(self.JoinCommitMessageAndUnwrappedDescription())
	self.SetPreservedMessageInView(commitMessage.GetHistoryMessage())
	commitMessage.SetShowingDraft(false)
	return nil
}

func (self *CommitsHelper) CloseCommitMessagePanel() {
	self.PreserveCommitMessage()

	// Fork: the draft state is per opening of the panel.
	self.c.Contexts().CommitMessage.SetShowingDraft(false)
	self.c.Contexts().CommitMessage.SetHistoryMessage("")

	self.c.Views().CommitMessage.Visible = false
	self.c.Views().CommitDescription.Visible = false

	self.c.Context().Pop()
}

func (self *CommitsHelper) OpenCommitMenu(suggestionFunc func(string) []*types.Suggestion) error {
	var disabledReasonForOpenInEditor *types.DisabledReason
	if !self.c.Contexts().CommitMessage.CanSwitchToEditor() {
		disabledReasonForOpenInEditor = &types.DisabledReason{
			Text: self.c.Tr.CommandDoesNotSupportOpeningInEditor,
		}
	}

	menuItems := []*types.MenuItem{
		{
			Label:     self.c.Tr.OpenInEditor,
			CommandID: "commit-editor.open-external-editor",
			Verb:      "edit-commit-message",
			OnPress: func() error {
				return self.SwitchToEditor()
			},
			Keys:           menuKey('e'),
			DisabledReason: disabledReasonForOpenInEditor,
		},
		{
			Label:     self.c.Tr.AddCoAuthor,
			CommandID: "commit-editor.add-coauthor",
			Verb:      "add-coauthor",
			OnPress: func() error {
				return self.addCoAuthor(suggestionFunc)
			},
			Keys: menuKey('c'),
		},
		{
			Label:     self.c.Tr.PasteCommitMessageFromClipboard,
			CommandID: "commit-editor.paste-message",
			Verb:      "paste-message",
			OnPress: func() error {
				return self.pasteCommitMessageFromClipboard()
			},
			Keys: menuKey('p'),
		},
	}
	return self.c.Menu(types.CreateMenuOptions{
		Title: self.c.Tr.CommitMenuTitle,
		Items: menuItems,
	})
}

func (self *CommitsHelper) addCoAuthor(suggestionFunc func(string) []*types.Suggestion) error {
	self.c.Prompt(types.PromptOpts{
		Title:               self.c.Tr.AddCoAuthorPromptTitle,
		FindSuggestionsFunc: suggestionFunc,
		HandleConfirm: func(value string) error {
			commitDescription := self.getCommitDescription()
			commitDescription = git_commands.AddCoAuthorToDescription(commitDescription, value)
			self.setCommitDescription(commitDescription)
			return nil
		},
	})

	return nil
}

func (self *CommitsHelper) pasteCommitMessageFromClipboard() error {
	message, err := self.c.OS().PasteFromClipboard()
	if err != nil {
		return err
	}
	if message == "" {
		return nil
	}

	currentMessage := self.JoinCommitMessageAndUnwrappedDescription()
	return self.c.ConfirmIf(currentMessage != "", types.ConfirmOpts{
		Title:  self.c.Tr.PasteCommitMessageFromClipboard,
		Prompt: self.c.Tr.SurePasteCommitMessage,
		HandleConfirm: func() error {
			self.SetMessageAndDescriptionInView(message)
			return nil
		},
	})
}
