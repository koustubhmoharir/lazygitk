package context

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/spf13/afero"
)

const PreservedCommitMessageFileName = "LAZYGIT_PENDING_COMMIT"

type CommitMessageContext struct {
	c *ContextCommon
	types.Context
	viewModel *CommitMessageViewModel
}

var _ types.Context = (*CommitMessageContext)(nil)

// when selectedIndex (see below) is set to this value, it means that we're not
// currently viewing a commit message of an existing commit: instead we're making our own
// new commit message
const NoCommitIndex = -1

type CommitMessageViewModel struct {
	// Fork: no longer drives anything. Up and Down recall the worktree's cancelled
	// draft (see fork_commit_draft.go), not earlier commits' messages. The field
	// and the index parameters that set it stay so that upstream call sites are
	// unchanged. It was: -1 is 'no commit', 0 is the HEAD commit, 1 is the prior
	// commit, and so on.
	selectedindex int
	// if true, then upon escaping from the commit message panel, we will preserve
	// the message so that it's still shown next time we open the panel
	preserveMessage bool
	// we remember the initial message so that we can tell whether we should preserve
	// the message; if it's still identical to the initial message, we don't
	initialMessage string
	// invoked when pressing enter in the commit message panel
	onConfirm func(string, string) error
	// invoked when pressing the switch-to-editor key binding
	onSwitchToEditor func(string) error

	// the following two fields are used for the display of the "hooks disabled" subtitle
	forceSkipHooks  bool
	skipHooksPrefix string

	// Fork: the text that Up replaced with the draft, which Down puts back. It is
	// kept apart from the preserved message, which is specifically for committing
	// staged files and must not be affected by recalling the draft. Reset when the
	// panel opens and closes.
	historyMessage string
	// Fork: true while Up shows the draft in place of the text being edited.
	showingDraft bool
}

func NewCommitMessageContext(
	c *ContextCommon,
) *CommitMessageContext {
	viewModel := &CommitMessageViewModel{}
	return &CommitMessageContext{
		c:         c,
		viewModel: viewModel,
		Context: NewSimpleContext(
			NewBaseContext(NewBaseContextOpts{
				Kind:                  types.PERSISTENT_POPUP,
				View:                  c.Views().CommitMessage,
				WindowName:            "commitMessage",
				Key:                   COMMIT_MESSAGE_CONTEXT_KEY,
				Focusable:             true,
				HasUncontrolledBounds: true,
			}),
		),
	}
}

// Fork: see selectedindex.
func (self *CommitMessageContext) SetSelectedIndex(value int) {
	self.viewModel.selectedindex = value
}

// Fork: see selectedindex.
func (self *CommitMessageContext) GetSelectedIndex() int {
	return self.viewModel.selectedindex
}

func (self *CommitMessageContext) GetPreservedMessagePath() string {
	return filepath.Join(self.c.Git().RepoPaths.WorktreeGitDirPath(), PreservedCommitMessageFileName)
}

func (self *CommitMessageContext) GetPreserveMessage() bool {
	return self.viewModel.preserveMessage
}

func (self *CommitMessageContext) getPreservedMessage() (string, error) {
	buf, err := afero.ReadFile(self.c.Fs, self.GetPreservedMessagePath())
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func (self *CommitMessageContext) GetPreservedMessageAndLogError() string {
	msg, err := self.getPreservedMessage()
	if err != nil {
		self.c.Log.Errorf("error when retrieving persisted commit message: %v", err)
	}
	return msg
}

func (self *CommitMessageContext) setPreservedMessage(message string) error {
	preservedFilePath := self.GetPreservedMessagePath()

	if len(message) == 0 {
		err := self.c.Fs.Remove(preservedFilePath)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return afero.WriteFile(self.c.Fs, preservedFilePath, []byte(message), 0o644)
}

func (self *CommitMessageContext) SetPreservedMessageAndLogError(message string) {
	if err := self.setPreservedMessage(message); err != nil {
		self.c.Log.Errorf("error when persisting commit message: %v", err)
	}
}

func (self *CommitMessageContext) GetInitialMessage() string {
	return self.viewModel.initialMessage
}

// Fork: returns the text that Up replaced with the draft; see historyMessage.
func (self *CommitMessageContext) GetHistoryMessage() string {
	return self.viewModel.historyMessage
}

// Fork: see GetHistoryMessage.
func (self *CommitMessageContext) SetHistoryMessage(message string) {
	self.viewModel.historyMessage = message
}

// Fork: whether Up is showing the draft; see showingDraft.
func (self *CommitMessageContext) GetShowingDraft() bool {
	return self.viewModel.showingDraft
}

// Fork: see GetShowingDraft.
func (self *CommitMessageContext) SetShowingDraft(showing bool) {
	self.viewModel.showingDraft = showing
}

func (self *CommitMessageContext) OnConfirm(summary string, description string) error {
	return self.viewModel.onConfirm(summary, description)
}

func (self *CommitMessageContext) SetPanelState(
	index int,
	summaryTitle string,
	descriptionTitle string,
	preserveMessage bool,
	initialMessage string,
	onConfirm func(string, string) error,
	onSwitchToEditor func(string) error,
	forceSkipHooks bool,
	skipHooksPrefix string,
) {
	// Fork: index no longer drives anything; see selectedindex.
	self.viewModel.selectedindex = index
	self.viewModel.historyMessage = ""
	self.viewModel.showingDraft = false
	self.viewModel.preserveMessage = preserveMessage
	self.viewModel.initialMessage = initialMessage
	self.viewModel.onConfirm = onConfirm
	self.viewModel.onSwitchToEditor = onSwitchToEditor
	self.viewModel.forceSkipHooks = forceSkipHooks
	self.viewModel.skipHooksPrefix = skipHooksPrefix
	self.GetView().Title = summaryTitle
	self.c.Views().CommitDescription.Title = descriptionTitle

	self.c.Views().CommitDescription.Subtitle = utils.ResolvePlaceholderString(self.c.Tr.CommitDescriptionSubTitle,
		map[string]string{
			"togglePanelKeyBinding": self.c.UserConfig().Keybinding.Universal.TogglePanel.String(),
			"commitMenuKeybinding":  self.c.UserConfig().Keybinding.CommitMessage.CommitMenu.String(),
		})

	self.c.Views().CommitDescription.Visible = true
}

func (self *CommitMessageContext) RenderSubtitle() {
	skipHookPrefix := self.viewModel.skipHooksPrefix
	subject := self.c.Views().CommitMessage.TextArea.GetContent()
	var subtitle string
	if self.viewModel.forceSkipHooks || (skipHookPrefix != "" && strings.HasPrefix(subject, skipHookPrefix)) {
		subtitle = self.c.Tr.CommitHooksDisabledSubTitle
	}
	if self.c.UserConfig().Gui.CommitLength.Show {
		if subtitle != "" {
			subtitle += "─"
		}
		subtitle += getBufferLength(subject)
	}
	self.c.Views().CommitMessage.Subtitle = subtitle
}

func getBufferLength(subject string) string {
	return " " + strconv.Itoa(strings.Count(subject, "")-1) + " "
}

func (self *CommitMessageContext) SwitchToEditor(message string) error {
	return self.viewModel.onSwitchToEditor(message)
}

func (self *CommitMessageContext) CanSwitchToEditor() bool {
	return self.viewModel.onSwitchToEditor != nil
}
