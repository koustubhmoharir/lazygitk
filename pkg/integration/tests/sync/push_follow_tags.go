package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PushFollowTags = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Push with --follow-tags configured in git config",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")

		shell.CloneIntoRemote("origin")

		shell.SetBranchUpstream("master", "origin/master")

		shell.EmptyCommit("two")
		shell.CreateAnnotatedTag("mytag", "message", "HEAD")

		shell.SetConfig("push.followTags", "true")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Shell().RunShellCommand("cp .git/config .git/follow-tags-config-before && git rev-parse HEAD > .git/follow-tags-head-before && git write-tree > .git/follow-tags-index-before && git diff --binary HEAD > .git/follow-tags-worktree-before && git status --porcelain --untracked-files=all > .git/follow-tags-status-before")
		t.Views().Status().Content(Equals("↑1 repo → master"))

		t.Views().Files().
			IsFocused().
			Press(keys.Universal.Push)

		t.ExpectPopup().Menu().Title(Equals("Push selected branch")).Select(Contains("Push this branch")).Confirm()
		t.Views().Status().EventuallyContent(Equals("✓ repo → master"))
		t.Shell().RunShellCommand("cp .git/config .git/follow-tags-config-after && git rev-parse HEAD > .git/follow-tags-head-after && git write-tree > .git/follow-tags-index-after && git diff --binary HEAD > .git/follow-tags-worktree-after && git status --porcelain --untracked-files=all > .git/follow-tags-status-after && cmp .git/follow-tags-config-before .git/follow-tags-config-after && cmp .git/follow-tags-head-before .git/follow-tags-head-after && cmp .git/follow-tags-index-before .git/follow-tags-index-after && cmp .git/follow-tags-worktree-before .git/follow-tags-worktree-after && cmp .git/follow-tags-status-before .git/follow-tags-status-after")
		t.Git().RemoteTagNames("origin", []string{})
		t.Git().TagNamesAt("HEAD", []string{"mytag"})

		t.Views().Remotes().
			Focus().
			Lines(
				Contains("origin"),
			).
			PressEnter()

		t.Views().RemoteBranches().
			IsFocused().
			Lines(
				Contains("master"),
			).
			PressEnter()

		t.Views().SubCommits().
			IsFocused().
			Lines(
				Contains("two").Contains("mytag"),
				Contains("one"),
			)
	},
})
