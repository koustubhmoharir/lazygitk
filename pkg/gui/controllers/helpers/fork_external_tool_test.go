package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type externalToolUI struct {
	*namedCheckoutFakeIGuiCommon
	state *capturedTagState
	menus []types.CreateMenuOptions
}

func (u *externalToolUI) State() types.IStateAccessor { return u.state }

func (u *externalToolUI) Menu(opts types.CreateMenuOptions) error {
	u.menus = append(u.menus, opts)
	return nil
}

type externalToolFixture struct {
	t        *testing.T
	root     string
	global   string
	ui       *externalToolUI
	c        *HelperCommon
	cancels  int
	launches [][]string
}

func newExternalToolFixture(t *testing.T) *externalToolFixture {
	t.Helper()
	root := t.TempDir()
	namedCheckoutGit(t, root, "init", "-q", "-b", "main")
	global := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(global, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	// Every test that saves reads and writes this file, never the user's config.
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	paths := git_commands.MockRepoPaths(root)
	osCmd := oscommands.NewDummyOSCommand()
	cmn := git_commands.NewGitCommon(common.NewDummyCommon(), nil, osCmd.Cmd, osCmd, paths, nil, nil)
	gitCmd := &commands.GitCommand{Diff: git_commands.NewDiffCommands(cmn), RepoPaths: paths}
	ui := &externalToolUI{
		namedCheckoutFakeIGuiCommon: &namedCheckoutFakeIGuiCommon{git: gitCmd},
		state:                       &capturedTagState{generation: 1},
	}
	return &externalToolFixture{
		t: t, root: root, global: global, ui: ui,
		c: &HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui},
	}
}

// launch records the environment the command would run with, after apply.
func (f *externalToolFixture) launch(apply func(*oscommands.CmdObj) error) error {
	cmd := oscommands.NewDummyOSCommand().Cmd.New([]string{"git", "difftool"})
	cmd.SetEnviron([]string{"PATH=/bin"})
	if err := apply(cmd); err != nil {
		return err
	}
	f.launches = append(f.launches, cmd.GetEnvVars())
	return nil
}

func (f *externalToolFixture) cancel() error {
	f.cancels++
	return nil
}

// run starts the flow and then runs what it queued, as the GUI would: the
// tool listing on its worker, then the UI callback that opens the menu.
func (f *externalToolFixture) run(mode git_commands.ExternalToolMode) error {
	if err := (&ExternalToolHelper{c: f.c}).WithExternalTool(mode, f.launch, f.cancel); err != nil {
		return err
	}
	for len(f.ui.workers) > 0 {
		worker := f.ui.workers[0]
		f.ui.workers = f.ui.workers[1:]
		if err := worker(nil); err != nil {
			return err
		}
	}
	for len(f.ui.ui) > 0 {
		callback := f.ui.ui[0]
		f.ui.ui = f.ui.ui[1:]
		if err := callback(); err != nil {
			return err
		}
	}
	return nil
}

func (f *externalToolFixture) lastMenu() types.CreateMenuOptions {
	f.t.Helper()
	if len(f.ui.menus) == 0 {
		f.t.Fatal("expected a menu")
	}
	return f.ui.menus[len(f.ui.menus)-1]
}

func findMenuItem(t *testing.T, opts types.CreateMenuOptions, commandID string) *types.MenuItem {
	t.Helper()
	for _, item := range opts.Items {
		if item.CommandID == commandID {
			return item
		}
	}
	t.Fatalf("no menu item %q in %q", commandID, opts.Title)
	return nil
}

// findToolItem returns the chooser item for the named tool, since the builtin
// tools that are installed here come before it.
func findToolItem(t *testing.T, opts types.CreateMenuOptions, name string) *types.MenuItem {
	t.Helper()
	for _, item := range opts.Items {
		if item.LabelColumns[0] == name {
			return item
		}
	}
	t.Fatalf("no tool %q in %q", name, opts.Title)
	return nil
}

func hasEnv(env []string, entry string) bool {
	for _, value := range env {
		if value == entry {
			return true
		}
	}
	return false
}

