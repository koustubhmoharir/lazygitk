package gui

import (
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// F5 replaces both upstream refresh routes with one complete reload and redraw.
// In particular, a nil upstream refresh scope omits selected-ref history.
func (gui *Gui) refreshAllForkState() error {
	git, generation := gui.git, gui.repoGeneration.Load()
	git.Config.DropConfigCache()
	gui.c.Refresh(types.RefreshOptions{
		Scope: []types.RefreshableView{
			types.COMMITS, types.REBASE_COMMITS, types.SUB_COMMITS, types.BRANCHES,
			types.FILES, types.STASH, types.REFLOG, types.TAGS, types.REMOTES,
			types.WORKTREES, types.STATUS, types.SUBMODULES, types.MERGE_CONFLICTS,
			types.COMMIT_FILES, types.BISECT_INFO, types.PULL_REQUESTS,
		},
		BatchUIUpdates: true,
		Then: func() error {
			if gui.git != git || gui.repoGeneration.Load() != generation {
				return nil
			}
			for _, window := range gui.State.WindowViewNameMap.Keys() {
				viewName, exists := gui.State.WindowViewNameMap.Get(window)
				if !exists {
					continue
				}
				// Layout-only windows (for example the options bar) have no context.
				ctx, exists := gui.helpers.View.ContextForView(viewName)
				if !exists {
					continue
				}
				if ctx.GetKind() == types.SIDE_CONTEXT || ctx.GetKind() == types.MAIN_CONTEXT || ctx.GetKind() == types.EXTRAS_CONTEXT {
					ctx.HandleRender()
				}
			}
			gui.State.ContextMgr.CurrentSide().HandleRenderToMain()
			gui.State.ContextMgr.UpdateSelectionHighlights()
			if err := gui.g.ForceLayoutAndRedraw(); err != nil {
				return err
			}
			gocui.Screen.Sync()
			return nil
		},
	})
	return nil
}
