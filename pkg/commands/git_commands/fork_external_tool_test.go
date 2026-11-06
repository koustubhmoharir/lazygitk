package git_commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func TestParseExternalToolHelp(t *testing.T) {
	gitOutput := "'git difftool --tool=<tool>' may be set to one of the following:\n" +
		"\t\tgvimdiff         Use gVim (requires a graphical session)\n" +
		"\t\tvimdiff          Use Vim\n" +
		"\n" +
		"\tuser-defined:\n" +
		"\t\tfoo.cmd true\n" +
		"\n" +
		"The following tools are valid, but not currently available:\n" +
		"\t\taraxis           Use Araxis Merge (requires a graphical session)\n" +
		"\n" +
		"Some of the tools listed above only work in a windowed\n" +
		"environment. If run in a terminal-only session, they will fail.\n"

	assert.Equal(t, []ExternalTool{
		{Name: "gvimdiff", Description: "Use gVim (requires a graphical session)"},
		{Name: "vimdiff", Description: "Use Vim"},
		{Name: "foo", Description: "true"},
	}, parseExternalToolHelp(gitOutput))
}

func TestParseExternalToolHelpWithoutAvailableTools(t *testing.T) {
	output := "'git difftool --tool=<tool>' may be set to one of the following:\n" +
		"\nThe following tools are valid, but not currently available:\n" +
		"\t\taraxis           Use Araxis Merge (requires a graphical session)\n"
	assert.Empty(t, parseExternalToolHelp(output))
	assert.Empty(t, parseExternalToolHelp(""))
}

func TestParseExternalToolHelpUserDefinedOnly(t *testing.T) {
	output := "'git difftool --tool=<tool>' may be set to one of the following:\n" +
		"\tuser-defined:\n" +
		"\t\tmy.tool.cmd code --diff \"$LOCAL\" \"$REMOTE\"\r\n" +
		"\t\tmy.tool.cmd duplicate\n" +
		"\nThe following tools are valid, but not currently available:\n"
	assert.Equal(t, []ExternalTool{
		{Name: "my.tool", Description: `code --diff "$LOCAL" "$REMOTE"`},
	}, parseExternalToolHelp(output))
}

// externalToolRepo returns a repository whose global config is a private file,
// so that nothing reads or writes the user's Git configuration. GUI selection
// starts from a known state: no GIT_MERGETOOL_GUI and no DISPLAY.
func externalToolRepo(t *testing.T) (string, string) {
	t.Helper()
	root := initHistoricalRepo(t)
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	assert.NoError(t, os.WriteFile(globalConfig, nil, 0o600))
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_MERGETOOL_GUI", "")
	t.Setenv("DISPLAY", "")
	return root, globalConfig
}

func externalToolDiffCommands(root string) *DiffCommands {
	osCommand := oscommands.NewDummyOSCommand()
	return NewDiffCommands(NewGitCommon(nil, nil, osCommand.Cmd, osCommand, MockRepoPaths(root), nil, nil))
}

// commitAndModifyFile commits file.txt and then changes it, so that a difftool
// of the file has something to show.
func commitAndModifyFile(t *testing.T, root string) {
	t.Helper()
	writeTestFile(t, filepath.Join(root, "file.txt"), []byte("before\n"), 0o644)
	gitTest(t, root, "add", "file.txt")
	gitTest(t, root, "commit", "-qm", "initial")
	writeTestFile(t, filepath.Join(root, "file.txt"), []byte("after\n"), 0o644)
}

// markerTool returns a difftool command that writes marker when Git runs it.
func markerTool(marker string) string {
	return "printf launched > '" + marker + "'"
}

