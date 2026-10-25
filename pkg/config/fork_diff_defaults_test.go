package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForkDiffDefaultsForPlatform(t *testing.T) {
	for _, platform := range []string{"", "linux", "darwin", "windows"} {
		t.Run(platform, func(t *testing.T) {
			config := GetDefaultConfigForPlatform(platform)
			universal := config.Keybinding.Universal

			assert.Equal(t, Keybinding{"w", "W"}, universal.DiffWithWorkingTree)
			assert.Equal(t, Keybinding{"v", "V"}, universal.DiffWithPreviousCommit)
			assert.Empty(t, universal.DiffWithWorkingFile)
			for _, keys := range []Keybinding{universal.DiffWithWorkingTree, universal.DiffWithPreviousCommit} {
				for _, key := range keys {
					assert.NotContains(t, key, "<ctrl+")
				}
			}

			assert.Empty(t, universal.NewWorktree)
			assert.Empty(t, universal.DiffingMenu)
			assert.Empty(t, universal.DiffingMenuAlt)
			assert.Empty(t, universal.ToggleRangeSelect)
			assert.Empty(t, config.Keybinding.Files.CommitChangesWithoutHook)
			assert.Empty(t, config.Keybinding.Commits.PasteCommits)
		})
	}
}

func TestForkDiffDefaultsConfigLoadingPreservesExplicitBindings(t *testing.T) {
	t.Setenv(HonourKeyConfigEnvVar, "1")
	for _, platform := range []string{"linux", "darwin", "windows"} {
		t.Run(platform, func(t *testing.T) {
			t.Setenv("LAZYGIT_KEYBINDING_PLATFORM", platform)
			userPath := filepath.Join(t.TempDir(), "user.yml")
			content := []byte(`keybinding:
  universal:
    diffWithWorkingTree: [x, X]
    diffWithPreviousCommit: [y, Y]
    diffWithWorkingFile: [z]
    newWorktree: n
    diffingMenu: d
    diffingMenu-alt: e
    toggleRangeSelect: r
  files:
    commitChangesWithoutHook: c
  commits:
    pasteCommits: p
`)
			if err := os.WriteFile(userPath, content, 0o600); err != nil {
				t.Fatal(err)
			}
			files := []*ConfigFile{{Path: userPath, Policy: ConfigFilePolicyErrorIfMissing}}

			loaded, err := loadUserConfigWithDefaults(files, false)
			if err != nil {
				t.Fatal(err)
			}
			universal := loaded.Keybinding.Universal
			assert.Equal(t, Keybinding{"x", "X"}, universal.DiffWithWorkingTree)
			assert.Equal(t, Keybinding{"y", "Y"}, universal.DiffWithPreviousCommit)
			assert.Equal(t, Keybinding{"z"}, universal.DiffWithWorkingFile)
			assert.Equal(t, Keybinding{"n"}, universal.NewWorktree)
			assert.Equal(t, Keybinding{"d", "e"}, universal.DiffingMenu)
			assert.Equal(t, Keybinding{"e"}, universal.DiffingMenuAlt)
			assert.Equal(t, Keybinding{"r"}, universal.ToggleRangeSelect)
			assert.Equal(t, Keybinding{"c"}, loaded.Keybinding.Files.CommitChangesWithoutHook)
			assert.Equal(t, Keybinding{"p"}, loaded.Keybinding.Commits.PasteCommits)
		})
	}
}
