package context

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/jesseduffield/lazygit/pkg/utils/filelock"
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

// Fork: the lock file beside the drafts file. See updateCommitDrafts.
func (self *CommitMessageContext) commitDraftsLockPath() string {
	return self.GetCommitDraftsPath() + ".lock"
}

var errMalformedCommitDrafts = errors.New("commit drafts file is malformed")

// Reads the drafts without the lock. The file is replaced atomically, so a read
// sees either the old or the new version. A malformed file is an error here, and
// updateCommitDrafts is what moves it aside.
func (self *CommitMessageContext) readCommitDrafts() ([]string, error) {
	buf, err := afero.ReadFile(self.c.Fs, self.GetCommitDraftsPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error when retrieving commit drafts: %w", err)
	}
	var drafts []string
	if err := json.Unmarshal(buf, &drafts); err != nil {
		return nil, fmt.Errorf("%w: %w", errMalformedCommitDrafts, err)
	}
	return drafts, nil
}

// Used to show the drafts. A read error is logged and reads as no drafts.
func (self *CommitMessageContext) loadCommitDrafts() []string {
	drafts, err := self.readCommitDrafts()
	if err != nil {
		self.c.Log.Errorf("%v", err)
	}
	return drafts
}

// Fork: the drafts file is shared by every lazygitk instance on this worktree.
// Applies change to the drafts as they are on disk, under the drafts file's lock,
// and saves the result. Every change goes through here: a change based on an
// earlier read would overwrite the drafts that another instance has added since.
//
// A malformed file is moved aside, not overwritten, so that its contents are not
// lost; the drafts then start empty. Returns an error if the file cannot be read
// or written, or if the lock cannot be taken. In that case the drafts are unchanged.
func (self *CommitMessageContext) updateCommitDrafts(change func([]string) []string) error {
	lock, err := filelock.Acquire(self.commitDraftsLockPath(), filelock.DefaultTimeout)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()

	drafts, err := self.readCommitDrafts()
	if errors.Is(err, errMalformedCommitDrafts) {
		aside := self.GetCommitDraftsPath() + ".corrupt-" + time.Now().Format("20060102-150405.000000000")
		self.c.Log.Errorf("%v; moving it to %s and starting with no drafts", err, aside)
		if renameErr := self.c.Fs.Rename(self.GetCommitDraftsPath(), aside); renameErr != nil {
			return renameErr
		}
		self.c.ErrorToast(utils.ResolvePlaceholderString(self.c.Tr.CommitDraftsMovedAside, map[string]string{"path": aside}))
		drafts = nil
	} else if err != nil {
		return err
	}
	return self.writeCommitDrafts(change(drafts))
}

// Writes through a temporary file, because a partial write would lose every
// draft rather than one message. An empty list removes the file.
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
	// A unique name, so that no other writer can share or clobber the temporary file.
	tmp, err := afero.TempFile(self.c.Fs, filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	// Removes the temporary file if the rename below does not happen.
	defer func() { _ = self.c.Fs.Remove(tmpPath) }()

	if _, err := tmp.Write(buf); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := self.c.Fs.Chmod(tmpPath, 0o644); err != nil {
		return err
	}
	return self.c.Fs.Rename(tmpPath, path)
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
// older draft, or past the end if there is none. If the edits cannot be saved,
// nothing changes and the error is returned.
func (self *CommitMessageContext) keepCommitDraftEdits(text string) (bool, error) {
	session := &self.viewModel.drafts
	shown := session.stops[session.index]
	if text == shown {
		return false, nil
	}
	if err := self.updateCommitDrafts(func(drafts []string) []string {
		return replaceCommitDraft(drafts, shown, text)
	}); err != nil {
		return false, err
	}
	before := len(session.stops)
	session.stops = replaceCommitDraft(session.stops, shown, text)
	return len(session.stops) < before, nil
}

// Up. Takes the text on screen and returns the text to show instead, or false to
// leave it. Fork: if the edits to the shown draft cannot be saved, nothing moves
// and the error is returned, so the text stays as it is.
func (self *CommitMessageContext) StepToOlderCommitDraft(text string) (string, bool, error) {
	session := &self.viewModel.drafts
	if session.index == noCommitDraft {
		stops := lo.Filter(self.loadCommitDrafts(), func(draft string, _ int) bool {
			return draft != text
		})
		if len(stops) == 0 {
			return "", false, nil
		}
		session.stops = stops
		session.typed = text
		session.index = 0
		return stops[0], true, nil
	}

	removed, err := self.keepCommitDraftEdits(text)
	if err != nil {
		return "", false, err
	}
	if removed {
		if len(session.stops) == 0 {
			session.index = noCommitDraft
			return session.typed, true, nil
		}
		session.index = min(session.index, len(session.stops)-1)
		return session.stops[session.index], true, nil
	}
	if session.index == len(session.stops)-1 {
		return "", false, nil
	}
	session.index++
	return session.stops[session.index], true, nil
}

// Down. Takes the text on screen and returns the text to show instead, or false
// to leave it. Fork: if the edits to the shown draft cannot be saved, nothing
// moves and the error is returned.
func (self *CommitMessageContext) StepToNewerCommitDraft(text string) (string, bool, error) {
	session := &self.viewModel.drafts
	if session.index == noCommitDraft {
		return "", false, nil
	}
	if _, err := self.keepCommitDraftEdits(text); err != nil {
		return "", false, err
	}
	session.index--
	if session.index == noCommitDraft {
		return session.typed, true, nil
	}
	return session.stops[session.index], true, nil
}

// Escape, quit and submit before the result is known: keeps the text on screen,
// and the text that Up replaced. Returns an error if the drafts cannot be saved,
// in which case the caller must keep the message box open.
func (self *CommitMessageContext) KeepCommitDraftsOnCancel(text string) error {
	session := self.viewModel.drafts
	return self.updateCommitDrafts(func(drafts []string) []string {
		if session.index == noCommitDraft {
			return keepTypedCommitDraft(drafts, session.opened, text)
		}
		drafts = replaceCommitDraft(drafts, session.stops[session.index], text)
		return keepTypedCommitDraft(drafts, session.opened, session.typed)
	})
}

// Switching to the editor takes the text on screen along. If that is a draft,
// the text that Up replaced is otherwise lost, so it is kept. Returns an error if
// that cannot be saved, in which case the caller must not switch.
func (self *CommitMessageContext) KeepTypedCommitDraftOnSwitchToEditor() error {
	session := self.viewModel.drafts
	if session.index == noCommitDraft {
		return nil
	}
	return self.updateCommitDrafts(func(drafts []string) []string {
		return keepTypedCommitDraft(drafts, session.opened, session.typed)
	})
}

// Submitting uses up the draft on screen, and the draft the box opened with if
// that is what was edited. The text that Up replaced is kept. Call the returned
// function once the submission has succeeded; it does not depend on the box
// still being open. Its error is for the caller to log: the commit has already
// succeeded, so nothing the user typed is lost by failing to update the drafts.
func (self *CommitMessageContext) CommitDraftsUsedBy(text string) func() error {
	session := self.viewModel.drafts
	return func() error {
		return self.updateCommitDrafts(func(drafts []string) []string {
			if session.index == noCommitDraft {
				drafts = removeCommitDraft(drafts, session.opened)
			} else {
				drafts = removeCommitDraft(drafts, session.stops[session.index])
				drafts = keepTypedCommitDraft(drafts, session.opened, session.typed)
			}
			return removeCommitDraft(drafts, text)
		})
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
