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

func TestApplyExternalToolAppendsToExistingConfigEnvironment(t *testing.T) {
	cmd := oscommands.NewDummyOSCommand().Cmd.New([]string{"git", "difftool"})
	cmd.SetEnviron([]string{
		"PATH=/bin",
		"GIT_CONFIG_COUNT=2",
		"GIT_CONFIG_KEY_0=user.name", "GIT_CONFIG_VALUE_0=kept",
		"GIT_CONFIG_KEY_1=core.pager", "GIT_CONFIG_VALUE_1=cat",
	})

	assert.NoError(t, ApplyExternalTool(cmd, ExternalToolDiff, "fake"))

	assert.Equal(t, []string{
		"PATH=/bin",
		"GIT_CONFIG_KEY_0=user.name", "GIT_CONFIG_VALUE_0=kept",
		"GIT_CONFIG_KEY_1=core.pager", "GIT_CONFIG_VALUE_1=cat",
		"GIT_CONFIG_COUNT=3",
		"GIT_CONFIG_KEY_2=diff.tool", "GIT_CONFIG_VALUE_2=fake",
	}, cmd.GetEnvVars())
}

func TestApplyExternalToolStartsCountWithoutExistingEnvironment(t *testing.T) {
	cmd := oscommands.NewDummyOSCommand().Cmd.New([]string{"git", "mergetool"})
	cmd.SetEnviron([]string{"PATH=/bin"})

	assert.NoError(t, ApplyExternalTool(cmd, ExternalToolMerge, "fake"))

	assert.Equal(t, []string{
		"PATH=/bin",
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=merge.tool", "GIT_CONFIG_VALUE_0=fake",
	}, cmd.GetEnvVars())
}

func TestApplyExternalToolRejectsMalformedCount(t *testing.T) {
	cmd := oscommands.NewDummyOSCommand().Cmd.New([]string{"git", "difftool"})
	cmd.SetEnviron([]string{"GIT_CONFIG_COUNT=many"})

	assert.Error(t, ApplyExternalTool(cmd, ExternalToolDiff, "fake"))
	assert.Equal(t, []string{"GIT_CONFIG_COUNT=many"}, cmd.GetEnvVars())
}

// externalToolRepo returns a repository whose global config is a private file,
// so that nothing reads or writes the user's Git configuration.
func externalToolRepo(t *testing.T) (string, string) {
	t.Helper()
	root := initHistoricalRepo(t)
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	assert.NoError(t, os.WriteFile(globalConfig, nil, 0o600))
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	return root, globalConfig
}

func externalToolDiffCommands(root string) *DiffCommands {
	osCommand := oscommands.NewDummyOSCommand()
	return NewDiffCommands(NewGitCommon(nil, nil, osCommand.Cmd, osCommand, MockRepoPaths(root), nil, nil))
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

// The fake tool records that it ran. The test then checks that Git ran it
// without the "not configured" guess, and that no diff.tool was written.
func TestAppliedExternalToolLaunchesWithoutGuessing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake tool is a POSIX shell command")
	}
	root, _ := externalToolRepo(t)
	writeTestFile(t, filepath.Join(root, "file.txt"), []byte("before\n"), 0o644)
	gitTest(t, root, "add", "file.txt")
	gitTest(t, root, "commit", "-qm", "initial")
	writeTestFile(t, filepath.Join(root, "file.txt"), []byte("after\n"), 0o644)
	marker := filepath.Join(t.TempDir(), "launched")
	gitTest(t, root, "config", "difftool.fake.cmd", "printf launched > '"+marker+"'")

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
