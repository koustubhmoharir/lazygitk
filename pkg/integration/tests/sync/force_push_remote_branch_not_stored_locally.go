package sync

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForcePushRemoteBranchNotStoredLocally = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Force push without a remote-tracking cache using an explicitly reviewed local lease reference",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
		shell.EmptyCommit("two")

		shell.Clone("some-remote")
		// A known local reference supplies the reviewed remote tip; merely adding
		// a named remote must not silently fetch a force-push lease.
		shell.RunCommand([]string{"git", "remote", "add", "some-remote", "../some-remote"})
		shell.RunCommand([]string{"git", "update-ref", "refs/reviewed/remote-master", "HEAD"})

		// remove the 'two' commit so that we have something to pull from the remote
		shell.HardReset("HEAD^")

		shell.SetConfig("branch.master.remote", "../some-remote")
		shell.SetConfig("branch.master.pushRemote", "some-remote")
		shell.SetConfig("branch.master.merge", "refs/heads/master")

		shell.CreateFileAndAdd("file1", "file1 content")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("HEAD")
		leaseOID := t.Git().GetCommitHash("refs/reviewed/remote-master")
		t.Shell().RunShellCommand("test -z \"$(git for-each-ref refs/remotes/some-remote)\"")
		t.Shell().RunShellCommand("cp .git/config .git/force-explicit-lease-config")
		t.Views().Commits().
			Lines(
				Contains("one"),
			)

		t.Views().Status().Content(Contains("? repo → master"))

		// Choose force explicitly. With no tracking cache, Git must not contact
		// the remote to invent a lease: the user chooses a known local ref.
		t.Views().Files().IsFocused().Press(keys.Universal.Push)
		t.ExpectPopup().Menu().Title(Equals("Push selected branch")).
			Select(Contains("Force with reviewed lease")).Confirm()
		t.ExpectPopup().Prompt().Title(Equals("Full locally available lease reference")).
			Type("refs/reviewed/remote-master").Confirm()
		forceReview := t.ExpectPopup().Menu().Title(Equals("Confirm force with lease"))
		t.Views().Menu().Content(Contains(sourceOID).Contains("refs/heads/master").
			Contains("refs/reviewed/remote-master").Contains(leaseOID))
		forceReview.ContainsLines(Contains("Cancel").IsSelected())
		t.Shell().RunShellCommand("test \"$(git --git-dir=../some-remote rev-parse master)\" = \"$(git rev-parse refs/reviewed/remote-master)\"")
		forceReview.Select(Contains("Push with reviewed lease")).Confirm()
		t.Views().Remotes().Focus().NavigateToLine(Contains("some-remote")).PressEnter()
		t.Views().RemoteBranches().IsFocused().Lines(Contains("master")).PressEnter()
		t.Views().SubCommits().IsFocused().Lines(Contains("one"))
		t.Views().Files().Focus().Lines(Contains("file1"))
		t.Shell().RunShellCommand("test \"$(git --git-dir=../some-remote rev-parse master)\" = \"$(git rev-parse HEAD)\"")
		t.Shell().RunShellCommand("cmp .git/config .git/force-explicit-lease-config")

		// Make a new local commit
		t.Views().Files().IsFocused().Press(keys.Files.CommitChanges)
		t.ExpectPopup().CommitMessagePanel().Type("new").Confirm()

		t.Views().Commits().
			Lines(
				Contains("new"),
				Contains("one"),
			)

		// Pushing this works without needing to force push
		t.Views().Files().IsFocused().Press(keys.Universal.Push)
		t.ExpectPopup().Menu().Title(Equals("Push selected branch")).Select(Contains("Push this branch")).Confirm()
		t.Views().Remotes().Focus().NavigateToLine(Contains("some-remote")).PressEnter()

		t.Views().RemoteBranches().IsFocused().Lines(
			Contains("master").IsSelected(),
		).
			PressEnter()

		t.Views().SubCommits().IsFocused().Lines(
			Contains("new"),
			Contains("one"),
		)
		t.Shell().RunShellCommand("cmp .git/config .git/force-explicit-lease-config")
	},
})
