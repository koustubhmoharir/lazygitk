package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var DeleteRemoteBranchWithCredentialPrompt = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Delete a remote branch where credentials are required",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")

		shell.CloneIntoRemote("origin")

		shell.NewBranch("mybranch")

		shell.PushBranchAndSetUpstream("origin", "mybranch")

		// actually getting a password prompt is tricky: it requires SSH'ing into localhost under a newly created, restricted, user.
		// This is not easy to do in a cross-platform way, nor is it easy to do in a docker container.
		// If you can think of a way to do it, please let me know!
		shell.CopyHelpFile("pre-push", ".git/hooks/pre-push")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("refs/heads/mybranch")
		deleteBranch := func() {
			t.Views().Branches().
				Focus().
				Press(keys.Universal.Remove)

			t.ExpectPopup().
				Menu().
				Title(Equals("Delete branch 'mybranch'?")).
				Select(MatchesRegexp(`^[[:space:]]*r Delete remote branch`)).
				Confirm()

			t.ExpectPopup().
				Confirmation().
				Title(Equals("Delete remote branch")).
				Content(Contains("refs/heads/mybranch").Contains(sourceOID).Contains("Current HEAD:").Contains("Cached commit IDs may differ from the server's current tips")).
				Confirm()
		}

		t.Views().Status().Content(Equals("✓ repo → mybranch"))

		deleteBranch()

		// correct credentials are: username=username, password=password

		t.ExpectPopup().Prompt().
			Title(Equals("Username")).
			Type("username").
			Confirm()

		// enter incorrect password
		t.ExpectPopup().Prompt().
			Title(Equals("Password")).
			Type("incorrect password").
			Confirm()

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("incorrect username/password")).
			Confirm()

		t.Views().Status().Content(Equals("✓ repo → mybranch"))

		// try again with correct password
		deleteBranch()

		t.ExpectPopup().Prompt().
			Title(Equals("Username")).
			Type("username").
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Password")).
			Type("password").
			Confirm()

		t.Views().Status().Content(Equals("(upstream gone) repo → mybranch"))
		t.Views().Branches().EventuallyContent(Contains("mybranch (upstream gone)")).TopLines(Contains("mybranch (upstream gone)"))
	},
})
