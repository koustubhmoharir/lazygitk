package controllers

import (
	"errors"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type branchSyncTestState struct {
	types.IStateAccessor
	generation int
	operations map[string]types.ItemOperation
}

func (state *branchSyncTestState) GetRepoGeneration() int { return state.generation }
func (state *branchSyncTestState) GetItemOperation(item types.HasUrn) types.ItemOperation {
	return state.operations[item.URN()]
}

func (state *branchSyncTestState) SetItemOperation(item types.HasUrn, operation types.ItemOperation) {
	state.operations[item.URN()] = operation
}

func (state *branchSyncTestState) ClearItemOperation(item types.HasUrn) {
	delete(state.operations, item.URN())
}

func TestBranchSyncTargetRepositoryAndOperationScope(t *testing.T) {
	first := &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths("/repo/one")}
	second := &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths("/repo/two")}
	state := &branchSyncTestState{generation: 4, operations: map[string]types.ItemOperation{}}
	ui := &branchSyncTargetTestUI{git: first, state: state}
	common := &helpers.HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui}
	c := NewControllerCommon(common, nil)
	target := newBranchSyncTarget(c, branchSyncSource{name: "topic", selected: true})
	if err := target.checkRepository(); err != nil {
		t.Fatal(err)
	}
	if syncOperationItem(first, "topic").URN() == syncOperationItem(first, "other").URN() {
		t.Fatal("branch operation URNs collided")
	}
	if syncOperationItem(first, "topic").URN() == syncOperationItem(second, "topic").URN() {
		t.Fatal("repository operation URNs collided")
	}
	if syncOperationItem(first, "topic").URN() == "topic" {
		t.Fatal("operation URN aliases ordinary branch URN")
	}
	ui.git = second
	if err := target.checkRepository(); err == nil {
		t.Fatal("repository pointer change was accepted")
	}
	ui.git = first
	state.generation++
	if err := target.checkRepository(); err == nil {
		t.Fatal("repository generation change was accepted")
	}
}

type branchSyncTargetTestUI struct {
	types.IGuiCommon
	git       *commands.GitCommand
	state     types.IStateAccessor
	workers   []func(gocui.Task) error
	refreshes int
}

func (ui *branchSyncTargetTestUI) Git() *commands.GitCommand   { return ui.git }
func (ui *branchSyncTargetTestUI) State() types.IStateAccessor { return ui.state }
func (ui *branchSyncTargetTestUI) WithWaitingStatus(_ string, f func(gocui.Task) error) error {
	ui.workers = append(ui.workers, f)
	return nil
}
func (ui *branchSyncTargetTestUI) RefreshFromWorker(types.RefreshOptions) { ui.refreshes++ }

func TestBranchSyncTargetQueuedWorkIsRepositoryScoped(t *testing.T) {
	first := &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths("/repo/one")}
	second := &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths("/repo/two")}
	state := &branchSyncTestState{generation: 4, operations: map[string]types.ItemOperation{}}
	ui := &branchSyncTargetTestUI{git: first, state: state}
	c := NewControllerCommon(&helpers.HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui}, nil)
	target := newBranchSyncTarget(c, branchSyncSource{name: "topic", selected: true})
	if err := target.inspect(func(git_commands.BranchSyncInfo) error { t.Fatal("stale inspect callback ran"); return nil }); err != nil {
		t.Fatal(err)
	}
	ui.git = second
	state.generation++
	if err := ui.workers[0](nil); err == nil {
		t.Fatal("queued inspect worker accepted a changed repository")
	}
	if ui.refreshes != 0 {
		t.Fatal("stale inspect refreshed another repository")
	}

	ui.git = first
	state.generation++
	target = newBranchSyncTarget(c, branchSyncSource{name: "topic", selected: true})
	called := false
	if err := target.withOperation(types.ItemOperationPushing, "Pushing %q", func(gocui.Task) error { called = true; return errors.New("operation failure") }); err != nil {
		t.Fatal(err)
	}
	oldItem := syncOperationItem(first, "topic")
	newItem := syncOperationItem(second, "topic")
	state.SetItemOperation(newItem, types.ItemOperationPushing)
	ui.git = second
	state.generation++
	if err := ui.workers[1](nil); err == nil {
		t.Fatal("queued operation accepted a changed repository")
	}
	if called {
		t.Fatal("stale operation callback ran")
	}
	if state.GetItemOperation(oldItem) != types.ItemOperationNone {
		t.Fatal("captured repo operation flag was not cleared")
	}
	if state.GetItemOperation(newItem) != types.ItemOperationPushing {
		t.Fatal("another repo's same-name operation flag was cleared")
	}
	if ui.refreshes != 0 {
		t.Fatal("stale worker refreshed the new repository")
	}

	ui.git = first
	state.generation--
	target = newBranchSyncTarget(c, branchSyncSource{name: "topic", selected: true})
	if err := target.withOperation(types.ItemOperationPushing, "Pushing %q", func(gocui.Task) error { return errors.New("expected") }); err != nil {
		t.Fatal(err)
	}
	if err := ui.workers[2](nil); err == nil {
		t.Fatal("expected operation failure")
	}
	if state.GetItemOperation(syncOperationItem(first, "topic")) != types.ItemOperationNone {
		t.Fatal("failed operation left its flag set")
	}
	if ui.refreshes != 1 {
		t.Fatalf("failed operation should refresh captured repo once, got %d", ui.refreshes)
	}
}
