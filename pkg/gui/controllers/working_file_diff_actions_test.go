package controllers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_config"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	guicontext "github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/modes/diffing"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

// workingFileDiffUI records the difftool commands the Files actions build and
// serves the model, sticky diff mode and repository generation they read.
type workingFileDiffUI struct {
	types.IGuiCommon
	git      *commands.GitCommand
	state    *historicalComparisonState
	model    *types.Model
	modes    *types.Modes
	prompts  []types.PromptOpts
	commands []*oscommands.CmdObj
}

func (ui *workingFileDiffUI) Git() *commands.GitCommand   { return ui.git }
func (ui *workingFileDiffUI) State() types.IStateAccessor { return ui.state }
func (ui *workingFileDiffUI) Model() *types.Model         { return ui.model }
func (ui *workingFileDiffUI) Modes() *types.Modes         { return ui.modes }
func (ui *workingFileDiffUI) Views() types.Views          { return types.Views{} }
func (ui *workingFileDiffUI) Prompt(opts types.PromptOpts) {
	ui.prompts = append(ui.prompts, opts)
}

func (ui *workingFileDiffUI) RunSubprocess(cmd *oscommands.CmdObj) (bool, error) {
	ui.commands = append(ui.commands, cmd)
	return false, nil
}

func (ui *workingFileDiffUI) RunSubprocessAndRefresh(cmd *oscommands.CmdObj) error {
	ui.commands = append(ui.commands, cmd)
	return nil
}
func (ui *workingFileDiffUI) Refresh(types.RefreshOptions) {}

type workingFileDiffFixture struct {
	controller *FilesController
	ui         *workingFileDiffUI
	ctx        *guicontext.WorkingTreeContext
	before     string
	head       string
}

// newWorkingFileDiffFixture builds a Files controller over a real repository
// with two commits of sample.txt, so HEAD and an older commit can be resolved.
func newWorkingFileDiffFixture(t *testing.T, files []*models.File, branches []*models.Branch) *workingFileDiffFixture {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q")
	run("config", "user.name", "Test")
	run("config", "user.email", "test@example.invalid")
	for _, content := range []string{"before\n", "head\n"} {
		if err := os.WriteFile(filepath.Join(root, "sample.txt"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		run("add", "sample.txt")
		run("commit", "-qm", content)
	}
	before := run("rev-parse", "HEAD~1")
	head := run("rev-parse", "HEAD")

	gitCmd := commands.NewGitCommandAux(common.NewDummyCommon(), nil, oscommands.NewDummyOSCommand(), git_config.NewFakeGitConfig(nil), git_commands.MockRepoPaths(root), nil)
	ui := &workingFileDiffUI{
		git:   gitCmd,
		state: &historicalComparisonState{},
		model: &types.Model{Files: files, Branches: branches},
		modes: &types.Modes{Diffing: diffing.New()},
	}
	cmn := common.NewDummyCommon()
	contextCommon := &guicontext.ContextCommon{Common: cmn, IGuiCommon: ui}
	ctx := guicontext.NewWorkingTreeContext(contextCommon)
	ctx.SetTree()
	helperCommon := &helpers.HelperCommon{
		Common:     cmn,
		IGuiCommon: ui,
		IGetContexts: historicalDiffContextsStub{tree: &guicontext.ContextTree{
			Files: ctx,
		}},
	}
	return &workingFileDiffFixture{
		controller: NewFilesController(NewControllerCommon(helperCommon, nil)),
		ui:         ui,
		ctx:        ctx,
		before:     before,
		head:       head,
	}
}

// node returns the Files tree node at path, failing the test if it is absent.
func (f *workingFileDiffFixture) node(t *testing.T, path string) *filetree.FileNode {
	t.Helper()
	for i := range f.ctx.Len() {
		f.ctx.SetSelectedLineIdx(i)
		if node := f.ctx.GetSelected(); node != nil && node.GetPath() == path {
			return node
		}
	}
	t.Fatalf("no Files node at %q", path)
	return nil
}

func (f *workingFileDiffFixture) onlyCommand(t *testing.T) []string {
	t.Helper()
	if len(f.ui.commands) != 1 {
		t.Fatalf("subprocesses = %d, want 1", len(f.ui.commands))
	}
	return f.ui.commands[0].Args()
}

func workingFileTestFiles() []*models.File {
	return []*models.File{
		{Path: "sample.txt", Tracked: true, HasUnstagedChanges: true},
		{Path: "dir/a.txt", Tracked: true, HasUnstagedChanges: true},
	}
}

func workingFileTestBranches(head string) []*models.Branch {
	return []*models.Branch{{Name: "master", Head: true, CommitHash: head}}
}

func TestWorkingFileDiffDisabledReasons(t *testing.T) {
	files := []*models.File{
		{Path: "tracked.txt", Tracked: true, HasUnstagedChanges: true},
		{Path: "staged.txt", Tracked: true, HasStagedChanges: true},
		{Path: "new.txt", Tracked: false, HasUnstagedChanges: true},
		{Path: "dir/a.txt", Tracked: true, HasStagedChanges: true},
	}
	const untracked = "untracked file: Git has no base version to compare"
	const noUnstaged = "no unstaged changes"
	const unborn = "HEAD has no commit yet"
	tests := []struct {
		name     string
		branches []*models.Branch
		path     string
		head     string // expected disabled text, or "" when allowed
		staged   string
		commit   string
	}{
		{name: "tracked with unstaged changes", path: "tracked.txt"},
		{name: "staged only file", path: "staged.txt", staged: noUnstaged},
		{name: "untracked file", path: "new.txt", head: untracked, staged: untracked, commit: untracked},
		{name: "directory allowed", path: "dir"},
		{name: "unborn HEAD", branches: []*models.Branch{{Name: "master", Head: true}}, path: "tracked.txt", head: unborn},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			branches := tc.branches
			if branches == nil {
				branches = workingFileTestBranches("0123456789abcdef0123456789abcdef01234567")
			}
			fixture := newWorkingFileDiffFixture(t, files, branches)
			node := fixture.node(t, tc.path)
			check := func(name string, reason *types.DisabledReason, want string) {
				t.Helper()
				got := ""
				if reason != nil {
					got = reason.Text
				}
				if got != want {
					t.Errorf("%s disabled reason = %q, want %q", name, got, want)
				}
			}
			check("head", fixture.controller.workingFileHeadDisabledReason(node), tc.head)
			check("staged", fixture.controller.workingFileStagedDisabledReason(node), tc.staged)
			check("commit", fixture.controller.workingFileCommitDisabledReason(node), tc.commit)
		})
	}
}

