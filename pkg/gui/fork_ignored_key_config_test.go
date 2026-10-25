package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func TestForkIgnoredKeyConfigMessage(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	setting := config.IgnoredKeySetting{
		Path:  "/home/user/.config/lazygit/config.yml",
		Kinds: []string{"keybinding", "commandBindings", "customCommands key"},
	}

	assert.Equal(t,
		"Key settings are fixed in lazygitk; ignored keybinding, commandBindings, customCommands key in /home/user/.config/lazygit/config.yml",
		forkIgnoredKeyConfigMessage(tr, setting))
	assert.Equal(t,
		"Key settings are fixed in lazygitk; ignored customCommands key in /repo/.lazygit.yml",
		forkIgnoredKeyConfigMessage(tr, config.IgnoredKeySetting{Path: "/repo/.lazygit.yml", Kinds: []string{"customCommands key"}}))
}

func TestForkIgnoredKeyConfigNoticeKeyDistinguishesFilesAndKinds(t *testing.T) {
	a := config.IgnoredKeySetting{Path: "/a.yml", Kinds: []string{"keybinding"}}
	b := config.IgnoredKeySetting{Path: "/a.yml", Kinds: []string{"keybinding", "commandBindings"}}
	c := config.IgnoredKeySetting{Path: "/b.yml", Kinds: []string{"keybinding"}}

	assert.Equal(t, forkIgnoredKeyConfigNoticeKey(a), forkIgnoredKeyConfigNoticeKey(a))
	assert.NotEqual(t, forkIgnoredKeyConfigNoticeKey(a), forkIgnoredKeyConfigNoticeKey(b))
	assert.NotEqual(t, forkIgnoredKeyConfigNoticeKey(a), forkIgnoredKeyConfigNoticeKey(c))
}
