package controllers

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// fastForwardRun is one `g f` invocation (roadmap item 16). Each captured
// branch moves to its configured upstream exactly as it is locally: nothing is
// fetched and no fetch review is shown. The only prompt is the existing
// ignored-file approval for the checked-out branch. Every other branch that
// cannot move is recorded and listed in one alert at the end.
type fastForwardRun struct {
	target  *branchSyncTarget
	skipped []fastForwardSkip
}

type fastForwardSkip struct{ name, reason string }

// fastForwardCapturedBranches copies names at entry so a later model
// replacement or range-selection change cannot redirect the update.
func (self *BranchesController) fastForwardCapturedBranches(names []string) error {
	if len(names) == 0 {
		return errors.New(self.c.Tr.BranchSyncSelectOneLocalBranch)
	}
	names = slices.Clone(names)
	target := newBranchSyncTarget(self.c, branchSyncSource{name: names[0], selected: true})
	if err := target.checkRepository(); err != nil {
		return err
	}
	if hasDuplicateName(names) {
		return git_commands.ErrBranchSyncChanged
	}
	// Mark synchronously on the UI thread, before the worker is dispatched.
	release, err := target.markBranchesBusy(names)
	if err != nil {
		return err
	}
	run := &fastForwardRun{target: target}
	err = self.c.WithWaitingStatus(self.c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		return run.execute(names, release)
	})
	if err != nil {
		release()
	}
	return err
}

func hasDuplicateName(names []string) bool {
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if seen[name] {
			return true
		}
		seen[name] = true
	}
	return false
}

// markBranchesBusy reserves every name against concurrent branch operations.
// The returned release is safe to call more than once.
func (target *branchSyncTarget) markBranchesBusy(names []string) (func(), error) {
	if err := target.checkRepository(); err != nil {
		return nil, err
	}
	state := target.c.State()
	for _, name := range names {
		item := syncOperationItem(target.git, name)
		ordinary := &models.Branch{Name: name}
		if state.GetItemOperation(item) != types.ItemOperationNone || state.GetItemOperation(ordinary) != types.ItemOperationNone {
			return nil, errors.New(target.c.Tr.CantPullOrPushSameBranchTwice)
		}
	}
	for _, name := range names {
		state.SetItemOperation(syncOperationItem(target.git, name), types.ItemOperationPulling)
	}
	return func() {
		for _, name := range names {
			state.ClearItemOperation(syncOperationItem(target.git, name))
		}
	}, nil
}

// execute runs on the worker. It owns release until it hands off to the UI
// thread, where the ignored-file approval may reserve the checked-out branch
// again.
func (run *fastForwardRun) execute(names []string, release func()) error {
	target := run.target
	if err := target.checkRepository(); err != nil {
		release()
		return err
	}
	var behind []git_commands.UpstreamFastForward
	var current *git_commands.UpstreamFastForward
	for _, name := range names {
		if err := target.checkRepository(); err != nil {
			release()
			return err
		}
		state, err := target.git.Sync.InspectUpstreamFastForward(name)
		if err != nil {
			run.skip(name, err.Error())
			continue
		}
		switch state.State {
		case git_commands.UpstreamFastForwardUpToDate:
		case git_commands.UpstreamFastForwardNoUpstream:
			run.skip(name, target.c.Tr.BranchSyncFastForwardNoUpstream)
		case git_commands.UpstreamFastForwardMissingTrackingRef:
			run.skip(name, fmt.Sprintf(target.c.Tr.BranchSyncFastForwardMissingRef, state.Info.UpstreamTrackingRef))
		case git_commands.UpstreamFastForwardDiverged:
			run.skip(name, fmt.Sprintf(target.c.Tr.BranchSyncFastForwardDiverged, state.Info.UpstreamTrackingRef))
		case git_commands.UpstreamFastForwardBehind:
			if state.Current() {
				current = &state
			} else {
				behind = append(behind, state)
			}
		}
	}
	// Off-branch refs move first so the checked-out branch is updated last.
	for _, state := range behind {
		if err := run.moveOffBranch(state); err != nil {
			release()
			return err
		}
	}
	if current == nil {
		return run.finishWorker(release)
	}
	return run.moveCurrent(*current, release)
}

// moveOffBranch uses the guarded ref update. The expected old OID is the
// inspected tip, and the target is the upstream OID captured at the same time.
func (run *fastForwardRun) moveOffBranch(state git_commands.UpstreamFastForward) error {
	target := run.target
	name := state.Info.Name
	if err := target.checkRepository(); err != nil {
		return err
	}
	plan, err := target.git.Sync.PrepareBranchFastForward(name, state.Info.OID, state.UpstreamOID)
	if err != nil {
		run.skip(name, err.Error())
		return nil
	}
	if err := target.checkRepository(); err != nil {
		return err
	}
	if err := target.git.Sync.ApplyBranchFastForward(plan); err != nil {
		run.skip(name, err.Error())
	}
	return nil
}

