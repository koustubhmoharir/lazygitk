package popup

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

// usageRecorder stands in for the gui hook. Like the real pending use, it hands
// out one claim and counts settlements.
type usageRecorder struct {
	claimed bool
	commits int
	drops   int
}

func (r *usageRecorder) hook() (func(), func(), bool) {
	if r.claimed {
		return nil, nil, false
	}
	r.claimed = true
	return func() { r.commits++ }, func() { r.drops++ }, true
}

// newTestPopupHandler returns a handler whose popup and menu creation just
// keep the options they receive.
func newTestPopupHandler(hook *usageRecorder) (*PopupHandler, *[]types.CreatePopupPanelOpts, *[]types.CreateMenuOptions) {
	var popups []types.CreatePopupPanelOpts
	var menus []types.CreateMenuOptions
	handler := NewPopupHandler(
		nil,
		func(_ context.Context, opts types.CreatePopupPanelOpts) { popups = append(popups, opts) },
		nil, nil, nil,
		func(opts types.CreateMenuOptions) error { menus = append(menus, opts); return nil },
		nil, nil, nil, nil, nil,
	)
	if hook != nil {
		handler.SetUsageHook(hook.hook)
	}
	return handler, &popups, &menus
}

func TestForkUsageConfirmAcceptCommitsOnce(t *testing.T) {
	hook := &usageRecorder{}
	handler, popups, _ := newTestPopupHandler(hook)
	accepted := 0
	handler.Confirm(types.ConfirmOpts{Title: "t", HandleConfirm: func() error { accepted++; return nil }})

	popup := (*popups)[0]
	assert.NoError(t, popup.HandleConfirm())
	assert.Equal(t, 1, accepted)
	assert.Equal(t, 1, hook.commits)
	assert.Equal(t, 0, hook.drops)
}

func TestForkUsageConfirmCancelDrops(t *testing.T) {
	hook := &usageRecorder{}
	handler, popups, _ := newTestPopupHandler(hook)
	closed := 0
	handler.Confirm(types.ConfirmOpts{Title: "t", HandleClose: func() error { closed++; return nil }})

	assert.NoError(t, (*popups)[0].HandleClose())
	assert.Equal(t, 1, closed)
	assert.Equal(t, 0, hook.commits)
	assert.Equal(t, 1, hook.drops)
}

func TestForkUsageConfirmWithoutUpstreamCallbackStillCommits(t *testing.T) {
	hook := &usageRecorder{}
	handler, popups, _ := newTestPopupHandler(hook)
	handler.Confirm(types.ConfirmOpts{Title: "alert"})

	assert.NoError(t, (*popups)[0].HandleConfirm())
	assert.NoError(t, (*popups)[0].HandleClose())
	assert.Equal(t, 1, hook.commits)
	assert.Equal(t, 1, hook.drops)
}

func TestForkUsageConfirmIfTrueClaims(t *testing.T) {
	hook := &usageRecorder{}
	handler, popups, _ := newTestPopupHandler(hook)
	assert.NoError(t, handler.ConfirmIf(true, types.ConfirmOpts{Title: "t"}))

	assert.Len(t, *popups, 1)
	assert.NoError(t, (*popups)[0].HandleConfirm())
	assert.Equal(t, 1, hook.commits)
}

func TestForkUsageConfirmIfFalseRunsHandlerWithoutClaiming(t *testing.T) {
	hook := &usageRecorder{}
	handler, popups, _ := newTestPopupHandler(hook)
	ran := false
	assert.NoError(t, handler.ConfirmIf(false, types.ConfirmOpts{HandleConfirm: func() error { ran = true; return nil }}))

	assert.Empty(t, *popups)
	assert.True(t, ran)
	assert.False(t, hook.claimed)
}

func TestForkUsagePromptConfirmCommitsWithContent(t *testing.T) {
	hook := &usageRecorder{}
	handler, popups, _ := newTestPopupHandler(hook)
	var got string
	handler.Prompt(types.PromptOpts{Title: "t", HandleConfirm: func(s string) error { got = s; return nil }})

	assert.NoError(t, (*popups)[0].HandleConfirmPrompt("typed"))
	assert.Equal(t, "typed", got)
	assert.Equal(t, 1, hook.commits)
	assert.Equal(t, 0, hook.drops)
}

