package controllers

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: c in Files commits or continues. Upstream commits there (commit.create,
// which keeps its place in the palette); the fork moves c to this binding, so
// the operation in progress is continued from the pane where the user works
// (Files) without a second key. Continuing reuses operation.continue-in-progress'
// handler and disabled reasons (fork_continue_operation.go).

// fileCommitOrContinue is the action that c in Files performs.
type fileCommitOrContinue int

const (
	fileActionCommit fileCommitOrContinue = iota
	fileActionContinue
)

// fileCommitOrContinueAction decides what c in Files does. With no operation in
// progress it commits. In a rebase stopped at a non-conflict stop (an edit stop,
// which Git marks with rebase-merge/amend) with staged changes, it commits too:
// the staged changes are the user's new commit, and Git's rebase --continue would
// refuse after a reset or amend them silently, whereas a commit keeps them as a new
// commit. Every other case continues: a merge, cherry-pick, revert, or a rebase
// stopped at a conflict or with nothing staged. a amends explicitly.
//
// The Effective state is used, so a cherry-pick that stopped inside a rebase
// continues the cherry-pick rather than committing into the rebase.
func fileCommitOrContinueAction(state models.WorkingTreeState, stoppedForEdit, anyStaged bool) fileCommitOrContinue {
	if state.None() {
		return fileActionCommit
	}
	if state.Effective() == models.WORKING_TREE_STATE_REBASING && stoppedForEdit && anyStaged {
		return fileActionCommit
	}
	return fileActionContinue
}

// forkFileCommitOrContinueBinding is Files' c. commit is the upstream commit
// handler, run unchanged when the action is to commit.
func forkFileCommitOrContinueBinding(c *ControllerCommon, opts types.KeybindingsOpts, commit func() error) *types.Binding {
	continueOperation := opts.Guards.NoPopupPanel(func() error { return forkContinueOperation(c) })
	action := func() fileCommitOrContinue {
		return fileCommitOrContinueAction(
			c.Model().WorkingTreeStateAtLastCommitRefresh,
			rebaseStoppedForEdit(c),
			c.Helpers().WorkingTree.AnyStagedFiles(),
		)
	}
	return &types.Binding{
		CommandID: "file.commit-or-continue",
		Verb:      "commit",
		VerbFunc: func() string {
			if action() == fileActionContinue {
				return "continue"
			}
			return "commit"
		},
		Handler: func() error {
			if action() == fileActionContinue {
				return continueOperation()
			}
			return commit()
		},
		Description: c.Tr.Commit,
		DescriptionFunc: func() string {
			if action() == fileActionContinue {
				// Named like the global hint in the options bar, e.g. "Continue merge".
				return fmt.Sprintf(c.Tr.ContinueOperationHint, c.Model().WorkingTreeStateAtLastCommitRefresh.CommandName())
			}
			return c.Tr.Commit
		},
		Tooltip: c.Tr.CommitTooltip,
		TooltipFunc: func() string {
			if action() == fileActionContinue {
				return ""
			}
			return c.Tr.CommitTooltip
		},
		GetDisabledReason: func() *types.DisabledReason {
			if action() == fileActionContinue {
				return forkContinueOperationDisabledReason(c)
			}
			return nil
		},
		DisplayOnScreen: true,
	}
}

// rebaseStoppedForEdit reports whether the worktree's rebase is stopped at a
// non-conflict stop, which Git records with rebase-merge/amend. A linked worktree
// keeps its rebase state in its own git dir.
func rebaseStoppedForEdit(c *ControllerCommon) bool {
	_, err := os.Stat(filepath.Join(c.Git().RepoPaths.WorktreeGitDirPath(), "rebase-merge", "amend"))
	return err == nil
}
