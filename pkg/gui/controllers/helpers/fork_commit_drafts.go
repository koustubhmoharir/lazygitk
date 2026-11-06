package helpers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/utils"
)

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
