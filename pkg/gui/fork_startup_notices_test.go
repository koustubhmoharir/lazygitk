package gui

import (
	"testing"
	"time"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

// fakeNoticeClock stands in for the toast display and the timers: a toast is
// recorded when shown, and fireNext plays the part of its visible time ending.
type fakeNoticeClock struct {
	toasts    []string
	scheduled []time.Duration
	timers    []func()
}

func newTestNoticeQueue() (*forkNoticeQueue, *fakeNoticeClock) {
	clock := &fakeNoticeClock{}
	queue := &forkNoticeQueue{
		show: func(message string) { clock.toasts = append(clock.toasts, message) },
		after: func(d time.Duration, f func()) {
			clock.scheduled = append(clock.scheduled, d)
			clock.timers = append(clock.timers, f)
		},
	}
	return queue, clock
}

// fireNext ends the visible time of the toast on screen.
func (c *fakeNoticeClock) fireNext() {
	f := c.timers[0]
	c.timers = c.timers[1:]
	f()
}

func TestForkNoticeQueueShowsNoticesInOrderForEachDuration(t *testing.T) {
	queue, clock := newTestNoticeQueue()

	queue.enqueue("usage", "ignored", "delta")

	// Only the first notice is shown; the others wait for its time to end.
	assert.Equal(t, []string{"usage"}, clock.toasts)
	assert.Len(t, clock.timers, 1)

	clock.fireNext()
	assert.Equal(t, []string{"usage", "ignored"}, clock.toasts)

	clock.fireNext()
	assert.Equal(t, []string{"usage", "ignored", "delta"}, clock.toasts)

	// The last one's time ends with nothing left to show.
	clock.fireNext()
	assert.Equal(t, []string{"usage", "ignored", "delta"}, clock.toasts)
	assert.Empty(t, clock.timers)

	// Each notice gets its own full duration, starting when it is shown.
	assert.Equal(t, []time.Duration{forkNoticeDuration, forkNoticeDuration, forkNoticeDuration}, clock.scheduled)
}

func TestForkNoticeQueueAppendsNoticeAddedWhileShowing(t *testing.T) {
	queue, clock := newTestNoticeQueue()

	queue.enqueue("first", "second")
	queue.enqueue("third")
	assert.Equal(t, []string{"first"}, clock.toasts)

	clock.fireNext()
	assert.Equal(t, []string{"first", "second"}, clock.toasts)

	clock.fireNext()
	assert.Equal(t, []string{"first", "second", "third"}, clock.toasts)

	// Once the queue has drained, a new notice is shown straight away.
	clock.fireNext()
	queue.enqueue("fourth")
	assert.Equal(t, []string{"first", "second", "third", "fourth"}, clock.toasts)
}

func TestForkIgnoredKeyNoticesAreMarkedShownWhenQueued(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	settings := []config.IgnoredKeySetting{
		{Path: "/global/config.yml", Kinds: []string{"keybinding"}},
		{Path: "/repo/.lazygitk.yml", Kinds: []string{"commandBindings"}},
	}
	shown := map[string]bool{}

	messages := forkIgnoredKeyConfigNotices(shown, settings, tr)

	assert.Equal(t, []string{
		"Key settings are fixed in lazygitk; ignored keybinding in /global/config.yml",
		"Key settings are fixed in lazygitk; ignored commandBindings in /repo/.lazygitk.yml",
	}, messages)
	for _, setting := range settings {
		assert.True(t, shown[forkIgnoredKeyConfigNoticeKey(setting)])
	}

	// A file already marked shown is not queued again.
	assert.Empty(t, forkIgnoredKeyConfigNotices(shown, settings, tr))
}
