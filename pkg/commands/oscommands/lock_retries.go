package oscommands

// DisableLockRetries opts this command out of the Git runner's automatic
// retries after lock errors. This is useful for operations that may have
// partially changed repository state before reporting a lock error.
func (self *CmdObj) DisableLockRetries() *CmdObj {
	self.disableLockRetries = true
	return self
}

// LockRetriesDisabled reports whether automatic Git lock retries are disabled
// for this command.
func (self *CmdObj) LockRetriesDisabled() bool {
	return self.disableLockRetries
}
