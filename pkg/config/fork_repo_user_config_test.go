package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrepareRepoUserConfigPublishesOnlyOnCommit(t *testing.T) {
	current := &AppConfig{}
	// The current config before a switch: a repo config with a known language.
	currentPath := filepath.Join(t.TempDir(), "current.yml")
	assert.NoError(t, os.WriteFile(currentPath, []byte("gui:\n  language: en\n"), 0o600))
	files := []*ConfigFile{{Path: currentPath, Policy: ConfigFilePolicyErrorIfMissing}}
	assert.NoError(t, current.ReloadUserConfigForRepo(files))
	before := current.GetUserConfig()

	// The target repo's config is loaded but not published.
	targetPath := filepath.Join(t.TempDir(), "target.yml")
	assert.NoError(t, os.WriteFile(targetPath, []byte("gui:\n  language: xx\n"), 0o600))
	prepared, err := current.PrepareRepoUserConfig([]*ConfigFile{{Path: targetPath, Policy: ConfigFilePolicyErrorIfMissing}})
	assert.NoError(t, err)
	assert.Equal(t, "xx", prepared.UserConfig().Gui.Language)
	assert.Same(t, before, current.GetUserConfig())
	assert.Equal(t, []string{currentPath}, current.GetUserConfigPaths())

	current.CommitRepoUserConfig(prepared)
	assert.Equal(t, "xx", current.GetUserConfig().Gui.Language)
	assert.Equal(t, []string{targetPath}, current.GetUserConfigPaths())
}

func TestPrepareRepoUserConfigInvalidYAMLFailsWithoutPublishing(t *testing.T) {
	current := &AppConfig{}
	assert.NoError(t, current.ReloadUserConfigForRepo(nil))
	before := current.GetUserConfig()

	badPath := filepath.Join(t.TempDir(), "bad.yml")
	assert.NoError(t, os.WriteFile(badPath, []byte("gui: [\n"), 0o600))
	_, err := current.PrepareRepoUserConfig([]*ConfigFile{{Path: badPath, Policy: ConfigFilePolicyErrorIfMissing}})
	assert.Error(t, err)
	assert.Same(t, before, current.GetUserConfig())
}