func TestExternalToolConfiguredLaunchesWithoutMenuOrEnvChange(t *testing.T) {
	f := newExternalToolFixture(t)
	namedCheckoutGit(t, f.root, "config", "diff.tool", "configured")

	if err := f.run(git_commands.ExternalToolDiff); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 0 || f.cancels != 0 || len(f.launches) != 1 {
		t.Fatalf("menus=%d cancels=%d launches=%d", len(f.ui.menus), f.cancels, len(f.launches))
	}
	for _, entry := range f.launches[0] {
		if strings.HasPrefix(entry, "GIT_CONFIG_") {
			t.Fatalf("configured launch changed the environment: %q", entry)
		}
	}
}

func TestExternalToolJustThisTimeAppliesToolForOneLaunch(t *testing.T) {
	f := newExternalToolFixture(t)
	namedCheckoutGit(t, f.root, "config", "difftool.fake.cmd", "true")

	if err := f.run(git_commands.ExternalToolDiff); err != nil {
		t.Fatal(err)
	}
	chooser := f.lastMenu()
	if chooser.Title != f.c.Tr.ExternalToolChooseDiffTitle {
		t.Fatalf("chooser title = %q", chooser.Title)
	}
	tool := findToolItem(t, chooser, "fake")
	if tool.LabelColumns[1] != "true" {
		t.Fatalf("unexpected tool label: %q", tool.LabelColumns)
	}
	if tool.CommandID != "diff-tool.choose" || tool.Verb != "choose" {
		t.Fatalf("tool item identity = %q/%q", tool.CommandID, tool.Verb)
	}
	if err := tool.OnPress(); err != nil {
		t.Fatal(err)
	}

	use := f.lastMenu()
	if use.Title != "Use fake" {
		t.Fatalf("use title = %q", use.Title)
	}
	onceItem := use.Items[0]
	if onceItem.Label != f.c.Tr.ExternalToolUseOnce || onceItem.CommandID != "diff-tool.use-once" || onceItem.Verb != "use-once" {
		t.Fatalf("first item is not Just this time: %+v", onceItem)
	}
	if use.Items[1].CommandID != "diff-tool.save" || use.Items[1].Verb != "save" {
		t.Fatalf("second item identity = %q/%q", use.Items[1].CommandID, use.Items[1].Verb)
	}
	if err := onceItem.OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(f.launches) != 1 || f.cancels != 0 {
		t.Fatalf("launches=%d cancels=%d", len(f.launches), f.cancels)
	}
	if !hasEnv(f.launches[0], "GIT_CONFIG_COUNT=1") || !hasEnv(f.launches[0], "GIT_CONFIG_KEY_0=diff.tool") || !hasEnv(f.launches[0], "GIT_CONFIG_VALUE_0=fake") {
		t.Fatalf("launch env does not select the tool: %q", f.launches[0])
	}
	if got, _ := os.ReadFile(f.global); len(got) != 0 {
		t.Fatalf("just this time wrote the global config: %q", got)
	}
}

func TestExternalToolEscapeCancelsAtEitherStep(t *testing.T) {
	f := newExternalToolFixture(t)
	namedCheckoutGit(t, f.root, "config", "difftool.fake.cmd", "true")
	if err := f.run(git_commands.ExternalToolDiff); err != nil {
		t.Fatal(err)
	}
	if err := f.lastMenu().OnCancel(); err != nil {
		t.Fatal(err)
	}
	if f.cancels != 1 || len(f.launches) != 0 {
		t.Fatalf("step 1 escape: cancels=%d launches=%d", f.cancels, len(f.launches))
	}

	if err := f.run(git_commands.ExternalToolDiff); err != nil {
		t.Fatal(err)
	}
	if err := f.lastMenu().Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.lastMenu().OnCancel(); err != nil {
		t.Fatal(err)
	}
	if f.cancels != 2 || len(f.launches) != 0 {
		t.Fatalf("step 2 escape: cancels=%d launches=%d", f.cancels, len(f.launches))
	}
}

