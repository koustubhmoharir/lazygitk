package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
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

func TestModeHintUsesEffectiveCommandOrPalettePath(t *testing.T) {
	key := func(r rune) gocui.Key { return gocui.NewKeyRune(r) }
	mode := &types.Binding{CommandID: "commit.cherry-pick", CommandPaths: [][]gocui.Key{{key('c'), key('p')}}}
	palette := &types.Binding{CommandID: "app.command-palette", CommandPaths: [][]gocui.Key{{key('?')}}}
	hint := effectiveModeHint([]*types.Binding{mode}, "commit.cherry-pick", "Paste", "Open palette", []*types.Binding{palette}, style.FgCyan)
	if hint.key != "c p" || hint.description != "Paste" {
		t.Fatalf("bound mode command should show its effective path: %+v", hint)
	}
	hint = effectiveModeHint(nil, "commit.cherry-pick", "Paste", "Open palette", []*types.Binding{palette}, style.FgCyan)
	if hint.key != "?" || hint.description != "Open palette" {
		t.Fatalf("unbound mode command should point to effective palette path: %+v", hint)
	}
	hint = effectiveModeHint(nil, "commit.cherry-pick", "Paste", "Open palette", nil, style.FgCyan)
	if hint.key != "" || appendModeHint(nil, hint) != nil {
		t.Fatalf("both commands unbound should omit the mode hint: %+v", hint)
	}
}
