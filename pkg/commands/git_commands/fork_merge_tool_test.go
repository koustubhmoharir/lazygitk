package git_commands

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func mergeToolWorkingTreeCommands(root string) *WorkingTreeCommands {
	osCommand := oscommands.NewDummyOSCommand()
	gitCommon := NewGitCommon(nil, nil, osCommand.Cmd, osCommand, MockRepoPaths(root), nil, nil)
	return NewWorkingTreeCommands(gitCommon, nil, nil)
}

func TestMergeToolCmdObjDisablesBackupsWhenKeyUnset(t *testing.T) {
	root, _ := externalToolRepo(t)
	cmd, err := mergeToolWorkingTreeCommands(root).MergeToolCmdObj([]string{"a.txt", "dir/b file.txt"})
	assert.NoError(t, err)
	assert.Equal(t, []string{
		"git", "-c", "mergetool.keepBackup=false",
		"--literal-pathspecs", "mergetool", "--no-prompt", "--",
		"a.txt", "dir/b file.txt",
	}, cmd.Args())
}

// The user's own setting wins, whatever its value, and it is read on each call:
// a setting made while lazygitk runs applies to the next launch.
func TestMergeToolCmdObjLeavesExplicitKeepBackupAloneAndReadsFresh(t *testing.T) {
	root, _ := externalToolRepo(t)
	commands := mergeToolWorkingTreeCommands(root)
	withoutOverride := []string{"git", "--literal-pathspecs", "mergetool", "--no-prompt", "--", "a.txt"}
	withOverride := []string{"git", "-c", "mergetool.keepBackup=false", "--literal-pathspecs", "mergetool", "--no-prompt", "--", "a.txt"}

	for _, value := range []string{"true", "false", ""} {
		gitTest(t, root, "config", "mergetool.keepBackup", value)
		cmd, err := commands.MergeToolCmdObj([]string{"a.txt"})
		assert.NoError(t, err)
		assert.Equal(t, withoutOverride, cmd.Args(), "value %q is the user's choice", value)
	}

	gitTest(t, root, "config", "--unset", "mergetool.keepBackup")
	cmd, err := commands.MergeToolCmdObj([]string{"a.txt"})
	assert.NoError(t, err)
	assert.Equal(t, withOverride, cmd.Args(), "unset again, so the override applies")
}

func TestMergeToolCmdObjRespectsGlobalKeepBackup(t *testing.T) {
	root, _ := externalToolRepo(t)
	gitTest(t, root, "config", "--global", "mergetool.keepBackup", "true")
	cmd, err := mergeToolWorkingTreeCommands(root).MergeToolCmdObj([]string{"a.txt"})
	assert.NoError(t, err)
	assert.NotContains(t, cmd.Args(), "mergetool.keepBackup=false")
}

// With no paths, git mergetool would merge every conflicted file, so the
// builder refuses.
func TestMergeToolCmdObjRequiresPaths(t *testing.T) {
	root, _ := externalToolRepo(t)
	_, err := mergeToolWorkingTreeCommands(root).MergeToolCmdObj(nil)
	assert.Error(t, err)
}
