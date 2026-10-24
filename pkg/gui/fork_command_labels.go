package gui

import "github.com/jesseduffield/lazygit/pkg/gui/types"

// An upstream binding's dynamic label describes whichever action its shared
// key performs now. Fork routes name each semantic action independently, even
// while another mode makes it unavailable in the command palette.
func (gui *Gui) labelForkCommandVariant(binding *types.Binding) {
	var description, tooltip string
	switch binding.CommandID {
	case "file.toggle-stage":
		description, tooltip = gui.Tr.ForkToggleFileStage, gui.Tr.StageTooltip
	case "conflict.options-menu":
		description, tooltip = gui.Tr.ViewMergeConflictOptions, gui.Tr.ViewMergeConflictOptionsTooltip
	case "diff.stage-selection":
		description, tooltip = gui.Tr.Stage, gui.Tr.StageSelectionTooltip
	case "diff.unstage-selection":
		description = gui.Tr.ForkUnstageSelection
	case "patch.toggle-selection":
		description = gui.Tr.ToggleSelectionForPatch
	case "patch.remove-selection":
		description = gui.Tr.ForkRemovePatchSelection
	case "diff.discard-worktree-selection":
		description, tooltip = gui.Tr.DiscardSelection, gui.Tr.DiscardSelectionTooltip
	case "commit.discard-selection":
		description, tooltip = gui.Tr.RemoveSelectionFromPatch, gui.Tr.RemoveSelectionFromPatchTooltip
	default:
		return
	}
	binding.Description, binding.Tooltip = description, tooltip
	binding.ShortDescription = ""
	binding.DescriptionFunc, binding.ShortDescriptionFunc, binding.TooltipFunc = nil, nil, nil
}
