package gui

import (
	"fmt"
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

// commandRoute names a command and the view whose effective key label is shown.
type commandRoute struct {
	commandID string
	view      string
}

// keyTip formats a tip with the effective label of each route. It returns ""
// when any route has no key, so that the tip is dropped rather than naming a
// key that does not exist.
func (gui *Gui) keyTip(format string, routes ...commandRoute) string {
	labels := make([]any, len(routes))
	for i, route := range routes {
		label := gui.c.CommandKeyLabel(route.commandID, route.view)
		if label == "" {
			return ""
		}
		labels[i] = label
	}
	return fmt.Sprintf(format, labels...)
}

func localFirstForkHintBindings(bindings []*types.Binding) []*types.Binding {
	result := make([]*types.Binding, 0, len(bindings))
	for _, binding := range bindings {
		if binding.ViewName != "" {
			result = append(result, binding)
		}
	}
	for _, binding := range bindings {
		if binding.ViewName == "" {
			result = append(result, binding)
		}
	}
	return result
}

func forkHintBindingInContext(binding *types.Binding, view string) bool {
	if binding.ViewName != "" && binding.ViewName != view {
		return false
	}
	return binding.ViewName != "" || forkCommandView(view) || strings.HasPrefix(binding.GetCommandID(), "custom.")
}
