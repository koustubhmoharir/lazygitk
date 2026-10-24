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
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type historicalComparisonState struct {
	types.IStateAccessor
	generation int
}

func (s *historicalComparisonState) GetRepoGeneration() int { return s.generation }

type historicalComparisonUI struct {
	types.IGuiCommon
	git      *commands.GitCommand
	state    *historicalComparisonState
	prompts  []types.PromptOpts
	commands []*oscommands.CmdObj
}

func (ui *historicalComparisonUI) Git() *commands.GitCommand   { return ui.git }
func (ui *historicalComparisonUI) State() types.IStateAccessor { return ui.state }
func (ui *historicalComparisonUI) Prompt(opts types.PromptOpts) {
	ui.prompts = append(ui.prompts, opts)
}

func (ui *historicalComparisonUI) RunSubprocess(cmd *oscommands.CmdObj) (bool, error) {
	ui.commands = append(ui.commands, cmd)
	return false, nil
}
func (ui *historicalComparisonUI) Refresh(types.RefreshOptions) {}

func historicalComparisonRepo(t *testing.T) (*commands.GitCommand, string, string) {
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
	if err := os.WriteFile(filepath.Join(root, "one"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	run("add", "one")
	run("commit", "-qm", "one")
	left := run("rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(root, "two"), []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	run("add", "two")
	run("commit", "-qm", "two")
	right := run("rev-parse", "HEAD")
	osCmd := oscommands.NewDummyOSCommand()
	git := commands.NewGitCommandAux(common.NewDummyCommon(), nil, osCmd, git_config.NewFakeGitConfig(nil), git_commands.MockRepoPaths(root), nil)
	return git, left, right
}

func TestHistoricalCommitComparisonPromptCancellationAndCapturedTarget(t *testing.T) {
	git, left, right := historicalComparisonRepo(t)
	ui := &historicalComparisonUI{git: git, state: &historicalComparisonState{}}
	c := NewControllerCommon(&helpers.HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui}, nil)
	paths := []string{"old-name"}
	target, err := newHistoricalCommitComparisonTarget(c, left, paths, false)
	if err != nil {
		t.Fatal(err)
	}
	paths[0] = "mutated-after-prompt"
	target.prompt()
	if len(ui.prompts) != 1 {
		t.Fatalf("prompts=%d", len(ui.prompts))
	}
	if err := ui.prompts[0].HandleClose(); err != nil {
		t.Fatal(err)
	}
	if len(ui.commands) != 0 {
		t.Fatal("cancelled prompt ran a subprocess")
	}
	if err := ui.prompts[0].HandleConfirm(right); err != nil {
		t.Fatal(err)
	}
	if len(ui.commands) != 1 {
		t.Fatalf("subprocesses=%d", len(ui.commands))
	}
	args := ui.commands[0].Args()
	if got := strings.Join(args, "\x00"); !strings.Contains(got, "\x00"+left+"\x00"+right+"\x00--\x00old-name") {
		t.Fatalf("comparison command lost captured selection: %#v", args)
	}
}

func TestHistoricalCommitComparisonRejectsRepositoryChange(t *testing.T) {
	git, left, right := historicalComparisonRepo(t)
	ui := &historicalComparisonUI{git: git, state: &historicalComparisonState{}}
	c := NewControllerCommon(&helpers.HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui}, nil)
	target, err := newHistoricalCommitComparisonTarget(c, left, []string{"."}, true)
	if err != nil {
		t.Fatal(err)
	}
	target.prompt()
	ui.state.generation++
	if err := ui.prompts[0].HandleConfirm(right); err == nil {
		t.Fatal("expected changed-repository refusal")
	}
	if len(ui.commands) != 0 {
		t.Fatal("changed repository ran a subprocess")
	}
}

func TestHistoricalCommitComparisonBindingsAreKeylessMenuActions(t *testing.T) {
	keybindings := types.KeybindingsOpts{Config: config.GetDefaultConfig().Keybinding, GetKeys: config.GetValidatedKeyBindingKeys}
	basic := historicalTestBasicController(nil, 0, 0)
	basic.ListControllerTrait = &ListControllerTrait[*models.Commit]{}
	foundCommit, foundFile := false, false
	for _, binding := range basic.historicalDiffBindings(keybindings) {
		if binding.CommandID != "diff.commit.selected-commit" {
			continue
		}
		if binding.Verb != "diff-commit" || len(binding.Keys) != 0 || !binding.OpensMenu {
			t.Fatalf("commit comparison binding = %#v", binding)
		}
		foundCommit = true
	}
	fileController, _ := historicalTestFileController(t, nil, nil, false)
	for _, binding := range fileController.historicalDiffBindings(keybindings) {
		if binding.CommandID != "diff.history-file.selected-commit" {
			continue
		}
		if binding.Verb != "diff-commit" || len(binding.Keys) != 0 || !binding.OpensMenu {
			t.Fatalf("historical-file comparison binding = %#v", binding)
		}
		foundFile = true
	}
	if !foundCommit || !foundFile {
		t.Fatalf("comparison bindings found: commit=%v file=%v", foundCommit, foundFile)
	}
}

func TestHistoricalCommitComparisonAllowsDeletedHistoricalFile(t *testing.T) {
	commit := historicalTestCommit("0123456789abcdef0123456789abcdef01234567", "1123456789abcdef0123456789abcdef01234567")
	controller, _ := historicalTestFileController(t, commit, []*models.CommitFile{{Path: "deleted.txt", ChangeStatus: "D"}}, false)
	if reason := controller.canHistoricalFileComparison(); reason != nil {
		t.Fatalf("deleted historical path comparison disabled: %v", reason)
	}
}

func TestHistoricalCommitComparisonSelectedOIDCannotBeShadowedByRef(t *testing.T) {
	git, left, right := historicalComparisonRepo(t)
	cmd := exec.Command("git", "-C", git.RepoPaths.WorktreePath(), "update-ref", "refs/heads/"+left, right)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create hex ref: %v: %s", err, output)
	}
	ui := &historicalComparisonUI{git: git, state: &historicalComparisonState{}}
	c := NewControllerCommon(&helpers.HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui}, nil)
	target, err := newHistoricalCommitComparisonTarget(c, left, []string{"."}, true)
	if err != nil {
		t.Fatal(err)
	}
	if target.left.OID != left {
		t.Fatalf("selected left = %s, want %s", target.left.OID, left)
	}
}
