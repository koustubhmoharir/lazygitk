package gui

// Fork: startup notice for the default diff renderer fallback (roadmap item 15).
// When delta is not on PATH, the fork's default renderer list drops delta and
// shows git's own diff. The notice is queued once per session.
func (gui *Gui) queueForkDiffRendererFallbackNotice() {
	if gui.forkDiffRendererNoticeQueued || !gui.diffRendererConfig.DefaultDiffRendererFellBack() {
		return
	}
	gui.forkDiffRendererNoticeQueued = true
	gui.enqueueForkNotices(gui.Tr.DiffRendererDeltaNotFound)
}
