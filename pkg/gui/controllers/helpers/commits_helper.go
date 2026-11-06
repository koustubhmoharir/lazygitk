package helpers

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

type CommitsHelper struct {
	c *HelperCommon

	getCommitSummary              func() string
	setCommitSummary              func(string)
	getCommitDescription          func() string
	getUnwrappedCommitDescription func() string
	setCommitDescription          func(string)

	// Fork: whether the open panel's OnConfirm reports its result through
	// TakeCommitMessageSubmitReporter. See OpenCommitMessagePanelOpts.
	submitReportsResult bool
	// Fork: takes the drafts used by the submission in progress, for OnConfirm to
	// claim with TakeCommitMessageSubmitReporter.
	pendingDraftUse func() error
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

	// Fork: the drafts are saved before the message file is created, so a failed
	// save leaves nothing behind and the message stays in the box. When the result
	// is reported later, the text is kept as a draft until it succeeds, as
	// HandleCommitConfirm does. Otherwise the text that Up replaced is kept.
	if self.submitReportsResult {
		commitMessage := self.c.Contexts().CommitMessage
		text := self.JoinCommitMessageAndUnwrappedDescription()
		useDrafts := commitMessage.CommitDraftsUsedBy(text)
		if err := commitMessage.KeepCommitDraftsOnCancel(text); err != nil {
			return draftNotSavedError(self.c.Tr.CommitDraftNotSavedBeforeEditor, err)
		}
		self.pendingDraftUse = useDrafts
	} else if err := self.c.Contexts().CommitMessage.KeepTypedCommitDraftOnSwitchToEditor(); err != nil {
		return draftNotSavedError(self.c.Tr.CommitDraftNotSavedBeforeEditor, err)
	}

	filepath := filepath.Join(self.c.OS().GetTempDir(), self.c.Git().RepoPaths.RepoName(), time.Now().Format("Jan _2 15.04.05.000000000")+".msg")
	err := self.c.OS().CreateFileWithContent(filepath, message)
	if err != nil {
		self.pendingDraftUse = nil
		return err
	}

	self.CloseCommitMessagePanel()

	err = self.c.Contexts().CommitMessage.SwitchToEditor(filepath)
	self.pendingDraftUse = nil
	return err
}

type OpenCommitMessagePanelOpts struct {
	// Fork: no longer drives anything. Up and Down recall the worktree's
	// cancelled drafts, not earlier commits' messages; see fork_commit_draft.go.
	// Kept so that upstream call sites are unchanged.
	CommitIndex      int
	SummaryTitle     string
	DescriptionTitle string
	PreserveMessage  bool
	OnConfirm        func(summary string, description string) error
	OnSwitchToEditor func(string) error
	InitialMessage   string
	// Fork: OnConfirm's work finishes later, and it reports success through
	// CommitsHelper.TakeCommitMessageSubmitReporter. Until
	// then the text stays a draft, so a failed submission is not lost.
	ReportsSubmitResult bool

	// The following two fields are only for the display of the "(hooks
	// disabled)" display in the commit message panel. They have no effect on
	// the actual behavior; make sure what you are passing in matches that.
	// Leave unassigned if the concept of skipping hooks doesn't make sense for
	// what you are doing, e.g. when creating a tag.
	ForceSkipHooks  bool
	SkipHooksPrefix string
}

func (self *CommitsHelper) OpenCommitMessagePanel(opts *OpenCommitMessagePanelOpts) {
	// Fork: see OpenCommitMessagePanelOpts.ReportsSubmitResult.
	self.submitReportsResult = opts.ReportsSubmitResult

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
	// Fork: taken from the box, because initialMessage may be in another format.
	self.c.Contexts().CommitMessage.SetCommitDraftOpenedText(self.JoinCommitMessageAndUnwrappedDescription())

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

	// Fork: OnConfirm closes the panel, which ends the draft session, so take what
	// the session knows before it runs.
	commitMessage := self.c.Contexts().CommitMessage
	text := self.JoinCommitMessageAndUnwrappedDescription()
	useDrafts := commitMessage.CommitDraftsUsedBy(text)

	// Fork: when the result is reported later, the text is kept as a draft until
	// it succeeds, so that a failure does not lose it. Keeping it here, before
	// OnConfirm, also keeps the text that Up replaced. If that cannot be saved,
	// nothing is submitted and the box stays open.
	if self.submitReportsResult {
		if err := commitMessage.KeepCommitDraftsOnCancel(text); err != nil {
			return draftNotSavedError(self.c.Tr.CommitDraftNotSavedBeforeSubmit, err)
		}
		self.pendingDraftUse = useDrafts
	}

	err := commitMessage.OnConfirm(summary, description)
	self.pendingDraftUse = nil
	if err != nil {
		return err
	}

	if !self.submitReportsResult {
		// Fork: the commit has succeeded, so a failure to update the drafts only
		// means a stale draft is left behind. Nothing the user typed is lost.
		if err := useDrafts(); err != nil {
			self.c.Log.Errorf("error when updating commit drafts after commit: %v", err)
		}
	}

	return nil
}

