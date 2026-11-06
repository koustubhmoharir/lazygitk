package ui

import (
	"os"
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// forkInvalidRepoConfigNewDirFile is the shell hand-off file that lazygit
// writes when it switches repos. A failed switch must not write it.
var forkInvalidRepoConfigNewDirFile string

var ForkRepoSwitchInvalidRepoConfig = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "A repository that cannot be opened because its lazygitk.yml is invalid, or names an unknown language, shows the error; the current repository stays in place for the panes, the Git commands and the shell hand-off; a failed switch does not join the recent repositories",
	SetupConfig: func(cfg *config.AppConfig) {
		forkInvalidRepoConfigNewDirFile, _ = filepath.Abs("../new-dir")
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("base")
		shell.CloneNonBare("other")
		shell.CloneNonBare("lang")
		shell.EmptyCommit("repo commit")
		// Both targets are valid repositories whose repo config cannot be applied.
		shell.CreateFile("../other/.git/lazygitk.yml", "gui: [\n")
		shell.CreateFile("../lang/.git/lazygitk.yml", "gui:\n  language: xx\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		os.Setenv("LAZYGIT_NEW_DIR_FILE", forkInvalidRepoConfigNewDirFile)
		defer os.Unsetenv("LAZYGIT_NEW_DIR_FILE")

		// The path prompt, not the recent repositories, so that the target is
		// never in the recent list.
		switchByPath := func(name string) {
			t.Views().Status().Focus().Press(config.Keybinding{"s"})
			t.ExpectPopup().Menu().Title(Equals("s"))
			t.Views().Menu().Press(config.Keybinding{"n"})
			t.ExpectPopup().Prompt().Type(name)
			t.ExpectPopup().Prompt().Title(Equals("Repository path")).Confirm()
		}
		expectStayedInRepo := func() {
			// The panes show the repository we stayed in, not the one that failed.
			t.Views().Status().Content(Contains("repo → master").DoesNotContain("other").DoesNotContain("lang"))
			t.Views().Commits().Content(Contains("repo commit").DoesNotContain("other commit"))
		}

		t.Views().Status().Content(Contains("repo → master"))
		t.Views().Commits().Content(Contains("repo commit"))

		// An invalid repo config is shown as an error.
		switchByPath("other")
		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("failed to parse YAML")).
			Confirm()
		expectStayedInRepo()

		// An unknown language is shown as an error too, before anything is published.
		switchByPath("lang")
		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("Language not found: xx")).
			Confirm()
		expectStayedInRepo()

		// Neither failed switch joined the recent repositories. The menu leaves
		// out the repository we are in, so only Cancel is listed.
		t.Views().Status().Focus().Press(config.Keybinding{"s"})
		t.ExpectPopup().Menu().Title(Equals("s"))
		t.Views().Menu().Press(config.Keybinding{"r"})
		t.ExpectPopup().Menu().Title(Equals("Recent repositories")).
			Lines(
				Contains("Cancel").IsSelected(),
			).Cancel()

		// The shell hand-off directory was not written for either failure.
		t.FileSystem().PathNotPresent("../new-dir")

		// A Git command acts on the repository we stayed in.
		t.Views().Branches().Focus().Press(keys.Universal.New)
		t.ExpectPopup().Prompt().
			Title(MatchesRegexp(`^New branch name`)).
			Type("from-repo").
			Confirm()
		head := t.Git().GetCommitHash("HEAD")
		t.Common().ReviewBranchCreateMenu("refs/heads/master", head, "refs/heads/master", head).
			Select(MatchesRegexp(`^[[:space:]]*Do not set up upstream tracking`)).Confirm()
		t.Views().Status().Content(Contains("repo → from-repo"))
		t.Git().CurrentBranchName("from-repo")

		// Once its config is valid, the same repository opens, and it is then
		// recorded for the shell hand-off, so the check above is not vacuous.
		t.Shell().DeleteFile("../other/.git/lazygitk.yml")
		switchByPath("other")
		t.ExpectToast(Equals("Opened other"))
		t.Views().Status().Content(Contains("other → master"))
		t.FileSystem().FileContent("../new-dir", Contains("other"))
	},
})
