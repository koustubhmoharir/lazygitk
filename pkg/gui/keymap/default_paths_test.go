package keymap

import (
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

func TestDefaultPathsUserRoutes(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"diff.commit.selected-commit", "d c"},
		{"diff.history-file.selected-commit", "d c"},
		{"diff.working-file.head", "d w"},
		{"diff.working-file.staged", "d s"},
		{"diff.working-file.selected-commit", "d c"},
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
		{"diff.stage-selection", "space"},
		{"diff.unstage-selection", "space"},
		{"branch.place-at-selected-commit", "n b"},
		{"app.refresh", "f5"},
		{"navigation.focus-diff", "v"},
		{"navigation.diff.previous-hunk", "["},
		{"navigation.diff.previous-file", "{"},
		{"navigation.conflict.previous", "["},
		{"clipboard.copy-commit-hash", "y h"},
		{"clipboard.copy-abbreviated-commit-hash", "y i"},
		{"clipboard.copy-commit-subject", "y s"},
		{"clipboard.copy-file-path", "y p"},
		{"branch.copy-pull-request-url", "y r"},
		{"clipboard.copy-confirmation", "y y"},
		{"ui.screen.next-mode", "="},
		{"ui.screen.previous-mode", "-"},
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

// "+" is the shifted "=" on common layouts, so it also cycles to the next mode.
func TestDefaultPathsScreenModeNextHasPlusAlias(t *testing.T) {
	paths, ok := DefaultPaths("ui.screen.next-mode")
	if !ok || len(paths) != 2 || pathString(paths[1]) != "+" {
		t.Fatalf("DefaultPaths(ui.screen.next-mode) = (%v, %t), want second path \"+\"", paths, ok)
	}
}

// The nested copy menus lost their y c default; they remain palette commands.
func TestDefaultPathsNestedCopyMenusHaveNoRoute(t *testing.T) {
	for _, id := range []string{"commit.copy-attribute-menu", "file.copy-menu", "history-file.copy-menu"} {
		if paths, ok := DefaultPaths(id); !ok || len(paths) != 0 {
			t.Errorf("%s returned (%v, %t), want known with no paths", id, paths, ok)
		}
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
		"branch.reset-menu",
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

// Palette-only commands have no default key. Being unknown here makes the
// projection give them no keyboard path.
func TestDefaultPathsPaletteOnlyCommandsHaveNoPath(t *testing.T) {
	if paths, ok := DefaultPaths("app.reset-command-usage"); ok || paths != nil {
		t.Fatalf("palette-only command returned (%v, %t), want (nil, false)", paths, ok)
	}
}

// Fork: roadmap item 14. Repository switching is under s, and no other default
// route uses s r or s n.
func TestDefaultPathsRepositorySwitchingRoutes(t *testing.T) {
	for id, want := range map[string]string{"repo.switch-menu": "s r", "repo.open-path": "s n"} {
		paths, ok := DefaultPaths(id)
		if !ok || len(paths) != 1 || pathString(paths[0]) != want {
			t.Errorf("DefaultPaths(%q) = (%v, %t), want exactly %q", id, paths, ok, want)
		}
	}
	for id, paths := range defaultPaths {
		if id == "repo.switch-menu" || id == "repo.open-path" {
			continue
		}
		for _, path := range paths {
			if got := pathString(path); got == "s r" || got == "s n" {
				t.Errorf("%s also uses %q", id, got)
			}
		}
	}
	for _, id := range []string{"repo.switch-menu", "repo.open-path"} {
		if paths, _ := DefaultPaths(id); len(paths) > 0 && !strings.HasPrefix(pathString(paths[0]), "s ") {
			t.Errorf("%s is not under s", id)
		}
	}
}

// Fork: roadmap item 18. Worktree switching moves to s w, and submodule entry
// is inspection (v and Space), so no default route remains under t r.
func TestDefaultPathsSubmoduleEnterAndWorktreeSwitchRoutes(t *testing.T) {
	for id, want := range map[string]string{"worktree.switch": "s w", "submodule.enter": "v"} {
		paths, ok := DefaultPaths(id)
		if !ok || len(paths) == 0 || pathString(paths[0]) != want {
			t.Errorf("DefaultPaths(%q) = (%v, %t), want first path %q", id, paths, ok, want)
		}
	}
	paths, _ := DefaultPaths("submodule.enter")
	got := make([]string, 0, len(paths))
	for _, path := range paths {
		got = append(got, pathString(path))
	}
	assert.Equal(t, []string{"v", "space"}, got)
	for id, paths := range defaultPaths {
		for _, path := range paths {
			if label := pathString(path); label == "t r" || strings.HasPrefix(label, "t r ") {
				t.Errorf("%s still uses %q", id, label)
			}
		}
	}
}

// Fork: the global s prefix merges with the local s h/s m/s s/s c routes.
func TestBuildCommandTreeMergesRepositorySwitchingWithLocalSPrefix(t *testing.T) {
	global := &types.Binding{CommandPaths: [][]gocui.Key{{key('s'), key('r')}, {key('s'), key('n')}}}
	local := &types.Binding{ViewName: "branches", CommandPaths: [][]gocui.Key{{key('s'), key('h')}, {key('s'), key('c')}}}
	root := BuildCommandTree([]*types.Binding{global, local}, "branches")

	s := childTree(t, root, key('s'))
	assert.Equal(t, []*types.Binding{global}, childTree(t, s, key('r')).Bindings)
	assert.Equal(t, []*types.Binding{global}, childTree(t, s, key('n')).Bindings)
	assert.Equal(t, []*types.Binding{local}, childTree(t, s, key('h')).Bindings)
	assert.Equal(t, []*types.Binding{local}, childTree(t, s, key('c')).Bindings)
}
