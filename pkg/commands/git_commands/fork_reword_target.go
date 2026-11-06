package git_commands

// CurrentHead returns the full ref of the checked-out branch (empty when HEAD is
// detached) and HEAD's full object ID. Fork: a reword captures these when it starts
// and verifies them before it writes; see controllers/fork_reword_target.go.
func (self *BranchCommands) CurrentHead() (ref, oid string, err error) {
	return self.namedCheckoutHead()
}