func TestConfiguredExternalToolReadsDiffThenMergeKeys(t *testing.T) {
	root, _ := externalToolRepo(t)
	diff := externalToolDiffCommands(root)

	configured, err := diff.ConfiguredExternalTool(ExternalToolDiff)
	assert.NoError(t, err, "a missing key is not an error")
	assert.Equal(t, "", configured)

	gitTest(t, root, "config", "merge.tool", "mergefake")
	configured, err = diff.ConfiguredExternalTool(ExternalToolDiff)
	assert.NoError(t, err)
	assert.Equal(t, "mergefake", configured, "git difftool falls back to merge.tool")
	configured, err = diff.ConfiguredExternalTool(ExternalToolMerge)
	assert.NoError(t, err)
	assert.Equal(t, "mergefake", configured)

	gitTest(t, root, "config", "diff.tool", "difffake")
	configured, err = diff.ConfiguredExternalTool(ExternalToolDiff)
	assert.NoError(t, err)
	assert.Equal(t, "difffake", configured, "diff.tool wins over merge.tool")
	configured, err = diff.ConfiguredExternalTool(ExternalToolMerge)
	assert.NoError(t, err)
	assert.Equal(t, "mergefake", configured, "merge mode never reads diff.tool")
}

// Git's GUI selection, as git-mergetool--lib and git difftool make it: the
// guiDefault key of the mode, where auto means DISPLAY is set. Each case has
// only a GUI tool and a terminal tool configured, so the result shows which one
// Git would pick.
func TestConfiguredExternalToolFollowsGuiDefault(t *testing.T) {
	cases := []struct {
		name       string
		mode       ExternalToolMode
		guiDefault string // the mode's own key; "" leaves it unset
		otherGui   string // the other mode's guiDefault, which must not matter
		display    string
		want       string
	}{
		{name: "diff unset is terminal", mode: ExternalToolDiff, want: "plain"},
		{name: "diff false is terminal", mode: ExternalToolDiff, guiDefault: "false", want: "plain"},
		{name: "diff true is GUI", mode: ExternalToolDiff, guiDefault: "true", want: "gui"},
		{name: "diff yes is GUI", mode: ExternalToolDiff, guiDefault: "yes", want: "gui"},
		{name: "diff auto with DISPLAY is GUI", mode: ExternalToolDiff, guiDefault: "auto", display: ":0", want: "gui"},
		{name: "diff Auto with DISPLAY is GUI", mode: ExternalToolDiff, guiDefault: "Auto", display: ":0", want: "gui"},
		{name: "diff auto without DISPLAY is terminal", mode: ExternalToolDiff, guiDefault: "auto", want: "plain"},
		{name: "diff ignores mergetool.guiDefault", mode: ExternalToolDiff, otherGui: "true", want: "plain"},
		{name: "merge unset is terminal", mode: ExternalToolMerge, want: "plain"},
		{name: "merge true is GUI", mode: ExternalToolMerge, guiDefault: "true", want: "gui"},
		{name: "merge auto with DISPLAY is GUI", mode: ExternalToolMerge, guiDefault: "auto", display: ":0", want: "gui"},
		{name: "merge auto without DISPLAY is terminal", mode: ExternalToolMerge, guiDefault: "auto", want: "plain"},
		{name: "merge ignores difftool.guiDefault", mode: ExternalToolMerge, otherGui: "true", want: "plain"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := externalToolRepo(t)
			t.Setenv("DISPLAY", tc.display)
			prefix := "diff"
			if tc.mode == ExternalToolMerge {
				prefix = "merge"
			}
			gitTest(t, root, "config", prefix+".tool", "plain")
			gitTest(t, root, "config", prefix+".guitool", "gui")
			if tc.guiDefault != "" {
				gitTest(t, root, "config", tc.mode.guiDefaultKey(), tc.guiDefault)
			}
			if tc.otherGui != "" {
				other := ExternalToolMerge
				if tc.mode == ExternalToolMerge {
					other = ExternalToolDiff
				}
				gitTest(t, root, "config", other.guiDefaultKey(), tc.otherGui)
			}

			configured, err := externalToolDiffCommands(root).ConfiguredExternalTool(tc.mode)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, configured)
		})
	}
}

