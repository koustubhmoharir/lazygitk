package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var DeleteRemoteBranchWithDifferentName = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Delete a remote branch that has a different name than the local branch",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.CloneIntoRemote("origin")
		shell.NewBranch("mybranch-local")
		shell.PushBranchAndSetUpstream("origin", "mybranch-local:mybranch-remote")
		shell.Checkout("master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("refs/heads/mybranch-local")
		t.Views().Branches().
			Focus().
			Lines(
				Contains("master").IsSelected(),
				Contains("mybranch-local ✓"),
			).
			SelectNextItem().
			Press(keys.Universal.Remove).
			Tap(func() {
				t.ExpectPopup().
					Menu().
					Title(Equals("Delete branch 'mybranch-local'?")).
					Select(MatchesRegexp(`^[[:space:]]*r Delete remote branch`)).
					Confirm()
			}).
			Tap(func() {
				t.ExpectPopup().
					Confirmation().
					Title(Equals("Delete remote branch")).
					Content(Contains("refs/heads/mybranch-local").Contains(sourceOID).Contains("refs/heads/mybranch-remote").Contains("Current HEAD:").Contains("Cached commit IDs may differ from the server's current tips")).
					Confirm()
			}).
			EventuallyContent(Contains("mybranch-local (upstream gone)")).Lines(
			Contains("master"),
			Contains("mybranch-local (upstream gone)").IsSelected(),
		)
	},
})
