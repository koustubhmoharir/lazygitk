package helpers

// SetOnOperationFinished sets the callback that runs once the operation in progress
// ends by continue or skip. A later call replaces an earlier one. Call it on the UI
// thread.
func (self *MergeAndRebaseHelper) SetOnOperationFinished(callback func() error) {
	self.onOperationFinished = callback
}

// settleOperationFinished runs after a continue, skip or abort. Once the operation
// has ended, a continue or skip that succeeded runs the callback that a pick stopped
// on conflicts registered. An abort drops that callback without running it, so the
// pick's selection stays. A step that leaves the operation in progress, such as a
// continue that stops on another conflict, keeps the callback.
func (self *MergeAndRebaseHelper) settleOperationFinished(command string, result error) {
	ended := self.c.Git().Status.WorkingTreeState().None()
	if command == REBASE_OPTION_ABORT {
		if ended {
			self.c.OnUIThread(func() error {
				self.onOperationFinished = nil
				return nil
			})
		}
		return
	}
	if result != nil || !ended {
		return
	}
	self.c.OnUIThread(func() error {
		callback := self.onOperationFinished
		self.onOperationFinished = nil
		if callback == nil {
			return nil
		}
		return callback()
	})
}
