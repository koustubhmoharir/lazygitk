package config

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func newForkFileRendererTestManager(config *UserConfig) *FileRendererConfigManager {
	return NewFileRendererConfigManager(func() *UserConfig { return config })
}

func TestUnsetFileRenderersGetTheBuiltInThenPlain(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for _, content := range []string{"gui:\n  showRandomTip: false\n", "git:\n  fileRenderers: []\n"} {
		manager := newForkFileRendererTestManager(loadForkDiffRendererConfig(t, content))

		_, total := manager.CurrentFileRendererIndex()
		assert.Equal(t, 2, total)
		assert.Equal(t, FileRendererType_Builtin, manager.CurrentFileRendererType())
		assert.Equal(t, tr.BuiltinFileRendererName, manager.CurrentFileRendererName(tr))

		manager.CycleFileRenderers()
		assert.Equal(t, FileRendererType_Plain, manager.CurrentFileRendererType())
		assert.Equal(t, tr.PlainFileRendererName, manager.CurrentFileRendererName(tr))
	}
}

func TestUserFileRenderersAreUsedAsGiven(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	loaded := loadForkDiffRendererConfig(t, `
git:
  fileRenderers:
    - type: plain
    - type: stdinFilter
      command: bat --style=numbers
    - type: builtin
      name: Chroma
`)
	manager := newForkFileRendererTestManager(loaded)

	_, total := manager.CurrentFileRendererIndex()
	assert.Equal(t, 3, total)
	assert.Equal(t, FileRendererType_Plain, manager.CurrentFileRendererType())

	manager.CycleFileRenderers()
	assert.Equal(t, FileRendererType_StdinFilter, manager.CurrentFileRendererType())
	assert.Equal(t, "bat", manager.CurrentFileRendererName(tr))

	manager.CycleFileRenderers()
	assert.Equal(t, "Chroma", manager.CurrentFileRendererName(tr))

	manager.CycleFileRenderers()
	assert.Equal(t, tr.PlainFileRendererName, manager.CurrentFileRendererName(tr), "cycling wraps around")
}

func TestFileRendererCyclingBackwardsWrapsAround(t *testing.T) {
	manager := newForkFileRendererTestManager(GetDefaultConfig())

	manager.CycleFileRenderersBackward()
	current, total := manager.CurrentFileRendererIndex()
	assert.Equal(t, 1, current)
	assert.Equal(t, 2, total)
	assert.Equal(t, FileRendererType_Plain, manager.CurrentFileRendererType())
}

func TestFileRendererIndexIsResetWhenTheListShrinks(t *testing.T) {
	config := GetDefaultConfig()
	manager := newForkFileRendererTestManager(config)
	manager.CycleFileRenderers()

	config.Git.FileRenderers = []FileRendererConfig{{Type: "plain"}}

	assert.Equal(t, FileRendererType_Plain, manager.CurrentFileRendererType())
	current, total := manager.CurrentFileRendererIndex()
	assert.Equal(t, 0, current)
	assert.Equal(t, 1, total)
}

func TestUserConfigValidate_fileRenderers(t *testing.T) {
	scenarios := []struct {
		name         string
		fileRenderer FileRendererConfig
		valid        bool
	}{
		{name: "builtin", fileRenderer: FileRendererConfig{Type: "builtin"}, valid: true},
		{name: "plain", fileRenderer: FileRendererConfig{Type: "plain", Name: "Text"}, valid: true},
		{name: "builtin with command", fileRenderer: FileRendererConfig{Type: "builtin", Command: "bat"}, valid: false},
		{name: "plain with command", fileRenderer: FileRendererConfig{Type: "plain", Command: "cat"}, valid: false},
		{name: "stdinFilter", fileRenderer: FileRendererConfig{Type: "stdinFilter", Command: "bat"}, valid: true},
		{name: "stdinFilter without command", fileRenderer: FileRendererConfig{Type: "stdinFilter"}, valid: false},
		{name: "without type", fileRenderer: FileRendererConfig{Command: "bat"}, valid: false},
		{name: "unknown type", fileRenderer: FileRendererConfig{Type: "unknown"}, valid: false},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			config := GetDefaultConfig()
			config.Git.FileRenderers = []FileRendererConfig{s.fileRenderer}
			err := config.Validate()

			if s.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestForkFileRendererNamesAreTheTypeNamesWhenUnnamed(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	assert.Equal(t, tr.BuiltinFileRendererName, newForkFileRendererTestManager(GetDefaultConfig()).CurrentFileRendererName(tr))
	config := GetDefaultConfig()
	config.Git.FileRenderers = []FileRendererConfig{{Type: "stdinFilter", Command: "bat --style=plain"}}
	assert.Equal(t, "bat", newForkFileRendererTestManager(config).CurrentFileRendererName(tr))
}
