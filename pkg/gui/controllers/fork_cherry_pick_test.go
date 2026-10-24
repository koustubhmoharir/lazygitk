package controllers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_config"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/modes/cherrypicking"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type forkCherryPickTestContextMgr struct {
	types.IContextMgr
	current types.Context
}

func (m *forkCherryPickTestContextMgr) CurrentStatic() types.Context { return m.current }
func (m *forkCherryPickTestContextMgr) CurrentSide() types.Context   { return m.current }

type forkCherryPickTestState struct {
	types.IStateAccessor
	generation int
	repo       selectedRebaseFlowRepoState
}

func (s *forkCherryPickTestState) GetRepoGeneration() int                 { return s.generation }
func (s *forkCherryPickTestState) GetRepoState() types.IRepoStateAccessor { return &s.repo }

type forkCherryPickTestUI struct {
	*selectedRebaseFlowUI
	contextMgr *forkCherryPickTestContextMgr
}

func (ui *forkCherryPickTestUI) Context() types.IContextMgr { return ui.contextMgr }
func (ui *forkCherryPickTestUI) LogAction(string)           {}

func forkCherryPickFixture(t *testing.T) (*ControllerCommon, *forkCherryPickTestUI, *historicalDiffCommitsStub, string, []string) {
	t.Helper()
	root := t.TempDir()
	selectedRebaseFlowGit(t, root, "init", "-q")
	selectedRebaseFlowGit(t, root, "config", "user.name", "Test")
	selectedRebaseFlowGit(t, root, "config", "user.email", "test@example.invalid")
	oids := make([]string, 0, 3)
	for _, name := range []string{"base", "first", "second"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		selectedRebaseFlowGit(t, root, "add", name)
		selectedRebaseFlowGit(t, root, "commit", "-qm", name)
		oids = append(oids, selectedRebaseFlowGit(t, root, "rev-parse", "HEAD"))
	}
	selectedRebaseFlowGit(t, root, "checkout", "-qb", "destination", oids[0])
	cmn := common.NewDummyCommon()
	git := commands.NewGitCommandAux(cmn, &git_commands.GitVersion{Major: 2, Minor: 51}, oscommands.NewDummyOSCommand(), git_config.NewFakeGitConfig(nil), git_commands.MockRepoPaths(root), nil)
	selection := &historicalDiffCommitsStub{items: []*models.Commit{historicalTestCommit(oids[2]), historicalTestCommit(oids[1])}, end: 1}
	ui := &forkCherryPickTestUI{selectedRebaseFlowUI: &selectedRebaseFlowUI{git: git, modes: &types.Modes{CherryPicking: cherrypicking.New()}, state: &forkCherryPickTestState{}}, contextMgr: &forkCherryPickTestContextMgr{current: selection}}
	c := NewControllerCommon(&helpers.HelperCommon{Common: cmn, IGuiCommon: ui}, nil)
	c.IGetHelpers = selectedRebaseFlowHelpers{value: &helpers.Helpers{MergeAndRebase: helpers.NewMergeAndRebaseHelper(c.HelperCommon)}}
	return c, ui, selection, root, oids
}

func forkCherryPickReview(t *testing.T, c *ControllerCommon, ui *forkCherryPickTestUI) types.ConfirmOpts {
	t.Helper()
	if err := beginForkCherryPick(c); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.confirmations) == 0 {
		t.Fatal("missing exact-list confirmation")
	}
	return ui.confirmations[len(ui.confirmations)-1]
}

func TestForkCherryPickSelectedReviewOrdersAndRequiresApproval(t *testing.T) {
	c, ui, selection, root, oids := forkCherryPickFixture(t)
	confirm := forkCherryPickReview(t, c, ui)
	if len(ui.menus) != 0 {
		t.Fatal("selected-only flow must not ask which buffer to use")
	}
	if first, second := strings.Index(confirm.Prompt, oids[1]), strings.Index(confirm.Prompt, oids[2]); first < 0 || second < first {
		t.Fatalf("missing ancestor-first exact list: %s", confirm.Prompt)
	}
	if !strings.Contains(confirm.Prompt, "refs/heads/destination") || !strings.Contains(confirm.Prompt, oids[0]) {
		t.Fatalf("missing current destination: %s", confirm.Prompt)
	}
	selection.items = []*models.Commit{historicalTestCommit(oids[0])}
	if selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") != oids[0] {
		t.Fatal("preparation moved HEAD")
	}
	if err := confirm.HandleClose(); err != nil {
		t.Fatal(err)
	}
	if selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") != oids[0] {
		t.Fatal("cancel moved HEAD")
	}
	if err := confirm.HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if selectedRebaseFlowGit(t, root, "show", "HEAD:first") != "first" || selectedRebaseFlowGit(t, root, "show", "HEAD:second") != "second" {
		t.Fatal("pick did not use the captured set")
	}
	if err := confirm.HandleConfirm(); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("replayed approval = %v", err)
	}
	if selectedRebaseFlowGit(t, root, "stash", "list") != "" {
		t.Fatal("pick used a stash")
	}
}