// The GUI keys come before the terminal keys, in Git's order: diff.guitool,
// merge.guitool, diff.tool, merge.tool. Each key is removed in turn, so every
// position in the order is checked.
func TestConfiguredExternalToolPrecedenceOrder(t *testing.T) {
	root, _ := externalToolRepo(t)
	diff := externalToolDiffCommands(root)
	for _, key := range []string{"diff.guitool", "merge.guitool", "diff.tool", "merge.tool"} {
		gitTest(t, root, "config", key, key+"-value")
	}

	gitTest(t, root, "config", "difftool.guiDefault", "true")
	gitTest(t, root, "config", "mergetool.guiDefault", "true")
	expectDiff := []string{"diff.guitool-value", "merge.guitool-value", "diff.tool-value", "merge.tool-value"}
	for _, want := range expectDiff {
		configured, err := diff.ConfiguredExternalTool(ExternalToolDiff)
		assert.NoError(t, err)
		assert.Equal(t, want, configured)
		gitTest(t, root, "config", "--unset", strings.TrimSuffix(want, "-value"))
	}
	gitTest(t, root, "config", "merge.guitool", "merge.guitool-value")
	gitTest(t, root, "config", "merge.tool", "merge.tool-value")
	gitTest(t, root, "config", "--unset", "merge.guitool")
	configured, err := diff.ConfiguredExternalTool(ExternalToolMerge)
	assert.NoError(t, err)
	assert.Equal(t, "merge.tool-value", configured, "merge GUI mode reads merge.guitool then merge.tool")

	gitTest(t, root, "config", "difftool.guiDefault", "false")
	gitTest(t, root, "config", "mergetool.guiDefault", "false")
	gitTest(t, root, "config", "diff.tool", "diff.tool-value")
	gitTest(t, root, "config", "merge.guitool", "merge.guitool-value")
	configured, err = diff.ConfiguredExternalTool(ExternalToolDiff)
	assert.NoError(t, err)
	assert.Equal(t, "diff.tool-value", configured, "terminal mode never reads the guitool keys")
	gitTest(t, root, "config", "--unset", "diff.tool")
	configured, err = diff.ConfiguredExternalTool(ExternalToolDiff)
	assert.NoError(t, err)
	assert.Equal(t, "merge.tool-value", configured)
}

// GIT_MERGETOOL_GUI is what --gui and --no-gui set, and Git reads it before
// guiDefault. Only "true" means GUI.
func TestConfiguredExternalToolGitMergetoolGuiOverride(t *testing.T) {
	root, _ := externalToolRepo(t)
	diff := externalToolDiffCommands(root)
	gitTest(t, root, "config", "diff.tool", "plain")
	gitTest(t, root, "config", "diff.guitool", "gui")
	gitTest(t, root, "config", "difftool.guiDefault", "true")

	t.Setenv("GIT_MERGETOOL_GUI", "false")
	configured, err := diff.ConfiguredExternalTool(ExternalToolDiff)
	assert.NoError(t, err)
	assert.Equal(t, "plain", configured, "GIT_MERGETOOL_GUI=false beats guiDefault=true")

	t.Setenv("GIT_MERGETOOL_GUI", "true")
	gitTest(t, root, "config", "difftool.guiDefault", "false")
	configured, err = diff.ConfiguredExternalTool(ExternalToolDiff)
	assert.NoError(t, err)
	assert.Equal(t, "gui", configured, "GIT_MERGETOOL_GUI=true beats guiDefault=false")
}

func TestConfiguredExternalToolRejectsBadGuiDefault(t *testing.T) {
	root, _ := externalToolRepo(t)
	gitTest(t, root, "config", "diff.tool", "plain")
	gitTest(t, root, "config", "difftool.guiDefault", "maybe")

	_, err := externalToolDiffCommands(root).ConfiguredExternalTool(ExternalToolDiff)
	assert.Error(t, err, "Git itself fails on a non-boolean guiDefault")
}

