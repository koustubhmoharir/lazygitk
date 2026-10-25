package keymap

import (
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

func TestDefaultPathsUserRoutes(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"diff.commit.selected-commit", "d c"},
		{"diff.history-file.selected-commit", "d c"},
		{"branch.reset.hard", "s h"},
		{"branch.reset.mixed", "s m"},
		{"branch.reset.soft", "s s"},
		{"branch.checkout", "s c"},
		{"commit.checkout-menu", "s c"},
		{"commit.combine-selected", "g c"},
		{"commit.mark-cherry-pick", "c y"},
		{"commit.unmark-cherry-pick", "c x"},
		{"commit.cherry-pick", "c p"},
		{"file.toggle-stage", "space"},
		{"diff.stage-selection", "g h s"},
		{"diff.unstage-selection", "g h u"},
		{"branch.place-at-selected-commit", "n b"},
		{"app.refresh", "f5"},
		{"navigation.focus-diff", "v"},
		{"navigation.diff.previous-hunk", "t p h"},
		{"navigation.diff.previous-file", "t p f"},
		{"navigation.conflict.previous", "t p c"},
	}
	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			paths, ok := DefaultPaths(test.id)
			if !ok || len(paths) == 0 {
				t.Fatalf("DefaultPaths(%q) = (%v, %t), want a path", test.id, paths, ok)
			}
			if got := pathString(paths[0]); got != test.want {
				t.Errorf("path = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDefaultPathsShapeAndCanonicalization(t *testing.T) {
	for id, paths := range defaultPaths {
		for _, path := range paths {
			if len(path) > 3 {
				t.Errorf("%s has %d-key path", id, len(path))
			}
			for _, key := range path {
				if key.Mod()&gocui.ModCtrl != 0 {
					t.Errorf("%s has Ctrl path key %v", id, key)
				}
				if normalized := gocui.NormalizeCommandKey(key); !normalized.Equals(key) {
					t.Errorf("%s has noncanonical key %v", id, key)
				}
			}
		}
	}
}

func TestDefaultPathsReturnsClones(t *testing.T) {
	got, ok := DefaultPaths("branch.reset.hard")
	if !ok || len(got) != 1 {
		t.Fatalf("DefaultPaths returned (%v, %t)", got, ok)
	}
	got[0][0] = gocui.NewKeyRune('z')
	got[0] = nil
	again, _ := DefaultPaths("branch.reset.hard")
	if len(again) != 1 || pathString(again[0]) != "s h" {
		t.Fatalf("mutation escaped lookup: %v", again)
	}
}

func TestDefaultPathsUnknownAndIntentionalEmpty(t *testing.T) {
	if paths, ok := DefaultPaths("not.a.command"); ok || paths != nil {
		t.Fatalf("unknown ID returned (%v, %t), want (nil, false)", paths, ok)
	}
	for _, id := range []string{
		"commit.squash-down", "commit.mark-fixup", "commit.set-fixup-message",
		"commit.create-fixup-menu", "commit.autosquash", "commit.find-fixup-target",
		"commit.clear-cherry-pick-selection", "branch.create-and-checkout",
		"branch.reset-menu", "branch.checkout-by-name", "branch.checkout-previous",
		"branch.force-checkout", "file.toggle-stage-all", "diff.primary-selection",
		"diff.discard-selection",
		"file.refresh",
	} {
		if paths, ok := DefaultPaths(id); !ok || len(paths) != 0 {
			t.Errorf("%s returned (%v, %t), want known with no paths", id, paths, ok)
		}
	}
}

func TestCombinePathIsLeaf(t *testing.T) {
	paths, ok := DefaultPaths("commit.combine-selected")
	if !ok || len(paths) != 1 || pathString(paths[0]) != "g c" {
		t.Fatalf("combine path = %v, %t", paths, ok)
	}
	for id, candidates := range defaultPaths {
		for _, path := range candidates {
			if len(path) > 2 && path[0].Str() == "g" && path[1].Str() == "c" {
				t.Errorf("%s creates a descendant under g c: %v", id, path)
			}
		}
	}
}

func pathString(path []gocui.Key) string {
	labels := make([]string, 0, len(path))
	for _, key := range path {
		label := config.LabelForKey(key)
		labels = append(labels, strings.Trim(label, "<>"))
	}
	return strings.Join(labels, " ")
}

func TestDefaultCherryCopyToggleHasNoNormalPath(t *testing.T) {
	paths, known := DefaultPaths("commit.copy-for-cherry-pick")
	if !known || len(paths) != 0 {
		t.Fatalf("legacy toggle paths = %v, known %t", paths, known)
	}
}
