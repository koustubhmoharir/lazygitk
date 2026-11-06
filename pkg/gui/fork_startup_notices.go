package gui

import (
	"time"

	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: startup notices are shown one after another. The status manager shows
// only the newest toast, and each toast expires two seconds after it was
// added, so notices added together would cover one another. The queue holds
// each notice until the previous one's visible time has ended.

// forkNoticeDuration is how long one notice stays visible. The status manager
// gives an info toast two seconds, but that value is inline in
// AddToastStatus (pkg/gui/status/status_manager.go) with no named constant to
// reference, so it is repeated here and must match.
const forkNoticeDuration = 2 * time.Second

// forkNoticeQueue holds notices waiting to be shown, in the order they were
// queued. Only the UI thread may call its methods.
type forkNoticeQueue struct {
	pending []string
	showing bool

	// show displays one notice as a toast.
	show func(message string)
	// after calls f on the UI thread once d has passed.
	after func(d time.Duration, f func())
}

// enqueue adds notices to the end of the queue. If nothing is showing, the
// first notice is shown now.
func (q *forkNoticeQueue) enqueue(messages ...string) {
	if len(messages) == 0 {
		return
	}
	q.pending = append(q.pending, messages...)
	if !q.showing {
		q.showNext()
	}
}

// showNext shows the next pending notice for its full duration. When the
// queue is empty, it stops.
func (q *forkNoticeQueue) showNext() {
	if len(q.pending) == 0 {
		q.showing = false
		return
	}
	message := q.pending[0]
	q.pending = q.pending[1:]
	q.showing = true
	q.show(message)
	q.after(forkNoticeDuration, q.showNext)
}

// enqueueForkNotices queues notices to be shown one after another.
func (gui *Gui) enqueueForkNotices(messages ...string) {
	if gui.forkNotices == nil {
		gui.forkNotices = &forkNoticeQueue{
			show: func(message string) { gui.c.Toast(message) },
			after: func(d time.Duration, f func()) {
				// Fork: the timer runs off the UI thread, so the next notice is
				// shown there, as current toasts are.
				time.AfterFunc(d, utils.SafeFunc(func() {
					gui.onUIThread(func() error {
						f()
						return nil
					})
				}))
			},
		}
	}
	gui.forkNotices.enqueue(messages...)
}

// queueForkUsageResetNotice queues the notice for a command usage file that
// was unreadable at startup. It is queued once.
func (gui *Gui) queueForkUsageResetNotice() {
	if !gui.forkUsageReset {
		return
	}
	gui.forkUsageReset = false
	gui.enqueueForkNotices(gui.Tr.CommandUsageReset)
}
