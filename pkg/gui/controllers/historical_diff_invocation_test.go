package controllers

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type historicalInvocationGui struct {
	types.IGuiCommon
	git         *commands.GitCommand
	args        [][]string
	workingDirs []string
	runResult   bool
	runErr      error
	refreshes   int
}

func (gui *historicalInvocationGui) Git() *commands.GitCommand { return gui.git }

func (gui *historicalInvocationGui) RunSubprocess(cmd *oscommands.CmdObj) (bool, error) {
	gui.args = append(gui.args, append([]string(nil), cmd.Args()...))
	gui.workingDirs = append(gui.workingDirs, cmd.GetCmd().Dir)
	return gui.runResult, gui.runErr
}

func (gui *historicalInvocationGui) Refresh(types.RefreshOptions) { gui.refreshes++ }

func historicalInvocationGit(t *testing.T, root string) (*commands.GitCommand, *oscommands.FakeCmdObjRunner) {
	t.Helper()
	runner := oscommands.NewFakeRunner(t)
	paths := git_commands.MockRepoPaths(root)
	diff := git_commands.NewDiffCommands(git_commands.NewGitCommon(
		common.NewDummyCommon(), nil,
		oscommands.NewDummyCmdObjBuilder(runner), oscommands.NewDummyOSCommand(),
		paths, nil, nil,
	))
	return &commands.GitCommand{Diff: diff, RepoPaths: paths}, runner
}

func TestHistoricalFileDiffInvocationArguments(t *testing.T) {
	const oid = "0123456789abcdef0123456789abcdef01234567"
	const parent = "1123456789abcdef0123456789abcdef01234567"
	tests := []struct {
		name     string
		files    []*models.CommitFile
		showTree bool
		handler  string
		want     []string
	}{
		{
			name: "directory working tree", files: []*models.CommitFile{{Path: "dir/a.txt"}},
			showTree: true, handler: "worktree",
			want: []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", "--dir-diff", oid, "--", "dir"},
		},
		{
			name: "directory previous", files: []*models.CommitFile{{Path: "dir/a.txt"}},
			showTree: true, handler: "previous",
			want: []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", "--dir-diff", parent, oid, "--", "dir"},
		},
		{
			name: "renamed file previous", files: []*models.CommitFile{{Path: "new/name.txt", PreviousPath: "old/name.txt"}},
			handler: "previous",
			want:    []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", parent, oid, "--", "new/name.txt", "old/name.txt"},
		},
		{
			name: "deleted file previous", files: []*models.CommitFile{{Path: "gone.txt", ChangeStatus: "D"}},
			handler: "previous",
			want:    []string{"git", "--literal-pathspecs", "difftool", "--no-prompt", parent, oid, "--", "gone.txt"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			controller, _ := historicalTestFileController(t, historicalTestCommit(oid, parent), tc.files, tc.showTree)
			root := t.TempDir()
			git, runner := historicalInvocationGit(t, root)
			gui := &historicalInvocationGui{git: git, runResult: true}
			controller.c.IGuiCommon = gui
			runner.ExpectGitArgs([]string{"rev-parse", "--verify", "--end-of-options", oid + "^{commit}"}, oid, nil)
			runner.ExpectGitArgs([]string{"rev-list", "--parents", "-n", "1", oid, "--"}, oid+" "+parent, nil)
			var err error
			if tc.handler == "previous" {
				err = controller.diffHistoricalFileWithPrevious()
			} else {
				err = controller.diffHistoricalFileWithWorktree()
			}
			if err != nil {
				t.Fatalf("invocation failed: %v", err)
			}
			if len(gui.args) != 1 || !reflect.DeepEqual(gui.args[0], tc.want) {
				t.Fatalf("subprocess args = %v, want %v", gui.args, tc.want)
			}
			if len(gui.workingDirs) != 1 || gui.workingDirs[0] != root {
				t.Fatalf("subprocess cwd = %v, want %q", gui.workingDirs, root)
			}
			if gui.refreshes != 1 {
				t.Fatalf("refreshes = %d, want 1", gui.refreshes)
			}
			runner.CheckForMissingCalls()
		})
	}
}

func TestHistoricalFileDiffInvocationGuardsAndFailureRefresh(t *testing.T) {
	const oid = "0123456789abcdef0123456789abcdef01234567"
	const parent = "1123456789abcdef0123456789abcdef01234567"
	for _, tc := range []struct {
		name    string
		ref     *models.Commit
		file    *models.CommitFile
		invoke  func(*CommitFilesController) error
		wantErr bool
	}{
		{name: "deleted worktree", ref: historicalTestCommit(oid, parent), file: &models.CommitFile{Path: "gone.txt", ChangeStatus: "D"}, invoke: (*CommitFilesController).diffHistoricalFileWithWorktree, wantErr: true},
		{name: "root previous", ref: historicalTestCommit(oid), file: &models.CommitFile{Path: "root.txt"}, invoke: (*CommitFilesController).diffHistoricalFileWithPrevious, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			controller, _ := historicalTestFileController(t, tc.ref, []*models.CommitFile{tc.file}, false)
			gui := &historicalInvocationGui{}
			controller.c.IGuiCommon = gui
			err := tc.invoke(controller)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, want error %v", err, tc.wantErr)
			}
			if len(gui.args) != 0 || gui.refreshes != 0 {
				t.Fatalf("guard performed I/O: subprocesses=%v refreshes=%d", gui.args, gui.refreshes)
			}
		})
	}

	root := t.TempDir()
	controller, _ := historicalTestFileController(t, historicalTestCommit(oid, parent), []*models.CommitFile{{Path: "dir/a.txt"}}, true)
	git, runner := historicalInvocationGit(t, root)
	sentinel := errors.New("terminal failed")
	gui := &historicalInvocationGui{git: git, runErr: sentinel}
	controller.c.IGuiCommon = gui
	runner.ExpectGitArgs([]string{"rev-parse", "--verify", "--end-of-options", oid + "^{commit}"}, oid, nil)
	runner.ExpectGitArgs([]string{"rev-list", "--parents", "-n", "1", oid, "--"}, oid+" "+parent, nil)
	err := controller.diffHistoricalFileWithWorktree()
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want terminal error %v", err, sentinel)
	}
	if len(gui.args) != 1 || gui.refreshes != 1 {
		t.Fatalf("subprocesses=%d refreshes=%d, want one each", len(gui.args), gui.refreshes)
	}
	runner.CheckForMissingCalls()
}
