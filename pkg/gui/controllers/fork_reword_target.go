package controllers

import (
	"errors"
	"slices"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: a reword captures its target when it starts: the repository, the commit's
// ID, a copy of the commits and the selected index, whether the commit was HEAD, and
// HEAD's ref and ID. Both submit paths verify that capture before any write, and then
// act only on the captured values. Upstream's handleReword and
// switchFromCommitMessagePanelToEditor read the live commits and selection when the
// message was submitted, so a commit that moved while the box was open was rewritten
// instead. The integration point is LocalCommitsController.reword.
type rewordTarget struct {
	lc         *LocalCommitsController
	git        *commands.GitCommand
	generation int
	commits    []*models.Commit
	index      int
	commitOID  string
	wasHead    bool
	headRef    string
	headOID    string
}

func (self *LocalCommitsController) captureRewordTarget(commit *models.Commit, index int) (*rewordTarget, error) {
	headRef, headOID, err := self.c.Git().Branch.CurrentHead()
	if err != nil {
		return nil, err
	}
	commits := self.c.Model().Commits
	// The captured commits drive the rebase, so they must describe the HEAD
	// captured above. A model that is already stale is refused before the box opens.
	if !self.rewordListMatchesHead(commit.Hash(), commits, headOID) {
		self.c.Refresh(types.RefreshOptions{})
		return nil, errors.New(self.c.Tr.RewordListStale)
	}
	rebaseCommits, rebaseIndex, err := self.rewordRebaseCommits(commit, commits, index)
	if err != nil {
		return nil, err
	}
	return &rewordTarget{
		lc:         self,
		git:        self.c.Git(),
		generation: self.c.State().GetRepoGeneration(),
		commits:    rebaseCommits,
		index:      rebaseIndex,
		commitOID:  commit.Hash(),
		// The hash, not the position: a filtered list may start with a commit that is not HEAD.
		wasHead: commit.Hash() == headOID,
		headRef: headRef,
		headOID: headOID,
	}, nil
}

// rewordListMatchesHead reports whether the captured commit list describes headOID. An
// unfiltered list starts with HEAD. A filtered list (a path, author or commit scope
// filter) holds only the matching commits, so HEAD may not be in it. Then the check is
// that the target is still reached from HEAD, which a rewrite outside Git's view would
// break. Reword is allowed in filter mode, as upstream decided in 52be6964e.
func (self *LocalCommitsController) rewordListMatchesHead(targetOID string, commits []*models.Commit, headOID string) bool {
	if !self.c.Modes().Filtering.Active() {
		headIdx := models.HeadCommitIdx(commits)
		return headIdx >= 0 && commits[headIdx].Hash() == headOID
	}
	return self.c.Git().Branch.IsAncestor(targetOID, headOID)
}

// rewordRebaseCommits returns the list and index that the rebase is started from. The
// rebase base is the commit after the target in the list (see getBaseHashOrRoot). An
// unfiltered list has the target's parent there, so it is used as it is. A filtered list
// holds only the matching commits, so the next row may be an older match, a commit on
// the other side of a merge, or nothing (--root). Starting from it widens the rebase
// and its ignored-file review to commits that have nothing to do with the target.
// Then the list is [target, first parent], with the parent as the base, or [target] for
// a root commit, which gives --root. Git resolves the parent now, because the model's
// parents can be rewritten in a filtered list. A merge target still uses itself as the
// base, as it does unfiltered, so the list does not change its behaviour.
func (self *LocalCommitsController) rewordRebaseCommits(commit *models.Commit, commits []*models.Commit, index int) ([]*models.Commit, int, error) {
	if !self.c.Modes().Filtering.Active() {
		return slices.Clone(commits), index, nil
	}
	// A shallow boundary commit has no first parent in Git's view, so it is treated as a
	// root, as the unfiltered list treats the last row of a shallow history.
	resolved, err := self.c.Git().Diff.ResolveHistoricalDiffCommit(commit.Hash())
	if err != nil {
		return nil, 0, err
	}
	rebaseCommits := []*models.Commit{commit}
	if resolved.FirstParent != "" {
		pool := &utils.StringPool{}
		rebaseCommits = append(rebaseCommits, models.NewCommit(pool, models.NewCommitOpts{Hash: resolved.FirstParent}))
	}
	return rebaseCommits, 0, nil
}

// verify reports an error unless the repository, HEAD's ref and ID, and (for a HEAD
// target) HEAD's ID are still the captured ones. It writes nothing.
func (target *rewordTarget) verify() error {
	tr := target.lc.c.Tr
	if target.lc.c.Git() != target.git || target.lc.c.State().GetRepoGeneration() != target.generation {
		return errors.New(tr.RewordRepositoryChanged)
	}
	ref, oid, err := target.git.Branch.CurrentHead()
	if err != nil {
		return err
	}
	if ref != target.headRef || oid != target.headOID {
		return errors.New(tr.RewordTargetChanged)
	}
	if target.wasHead && oid != target.commitOID {
		return errors.New(tr.RewordTargetChanged)
	}
	return nil
}

// refuse returns a failed verification. Nothing was written, so the submit paths have
// not reported success and the text stays a draft. The panes refresh to the actual state.
func (target *rewordTarget) refuse(err error) error {
	target.lc.c.Refresh(types.RefreshOptions{})
	return err
}

// submit is the panel's OnConfirm.
func (target *rewordTarget) submit(summary string, description string) error {
	// Fork: see TakeCommitMessageSubmitReporter. It is called only after a success.
	reportSuccess := target.lc.c.Helpers().Commits.TakeCommitMessageSubmitReporter()
	if err := target.verify(); err != nil {
		return target.refuse(err)
	}
	if target.wasHead {
		return target.lc.c.Helpers().GPG.WithGpgHandling(target.git.Commit.RewordLastCommit(summary, description),
			git_commands.CommitGpgSign,
			target.lc.c.Tr.RewordingStatus,
			// Fork: the reword has succeeded.
			func() error {
				reportSuccess()
				return nil
			}, nil)
	}

	return target.lc.c.WithWaitingStatusBlockingInput(types.WaitingStatusOpts{
		Message:              target.lc.c.Tr.RewordingStatus,
		HideWorkingTreeState: true,
	}, func(gocui.Task) error {
		// Fork: the rebase start is reviewed for ignored files; see fork_rebase_ignored_review.go.
		// The target is verified again inside each attempt, so an approval given later never
		// rewrites a commit that has moved since.
		rebaseStarted := false
		return runWithIgnoredRebaseReview(target.lc.c, func(rebase *git_commands.RebaseCommands) error {
			// The rebase is the write, so verify again on the worker just before it.
			if err := target.verify(); err != nil {
				return err
			}
			rebaseStarted = true
			return rebase.RewordCommit(target.commits, target.index, summary, description)
		}, func(err error) error {
			if err != nil {
				// Fork: the rebase may be left stopped, and the text is kept as a draft. Refresh
				// first, as on success, so that the model shows the stopped rebase before input
				// is released; see fork_reword_failure.go. A verification refusal starts nothing.
				target.lc.c.RefreshFromWorker(types.RefreshOptions{})
				if rebaseStarted {
					return target.lc.rewordFailure(err)
				}
				return err
			}
			// Fork: the reword has succeeded.
			reportSuccess()
			target.lc.c.RefreshFromWorker(types.RefreshOptions{})
			return nil
		})
	})
}

// switchToEditor is the panel's OnSwitchToEditor. The text it is given is the file the
// editor opens for the message.
func (target *rewordTarget) switchToEditor(filepath string) error {
	// Fork: see TakeCommitMessageSubmitReporter. Called only once the editor is about
	// to get the text, so a refusal or a failed rebase start keeps the draft.
	reportSuccess := target.lc.c.Helpers().Commits.TakeCommitMessageSubmitReporter()
	if err := target.verify(); err != nil {
		return target.refuse(err)
	}
	if target.wasHead {
		reportSuccess()
		return target.lc.c.RunSubprocessAndRefresh(
			target.git.Commit.RewordLastCommitInEditorWithMessageFileCmdObj(filepath))
	}

	// Fork: the rebase start is reviewed for ignored files, and the editor and the continue
	// run after the review on the UI thread, which the subprocess needs.
	return runWithIgnoredRebaseReview(target.lc.c, func(rebase *git_commands.RebaseCommands) error {
		if err := target.verify(); err != nil {
			return err
		}
		return rebase.BeginInteractiveRebaseForCommit(target.commits, target.index, false)
	}, func(err error) error {
		if err != nil {
			return err
		}
		reportSuccess()
		target.lc.c.OnUIThread(func() error {
			// now the selected commit should be our head so we'll amend it with the new message
			err := target.lc.c.RunSubprocessAndRefresh(
				target.git.Commit.RewordLastCommitInEditorWithMessageFileCmdObj(filepath))
			if err != nil {
				return err
			}

			err = target.git.Rebase.ContinueRebase()
			if err != nil {
				return err
			}

			target.lc.c.Refresh(types.RefreshOptions{})
			return nil
		})
		return nil
	})
}
