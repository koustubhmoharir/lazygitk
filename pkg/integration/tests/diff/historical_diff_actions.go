package diff

import (
	"os"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var HistoricalDiffActions = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Compare a historical commit or file with fixed versions and a chosen worktree file",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(appConfig *config.AppConfig) {
		// Shared integration config can carry legacy alternate bindings from an
		// earlier scenario; exercise the real constructor defaults for this one.
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowWholeGraph = false
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFile(".gitignore", "/chosen file.txt\n")
		shell.CreateFileAndAdd("sample.txt", "before\n")
		shell.GitAdd(".gitignore").Commit("before")
		shell.UpdateFileAndAdd("sample.txt", "historical\n")
		shell.Commit("historical")
		shell.UpdateFileAndAdd("sample.txt", "staged\n")
		shell.UpdateFile("sample.txt", "working\n")
		shell.CreateFile("chosen file.txt", "chosen\n")

		// Record bytes rather than rendered diffs so the scenario verifies which
		// concrete endpoints Git passed to the configured tool.
		shell.CreateFile(".git/fork-diff-tool", `#!/bin/sh
set -eu
git_dir=$(git rev-parse --absolute-git-dir)
left=$1
right=$2
if [ -d "$left" ]; then left="$left/sample.txt"; fi
if [ -d "$right" ]; then right="$right/sample.txt"; fi
printf '%s' "$left" > "$git_dir/diff-source-path"
cat "$left" > "$git_dir/diff-left"
cat "$right" > "$git_dir/diff-right"
if [ -f "$git_dir/fail-diff-tool" ]; then exit 7; fi
exit 0
`)
		shell.MakeExecutable(".git/fork-diff-tool")
		shell.SetConfig("diff.tool", "fork-test")
		shell.RunCommand([]string{
			"git", "config", "--local", "difftool.fork-test.cmd",
			`"$(git rev-parse --absolute-git-dir)/fork-diff-tool" "$LOCAL" "$REMOTE"`,
		})
		shell.SetConfig("difftool.prompt", "false")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		assertToolInputs := func(left, right string) {
			t.FileSystem().
				FileContent(".git/diff-left", Equals(left)).
				FileContent(".git/diff-right", Equals(right))
		}
		assertSourceRemoved := func() bool {
			source, err := os.ReadFile(".git/diff-source-path")
			if err != nil {
				t.Fail("Reading recorded historical diff source path: " + err.Error())
				return false
			}
			t.FileSystem().PathNotPresent(string(source))
			return true
		}
		openWorkingFileAction := func() {
			t.Views().CommitFiles().
				IsFocused().
				Press(keys.Universal.OptionMenu)
			t.ExpectPopup().Menu().
				Title(Equals("Keybindings")).
				Filter("Diff with working file").
				Confirm()
		}

		commitHash := t.Git().GetCommitHash("HEAD")
		t.Views().Commits().
			Focus().
			SelectedLine(Contains("historical"))
		t.Views().Options().Content(Contains("Diff with working tree: d w"))
		t.Views().Commits().Press(config.Keybinding{"c"})
		t.GlobalPress(config.Keybinding{"y"})
		t.Views().Options().Content(Contains("Paste (cherry-pick): c p"))
		t.Views().Commits().Press(keys.Universal.Return)

		// Set sticky diff state to a different base and direction before checking
		// that the direct actions continue to use their captured endpoints.
		t.Views().Commits().Press(keys.Universal.OptionMenu)
		t.ExpectPopup().Menu().
			Title(Equals("Keybindings")).
			Filter("View diffing options").
			Confirm()
		t.ExpectPopup().Menu().
			Title(Equals("Diffing")).
			Select(Contains("Enter ref")).
			Confirm()
		t.ExpectPopup().Prompt().
			Title(Equals("Enter ref:")).
			Type("HEAD~1").
			Confirm()

		t.Views().Commits().Press(keys.Universal.OptionMenu)
		t.ExpectPopup().Menu().
			Title(Equals("Keybindings")).
			Filter("View diffing options").
			Confirm()
		t.ExpectPopup().Menu().
			Title(Equals("Diffing")).
			Select(Contains("Reverse diff direction")).
			Confirm()

		t.Views().Commits().
			Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"w"})
		t.Views().Commits().IsFocused()
		assertToolInputs("historical\n", "working\n")

		// Letters are case-insensitive by design, so uppercase reaches the same action.
		t.Views().Commits().Press(config.Keybinding{"D"})
		t.GlobalPress(config.Keybinding{"W"})
		t.Views().Commits().IsFocused()
		assertToolInputs("historical\n", "working\n")

		t.Views().Commits().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"p"})
		t.Views().Commits().IsFocused()
		assertToolInputs("before\n", "historical\n")

		t.Views().Commits().Press(config.Keybinding{"<space>"})
		t.Views().CommitFiles().
			IsFocused().
			NavigateToLine(Contains("sample.txt"))
		t.Views().Options().Content(Contains("Diff with working tree: d w"))
		t.Views().CommitFiles().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"w"})
		t.Views().CommitFiles().IsFocused()
		assertToolInputs("historical\n", "working\n")

		t.FileSystem().FileContent("sample.txt", Equals("working\n"))

		t.Views().CommitFiles().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"p"})
		t.Views().CommitFiles().IsFocused()
		assertToolInputs("before\n", "historical\n")

		openWorkingFileAction()
		t.ExpectPopup().Prompt().
			Title(Contains(commitHash[:7]).Contains("sample.txt")).
			InitialText(Equals("sample.txt")).
			Cancel()
		// Cancellation leaves the previous completed tool invocation as the
		// latest record and never starts a comparison.
		assertToolInputs("before\n", "historical\n")

		openWorkingFileAction()
		t.ExpectPopup().Prompt().
			Title(Contains(commitHash[:7]).Contains("sample.txt")).
			InitialText(Equals("sample.txt")).
			Clear().
			Type("chosen file.txt").
			Confirm()

		assertToolInputs("historical\n", "chosen\n")
		if !assertSourceRemoved() {
			return
		}
		assertRepoUnchanged := func() {
			t.FileSystem().
				FileContent("sample.txt", Equals("working\n")).
				FileContent("chosen file.txt", Equals("chosen\n"))
			t.Shell().RunShellCommand("git show :sample.txt > .git/diff-index-content")
			t.FileSystem().FileContent(".git/diff-index-content", Equals("staged\n"))
			t.Git().CurrentBranchName("master")
			if actualHead := t.Git().GetCommitHash("HEAD"); actualHead != commitHash {
				t.Fail("Historical diff actions changed HEAD")
			}
		}
		assertRepoUnchanged()

		t.Shell().CreateFile(".git/fail-diff-tool", "fail")
		t.Views().CommitFiles().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"w"})
		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("128")).
			Confirm()
		assertToolInputs("historical\n", "working\n")
		if !assertSourceRemoved() {
			return
		}
		assertRepoUnchanged()

		if err := os.Remove(".git/fail-diff-tool"); err != nil {
			t.Fail("Removing historical diff tool failure flag: " + err.Error())
			return
		}
		t.Views().CommitFiles().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"w"})
		t.Views().CommitFiles().IsFocused()
		assertToolInputs("historical\n", "working\n")
		if !assertSourceRemoved() {
			return
		}
		assertRepoUnchanged()
	},
})