func TestWorkingFileDiffFileEndpoints(t *testing.T) {
	fixture := newWorkingFileDiffFixture(t, workingFileTestFiles(), workingFileTestBranches("0123456789abcdef0123456789abcdef01234567"))
	fixture.ui.model.Branches = workingFileTestBranches(fixture.head)
	node := fixture.node(t, "sample.txt")

	if err := fixture.controller.diffWorkingFileWithHead(node); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", fixture.head, "--", "sample.txt"}, fixture.onlyCommand(t))

	fixture.ui.commands = nil
	if err := fixture.controller.diffWorkingFileWithStaged(node); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"git", "difftool", "--no-prompt", "--", "sample.txt"}, fixture.onlyCommand(t))

	fixture.ui.commands = nil
	if err := fixture.controller.promptWorkingFileCommitDiff(node); err != nil {
		t.Fatal(err)
	}
	if len(fixture.ui.prompts) != 1 {
		t.Fatalf("prompts = %d, want 1", len(fixture.ui.prompts))
	}
	if err := fixture.ui.prompts[0].HandleConfirm(fixture.before[:7]); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", fixture.before, "--", "sample.txt"}, fixture.onlyCommand(t))
}

func TestWorkingFileDiffDirectoryEndpoints(t *testing.T) {
	fixture := newWorkingFileDiffFixture(t, workingFileTestFiles(), workingFileTestBranches(""))
	fixture.ui.model.Branches = workingFileTestBranches(fixture.head)
	node := fixture.node(t, "dir")

	if err := fixture.controller.diffWorkingFileWithHead(node); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", "--dir-diff", fixture.head, "--", "dir"}, fixture.onlyCommand(t))

	fixture.ui.commands = nil
	if err := fixture.controller.diffWorkingFileWithStaged(node); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"git", "difftool", "--no-prompt", "--dir-diff", "--", "dir"}, fixture.onlyCommand(t))

	fixture.ui.commands = nil
	if err := fixture.controller.promptWorkingFileCommitDiff(node); err != nil {
		t.Fatal(err)
	}
	if err := fixture.ui.prompts[0].HandleConfirm(fixture.before); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", "--dir-diff", fixture.before, "--", "dir"}, fixture.onlyCommand(t))
}

func TestWorkingFileDiffStagedRenameIncludesOldPath(t *testing.T) {
	files := []*models.File{{Path: "new.txt", PreviousPath: "sample.txt", Tracked: true, HasStagedChanges: true, HasUnstagedChanges: true}}
	fixture := newWorkingFileDiffFixture(t, files, workingFileTestBranches(""))
	fixture.ui.model.Branches = workingFileTestBranches(fixture.head)
	node := fixture.node(t, "new.txt")

	if err := fixture.controller.diffWorkingFileWithHead(node); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", fixture.head, "--", "new.txt", "sample.txt"}, fixture.onlyCommand(t))

	fixture.ui.commands = nil
	if err := fixture.controller.promptWorkingFileCommitDiff(node); err != nil {
		t.Fatal(err)
	}
	if err := fixture.ui.prompts[0].HandleConfirm(fixture.before); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", fixture.before, "--", "new.txt", "sample.txt"}, fixture.onlyCommand(t))
}

