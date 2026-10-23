package git_commands

import "fmt"

// PrepareBranchActionHeadTarget captures current HEAD for actions whose source
// defaults to HEAD. It never leaves a live HEAD expression in a queued command.
func (self *BranchCommands) PrepareBranchActionHeadTarget() (BranchActionTarget, error) {
	ref, oid, err := self.namedCheckoutHead()
	if err != nil {
		return BranchActionTarget{}, err
	}
	if oid == "" {
		return BranchActionTarget{}, fmt.Errorf("current HEAD has no commit to use as a source")
	}
	source := ref
	if source == "" {
		source = oid
	}
	target, err := self.PrepareBranchActionTarget(source, oid)
	if err != nil {
		return BranchActionTarget{}, err
	}
	if target.HeadRef() != ref || target.HeadOID() != oid {
		return BranchActionTarget{}, ErrBranchActionChanged
	}
	return target, nil
}
