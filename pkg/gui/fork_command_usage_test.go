package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

func TestForkUsageRecordable(t *testing.T) {
	tests := []struct {
		name    string
		binding *types.Binding
		want    bool
	}{
		{"global command", &types.Binding{CommandID: "app.quit"}, true},
		{"fork view command", &types.Binding{CommandID: "files.stage", ViewName: "files"}, true},
		{"empty command ID", &types.Binding{}, false},
		{"command palette opener", &types.Binding{CommandID: "app.command-palette"}, false},
		{"palette reset command", &types.Binding{CommandID: "app.reset-command-usage"}, false},
		{"navigation prefix", &types.Binding{CommandID: "navigation.move-down"}, false},
		{"navigation tag", &types.Binding{CommandID: "files.move", Tag: "navigation"}, false},
		{"modal view", &types.Binding{CommandID: "menu.cancel", ViewName: "menu"}, false},
		{"dynamic command ID", &types.Binding{CommandIDFunc: func() string { return "" }}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, forkUsageRecordable(tt.binding))
		})
	}
}

func TestForkUsageRecordableUsesDynamicCommandID(t *testing.T) {
	binding := &types.Binding{CommandID: "app.command-palette", CommandIDFunc: func() string { return "files.stage" }}
	assert.True(t, forkUsageRecordable(binding))
}

type forkUsageRecord struct {
	id    string
	delta int
}

func newRecordingForkUsage() (*forkUsageTracker, *[]forkUsageRecord) {
	var records []forkUsageRecord
	tracker := newForkUsageTracker(func(id string, delta int) {
		records = append(records, forkUsageRecord{id, delta})
	})
	return tracker, &records
}

func TestForkUsageCommitsWhenHandlerReturnsWithoutPopup(t *testing.T) {
	tracker, records := newRecordingForkUsage()
	binding := &types.Binding{CommandID: "files.stage"}

	pending := tracker.begin(binding, forkShortcutUsageDelta)
	assert.Empty(t, *records, "the use is pending while its handler runs")
	tracker.end(pending)

	assert.Equal(t, []forkUsageRecord{{"files.stage", -1}}, *records)
}

func TestForkUsagePaletteUseAddsToCount(t *testing.T) {
	tracker, records := newRecordingForkUsage()
	pending := tracker.begin(&types.Binding{CommandID: "files.stage"}, forkPaletteUsageDelta)
	tracker.end(pending)

	assert.Equal(t, []forkUsageRecord{{"files.stage", 1}}, *records)
}

func TestForkUsageCommitsAfterErrorToo(t *testing.T) {
	tracker, records := newRecordingForkUsage()
	// The handler's error does not reach the tracker; end runs regardless.
	pending := tracker.begin(&types.Binding{CommandID: "files.stage"}, forkShortcutUsageDelta)
	tracker.end(pending)

	assert.Len(t, *records, 1)
}

func TestForkUsageAttachedPopupDefersCommitToAcceptance(t *testing.T) {
	tracker, records := newRecordingForkUsage()
	pending := tracker.begin(&types.Binding{CommandID: "files.stage"}, forkShortcutUsageDelta)

	commit, drop, ok := tracker.attach()
	assert.True(t, ok)
	tracker.end(pending)
	assert.Empty(t, *records, "an attached use is not committed when its handler returns")

	commit()
	assert.Equal(t, []forkUsageRecord{{"files.stage", -1}}, *records)
	drop()
	commit()
	assert.Len(t, *records, 1, "a use settles at most once")
}

func TestForkUsageDroppedPopupRecordsNothing(t *testing.T) {
	tracker, records := newRecordingForkUsage()
	pending := tracker.begin(&types.Binding{CommandID: "files.stage"}, forkShortcutUsageDelta)
	_, drop, ok := tracker.attach()
	assert.True(t, ok)
	tracker.end(pending)

	drop()
	assert.Empty(t, *records)
}

func TestForkUsageSecondPopupIsNotClaimed(t *testing.T) {
	tracker, _ := newRecordingForkUsage()
	pending := tracker.begin(&types.Binding{CommandID: "files.stage"}, forkShortcutUsageDelta)
	defer tracker.end(pending)

	_, _, ok := tracker.attach()
	assert.True(t, ok)
	_, _, ok = tracker.attach()
	assert.False(t, ok, "the first popup claims the use")
}

func TestForkUsageAttachWithoutPendingUseClaimsNothing(t *testing.T) {
	tracker, records := newRecordingForkUsage()
	_, _, ok := tracker.attach()
	assert.False(t, ok)
	assert.Empty(t, *records)
}

func TestForkUsageNestedUseClaimsInnermostPopup(t *testing.T) {
	tracker, records := newRecordingForkUsage()
	outer := tracker.begin(&types.Binding{CommandID: "outer.command"}, forkShortcutUsageDelta)
	inner := tracker.begin(&types.Binding{CommandID: "inner.command"}, forkShortcutUsageDelta)

	innerCommit, _, ok := tracker.attach()
	assert.True(t, ok)
	tracker.end(inner)
	assert.Empty(t, *records, "the innermost use waits for its popup")

	tracker.end(outer)
	assert.Equal(t, []forkUsageRecord{{"outer.command", -1}}, *records, "the outer use has no popup, so it commits on return")

	innerCommit()
	assert.Equal(t, []forkUsageRecord{{"outer.command", -1}, {"inner.command", -1}}, *records)
}

func TestForkUsageIgnoresUnrecordableBindings(t *testing.T) {
	tracker, records := newRecordingForkUsage()
	pending := tracker.begin(&types.Binding{CommandID: "app.command-palette"}, forkShortcutUsageDelta)
	assert.Nil(t, pending)
	tracker.end(pending)

	_, _, ok := tracker.attach()
	assert.False(t, ok)
	assert.Empty(t, *records)
}