// Fork: the OnConfirm of a panel opened with ReportsSubmitResult calls this before
// it starts its work, and calls the returned function once that has succeeded,
// from any goroutine. Taking it up front ties it to this submission, even if
// another box is submitted while the work runs.
func (self *CommitsHelper) TakeCommitMessageSubmitReporter() func() {
	useDrafts := self.pendingDraftUse
	self.pendingDraftUse = nil
	return func() {
		if useDrafts != nil {
			self.c.OnUIThread(func() error {
				// Fork: the commit has succeeded, so a failure here is only logged; see
				// HandleCommitConfirm.
				if err := useDrafts(); err != nil {
					self.c.Log.Errorf("error when updating commit drafts after commit: %v", err)
				}
				return nil
			})
		}
	}
}

func (self *CommitsHelper) PreserveCommitMessage() {
	if self.c.Contexts().CommitMessage.GetPreserveMessage() {
		message := self.JoinCommitMessageAndUnwrappedDescription()
		if message != self.c.Contexts().CommitMessage.GetInitialMessage() {
			self.c.Contexts().CommitMessage.SetPreservedMessageAndLogError(message)
		}
	}
}

// Fork: Escape and quit keep the text as a draft. Confirming and switching to
// the editor keep it only while a reported submit is in progress, until it
// succeeds (see TakeCommitMessageSubmitReporter); otherwise they do not. See
// docs-master/fork/Guide.md.
func (self *CommitsHelper) SaveCommitDraftIfChanged() error {
	return self.c.Contexts().CommitMessage.KeepCommitDraftsOnCancel(self.JoinCommitMessageAndUnwrappedDescription())
}

// Fork: Escape in either box. Unlike CloseCommitMessagePanel, which confirm
// also uses, this saves the draft first. If the draft cannot be saved, the box
// stays open with its text, and the error is returned for the user to see.
func (self *CommitsHelper) CancelCommitMessagePanel() error {
	if err := self.SaveCommitDraftIfChanged(); err != nil {
		return draftNotSavedError(self.c.Tr.CommitDraftNotSavedOnCancel, err)
	}
	self.CloseCommitMessagePanel()
	return nil
}

// Fork: the error shown when a draft cannot be saved, which keeps the box open.
func draftNotSavedError(template string, err error) error {
	return errors.New(utils.ResolvePlaceholderString(template, map[string]string{"error": err.Error()}))
}

// Fork: Up in the summary box. If the shown text cannot be saved as a draft,
// the text is left as it is and the error is shown.
func (self *CommitsHelper) RecallOlderCommitDraft() error {
	text, ok, err := self.c.Contexts().CommitMessage.StepToOlderCommitDraft(self.JoinCommitMessageAndUnwrappedDescription())
	if err != nil {
		return draftNotSavedError(self.c.Tr.CommitDraftNotSavedOnRecall, err)
	}
	if ok {
		self.SetPreservedMessageInView(text)
	}
	return nil
}

// Fork: Down in the summary box. See RecallOlderCommitDraft.
func (self *CommitsHelper) RecallNewerCommitDraft() error {
	text, ok, err := self.c.Contexts().CommitMessage.StepToNewerCommitDraft(self.JoinCommitMessageAndUnwrappedDescription())
	if err != nil {
		return draftNotSavedError(self.c.Tr.CommitDraftNotSavedOnRecall, err)
	}
	if ok {
		self.SetPreservedMessageInView(text)
	}
	return nil
}

func (self *CommitsHelper) CloseCommitMessagePanel() {
	self.PreserveCommitMessage()

	// Fork: the draft session is per opening of the panel.
	self.c.Contexts().CommitMessage.EndCommitDraftSession()

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
