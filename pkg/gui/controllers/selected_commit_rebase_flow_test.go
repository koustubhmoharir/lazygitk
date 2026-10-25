package controllers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_config"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type selectedRebaseFlowUI struct {
	types.IGuiCommon
	git           *commands.GitCommand
	model         types.Model
	modes         *types.Modes
	menus         []types.CreateMenuOptions
	confirmations []types.ConfirmOpts
	workers       []func(gocui.Task) error
	ui            []func() error
	refreshes     []types.RefreshOptions
	runCalls      int
	state         types.IStateAccessor
}

func (ui *selectedRebaseFlowUI) Git() *commands.GitCommand   { return ui.git }
func (ui *selectedRebaseFlowUI) Model() *types.Model         { return &ui.model }
func (ui *selectedRebaseFlowUI) Modes() *types.Modes         { return ui.modes }
func (ui *selectedRebaseFlowUI) State() types.IStateAccessor { return ui.state }
func (ui *selectedRebaseFlowUI) Menu(opts types.CreateMenuOptions) error {
	ui.menus = append(ui.menus, opts)
	return nil
}

func (ui *selectedRebaseFlowUI) Confirm(opts types.ConfirmOpts) {
	ui.confirmations = append(ui.confirmations, opts)
}
func (ui *selectedRebaseFlowUI) OnUIThread(f func() error) { ui.ui = append(ui.ui, f) }

// Mode resets re-render list contexts; the flows under test do not render.
func (ui *selectedRebaseFlowUI) Contexts() *context.ContextTree  { return &context.ContextTree{} }
func (ui *selectedRebaseFlowUI) PostRefreshUpdate(types.Context) {}
func (ui *selectedRebaseFlowUI) Refresh(opts types.RefreshOptions) {
	ui.refreshes = append(ui.refreshes, opts)
}

func (ui *selectedRebaseFlowUI) RefreshFromWorker(opts types.RefreshOptions) {
	ui.refreshes = append(ui.refreshes, opts)
}

func (ui *selectedRebaseFlowUI) RunSubprocess(*oscommands.CmdObj) (bool, error) {
	ui.runCalls++
	return false, fmt.Errorf("unexpected subprocess")
}

func (ui *selectedRebaseFlowUI) WithWaitingStatus(_ string, f func(gocui.Task) error) error {
	ui.workers = append(ui.workers, f)
	return nil
}

func (ui *selectedRebaseFlowUI) WithWaitingStatusBlockingInput(_ types.WaitingStatusOpts, f func(gocui.Task) error) error {
	ui.workers = append(ui.workers, f)
	return nil
}

func (ui *selectedRebaseFlowUI) runWorker(t *testing.T) error {
	t.Helper()
	if len(ui.workers) == 0 {
		t.Fatal("expected queued worker")
	}
	f := ui.workers[0]
	ui.workers = ui.workers[1:]
	return f(nil)
}

func (ui *selectedRebaseFlowUI) runUI(t *testing.T) error {
	t.Helper()
	if len(ui.ui) == 0 {
		t.Fatal("expected queued UI callback")
	}
	f := ui.ui[0]
	ui.ui = ui.ui[1:]
	return f()
}

type selectedRebaseFlowHelpers struct{ value *helpers.Helpers }

func (h selectedRebaseFlowHelpers) Helpers() *helpers.Helpers { return h.value }

type selectedRebaseFlowState struct {
	types.IStateAccessor
	repo *selectedRebaseFlowRepoState
}

func (s *selectedRebaseFlowState) GetRepoState() types.IRepoStateAccessor { return s.repo }

type selectedRebaseFlowRepoState struct{ types.IRepoStateAccessor }

func (*selectedRebaseFlowRepoState) SetMergeOrRebaseStartedInLazygit(bool) {}

func selectedRebaseFlowGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func selectedRebaseFlowFixture(t *testing.T) (*BasicCommitsController, *selectedRebaseFlowUI, string, []string) {
	t.Helper()
	root := t.TempDir()
	selectedRebaseFlowGit(t, root, "init", "-q")
	selectedRebaseFlowGit(t, root, "config", "user.name", "Test")
	selectedRebaseFlowGit(t, root, "config", "user.email", "test@example.invalid")
	initialBranch := selectedRebaseFlowGit(t, root, "symbolic-ref", "--short", "HEAD")
	writeCommit := func(name, path, data string) string {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		selectedRebaseFlowGit(t, root, "add", "-A")
		selectedRebaseFlowGit(t, root, "commit", "-qm", name)
		return selectedRebaseFlowGit(t, root, "rev-parse", "HEAD")
	}
	base := writeCommit("base", "base", "base\n")
	selectedRebaseFlowGit(t, root, "checkout", "-qb", "topic")
	source := writeCommit("source", "source", "source\n")
	selectedRebaseFlowGit(t, root, "checkout", "-q", initialBranch)
	writeCommit("target", "target", "target\n")
	if err := os.WriteFile(filepath.Join(root, "historical-second"), []byte("also target\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	selectedRebaseFlowGit(t, root, "add", "historical-second")
	selectedRebaseFlowGit(t, root, "commit", "-qm", "second target path")
	target := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD")
	selectedRebaseFlowGit(t, root, "checkout", "-q", "topic")
	paths := git_commands.MockRepoPaths(root)
	osCmd := oscommands.NewDummyOSCommand()
	cmn := common.NewDummyCommon()
	git := commands.NewGitCommandAux(cmn, nil, osCmd, git_config.NewFakeGitConfig(nil), paths, nil)
	ui := &selectedRebaseFlowUI{git: git, modes: &types.Modes{}, state: &selectedRebaseFlowState{repo: &selectedRebaseFlowRepoState{}}}
	controller := historicalTestBasicController([]*models.Commit{historicalTestCommit(target)}, 0, 0)
	controller.c.IGuiCommon = ui
	hc := &helpers.HelperCommon{Common: cmn, IGuiCommon: ui}
	controller.c.IGetHelpers = selectedRebaseFlowHelpers{value: &helpers.Helpers{Suggestions: helpers.NewSuggestionsHelper(controller.c.HelperCommon), MergeAndRebase: helpers.NewMergeAndRebaseHelper(hc)}}
	return controller, ui, root, []string{base, source, target}
}

func selectedRebaseFlowBegin(t *testing.T, c *BasicCommitsController, ui *selectedRebaseFlowUI) {
	t.Helper()
	if err := c.beginSelectedCommitRebase(); err != nil {
		t.Fatal(err)
	}
	if len(ui.menus) != 1 {
		t.Fatalf("menus = %d, want topology menu", len(ui.menus))
	}
}

func selectedRebaseFlowRiskMenu(t *testing.T, includeSecond bool) (*BasicCommitsController, *selectedRebaseFlowUI, string) {
	t.Helper()
	c, ui, root, _ := selectedRebaseFlowFixture(t)
	ignore := "target\nhistorical*\n"
	if includeSecond {
		ignore += "unrelated*\n"
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(ignore), 0o600); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"target": "safe target\n"}
	if includeSecond {
		files["historical-second"] = "safe second\n"
	}
	files["unrelated-ignored"] = "unrelated\n"
	for path, contents := range files {
		if err := os.WriteFile(filepath.Join(root, path), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	selectedRebaseFlowBegin(t, c, ui)
	if err := ui.menus[0].Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.confirmations[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if len(ui.menus) < 2 {
		t.Fatal("expected ignored risk menu")
	}
	return c, ui, root
}

func TestSelectedRebaseGuardsAndTopologyMenu(t *testing.T) {
	valid := historicalTestCommit("0123456789abcdef0123456789abcdef01234567")
	for _, tc := range []struct {
		name       string
		items      []*models.Commit
		start, end int
		allowed    bool
	}{
		{name: "none"},
		{name: "range", items: []*models.Commit{valid, valid}, start: 0, end: 1},
		{name: "placeholder", items: []*models.Commit{historicalTestCommit("HEAD")}},
		{name: "valid", items: []*models.Commit{valid}, allowed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := historicalTestBasicController(tc.items, tc.start, tc.end)
			_, reason := c.selectedRebaseSource()
			if (reason == nil) != tc.allowed {
				t.Fatalf("reason=%v allowed=%v", reason, tc.allowed)
			}
			if tc.allowed {
				return
			}
			if err := c.beginSelectedCommitRebase(); err == nil {
				t.Fatal("guard accepted invalid selection")
			}
		})
	}
	c, ui, root, oids := selectedRebaseFlowFixture(t)
	ui.modes.MarkedBaseCommit.SetHash(oids[0])
	before := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD")
	selectedRebaseFlowBegin(t, c, ui)
	items := ui.menus[0].Items
	if len(items) != 2 || items[0].CommandID != "branch.rebase-current-onto-selected-commit.ordinary" || items[1].CommandID != "branch.rebase-current-onto-selected-commit.preserve-merges" {
		t.Fatalf("topology menu IDs: %+v", items)
	}
	if err := items[1].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") != before || selectedRebaseFlowGit(t, root, "rev-parse", "refs/heads/topic") != oids[1] {
		t.Fatal("topology selection mutated Git before history confirmation")
	}
	if len(ui.confirmations) != 1 || !strings.Contains(ui.confirmations[0].Prompt, "topic") || !strings.Contains(ui.confirmations[0].Prompt, oids[2]) || !strings.Contains(ui.confirmations[0].Prompt, "preserv") {
		t.Fatalf("confirmation omitted captured source/target/topology: %+v", ui.confirmations)
	}
	if !strings.Contains(ui.confirmations[0].Prompt, "marked base is not used") || ui.modes.MarkedBaseCommit.GetHash() != oids[0] {
		t.Fatal("selected rebase used or cleared the marked base")
	}
}

func TestSelectedRebaseRiskMenuCancelsFirstAndApprovesOnlySnapshot(t *testing.T) {
	_, ui, root := selectedRebaseFlowRiskMenu(t, true)
	risk := ui.menus[len(ui.menus)-1]
	if risk.Prompt == "" || !risk.HideCancel || len(risk.Items) != 4 || !strings.Contains(strings.ToLower(risk.Items[0].Label), "cancel") {
		t.Fatalf("unexpected risk menu: %+v", risk)
	}
	if risk.Items[2].Label != fmt.Sprintf("%q", "historical-second") || risk.Items[2].Tooltip != risk.Items[2].Label || risk.Items[3].Label != `"target"` {
		t.Fatalf("risk paths are not individually quoted in stable rows: %+v", risk.Items)
	}
	if err := risk.Items[2].OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(ui.menus) != 3 || len(ui.workers) != 0 {
		t.Fatal("pressing a path row must reopen review without approving")
	}
	// Reopened review remains scrollable and keeps every path available.
	if len(ui.menus[len(ui.menus)-1].Items) != 4 {
		t.Fatal("reopened menu dropped risk paths")
	}
	if len(ui.workers) != 0 || selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") == "" {
		t.Fatal("opening risk menu started work")
	}
	oldHead := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD")
	if err := risk.Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(ui.workers) != 0 || selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") != oldHead {
		t.Fatal("risk cancellation executed rebase")
	}
}

