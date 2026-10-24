package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForkEditorDefaultsForPlatform(t *testing.T) {
	for _, platform := range []string{"linux", "darwin", "windows"} {
		t.Run(platform, func(t *testing.T) {
			defaults := GetDefaultConfigForPlatform(platform).Keybinding.Universal
			assert.Contains(t, defaults.MoveWordLeft, "<ctrl+left>")
			assert.Contains(t, defaults.MoveWordRight, "<ctrl+right>")
			if platform == "darwin" {
				assert.Contains(t, defaults.MoveWordLeft, "<alt+left>")
				assert.Contains(t, defaults.MoveWordRight, "<alt+right>")
			}
			assert.Equal(t, 1, countKey(defaults.MoveWordLeft, "<ctrl+left>"))
			assert.Equal(t, 1, countKey(defaults.MoveWordRight, "<ctrl+right>"))
		})
	}
}

func TestForkEditorDefaultsConfigLoadingPreservesExplicitBindings(t *testing.T) {
	for _, platform := range []string{"linux", "darwin", "windows"} {
		t.Run(platform, func(t *testing.T) {
			t.Setenv("LAZYGIT_KEYBINDING_PLATFORM", platform)
			path := filepath.Join(t.TempDir(), "user.yml")
			content := []byte("keybinding:\n  universal:\n    moveWordLeft: [x]\n    moveWordRight: []\n")
			if err := os.WriteFile(path, content, 0o600); err != nil {
				t.Fatal(err)
			}
			loaded, err := loadUserConfigWithDefaults([]*ConfigFile{{Path: path, Policy: ConfigFilePolicyErrorIfMissing}}, false)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, Keybinding{"x"}, loaded.Keybinding.Universal.MoveWordLeft)
			assert.Empty(t, loaded.Keybinding.Universal.MoveWordRight)
		})
	}
}

func countKey(keys Keybinding, target string) int {
	count := 0
	for _, key := range keys {
		if key == target {
			count++
		}
	}
	return count
}