func TestForkCherryPickMarkedAndSelectedAreSeparateChoices(t *testing.T) {
	c, ui, selection, root, oids := forkCherryPickFixture(t)
	ui.modes.CherryPicking.MarkCommits([]*models.Commit{historicalTestCommit(oids[1])})
	selection.items = []*models.Commit{historicalTestCommit(oids[2])}
	if err := beginForkCherryPick(c); err != nil {
		t.Fatal(err)
	}
	if len(ui.menus) != 1 || len(ui.workers) != 0 {
		t.Fatal("ambiguous sources were not presented before inspection")
	}
	menu := ui.menus[0]
	if len(menu.Items) != 2 || menu.Items[0].CommandID != "commit.cherry-pick.marked" || menu.Items[1].CommandID != "commit.cherry-pick.selected" {
		t.Fatal("source choices must keep marks first and selection separate")
	}
	ui.modes.CherryPicking.CherryPickedCommits = nil
	if err := menu.Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	confirm := ui.confirmations[0]
	if !strings.Contains(confirm.Prompt, oids[1]) || strings.Contains(confirm.Prompt, oids[2]) {
		t.Fatal("source choice silently merged or retargeted marks")
	}
	if err := confirm.HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if selectedRebaseFlowGit(t, root, "show", "HEAD:first") != "first" {
		t.Fatal("marked pick did not apply")
	}
	if ui.modes.CherryPicking.DidPaste {
		t.Fatal("an unrelated replacement mark buffer was hidden")
	}
}

func TestForkCherryPickStaleHeadRequiresFreshReview(t *testing.T) {
	c, ui, _, root, oids := forkCherryPickFixture(t)
	confirm := forkCherryPickReview(t, c, ui)
	selectedRebaseFlowGit(t, root, "commit", "--allow-empty", "-qm", "external change")
	changed := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD")
	if err := confirm.HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") != changed {
		t.Fatal("stale approval ran Git")
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.confirmations) != 2 || !strings.Contains(ui.confirmations[1].Prompt, changed) || !strings.Contains(ui.confirmations[1].Prompt, oids[1]) {
		t.Fatal("missing refreshed exact-list review")
	}
	if err := confirm.HandleConfirm(); !errors.Is(err, git_commands.ErrBranchActionChanged) {
		t.Fatalf("old review remained reusable: %v", err)
	}
}

func TestForkCherryPickRefusesRepositorySwitchAndNativeCollision(t *testing.T) {
	t.Run("repository", func(t *testing.T) {
		c, ui, _, root, oids := forkCherryPickFixture(t)
		confirm := forkCherryPickReview(t, c, ui)
		ui.state.(*forkCherryPickTestState).generation++
		if err := confirm.HandleConfirm(); err == nil {
			t.Fatal("repository switch accepted")
		}
		if len(ui.workers) != 0 || selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") != oids[0] {
			t.Fatal("switched repository was mutated")
		}
	})
	t.Run("collision", func(t *testing.T) {
		c, ui, _, root, oids := forkCherryPickFixture(t)
		confirm := forkCherryPickReview(t, c, ui)
		if err := os.WriteFile(filepath.Join(root, "first"), []byte("user data\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := confirm.HandleConfirm(); err != nil {
			t.Fatal(err)
		}
		if err := ui.runWorker(t); err == nil {
			t.Fatal("native untracked collision was not reported")
		}
		data, err := os.ReadFile(filepath.Join(root, "first"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "user data\n" || selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") != oids[0] || selectedRebaseFlowGit(t, root, "stash", "list") != "" {
			t.Fatal("collision overwrote data, moved HEAD or used a stash")
		}
	})
}

func TestForkCherryPickSuccessPreservesUpstreamHiddenPickBuffer(t *testing.T) {
	c, ui, selection, _, oids := forkCherryPickFixture(t)
	selection.items = nil
	ui.modes.CherryPicking.MarkCommits([]*models.Commit{historicalTestCommit(oids[1])})
	confirm := forkCherryPickReview(t, c, ui)
	if err := confirm.HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if ui.modes.CherryPicking.Active() || !ui.modes.CherryPicking.CanPaste() || !ui.modes.CherryPicking.DidPaste {
		t.Fatal("successful marked pick changed upstream's retained-buffer behavior")
	}
}