func TestSelectedRebaseIgnoredNewlinePathIsQuoted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Git paths containing line feeds are not portable to Windows")
	}
	c, ui, root, _ := selectedRebaseFlowFixture(t)
	branches := strings.Split(selectedRebaseFlowGit(t, root, "branch", "--format=%(refname:short)"), "\n")
	var baseBranch string
	for _, branch := range branches {
		if branch != "topic" {
			baseBranch = branch
			break
		}
	}
	if baseBranch == "" {
		t.Fatal("fixture has no base branch")
	}
	selectedRebaseFlowGit(t, root, "checkout", "-q", baseBranch)
	path := "historical\nsecond"
	if err := os.WriteFile(filepath.Join(root, path), []byte("target\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	selectedRebaseFlowGit(t, root, "add", path)
	selectedRebaseFlowGit(t, root, "commit", "-qm", "target newline path")
	target := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD")
	selectedRebaseFlowGit(t, root, "checkout", "-q", "topic")
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("historical*\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, path), []byte("ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c.context.(*historicalDiffCommitsStub).items = []*models.Commit{historicalTestCommit(target)}
	selectedRebaseFlowBegin(t, c, ui)
	if err := ui.menus[0].Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.confirmations[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if len(ui.menus) != 2 {
		t.Fatal("expected ignored path review")
	}
	row := ui.menus[1].Items[2]
	want := fmt.Sprintf("%q", path)
	if row.Label != want || row.Tooltip != want {
		t.Fatalf("newline path label/tooltip = %q/%q, want %q", row.Label, row.Tooltip, want)
	}
}

func TestSelectedRebaseCaptureAndIgnoreHistoryApproval(t *testing.T) {
	c, ui, root, oids := selectedRebaseFlowFixture(t)
	selectedRebaseFlowBegin(t, c, ui)
	first := ui.menus[0].Items[0]
	controllerSelection := c.context.(*historicalDiffCommitsStub)
	controllerSelection.items = []*models.Commit{historicalTestCommit(oids[1])}
	if err := first.OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.confirmations) != 1 || !strings.Contains(ui.confirmations[0].Prompt, oids[2]) {
		t.Fatal("changed selection retargeted captured commit")
	}
	if selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") != oids[1] {
		t.Fatal("review changed HEAD")
	}
	c, ui, _, _ = selectedRebaseFlowFixture(t)
	selectedRebaseFlowBegin(t, c, ui)
	first = ui.menus[0].Items[0]
	ui.git = &commands.GitCommand{}
	if err := first.OnPress(); err == nil {
		t.Fatal("topology callback accepted a different repository")
	}
	_ = oids
}

func TestSelectedRebaseApprovalRevalidatesIgnoredContentsAndNewPaths(t *testing.T) {
	for _, tc := range []struct {
		name    string
		addPath bool
	}{
		{name: "changed contents"}, {name: "new ignored collision", addPath: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ui, root := selectedRebaseFlowRiskMenu(t, false)
			risk := ui.menus[len(ui.menus)-1]
			beforeHead := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD")
			beforeIndex := selectedRebaseFlowGit(t, root, "write-tree")
			if err := risk.Items[1].OnPress(); err != nil {
				t.Fatal(err)
			}
			if len(ui.workers) != 1 {
				t.Fatal("explicit approval did not queue apply")
			}
			path := "target"
			contents := "changed after approval\n"
			if tc.addPath {
				path = "historical-second"
				contents = "new ignored collision\n"
			}
			if err := os.WriteFile(filepath.Join(root, path), []byte(contents), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if len(ui.confirmations) != 2 || ui.confirmations[1].Title == ui.confirmations[0].Title {
				t.Fatalf("changed snapshot did not require a new confirmation: %+v", ui.confirmations)
			}
			if err := ui.confirmations[1].HandleConfirm(); err != nil {
				t.Fatal(err)
			}
			containsNewPath := false
			if len(ui.menus) == 3 {
				for _, item := range ui.menus[2].Items {
					if strings.Contains(item.Label, fmt.Sprintf("%q", "historical-second")) {
						containsNewPath = true
					}
				}
			}
			if len(ui.menus) != 3 || (tc.addPath && !containsNewPath) {
				t.Fatal("fresh review omitted the newly detected risk path")
			}
			if selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") != beforeHead || selectedRebaseFlowGit(t, root, "write-tree") != beforeIndex {
				t.Fatal("stale approval mutated Git")
			}
			if len(ui.workers) != 0 {
				t.Fatal("stale approval retried apply without fresh confirmation")
			}
		})
	}
}

