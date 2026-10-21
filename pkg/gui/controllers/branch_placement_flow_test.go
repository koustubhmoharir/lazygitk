package controllers

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type placementFlowUI struct {
	types.IGuiCommon
	git           *commands.GitCommand
	model         types.Model
	prompts       []types.PromptOpts
	confirmations []types.ConfirmOpts
	alerts        []string
	workers       []func(gocui.Task) error
	ui            []func() error
	refreshes     []types.RefreshOptions
}

func (ui *placementFlowUI) Git() *commands.GitCommand    { return ui.git }
func (ui *placementFlowUI) Model() *types.Model          { return &ui.model }
func (ui *placementFlowUI) Prompt(opts types.PromptOpts) { ui.prompts = append(ui.prompts, opts) }

func (ui *placementFlowUI) Confirm(opts types.ConfirmOpts) {
	ui.confirmations = append(ui.confirmations, opts)
}

func (ui *placementFlowUI) Alert(title, prompt string) {
	ui.alerts = append(ui.alerts, title+"\n"+prompt)
}
func (ui *placementFlowUI) OnUIThread(f func() error) { ui.ui = append(ui.ui, f) }
func (ui *placementFlowUI) LogAction(string)          {}
func (ui *placementFlowUI) Refresh(opts types.RefreshOptions) {
	ui.refreshes = append(ui.refreshes, opts)
}

func (ui *placementFlowUI) RefreshFromWorker(opts types.RefreshOptions) {
	ui.refreshes = append(ui.refreshes, opts)
}

func (ui *placementFlowUI) WithWaitingStatus(_ string, f func(gocui.Task) error) error {
	ui.workers = append(ui.workers, f)
	return nil
}

func (ui *placementFlowUI) runWorker(t *testing.T) {
	t.Helper()
	if len(ui.workers) == 0 {
		t.Fatal("expected a foreground worker")
	}
	f := ui.workers[0]
	ui.workers = ui.workers[1:]
	if err := f(nil); err != nil {
		t.Fatal(err)
	}
}

func (ui *placementFlowUI) runUI(t *testing.T) {
	t.Helper()
	if len(ui.ui) == 0 {
		t.Fatal("expected a UI callback")
	}
	f := ui.ui[0]
	ui.ui = ui.ui[1:]
	if err := f(); err != nil {
		t.Fatal(err)
	}
}

type placementFlowHelpers struct{ value *helpers.Helpers }

func (h placementFlowHelpers) Helpers() *helpers.Helpers { return h.value }

func placementFlowGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func placementFlowWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func placementFlowFixture(t *testing.T) (*BasicCommitsController, *placementFlowUI, string, []string) {
	t.Helper()
	root := t.TempDir()
	placementFlowGit(t, root, "init", "-q")
	placementFlowGit(t, root, "config", "user.name", "Test")
	placementFlowGit(t, root, "config", "user.email", "test@example.invalid")
	commits := make([]string, 0, 3)
	for _, content := range []string{"one", "two", "three"} {
		placementFlowWrite(t, filepath.Join(root, "tracked"), content)
		placementFlowGit(t, root, "add", "tracked")
		placementFlowGit(t, root, "commit", "-qm", content)
		commits = append(commits, placementFlowGit(t, root, "rev-parse", "HEAD"))
	}
	placementFlowGit(t, root, "branch", "topic", commits[1])
	placementFlowWrite(t, filepath.Join(root, "tracked"), "staged")
	placementFlowGit(t, root, "add", "tracked")
	placementFlowWrite(t, filepath.Join(root, "tracked"), "working")
	placementFlowWrite(t, filepath.Join(root, "loose"), "untracked")
	osCommand := oscommands.NewDummyOSCommand()
	paths := git_commands.MockRepoPaths(root)
	backend := git_commands.NewBranchCommands(git_commands.NewGitCommon(common.NewDummyCommon(), nil, osCommand.Cmd, osCommand, paths, nil, nil))
	ui := &placementFlowUI{git: &commands.GitCommand{Branch: backend, RepoPaths: paths}}
	controller := historicalTestBasicController([]*models.Commit{historicalTestCommit(commits[0])}, 0, 0)
	controller.c.IGuiCommon = ui
	controller.c.IGetHelpers = placementFlowHelpers{value: &helpers.Helpers{Suggestions: helpers.NewSuggestionsHelper(controller.c.HelperCommon)}}
	return controller, ui, root, commits
}

