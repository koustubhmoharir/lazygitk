package controllers

import (
	"errors"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type forkDiffDiscardState struct {
	types.IStateAccessor
	generation int
}

func (state *forkDiffDiscardState) GetRepoGeneration() int { return state.generation }

type forkDiffDiscardUI struct {
	types.IGuiCommon
	git      *commands.GitCommand
	state    *forkDiffDiscardState
	confirms []types.ConfirmOpts
}

func (ui *forkDiffDiscardUI) Git() *commands.GitCommand   { return ui.git }
func (ui *forkDiffDiscardUI) State() types.IStateAccessor { return ui.state }
func (ui *forkDiffDiscardUI) Confirm(opts types.ConfirmOpts) {
	ui.confirms = append(ui.confirms, opts)
}

func newForkDiffDiscardActions(t *testing.T, skipWarning bool) (*WorkingTreeDiffActions, *forkDiffDiscardUI) {
	t.Helper()
	userConfig := config.GetDefaultConfig()
	userConfig.Gui.SkipDiscardChangeWarning = skipWarning
	cmn := common.NewDummyCommon()
	cmn.SetUserConfig(userConfig)
	ui := &forkDiffDiscardUI{
		git:   &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths(t.TempDir())},
		state: &forkDiffDiscardState{generation: 7},
	}
	c := NewControllerCommon(&helpers.HelperCommon{Common: cmn, IGuiCommon: ui}, nil)
	return NewWorkingTreeDiffActions(c), ui
}

func TestForkDiffDiscardStagedAppliesImmediatelyAndPropagatesError(t *testing.T) {
	for _, skipWarning := range []bool{false, true} {
		t.Run(map[bool]string{false: "warning-enabled", true: "warning-skipped"}[skipWarning], func(t *testing.T) {
			actions, ui := newForkDiffDiscardActions(t, skipWarning)
			wantErr := errors.New("unstage failed")
			calls := 0
			err := actions.confirmDiffDiscard(true, func() error { calls++; return wantErr })
			if !errors.Is(err, wantErr) {
				t.Fatalf("error = %v, want %v", err, wantErr)
			}
			if calls != 1 {
				t.Fatalf("apply calls = %d, want 1", calls)
			}
			if len(ui.confirms) != 0 {
				t.Fatalf("staged discard prompted %d times", len(ui.confirms))
			}
		})
	}
}

func TestForkDiffDiscardUnstagedRequiresConfirmationRegardlessOfWarningConfig(t *testing.T) {
	for _, skipWarning := range []bool{false, true} {
		t.Run(map[bool]string{false: "warning-enabled", true: "warning-skipped"}[skipWarning], func(t *testing.T) {
			actions, ui := newForkDiffDiscardActions(t, skipWarning)
			calls := 0
			if err := actions.confirmDiffDiscard(false, func() error { calls++; return nil }); err != nil {
				t.Fatal(err)
			}
			if calls != 0 {
				t.Fatal("unstaged change was applied before confirmation")
			}
			if len(ui.confirms) != 1 {
				t.Fatalf("confirmations = %d, want 1", len(ui.confirms))
			}
			prompt := ui.confirms[0].Prompt
			if strings.Contains(strings.ToLower(prompt), "skipdiscardchangewarning") || strings.Contains(strings.ToLower(prompt), "disable") {
				t.Fatalf("prompt suggests disabling the warning: %q", prompt)
			}

			// Closing the confirmation has no callback; leave it unapproved and
			// verify that cancellation does not apply the operation.
			if ui.confirms[0].HandleClose != nil {
				if err := ui.confirms[0].HandleClose(); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 0 {
				t.Fatal("cancelled discard applied the change")
			}
			if err := ui.confirms[0].HandleConfirm(); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("apply calls after approval = %d, want 1", calls)
			}
		})
	}
}

func TestForkDiffDiscardUnstagedRejectsRepositoryChange(t *testing.T) {
	for _, changeGit := range []bool{true, false} {
		t.Run(map[bool]string{true: "git-pointer", false: "generation"}[changeGit], func(t *testing.T) {
			actions, ui := newForkDiffDiscardActions(t, false)
			calls := 0
			if err := actions.confirmDiffDiscard(false, func() error { calls++; return nil }); err != nil {
				t.Fatal(err)
			}
			if len(ui.confirms) != 1 {
				t.Fatalf("confirmations = %d, want 1", len(ui.confirms))
			}
			if changeGit {
				ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths(t.TempDir())}
			} else {
				ui.state.generation++
			}
			if err := ui.confirms[0].HandleConfirm(); err == nil {
				t.Fatal("repository change was accepted")
			}
			if calls != 0 {
				t.Fatal("repository change ran the discard operation")
			}
		})
	}
}
