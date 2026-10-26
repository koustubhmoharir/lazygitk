package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var WorkingFileDiffDirectory = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Compare a working directory as a directory diff with HEAD, the index, or a pasted commit",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(appConfig *config.AppConfig) {
		// Shared integration config can carry legacy alternate bindings from an
		// earlier scenario; exercise the real constructor defaults for this one.
		loaded := appConfig.GetUserConfig()
		appearance := loaded.Gui
		*loaded = *config.GetDefaultConfigForPlatform(config.KeybindingPlatform())
		loaded.Gui = appearance
		loaded.DisableStartupPopups = true
		loaded.PromptToReturnFromSubprocess = false
		loaded.Git.AutoDetectExternalChanges = false
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateDir("dir")
		shell.CreateFileAndAdd("dir/a.txt", "before\n")
		shell.Commit("before")
		shell.UpdateFileAndAdd("dir/a.txt", "head\n")
		shell.Commit("head")
		// HEAD is "head", the index is "staged" and the working tree is "working".
		shell.UpdateFileAndAdd("dir/a.txt", "staged\n")
		shell.UpdateFile("dir/a.txt", "working\n")
		// Git leaves untracked files out of a directory comparison.
		shell.CreateFile("dir/untracked.txt", "untracked\n")

		// Record every regular file Git exposes on each side of the directory
		// diff, so the scenario checks the endpoints and the file set.
		shell.CreateFile(".git/fork-diff-tool", `#!/bin/sh
set -eu
git_dir=$(git rev-parse --absolute-git-dir)
{
	printf 'LEFT\n'
	(cd "$1" && find -L . -path ./.git -prune -o -type f -print | sort | while IFS= read -r f; do printf '%s\n' "$f"; cat "$f"; done)
	printf 'RIGHT\n'
	(cd "$2" && find -L . -path ./.git -prune -o -type f -print | sort | while IFS= read -r f; do printf '%s\n' "$f"; cat "$f"; done)
} > "$git_dir/diff-record"
exit 0
`)
		shell.MakeExecutable(".git/fork-diff-tool")
		shell.SetConfig("diff.tool", "fork-test")
		shell.RunCommand([]string{
			"git", "config", "--local", "difftool.fork-test.cmd",
			`"$(git rev-parse --absolute-git-dir)/fork-diff-tool" "$LOCAL" "$REMOTE"`,
		})
		shell.SetConfig("difftool.prompt", "false")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		beforeHash := t.Git().GetCommitHash("HEAD~1")

		t.Views().Files().
			Focus().
			NavigateToLine(Contains("dir"))

		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"w"})
		t.Views().Files().IsFocused()
		t.FileSystem().FileContent(".git/diff-record", Equals("LEFT\n./dir/a.txt\nhead\nRIGHT\n./dir/a.txt\nworking\n"))

		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"s"})
		t.Views().Files().IsFocused()
		t.FileSystem().FileContent(".git/diff-record", Equals("LEFT\n./dir/a.txt\nstaged\nRIGHT\n./dir/a.txt\nworking\n"))

		t.Views().Files().Press(config.Keybinding{"d"})
		t.GlobalPress(config.Keybinding{"c"})
		t.ExpectPopup().Prompt().
			Title(Equals("Compare commit hash (left) with working file (right)")).
			Type(beforeHash[:7]).
			Confirm()
		t.Views().Files().IsFocused()
		t.FileSystem().FileContent(".git/diff-record", Equals("LEFT\n./dir/a.txt\nbefore\nRIGHT\n./dir/a.txt\nworking\n"))

		t.FileSystem().FileContent("dir/a.txt", Equals("working\n"))
	},
})
