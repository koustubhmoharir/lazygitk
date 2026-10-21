package gui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
)

// A controller's attachment order should not push the primary historical
// comparison off the visible end of the options bar. Copy only the display
// list, leaving dispatch order and every binding untouched.
func prioritizeHistoricalDiffHints(bindings []*types.Binding) []*types.Binding {
	isWorkingDiff := func(binding *types.Binding) bool {
		id := binding.GetCommandID()
		return id == "diff.commit.worktree" || id == "diff.history-file.worktree"
	}
	result := make([]*types.Binding, 0, len(bindings))
	for _, primary := range []bool{true, false} {
		for _, binding := range bindings {
			if isWorkingDiff(binding) == primary {
				result = append(result, binding)
			}
		}
	}
	return result
}

// cherryPickModeHint keeps the copied-commit workflow discoverable when paste
// has no shortcut; a missing key must not produce an empty action hint.
func cherryPickModeHint(keys config.KeybindingConfig, tr *i18n.TranslationSet) bindingInfo {
	key, description := keys.Commits.PasteCommits.String(), tr.PasteCommits
	if len(keys.Commits.PasteCommits) == 0 {
		key, description = keys.Universal.OptionMenu.String(), tr.PasteCommitsPaletteHint
	}
	return bindingInfo{key: key, description: description, style: style.FgCyan}
}