func TestWorkingFileDiffStickyModeIsIgnored(t *testing.T) {
	fixture := newWorkingFileDiffFixture(t, workingFileTestFiles(), workingFileTestBranches(""))
	fixture.ui.model.Branches = workingFileTestBranches(fixture.head)
	fixture.ui.modes.Diffing.Ref = fixture.before
	fixture.ui.modes.Diffing.Reverse = true
	node := fixture.node(t, "sample.txt")

	if err := fixture.controller.diffWorkingFileWithStaged(node); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"git", "difftool", "--no-prompt", "--", "sample.txt"}, fixture.onlyCommand(t))

	fixture.ui.commands = nil
	if err := fixture.controller.diffWorkingFileWithHead(node); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", fixture.head, "--", "sample.txt"}, fixture.onlyCommand(t))
}

func TestWorkingFileDiffUntrackedFileRefused(t *testing.T) {
	files := []*models.File{{Path: "new.txt", Tracked: false, HasUnstagedChanges: true}}
	fixture := newWorkingFileDiffFixture(t, files, workingFileTestBranches(""))
	fixture.ui.model.Branches = workingFileTestBranches(fixture.head)
	node := fixture.node(t, "new.txt")

	for name, handler := range map[string]func(*filetree.FileNode) error{
		"head":   fixture.controller.diffWorkingFileWithHead,
		"staged": fixture.controller.diffWorkingFileWithStaged,
		"commit": fixture.controller.promptWorkingFileCommitDiff,
	} {
		if err := handler(node); err == nil {
			t.Errorf("%s: expected untracked refusal", name)
		}
	}
	if len(fixture.ui.commands) != 0 || len(fixture.ui.prompts) != 0 {
		t.Fatalf("untracked file ran %d subprocesses and opened %d prompts", len(fixture.ui.commands), len(fixture.ui.prompts))
	}
}

func TestWorkingFileDiffCommitHashValidation(t *testing.T) {
	fixture := newWorkingFileDiffFixture(t, workingFileTestFiles(), workingFileTestBranches(""))
	fixture.ui.model.Branches = workingFileTestBranches(fixture.head)
	node := fixture.node(t, "sample.txt")
	if err := fixture.controller.promptWorkingFileCommitDiff(node); err != nil {
		t.Fatal(err)
	}
	prompt := fixture.ui.prompts[0]
	if prompt.Title != fixture.controller.c.Tr.WorkingFileCommitPrompt {
		t.Fatalf("prompt title = %q", prompt.Title)
	}
	for _, input := range []string{"", "HEAD", "zz12", "abc", strings.Repeat("a", 65)} {
		if err := prompt.HandleConfirm(input); err == nil {
			t.Errorf("input %q accepted", input)
		}
	}
	if len(fixture.ui.commands) != 0 {
		t.Fatalf("rejected hashes ran %d subprocesses", len(fixture.ui.commands))
	}
}

func TestWorkingFileDiffCommitPromptRefusesRepositoryChange(t *testing.T) {
	fixture := newWorkingFileDiffFixture(t, workingFileTestFiles(), workingFileTestBranches(""))
	fixture.ui.model.Branches = workingFileTestBranches(fixture.head)
	if err := fixture.controller.promptWorkingFileCommitDiff(fixture.node(t, "sample.txt")); err != nil {
		t.Fatal(err)
	}
	fixture.ui.state.generation++
	if err := fixture.ui.prompts[0].HandleConfirm(fixture.before); err == nil {
		t.Fatal("expected changed-repository refusal")
	}
	if len(fixture.ui.commands) != 0 {
		t.Fatal("changed repository ran a subprocess")
	}
}

func TestWorkingFileDiffBindingMetadata(t *testing.T) {
	fixture := newWorkingFileDiffFixture(t, nil, nil)
	bindings := fixture.controller.workingFileDiffBindings()
	want := map[string]string{
		"diff.working-file.head":            "diff-head",
		"diff.working-file.staged":          "diff-staged",
		"diff.working-file.selected-commit": "diff-commit",
	}
	seen := map[string]bool{}
	for _, binding := range bindings {
		if seen[binding.CommandID] {
			t.Fatalf("duplicate command ID %q", binding.CommandID)
		}
		seen[binding.CommandID] = true
		if binding.Verb != want[binding.CommandID] {
			t.Errorf("%s verb = %q, want %q", binding.CommandID, binding.Verb, want[binding.CommandID])
		}
		if len(binding.Keys) != 0 {
			t.Errorf("%s keys = %v, want none (path routes only)", binding.CommandID, binding.Keys)
		}
		if binding.OpensMenu != (binding.CommandID == "diff.working-file.selected-commit") {
			t.Errorf("%s OpensMenu = %v", binding.CommandID, binding.OpensMenu)
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("got IDs %v, want %v", seen, want)
	}
}
