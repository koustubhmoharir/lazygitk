package filter_and_search

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var UnboundActionFromPalette = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Running an unbound branch action from the keybindings palette",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Keybinding.Branches.RenameBranch = nil
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("commit")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Branches().
			Focus().
			Lines(
				Contains("master"),
			).
			Press(keys.Universal.OptionMenu).
			Tap(func() {
				t.ExpectPopup().Menu().
					Title(Equals("Keybindings")).
					Filter("Rename branch").
					Lines(
						Equals("─── Local"),
						Contains("Rename branch").IsSelected(),
					).
					Confirm()

				t.ExpectPopup().Prompt().
					Title(Contains("Enter new branch name")).
					InitialText(Equals("master")).
					Clear().
					Type("renamed-from-palette").
					Confirm()

				t.ExpectPopup().Confirmation().
					Title(Equals("Rename branch")).
					Content(Contains("renamed-from-palette")).
					Confirm()
			}).
			Lines(
				Contains("renamed-from-palette"),
			)

		t.Git().CurrentBranchName("renamed-from-palette")
	},
})
