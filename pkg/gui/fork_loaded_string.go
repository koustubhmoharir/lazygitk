package gui

import (
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/tasks"
)

// Fork: renders a types.LoadedStringTask. It is a string render whose text is produced
// by the task's goroutine, as the folder listing and submodule views need git for it.
// The view's title and content are set on the UI thread once the text is ready, as a
// string render sets its content.
func (gui *Gui) newLoadedStringTask(view *gocui.View, task *types.LoadedStringTask) error {
	manager := gui.getManager(view)
	// Whatever the view was going to be put back to belonged to a re-render of its
	// content; this is a message instead, so there is nothing to put back.
	manager.DropRestoreForNextTask()

	f := func(opts tasks.TaskOpts) error {
		title, text := task.Load()
		// A newer render asked for the view while this one loaded; its text is stale.
		select {
		case <-opts.Stop:
			return nil
		default:
		}
		return gui.g.OnUIThreadAndWaitBackground(func() {
			view.Title = title
			gui.c.ResetViewOrigin(view)
			gui.c.SetViewContent(view, text)
			gui.updateDiffPaneDecorations(view, true)
			gui.reApplySearch(view)
		})
	}

	return manager.NewTask(f, task.Key)
}
