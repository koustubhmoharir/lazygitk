package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ForkCopyRefCommitKeys = NewIntegrationTest(NewIntegrationTestArgs{
	Description: "The flat y copy menu copies the commit that a local branch, remote branch or tag points at",
	SetupConfig: func(appConfig *config.AppConfig) {
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		// Use the shipped defaults so the y routes come from the fork catalogue.
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoFetch = false
		loaded.Git.AutoRefresh = false
		loaded.Git.AutoDetectExternalChanges = false
		loaded.Git.Log.ShowGraph = "never"
		loaded.OS.CopyToClipboardCmd = "printf '%s' {{text}} > clipboard"
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("first").
			EmptyCommit("My Subject").
			CloneIntoRemote("origin").
			NewBranch("feature").
			EmptyCommit("Feature subject").
			PushBranch("origin", "feature").
			EmptyCommit("Local only").
			Checkout("master").
			CreateAnnotatedTag("v1-annotated", "release", "HEAD").
			CreateLightweightTag("v1-light", "HEAD")
	},
	Run: func(t *TestDriver, _ config.KeybindingConfig) {
		masterTip := t.Git().GetCommitHash("master")
		localTip := t.Git().GetCommitHash("feature")
		remoteTip := t.Git().GetCommitHash("refs/remotes/origin/feature")
		tagObject := t.Git().GetCommitHash("refs/tags/v1-annotated")
		tagCommit := t.Git().GetCommitHash("refs/tags/v1-annotated^{commit}")

		// The local branch is not checked out, so its tip differs from HEAD.
		branches := t.Views().Branches().
			Focus().
			NavigateToLine(Contains("feature"))

		branches.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Content(
			Contains("Copy commit hash").
				Contains("Copy pull request URL"),
		)
		t.Views().Menu().Press(config.Keybinding{"h"})
		t.ExpectToast(Equals("'" + localTip + "' copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals(localTip))

		branches.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"s"})
		t.ExpectToast(Equals("Commit subject copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("Local only"))

		branches.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"n"})
		t.ExpectToast(Equals("'feature' copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("feature"))

		// The remote branch copies its own commit, which is behind the local branch.
		// The fork's default inspect key (v) opens the remote's branches.
		t.Views().Remotes().Focus().
			Lines(
				Contains("origin").IsSelected(),
			).
			Press(config.Keybinding{"v"})
		remoteBranches := t.Views().RemoteBranches().
			IsFocused().
			NavigateToLine(Contains("feature"))

		remoteBranches.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"h"})
		t.ExpectToast(Equals("'" + remoteTip + "' copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals(remoteTip))

		// An annotated tag resolves to the commit it peels to, not the tag object.
		tags := t.Views().Tags().
			Focus().
			NavigateToLine(Contains("v1-annotated"))

		tags.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"h"})
		t.ExpectToast(Equals("'" + tagCommit + "' copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals(tagCommit))
		if tagCommit == tagObject {
			t.Fail("The annotated tag object must differ from the commit it peels to")
		}
		if tagCommit != masterTip {
			t.Fail("Expected the annotated tag to point at " + masterTip + ", got " + tagCommit)
		}

		// Both tags sit on the same commit, so the tags action lists both.
		tags.Press(config.Keybinding{"y"})
		t.ExpectPopup().Menu().Title(Equals("y"))
		t.Views().Menu().Press(config.Keybinding{"t"})
		t.ExpectToast(Equals("Commit tags copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("v1-annotated\nv1-light"))
	},
})
