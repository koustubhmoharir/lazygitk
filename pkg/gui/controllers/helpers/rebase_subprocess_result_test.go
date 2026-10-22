package helpers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

type subprocessRebaseRepoState struct {
	types.IRepoStateAccessor
	started bool
}

func (state *subprocessRebaseRepoState) SetMergeOrRebaseStartedInLazygit(value bool) {
	state.started = value
}

type subprocessRebaseState struct {
	types.IStateAccessor
	repo *subprocessRebaseRepoState
}

func (state *subprocessRebaseState) GetRepoState() types.IRepoStateAccessor { return state.repo }

type subprocessRebaseUI struct {
	types.IGuiCommon
	git       *commands.GitCommand
	state     *subprocessRebaseState
	refreshes []types.RefreshOptions
	menus     []types.CreateMenuOptions
}

func (ui *subprocessRebaseUI) Git() *commands.GitCommand   { return ui.git }
func (ui *subprocessRebaseUI) State() types.IStateAccessor { return ui.state }
func (ui *subprocessRebaseUI) RefreshFromWorker(opts types.RefreshOptions) {
	ui.refreshes = append(ui.refreshes, opts)
}

func (ui *subprocessRebaseUI) Menu(opts types.CreateMenuOptions) error {
	ui.menus = append(ui.menus, opts)
	return nil
}

func TestRebaseSubprocessResultUsesActualConflictState(t *testing.T) {
	for _, tc := range []struct {
		name                                      string
		rebasing, unmerged, success, queryFailure bool
	}{
		{name: "completed", success: true},
		{name: "failure before rebase"},
		{name: "unresolved conflict", rebasing: true, unmerged: true},
		{name: "paused signing failure", rebasing: true},
		{name: "index query failure", rebasing: true, queryFailure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.rebasing {
				if err := os.MkdirAll(filepath.Join(root, ".git", "rebase-merge"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			runner := oscommands.NewFakeRunner(t)
			queryErr := errors.New("cannot read index")
			if tc.rebasing && !tc.success {
				output := ""
				if tc.unmerged {
					output = "literal\nfile\x00"
				}
				var err error
				if tc.queryFailure {
					err = queryErr
				}
				runner.ExpectGitArgs([]string{"diff", "--name-only", "--diff-filter=U", "-z", "--"}, output, err)
			}
			cmn := common.NewDummyCommon()
			osCmd := oscommands.NewDummyOSCommand()
			backend := git_commands.NewGitCommon(cmn, nil, oscommands.NewDummyCmdObjBuilder(runner), osCmd, git_commands.MockRepoPaths(root), nil, nil)
			ui := &subprocessRebaseUI{
				git:   &commands.GitCommand{Status: git_commands.NewStatusCommands(backend), Rebase: git_commands.NewRebaseCommands(backend, nil, nil)},
				state: &subprocessRebaseState{repo: &subprocessRebaseRepoState{}},
			}
			helper := NewMergeAndRebaseHelper(&HelperCommon{Common: cmn, IGuiCommon: ui})
			original := errors.New("exit status 1")
			result := original
			if tc.success {
				result = nil
			}
			err := helper.CheckRebaseSubprocessResult(result, types.RefreshOptions{BatchUIUpdates: true})
			assert.Len(t, ui.refreshes, 1)
			assert.True(t, ui.refreshes[0].BatchUIUpdates)
			assert.Equal(t, tc.rebasing, ui.state.repo.started)
			if tc.unmerged {
				assert.NoError(t, err)
				if len(ui.menus) != 1 {
					t.Fatal("expected conflict recovery menu")
				}
				assert.Equal(t, cmn.Tr.FoundConflictsTitle, ui.menus[0].Title)
			} else if tc.success {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, original)
				assert.Empty(t, ui.menus, "non-conflict errors must not be mislabeled as conflicts")
			}
			if tc.queryFailure {
				assert.ErrorIs(t, err, queryErr)
			}
			runner.CheckForMissingCalls()
		})
	}
}
