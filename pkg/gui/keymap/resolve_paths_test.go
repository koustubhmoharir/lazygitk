package keymap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func requireKeymapNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func loadedKeymapConfig(t *testing.T, content string) *config.UserConfig {
	t.Helper()
	file := filepath.Join(t.TempDir(), "config.yml")
	requireKeymapNoError(t, os.WriteFile(file, []byte(content), 0o600))
	app := config.NewDummyAppConfig()
	requireKeymapNoError(t, app.ReloadUserConfigForRepo([]*config.ConfigFile{{Path: file, Policy: config.ConfigFilePolicyErrorIfMissing}}))
	return app.GetUserConfig()
}

func TestResolveCommandPathsExplicitPrecedenceAndDisable(t *testing.T) {
	cfg := config.GetDefaultConfig()
	paths, explicit, known, err := ResolvePaths(cfg, "diff.commit.worktree", "diff.commit.worktree", "commits")
	requireKeymapNoError(t, err)
	assert.True(t, known)
	assert.False(t, explicit)
	assert.Equal(t, "d w", pathString(paths[0]))
	requireKeymapNoError(t, yaml.Unmarshal([]byte("commandBindings: {diff.commit.worktree: [[G, W], '<ctrl+t>']}"), cfg))
	paths, explicit, known, err = ResolvePaths(cfg, "diff.commit.worktree", "diff.commit.worktree", "commits")
	requireKeymapNoError(t, err)
	assert.True(t, known && explicit)
	assert.Equal(t, "g w", pathString(paths[0]))
	assert.Equal(t, "ctrl+t", pathString(paths[1]))
	cfg.CommandBindings["diff.commit.worktree"] = nil
	paths, explicit, known, err = ResolvePaths(cfg, "diff.commit.worktree", "diff.commit.worktree", "commits")
	requireKeymapNoError(t, err)
	assert.True(t, known && explicit)
	assert.Empty(t, paths)
}

func TestLegacyFieldProvenanceUsesViewAndYAMLNames(t *testing.T) {
	cfg := config.GetDefaultConfig()
	keys, path, ok := legacyField(cfg, "Universal.Push")
	if !ok {
		t.Fatal("expected recognized legacy field")
	}
	assert.Equal(t, "universal.pushFiles", path)
	assert.Equal(t, cfg.Keybinding.Universal.Push, keys)
	assert.Equal(t, []string{"Branches.CreateTag"}, LegacyFields("tag.create-at-selection", "localBranches"))
	assert.Equal(t, []string{"Commits.CreateTag"}, LegacyFields("tag.create-at-selection", "commits"))
	assert.Contains(t, LegacyAlternates("Universal.PrevItem"), "Universal.PrevItemAlt")
	_, _, ok = legacyField(cfg, "Universal.Unknown")
	assert.False(t, ok)
}

func TestResolveLegacyOverridesIncludingSameDefaultAndExplicitDisable(t *testing.T) {
	cfg := loadedKeymapConfig(t, "keybinding:\n  universal:\n    diffWithWorkingTree: w\n    pushFiles: []\n    prevItem-alt: '<alt+k>'\n")
	paths, explicit, _, err := ResolvePaths(cfg, "diff.commit.worktree", "diff.commit.worktree", "commits")
	requireKeymapNoError(t, err)
	assert.True(t, explicit)
	if len(paths) != 1 {
		t.Fatalf("expected one path, got %v", paths)
	}
	assert.Equal(t, "w", pathString(paths[0]), "an explicit old default overrides the new menu path")
	paths, explicit, known, err := ResolvePaths(cfg, "branch.push-selected", "branch.push-current", "localBranches")
	requireKeymapNoError(t, err)
	assert.True(t, explicit && known)
	assert.Empty(t, paths, "an explicit disable must not restore the catalogue key")
	paths, explicit, _, err = ResolvePaths(cfg, "navigation.list.previous", "navigation.list.previous", "files")
	requireKeymapNoError(t, err)
	assert.True(t, explicit)
	labels := make([]string, 0, len(paths))
	for _, path := range paths {
		labels = append(labels, pathString(path))
	}
	assert.Contains(t, labels, "alt+k")
}

func TestResolveLegacySharedIDDoesNotLeakBranchOverrideIntoCommitView(t *testing.T) {
	cfg := loadedKeymapConfig(t, "keybinding:\n  branches:\n    createTag: '<f7>'\n")
	paths, explicit, _, err := ResolvePaths(cfg, "tag.create-at-selection", "tag.create-at-selection", "localBranches")
	requireKeymapNoError(t, err)
	assert.True(t, explicit)
	assert.Equal(t, "f7", pathString(paths[0]))
	paths, explicit, _, err = ResolvePaths(cfg, "tag.create-at-selection", "tag.create-at-selection", "commits")
	requireKeymapNoError(t, err)
	assert.False(t, explicit)
	assert.Equal(t, "n t", pathString(paths[0]))
}