func TestSelectedRebaseRechecksChangedHeadAndDirtyTrackedState(t *testing.T) {
	t.Run("same target new source tip", func(t *testing.T) {
		c, ui, root, oids := selectedRebaseFlowFixture(t)
		selectedRebaseFlowBegin(t, c, ui)
		if err := ui.menus[0].Items[0].OnPress(); err != nil {
			t.Fatal(err)
		}
		if err := ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.confirmations[0].HandleConfirm(); err != nil {
			t.Fatal(err)
		}
		if len(ui.workers) != 1 {
			t.Fatal("confirmation did not queue apply")
		}
		if err := os.WriteFile(filepath.Join(root, "later"), []byte("later\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		selectedRebaseFlowGit(t, root, "add", "later")
		selectedRebaseFlowGit(t, root, "commit", "-qm", "new source tip")
		newHead := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD")
		if err := ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if len(ui.confirmations) != 2 || ui.confirmations[1].Title == ui.confirmations[0].Title || !strings.Contains(ui.confirmations[1].Prompt, newHead[:8]) || !strings.Contains(ui.confirmations[1].Prompt, oids[2]) {
			t.Fatalf("new source tip was not freshly reviewed: %+v", ui.confirmations)
		}
		if selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") != newHead {
			t.Fatal("stale confirmation moved HEAD")
		}
	})
	t.Run("tracked dirt refuses without stash", func(t *testing.T) {
		c, ui, root, _ := selectedRebaseFlowFixture(t)
		selectedRebaseFlowBegin(t, c, ui)
		if err := ui.menus[0].Items[0].OnPress(); err != nil {
			t.Fatal(err)
		}
		if err := ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := ui.runUI(t); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "source"), []byte("dirty\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		beforeHead := selectedRebaseFlowGit(t, root, "rev-parse", "HEAD")
		if err := ui.confirmations[0].HandleConfirm(); err != nil {
			t.Fatal(err)
		}
		if err := ui.runWorker(t); err == nil {
			t.Fatal("tracked dirt was accepted")
		}
		if selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") != beforeHead {
			t.Fatal("refused rebase moved HEAD")
		}
		if got := selectedRebaseFlowGit(t, root, "stash", "list"); got != "" {
			t.Fatalf("tracked dirt was autostashed: %s", got)
		}
		if len(ui.refreshes) == 0 {
			t.Fatal("failed apply did not refresh")
		}
	})
}

