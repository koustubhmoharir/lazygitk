package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FastForwardCheckedOutIgnoredFileInTheWay = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Ask before fast-forwarding the checked out branch over an ignored file, and move it only after approval",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd(".gitignore", "ignored-incoming\n")
		shell.Commit("one")
		shell.CreateFile("ignored-incoming", "committed")
		shell.RunCommand([]string{"git", "add", "-f", "ignored-incoming"})
		shell.Commit("two")

		shell.CloneIntoRemote("origin")
		shell.SetBranchUpstream("master", "origin/master")

		// remove the commit that adds the file so that it is incoming
		shell.HardReset("HEAD~1")

		// an ignored file at the path the incoming commit adds
		shell.CreateFile("ignored-incoming", "ignored user bytes")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		headOID := t.Git().GetCommitHash("HEAD")
		targetOID := t.Git().GetCommitHash("refs/remotes/origin/master")

		t.Views().Branches().
			Focus().
			Lines(
				Contains("master ↓1").IsSelected(),
			).
			Press(keys.Branches.FastForward)

		// The ignored-file risk is shown first, with Cancel selected by default.
		risk := t.ExpectPopup().Menu().Title(Equals("Ignored files may be overwritten"))
		t.Views().Menu().SelectedLine(Contains("Cancel")).
			Content(Contains("Approve ignored-file overwrite risk and integrate").Contains("\"ignored-incoming\""))
		risk.Confirm()

		t.Views().Branches().
			Lines(
				Contains("master ↓1").IsSelected(),
			)
		if got := t.Git().GetCommitHash("HEAD"); got != headOID {
			t.Fail("cancelled ignored-file risk changed HEAD: " + got)
		}
		t.FileSystem().FileContent("ignored-incoming", Equals("ignored user bytes"))

		t.Views().Branches().Press(keys.Branches.FastForward)
		t.ExpectPopup().Menu().Title(Equals("Ignored files may be overwritten")).
			Select(Contains("Approve ignored-file overwrite risk and integrate")).
			Confirm()

		t.Views().Branches().
			Lines(
				Contains("master ✓").IsSelected(),
			)
		if got := t.Git().GetCommitHash("HEAD"); got != targetOID {
			t.Fail("approved fast-forward did not move HEAD to the upstream: " + got)
		}
		t.FileSystem().FileContent("ignored-incoming", Equals("committed"))
	},
})
