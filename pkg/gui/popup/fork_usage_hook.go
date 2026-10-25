package popup

import "github.com/jesseduffield/lazygit/pkg/gui/types"

// Fork: command usage counts (roadmap item 9). A popup opened while a command
// runs claims that command's use: accepting the popup commits it, and closing
// or cancelling it drops it. The hook is nil when usage tracking is absent, and
// popups then behave exactly as upstream.

// SetUsageHook installs the claim function. It returns ok=false when no use is
// pending or the pending use was already claimed by an earlier popup.
func (self *PopupHandler) SetUsageHook(hook func() (commit func(), drop func(), ok bool)) {
	self.usageHookFn = hook
}

func (self *PopupHandler) forkConfirmUsage(opts types.ConfirmOpts) types.ConfirmOpts {
	commit, drop, ok := self.claimUsage()
	if !ok {
		return opts
	}
	return forkUsageConfirmOpts(opts, commit, drop)
}

func (self *PopupHandler) forkPromptUsage(opts types.PromptOpts) types.PromptOpts {
	commit, drop, ok := self.claimUsage()
	if !ok {
		return opts
	}
	return forkUsagePromptOpts(opts, commit, drop)
}

func (self *PopupHandler) forkMenuUsage(opts types.CreateMenuOptions) types.CreateMenuOptions {
	commit, drop, ok := self.claimUsage()
	if !ok {
		return opts
	}
	return forkUsageMenuOpts(opts, commit, drop)
}

func (self *PopupHandler) claimUsage() (commit func(), drop func(), ok bool) {
	if self.usageHookFn == nil {
		return nil, nil, false
	}
	return self.usageHookFn()
}

// forkUsageConfirmOpts commits on confirmation and drops on close. A missing
// upstream callback is still wrapped, because confirming accepts the use.
func forkUsageConfirmOpts(opts types.ConfirmOpts, commit func(), drop func()) types.ConfirmOpts {
	handleConfirm := opts.HandleConfirm
	opts.HandleConfirm = func() error {
		commit()
		if handleConfirm == nil {
			return nil
		}
		return handleConfirm()
	}
	handleClose := opts.HandleClose
	opts.HandleClose = func() error {
		drop()
		if handleClose == nil {
			return nil
		}
		return handleClose()
	}
	return opts
}

func forkUsagePromptOpts(opts types.PromptOpts, commit func(), drop func()) types.PromptOpts {
	handleConfirm := opts.HandleConfirm
	opts.HandleConfirm = func(content string) error {
		commit()
		if handleConfirm == nil {
			return nil
		}
		return handleConfirm(content)
	}
	handleClose := opts.HandleClose
	opts.HandleClose = func() error {
		drop()
		if handleClose == nil {
			return nil
		}
		return handleClose()
	}
	return opts
}

// forkUsageMenuOpts commits when an item with an OnPress is pressed and drops on
// cancel. Items are copied, so the caller's items are not rewrapped if reused.
func forkUsageMenuOpts(opts types.CreateMenuOptions, commit func(), drop func()) types.CreateMenuOptions {
	items := make([]*types.MenuItem, len(opts.Items))
	for i, item := range opts.Items {
		wrapped := *item
		if onPress := item.OnPress; onPress != nil {
			wrapped.OnPress = func() error {
				commit()
				return onPress()
			}
		}
		items[i] = &wrapped
	}
	opts.Items = items

	onCancel := opts.OnCancel
	opts.OnCancel = func() error {
		drop()
		if onCancel == nil {
			return nil
		}
		return onCancel()
	}
	return opts
}
