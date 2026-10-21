package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertExplicitActionDefaults(t *testing.T, config *UserConfig) {
	t.Helper()
	assert.False(t, config.Git.AutoFetch)
	assert.False(t, config.Git.AutoRefresh)
	assert.Equal(t, "none", config.Git.AutoForwardBranches)
	assert.False(t, config.Git.AutoStageResolvedConflicts)
	assert.True(t, config.Git.AutoDetectExternalChanges)
	assert.Positive(t, config.Refresher.ExternalChangeCheckInterval)
}

func TestForkDefaultsForPlatform(t *testing.T) {
	assertExplicitActionDefaults(t, GetDefaultConfig())
	for _, platform := range []string{"", "linux", "darwin", "windows"} {
		t.Run(platform, func(t *testing.T) {
			assertExplicitActionDefaults(t, GetDefaultConfigForPlatform(platform))
		})
	}
}

func TestForkDefaultsConfigLoading(t *testing.T) {
	mustSucceed := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, platform := range []string{"linux", "darwin", "windows"} {
		t.Run(platform, func(t *testing.T) {
			t.Setenv("LAZYGIT_KEYBINDING_PLATFORM", platform)
			dir := t.TempDir()
			userPath := filepath.Join(dir, "user.yml")
			repoPath := filepath.Join(dir, "repo.yml")
			userContent := []byte("git:\n  autoFetch: true\n  autoRefresh: true\n  autoForwardBranches: allBranches\n  autoStageResolvedConflicts: true\n  autoDetectExternalChanges: false\n")
			mustSucceed(t, os.WriteFile(userPath, userContent, 0o600))
			files := make([]*ConfigFile, 1, 2)
			files[0] = &ConfigFile{Path: userPath, Policy: ConfigFilePolicyErrorIfMissing}

			// Explicit opt-ins survive both initial loading and GUI config reload.
			for _, initialized := range []bool{false, true} {
				loaded, err := loadUserConfigWithDefaults(files, initialized)
				mustSucceed(t, err)
				assert.True(t, loaded.Git.AutoFetch)
				assert.True(t, loaded.Git.AutoRefresh)
				assert.Equal(t, "allBranches", loaded.Git.AutoForwardBranches)
				assert.True(t, loaded.Git.AutoStageResolvedConflicts)
				assert.False(t, loaded.Git.AutoDetectExternalChanges)
			}

			// Repository configuration retains the last word, including false.
			repoContent := []byte("git:\n  autoFetch: false\n  autoRefresh: false\n  autoForwardBranches: none\n  autoStageResolvedConflicts: false\n  autoDetectExternalChanges: true\n")
			mustSucceed(t, os.WriteFile(repoPath, repoContent, 0o600))
			files = append(files, &ConfigFile{Path: repoPath, Policy: ConfigFilePolicyErrorIfMissing})
			loaded, err := loadUserConfigWithDefaults(files, false)
			mustSucceed(t, err)
			assertExplicitActionDefaults(t, loaded)
			for path, expected := range map[string][]byte{userPath: userContent, repoPath: repoContent} {
				actual, err := os.ReadFile(path)
				mustSucceed(t, err)
				assert.Equal(t, expected, actual)
			}

			// Removing overrides restores defaults rather than retaining old values.
			mustSucceed(t, os.WriteFile(userPath, []byte("{}\n"), 0o600))
			loaded, err = loadUserConfigWithDefaults(files[:1], true)
			mustSucceed(t, err)
			assertExplicitActionDefaults(t, loaded)
		})
	}
}