func TestExternalToolSaveWritesGlobalConfigAndLaunchesUnchanged(t *testing.T) {
	f := newExternalToolFixture(t)
	namedCheckoutGit(t, f.root, "config", "difftool.fake.cmd", "true")
	if err := f.run(git_commands.ExternalToolDiff); err != nil {
		t.Fatal(err)
	}
	if err := findToolItem(t, f.lastMenu(), "fake").OnPress(); err != nil {
		t.Fatal(err)
	}
	save := findMenuItem(t, f.lastMenu(), "diff-tool.save")
	if err := save.OnPress(); err != nil {
		t.Fatal(err)
	}
	if got := namedCheckoutGit(t, f.root, "config", "--global", "--get", "diff.tool"); got != "fake" {
		t.Fatalf("global diff.tool = %q", got)
	}
	if len(f.launches) != 1 || f.cancels != 0 {
		t.Fatalf("launches=%d cancels=%d", len(f.launches), f.cancels)
	}
	for _, entry := range f.launches[0] {
		if strings.HasPrefix(entry, "GIT_CONFIG_") {
			t.Fatalf("save launch changed the environment: %q", entry)
		}
	}
}

func TestExternalToolFailedSaveLaunchesNothingAndCancels(t *testing.T) {
	f := newExternalToolFixture(t)
	namedCheckoutGit(t, f.root, "config", "difftool.fake.cmd", "true")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "missing", "gitconfig"))
	if err := f.run(git_commands.ExternalToolDiff); err != nil {
		t.Fatal(err)
	}
	if err := findToolItem(t, f.lastMenu(), "fake").OnPress(); err != nil {
		t.Fatal(err)
	}
	err := findMenuItem(t, f.lastMenu(), "diff-tool.save").OnPress()
	if err == nil {
		t.Fatal("expected the save error")
	}
	if len(f.launches) != 0 || f.cancels != 1 {
		t.Fatalf("launches=%d cancels=%d", len(f.launches), f.cancels)
	}
}

func TestExternalToolNoToolsReturnsErrorAndCancels(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH is restricted with a symlink")
	}
	f := newExternalToolFixture(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	// Only git is on PATH, so Git lists no tool as available.
	bin := t.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	err = f.run(git_commands.ExternalToolDiff)
	if err == nil || err.Error() != f.c.Tr.ExternalToolNotFoundDiff {
		t.Fatalf("error = %v", err)
	}
	if len(f.ui.menus) != 0 || len(f.launches) != 0 || f.cancels != 1 {
		t.Fatalf("menus=%d launches=%d cancels=%d", len(f.ui.menus), len(f.launches), f.cancels)
	}
}

func TestExternalToolRepositoryChangeRefusesMenuItems(t *testing.T) {
	f := newExternalToolFixture(t)
	namedCheckoutGit(t, f.root, "config", "difftool.fake.cmd", "true")
	if err := f.run(git_commands.ExternalToolDiff); err != nil {
		t.Fatal(err)
	}
	f.ui.state.generation = 2
	err := findToolItem(t, f.lastMenu(), "fake").OnPress()
	if err == nil || err.Error() != f.c.Tr.BranchSyncRepositoryChanged {
		t.Fatalf("tool press error = %v", err)
	}
	if len(f.ui.menus) != 1 || f.cancels != 1 {
		t.Fatalf("menus=%d cancels=%d", len(f.ui.menus), f.cancels)
	}

	f.ui.state.generation = 1
	if err := f.run(git_commands.ExternalToolDiff); err != nil {
		t.Fatal(err)
	}
	if err := findToolItem(t, f.lastMenu(), "fake").OnPress(); err != nil {
		t.Fatal(err)
	}
	f.ui.state.generation = 3
	err = f.lastMenu().Items[0].OnPress()
	if err == nil || err.Error() != f.c.Tr.BranchSyncRepositoryChanged {
		t.Fatalf("use-once error = %v", err)
	}
	if len(f.launches) != 0 || f.cancels != 2 {
		t.Fatalf("launches=%d cancels=%d", len(f.launches), f.cancels)
	}
}

func TestExternalToolMergeModeUsesMergeKeys(t *testing.T) {
	f := newExternalToolFixture(t)
	namedCheckoutGit(t, f.root, "config", "mergetool.fake.cmd", "true")
	if err := f.run(git_commands.ExternalToolMerge); err != nil {
		t.Fatal(err)
	}
	chooser := f.lastMenu()
	if chooser.Title != f.c.Tr.ExternalToolChooseMergeTitle {
		t.Fatalf("chooser title = %q", chooser.Title)
	}
	if err := findToolItem(t, chooser, "fake").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.lastMenu().Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(f.launches) != 1 || !hasEnv(f.launches[0], "GIT_CONFIG_KEY_0=merge.tool") {
		t.Fatalf("merge launch env = %q", f.launches)
	}
}
