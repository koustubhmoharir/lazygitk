package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PlaceAtAnnotatedTag = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default n b from Tags creates a branch at the commit an annotated tag peels to, not at the tag object",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, rather than accidentally testing explicit overrides.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one").
			EmptyCommit("two").
			CreateAnnotatedTag("v1", "release", "HEAD").
			EmptyCommit("three")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		twoOID := t.Git().GetCommitHash("HEAD~1")
		threeOID := t.Git().GetCommitHash("HEAD")
		if tagOID := t.Git().GetCommitHash("refs/tags/v1"); tagOID == twoOID {
			t.Fail("Expected v1 to be an annotated tag object, not a lightweight tag")
		}

		tags := t.Views().Tags()
		tags.Focus().NavigateToLine(Contains("v1"))
		tags.Press(config.Keybinding{"n"})
		t.ExpectPopup().Menu().Title(Equals("n"))
		t.Views().Menu().Press(config.Keybinding{"b"})
		t.ExpectPopup().Prompt().
			Title(Contains("Local branch name at").Contains("(without checkout)")).
			Type("from-tag").
			Confirm()

		tags.IsFocused()
		if got := t.Git().GetCommitHash("refs/heads/from-tag"); got != twoOID {
			t.Fail("Expected refs/heads/from-tag at the tagged commit " + twoOID + ", got " + got)
		}
		if got := t.Git().GetCommitHash("HEAD"); got != threeOID {
			t.Fail("Branch placement changed HEAD to " + got)
		}
		t.Git().CurrentBranchName("master")
	},
})
