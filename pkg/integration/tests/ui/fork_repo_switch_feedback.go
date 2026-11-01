package ui

import (
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkRepoSwitchFeedback = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Switching repos toasts the name of the repo once it has loaded, and after a failed switch the current repo is shown again",
	SetupConfig: func(cfg *config.AppConfig) {
		other, _ := filepath.Abs("../other")
		gone, _ := filepath.Abs("../gone")
		cfg.GetAppState().RecentRepos = []string{other, gone}
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("base")
		shell.CloneNonBare("other")
		shell.CloneNonBare("gone")
		shell.EmptyCommit("repo commit")
		shell.Chdir("../other")
		shell.EmptyCommit("other commit")
		shell.Chdir("../repo")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		switchToRepo := func(repo string) {
			t.GlobalPress(keys.Universal.OpenRecentRepos)
			t.ExpectPopup().Menu().Title(Equals("Recent repositories")).
				Select(MatchesRegexp("^" + repo + " ")).
				Confirm()
		}

		t.Views().Commits().Content(Contains("repo commit").DoesNotContain("other commit"))

		// The other repo has never been opened in this session: its panes show
		// its own commits, and the toast names it once it has loaded.
		switchToRepo("other")
		t.ExpectToast(Equals("Opened other"))
		t.Views().Status().Content(Contains("other → master"))
		t.Views().Commits().Content(Contains("other commit").DoesNotContain("repo commit"))

		// Going back to a visited repo gets the same feedback, with its own panes.
		switchToRepo("repo")
		t.ExpectToast(Equals("Opened repo"))
		t.Views().Status().Content(Contains("repo → master"))
		t.Views().Commits().Content(Contains("repo commit").DoesNotContain("other commit"))

		// A recent repo whose directory has been deleted cannot be opened. The
		// error alert is shown, and the current repo's panes are rendered again.
		t.Shell().DeleteFile("../gone")
		switchToRepo("gone")
		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("Cannot find repo")).
			Confirm()
		t.Views().Status().Content(Contains("repo → master"))
		t.Views().Commits().Content(Contains("repo commit").DoesNotContain("other commit"))
	},
})
