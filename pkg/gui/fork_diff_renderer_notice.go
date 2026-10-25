package gui

// Fork: startup notice for the default diff renderer fallback (roadmap item 15).
// When delta is not on PATH, the fork's default renderer list drops delta and
// shows git's own diff. onInitialViewsCreation runs once per session, so the
// toast is shown once per session without further state.
func (gui *Gui) showForkDiffRendererFallbackNotice() {
	if !gui.diffRendererConfig.DefaultDiffRendererFellBack() {
		return
	}
	gui.c.Toast(gui.Tr.DiffRendererDeltaNotFound)
}
