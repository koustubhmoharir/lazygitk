package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var PullRebaseInteractiveConflict = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Pull with an interactive rebase strategy, where a conflict occurs",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.AutoStageResolvedConflicts = true
	},
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
		shell.UpdateFileAndAdd("file", "content4")
		shell.Commit("four")
		shell.CreateFileAndAdd("file5", "content5")
		shell.Commit("five")

		shell.SetConfig("pull.rebase", "interactive")
		// Model accepting the initial native sequence-editor todo list; this
		// fixture exercises editing/dropping todos in lazygit after a conflict.
		shell.SetConfig("sequence.editor", "true")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Lines(
				Contains("five"),
				Contains("four"),
				Contains("one"),
			)

		t.Views().Status().Content(Equals("↓2↑2 repo → master"))

		targetOID := reviewCurrentPull(t, keys)

		t.Common().AcknowledgeConflicts()

		t.Views().Commits().
			EventuallyContent(Contains("─── Pending rebase todos")).
			Lines(
				Contains("─── Pending rebase todos"),
				Contains("pick").Contains("five"),
				Contains("pick").Contains("CONFLICT").Contains("four").IsSelected(),
				Contains("─── Commits"),
				Contains("three"),
				Contains("two"),
				Contains("one"),
			)

		t.Views().Files().
			IsFocused().
			Lines(
				Contains("UU").Contains("file"),
			).
			PressEnter()

		t.Views().MergeConflicts().
			IsFocused().
			TopLines(
				Contains("<<<<<<< HEAD"),
				Contains("content2"),
				Contains("======="),
				Contains("content4"),
				Contains(">>>>>>>"),
			).
			SelectNextItem().
			PressPrimaryAction() // choose 'content4'

		t.Common().ContinueOnConflictsResolved("rebase")

		t.Views().Commits().
			EventuallyContent(Contains("four")).
			Focus().
			Lines(
				Contains("five"),
				Contains("four").IsSelected(),
				Contains("three"),
				Contains("two"),
				Contains("one"),
			)
		t.Views().Status().EventuallyContent(Equals("↑2 repo → master"))
		t.Shell().RunShellCommand("git merge-base --is-ancestor " + targetOID + " HEAD")

		t.Views().Main().
			Content(
				Contains("-content2").
					Contains("+content4"),
			)
	},
})