func TestAvailableExternalToolsListsUserDefinedTool(t *testing.T) {
	root, _ := externalToolRepo(t)
	gitTest(t, root, "config", "difftool.fake.cmd", "true")

	tools, err := externalToolDiffCommands(root).AvailableExternalTools(ExternalToolDiff)
	assert.NoError(t, err)
	assert.Contains(t, tools, ExternalTool{Name: "fake", Description: "true"})
}

func TestSaveExternalToolGloballyWritesOnlyGlobalConfig(t *testing.T) {
	root, globalConfig := externalToolRepo(t)
	diff := externalToolDiffCommands(root)

	assert.NoError(t, diff.SaveExternalToolGlobally(ExternalToolDiff, "savedfake"))
	assert.Equal(t, "savedfake", gitTest(t, root, "config", "--global", "--get", "diff.tool"))
	written, err := os.ReadFile(globalConfig)
	assert.NoError(t, err)
	assert.Contains(t, string(written), "savedfake")
	configured, err := diff.ConfiguredExternalTool(ExternalToolDiff)
	assert.NoError(t, err)
	assert.Equal(t, "savedfake", configured)

	assert.NoError(t, diff.SaveExternalToolGlobally(ExternalToolMerge, "savedmerge"))
	assert.Equal(t, "savedmerge", gitTest(t, root, "config", "--global", "--get", "merge.tool"))
}

func TestApplyExternalToolPinsDiffToolAfterSubcommand(t *testing.T) {
	cmd := oscommands.NewDummyOSCommand().Cmd.New([]string{"git", "difftool", "--no-prompt", "--dir-diff", "--", "difftool"})

	assert.NoError(t, ApplyExternalTool(cmd, ExternalToolDiff, "fake"))

	assert.Equal(t, []string{"git", "difftool", "--tool=fake", "--no-prompt", "--dir-diff", "--", "difftool"}, cmd.Args(),
		"the path named like the subcommand is left alone")
}

func TestApplyExternalToolPinsMergeToolAfterGlobalOptions(t *testing.T) {
	cmd := oscommands.NewDummyOSCommand().Cmd.New([]string{"git", "--literal-pathspecs", "mergetool", "--no-prompt", "--", "a.txt"})

	assert.NoError(t, ApplyExternalTool(cmd, ExternalToolMerge, "fake"))

	assert.Equal(t, []string{"git", "--literal-pathspecs", "mergetool", "--tool=fake", "--no-prompt", "--", "a.txt"}, cmd.Args())
}

func TestApplyExternalToolRefusesCommandWithoutSubcommand(t *testing.T) {
	cmd := oscommands.NewDummyOSCommand().Cmd.New([]string{"git", "diff", "--", "difftool"})

	assert.Error(t, ApplyExternalTool(cmd, ExternalToolDiff, "fake"))
	assert.Equal(t, []string{"git", "diff", "--", "difftool"}, cmd.Args(), "nothing is changed on error")
}

// The pinned tool is the one Git runs, and the tool is picked by --tool. The
// fake tool records that it ran, and Git does not print the "not configured"
// guess.
func TestAppliedExternalToolLaunchesWithoutGuessing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake tool is a POSIX shell command")
	}
	root, _ := externalToolRepo(t)
	commitAndModifyFile(t, root)
	marker := filepath.Join(t.TempDir(), "launched")
	gitTest(t, root, "config", "difftool.fake.cmd", markerTool(marker))

	diff := externalToolDiffCommands(root)
	cmd := diff.OpenDiffToolCmdObj(DiffToolCmdOptions{Filepath: "file.txt"}).SetWd(root)
	assert.NoError(t, ApplyExternalTool(cmd, ExternalToolDiff, "fake"))

	_, stderr, err := cmd.RunWithOutputs()
	assert.NoError(t, err, stderr)
	assert.NotContains(t, stderr, "is not configured")
	content, readErr := os.ReadFile(marker)
	assert.NoError(t, readErr, "the fake tool did not run")
	assert.Equal(t, "launched", strings.TrimSpace(string(content)))

	configured, err := diff.ConfiguredExternalTool(ExternalToolDiff)
	assert.NoError(t, err)
	assert.Equal(t, "", configured, "the launch must not save the tool")
}

