package gui

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
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

func effectiveModeHint(bindings []*types.Binding, commandID, description, fallbackDescription string, paletteBindings []*types.Binding, color style.TextStyle) bindingInfo {
	for _, binding := range bindings {
		if binding.GetCommandID() == commandID && (binding.CommandActive == nil || binding.CommandActive()) {
			labels := binding.KeyLabels()
			if len(labels) > 0 {
				return bindingInfo{key: strings.Join(labels, ", "), description: description, style: color}
			}
		}
	}
	for _, binding := range paletteBindings {
		if binding.GetCommandID() == "app.command-palette" && (binding.CommandActive == nil || binding.CommandActive()) {
			labels := binding.KeyLabels()
			if len(labels) > 0 {
				return bindingInfo{key: strings.Join(labels, ", "), description: fallbackDescription, style: color}
			}
		}
	}
	return bindingInfo{description: description, style: color}
}

func appendModeHint(options []bindingInfo, hint bindingInfo) []bindingInfo {
	if hint.key == "" {
		return options
	}
	return utils.Prepend(options, hint)
}
