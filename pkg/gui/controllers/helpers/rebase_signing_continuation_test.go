package helpers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_config"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

type rebaseSigningContinuationUI struct {
	*subprocessRebaseUI
	queuedWorkers       int
	commands            [][]string
	subprocessErr       error
	lockRetriesDisabled []bool
}

func (ui *rebaseSigningContinuationUI) RunSubprocess(cmd *oscommands.CmdObj) (bool, error) {
	ui.commands = append(ui.commands, cmd.Args())
	ui.lockRetriesDisabled = append(ui.lockRetriesDisabled, cmd.LockRetriesDisabled())
	return false, ui.subprocessErr
}

func (ui *rebaseSigningContinuationUI) Refresh(opts types.RefreshOptions) {
	ui.refreshes = append(ui.refreshes, opts)
}

func (ui *rebaseSigningContinuationUI) LogAction(string) {}

func (ui *rebaseSigningContinuationUI) LogCommand(string, bool) {}

func (ui *rebaseSigningContinuationUI) Model() *types.Model { return &types.Model{} }

func (ui *rebaseSigningContinuationUI) WithWaitingStatus(_ string, _ func(gocui.Task) error) error {
	ui.queuedWorkers++
	return nil
}

func newRebaseSigningContinuationHelper(t *testing.T, overrideGpg bool) (*MergeAndRebaseHelper, *rebaseSigningContinuationUI) {
	t.Helper()
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if !assert.NoError(t, os.MkdirAll(filepath.Join(gitDir, "rebase-merge"), 0o700)) {
		t.FailNow()
	}

	cmn := common.NewDummyCommon()
	cmn.UserConfig().Git.OverrideGpg = overrideGpg
	runner := oscommands.NewFakeRunner(t)
	backend := git_commands.NewGitCommon(cmn, nil, oscommands.NewDummyCmdObjBuilder(runner), oscommands.NewDummyOSCommand(), git_commands.MockRepoPaths(root), nil, nil)
	gitConfig := git_commands.NewConfigCommands(cmn, git_config.NewFakeGitConfig(map[string]string{"commit.gpgSign": "true"}))
	baseUI := &subprocessRebaseUI{
		git:   &commands.GitCommand{Status: git_commands.NewStatusCommands(backend), Rebase: git_commands.NewRebaseCommands(backend, nil, nil), Config: gitConfig, RepoPaths: git_commands.MockRepoPaths(root)},
		state: &subprocessRebaseState{repo: &subprocessRebaseRepoState{}},
	}
	ui := &rebaseSigningContinuationUI{subprocessRebaseUI: baseUI, subprocessErr: errors.New("signing prompt failed")}
	helper := NewMergeAndRebaseHelper(&HelperCommon{Common: cmn, IGuiCommon: ui})
	return helper, ui
}

func TestRebaseSigningContinueRoutesThroughSubprocess(t *testing.T) {
	helper, ui := newRebaseSigningContinuationHelper(t, false)
	err := helper.ContinueRebase()
	assert.EqualError(t, err, "signing prompt failed")
	if !assert.Len(t, ui.commands, 1) {
		return
	}
	assert.Equal(t, []string{"git", "rebase", "--continue"}, ui.commands[0])
	assert.Equal(t, []bool{true}, ui.lockRetriesDisabled)
	assert.Zero(t, ui.queuedWorkers)
	assert.Len(t, ui.refreshes, 1)
	assert.True(t, ui.state.repo.started)
}

func TestRebaseSigningSkipRoutesThroughSubprocess(t *testing.T) {
	helper, ui := newRebaseSigningContinuationHelper(t, false)
	err := helper.genericMergeCommand(REBASE_OPTION_SKIP)
	assert.EqualError(t, err, "signing prompt failed")
	if !assert.Len(t, ui.commands, 1) {
		return
	}
	assert.Equal(t, []string{"git", "rebase", "--skip"}, ui.commands[0])
	assert.Equal(t, []bool{true}, ui.lockRetriesDisabled)
	assert.Zero(t, ui.queuedWorkers)
	assert.Len(t, ui.refreshes, 1)
	assert.True(t, ui.state.repo.started)
}

func TestRebaseSigningAbortUsesNormalWorker(t *testing.T) {
	helper, ui := newRebaseSigningContinuationHelper(t, false)
	err := helper.genericMergeCommand(REBASE_OPTION_ABORT)
	assert.NoError(t, err)
	assert.Empty(t, ui.commands)
	assert.Equal(t, 1, ui.queuedWorkers)
	assert.Empty(t, ui.refreshes)
	assert.False(t, ui.state.repo.started)
}

func TestRebaseSigningOverrideUsesNormalWorker(t *testing.T) {
	helper, ui := newRebaseSigningContinuationHelper(t, true)
	err := helper.ContinueRebase()
	assert.NoError(t, err)
	assert.Empty(t, ui.commands)
	assert.Equal(t, 1, ui.queuedWorkers)
	assert.Empty(t, ui.refreshes)
	assert.False(t, ui.state.repo.started)
}
