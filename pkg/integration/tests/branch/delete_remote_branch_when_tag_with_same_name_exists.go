package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var DeleteRemoteBranchWhenTagWithSameNameExists = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Delete a remote branch when a remote tag with the same name exists",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.CloneIntoRemote("origin")
		shell.CreateLightweightTag("xyz", "HEAD")
		shell.PushBranch("origin", "HEAD:refs/tags/xyz") // abusing PushBranch to push a tag
		shell.PushBranch("origin", "HEAD:refs/heads/xyz")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		sourceOID := t.Git().GetCommitHash("refs/remotes/origin/xyz")
		t.Views().Remotes().
			Focus().
			Lines(
				Contains("origin").IsSelected(),
			).
			PressEnter()

		t.Views().RemoteBranches().
			IsFocused().
			Lines(
				Contains("master").IsSelected(),
				Contains("xyz"),
			).
			SelectNextItem().
			Press(keys.Universal.Remove)

		t.ExpectPopup().
			Confirmation().
			Title(Equals("Delete remote branch")).
			Content(Contains("refs/remotes/origin/xyz").Contains(sourceOID).Contains("refs/heads/xyz").Contains("Current HEAD:").Contains("Cached commit IDs may differ from the server's current tips")).
			Confirm()

		t.Views().RemoteBranches().
			EventuallyContent(Contains("master").DoesNotContain("xyz")).Lines(
			Contains("master").IsSelected(),
		)
	},
})