// moveCurrent integrates the upstream fast-forward-only, in process. The
// ignored-file overwrite risk is the only case that needs a decision, so it is
// handed to the UI thread for approval before anything moves.
func (run *fastForwardRun) moveCurrent(state git_commands.UpstreamFastForward, release func()) error {
	target := run.target
	name := state.Info.Name
	if err := target.checkRepository(); err != nil {
		release()
		return err
	}
	review, err := target.git.Sync.PrepareCurrentPullReview(name, state.Info.OID, "")
	if err != nil {
		run.skip(name, err.Error())
		return run.finishWorker(release)
	}
	plan, err := target.git.Sync.PrepareCurrentPullIntegration(review, state.UpstreamOID, git_commands.PullIntegrationFastForwardOnly)
	if err != nil {
		run.skip(name, err.Error())
		return run.finishWorker(release)
	}
	if len(plan.IgnoredFiles()) > 0 {
		release()
		// Show the off-branch moves already made while the approval is pending,
		// and keep them visible if the user cancels.
		target.c.RefreshFromWorker(types.RefreshOptions{})
		target.c.OnUIThread(func() error { return run.reviewIgnored(plan) })
		return nil
	}
	if err := run.executeCurrent(plan); err != nil {
		release()
		return err
	}
	return run.finishWorker(release)
}

// executeCurrent runs the reviewed ff-only merge in process. Git refusing the
// merge (for example, untracked files in the way) is a skipped entry carrying
// Git's own error; it is not a terminal prompt.
func (run *fastForwardRun) executeCurrent(plan git_commands.CurrentPullIntegrationPlan) error {
	target := run.target
	if err := target.checkRepository(); err != nil {
		return err
	}
	cmd, err := target.git.Sync.CurrentPullIntegrationCommand(plan)
	if err != nil {
		run.skip(plan.Name(), err.Error())
		return nil
	}
	if err := target.checkRepository(); err != nil {
		return err
	}
	if err := cmd.Run(); err != nil {
		run.skip(plan.Name(), err.Error())
	}
	return nil
}

func (run *fastForwardRun) reviewIgnored(plan git_commands.CurrentPullIntegrationPlan) error {
	target := run.target
	if err := target.checkRepository(); err != nil {
		return err
	}
	prompt := fmt.Sprintf(target.c.Tr.BranchSyncFastForwardIgnoredPrompt, plan.Name(), plan.SourceOID(), plan.TargetOID())
	return target.reviewIgnoredApproval(plan, prompt, run.approveCurrent, run.finishUI)
}

// approveCurrent runs on the UI thread after approval. It reserves the branch
// again, because the reservation was released when the review was handed off.
func (run *fastForwardRun) approveCurrent(plan git_commands.CurrentPullIntegrationPlan) error {
	target := run.target
	if err := target.checkRepository(); err != nil {
		return err
	}
	name := plan.Name()
	release, err := target.markBranchesBusy([]string{name})
	if err != nil {
		run.skip(name, err.Error())
		return run.finishUI()
	}
	err = target.c.WithWaitingStatus(target.c.Tr.BranchSyncIntegrationStatus, func(gocui.Task) error {
		if err := run.executeCurrent(plan.WithIgnoredFilesApproved()); err != nil {
			release()
			return err
		}
		return run.finishWorker(release)
	})
	if err != nil {
		release()
	}
	return err
}

// finishWorker clears the reservations, refreshes from the worker, and queues
// the summary on the UI thread so it is the last thing the user sees.
func (run *fastForwardRun) finishWorker(release func()) error {
	release()
	if run.target.checkRepository() == nil {
		run.target.c.RefreshFromWorker(types.RefreshOptions{})
	}
	run.target.c.OnUIThread(run.finishUI)
	return nil
}

// finishUI shows the single alert for skipped branches. Success is silent.
func (run *fastForwardRun) finishUI() error {
	if err := run.target.checkRepository(); err != nil {
		return err
	}
	if len(run.skipped) == 0 {
		return nil
	}
	lines := make([]string, 0, len(run.skipped))
	for _, skip := range run.skipped {
		lines = append(lines, skip.name+": "+skip.reason)
	}
	run.target.c.Alert(run.target.c.Tr.BranchSyncFastForwardSkippedTitle, strings.Join(lines, "\n"))
	return nil
}

// skip records why a branch did not move. Git's error text can span several
// lines (such as the paths of untracked files in the way); its layout is kept,
// indented under the branch.
func (run *fastForwardRun) skip(name, reason string) {
	reason = strings.ReplaceAll(strings.TrimSpace(reason), "\n", "\n  ")
	run.skipped = append(run.skipped, fastForwardSkip{name: name, reason: reason})
}
