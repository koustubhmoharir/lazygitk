package gui

import (
	"github.com/jesseduffield/lazygit/pkg/i18n"
)

// Fork: checkRepoSwitchLanguage is the language check that onUserConfigLoaded
// makes before it applies a config, run before a repo switch publishes
// anything. It uses the same condition and the same call, so a language that
// passes here cannot fail there. i18n.NewTranslationSetFromConfig has no side
// effects; the translation set it builds is discarded here and built again by
// onUserConfigLoaded when the language changes.
func (gui *Gui) checkRepoSwitchLanguage(language string) error {
	if gui.previousLanguageConfig == language {
		return nil
	}
	_, err := i18n.NewTranslationSetFromConfig(gui.Log, language)
	return err
}
