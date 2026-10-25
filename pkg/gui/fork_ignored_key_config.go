package gui

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/i18n"
)

// Fork: startup notice for key settings that the fork ignores (roadmap item 13).
// The loader discards keybinding, commandBindings and custom command keys (see
// pkg/config/fork_ignore_key_config.go); this reports each file that contained
// them, once per file and set of kinds for the session.
type forkKeyConfigNotices struct {
	shown map[string]bool
}

// showForkIgnoredKeyConfigNotices toasts each ignored-settings entry that has
// not been shown yet. It runs after the startup layout and after each
// repository's config is loaded, so global and per-repo files are both covered.
func (gui *Gui) showForkIgnoredKeyConfigNotices() {
	for _, setting := range gui.Config.GetUserConfig().IgnoredKeySettings() {
		key := forkIgnoredKeyConfigNoticeKey(setting)
		if gui.forkKeyConfigNotices.shown[key] {
			continue
		}
		if gui.forkKeyConfigNotices.shown == nil {
			gui.forkKeyConfigNotices.shown = make(map[string]bool)
		}
		gui.forkKeyConfigNotices.shown[key] = true
		gui.c.Toast(forkIgnoredKeyConfigMessage(gui.Tr, setting))
	}
}

func forkIgnoredKeyConfigNoticeKey(setting config.IgnoredKeySetting) string {
	return setting.Path + "\x00" + strings.Join(setting.Kinds, ",")
}

func forkIgnoredKeyConfigMessage(tr *i18n.TranslationSet, setting config.IgnoredKeySetting) string {
	return fmt.Sprintf(tr.IgnoredKeyConfig, strings.Join(setting.Kinds, ", "), setting.Path)
}