func TestForkUsagePromptCloseDrops(t *testing.T) {
	hook := &usageRecorder{}
	handler, popups, _ := newTestPopupHandler(hook)
	handler.Prompt(types.PromptOpts{Title: "t"})

	assert.NoError(t, (*popups)[0].HandleClose())
	assert.Equal(t, 0, hook.commits)
	assert.Equal(t, 1, hook.drops)
}

func TestForkUsageMenuItemPressCommitsAndRunsItem(t *testing.T) {
	hook := &usageRecorder{}
	handler, _, menus := newTestPopupHandler(hook)
	pressed := 0
	assert.NoError(t, handler.Menu(types.CreateMenuOptions{Items: []*types.MenuItem{
		{Label: "first", OnPress: func() error { pressed++; return nil }},
		{Label: "no-op"},
	}}))

	items := (*menus)[0].Items
	assert.Len(t, items, 2)
	assert.NoError(t, items[0].OnPress())
	assert.Equal(t, 1, pressed)
	assert.Equal(t, 1, hook.commits)
	assert.Equal(t, 0, hook.drops)
	assert.Nil(t, items[1].OnPress, "items without a callback are left alone")
}

func TestForkUsageMenuItemPressPropagatesError(t *testing.T) {
	hook := &usageRecorder{}
	handler, _, menus := newTestPopupHandler(hook)
	failure := errors.New("boom")
	assert.NoError(t, handler.Menu(types.CreateMenuOptions{Items: []*types.MenuItem{{OnPress: func() error { return failure }}}}))

	assert.ErrorIs(t, (*menus)[0].Items[0].OnPress(), failure)
	assert.Equal(t, 1, hook.commits)
}

func TestForkUsageMenuCancelDrops(t *testing.T) {
	hook := &usageRecorder{}
	handler, _, menus := newTestPopupHandler(hook)
	cancelled := 0
	assert.NoError(t, handler.Menu(types.CreateMenuOptions{
		Items:    []*types.MenuItem{{OnPress: func() error { return nil }}},
		OnCancel: func() error { cancelled++; return nil },
	}))

	assert.NoError(t, (*menus)[0].OnCancel())
	assert.Equal(t, 1, cancelled)
	assert.Equal(t, 0, hook.commits)
	assert.Equal(t, 1, hook.drops)
}

func TestForkUsageMenuDoesNotRewrapCallerItems(t *testing.T) {
	hook := &usageRecorder{}
	handler, _, menus := newTestPopupHandler(hook)
	item := &types.MenuItem{Label: "reused", OnPress: func() error { return nil }}
	original := item.OnPress
	assert.NoError(t, handler.Menu(types.CreateMenuOptions{Items: []*types.MenuItem{item}}))

	assert.NotSame(t, item, (*menus)[0].Items[0])
	assert.Equal(t, reflectPointer(original), reflectPointer(item.OnPress))
}

func TestForkUsageWithoutHookLeavesOptionsUnchanged(t *testing.T) {
	handler, popups, menus := newTestPopupHandler(nil)
	item := &types.MenuItem{Label: "item", OnPress: func() error { return nil }}
	assert.NoError(t, handler.Menu(types.CreateMenuOptions{Items: []*types.MenuItem{item}}))
	handler.Confirm(types.ConfirmOpts{HandleConfirm: func() error { return nil }})

	assert.Same(t, item, (*menus)[0].Items[0])
	assert.Nil(t, (*menus)[0].OnCancel)
	assert.Nil(t, (*popups)[0].HandleClose)
}

func TestForkUsageHookWithNothingPendingLeavesOptionsUnchanged(t *testing.T) {
	hook := &usageRecorder{claimed: true} // hook reports no pending use
	handler, popups, _ := newTestPopupHandler(hook)
	handler.Confirm(types.ConfirmOpts{HandleConfirm: nil})

	assert.Nil(t, (*popups)[0].HandleConfirm)
	assert.Equal(t, 0, hook.commits+hook.drops)
}

// reflectPointer identifies a func value by its code pointer, so tests can tell
// an unwrapped callback from a wrapper. It is only used for equality checks.
func reflectPointer(f func() error) uintptr {
	return reflect.ValueOf(f).Pointer()
}
