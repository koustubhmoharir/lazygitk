package branch

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ShowDivergenceFromLocalUpstream = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Show selected branch divergence from its local upstream ref",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(*config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("tracked.txt", "base\n").Commit("base")
		shell.RunCommand([]string{"git", "tag", "upstream", "HEAD"})
		shell.NewBranch("feature").NewBranch("upstream").Checkout("feature").EmptyCommit("feature-only")
		shell.Checkout("upstream").EmptyCommit("upstream-only")
		shell.Checkout("master").UpdateFileAndAdd("tracked.txt", "staged\n").UpdateFile("tracked.txt", "unstaged\n").CreateFile("untracked.txt", "untracked\n")
		shell.SetConfig("branch.feature.remote", ".").SetConfig("branch.feature.merge", "refs/heads/upstream")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		featureTip := t.Git().GetCommitHash("refs/heads/feature")
		upstreamTip := t.Git().GetCommitHash("refs/heads/upstream")
		head := t.Git().GetCommitHash("HEAD")
		branches := t.Views().Branches()
		branches.Focus().NavigateToLine(Contains("feature")).Press(keys.Branches.SetUpstream)
		menu := t.ExpectPopup().Menu().Title(Equals("Upstream options"))
		t.Views().Menu().Content(Contains("feature").Contains("refs/heads/upstream"))
		menu.Select(Contains("View divergence from upstream")).Confirm()
		t.Views().SubCommits().IsFocused().Title(Contains("feature <-> upstream")).
			EventuallyContent(Contains("feature-only").Contains("upstream-only")).
			ContainsLines(Contains("↓").Contains("upstream-only")).
			ContainsLines(Contains("↑").Contains("feature-only"))
		t.Git().CurrentBranchName("master")
		if t.Git().GetCommitHash("HEAD") != head || t.Git().GetCommitHash("refs/heads/feature") != featureTip || t.Git().GetCommitHash("refs/heads/upstream") != upstreamTip {
			t.Fail("Viewing divergence changed a branch ref or HEAD")
		}
		t.FileSystem().FileContent("tracked.txt", Equals("unstaged\n")).FileContent("untracked.txt", Equals("untracked\n"))
		t.Shell().RunShellCommand("git show :tracked.txt > .git/divergence-index")
		t.FileSystem().FileContent(".git/divergence-index", Equals("staged\n"))
		t.Shell().RunCommandExpectError([]string{"git", "rev-parse", "--verify", "refs/stash"})
	},
})