// Execution-level check of the guess Design forbids: diff.tool names a tool that
// Git does not know, and a harmless difftool.vimdiff.cmd would run if Git fell
// back to guessing. The pinned launch must fail, and the guess must not run.
func TestPinnedUnknownToolFailsInsteadOfGuessing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the marker tool is a POSIX shell command")
	}
	root, _ := externalToolRepo(t)
	commitAndModifyFile(t, root)
	guessMarker := filepath.Join(t.TempDir(), "guessed")
	gitTest(t, root, "config", "diff.tool", "definitely-not-a-real-tool")
	gitTest(t, root, "config", "difftool.vimdiff.cmd", markerTool(guessMarker))

	diff := externalToolDiffCommands(root)
	configured, err := diff.ConfiguredExternalTool(ExternalToolDiff)
	assert.NoError(t, err)
	assert.Equal(t, "definitely-not-a-real-tool", configured)

	cmd := diff.OpenDiffToolCmdObj(DiffToolCmdOptions{Filepath: "file.txt"}).SetWd(root)
	assert.NoError(t, ApplyExternalTool(cmd, ExternalToolDiff, configured))
	_, stderr, err := cmd.RunWithOutputs()
	assert.Error(t, err, "an unknown pinned tool must fail")
	assert.Contains(t, stderr, "definitely-not-a-real-tool")
	assert.NotContains(t, stderr, "Resetting to default")
	_, statErr := os.Stat(guessMarker)
	assert.True(t, os.IsNotExist(statErr), "Git guessed a tool and ran it")
}

// Execution-level check of the GUI case: diff.guitool would win in GUI mode, and
// the user chose a terminal tool. The pinned launch runs the chosen tool, and
// the GUI tool does not run.
func TestPinnedToolIgnoresConfiguredGuiTool(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the marker tools are POSIX shell commands")
	}
	root, _ := externalToolRepo(t)
	commitAndModifyFile(t, root)
	t.Setenv("DISPLAY", ":0")
	guiMarker := filepath.Join(t.TempDir(), "gui")
	chosenMarker := filepath.Join(t.TempDir(), "chosen")
	gitTest(t, root, "config", "difftool.guiDefault", "true")
	gitTest(t, root, "config", "diff.guitool", "guiprobe")
	gitTest(t, root, "config", "difftool.guiprobe.cmd", markerTool(guiMarker))
	gitTest(t, root, "config", "difftool.chosen.cmd", markerTool(chosenMarker))

	diff := externalToolDiffCommands(root)
	configured, err := diff.ConfiguredExternalTool(ExternalToolDiff)
	assert.NoError(t, err)
	assert.Equal(t, "guiprobe", configured, "detection sees the GUI tool")

	cmd := diff.OpenDiffToolCmdObj(DiffToolCmdOptions{Filepath: "file.txt"}).SetWd(root)
	assert.NoError(t, ApplyExternalTool(cmd, ExternalToolDiff, "chosen"))
	_, stderr, err := cmd.RunWithOutputs()
	assert.NoError(t, err, stderr)
	_, chosenErr := os.Stat(chosenMarker)
	assert.NoError(t, chosenErr, "the chosen tool did not run")
	_, guiErr := os.Stat(guiMarker)
	assert.True(t, os.IsNotExist(guiErr), "the configured GUI tool ran instead of the chosen one")
}
