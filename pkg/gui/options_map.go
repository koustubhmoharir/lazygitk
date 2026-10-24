package gui

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/theme"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

type OptionsMapMgr struct {
	c *helpers.HelperCommon
}

func (gui *Gui) renderContextOptionsMap() {
	// In demos, we render our own content to this view
	if gui.integrationTest != nil && gui.integrationTest.IsDemo() {
		return
	}
	mgr := OptionsMapMgr{c: gui.c}
	mgr.renderContextOptionsMap()
}

// Render the options available for the current context at the bottom of the screen
// STYLE GUIDE: we use the default options fg color for most keybindings. We can
// only use a different color if we're in a specific mode where the user is likely
// to want to press that key. For example, when in cherry-picking mode, we
// want to prominently show the keybinding for pasting commits.
func (self *OptionsMapMgr) renderContextOptionsMap() {
	currentContext := self.c.Context().Current()

	allBindings, _ := self.c.GetInitialKeybindingsWithCustomCommands()
	view := currentContext.GetViewName()
	displayBindings := make([]*types.Binding, 0, len(allBindings))
	for _, binding := range allBindings {
		// Normal command compilation owns global core routes only in normal
		// panes. Popup/editor hints retain native custom-command registrations.
		if !forkHintBindingInContext(binding, view) {
			continue
		}
		displayBindings = append(displayBindings, binding)
	}
	// Preserve native local-before-global precedence for aliases and footer order.
	displayBindings = localFirstForkHintBindings(displayBindings)
	displayBindings = deduplicateForkDisplayAliases(displayBindings)
	modeHintBindings := displayBindings

	bindingsToDisplay := lo.Filter(displayBindings, func(binding *types.Binding, _ int) bool {
		// A binding that describes itself as nothing has nothing to do where we are —
		// that is how a command which only applies to some of a view's contents says so
		// — and an empty entry in the options bar would say nothing about it.
		return len(binding.KeyLabels()) > 0 && binding.DisplayOnScreen && !binding.IsDisabled() &&
			binding.GetShortDescription() != ""
	})
	// Fork: prioritize the primary history comparison in hints only; upstream
	// controller/dispatch ordering remains unchanged.
	bindingsToDisplay = prioritizeHistoricalDiffHints(bindingsToDisplay)

	optionsMap := lo.Map(bindingsToDisplay, func(binding *types.Binding, _ int) bindingInfo {
		displayStyle := theme.OptionsFgColor
		if binding.DisplayStyle != nil {
			displayStyle = *binding.DisplayStyle
		}

		return bindingInfo{
			key:         strings.Join(binding.KeyLabels(), ", "),
			description: binding.GetShortDescription(),
			style:       displayStyle,
		}
	})

	// Mode-specific local keybindings
	if currentContext.GetKey() == context.LOCAL_COMMITS_CONTEXT_KEY {
		if self.c.Modes().CherryPicking.Active() {
			optionsMap = appendModeHint(optionsMap, effectiveModeHint(displayBindings, "commit.cherry-pick", self.c.Tr.PasteCommits, self.c.Tr.PasteCommitsPaletteHint, modeHintBindings, style.FgCyan))
		}

		if self.c.Model().BisectInfo.Started() {
			optionsMap = appendModeHint(optionsMap, effectiveModeHint(displayBindings, "bisect.options-menu", self.c.Tr.ViewBisectOptions, "View bisect options", modeHintBindings, style.FgGreen))
		}
	}

	// Mode-specific global keybindings
	if state := self.c.Model().WorkingTreeStateAtLastCommitRefresh; state.Any() {
		optionsMap = appendModeHint(optionsMap, effectiveModeHint(displayBindings, "branch.merge-rebase-menu", state.OptionsMapTitle(self.c.Tr), state.OptionsMapTitle(self.c.Tr), modeHintBindings, style.FgYellow))
	}

	if self.c.Git().Patch.PatchBuilder.Active() {
		optionsMap = appendModeHint(optionsMap, effectiveModeHint(displayBindings, "patch.options-menu", self.c.Tr.ViewPatchOptions, self.c.Tr.ViewPatchOptions, modeHintBindings, style.FgYellow))
	}

	self.renderOptions(self.formatBindingInfos(optionsMap))
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

func (self *OptionsMapMgr) formatBindingInfos(bindingInfos []bindingInfo) string {
	width := self.c.Views().Options.InnerWidth() - 2 // -2 for some padding
	var builder strings.Builder
	ellipsis := "…"
	separator := " | "

	length := 0

	for i, info := range bindingInfos {
		plainText := fmt.Sprintf("%s: %s", info.description, info.key)

		// Check if adding the next formatted string exceeds the available width
		textLen := utils.StringWidth(plainText)
		if i > 0 && length+len(separator)+textLen > width {
			builder.WriteString(theme.OptionsFgColor.Sprint(separator + ellipsis))
			break
		}

		formatted := info.style.Sprint(plainText)

		if i > 0 {
			builder.WriteString(theme.OptionsFgColor.Sprint(separator))
			length += len(separator)
		}
		builder.WriteString(formatted)
		length += textLen
	}

	return builder.String()
}

func (self *OptionsMapMgr) renderOptions(options string) {
	self.c.SetViewContent(self.c.Views().Options, options)
}

type bindingInfo struct {
	key         string
	description string
	style       style.TextStyle
}
