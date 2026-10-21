package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
)

func TestHistoricalDiffHintPriorityDoesNotReorderDispatch(t *testing.T) {
	first := &types.Binding{CommandID: "other.first"}
	working := &types.Binding{CommandID: "diff.history-file.worktree"}
	last := &types.Binding{CommandID: "other.last"}
	bindings := []*types.Binding{first, working, last}
	hints := prioritizeHistoricalDiffHints(bindings)
	if len(hints) != 3 || hints[0] != working || hints[1] != first || hints[2] != last {
		t.Fatalf("unexpected hint order: %v", hints)
	}
	if bindings[0] != first || bindings[1] != working || bindings[2] != last {
		t.Fatal("prioritizing display hints changed dispatch order")
	}
}

func TestCherryPickModeHint(t *testing.T) {
	keys := config.GetDefaultConfig().Keybinding
	tr := i18n.EnglishTranslationSet()
	hint := cherryPickModeHint(keys, tr)
	if hint.key != keys.Universal.OptionMenu.String() || hint.description != tr.PasteCommitsPaletteHint {
		t.Fatalf("unbound paste should point to palette: %+v", hint)
	}
	keys.Commits.PasteCommits = config.Keybinding{"x"}
	hint = cherryPickModeHint(keys, tr)
	if hint.key != "x" || hint.description != tr.PasteCommits {
		t.Fatalf("configured paste should keep its shortcut: %+v", hint)
	}
}
