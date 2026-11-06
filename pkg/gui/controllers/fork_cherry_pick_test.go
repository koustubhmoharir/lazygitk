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

// forkCherryPickPickSource is the selected list with a list that can clear its selection,
// as the real commit list does.
type forkCherryPickPickSource struct{ *historicalDiffCommitsStub }

func (s forkCherryPickPickSource) GetList() types.IList {
	return forkCherryPickClearableList{source: s.historicalDiffCommitsStub}
}

// forkCherryPickClearableList clears the stub's selection; the cursor methods are unused.
type forkCherryPickClearableList struct {
	types.IList
	source *historicalDiffCommitsStub
}

func (l forkCherryPickClearableList) CancelRangeSelect() { l.source.items = nil }

func forkCherryPickFixture(t *testing.T) (*ControllerCommon, *forkCherryPickTestUI, *historicalDiffCommitsStub, string, []string) {
	t.Helper()
	return forkCherryPickFixtureAt(t, 2, 51, 0)
}

// forkCherryPickFixtureAt builds the fixture with the given Git version, and with the
// destination at the base commit (destination 0) or at the first pick (destination 1),
// which already contains that pick's change.
func forkCherryPickFixtureAt(t *testing.T, major, minor, destination int) (*ControllerCommon, *forkCherryPickTestUI, *historicalDiffCommitsStub, string, []string) {
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
	selectedRebaseFlowGit(t, root, "checkout", "-qb", "destination", oids[destination])
	cmn := common.NewDummyCommon()
	git := commands.NewGitCommandAux(cmn, &git_commands.GitVersion{Major: major, Minor: minor}, oscommands.NewDummyOSCommand(), git_config.NewFakeGitConfig(nil), git_commands.MockRepoPaths(root), nil)
	selection := &historicalDiffCommitsStub{items: []*models.Commit{historicalTestCommit(oids[2]), historicalTestCommit(oids[1])}, end: 1}
	ui := &forkCherryPickTestUI{selectedRebaseFlowUI: &selectedRebaseFlowUI{git: git, modes: &types.Modes{CherryPicking: cherrypicking.New()}, state: &forkCherryPickTestState{}}, contextMgr: &forkCherryPickTestContextMgr{current: forkCherryPickPickSource{selection}}}
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

func TestForkCherryPickDisabledWithoutSelection(t *testing.T) {
	c, _, selection, _, _ := forkCherryPickFixture(t)
	selection.items = nil
	reason := forkCherryPickDisabledReason(c)
	if reason == nil || reason.Text != c.Tr.NoCommitsSelected {
		t.Fatalf("disabled reason = %v, want %q", reason, c.Tr.NoCommitsSelected)
	}
}

func TestForkCherryPickSuccessLeavesChangedSelection(t *testing.T) {
	c, ui, selection, root, oids := forkCherryPickFixture(t)
	confirm := forkCherryPickReview(t, c, ui)
	// The selection moved after the review: a successful pick must not clear it.
	selection.items = []*models.Commit{historicalTestCommit(oids[0])}
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
		t.Fatal("pick did not apply")
	}
	if len(selection.items) != 1 || selection.items[0].Hash() != oids[0] {
		t.Fatal("successful pick changed a selection the user made since")
	}
}

// Before Git 2.45 the pick has no --empty=keep, so an already-applied first pick stops
// Git. The helper skips it and the sequence finishes; the selection must still be cleared.
func TestForkCherryPickEmptyPickAutoSkipClearsSelection(t *testing.T) {
	c, ui, selection, root, _ := forkCherryPickFixtureAt(t, 2, 44, 1)
	confirm := forkCherryPickReview(t, c, ui)
	if err := confirm.HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if selectedRebaseFlowGit(t, root, "show", "HEAD:second") != "second" || c.Git().Status.WorkingTreeState().Any() {
		t.Fatal("auto-skip did not finish the sequence")
	}
	forkCherryPickDrainUI(t, ui)
	if len(selection.items) != 0 {
		t.Fatalf("picked selection %v was not cleared", selection.items)
	}
}

// forkCherryPickDrainUI runs every queued UI callback, in order, as the UI loop does.
func forkCherryPickDrainUI(t *testing.T, ui *forkCherryPickTestUI) {
	t.Helper()
	for len(ui.ui) > 0 {
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
	}
}

// A pick that stops on conflicts keeps the selection and registers the cleanup; the
// cleanup runs once the operation is continued.
func TestForkCherryPickConflictKeepsSelectionUntilContinued(t *testing.T) {
	c, ui, selection, root, oids := forkCherryPickFixture(t)
	// Both sides change "base" after the fixture's base commit, so picking the side commit
	// is a content conflict, which the helper recognizes.
	selectedRebaseFlowGit(t, root, "checkout", "-qb", "side", oids[0])
	if err := os.WriteFile(filepath.Join(root, "base"), []byte("side\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	selectedRebaseFlowGit(t, root, "commit", "-qam", "side")
	side := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD")
	selectedRebaseFlowGit(t, root, "checkout", "-q", "destination")
	if err := os.WriteFile(filepath.Join(root, "base"), []byte("destination\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	selectedRebaseFlowGit(t, root, "commit", "-qam", "destination")
	selection.items = []*models.Commit{historicalTestCommit(side)}
	confirm := forkCherryPickReview(t, c, ui)
	if err := confirm.HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if !c.Git().Status.WorkingTreeState().CherryPicking {
		t.Fatal("pick did not stop on a conflict")
	}
	forkCherryPickDrainUI(t, ui)
	if len(selection.items) != 1 {
		t.Fatal("conflicting pick cleared the selection")
	}
	if err := os.WriteFile(filepath.Join(root, "base"), []byte("resolved\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	selectedRebaseFlowGit(t, root, "add", "base")
	if err := c.Helpers().MergeAndRebase.ContinueRebase(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	forkCherryPickDrainUI(t, ui)
	if len(selection.items) != 0 {
		t.Fatal("continuing the conflicted pick did not clear the selection")
	}
}

// A failed pick is an error and keeps the selection.
func TestForkCherryPickErrorKeepsSelection(t *testing.T) {
	c, ui, selection, root, _ := forkCherryPickFixture(t)
	if err := os.WriteFile(filepath.Join(root, "first"), []byte("user data\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	confirm := forkCherryPickReview(t, c, ui)
	if err := confirm.HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err == nil {
		t.Fatal("native untracked collision was not reported")
	}
	if len(ui.ui) != 0 || len(selection.items) != 2 {
		t.Fatal("failed pick settled the selection")
	}
}
