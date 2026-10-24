package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type branchPlacementRefTestState struct{ types.IStateAccessor }

func (branchPlacementRefTestState) GetRepoGeneration() int { return 0 }

func (ui *placementFlowUI) State() types.IStateAccessor { return branchPlacementRefTestState{} }

func TestBranchRefPlacementResolvesCapturedRef(t *testing.T) {
	for _, remote := range []bool{false, true} {
		t.Run(map[bool]string{false: "local", true: "remote"}[remote], func(t *testing.T) {
			controller, ui, root, commits := placementFlowFixture(t)
			ref := "refs/heads/topic"
			if remote {
				ref = "refs/remotes/origin/topic"
				placementFlowGit(t, root, "update-ref", ref, commits[1])
				branchController := &RemoteBranchesController{c: controller.c}
				if err := branchController.beginPlacementAtRemoteBranch(&models.RemoteBranch{Name: "topic", RemoteName: "origin"}); err != nil {
					t.Fatal(err)
				}
			} else {
				branchController := &BranchesController{c: controller.c}
				if err := branchController.beginPlacementAtLocalBranch(&models.Branch{Name: "topic"}); err != nil {
					t.Fatal(err)
				}
			}
			// Resolve the captured ref after its displayed row could have changed.
			placementFlowGit(t, root, "update-ref", ref, commits[2], commits[1])
			ui.runWorker(t)
			ui.runUI(t)
			if len(ui.prompts) != 1 || len(ui.confirmations) != 0 {
				t.Fatal("expected the shared name prompt")
			}
			if err := ui.prompts[0].HandleConfirm("placed"); err != nil {
				t.Fatal(err)
			}
			ui.runWorker(t)
			ui.runUI(t)
			if len(ui.confirmations) != 0 {
				t.Fatal("fresh target unexpectedly requested a move warning")
			}
			ui.runWorker(t)
			if got := placementFlowGit(t, root, "rev-parse", "refs/heads/placed"); got != commits[2] {
				t.Fatalf("branch placed at %s, want resolved ref tip %s", got, commits[2])
			}
		})
	}
}

type branchRefPlacementChangingState struct {
	types.IStateAccessor
	generation int
}

func (s *branchRefPlacementChangingState) GetRepoGeneration() int { return s.generation }

type branchRefPlacementChangingUI struct {
	*placementFlowUI
	state branchRefPlacementChangingState
}

func (ui *branchRefPlacementChangingUI) State() types.IStateAccessor { return &ui.state }

func TestBranchRefPlacementRejectsRepositoryChangeDuringNameEntry(t *testing.T) {
	controller, original, root, _ := placementFlowFixture(t)
	ui := &branchRefPlacementChangingUI{placementFlowUI: original}
	controller.c.IGuiCommon = ui
	if err := beginBranchPlacementAtRef(controller.c, "refs/heads/topic"); err != nil {
		t.Fatal(err)
	}
	ui.runWorker(t)
	ui.runUI(t)
	ui.state.generation++
	if err := ui.prompts[0].HandleConfirm("must-not-create"); err == nil {
		t.Fatal("name entry followed a changed repository generation")
	}
	if len(ui.workers) != 0 {
		t.Fatal("repository change queued a mutation")
	}
	if got := placementFlowGit(t, root, "branch", "--list", "must-not-create"); got != "" {
		t.Fatal("created branch after repository change")
	}
}