func placementFlowAssertState(t *testing.T, root, head, index string) {
	t.Helper()
	if got := placementFlowGit(t, root, "rev-parse", "HEAD"); got != head {
		t.Fatalf("HEAD changed: %s", got)
	}
	if got := placementFlowGit(t, root, "write-tree"); got != index {
		t.Fatalf("index changed: %s", got)
	}
	for file, want := range map[string]string{"tracked": "working", "loose": "untracked"} {
		data, err := os.ReadFile(filepath.Join(root, file))
		if err != nil || string(data) != want {
			t.Fatalf("working file %s changed: %q, %v", file, data, err)
		}
	}
}

func placementFlowStart(t *testing.T, controller *BasicCommitsController, ui *placementFlowUI, name string) {
	t.Helper()
	if err := controller.beginBranchPlacement(); err != nil {
		t.Fatal(err)
	}
	if err := ui.prompts[len(ui.prompts)-1].HandleConfirm(name); err != nil {
		t.Fatal(err)
	}
	ui.runWorker(t)
	ui.runUI(t)
}

func TestBranchPlacementFlowCaptureCreationAndCancel(t *testing.T) {
	controller, ui, root, commits := placementFlowFixture(t)
	index := placementFlowGit(t, root, "write-tree")
	headRef := placementFlowGit(t, root, "symbolic-ref", "HEAD")
	if err := controller.beginBranchPlacement(); err != nil {
		t.Fatal(err)
	}
	if len(ui.workers) != 0 || len(ui.confirmations) != 0 {
		t.Fatal("opening/canceling a name prompt must not schedule work")
	}
	// Model changes while the prompt is open must not retarget its source.
	controller.context.(*historicalDiffCommitsStub).items = []*models.Commit{historicalTestCommit(commits[2])}
	if err := ui.prompts[0].HandleConfirm("created"); err != nil {
		t.Fatal(err)
	}
	ui.runWorker(t)
	ui.runUI(t)
	if len(ui.confirmations) != 0 || len(ui.workers) != 1 {
		t.Fatal("fresh creation should follow the accepted name prompt without a move warning")
	}
	ui.runWorker(t)
	if got := placementFlowGit(t, root, "rev-parse", "refs/heads/created"); got != commits[0] {
		t.Fatalf("prompt source was retargeted: %s", got)
	}
	placementFlowAssertState(t, root, commits[2], index)
	if got := placementFlowGit(t, root, "symbolic-ref", "HEAD"); got != headRef {
		t.Fatal("symbolic HEAD changed")
	}
	want := branchPlacementRefreshOptions()
	if len(ui.refreshes) != 1 || !reflect.DeepEqual(ui.refreshes[0].Scope, want.Scope) || ui.refreshes[0].CommitSelection != types.KeepCommitSelectionByHash || ui.refreshes[0].BranchSelection != types.KeepBranchSelectionByName {
		t.Fatalf("unexpected selection-changing refresh: %+v", ui.refreshes)
	}
}

