package context

import (
	"fmt"
	"slices"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context/traits"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
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
// cursor are drawn by syncCommitSet below.
//
// Commits, Sub-commits and Reflog share this code. Each holds a forkCommitSet next to its
// list view model, and each has the exported wrappers below, which pass the view model's
// own list to the set's methods.

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

// forkCommitSet is the multi-select state and behaviour of a commit list: the explicit set,
// the run in progress and the view's drawn state. It holds no reference to the list: the
// methods that read or move the list take it as a parameter, and the view models pass their
// own. The zero value is an empty set, with no members, no run and no list. Nothing in it
// is written except by the UI thread, and the reads (for the labels and the background
// refresh) only read.
type forkCommitSet struct {
	// multiSelect holds the set and the drawn state; see commitMultiSelect.
	multiSelect commitMultiSelect
}

// commitListModel is the list that a commit list view model draws.
type commitListModel = ListViewModel[*models.Commit]

// CommitSetHolder is a commit list that can hold a multi-select set. The controllers reach
// the set through it, so that Commits, Sub-commits and Reflog share their checks.
type CommitSetHolder interface {
	HasCommitSet() bool
	MultiSelectActive() bool
	SelectedCommitHashes() []string
	CommitSelectionCounts() (selected int, notShown int)
	SelectedCommitHashesNotIn(commits []*models.Commit) []string
	RemoveFromCommitSet(hashes []string)
	NewUnreachableNotices(hashes []string) []string
	ClearMultiSelection()
	DropRun()
	ToggleCommitSelection()
	IsSelectingRange() bool
}

// CommitSetContext is a focused commit list that holds a set; it is also the view that a
// change to the set is posted to.
type CommitSetContext interface {
	types.Context
	CommitSetHolder
}

// HasCommitSet reports whether the explicit set holds any commit. A run alone, as
// upstream's range selection, does not make a set.
func (self *forkCommitSet) HasCommitSet() bool {
	return len(self.multiSelect.hashes) > 0
}

// multiSelectActive reports whether the pane is in multi-select mode: a set is held, or a
// run is in progress in list.
func (self *forkCommitSet) multiSelectActive(list *commitListModel) bool {
	return self.HasCommitSet() || (list != nil && list.IsSelectingRange())
}

// areMultipleItemsSelected overrides the cursor's answer, which knows only about a run. It
// is true with an explicit set too, so that single-item checks (singleItemSelected, the
// discard and custom patch paths) refuse a multi-selection rather than act on the cursor
// commit. The set's commands are scoped in controllers/fork_commit_multi_select.go.
func (self *forkCommitSet) areMultipleItemsSelected(list *commitListModel) bool {
	return self.HasCommitSet() || (list != nil && list.AreMultipleItemsSelected())
}

// dropRun cancels the run in progress without adding it to the set, and keeps the set. A
// view that is reopened on the same ref uses it, so that the reopening does not add a run
// that the user did not end.
func (self *forkCommitSet) dropRun(list *commitListModel) {
	if list != nil {
		list.CancelRangeSelect()
	}
}

// clearMultiSelection empties the set and cancels the run in progress, without adding the
// run to the set. This is what Escape does.
func (self *forkCommitSet) clearMultiSelection(list *commitListModel) {
	self.multiSelect.hashes = nil
	self.multiSelect.notified = nil
	if list != nil {
		list.CancelRangeSelect()
	}
}

// selectedCommitHashes returns the hashes that the selection covers, each once: the set,
// including the members that the loaded list does not show, and the run in progress. The
// order is not meaningful. In single-select mode it is empty; the cursor commit is the
// selection there, and getSelectedItems gives it.
func (self *forkCommitSet) selectedCommitHashes(list *commitListModel) []string {
	if !self.multiSelectActive(list) {
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
	for _, commit := range self.runCommits(list) {
		add(commit.Hash())
	}
	return hashes
}

// commitSelectionCounts returns how many commits the selection covers, and how many of
// them the loaded list does not show. Both are zero in single-select mode.
func (self *forkCommitSet) commitSelectionCounts(list *commitListModel) (selected int, notShown int) {
	hashes := self.selectedCommitHashes(list)
	var items []*models.Commit
	if list != nil {
		items = list.GetItems()
	}
	return len(hashes), len(self.SelectedCommitHashesNotIn(items))
}

// SelectedCommitHashesNotIn returns the set's hashes that commits does not show, in
// sorted order. A rebase todo item in the list does not count as showing its commit.
func (self *forkCommitSet) SelectedCommitHashesNotIn(commits []*models.Commit) []string {
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
func (self *forkCommitSet) RemoveFromCommitSet(hashes []string) {
	for _, hash := range hashes {
		self.removeFromSet(hash)
	}
}

// NewUnreachableNotices returns the set's hashes among hashes that have not been named
// yet, and records them as named. Hashes that are not in the set are ignored.
func (self *forkCommitSet) NewUnreachableNotices(hashes []string) []string {
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
func (self *forkCommitSet) removeFromSet(hash string) {
	delete(self.multiSelect.hashes, hash)
	delete(self.multiSelect.notified, hash)
}

// setSelectionAfterRewrite moves the cursor to idx for an operation that rewrites the
// commits of the run in progress: a drop, fixup or squash starting on the run. The run
// ends without joining the set, since the operation takes its commits. A plain move
// would add them to the set, and they would then be unreachable once rewritten.
func (self *forkCommitSet) setSelectionAfterRewrite(list *commitListModel, idx int) {
	if list == nil {
		return
	}
	list.CancelRangeSelect()
	list.SetSelection(idx)
}

// ForgetRewrittenCommits removes commits from the set before lazygitk rewrites them by
// moving them. A drag ends its run when the cursor moves during it, which adds the run to
// the set; the rewrite would then leave those hashes unreachable and a notice would name
// them. The moved commits are selected again as the run, so they leave the set.
func (self *forkCommitSet) ForgetRewrittenCommits(commits []*models.Commit) {
	for _, commit := range realCommits(commits) {
		self.removeFromSet(commit.Hash())
	}
}

// toggleCommitSelection is Space in the commit lists. With a run in progress it toggles the
// whole run and ends the run; otherwise it toggles the cursor commit in the set.
func (self *forkCommitSet) toggleCommitSelection(list *commitListModel) {
	if list == nil {
		return
	}
	if list.IsSelectingRange() {
		self.toggleCommits(self.runCommits(list))
		list.CancelRangeSelect()
		return
	}
	if commit := list.GetSelected(); commit != nil {
		self.toggleCommits([]*models.Commit{commit})
	}
}

// toggleCommits removes the commits from the set when all of them are in it already,
// and adds them otherwise.
func (self *forkCommitSet) toggleCommits(commits []*models.Commit) {
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

func (self *forkCommitSet) addToSet(commits []*models.Commit) {
	for _, commit := range realCommits(commits) {
		if self.multiSelect.hashes == nil {
			self.multiSelect.hashes = map[string]bool{}
		}
		self.multiSelect.hashes[commit.Hash()] = true
	}
}

// endRun ends the run in progress, if there is one, and its commits join the set.
func (self *forkCommitSet) endRun(list *commitListModel) {
	if list == nil || !list.IsSelectingRange() {
		return
	}
	self.addToSet(self.runCommitsAsDrawn(list))
	list.CancelRangeSelect()
}

// runCommits returns the commits of the run in progress, in list order, by the current
// indices of the run. It returns nil when no run is in progress.
func (self *forkCommitSet) runCommits(list *commitListModel) []*models.Commit {
	if list == nil || !list.IsSelectingRange() {
		return nil
	}
	commits := list.GetItems()
	start, end := list.GetSelectionRange()
	if start < 0 || start >= len(commits) {
		return nil
	}
	end = min(end, len(commits)-1)
	return commits[start : end+1]
}

// runCommitsAsDrawn is runCommits for a run that ends: when the list was replaced after
// the run was last drawn, the run is the one that was drawn, found again by its ends'
// hashes, since a refresh may have moved the indices. Commits that are gone are left out.
func (self *forkCommitSet) runCommitsAsDrawn(list *commitListModel) []*models.Commit {
	if list == nil {
		return nil
	}
	commits := list.GetItems()
	if self.multiSelect.drawn == nil || sameCommitList(commits, self.multiSelect.drawn) {
		return self.runCommits(list)
	}
	anchor := indexOfCommitHash(commits, self.multiSelect.drawnAnchor)
	cursor := indexOfCommitHash(commits, self.multiSelect.drawnCursor)
	if anchor < 0 || cursor < 0 {
		return nil
	}
	start, end := utils.SortRange(anchor, cursor)
	return commits[start : end+1]
}

// recordDrawn remembers the list and the run's ends as the view now draws them. It runs on
// the UI thread, from syncCommitSet.
func (self *forkCommitSet) recordDrawn(list *commitListModel, commits []*models.Commit) {
	cursor := list.GetSelectedLineIdx()
	anchor := cursor
	if idx, inRun := list.GetRangeStartIdx(); inRun {
		anchor = idx
	}
	self.multiSelect.drawn = commits
	self.multiSelect.drawnAnchor = commitHashAt(commits, anchor)
	self.multiSelect.drawnCursor = commitHashAt(commits, cursor)
}

// getSelectedItems returns the loaded commits that the selection covers. In multi-select
// mode that is the set and the run, in list order, and the indices of the first and the
// last of them. Members the loaded list does not show are left out; selectedCommitHashes
// has the whole selection. In single-select mode it is upstream's selection.
func (self *forkCommitSet) getSelectedItems(list *commitListModel) ([]*models.Commit, int, int) {
	if !self.multiSelectActive(list) {
		if list == nil {
			return nil, -1, -1
		}
		return list.GetSelectedItems()
	}
	inRun := map[*models.Commit]bool{}
	for _, commit := range self.runCommits(list) {
		inRun[commit] = true
	}
	var selected []*models.Commit
	first, last := -1, -1
	for i, commit := range list.GetItems() {
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

// getSelectedItemIds is getSelectedItems by ID.
func (self *forkCommitSet) getSelectedItemIds(list *commitListModel) ([]string, int, int) {
	items, first, last := self.getSelectedItems(list)
	ids := lo.Map(items, func(commit *models.Commit, _ int) string { return commit.ID() })
	return ids, first, last
}

// The methods below are the entry points that end a range. Moves within a sticky range keep
// it, as upstream does, so only a non-sticky run ends on a plain move.

func (self *forkCommitSet) moveSelectedLine(list *commitListModel, change int) {
	if list == nil {
		return
	}
	if _, _, mode := list.GetSelectionRangeAndMode(); mode == traits.RangeSelectModeNonSticky {
		self.endRun(list)
	}
	list.MoveSelectedLine(change)
}

func (self *forkCommitSet) setSelection(list *commitListModel, value int) {
	if list == nil {
		return
	}
	self.endRun(list)
	list.SetSelection(value)
}

func (self *forkCommitSet) cancelRangeSelect(list *commitListModel) {
	self.endRun(list)
}

func (self *forkCommitSet) collapseRangeSelectionToTop(list *commitListModel) {
	if list == nil {
		return
	}
	start, _ := list.GetSelectionRange()
	self.setSelection(list, start)
}

// toggleStickyRange ends a range when one is in progress, as upstream does, and starts a
// sticky range otherwise. Turning it off adds the range to the set.
func (self *forkCommitSet) toggleStickyRange(list *commitListModel) {
	if list == nil {
		return
	}
	if list.IsSelectingRange() {
		self.endRun(list)
		return
	}
	list.ToggleStickyRange()
}

// The wrappers below are the view models' side of the set. Each passes its own list, so that
// the list's cursor entry points are the set's, and a view model made as a literal works
// without a constructor. The exported names shadow ListCursor's, and the holder methods of
// CommitSetHolder that take the list live here, not on the set.

// Commits: the list is the view model's own.

func (self *LocalCommitsViewModel) commitSetList() *commitListModel { return self.ListViewModel }

func (self *LocalCommitsViewModel) GetSelectedItems() ([]*models.Commit, int, int) {
	return self.forkCommitSet.getSelectedItems(self.ListViewModel)
}

func (self *LocalCommitsViewModel) GetSelectedItemIds() ([]string, int, int) {
	return self.forkCommitSet.getSelectedItemIds(self.ListViewModel)
}

func (self *LocalCommitsViewModel) AreMultipleItemsSelected() bool {
	return self.forkCommitSet.areMultipleItemsSelected(self.ListViewModel)
}

func (self *LocalCommitsViewModel) MultiSelectActive() bool {
	return self.forkCommitSet.multiSelectActive(self.ListViewModel)
}

func (self *LocalCommitsViewModel) SelectedCommitHashes() []string {
	return self.forkCommitSet.selectedCommitHashes(self.ListViewModel)
}

func (self *LocalCommitsViewModel) CommitSelectionCounts() (int, int) {
	return self.forkCommitSet.commitSelectionCounts(self.ListViewModel)
}

func (self *LocalCommitsViewModel) ClearMultiSelection() {
	self.forkCommitSet.clearMultiSelection(self.ListViewModel)
}

func (self *LocalCommitsViewModel) DropRun() {
	self.forkCommitSet.dropRun(self.ListViewModel)
}

func (self *LocalCommitsViewModel) ToggleCommitSelection() {
	self.forkCommitSet.toggleCommitSelection(self.ListViewModel)
}

func (self *LocalCommitsViewModel) SetSelectionAfterRewrite(idx int) {
	self.forkCommitSet.setSelectionAfterRewrite(self.ListViewModel, idx)
}

func (self *LocalCommitsViewModel) MoveSelectedLine(change int) {
	self.forkCommitSet.moveSelectedLine(self.ListViewModel, change)
}

func (self *LocalCommitsViewModel) SetSelection(value int) {
	self.forkCommitSet.setSelection(self.ListViewModel, value)
}

func (self *LocalCommitsViewModel) CancelRangeSelect() {
	self.forkCommitSet.cancelRangeSelect(self.ListViewModel)
}

func (self *LocalCommitsViewModel) CollapseRangeSelectionToTop() {
	self.forkCommitSet.collapseRangeSelectionToTop(self.ListViewModel)
}

func (self *LocalCommitsViewModel) ToggleStickyRange() {
	self.forkCommitSet.toggleStickyRange(self.ListViewModel)
}

// Sub-commits: the list is the view model's own.

func (self *SubCommitsViewModel) commitSetList() *commitListModel { return self.ListViewModel }

func (self *SubCommitsViewModel) GetSelectedItems() ([]*models.Commit, int, int) {
	return self.forkCommitSet.getSelectedItems(self.ListViewModel)
}

func (self *SubCommitsViewModel) GetSelectedItemIds() ([]string, int, int) {
	return self.forkCommitSet.getSelectedItemIds(self.ListViewModel)
}

func (self *SubCommitsViewModel) AreMultipleItemsSelected() bool {
	return self.forkCommitSet.areMultipleItemsSelected(self.ListViewModel)
}

func (self *SubCommitsViewModel) MultiSelectActive() bool {
	return self.forkCommitSet.multiSelectActive(self.ListViewModel)
}

func (self *SubCommitsViewModel) SelectedCommitHashes() []string {
	return self.forkCommitSet.selectedCommitHashes(self.ListViewModel)
}

func (self *SubCommitsViewModel) CommitSelectionCounts() (int, int) {
	return self.forkCommitSet.commitSelectionCounts(self.ListViewModel)
}

func (self *SubCommitsViewModel) ClearMultiSelection() {
	self.forkCommitSet.clearMultiSelection(self.ListViewModel)
}

func (self *SubCommitsViewModel) DropRun() {
	self.forkCommitSet.dropRun(self.ListViewModel)
}

func (self *SubCommitsViewModel) ToggleCommitSelection() {
	self.forkCommitSet.toggleCommitSelection(self.ListViewModel)
}

func (self *SubCommitsViewModel) MoveSelectedLine(change int) {
	self.forkCommitSet.moveSelectedLine(self.ListViewModel, change)
}

func (self *SubCommitsViewModel) SetSelection(value int) {
	self.forkCommitSet.setSelection(self.ListViewModel, value)
}

func (self *SubCommitsViewModel) CancelRangeSelect() {
	self.forkCommitSet.cancelRangeSelect(self.ListViewModel)
}

func (self *SubCommitsViewModel) CollapseRangeSelectionToTop() {
	self.forkCommitSet.collapseRangeSelectionToTop(self.ListViewModel)
}

func (self *SubCommitsViewModel) ToggleStickyRange() {
	self.forkCommitSet.toggleStickyRange(self.ListViewModel)
}

// Reflog: the list is the filtered list's, under the filter view model.

func (self *ReflogCommitsContext) commitSetList() *commitListModel {
	return self.FilteredListViewModel.ListViewModel
}

func (self *ReflogCommitsContext) GetSelectedItems() ([]*models.Commit, int, int) {
	return self.forkCommitSet.getSelectedItems(self.commitSetList())
}

func (self *ReflogCommitsContext) GetSelectedItemIds() ([]string, int, int) {
	return self.forkCommitSet.getSelectedItemIds(self.commitSetList())
}

func (self *ReflogCommitsContext) AreMultipleItemsSelected() bool {
	return self.forkCommitSet.areMultipleItemsSelected(self.commitSetList())
}

func (self *ReflogCommitsContext) MultiSelectActive() bool {
	return self.forkCommitSet.multiSelectActive(self.commitSetList())
}

func (self *ReflogCommitsContext) SelectedCommitHashes() []string {
	return self.forkCommitSet.selectedCommitHashes(self.commitSetList())
}

func (self *ReflogCommitsContext) CommitSelectionCounts() (int, int) {
	return self.forkCommitSet.commitSelectionCounts(self.commitSetList())
}

func (self *ReflogCommitsContext) ClearMultiSelection() {
	self.forkCommitSet.clearMultiSelection(self.commitSetList())
}

func (self *ReflogCommitsContext) DropRun() {
	self.forkCommitSet.dropRun(self.commitSetList())
}

func (self *ReflogCommitsContext) ToggleCommitSelection() {
	self.forkCommitSet.toggleCommitSelection(self.commitSetList())
}

func (self *ReflogCommitsContext) MoveSelectedLine(change int) {
	self.forkCommitSet.moveSelectedLine(self.commitSetList(), change)
}

func (self *ReflogCommitsContext) SetSelection(value int) {
	self.forkCommitSet.setSelection(self.commitSetList(), value)
}

func (self *ReflogCommitsContext) CancelRangeSelect() {
	self.forkCommitSet.cancelRangeSelect(self.commitSetList())
}

func (self *ReflogCommitsContext) CollapseRangeSelectionToTop() {
	self.forkCommitSet.collapseRangeSelectionToTop(self.commitSetList())
}

func (self *ReflogCommitsContext) ToggleStickyRange() {
	self.forkCommitSet.toggleStickyRange(self.commitSetList())
}

// forkCommitSetView is a commit list view that draws its set: Commits, Sub-commits and
// Reflog. It is satisfied by the contexts through their embedded view models.
type forkCommitSetView interface {
	forkSet() *forkCommitSet
	commitSetList() *commitListModel
	GetView() *gocui.View
	GetCommits() []*models.Commit
	GetSelectedLineIdx() int
	TotalContentHeight() int
	modelToViewIndexConverter() func(int) int
}

// forkSet returns the set of the view model that embeds it.
func (self *forkCommitSet) forkSet() *forkCommitSet {
	return self
}

// commitSetSubtitle is the subtitle of a list while its set is held: how many commits it
// holds, and how many of them the loaded list does not show. It is empty otherwise.
func commitSetSubtitle(set *forkCommitSet, list *commitListModel, tr *i18n.TranslationSet) string {
	if !set.HasCommitSet() {
		return ""
	}
	selected, notShown := set.commitSelectionCounts(list)
	if notShown == 0 {
		return fmt.Sprintf(tr.CommitSetCount, selected)
	}
	return fmt.Sprintf(tr.CommitSetCountNotShown, selected, notShown)
}

// syncCommitSet brings the view of a list in line with its multi-select state: the
// highlighted lines of the set and the run, the cursor marker and the subtitle. It runs
// whenever the list is focused, moved or rendered, so it costs time in the size of the
// loaded list. A highlighted line is a row whose commit's hash is in the set, so every
// row of a commit that appears twice (in the reflog) is highlighted.
func syncCommitSet(ctx forkCommitSetView, tr *i18n.TranslationSet) {
	set := ctx.forkSet()
	list := ctx.commitSetList()
	commits := ctx.GetCommits()

	view := ctx.GetView()
	view.Subtitle = commitSetSubtitle(set, list, tr)
	if !set.multiSelectActive(list) || len(commits) == 0 {
		view.SetSelectedLines(nil)
		view.SetInclusionGutter(false, nil)
		set.recordDrawn(list, commits)
		return
	}

	convert := ctx.modelToViewIndexConverter()
	lines := make([]bool, ctx.TotalContentHeight())
	inRun := map[*models.Commit]bool{}
	for _, commit := range set.runCommits(list) {
		inRun[commit] = true
	}
	for i, commit := range commits {
		if inRun[commit] || (!commit.IsTODO() && set.multiSelect.hashes[commit.Hash()]) {
			lines[convert(i)] = true
		}
	}
	marks := make([]bool, len(lines))
	marks[convert(ctx.GetSelectedLineIdx())] = true

	view.SetSelectedLines(lines)
	view.SetInclusionGutter(true, marks)
	set.recordDrawn(list, commits)
}

// HandleFocus, FocusLine and HandleRender sync the view after the list's own work. The
// list trait calls its own FocusLine from HandleFocus, so FocusLine is overridden too.
// Each context that holds a set defines these three itself: ListContextTrait defines them
// at the same depth as the set, so promoting them from the set would shadow the sync.

func (self *LocalCommitsContext) HandleFocus(opts types.OnFocusOpts) {
	self.ListContextTrait.HandleFocus(opts)
	syncCommitSet(self, self.ListContextTrait.c.Tr)
}

func (self *LocalCommitsContext) FocusLine(scrollIntoView bool) {
	self.ListContextTrait.FocusLine(scrollIntoView)
	syncCommitSet(self, self.ListContextTrait.c.Tr)
}

func (self *LocalCommitsContext) HandleRender() {
	self.ListContextTrait.HandleRender()
	syncCommitSet(self, self.ListContextTrait.c.Tr)
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
