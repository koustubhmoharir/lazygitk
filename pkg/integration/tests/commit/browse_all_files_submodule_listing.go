package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var BrowseAllFilesSubmoduleListing = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "In the full listing of a commit's files, a submodule shows the commit it records, and the commit before it when the commit changed the submodule",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("first")
		shell.CloneIntoSubmodule("foo", "libs/foo")
		shell.Commit("add submodule")

		shell.AddFileInWorktreeOrSubmodule("libs/foo", "file", "content")
		shell.CommitInWorktreeOrSubmodule("libs/foo", "Fix the parser")
		shell.GitAdd("libs/foo")
		shell.Commit("update submodule")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Lines(
				Contains("update submodule").IsSelected(),
				Contains("add submodule"),
				Contains("first"),
			).
			Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v"))
		t.Views().Menu().Press(config.Keybinding{"f"})

		// The submodule is in the libs folder, which starts collapsed.
		t.Views().CommitFiles().
			IsFocused().
			NavigateToLine(Equals("  ▶ libs")).
			Press(keys.Universal.GoInto).
			NavigateToLine(Contains("foo"))

		// The commit changed the submodule: the commit it records, with its subject from
		// the checkout, and the commit it recorded before.
		t.Views().Main().
			Title(Equals("Submodule: libs/foo")).
			TopLines(
				MatchesRegexp(`^Commit: +[0-9a-f]{40} Fix the parser$`),
				MatchesRegexp(`^Previous: +[0-9a-f]{40} first$`),
			)

		// The commit that added the submodule had no commit before it.
		t.Views().CommitFiles().PressEscape()
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("add submodule")).
			Press(config.Keybinding{"v"})
		t.ExpectPopup().Menu().Title(Equals("v"))
		t.Views().Menu().Press(config.Keybinding{"f"})

		t.Views().CommitFiles().
			IsFocused().
			NavigateToLine(Equals("  ▶ libs")).
			Press(keys.Universal.GoInto).
			NavigateToLine(Contains("foo"))
		t.Views().Main().
			Title(Equals("Submodule: libs/foo")).
			TopLines(
				MatchesRegexp(`^Commit: +[0-9a-f]{40} first$`),
			).
			Content(DoesNotContain("Previous:"))
	},
})
