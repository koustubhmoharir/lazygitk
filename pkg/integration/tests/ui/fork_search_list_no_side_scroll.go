package ui

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkSearchListNoSideScroll = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Searching a list selects a match past the right edge without scrolling the list sideways",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Replace the entire value to clear legacy-field provenance from the shared
		// compatibility config, rather than accidentally testing explicit overrides.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.Gui.ShowRootItemInFileTree = true
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		// The match is far past the right edge of the commits panel, so a sideways
		// scroll would hide the hash at the start of the line.
		longSubject := "a subject that runs on " + strings.Repeat("x", 60) + " before the needle"
		shell.EmptyCommitWithDate("base", "2001-01-01 12:00:00 +0000").
			EmptyCommitWithDate(longSubject, "2001-01-02 12:00:00 +0000").
			EmptyCommitWithDate("top", "2001-01-03 12:00:00 +0000")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		hashAtStart := MatchesRegexp(`^[0-9a-f]{8} `)

		commits := t.Views().Commits().Focus()
		commits.Lines(
			hashAtStart,
			hashAtStart,
			hashAtStart,
		)

		commits.FilterOrSearch("needle")

		// The match is selected, but the list is not scrolled sideways to it, so the
		// selected line still starts with its hash.
		commits.OriginX(0).
			SelectedLines(Contains("needle")).
			Lines(
				hashAtStart,
				hashAtStart,
				hashAtStart,
			)

		// Escape leaves the search, and the list still starts with the hashes.
		commits.Press(keys.Universal.Return).
			OriginX(0).
			Lines(
				hashAtStart,
				hashAtStart,
				hashAtStart,
			)
	},
})