func TestBranchPlacementFlowWarningAndFreshConfirmation(t *testing.T) {
	for _, deleted := range []bool{false, true} {
		t.Run(map[bool]string{false: "moved", true: "deleted"}[deleted], func(t *testing.T) {
			controller, ui, root, commits := placementFlowFixture(t)
			index := placementFlowGit(t, root, "write-tree")
			placementFlowStart(t, controller, ui, "topic")
			if len(ui.workers) != 0 || len(ui.confirmations) != 1 {
				t.Fatal("a move must wait for explicit confirmation")
			}
			warning := ui.confirmations[0]
			if !strings.Contains(warning.Prompt, commits[1][:7]) || !strings.Contains(warning.Prompt, commits[0][:7]) || !strings.Contains(warning.Prompt, "topic") {
				t.Fatalf("move warning omitted its destination/tips: %q", warning.Prompt)
			}
			// Dropping this callback represents cancellation: there is no write.
			if got := placementFlowGit(t, root, "rev-parse", "refs/heads/topic"); got != commits[1] {
				t.Fatal("showing the warning moved the branch")
			}
			if err := warning.HandleConfirm(); err != nil {
				t.Fatal(err)
			}
			if deleted {
				placementFlowGit(t, root, "update-ref", "-d", "refs/heads/topic", commits[1])
			} else {
				placementFlowGit(t, root, "update-ref", "refs/heads/topic", commits[2], commits[1])
			}
			ui.runWorker(t) // detects stale state, not a silent destructive retry
			ui.runUI(t)
			ui.runWorker(t)
			ui.runUI(t)
			if len(ui.confirmations) != 2 || len(ui.workers) != 0 {
				t.Fatal("changed state must require a new confirmation")
			}
			fresh := ui.confirmations[1]
			if fresh.Title != controller.c.Tr.PlaceBranchChangedTitle {
				t.Fatalf("missing changed-state warning: %q", fresh.Title)
			}
			if deleted {
				if !strings.Contains(fresh.Prompt, "no longer exists") {
					t.Fatalf("recreation was not an explicit choice: %q", fresh.Prompt)
				}
				cmd := exec.Command("git", "-C", root, "show-ref", "--verify", "--quiet", "refs/heads/topic")
				if cmd.Run() == nil {
					t.Fatal("deleted branch was silently recreated")
				}
			} else if !strings.Contains(fresh.Prompt, commits[2][:7]) || placementFlowGit(t, root, "rev-parse", "refs/heads/topic") != commits[2] {
				t.Fatal("the fresh warning did not preserve/show the external tip")
			}
			if err := fresh.HandleConfirm(); err != nil {
				t.Fatal(err)
			}
			ui.runWorker(t)
			if got := placementFlowGit(t, root, "rev-parse", "refs/heads/topic"); got != commits[0] {
				t.Fatalf("explicitly confirmed placement failed: %s", got)
			}
			placementFlowAssertState(t, root, commits[2], index)
		})
	}
}

func TestBranchPlacementFlowNoopAndRepositoryGuard(t *testing.T) {
	controller, ui, root, commits := placementFlowFixture(t)
	placementFlowGit(t, root, "update-ref", "refs/heads/topic", commits[0], commits[1])
	before := placementFlowGit(t, root, "reflog", "-1", "--format=%H %gs", "topic")
	placementFlowStart(t, controller, ui, "topic")
	if len(ui.alerts) != 1 || len(ui.workers) != 0 || len(ui.confirmations) != 0 || len(ui.refreshes) != 1 {
		t.Fatal("same-tip placement should inform/refresh without writing")
	}
	if got := placementFlowGit(t, root, "reflog", "-1", "--format=%H %gs", "topic"); got != before {
		t.Fatal("no-op appended a reflog entry")
	}
	if err := controller.beginBranchPlacement(); err != nil {
		t.Fatal(err)
	}
	ui.git = &commands.GitCommand{}
	if err := ui.prompts[len(ui.prompts)-1].HandleConfirm("another"); err == nil || len(ui.workers) != 0 {
		t.Fatal("a name prompt must not redirect to a different repository")
	}
}

func TestBranchPlacementFlowWarningCannotRedirectRepository(t *testing.T) {
	controller, ui, root, commits := placementFlowFixture(t)
	placementFlowStart(t, controller, ui, "topic")
	if len(ui.confirmations) != 1 || len(ui.workers) != 0 {
		t.Fatal("expected a pending move warning")
	}
	ui.git = &commands.GitCommand{}
	if err := ui.confirmations[0].HandleConfirm(); err == nil || len(ui.workers) != 0 {
		t.Fatal("a move warning must not redirect to another repository")
	}
	if got := placementFlowGit(t, root, "rev-parse", "refs/heads/topic"); got != commits[1] {
		t.Fatal("rejected cross-repository confirmation changed the original branch")
	}
}
