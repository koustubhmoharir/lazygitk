package context

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/afero"
)

// Fork: the worktree's cancelled commit messages. Up and Down in the commit
// message box step through them. They sit beside
// PreservedCommitMessageFileName, but are not that message: the preserved
// message is the pending text of a new commit, and the drafts are recalled for
// any commit message box. See docs-master/fork/Guide.md.
const CommitDraftsFileName = "LAZYGITK_COMMIT_DRAFTS"

// As many as TortoiseGit's recent messages keep by default.
const maxCommitDrafts = 25

// Fork: what Up and Down have done since the box opened.
type commitDraftSession struct {
	// The text the box opened with. If that is a draft, as when a new commit's
	// box reopens with its cancelled text, then editing it edits that draft.
	opened string
	// The drafts that Up and Down step through, newest first. Taken when Up
	// leaves the typed text.
	stops []string
	// The position in stops, or noCommitDraft while the typed text is showing.
	index int
	// The text that Up replaced with the first draft.
	typed string
}

const noCommitDraft = -1

// Each of the functions below takes the list newest first and returns a new one.

func removeCommitDraft(drafts []string, message string) []string {
	return lo.Without(drafts, message)
}

// Puts the message first, removing any older copy of it.
func addCommitDraft(drafts []string, message string) []string {
	result := append([]string{message}, removeCommitDraft(drafts, message)...)
	if len(result) > maxCommitDrafts {
		result = result[:maxCommitDrafts]
	}
	return result
}

// An edit keeps the draft's position. Clearing a draft removes it, and so does
// editing it to match another draft.
func replaceCommitDraft(drafts []string, oldMessage string, newMessage string) []string {
	if strings.TrimSpace(newMessage) == "" || (newMessage != oldMessage && lo.Contains(drafts, newMessage)) {
		return removeCommitDraft(drafts, oldMessage)
	}
	return lo.Map(drafts, func(draft string, _ int) string {
		return lo.Ternary(draft == oldMessage, newMessage, draft)
	})
}

// Records the text being typed when the box closes without using it. Text that
// is unchanged from what the box opened with is not a draft. Otherwise it
// replaces the draft that the box opened with, if any, and becomes the newest.
func keepTypedCommitDraft(drafts []string, opened string, typed string) []string {
	if typed == opened {
		return drafts
	}
	drafts = removeCommitDraft(drafts, opened)
	if strings.TrimSpace(typed) == "" {
		return drafts
	}
	return addCommitDraft(drafts, typed)
}

func (self *CommitMessageContext) GetCommitDraftsPath() string {
	return filepath.Join(self.c.Git().RepoPaths.WorktreeGitDirPath(), CommitDraftsFileName)
}

func (self *CommitMessageContext) loadCommitDrafts() []string {
	buf, err := afero.ReadFile(self.c.Fs, self.GetCommitDraftsPath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		self.c.Log.Errorf("error when retrieving commit drafts: %v", err)
		return nil
	}
	var drafts []string
	if err := json.Unmarshal(buf, &drafts); err != nil {
		self.c.Log.Errorf("error when parsing commit drafts: %v", err)
		return nil
	}
	return drafts
}

// Writes through a temporary file, because a partial write would lose every
// draft rather than one message. An empty list removes the file.
func (self *CommitMessageContext) saveCommitDrafts(drafts []string) {
	if err := self.writeCommitDrafts(drafts); err != nil {
		self.c.Log.Errorf("error when persisting commit drafts: %v", err)
	}
}