func TestSelectedRebaseSigningQueuesTerminalOnUIThread(t *testing.T) {
	c, ui, root, _ := selectedRebaseFlowFixture(t)
	ui.git.Config = git_commands.NewConfigCommands(common.NewDummyCommon(), git_config.NewFakeGitConfig(map[string]string{"commit.gpgSign": "true"}))
	selectedRebaseFlowBegin(t, c, ui)
	if err := ui.menus[0].Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.confirmations[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if len(ui.workers) != 1 {
		t.Fatal("confirmation did not queue apply")
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if len(ui.ui) != 1 || ui.runCalls != 0 {
		t.Fatalf("signing path must queue terminal UI work; pending UI=%d subprocess calls=%d", len(ui.ui), ui.runCalls)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if ui.runCalls != 1 || len(ui.workers) != 1 {
		t.Fatal("terminal callback did not use the fake subprocess and queue result handling")
	}
	if err := ui.runWorker(t); err == nil || !strings.Contains(err.Error(), "unexpected subprocess") {
		t.Fatalf("subprocess failure was not preserved: %v", err)
	}
	if len(ui.refreshes) == 0 {
		t.Fatal("terminal result handling did not preserve refresh")
	}
	if selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") == "" {
		t.Fatal("unable to inspect source head")
	}
}
