package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullRebase = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Pull with a rebase strategy",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "content1")
		shell.Commit("one")
		shell.UpdateFileAndAdd("file", "content2")
		shell.Commit("two")
		shell.CreateFileAndAdd("file3", "content3")
		shell.Commit("three")

		shell.CloneIntoRemote("origin")

		shell.SetBranchUpstream("master", "origin/master")

		shell.HardReset("HEAD^^")
		shell.CreateFileAndAdd("file4", "content4")
		shell.Commit("four")

		shell.SetConfig("pull.rebase", "true")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Lines(
				Contains("four"),
				Contains("one"),
			)

		t.Views().Status().Content(Equals("↓2↑1 repo → master"))

		targetOID := reviewCurrentPull(t, keys)

		t.Views().Commits().
			EventuallyContent(Contains("three")).
			Lines(
				Contains("four"),
				Contains("three"),
				Contains("two"),
				Contains("one"),
			)
		t.Views().Status().EventuallyContent(Equals("↑1 repo → master"))
		t.Shell().RunShellCommand("git merge-base --is-ancestor " + targetOID + " HEAD")
	},
})
