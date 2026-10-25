package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkDefaultRefresh = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "Real default F5 refreshes repository state and re-renders panes without fetching",
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
		// Background fetch and refresh stay off so that only the explicit F5
		// press can change what the panes show.
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.Log.ShowGraph = "never"
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("base").
			CloneIntoRemote("origin").
			SetBranchUpstream("master", "origin/master")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		remoteBefore := t.Git().GetCommitHash("refs/remotes/origin/master")

		files := t.Views().Files()
		files.Focus().IsEmpty()

		// Change external state without going through lazygit: a new untracked
		// file, a new local branch, and a commit on the remote that is not fetched.
		// The remote is bare, so the commit is pushed by path, which does not
		// update the local refs/remotes/origin/master tracking ref.
		t.Shell().
			CreateFile("external.txt", "external\n").
			RunCommand([]string{"git", "branch", "ext-branch"}).
			RunShellCommand(`git push ../origin "$(git commit-tree HEAD^{tree} -p HEAD -m remote-only)":refs/heads/master`)

		// F5 refreshes everything without fetching; focus must not move and no
		// popup may open.
		files.Press(config.Keybinding{"<f5>"}).IsFocused()
		t.Views().Menu().IsInvisible()
		t.Views().Confirmation().IsInvisible()
		t.Views().Prompt().IsInvisible()

		files.EventuallyContent(Contains("external.txt"))
		t.Views().Branches().EventuallyContent(Contains("ext-branch"))
		files.IsFocused()

		t.Views().Branches().Focus().ContainsLines(Contains("ext-branch"))
		t.Views().Menu().IsInvisible()

		if got := t.Git().GetCommitHash("refs/remotes/origin/master"); got != remoteBefore {
			t.Fail("Expected refs/remotes/origin/master to stay at " + remoteBefore + " without fetch, got " + got)
		}

		// F5 from Commits also refreshes and keeps focus in place.
		commits := t.Views().Commits()
		commits.Focus().Press(config.Keybinding{"<f5>"}).IsFocused()
		t.Views().Menu().IsInvisible()
		t.Views().Confirmation().IsInvisible()
		t.Views().Prompt().IsInvisible()
		commits.ContainsLines(Contains("base"))
	},
})