func (self *CommitMessageContext) writeCommitDrafts(drafts []string) error {
	path := self.GetCommitDraftsPath()
	if len(drafts) == 0 {
		err := self.c.Fs.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	buf, err := json.Marshal(drafts)
	if err != nil {
		return err
	}
	tempPath := path + ".tmp"
	if err := afero.WriteFile(self.c.Fs, tempPath, buf, 0o644); err != nil {
		return err
	}
	return self.c.Fs.Rename(tempPath, path)
}

func (self *CommitMessageContext) resetCommitDraftSession() {
	self.viewModel.drafts = commitDraftSession{index: noCommitDraft}
}

// Called once the box shows the text it opened with.
func (self *CommitMessageContext) SetCommitDraftOpenedText(text string) {
	self.viewModel.drafts.opened = text
}

// Called when the box closes.
func (self *CommitMessageContext) EndCommitDraftSession() {
	self.resetCommitDraftSession()
}

// Writes any edits to the shown draft to the file, and to stops. Returns whether
// that removed the draft from stops, in which case index is now at the next
// older draft, or past the end if there is none.
func (self *CommitMessageContext) keepCommitDraftEdits(text string) bool {
	session := &self.viewModel.drafts
	shown := session.stops[session.index]
	if text == shown {
		return false
	}
	self.saveCommitDrafts(replaceCommitDraft(self.loadCommitDrafts(), shown, text))
	before := len(session.stops)
	session.stops = replaceCommitDraft(session.stops, shown, text)
	return len(session.stops) < before
}

// Up. Takes the text on screen and returns the text to show instead, or false to
// leave it.
func (self *CommitMessageContext) StepToOlderCommitDraft(text string) (string, bool) {
	session := &self.viewModel.drafts
	if session.index == noCommitDraft {
		stops := lo.Filter(self.loadCommitDrafts(), func(draft string, _ int) bool {
			return draft != text
		})
		if len(stops) == 0 {
			return "", false
		}
		session.stops = stops
		session.typed = text
		session.index = 0
		return stops[0], true
	}

	if self.keepCommitDraftEdits(text) {
		if len(session.stops) == 0 {
			session.index = noCommitDraft
			return session.typed, true
		}
		session.index = min(session.index, len(session.stops)-1)
		return session.stops[session.index], true
	}
	if session.index == len(session.stops)-1 {
		return "", false
	}
	session.index++
	return session.stops[session.index], true
}

// Down. Takes the text on screen and returns the text to show instead, or false
// to leave it.
func (self *CommitMessageContext) StepToNewerCommitDraft(text string) (string, bool) {
	session := &self.viewModel.drafts
	if session.index == noCommitDraft {
		return "", false
	}
	self.keepCommitDraftEdits(text)
	session.index--
	if session.index == noCommitDraft {
		return session.typed, true
	}
	return session.stops[session.index], true
}

// Escape and quit: keeps the text on screen, and the text that Up replaced.
func (self *CommitMessageContext) KeepCommitDraftsOnCancel(text string) {
	session := self.viewModel.drafts
	if session.index == noCommitDraft {
		self.saveCommitDrafts(keepTypedCommitDraft(self.loadCommitDrafts(), session.opened, text))
		return
	}
	drafts := replaceCommitDraft(self.loadCommitDrafts(), session.stops[session.index], text)
	self.saveCommitDrafts(keepTypedCommitDraft(drafts, session.opened, session.typed))
}

// Switching to the editor takes the text on screen along. If that is a draft,
// the text that Up replaced is otherwise lost, so it is kept.
func (self *CommitMessageContext) KeepTypedCommitDraftOnSwitchToEditor() {
	session := self.viewModel.drafts
	if session.index != noCommitDraft {
		self.saveCommitDrafts(keepTypedCommitDraft(self.loadCommitDrafts(), session.opened, session.typed))
	}
}

// Submitting uses up the draft on screen, and the draft the box opened with if
// that is what was edited. The text that Up replaced is kept. Call the returned
// function once the submission has succeeded; it does not depend on the box
// still being open.
func (self *CommitMessageContext) CommitDraftsUsedBy(text string) func() {
	session := self.viewModel.drafts
	return func() {
		drafts := self.loadCommitDrafts()
		if session.index == noCommitDraft {
			drafts = removeCommitDraft(drafts, session.opened)
		} else {
			drafts = removeCommitDraft(drafts, session.stops[session.index])
			drafts = keepTypedCommitDraft(drafts, session.opened, session.typed)
		}
		self.saveCommitDrafts(removeCommitDraft(drafts, text))
	}
}

// The position of the shown draft, counting from 1, and how many there are to
// step through. Zero if no draft is shown.
func (self *CommitMessageContext) commitDraftPosition() (int, int) {
	session := self.viewModel.drafts
	if session.index == noCommitDraft {
		return 0, 0
	}
	return session.index + 1, len(session.stops)
}
