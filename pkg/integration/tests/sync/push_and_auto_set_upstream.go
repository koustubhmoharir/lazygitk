package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushAndAutoSetUpstream = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "push.autoSetupRemote does not silently save upstream tracking",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")

		shell.CloneIntoRemote("origin")

		shell.EmptyCommit("two")

		shell.SetConfig("push.default", "current")
		shell.SetConfig("push.autoSetupRemote", "true")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("refs/heads/master")
		t.Shell().RunShellCommand("cp .git/config .git/push-auto-setup-config-before && git rev-parse HEAD > .git/push-auto-setup-head-before && git write-tree > .git/push-auto-setup-index-before && git diff --binary HEAD > .git/push-auto-setup-worktree-before && git status --porcelain --untracked-files=all > .git/push-auto-setup-status-before")
		t.Views().Status().Content(Equals("repo → master"))

		t.Views().Files().
			IsFocused().
			Press(keys.Universal.Push)

		t.ExpectPopup().Menu().Title(Equals("Push selected branch")).
			Select(MatchesRegexp("^Choose remote")).Confirm()
		t.ExpectPopup().Prompt().Title(Equals("Remote name")).Type("origin").Confirm()
		// Push completion refreshes the remote branch history; wait for that
		// visible result before checking the bare remote and saved config.
		t.Views().Remotes().Focus().Lines(Contains("origin")).PressEnter()
		t.Views().RemoteBranches().IsFocused().Lines(Contains("master")).PressEnter()
		t.Views().SubCommits().IsFocused().EventuallyContent(Contains("two").Contains("one"))
		t.Views().SubCommits().Lines(Contains("two"), Contains("one"))

		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/master > .git/assert-push-auto-setup-remote")
		t.FileSystem().FileContent(".git/assert-push-auto-setup-remote", Equals(sourceOID+"\n"))
		t.Shell().RunShellCommand("git config --get-regexp '^branch\\.master\\.(remote|merge)$' > .git/assert-push-auto-setup-tracking 2>/dev/null || true")
		t.FileSystem().FileContent(".git/assert-push-auto-setup-tracking", Equals(""))
		t.Shell().RunShellCommand("git config --get push.autoSetupRemote > .git/assert-push-auto-setup-config")
		t.FileSystem().FileContent(".git/assert-push-auto-setup-config", Equals("true\n"))
		t.Shell().RunShellCommand("cp .git/config .git/push-auto-setup-config-after && git rev-parse HEAD > .git/push-auto-setup-head-after && git write-tree > .git/push-auto-setup-index-after && git diff --binary HEAD > .git/push-auto-setup-worktree-after && git status --porcelain --untracked-files=all > .git/push-auto-setup-status-after && cmp .git/push-auto-setup-config-before .git/push-auto-setup-config-after && cmp .git/push-auto-setup-head-before .git/push-auto-setup-head-after && cmp .git/push-auto-setup-index-before .git/push-auto-setup-index-after && cmp .git/push-auto-setup-worktree-before .git/push-auto-setup-worktree-after && cmp .git/push-auto-setup-status-before .git/push-auto-setup-status-after")
	},
})
