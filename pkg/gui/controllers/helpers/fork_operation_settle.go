package helpers

// An operation-finished step is registered by a pick that stopped on conflicts, to
// run once the operation it left has ended. Its lifecycle:
//
//   - Registration: SetOnOperationFinished stores the step on the UI thread and
//     stamps it with a new epoch.
//   - Continue or skip: settleOperationFinished runs the step once a continue or
//     skip has ended the operation without an error.
//   - Abort: settleOperationFinished drops the step without running it, so the
//     pick's selection stays.
//   - Ended outside lazygitk: no lazygitk action settles the step. The files
//     refresh sees that no operation is in progress and calls
//     DropAbandonedOperationStep, which drops the step and runs its onAbandoned
//     notice instead. The step is never run then: the user finished the operation
//     by hand, and the work it would do was not asked for by that finish.
//
// The refresh drops the step only when two guards allow it:
//
//   - In-flight: operationsInFlight counts the continue, skip and abort runs that
//     have started, and is decremented by the UI callback that settleOperationFinished
//     always enqueues. While a run is open, the operation may still be ending through
//     lazygitk, so the step must be kept for that run to settle.
//   - Epoch: the refresh loads OperationFinishedEpoch before it reads the working
//     tree state. A step registered after that load has a newer epoch, so the
//     refresh's "no operation" reading predates the step and says nothing about it.

// SetOnOperationFinished sets the step that runs once the operation in progress ends
// by continue or skip. onAbandoned runs instead if the operation ends outside
// lazygitk. A later call replaces an earlier one. Call it on the UI thread.
func (self *MergeAndRebaseHelper) SetOnOperationFinished(callback func() error, onAbandoned func()) {
	self.onOperationFinished = callback
	self.onOperationFinishedAbandoned = onAbandoned
	self.onOperationFinishedEpoch = self.operationFinishedEpochs.Add(1)
}

// OperationFinishedEpoch returns the epoch of the latest registration. The files
// refresh calls it before reading the working tree state, from a worker. The refresh
// passes the value it loaded to DropAbandonedOperationStep.
func (self *MergeAndRebaseHelper) OperationFinishedEpoch() int64 {
	return self.operationFinishedEpochs.Load()
}

// DropAbandonedOperationStep runs on the UI thread from the files refresh, after that
// refresh found no operation in progress. epoch is the value OperationFinishedEpoch
// returned before the refresh read the working tree state. If a step is pending, was
// registered no later than that read, and no continue, skip or abort is in flight,
// the step is dropped without running and its onAbandoned notice runs. Otherwise
// nothing changes.
func (self *MergeAndRebaseHelper) DropAbandonedOperationStep(epoch int64) {
	if self.onOperationFinished == nil || self.onOperationFinishedEpoch > epoch || self.operationsInFlight.Load() != 0 {
		return
	}
	onAbandoned := self.onOperationFinishedAbandoned
	self.clearOperationFinished()
	if onAbandoned != nil {
		onAbandoned()
	}
}

// settleOperationFinished runs after a continue, skip or abort. It always enqueues
// exactly one UI callback, which releases the in-flight count taken in
// genericMergeCommandImpl. Once the operation has ended, a continue or skip that
// succeeded runs the step. An abort drops the step without running it, so the pick's
// selection stays. A step that leaves the operation in progress, such as a continue
// that stops on another conflict, keeps the step.
func (self *MergeAndRebaseHelper) settleOperationFinished(command string, result error) {
	ended := self.c.Git().Status.WorkingTreeState().None()
	self.c.OnUIThread(func() error {
		defer self.operationsInFlight.Add(-1)
		if command == REBASE_OPTION_ABORT {
			if ended {
				self.clearOperationFinished()
			}
			return nil
		}
		if result != nil || !ended {
			return nil
		}
		callback := self.onOperationFinished
		self.clearOperationFinished()
		if callback == nil {
			return nil
		}
		return callback()
	})
}

func (self *MergeAndRebaseHelper) clearOperationFinished() {
	self.onOperationFinished = nil
	self.onOperationFinishedAbandoned = nil
}
