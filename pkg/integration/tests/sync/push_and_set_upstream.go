package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushAndSetUpstream = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Push a commit and set the upstream via a prompt",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")

		shell.CloneIntoRemote("origin")
		shell.RunCommand([]string{"git", "config", "branch.master.rebase", "false"})

		shell.EmptyCommit("two")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("refs/heads/master")
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/push-upstream-head-before && git write-tree > .git/push-upstream-index-before && git diff --binary HEAD > .git/push-upstream-worktree-before && git status --porcelain --untracked-files=all > .git/push-upstream-status-before")
		t.Views().Status().Content(Equals("repo → master"))

		t.Views().Files().
			IsFocused().
			Press(keys.Universal.Push)

		t.ExpectPopup().Menu().Title(Equals("Push selected branch")).
			Select(Contains("Toggle set upstream for selected local branch")).Confirm()
		t.ExpectPopup().Menu().Title(Equals("Push selected branch")).
			Select(MatchesRegexp("^Choose remote")).Confirm()
		t.ExpectPopup().Prompt().Title(Equals("Remote name")).Type("origin").Confirm()

		t.ExpectPopup().Confirmation().Title(Equals("Review branch push")).
			Content(Contains(`source "master"`).Contains(sourceOID).
				Contains(`remote "origin"`).Contains(`destination "refs/heads/master"`).
				Contains("Set upstream for selected local branch: true")).Confirm()

		t.Views().Status().EventuallyContent(Contains("✓ repo → master"))
		t.Shell().RunShellCommand("git --git-dir=../origin rev-parse --verify refs/heads/master > .git/assert-push-upstream-remote")
		t.FileSystem().FileContent(".git/assert-push-upstream-remote", Equals(sourceOID+"\n"))
		t.Shell().RunShellCommand("git rev-parse HEAD > .git/push-upstream-head-after && git write-tree > .git/push-upstream-index-after && git diff --binary HEAD > .git/push-upstream-worktree-after && git status --porcelain --untracked-files=all > .git/push-upstream-status-after")
		t.Shell().RunShellCommand("cmp .git/push-upstream-head-before .git/push-upstream-head-after && cmp .git/push-upstream-index-before .git/push-upstream-index-after && cmp .git/push-upstream-worktree-before .git/push-upstream-worktree-after && cmp .git/push-upstream-status-before .git/push-upstream-status-after")
		if got := t.Git().GetCommitHash("HEAD"); got != sourceOID {
			t.Fail("The checked-out source changed during the reviewed push")
		}
		t.Shell().RunShellCommand("git config --get branch.master.remote > .git/assert-push-upstream-config-remote")
		t.FileSystem().FileContent(".git/assert-push-upstream-config-remote", Equals("origin\n"))
		t.Shell().RunShellCommand("git config --get branch.master.merge > .git/assert-push-upstream-merge")
		t.FileSystem().FileContent(".git/assert-push-upstream-merge", Equals("refs/heads/master\n"))
		t.Shell().RunShellCommand("git config --get branch.master.rebase > .git/assert-push-upstream-rebase")
		t.FileSystem().FileContent(".git/assert-push-upstream-rebase", Equals("false\n"))
	},
})
