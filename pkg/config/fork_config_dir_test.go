package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/assert"
)

// setupXDGDirs points the XDG directories at a fresh temp dir and returns the
// config and state homes. The cleanup is registered before the Setenv calls so
// that, running LIFO, it reloads xdg after the environment has been restored.
func setupXDGDirs(t *testing.T) (configHome string, stateHome string) {
	t.Helper()
	root := t.TempDir()
	configHome = filepath.Join(root, "config")
	stateHome = filepath.Join(root, "state")

	t.Cleanup(xdg.Reload)
	t.Setenv("CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_CONFIG_DIRS", filepath.Join(root, "config-dirs"))
	t.Setenv("XDG_STATE_HOME", stateHome)
	xdg.Reload()

	return configHome, stateHome
}

func touchFile(t *testing.T, path string) {
	t.Helper()
	assert.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	assert.NoError(t, os.WriteFile(path, []byte("{}"), 0o644))
}

func TestFindConfigFileUsesLazygitkDir(t *testing.T) {
	scenarios := []struct {
		name           string
		existing       []string // paths relative to XDG_CONFIG_HOME
		expectedExists bool
	}{
		{
			name:           "No files exist returns the lazygitk default",
			expectedExists: false,
		},
		{
			name:           "Upstream lazygit and legacy directories are ignored",
			existing:       []string{"lazygit/config.yml", "jesseduffield/lazygit/config.yml"},
			expectedExists: false,
		},
		{
			name:           "An existing lazygitk file is found",
			existing:       []string{"lazygitk/config.yml"},
			expectedExists: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			configHome, _ := setupXDGDirs(t)
			for _, rel := range scenario.existing {
				touchFile(t, filepath.Join(configHome, filepath.FromSlash(rel)))
			}

			exists, path := findConfigFile("config.yml")
			assert.Equal(t, scenario.expectedExists, exists)
			assert.Equal(t, filepath.Join(configHome, "lazygitk", "config.yml"), path)
		})
	}
}

func TestFindConfigFileHonorsConfigDirOverride(t *testing.T) {
	setupXDGDirs(t)
	configDir := t.TempDir()
	t.Setenv("CONFIG_DIR", configDir)

	exists, path := findConfigFile("config.yml")
	assert.True(t, exists)
	assert.Equal(t, filepath.Join(configDir, "config.yml"), path)
}

func TestStateFilePathIgnoresUpstreamStateDir(t *testing.T) {
	_, stateHome := setupXDGDirs(t)
	touchFile(t, filepath.Join(stateHome, "lazygit", "state.yml"))

	path, err := stateFilePath("state.yml")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(stateHome, "lazygitk", "state.yml"), path)
}
