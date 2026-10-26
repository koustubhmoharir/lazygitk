package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkFetchFromAnyPane = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "f fetches every remote from Branches, Commits, Status, Stash and the main diff; in Remotes it fetches only the selected remote",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the f routes come from the fork catalogue.
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
		shell.EmptyCommit("first")
		// One stash entry, so the Stash pane has content to focus.
		shell.CreateFileAndAdd("stashed.txt", "stashed\n")
		shell.RunCommand([]string{"git", "stash", "-q"})
		shell.CloneIntoRemote("origin")
		shell.CloneIntoRemote("upstream")
		shell.SetBranchUpstream("master", "origin/master")
		// dev tracks the second remote, so only fetching upstream moves it.
		shell.NewBranch("dev")
		shell.SetBranchUpstream("dev", "upstream/master")
		shell.Checkout("master")
		// Working clones push new commits to the bare remotes; the repository
		// only sees them once something fetches.
		shell.RunCommand([]string{"git", "clone", "-q", "../origin", "../origin-work"})
		shell.RunCommand([]string{"git", "clone", "-q", "../upstream", "../upstream-work"})
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		pushRemoteCommit := func(remote string, message string) {
			work := "../" + remote + "-work"
			t.Shell().RunCommand([]string{"git", "-C", work, "commit", "-q", "--allow-empty", "-m", message})
			t.Shell().RunCommand([]string{"git", "-C", work, "push", "-q", "origin", "master"})
		}

		t.Views().Branches().
			Lines(
				Contains("master").DoesNotContain("↓"),
				Contains("dev").DoesNotContain("↓"),
			)

		// Branches: f fetches origin, so master shows it is behind by one.
		pushRemoteCommit("origin", "origin 1")
		t.Views().Branches().Focus().
			Press(config.Keybinding{"f"})
		t.Views().Branches().
			Lines(
				Contains("master ↓1"),
				Contains("dev").DoesNotContain("↓"),
			)

		// Commits: f fetches upstream, so dev shows it is behind.
		pushRemoteCommit("upstream", "upstream 1")
		t.Views().Commits().Focus().
			Press(config.Keybinding{"f"})
		t.Views().Branches().
			ContainsLines(
				Contains("dev ↓1"),
			)

		// Status: f fetches origin again.
		pushRemoteCommit("origin", "origin 2")
		t.Views().Status().Focus().
			Press(config.Keybinding{"f"})
		t.Views().Branches().
			ContainsLines(
				Contains("master ↓2"),
			)

		// Stash: f fetches origin again.
		pushRemoteCommit("origin", "origin 3")
		t.Views().Stash().Focus().
			Press(config.Keybinding{"f"})
		t.Views().Branches().
			ContainsLines(
				Contains("master ↓3"),
			)

		// Focused main diff: f fetches origin again.
		pushRemoteCommit("origin", "origin 4")
		// v focuses the diff of the selected pane's item (navigation.focus-diff).
		t.Views().Status().Focus().
			Press(config.Keybinding{"v"})
		t.Views().Main().IsFocused().
			Press(config.Keybinding{"f"})
		t.Views().Branches().
			ContainsLines(
				Contains("master ↓4"),
			)

		// Remotes: f fetches only the selected remote. Both remotes get new
		// commits; selecting origin must leave upstream's tracking ref alone.
		pushRemoteCommit("origin", "origin 5")
		pushRemoteCommit("upstream", "upstream 2")
		upstreamBefore := t.Git().GetCommitHash("refs/remotes/upstream/master")
		t.Views().Remotes().Focus().
			NavigateToLine(Contains("origin")).
			Press(config.Keybinding{"f"})
		t.Views().Branches().
			Lines(
				Contains("master ↓5"),
				Contains("dev ↓1").DoesNotContain("↓2"),
			)
		if got := t.Git().GetCommitHash("refs/remotes/upstream/master"); got != upstreamBefore {
			t.Fail("fetching the selected origin remote moved upstream/master")
		}

		// Selecting upstream fetches it, and dev catches up.
		t.Views().Remotes().Focus().
			NavigateToLine(Contains("upstream")).
			Press(config.Keybinding{"f"})
		t.Views().Branches().
			ContainsLines(
				Contains("dev ↓2"),
			)
	},
})
