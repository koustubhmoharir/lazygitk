package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

func cherryPickCopyTooltip(tr *i18n.TranslationSet, keys config.KeybindingConfig) string {
	template := tr.CherryPickCopyTooltip
	if len(keys.Commits.PasteCommits) == 0 {
		template = tr.CherryPickCopyPaletteTooltip
	}
	return utils.ResolvePlaceholderString(template, map[string]string{
		"paste": keys.Commits.PasteCommits.String(), "escape": keys.Universal.Return.String(),
	})
}
