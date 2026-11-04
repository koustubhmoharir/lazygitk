package controllers

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: c c continues the cherry-pick, rebase, merge or revert in progress from
// any pane. Upstream continues only from the merge/rebase options menu (g o).
// Unmerged paths are checked fresh from Git first, so a file whose markers are
// gone but which is not staged is refused rather than silently committed.
func forkContinueOperationBinding(c *ControllerCommon, opts types.KeybindingsOpts) *types.Binding {
	return &types.Binding{
		CommandID:         "operation.continue-in-progress",
		Verb:              "continue",
		Handler:           opts.Guards.NoPopupPanel(func() error { return forkContinueOperation(c) }),
		Description:       c.Tr.ContinueOperation,
		GetDisabledReason: func() *types.DisabledReason { return forkContinueOperationDisabledReason(c) },
	}
}

func forkContinueOperationDisabledReason(c *ControllerCommon) *types.DisabledReason {
	if c.Model().WorkingTreeStateAtLastCommitRefresh.None() {
		return &types.DisabledReason{Text: c.Tr.NotMergingOrRebasing, Inapplicable: true}
	}
	return nil
}

func forkContinueOperation(c *ControllerCommon) error {
	paths, err := c.Git().Status.UnmergedPaths()
	if err != nil {
		return err
	}
	if len(paths) > 0 {
		return fmt.Errorf(c.Tr.ContinueUnmergedFiles, strings.Join(paths, "\n- "))
	}
	return c.Helpers().MergeAndRebase.ContinueRebase()
}
