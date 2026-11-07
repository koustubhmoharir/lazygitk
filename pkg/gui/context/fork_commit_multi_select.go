package context

import (
	"fmt"
	"slices"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context/traits"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// Fork: multi-select mode in the Commits pane (item 24). Commits has two selection
// modes. In single-select mode, with no set and no run, the cursor commit is the
// selection, as upstream has it. In multi-select mode the selection is an explicit set
// of commits, kept by hash, plus the run in progress: a Shift run (non-sticky), or a
// sticky range from t h r. The cursor moves without changing the set.
//
// A run ends on any cursor operation that cancels a range, and then its commits join
// the set. ListCursor's own calls do not go through this view model, so the overrides
// below intercept each entry point that cancels a range, before the cursor moves.
// Space and Escape are in controllers/fork_commit_multi_select.go; the selection and
// cursor are drawn by syncMultiSelect below.

// commitMultiSelect is the explicit set of multi-select mode, and what the view last
// drew of the commit list.
type commitMultiSelect struct {
	// hashes holds the commits of the set, by hash. Rebase todo items never join it. A
	// hash stays until the user removes it or a refresh finds its object missing: a
	// filter or the commit limit hiding it does not remove it.
	hashes map[string]bool
	// notified holds the set's hashes that a refresh has already named as no longer
	// reached by any ref. A hash leaves it when it leaves the set, so a later rewrite
	// is named again. See NewUnreachableNotices.
	notified map[string]bool
	// drawn is the commit list the view last synchronised with, and the hashes of the
	// run's ends at that time. When a refresh has replaced the list since, the run's
	// indices may point at other commits, so the run that ends then is found again by
	// these hashes; see runCommitsAsDrawn.
	drawn       []*models.Commit
	drawnAnchor string
	drawnCursor string
}

// AreMultipleItemsSelected overrides the cursor's answer, which knows only about a run.
// It is true with an explicit set too, so that single-item checks (singleItemSelected,
// the discard and custom patch paths) refuse a multi-selection rather than act on the
// cursor commit. The set's commands are scoped in controllers/fork_commit_multi_select.go.
func (self *LocalCommitsViewModel) AreMultipleItemsSelected() bool {
	return self.HasCommitSet() || self.ListCursor.AreMultipleItemsSelected()
}

// MultiSelectActive reports whether the pane is in multi-select mode: a set is held, or
// a run is in progress.
func (self *LocalCommitsViewModel) MultiSelectActive() bool {
	return self.HasCommitSet() || self.IsSelectingRange()
}

// HasCommitSet reports whether the explicit set holds any commit. A run alone, as
// upstream's range selection, does not make a set.
func (self *LocalCommitsViewModel) HasCommitSet() bool {
	return len(self.multiSelect.hashes) > 0
}

// ClearMultiSelection empties the set and cancels the run in progress, without adding
// the run to the set. This is what Escape does.
func (self *LocalCommitsViewModel) ClearMultiSelection() {
	self.multiSelect.hashes = nil
	self.multiSelect.notified = nil
	self.ListCursor.CancelRangeSelect()
}

// SelectedCommitHashes returns the hashes that the selection covers, each once: the set,
// including the members that the loaded list does not show, and the run in progress. The
// order is not meaningful. In single-select mode it is empty; the cursor commit is the
// selection there, and GetSelectedItems gives it.
func (self *LocalCommitsViewModel) SelectedCommitHashes() []string {
	if !self.MultiSelectActive() {
		return nil
	}
	seen := make(map[string]bool, len(self.multiSelect.hashes))
	hashes := make([]string, 0, len(self.multiSelect.hashes))
	add := func(hash string) {
		if !seen[hash] {
			seen[hash] = true
			hashes = append(hashes, hash)
		}
	}
	for hash := range self.multiSelect.hashes {
		add(hash)
	}
	for _, commit := range self.runCommits() {
		add(commit.Hash())
	}
	return hashes
}

// CommitSelectionCounts returns how many commits the selection covers, and how many of
// them the loaded list does not show. Both are zero in single-select mode.
func (self *LocalCommitsViewModel) CommitSelectionCounts() (selected int, notShown int) {
	hashes := self.SelectedCommitHashes()
	return len(hashes), len(self.SelectedCommitHashesNotIn(self.GetCommits()))
}

// SelectedCommitHashesNotIn returns the set's hashes that commits does not show, in
// sorted order. A rebase todo item in the list does not count as showing its commit.
func (self *LocalCommitsViewModel) SelectedCommitHashesNotIn(commits []*models.Commit) []string {
	present := make(map[string]bool, len(commits))
	for _, commit := range realCommits(commits) {
		present[commit.Hash()] = true
	}
	absent := []string{}
	for hash := range self.multiSelect.hashes {
		if !present[hash] {
			absent = append(absent, hash)
		}
	}
	slices.Sort(absent)
	return absent
}

// RemoveFromCommitSet removes hashes from the set, as a refresh does for commits whose
// objects no longer exist. Nothing else about the selection changes.
func (self *LocalCommitsViewModel) RemoveFromCommitSet(hashes []string) {
	for _, hash := range hashes {
		self.removeFromSet(hash)
	}
}

// NewUnreachableNotices returns the set's hashes among hashes that have not been named
// yet, and records them as named. Hashes that are not in the set are ignored.
func (self *LocalCommitsViewModel) NewUnreachableNotices(hashes []string) []string {
	var fresh []string
	for _, hash := range hashes {
		if !self.multiSelect.hashes[hash] || self.multiSelect.notified[hash] {
			continue
		}
		if self.multiSelect.notified == nil {
			self.multiSelect.notified = map[string]bool{}
		}
		self.multiSelect.notified[hash] = true
		fresh = append(fresh, hash)
	}
	return fresh
}

// removeFromSet removes a hash from the set and forgets that it was named.
func (self *LocalCommitsViewModel) removeFromSet(hash string) {
	delete(self.multiSelect.hashes, hash)
	delete(self.multiSelect.notified, hash)
}

// SetSelectionAfterRewrite moves the cursor to idx for an operation that rewrites the
// commits of the run in progress: a drop, fixup or squash starting on the run. The run
// ends without joining the set, since the operation takes its commits. A plain move
// would add them to the set, and they would then be unreachable once rewritten.
func (self *LocalCommitsViewModel) SetSelectionAfterRewrite(idx int) {
	self.ListCursor.CancelRangeSelect()
	self.ListCursor.SetSelection(idx)
}

// ForgetRewrittenCommits removes commits from the set before lazygitk rewrites them by
// moving them. A drag ends its run when the cursor moves during it, which adds the run to
// the set; the rewrite would then leave those hashes unreachable and a notice would name
// them. The moved commits are selected again as the run, so they leave the set.
func (self *LocalCommitsViewModel) ForgetRewrittenCommits(commits []*models.Commit) {
	for _, commit := range realCommits(commits) {
		self.removeFromSet(commit.Hash())
	}
}

// ToggleCommitSelection is Space in Commits. With a run in progress it toggles the
// whole run and ends the run; otherwise it toggles the cursor commit in the set.
func (self *LocalCommitsViewModel) ToggleCommitSelection() {
	if self.IsSelectingRange() {
		self.toggleCommits(self.runCommits())
		self.ListCursor.CancelRangeSelect()
		return
	}
	if commit := self.GetSelected(); commit != nil {
		self.toggleCommits([]*models.Commit{commit})
	}
}

// toggleCommits removes the commits from the set when all of them are in it already,
// and adds them otherwise.
func (self *LocalCommitsViewModel) toggleCommits(commits []*models.Commit) {
	commits = realCommits(commits)
	if len(commits) == 0 {
		return
	}
	allInSet := lo.EveryBy(commits, func(commit *models.Commit) bool {
		return self.multiSelect.hashes[commit.Hash()]
	})
	if allInSet {
		for _, commit := range commits {
			self.removeFromSet(commit.Hash())
		}
		return
	}
	self.addToSet(commits)
}

func (self *LocalCommitsViewModel) addToSet(commits []*models.Commit) {
	for _, commit := range realCommits(commits) {
		if self.multiSelect.hashes == nil {
			self.multiSelect.hashes = map[string]bool{}
		}
		self.multiSelect.hashes[commit.Hash()] = true
	}
}

// endRun ends the run in progress, if there is one, and its commits join the set.
func (self *LocalCommitsViewModel) endRun() {
	if !self.IsSelectingRange() {
		return
	}
	self.addToSet(self.runCommitsAsDrawn())
	self.ListCursor.CancelRangeSelect()
}

// runCommits returns the commits of the run in progress, in list order, by the current
// indices of the run. It returns nil when no run is in progress.
func (self *LocalCommitsViewModel) runCommits() []*models.Commit {
	if !self.IsSelectingRange() {
		return nil
	}
	commits := self.GetCommits()
	start, end := self.GetSelectionRange()
	if start < 0 || start >= len(commits) {
		return nil
	}
	end = min(end, len(commits)-1)
	return commits[start : end+1]
}

// runCommitsAsDrawn is runCommits for a run that ends: when the list was replaced after
// the run was last drawn, the run is the one that was drawn, found again by its ends'
// hashes, since a refresh may have moved the indices. Commits that are gone are left out.
func (self *LocalCommitsViewModel) runCommitsAsDrawn() []*models.Commit {
	commits := self.GetCommits()
	if self.multiSelect.drawn == nil || sameCommitList(commits, self.multiSelect.drawn) {
		return self.runCommits()
	}
	anchor := indexOfCommitHash(commits, self.multiSelect.drawnAnchor)
	cursor := indexOfCommitHash(commits, self.multiSelect.drawnCursor)
	if anchor < 0 || cursor < 0 {
		return nil
	}
	start, end := utils.SortRange(anchor, cursor)
	return commits[start : end+1]
}

// recordDrawn remembers the list and the run's ends as the view now draws them.
func (self *LocalCommitsViewModel) recordDrawn(commits []*models.Commit) {
	cursor := self.GetSelectedLineIdx()
	anchor := cursor
	if idx, inRun := self.GetRangeStartIdx(); inRun {
		anchor = idx
	}
	self.multiSelect.drawn = commits
	self.multiSelect.drawnAnchor = commitHashAt(commits, anchor)
	self.multiSelect.drawnCursor = commitHashAt(commits, cursor)
}

// GetSelectedItems returns the loaded commits that the selection covers. In multi-select
// mode that is the set and the run, in list order, and the indices of the first and the
// last of them. Members the loaded list does not show are left out; SelectedCommitHashes
// has the whole selection. In single-select mode it is upstream's selection.
func (self *LocalCommitsViewModel) GetSelectedItems() ([]*models.Commit, int, int) {
	if !self.MultiSelectActive() {
		return self.ListViewModel.GetSelectedItems()
	}
	inRun := map[*models.Commit]bool{}
	for _, commit := range self.runCommits() {
		inRun[commit] = true
	}
	var selected []*models.Commit
	first, last := -1, -1
	for i, commit := range self.GetCommits() {
		if inRun[commit] || (!commit.IsTODO() && self.multiSelect.hashes[commit.Hash()]) {
			if first < 0 {
				first = i
			}
			last = i
			selected = append(selected, commit)
		}
	}
	if len(selected) == 0 {
		return nil, -1, -1
	}
	return selected, first, last
}

// GetSelectedItemIds is GetSelectedItems by ID.
func (self *LocalCommitsViewModel) GetSelectedItemIds() ([]string, int, int) {
	items, first, last := self.GetSelectedItems()
	ids := lo.Map(items, func(commit *models.Commit, _ int) string { return commit.ID() })
	return ids, first, last
}

// The overrides below are the entry points that end a range. Moves within a sticky
// range keep it, as upstream does, so only a non-sticky run ends on a plain move.

func (self *LocalCommitsViewModel) MoveSelectedLine(change int) {
	if _, _, mode := self.GetSelectionRangeAndMode(); mode == traits.RangeSelectModeNonSticky {
		self.endRun()
	}
	self.ListCursor.MoveSelectedLine(change)
}

func (self *LocalCommitsViewModel) SetSelection(value int) {
	self.endRun()
	self.ListCursor.SetSelection(value)
}

func (self *LocalCommitsViewModel) CancelRangeSelect() {
	self.endRun()
}

func (self *LocalCommitsViewModel) CollapseRangeSelectionToTop() {
	start, _ := self.GetSelectionRange()
	self.SetSelection(start)
}

// ToggleStickyRange ends a range when one is in progress, as upstream does, and starts a
// sticky range otherwise. Turning it off adds the range to the set.
func (self *LocalCommitsViewModel) ToggleStickyRange() {
	if self.IsSelectingRange() {
		self.endRun()
		return
	}
	self.ListCursor.ToggleStickyRange()
}

// commitSetSubtitle is the Commits subtitle while a set is held: how many commits it
// holds, and how many of them the loaded list does not show. It is empty otherwise.
func (self *LocalCommitsContext) commitSetSubtitle() string {
	if !self.HasCommitSet() {
		return ""
	}
	tr := self.ListContextTrait.c.Tr
	selected, notShown := self.CommitSelectionCounts()
	if notShown == 0 {
		return fmt.Sprintf(tr.CommitSetCount, selected)
	}
	return fmt.Sprintf(tr.CommitSetCountNotShown, selected, notShown)
}

// syncMultiSelect brings the view in line with the multi-select state: the highlighted
// lines of the set and the run, the cursor marker and the subtitle. It runs whenever the
// list is focused, moved or rendered, so it costs time in the size of the loaded list.
func (self *LocalCommitsContext) syncMultiSelect() {
	commits := self.GetCommits()

	view := self.GetView()
	view.Subtitle = self.commitSetSubtitle()
	if !self.MultiSelectActive() || len(commits) == 0 {
		view.SetSelectedLines(nil)
		view.SetInclusionGutter(false, nil)
		self.recordDrawn(commits)
		return
	}

	convert := self.modelToViewIndexConverter()
	lines := make([]bool, self.TotalContentHeight())
	inRun := map[*models.Commit]bool{}
	for _, commit := range self.runCommits() {
		inRun[commit] = true
	}
	for i, commit := range commits {
		if inRun[commit] || (!commit.IsTODO() && self.multiSelect.hashes[commit.Hash()]) {
			lines[convert(i)] = true
		}
	}
	marks := make([]bool, len(lines))
	marks[convert(self.GetSelectedLineIdx())] = true

	view.SetSelectedLines(lines)
	view.SetInclusionGutter(true, marks)
	self.recordDrawn(commits)
}

// HandleFocus, FocusLine and HandleRender sync the view after the list's own work. The
// list trait calls its own FocusLine from HandleFocus, so FocusLine is overridden too.

func (self *LocalCommitsContext) HandleFocus(opts types.OnFocusOpts) {
	self.ListContextTrait.HandleFocus(opts)
	self.syncMultiSelect()
}

func (self *LocalCommitsContext) FocusLine(scrollIntoView bool) {
	self.ListContextTrait.FocusLine(scrollIntoView)
	self.syncMultiSelect()
}

func (self *LocalCommitsContext) HandleRender() {
	self.ListContextTrait.HandleRender()
	self.syncMultiSelect()
}

func indexOfCommitHash(commits []*models.Commit, hash string) int {
	if hash == "" {
		return -1
	}
	_, idx, found := lo.FindIndexOf(commits, func(commit *models.Commit) bool { return commit.Hash() == hash })
	if !found {
		return -1
	}
	return idx
}

func commitHashAt(commits []*models.Commit, idx int) string {
	if idx < 0 || idx >= len(commits) {
		return ""
	}
	return commits[idx].Hash()
}

// sameCommitList reports whether a and b are the same list, not merely equal lists: a
// refresh replaces the list with a new one.
func sameCommitList(a, b []*models.Commit) bool {
	if len(a) != len(b) {
		return false
	}
	return len(a) == 0 || &a[0] == &b[0]
}

// realCommits returns the commits that are not rebase todo items. Their hashes may not
// name a commit of the list, so they never join the set.
func realCommits(commits []*models.Commit) []*models.Commit {
	return lo.Filter(commits, func(commit *models.Commit, _ int) bool { return !commit.IsTODO() })
}
