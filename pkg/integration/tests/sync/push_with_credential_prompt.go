package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushWithCredentialPrompt = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Push a commit to a pre-configured upstream, where credentials are required",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")

		shell.CloneIntoRemote("origin")

		shell.SetBranchUpstream("master", "origin/master")

		shell.EmptyCommit("two")

		// actually getting a password prompt is tricky: it requires SSH'ing into localhost under a newly created, restricted, user.
		// This is not easy to do in a cross-platform way, nor is it easy to do in a docker container.
		// If you can think of a way to do it, please let me know!
		shell.CopyHelpFile("pre-push", ".git/hooks/pre-push")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("refs/heads/master")
		t.Shell().RunShellCommand("cp .git/config .git/credential-push-config-before && git rev-parse HEAD > .git/credential-push-head-before && git write-tree > .git/credential-push-index-before && git diff --binary HEAD > .git/credential-push-worktree-before && git status --porcelain --untracked-files=all > .git/credential-push-status-before")
		t.Views().Status().Content(Equals("↑1 repo → master"))

		t.Views().Files().
			IsFocused().
			Press(keys.Universal.Push)
		t.ReviewBranchPush("master", "origin", "master", sourceOID, false).Confirm()

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
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/credential-push-head-after-bad && git write-tree > .git/credential-push-index-after-bad && git diff --binary HEAD > .git/credential-push-worktree-after-bad && git status --porcelain --untracked-files=all > .git/credential-push-status-after-bad && cmp .git/credential-push-config-before .git/config && cmp .git/credential-push-head-before .git/credential-push-head-after-bad && cmp .git/credential-push-index-before .git/credential-push-index-after-bad && cmp .git/credential-push-worktree-before .git/credential-push-worktree-after-bad && cmp .git/credential-push-status-before .git/credential-push-status-after-bad")

		t.Views().Status().Content(Equals("↑1 repo → master"))

		// try again with correct password
		t.Views().Files().
			IsFocused().
			Press(keys.Universal.Push)
		t.ReviewBranchPush("master", "origin", "master", sourceOID, false).Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Username")).
			Type("username").
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Password")).
			Type("password").
			Confirm()

		t.Views().Status().EventuallyContent(Equals("✓ repo → master"))
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/credential-push-head-after-good && git write-tree > .git/credential-push-index-after-good && git diff --binary HEAD > .git/credential-push-worktree-after-good && git status --porcelain --untracked-files=all > .git/credential-push-status-after-good && cmp .git/credential-push-config-before .git/config && cmp .git/credential-push-head-before .git/credential-push-head-after-good && cmp .git/credential-push-index-before .git/credential-push-index-after-good && cmp .git/credential-push-worktree-before .git/credential-push-worktree-after-good && cmp .git/credential-push-status-before .git/credential-push-status-after-good")

		assertSuccessfullyPushed(t)
	},
})
