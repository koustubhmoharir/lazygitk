package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForkLogDefaultShowsWholeGraphForEveryPlatform(t *testing.T) {
	for _, platform := range []string{"", "linux", "darwin", "windows"} {
		t.Run(platform, func(t *testing.T) {
			defaults := GetDefaultConfigForPlatform(platform)
			assert.True(t, defaults.Git.Log.ShowWholeGraph)
			assert.Equal(t, "always", defaults.Git.Log.ShowGraph)
		})
	}
}

func TestForkLogDefaultConfigLoadingPreservesOverridesAndReload(t *testing.T) {
	t.Setenv("LAZYGIT_KEYBINDING_PLATFORM", "linux")
	dir := t.TempDir()
	userPath := filepath.Join(dir, "user.yml")
	repoPath := filepath.Join(dir, "repo.yml")
	userContent := []byte("git:\n  log:\n    showWholeGraph: false\n")
	repoContent := []byte("git:\n  log:\n    showWholeGraph: true\n")
	if err := os.WriteFile(userPath, userContent, 0o600); err != nil {
		t.Fatal(err)
	}
	files := make([]*ConfigFile, 1, 2)
	files[0] = &ConfigFile{Path: userPath, Policy: ConfigFilePolicyErrorIfMissing}

	loaded, err := loadUserConfigWithDefaults(files, false)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, loaded.Git.Log.ShowWholeGraph)
	assert.Equal(t, userContent, mustReadFile(t, userPath))

	if err := os.WriteFile(repoPath, repoContent, 0o600); err != nil {
		t.Fatal(err)
	}
	files = append(files, &ConfigFile{Path: repoPath, Policy: ConfigFilePolicyErrorIfMissing})
	loaded, err = loadUserConfigWithDefaults(files, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, loaded.Git.Log.ShowWholeGraph)
	assert.Equal(t, userContent, mustReadFile(t, userPath))
	assert.Equal(t, repoContent, mustReadFile(t, repoPath))

	if err := os.Remove(repoPath); err != nil {
		t.Fatal(err)
	}
	loaded, err = loadUserConfigWithDefaults(files[:1], true)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, loaded.Git.Log.ShowWholeGraph)

	if err := os.WriteFile(userPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err = loadUserConfigWithDefaults(files[:1], true)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, loaded.Git.Log.ShowWholeGraph)
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
