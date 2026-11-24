package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
	"github.com/jesseduffield/lazygit/pkg/integration/tests/shared"
)

var ForkDefaultFilesInspectMenu = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default Files v on a conflicted file lists resolve, merge tool and diff; on a submodule entry it lists enter and diff; elsewhere it does nothing",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the v routes come from the fork catalogue.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		// A submodule with a new commit, so the parent's Files pane lists it as modified.
		shell.EmptyCommit("first")
		shell.CloneIntoSubmodule("my_submodule_name", "my_submodule_path")
		shell.GitAddAll()
		shell.Commit("add submodule")
		shell.Chdir("my_submodule_path")
		shell.EmptyCommit("submodule change")
		shell.Chdir("..")

		// Conflicts in file1 and file2, a cleanly merged file3, and the submodule modified.
		shared.CreateMergeConflictFiles(shell)

		// The fake tool writes a resolution to $MERGED and exits 0. It is local to this repo.
		shell.RunCommand([]string{"git", "config", "--local", "merge.tool", "fake"})
		shell.RunCommand([]string{"git", "config", "--local", "mergetool.fake.cmd", `printf 'resolved by tool\n' > "$MERGED"`})
		shell.RunCommand([]string{"git", "config", "--local", "mergetool.fake.trustExitCode", "true"})
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		t.Common().PretendMergeOrRebaseStartedInLazygit()

		// a. On a conflicted file, v lists resolve, merge tool and diff, in that order.
		t.Views().Files().
			IsFocused().
			NavigateToLine(Contains("UU file1")).
			Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v")).Lines(
			Contains("Resolve conflict"),
			Contains("Open merge tool"),
			Contains("Show diff"),
			Contains("Cancel"),
		)

		// d focuses the main view, not the merge conflicts view. Escape returns to Files.
		t.Views().Menu().Press(config.Keybinding{"d"})
		t.Views().Main().IsFocused()
		t.Views().Main().PressEscape()
		t.Views().Files().IsFocused()

		// b. v r resolves here, as Enter does: the merge conflicts view opens.
		t.Views().Files().
			NavigateToLine(Contains("UU file1")).
			Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v"))
		t.Views().Menu().Press(config.Keybinding{"r"})
		t.Views().MergeConflicts().IsFocused()
		t.Views().MergeConflicts().PressEscape()
		t.Views().Files().IsFocused()

		// The merge filters Files to its conflicts. Clear the filter so that the plain
		// file and the submodule entry are listed.
		t.Views().Files().IsFocused().Press(config.Keybinding{"w"})
		t.Views().Menu().Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("Filtering")).Select(Contains("No filter")).Confirm()
		t.Views().Files().IsFocused()

		// c. On a plain file, v does nothing: no menu opens and Files keeps focus.
		t.Views().Files().
			NavigateToLine(Contains("A  file3")).
			Press(config.Keybinding{"v"})
		t.Views().Files().IsFocused()

		// d. On the submodule entry, v lists enter and diff, and v s enters the submodule.
		t.Views().Files().
			NavigateToLine(Contains("my_submodule_path")).
			Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v")).Lines(
			Contains("Enter submodule"),
			Contains("Show diff"),
			Contains("Cancel"),
		)
		t.Views().Menu().Press(config.Keybinding{"s"})
		t.ExpectToast(Equals("Opened my_submodule_path"))
		t.Views().Status().Content(Contains("my_submodule_path"))
		t.Views().Files().IsFocused().PressEscape()
		t.ExpectToast(Equals("Opened repo"))
		t.Views().Status().Content(DoesNotContain("my_submodule_path"))

		// e. v m on a conflicted file opens the merge tool for that file alone.
		t.Views().Files().
			NavigateToLine(Contains("UU file2")).
			Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v"))
		t.Views().Menu().Press(config.Keybinding{"m"})
		t.Views().Files().IsFocused()
		t.FileSystem().FileContent("file2", Equals("resolved by tool\n"))
	},
})
