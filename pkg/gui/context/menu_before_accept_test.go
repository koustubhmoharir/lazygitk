package context

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

type menuBeforeAcceptGuiFake struct {
	types.IGuiCommon
	contextMgr *menuBeforeAcceptContextMgrFake
}

func (f *menuBeforeAcceptGuiFake) Context() types.IContextMgr { return f.contextMgr }

type menuBeforeAcceptContextMgrFake struct {
	types.IContextMgr
	popCount int
}

func (f *menuBeforeAcceptContextMgrFake) Pop() { f.popCount++ }

func newMenuBeforeAcceptContext() (*MenuContext, *menuBeforeAcceptContextMgrFake) {
	contextMgr := &menuBeforeAcceptContextMgrFake{}
	common := &ContextCommon{IGuiCommon: &menuBeforeAcceptGuiFake{contextMgr: contextMgr}}
	viewModel := NewMenuViewModel(common)
	return &MenuContext{c: common, MenuViewModel: viewModel}, contextMgr
}

func TestMenuBeforeAcceptFalseConsumesItemAndCancel(t *testing.T) {
	for _, selectedItem := range []*types.MenuItem{{OnPress: func() error { t.Error("item callback ran"); return nil }}, nil} {
		t.Run(map[bool]string{true: "item", false: "cancel"}[selectedItem != nil], func(t *testing.T) {
			menu, contextMgr := newMenuBeforeAcceptContext()
			hookCalls, callbackCalls := 0, 0
			menu.SetBeforeAccept(func() bool { hookCalls++; return false })
			menu.SetOnCancel(func() error { callbackCalls++; return nil })

			if err := menu.OnMenuPress(selectedItem); err != nil {
				t.Fatalf("OnMenuPress returned error: %v", err)
			}
			assert.Equal(t, 1, hookCalls)
			assert.Zero(t, contextMgr.popCount)
			assert.Zero(t, callbackCalls)
		})
	}
}

func TestMenuBeforeAcceptTrueRunsBeforePopAndCallbackAfter(t *testing.T) {
	menu, contextMgr := newMenuBeforeAcceptContext()
	hookSawPopCount, callbackSawPopCount := -1, -1
	menu.SetBeforeAccept(func() bool { hookSawPopCount = contextMgr.popCount; return true })
	item := &types.MenuItem{OnPress: func() error { callbackSawPopCount = contextMgr.popCount; return nil }}

	if err := menu.OnMenuPress(item); err != nil {
		t.Fatalf("OnMenuPress returned error: %v", err)
	}
	assert.Zero(t, hookSawPopCount)
	assert.Equal(t, 1, contextMgr.popCount)
	assert.Equal(t, 1, callbackSawPopCount)
}

func TestMenuBeforeAcceptDisabledItemDoesNotCallHook(t *testing.T) {
	menu, contextMgr := newMenuBeforeAcceptContext()
	hookCalls := 0
	menu.SetBeforeAccept(func() bool { hookCalls++; return true })
	item := &types.MenuItem{DisabledReason: &types.DisabledReason{Text: "disabled", ShowErrorInPanel: true}}

	err := menu.OnMenuPress(item)
	assert.EqualError(t, err, "disabled")
	assert.Zero(t, hookCalls)
	assert.Zero(t, contextMgr.popCount)
}

func TestMenuBeforeAcceptNilReplacementClearsCallback(t *testing.T) {
	menu, contextMgr := newMenuBeforeAcceptContext()
	hookCalls := 0
	menu.SetBeforeAccept(func() bool { hookCalls++; return false })
	menu.SetBeforeAccept(nil)

	if err := menu.OnMenuPress(&types.MenuItem{OnPress: func() error { return nil }}); err != nil {
		t.Fatalf("OnMenuPress returned error: %v", err)
	}
	assert.Zero(t, hookCalls)
	assert.Equal(t, 1, contextMgr.popCount)
}
