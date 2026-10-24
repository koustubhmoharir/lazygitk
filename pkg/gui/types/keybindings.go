package types

import (
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
)

// Binding - a keybinding mapping a key and modifier to a handler. The keypress
// is only handled if the given view has focus, or handled globally if the view
// is ""
type Binding struct {
	// CommandID is the stable, non-translated identity of this action.
	CommandID string
	// CommandIDFunc resolves mode-dependent identity at invocation time. Keep it
	// cheap and independent of keys, translations and selected object values.
	// CommandID remains the generic identity for static tooling.
	CommandIDFunc func() string
	// Verb describes the action's semantics independently of its target.
	Verb string
	// VerbFunc resolves the semantic verb alongside CommandIDFunc when needed.
	VerbFunc func() string
	ViewName string
	Handler  func() error
	Keys     []gocui.Key
	// CommandPaths are visible menu paths, not alternate single-key bindings.
	// Keep Keys for direct shortcuts; each path here contains at least two keys.
	CommandPaths [][]gocui.Key
	// DefaultCommand enables fork applicability/no-op behavior. Explicit key
	// overrides and custom commands retain their existing error semantics.
	DefaultCommand bool
	// CommandActive selects a semantic route of a shared upstream binding.
	// It is independent of disabled-action errors and explicit key overrides.
	CommandActive func() bool
	Description   string
	// DescriptionFunc is used instead of Description if non-nil, and is useful for dynamic
	// descriptions that change depending on context. Important: this must not be an expensive call.
	// Note that you should still provide a generic, non-dynamic description in the Description field,
	// as this is used in the cheatsheet.
	DescriptionFunc func() string
	// If defined, this is used in place of Description when showing the keybinding
	// in the options view at the bottom left of the screen.
	ShortDescription string
	// ShortDescriptionFunc is used instead of ShortDescription if non-nil, and is useful for dynamic
	// descriptions that change depending on context. Important: this must not be an expensive call.
	ShortDescriptionFunc func() string
	Alternative          string
	Tag                  string // e.g. 'navigation'. Used for grouping things in the cheatsheet
	OpensMenu            bool

	// If true, the keybinding will appear at the bottom of the screen.
	// Even if set to true, the keybinding will not be displayed if it is currently
	// disabled. We could instead display it with a strikethrough, but there's
	// limited realestate to show all the keybindings we want, so we're hiding it instead.
	DisplayOnScreen bool
	// if unset, the binding will be displayed in the default color. Only applies to the keybinding
	// on-screen, not in the keybindings menu.
	DisplayStyle *style.TextStyle

	// to be displayed if the keybinding is highlighted from within a menu
	Tooltip string
	// TooltipFunc is used instead of Tooltip if non-nil, for a command whose tooltip
	// depends on context, as DescriptionFunc is for its description — and with the
	// same two conditions: it must not be an expensive call, and a generic Tooltip
	// must still be given, since that is the one the cheatsheet prints.
	TooltipFunc func() string

	// Function to decide whether the command is enabled, and why. If this
	// returns an empty string, it is; if it returns a non-empty string, it is
	// disabled and we show the given text in an error message when trying to
	// invoke it. When left nil, the command is always enabled. Note that this
	// function must not do expensive calls.
	GetDisabledReason func() *DisabledReason
}

func (b *Binding) IsDisabled() bool {
	return b.GetDisabledReason != nil && b.GetDisabledReason() != nil
}

func (b *Binding) GetDescription() string {
	if b.DescriptionFunc != nil {
		return b.DescriptionFunc()
	}
	return b.Description
}

func (b *Binding) GetTooltip() string {
	if b.TooltipFunc != nil {
		return b.TooltipFunc()
	}
	return b.Tooltip
}

func (b *Binding) GetShortDescription() string {
	if b.ShortDescriptionFunc != nil {
		return b.ShortDescriptionFunc()
	}
	if b.ShortDescription != "" {
		return b.ShortDescription
	}
	return b.GetDescription()
}

// A guard is a decorator which checks something before executing a handler
// and potentially early-exits if some precondition hasn't been met.
type Guard func(func() error) func() error

type KeybindingGuards struct {
	OutsideFilterMode Guard
	NoPopupPanel      Guard
}

type ErrKeybindingNotHandled struct {
	DisabledReason *DisabledReason
}

func (e ErrKeybindingNotHandled) Error() string {
	return e.DisabledReason.Text
}

func (e ErrKeybindingNotHandled) Unwrap() error {
	return gocui.ErrKeybindingNotHandled
}
