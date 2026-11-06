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

// showForkIgnoredKeyConfigNotices queues a notice for each ignored-settings
// entry that has not been shown yet. onNewRepo calls it after each
// repository's config is loaded, so global and per-repo files are both
// covered. An entry is marked shown when it is queued.
func (gui *Gui) showForkIgnoredKeyConfigNotices() {
	if gui.forkKeyConfigNotices.shown == nil {
		gui.forkKeyConfigNotices.shown = make(map[string]bool)
	}
	gui.enqueueForkNotices(forkIgnoredKeyConfigNotices(gui.forkKeyConfigNotices.shown, gui.Config.GetUserConfig().IgnoredKeySettings(), gui.Tr)...)
}

// forkIgnoredKeyConfigNotices returns the messages for the settings not yet in
// shown, and marks each of them shown.
func forkIgnoredKeyConfigNotices(shown map[string]bool, settings []config.IgnoredKeySetting, tr *i18n.TranslationSet) []string {
	var messages []string
	for _, setting := range settings {
		key := forkIgnoredKeyConfigNoticeKey(setting)
		if shown[key] {
			continue
		}
		shown[key] = true
		messages = append(messages, forkIgnoredKeyConfigMessage(tr, setting))
	}
	return messages
}

func forkIgnoredKeyConfigNoticeKey(setting config.IgnoredKeySetting) string {
	return setting.Path + "\x00" + strings.Join(setting.Kinds, ",")
}

func forkIgnoredKeyConfigMessage(tr *i18n.TranslationSet, setting config.IgnoredKeySetting) string {
	return fmt.Sprintf(tr.IgnoredKeyConfig, strings.Join(setting.Kinds, ", "), setting.Path)
}
