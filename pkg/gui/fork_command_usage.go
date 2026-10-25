package gui

import (
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commandusage"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: command usage counts (roadmap item 9). A shortcut or prefix-menu leaf
// subtracts from its command's count; the `?` palette adds to it. A use counts
// only once its action is accepted: while the handler runs it is pending, and a
// popup opened through PopupHandler claims it. See fork_usage_hook.go.

const (
	forkShortcutUsageDelta = -1
	forkPaletteUsageDelta  = 1
	forkUsageFlushDelay    = 2 * time.Second
)

// forkUsageRecordable reports whether invoking binding counts as a command use.
// Navigation, the palette opener itself, unnamed bindings and bindings in
// modal or popup views are excluded.
func forkUsageRecordable(binding *types.Binding) bool {
	id := binding.GetCommandID()
	if id == "" || id == "app.command-palette" {
		return false
	}
	if strings.HasPrefix(id, "navigation.") || binding.Tag == "navigation" {
		return false
	}
	return forkCommandView(binding.ViewName)
}

// forkUsageTracker holds the uses whose handlers are running on the UI thread.
// Popups may be opened from worker goroutines too, so access is locked.
type forkUsageTracker struct {
	record func(id string, delta int)

	mu sync.Mutex
	// stack holds running uses, innermost last.
	stack []*forkPendingUsage
}

// forkPendingUsage is one accepted-or-not command use. The first popup that
// claims it decides its fate; settling happens at most once.
type forkPendingUsage struct {
	tracker  *forkUsageTracker
	id       string
	delta    int
	attached bool // guarded by tracker.mu
	settled  sync.Once
}

func newForkUsageTracker(record func(id string, delta int)) *forkUsageTracker {
	return &forkUsageTracker{record: record}
}

// begin marks binding's use as pending while its handler runs. It returns nil
// for uses that are not recorded.
func (t *forkUsageTracker) begin(binding *types.Binding, delta int) *forkPendingUsage {
	if !forkUsageRecordable(binding) {
		return nil
	}
	pending := &forkPendingUsage{tracker: t, id: binding.GetCommandID(), delta: delta}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stack = append(t.stack, pending)
	return pending
}

// end is called when the handler has returned, whatever its error. A use that
// no popup claimed is committed now.
func (t *forkUsageTracker) end(pending *forkPendingUsage) {
	if pending == nil {
		return
	}
	t.mu.Lock()
	t.stack = slices.DeleteFunc(t.stack, func(p *forkPendingUsage) bool { return p == pending })
	claimed := pending.claimLocked()
	t.mu.Unlock()

	if claimed {
		pending.commit()
	}
}

// attach claims the innermost running use that no popup has claimed yet.
// The returned functions settle it: commit records the use, drop discards it.
func (t *forkUsageTracker) attach() (commit func(), drop func(), ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := len(t.stack) - 1; i >= 0; i-- {
		if pending := t.stack[i]; pending.claimLocked() {
			return pending.commit, pending.drop, true
		}
	}
	return nil, nil, false
}

// claimLocked marks the use as attached. It reports false if it already was.
// The caller holds tracker.mu.
func (p *forkPendingUsage) claimLocked() bool {
	if p.attached {
		return false
	}
	p.attached = true
	return true
}

func (p *forkPendingUsage) commit() {
	p.settled.Do(func() { p.tracker.record(p.id, p.delta) })
}

func (p *forkPendingUsage) drop() {
	p.settled.Do(func() {})
}

// Fork: opens the shared counts file (see pkg/commandusage). If the path cannot
// be resolved, recording is disabled for this session; the error is logged.
func (gui *Gui) openForkCommandUsage() {
	path, err := config.CommandUsageFilePath()
	if err != nil {
		gui.c.Log.Errorf("command usage counts are disabled: %v", err)
		return
	}
	store, reset, err := commandusage.Open(path)
	if err != nil {
		gui.c.Log.Errorf("could not read command usage counts: %v", err)
	}
	gui.commandUsage = store
	gui.forkUsageReset = reset
}

// recordForkUsage adds a committed use to the store and arms the flush timer.
func (gui *Gui) recordForkUsage(id string, delta int) {
	if gui.commandUsage == nil {
		return
	}
	gui.commandUsage.Add(id, delta)

	gui.forkUsageFlushMu.Lock()
	defer gui.forkUsageFlushMu.Unlock()
	if gui.forkUsageFlushTimer == nil {
		gui.forkUsageFlushTimer = time.AfterFunc(forkUsageFlushDelay, gui.flushForkUsageInBackground)
	}
}

func (gui *Gui) flushForkUsageInBackground() {
	gui.forkUsageFlushMu.Lock()
	gui.forkUsageFlushTimer = nil
	gui.forkUsageFlushMu.Unlock()

	if gui.flushForkUsage() {
		gui.onUIThread(func() error {
			gui.c.Toast(gui.Tr.CommandUsageReset)
			return nil
		})
	}
}

// flushForkUsage writes pending counts to disk and reports whether a corrupt
// counts file was replaced.
func (gui *Gui) flushForkUsage() (reset bool) {
	if gui.commandUsage == nil {
		return false
	}
	reset, err := gui.commandUsage.Flush()
	if err != nil {
		gui.c.Log.Errorf("could not save command usage counts: %v", err)
	}
	if reset {
		gui.c.Log.Warn("command usage counts file was unreadable and has been reset")
	}
	return reset
}

// Fork: called from Run after the main loop exits, so counts recorded since the
// last timer flush are not lost.
func (gui *Gui) stopForkCommandUsage() {
	gui.forkUsageFlushMu.Lock()
	if gui.forkUsageFlushTimer != nil {
		gui.forkUsageFlushTimer.Stop()
		gui.forkUsageFlushTimer = nil
	}
	gui.forkUsageFlushMu.Unlock()

	gui.flushForkUsage()
}

// attachForkUsage is the PopupHandler hook: a popup opened while a command
// runs claims that command's use.
func (gui *Gui) attachForkUsage() (commit func(), drop func(), ok bool) {
	return gui.forkUsage.attach()
}
