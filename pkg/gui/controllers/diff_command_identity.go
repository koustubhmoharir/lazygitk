package controllers

type diffCommandSource uint8

const (
	diffCommandSourceNone diffCommandSource = iota
	diffCommandSourceWorkingTree
	diffCommandSourceCommit
)

// diffSelectionIdentity describes the existing mode-dependent operations, not
// the selected paths, refs or line contents. It performs no Git calls.
func diffSelectionIdentity(source diffCommandSource, secondary, discard bool) (string, string) {
	switch source {
	case diffCommandSourceWorkingTree:
		if secondary {
			return "diff.unstage-selection", "unstage"
		}
		if discard {
			return "diff.discard-worktree-selection", "discard"
		}
		return "diff.stage-selection", "stage"
	case diffCommandSourceCommit:
		if discard {
			// Disabled on the custom-patch side by the existing handler guard.
			return "commit.discard-selection", "discard-from-commit"
		}
		if secondary {
			return "patch.remove-selection", "remove-from-patch"
		}
		return "patch.toggle-selection", "toggle-patch-selection"
	case diffCommandSourceNone:
		// Off-stack panes still contribute generic metadata to static tooling.
		// These operations are unavailable there and must not count as usage.
		if discard {
			return "diff.discard-selection", "discard"
		}
		return "diff.primary-selection", "primary-action"
	}
	panic("unknown diff command source")
}

func (self *MainViewController) diffSelectionCommandIdentity(discard bool) (string, string) {
	source := diffCommandSourceNone
	switch self.diffSource().(type) {
	case *WorkingTreeDiffActions:
		source = diffCommandSourceWorkingTree
	case *CommitDiffActions:
		source = diffCommandSourceCommit
	}
	secondary := self.context.GetKey() == self.c.Contexts().NormalSecondary.GetKey()
	return diffSelectionIdentity(source, secondary, discard)
}

func (self *MainViewController) primaryActionCommandID() string {
	id, _ := self.diffSelectionCommandIdentity(false)
	return id
}

func (self *MainViewController) primaryActionVerb() string {
	_, verb := self.diffSelectionCommandIdentity(false)
	return verb
}

func (self *MainViewController) discardSelectionCommandID() string {
	id, _ := self.diffSelectionCommandIdentity(true)
	return id
}

func (self *MainViewController) discardSelectionVerb() string {
	_, verb := self.diffSelectionCommandIdentity(true)
	return verb
}
