package controllers

import (
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/i18n"
)

func TestCherryPickCopyTooltip(t *testing.T) {
	keys := config.GetDefaultConfig().Keybinding
	tr := i18n.EnglishTranslationSet()
	if got := cherryPickCopyTooltip(tr, keys); !strings.Contains(got, "command palette") || strings.Contains(got, "``") {
		t.Fatalf("unbound paste tooltip should explain palette: %q", got)
	}
	keys.Commits.PasteCommits = config.Keybinding{"x"}
	if got := cherryPickCopyTooltip(tr, keys); !strings.Contains(got, "`x`") || strings.Contains(got, "{{") {
		t.Fatalf("configured paste tooltip should explain its key: %q", got)
	}
}
