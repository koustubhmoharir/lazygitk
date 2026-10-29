package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

// setForkLookPath makes delta appear on PATH or not for one test, and counts
// the lookups so that the caching can be checked.
func setForkLookPath(t *testing.T, deltaFound bool) *int {
	t.Helper()
	calls := 0
	original := forkLookPath
	forkLookPath = func(file string) (string, error) {
		calls++
		if file == "delta" && deltaFound {
			return "/usr/bin/delta", nil
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() { forkLookPath = original })
	return &calls
}

func newForkDiffRendererTestManager(config *UserConfig) *DiffRendererConfigManager {
	return NewDiffRendererConfigManager(func() *UserConfig { return config })
}

func loadForkDiffRendererConfig(t *testing.T, content string) *UserConfig {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadUserConfigWithDefaults([]*ConfigFile{{Path: path, Policy: ConfigFilePolicyErrorIfMissing}}, false)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func TestForkDefaultDiffRenderersWhenDeltaIsFound(t *testing.T) {
	setForkLookPath(t, true)
	config := GetDefaultConfig()
	manager := newForkDiffRendererTestManager(config)

	current, total := manager.CurrentDiffRendererIndex()
	assert.Equal(t, 0, current)
	assert.Equal(t, 2, total)
	assert.False(t, manager.DefaultDiffRendererFellBack())

	tr := &i18n.TranslationSet{DefaultDiffRendererName: "(default)"}
	assert.Equal(t, "delta", manager.CurrentDiffRendererName(tr))
	assert.Equal(t, DiffRendererType_StdinFilter, manager.GetDiffRendererType())
	command, err := manager.GetStdinFilterCommand(DiffRendererValues{Width: 80, LightBackground: true})
	assert.NoError(t, err)
	assert.Equal(t, "delta --light --paging=never --tabs=4"+forkDeltaArgs(true), command)
	assert.Equal(t, "always", manager.GetColorArg())

	manager.CycleDiffRenderers()
	assert.Equal(t, DiffRendererType_RawGit, manager.GetDiffRendererType())
	assert.Equal(t, "default", manager.CurrentDiffRendererName(tr))
	current, total = manager.CurrentDiffRendererIndex()
	assert.Equal(t, 1, current)
	assert.Equal(t, 2, total)
}

func TestForkDefaultDiffRenderersWhenDeltaIsNotFound(t *testing.T) {
	setForkLookPath(t, false)
	manager := newForkDiffRendererTestManager(GetDefaultConfig())

	current, total := manager.CurrentDiffRendererIndex()
	assert.Equal(t, 0, current)
	assert.Equal(t, 1, total)
	assert.True(t, manager.DefaultDiffRendererFellBack())
	assert.Equal(t, DiffRendererType_RawGit, manager.GetDiffRendererType())
	assert.Equal(t, "default", manager.CurrentDiffRendererName(&i18n.TranslationSet{}))

	// With one renderer left, cycling is a no-op and does not divide by zero.
	manager.CycleDiffRenderers()
	manager.CycleDiffRenderersBackward()
	current, total = manager.CurrentDiffRendererIndex()
	assert.Equal(t, 0, current)
	assert.Equal(t, 1, total)
}

func TestForkDefaultDiffRendererLookPathIsCached(t *testing.T) {
	calls := setForkLookPath(t, false)
	manager := newForkDiffRendererTestManager(GetDefaultConfig())

	for range 5 {
		manager.CurrentDiffRendererIndex()
		manager.DefaultDiffRendererFellBack()
		manager.CycleDiffRenderers()
	}
	assert.Equal(t, 1, *calls)
}

func TestUserDiffRenderersAreNeverFiltered(t *testing.T) {
	setForkLookPath(t, false)
	loaded := loadForkDiffRendererConfig(t, `git:
  diffRenderers:
    - command: delta --paging=never
    - type: rawGit
      name: plain
`)
	manager := newForkDiffRendererTestManager(loaded)

	assert.Len(t, loaded.Git.DiffRenderers, 2)
	current, total := manager.CurrentDiffRendererIndex()
	assert.Equal(t, 0, current)
	assert.Equal(t, 2, total)
	assert.False(t, manager.DefaultDiffRendererFellBack())
	assert.Equal(t, DiffRendererType_StdinFilter, manager.GetDiffRendererType())
	assert.Equal(t, "delta", manager.CurrentDiffRendererName(&i18n.TranslationSet{}))
}

func TestUserDiffRenderersReplaceForkDefaultsWhenLoaded(t *testing.T) {
	setForkLookPath(t, true)
	loaded := loadForkDiffRendererConfig(t, `git:
  diffRenderers:
    - type: rawGit
`)
	assert.Len(t, loaded.Git.DiffRenderers, 1)
	manager := newForkDiffRendererTestManager(loaded)
	_, total := manager.CurrentDiffRendererIndex()
	assert.Equal(t, 1, total)
	assert.Equal(t, DiffRendererType_RawGit, manager.GetDiffRendererType())
}

func TestUnsetDiffRenderersGetTheForkDefault(t *testing.T) {
	setForkLookPath(t, true)
	loaded := loadForkDiffRendererConfig(t, "gui:\n  showRandomTip: false\n")
	assert.Empty(t, loaded.Git.DiffRenderers)
	manager := newForkDiffRendererTestManager(loaded)

	_, total := manager.CurrentDiffRendererIndex()
	assert.Equal(t, 2, total)
	assert.Equal(t, DiffRendererType_StdinFilter, manager.GetDiffRendererType())
}

func TestExplicitEmptyDiffRenderersBehaveLikeUnset(t *testing.T) {
	for _, deltaFound := range []bool{true, false} {
		setForkLookPath(t, deltaFound)
		unset := newForkDiffRendererTestManager(loadForkDiffRendererConfig(t, "gui:\n  showRandomTip: false\n"))
		explicit := newForkDiffRendererTestManager(loadForkDiffRendererConfig(t, "git:\n  diffRenderers: []\n"))
		tr := &i18n.TranslationSet{DefaultDiffRendererName: "(default)"}

		for _, manager := range []*DiffRendererConfigManager{unset, explicit} {
			assert.Equal(t, deltaFound, !manager.DefaultDiffRendererFellBack())
			assert.Equal(t, deltaFound, manager.GetDiffRendererType() == DiffRendererType_StdinFilter)
		}
		_, unsetTotal := unset.CurrentDiffRendererIndex()
		_, explicitTotal := explicit.CurrentDiffRendererIndex()
		assert.Equal(t, unsetTotal, explicitTotal)
		assert.Equal(t, unset.CurrentDiffRendererName(tr), explicit.CurrentDiffRendererName(tr))
	}
}

func TestDiffRendererCyclingNeverSeesAnEmptyList(t *testing.T) {
	setForkLookPath(t, false)
	manager := newForkDiffRendererTestManager(loadForkDiffRendererConfig(t, "git:\n  diffRenderers: []\n"))
	// Each of these would panic with a zero-length list.
	manager.CycleDiffRenderers()
	manager.CycleDiffRenderersBackward()
	assert.Equal(t, DiffRendererType_RawGit, manager.GetDiffRendererType())
	_, err := manager.GetStdinFilterCommand(DiffRendererValues{})
	assert.NoError(t, err)
}

// forkDeltaArgs is what the default delta command passes when the git config
// sets none of its settings, for a background of black or white.
func forkDeltaArgs(lightBackground bool) string {
	if lightBackground {
		return ` --true-color="always"` + forkDeltaBlueArgs("#1e66f5") +
			` --plus-style="syntax #e8f4e6"` +
			` --plus-emph-style="syntax #d5ead0"` +
			` --plus-empty-line-marker-style="normal #d5ead0"` +
			` --minus-style="normal #fae2e7"` +
			` --minus-emph-style="normal #f5cad3"` +
			` --minus-empty-line-marker-style="normal #f5cad3"`
	}
	return ` --true-color="always"` + forkDeltaBlueArgs("#89b4fa") +
		` --plus-style="syntax #141b13"` +
		` --plus-emph-style="syntax #253223"` +
		` --plus-empty-line-marker-style="normal #253223"` +
		` --minus-style="normal #1d1114"` +
		` --minus-emph-style="normal #351f25"` +
		` --minus-empty-line-marker-style="normal #351f25"`
}

func forkDeltaBlueArgs(blue string) string {
	return ` --file-style="` + blue + `"` +
		` --file-decoration-style="` + blue + ` ul"` +
		` --hunk-header-file-style="` + blue + `"` +
		` --hunk-header-line-number-style="` + blue + `"` +
		` --hunk-header-decoration-style="` + blue + ` box"` +
		` --line-numbers-left-style="` + blue + `"` +
		` --line-numbers-right-style="` + blue + `"`
}

func TestForkDeltaCommandFollowsThePalette(t *testing.T) {
	setForkLookPath(t, true)
	manager := newForkDiffRendererTestManager(GetDefaultConfig())

	// An unknown background counts as black or white.
	command, err := manager.GetStdinFilterCommand(DiffRendererValues{Width: 80})
	assert.NoError(t, err)
	assert.Equal(t, "delta --dark --paging=never --tabs=4"+forkDeltaArgs(false), command)

	command, err = manager.GetStdinFilterCommand(DiffRendererValues{Width: 80, BackgroundColor: "#000000"})
	assert.NoError(t, err)
	assert.Equal(t, "delta --dark --paging=never --tabs=4"+forkDeltaArgs(false), command)
}

func TestForkDeltaCommandTintsTheTerminalBackground(t *testing.T) {
	setForkLookPath(t, true)
	manager := newForkDiffRendererTestManager(GetDefaultConfig())

	command, err := manager.GetStdinFilterCommand(DiffRendererValues{Width: 80, BackgroundColor: "#1e1e2e"})
	assert.NoError(t, err)
	assert.Contains(t, command, ` --plus-style="syntax #2e363c"`)
	assert.Contains(t, command, ` --minus-emph-style="normal #4d3649"`)
}

func TestForkDeltaCommandLeavesSettingsOfTheGitConfigToDelta(t *testing.T) {
	setForkLookPath(t, true)
	manager := newForkDiffRendererTestManager(GetDefaultConfig())
	// Keys of the main section and of a feature both count, in any case.
	manager.SetForkGitDeltaConfig(func() string {
		return "delta.side-by-side true\n" +
			"delta.true-color never\n" +
			"delta.file-style red\n" +
			"delta.mytheme.Hunk-Header-Decoration-Style green box\n" +
			"delta.line-numbers-left-style\n" +
			"delta.plus-style syntax #003800\n" +
			"delta.mytheme.minus-emph-style normal #901011"
	})

	command, err := manager.GetStdinFilterCommand(DiffRendererValues{Width: 80, BackgroundColor: "#000000"})
	assert.NoError(t, err)
	assert.Equal(t, "delta --dark --paging=never --tabs=4"+
		` --file-decoration-style="#89b4fa ul"`+
		` --hunk-header-file-style="#89b4fa"`+
		` --hunk-header-line-number-style="#89b4fa"`+
		` --line-numbers-right-style="#89b4fa"`+
		` --plus-emph-style="syntax #253223"`+
		` --plus-empty-line-marker-style="normal #253223"`+
		` --minus-style="normal #1d1114"`+
		` --minus-empty-line-marker-style="normal #351f25"`, command)
}
