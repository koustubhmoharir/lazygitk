package controllers

import (
	"strconv"

	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: while the main view shows a file's content from the full listing of a commit's
// files, t d n and t d p cycle the file renderers (git.fileRenderers) rather than the diff
// renderers. The keys and their notification are the same; the cycle handlers in
// global_controller.go choose between the two.

// showsFileContent reports whether the main view shows the content of a file from the
// full listing, which is what the cycle keys then act on.
func (self *GlobalController) showsFileContent() bool {
	commitFiles, ok := self.c.Context().CurrentSide().(*context.CommitFilesContext)
	return ok && commitFiles.ShowsFileContent()
}

// onFileRenderersChanged re-renders the main view with the newly selected file renderer,
// and shows a toast naming it.
func (self *GlobalController) onFileRenderersChanged() {
	self.c.Helpers().Diff.RenderToMainAgain()

	fileRendererConfigManager := self.c.State().GetFileRendererConfigManager()
	current, total := fileRendererConfigManager.CurrentFileRendererIndex()
	name := fileRendererConfigManager.CurrentFileRendererName(self.c.Tr)
	self.c.Toast(utils.ResolvePlaceholderString(self.c.Tr.SelectedFileRenderers, map[string]string{
		"name":    name,
		"current": strconv.Itoa(current + 1),
		"total":   strconv.Itoa(total),
	}))
}

func (self *GlobalController) canCycleFileRenderers() *types.DisabledReason {
	_, total := self.c.State().GetFileRendererConfigManager().CurrentFileRendererIndex()
	if total <= 1 {
		return &types.DisabledReason{Text: self.c.Tr.CycleFileRenderersDisabledReason}
	}
	return nil
}
