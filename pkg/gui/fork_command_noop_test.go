package gui

import (
	"errors"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

func TestForkDefaultInapplicableConsumesWithoutHandlerOrFallback(t *testing.T) {
	for _, reason := range []*types.DisabledReason{{Text: "no target", Inapplicable: true}, {Text: "no target", AllowFurtherDispatching: true}} {
		gui := &Gui{}
		called := false
		binding := &types.Binding{DefaultCommand: true, GetDisabledReason: func() *types.DisabledReason { return reason }, Handler: func() error { called = true; return nil }}
		assert.NoError(t, gui.callKeybindingHandler(binding))
		assert.False(t, called)
	}
}

func TestForkExplicitOverrideRetainsDisabledDispatchAndRealErrors(t *testing.T) {
	gui := &Gui{}
	binding := &types.Binding{GetDisabledReason: func() *types.DisabledReason {
		return &types.DisabledReason{Text: "not handled", AllowFurtherDispatching: true}
	}}
	assert.ErrorIs(t, gui.callKeybindingHandler(binding), gocui.ErrKeybindingNotHandled)
	binding.DefaultCommand = true
	binding.GetDisabledReason = func() *types.DisabledReason {
		return &types.DisabledReason{Text: "dirty working tree", ShowErrorInPanel: true}
	}
	assert.EqualError(t, gui.callKeybindingHandler(binding), "dirty working tree")
	binding.GetDisabledReason = nil
	failure := errors.New("Git rejected the operation")
	binding.Handler = func() error { return failure }
	assert.ErrorIs(t, gui.callKeybindingHandler(binding), failure)
}
