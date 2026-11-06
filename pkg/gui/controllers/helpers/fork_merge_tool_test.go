package helpers

import (
	"errors"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_config"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

type mergeToolUI struct {
	*namedCheckoutFakeIGuiCommon
	state         *capturedTagState
	subprocessErr error
	subprocesses  int
	refreshes     []types.RefreshOptions
}

func (u *mergeToolUI) State() types.IStateAccessor { return u.state }
func (u *mergeToolUI) Refresh(opts types.RefreshOptions) {
	u.refreshes = append(u.refreshes, opts)
}

func (u *mergeToolUI) RunSubprocess(*oscommands.CmdObj) (bool, error) {
	u.subprocesses++
	return false, u.subprocessErr
}

// RunSubprocessAndRefresh has the semantics of the production gui method: a failed
// subprocess returns before the refresh.
func (u *mergeToolUI) RunSubprocessAndRefresh(cmd *oscommands.CmdObj) error {
	if _, err := u.RunSubprocess(cmd); err != nil {
		return err
	}
	u.Refresh(types.RefreshOptions{DontBlockRepoSwitch: true})
	return nil
}

// A merge tool can resolve some files before it fails, so Files must refresh after a
// failed run too, and the tool's error is still returned.
func TestOpenMergeToolRefreshesFilesWhenToolFails(t *testing.T) {
	toolErr := errors.New("exit status 1")
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "tool succeeds"},
		{name: "tool fails after partial resolution", err: toolErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newNamedCheckoutFixture(t)
			namedCheckoutGit(t, f.root, "config", "merge.tool", "fake")
			// The merge tool commands need the whole Git command set, not the branch
			// commands the named checkout fixture provides.
			osCmd := oscommands.NewDummyOSCommand()
			f.ui.git = commands.NewGitCommandAux(common.NewDummyCommon(), nil, osCmd, git_config.NewFakeGitConfig(nil), git_commands.MockRepoPaths(f.root), nil)
			u := &mergeToolUI{namedCheckoutFakeIGuiCommon: f.ui, state: &capturedTagState{}, subprocessErr: tc.err}
			f.c.IGuiCommon = u
			helper := &WorkingTreeHelper{c: f.c, externalTool: NewExternalToolHelper(f.c)}

			err := helper.OpenMergeTool([]string{"conflicted"})
			assert.Equal(t, 1, u.subprocesses)
			if tc.err != nil {
				assert.ErrorIs(t, err, toolErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Len(t, u.refreshes, 1, "Files must refresh whether the tool succeeded or failed")
		})
	}
}
