package git_commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/app/daemon"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCombineCommandEditorProcess(t *testing.T) {
	if os.Getenv("LAZYGIT_TEST_COMBINE_COMMAND_EDITOR") != "1" {
		return
	}
	os.Args = []string{os.Args[0], os.Args[len(os.Args)-1]}
	daemon.Handle(logrus.NewEntry(logrus.New()))
	os.Exit(0)
}

func combineCommandPayload(t *testing.T, cmd *oscommands.CmdObj) string {
	t.Helper()
	for _, entry := range cmd.GetCmd().Env {
		if data, ok := strings.CutPrefix(entry, daemon.DaemonInstructionEnvKey+"="); ok {
			var instruction daemon.CombineRebaseInstruction
			if err := json.Unmarshal([]byte(data), &instruction); err != nil {
				t.Fatal(err)
			}
			return instruction.PayloadPath
		}
	}
	t.Fatal("missing private payload instruction")
	return ""
}

func TestCombineCommandRunsReviewedRootAndNonrootHistory(t *testing.T) {
	for _, rootCombine := range []bool{true, false} {
		t.Run(map[bool]string{true: "root", false: "nonroot"}[rootCombine], func(t *testing.T) {
			root := initHistoricalRepo(t)
			if !rootCombine {
				capturedCherryPickCommit(t, root, "base", "base\n", "base", capturedCherryPickDate)
			}
			keeper := capturedCherryPickCommit(t, root, "keeper", "keeper\n", "keeper", capturedCherryPickDate)
			middle := capturedCherryPickCommit(t, root, "middle", "middle\n", "middle", capturedCherryPickDate)
			selected := capturedCherryPickCommit(t, root, "selected", "selected\n", "selected", capturedCherryPickDate)
			gitTest(t, root, "branch", "unchanged", selected)
			gitTest(t, root, "config", "rebase.updateRefs", "true")
			gitTest(t, root, "config", "rebase.autoStash", "true")
			gitTest(t, root, "config", "rebase.autoSquash", "true")
			writeTestFile(t, filepath.Join(root, "harmless"), []byte("untracked\n"), 0o600)
			commands := selectedCommitRebaseCommands(t, root)
			plan, err := commands.PrepareCombinePlan([]string{selected, keeper})
			selectedRebaseRequireNoError(t, err)
			message := "Approved summary\n\n" + strings.Repeat("preserved text \n", 12000)
			cmd, cleanup, err := commands.CombineCommand(plan, message)
			selectedRebaseRequireNoError(t, err)
			defer cleanup()
			assert.True(t, cmd.LockRetriesDisabled())
			assert.Equal(t, root, cmd.GetCmd().Dir)
			payloadPath := combineCommandPayload(t, cmd)
			info, err := os.Stat(payloadPath)
			selectedRebaseRequireNoError(t, err)
			if filepath.Separator != '\\' {
				assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
			}
			for _, entry := range cmd.GetCmd().Env {
				if strings.Contains(entry, message) {
					t.Fatal("large message was embedded in environment")
				}
			}
			binary, err := os.Executable()
			selectedRebaseRequireNoError(t, err)
			cmd.AddEnvVars("LAZYGIT_TEST_COMBINE_COMMAND_EDITOR=1", `GIT_SEQUENCE_EDITOR="`+filepath.ToSlash(binary)+`" -test.run=^TestCombineCommandEditorProcess$ --`)
			selectedRebaseRequireNoError(t, cmd.Run())
			cleanup()
			_, err = os.Stat(payloadPath)
			assert.True(t, os.IsNotExist(err))
			assert.Equal(t, selected, gitTest(t, root, "rev-parse", "unchanged"), "rebase.updateRefs must not move unrelated branches")
			assert.Equal(t, "middle", gitTest(t, root, "show", "-s", "--format=%s", "HEAD"))
			combined := gitTest(t, root, "rev-parse", "HEAD^")
			assert.NotEqual(t, keeper, combined)
			assert.NotEqual(t, middle, gitTest(t, root, "rev-parse", "HEAD"))
			commit, err := commands.readCombineCommit(combined)
			selectedRebaseRequireNoError(t, err)
			assert.Equal(t, message, commit.Message)
			content, err := os.ReadFile(filepath.Join(root, "harmless"))
			selectedRebaseRequireNoError(t, err)
			assert.Equal(t, "untracked\n", string(content))
			assert.Empty(t, gitTest(t, root, "stash", "list"))
			_, err = os.Stat(filepath.Join(root, ".git", "rebase-merge"))
			assert.True(t, os.IsNotExist(err))
		})
	}
}

func TestCombineCommandRejectsInvalidMessageAndStaleCapture(t *testing.T) {
	root := initHistoricalRepo(t)
	first := capturedCherryPickCommit(t, root, "one", "one\n", "one", capturedCherryPickDate)
	second := capturedCherryPickCommit(t, root, "two", "two\n", "two", capturedCherryPickDate)
	commands := selectedCommitRebaseCommands(t, root)
	plan, err := commands.PrepareCombinePlan([]string{second, first})
	selectedRebaseRequireNoError(t, err)
	for _, message := range []string{"", " \n", "subject\x00body"} {
		cmd, cleanup, err := commands.CombineCommand(plan, message)
		assert.Error(t, err)
		assert.Nil(t, cmd)
		assert.Nil(t, cleanup)
	}
	capturedCherryPickCommit(t, root, "external", "external\n", "external", capturedCherryPickDate)
	cmd, cleanup, err := commands.CombineCommand(plan, "approved")
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	assert.Nil(t, cmd)
	assert.Nil(t, cleanup)
}
