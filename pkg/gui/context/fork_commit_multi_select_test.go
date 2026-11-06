package context

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// newMultiSelectModel returns a commits view model over a list of the given hashes,
// newest first as the list is shown, and the variable that holds the list, so that a
// test can replace it the way a refresh does.
func newMultiSelectModel(hashes ...string) (*LocalCommitsViewModel, *[]*models.Commit) {
	pool := &utils.StringPool{}
	list := &[]*models.Commit{}
	for _, hash := range hashes {
		*list = append(*list, models.NewCommit(pool, models.NewCommitOpts{Hash: hash}))
	}
	vm := &LocalCommitsViewModel{ListViewModel: NewListViewModel(func() []*models.Commit { return *list })}
	return vm, list
}

func selectedHashes(vm *LocalCommitsViewModel) []string {
	items, _, _ := vm.GetSelectedItems()
	hashes := []string{}
	for _, item := range items {
		hashes = append(hashes, item.Hash())
	}
	return hashes
}

func setHashes(vm *LocalCommitsViewModel) []string {
	hashes := []string{}
	for _, commit := range vm.GetCommits() {
		if vm.multiSelect.hashes[commit.Hash()] {
			hashes = append(hashes, commit.Hash())
		}
	}
	return hashes
}

func TestMultiSelectSingleModeIsUpstreamsCursor(t *testing.T) {
	vm, _ := newMultiSelectModel("h0", "h1", "h2", "h3")
	vm.SetSelection(2)

	assert.False(t, vm.MultiSelectActive())
	assert.Equal(t, []string{"h2"}, selectedHashes(vm))
	_, first, last := vm.GetSelectedItems()
	assert.Equal(t, []int{2, 2}, []int{first, last})
}

func TestMultiSelectSpaceAddsCursorAndPlainMovesKeepTheSet(t *testing.T) {
	vm, _ := newMultiSelectModel("h0", "h1", "h2", "h3", "h4", "h5")
	vm.SetSelection(5)
	vm.ToggleCommitSelection()
	assert.True(t, vm.MultiSelectActive())

	vm.MoveSelectedLine(-1)
	vm.MoveSelectedLine(-1)
	assert.Equal(t, 3, vm.GetSelectedLineIdx())
	assert.Equal(t, []string{"h5"}, setHashes(vm))
	assert.Equal(t, []string{"h5"}, selectedHashes(vm))

	vm.ToggleCommitSelection()
	assert.Equal(t, []string{"h3", "h5"}, setHashes(vm), "the cursor commit joins the set")
	vm.ToggleCommitSelection()
	assert.Equal(t, []string{"h5"}, setHashes(vm), "and leaves it again")
	vm.SetSelection(5)
	vm.ToggleCommitSelection()
	assert.False(t, vm.MultiSelectActive())
	assert.Empty(t, setHashes(vm))
}

func TestMultiSelectShiftRunJoinsTheSetWhenAPlainMoveEndsIt(t *testing.T) {
	vm, _ := newMultiSelectModel("h0", "h1", "h2", "h3", "h4")
	vm.SetSelection(3)

	vm.ExpandNonStickyRange(-1)
	vm.ExpandNonStickyRange(-1)
	assert.True(t, vm.MultiSelectActive())
	assert.Equal(t, []string{"h1", "h2", "h3"}, selectedHashes(vm))
	assert.Empty(t, setHashes(vm), "the run is not in the set until it ends")

	vm.MoveSelectedLine(1)
	assert.False(t, vm.IsSelectingRange())
	assert.Equal(t, 2, vm.GetSelectedLineIdx())
	assert.Equal(t, []string{"h1", "h2", "h3"}, setHashes(vm))
}

func TestMultiSelectShiftRunShrinksBeforeItEnds(t *testing.T) {
	vm, _ := newMultiSelectModel("h0", "h1", "h2", "h3", "h4")
	vm.SetSelection(1)

	vm.ExpandNonStickyRange(1)
	vm.ExpandNonStickyRange(1)
	vm.ExpandNonStickyRange(-1)
	assert.Equal(t, []string{"h1", "h2"}, selectedHashes(vm))

	vm.SetSelection(0)
	assert.Equal(t, []string{"h1", "h2"}, setHashes(vm))
}

func TestMultiSelectStickyRangeEndsOnlyWhenToggledOff(t *testing.T) {
	vm, _ := newMultiSelectModel("h0", "h1", "h2", "h3")
	vm.SetSelection(0)

	vm.ToggleStickyRange()
	vm.MoveSelectedLine(1)
	vm.MoveSelectedLine(1)
	assert.True(t, vm.IsSelectingRange(), "a plain move keeps a sticky range")
	assert.Empty(t, setHashes(vm))

	vm.ToggleStickyRange()
	assert.False(t, vm.IsSelectingRange())
	assert.Equal(t, []string{"h0", "h1", "h2"}, setHashes(vm))
}

func TestMultiSelectSpaceOnARunTogglesTheWholeRun(t *testing.T) {
	vm, _ := newMultiSelectModel("h0", "h1", "h2", "h3", "h4")
	vm.SetSelection(1)
	vm.ToggleCommitSelection()
	vm.SetSelection(2)
	vm.ToggleCommitSelection()
	assert.Equal(t, []string{"h1", "h2"}, setHashes(vm))

	// A run that covers a commit outside the set adds the whole run.
	vm.SetSelection(0)
	vm.ExpandNonStickyRange(1)
	vm.ExpandNonStickyRange(1)
	vm.ToggleCommitSelection()
	assert.False(t, vm.IsSelectingRange())
	assert.Equal(t, []string{"h0", "h1", "h2"}, setHashes(vm))

	// A run whose commits are all in the set removes them all, and ends the run.
	vm.SetSelection(0)
	vm.ExpandNonStickyRange(1)
	vm.ExpandNonStickyRange(1)
	vm.ToggleCommitSelection()
	assert.False(t, vm.IsSelectingRange())
	assert.Empty(t, setHashes(vm))
}

func TestMultiSelectSelectionIsInListOrderWithFirstAndLast(t *testing.T) {
	vm, _ := newMultiSelectModel("h0", "h1", "h2", "h3", "h4")
	vm.SetSelection(4)
	vm.ToggleCommitSelection()
	vm.SetSelection(1)
	vm.ToggleCommitSelection()

	items, first, last := vm.GetSelectedItems()
	assert.Equal(t, []string{"h1", "h4"}, []string{items[0].Hash(), items[1].Hash()})
	assert.Equal(t, []int{1, 4}, []int{first, last})
}

func TestMultiSelectClearDropsTheSetAndRunWithoutAddingTheRun(t *testing.T) {
	vm, _ := newMultiSelectModel("h0", "h1", "h2", "h3")
	vm.SetSelection(0)
	vm.ToggleCommitSelection()
	vm.SetSelection(2)
	vm.ExpandNonStickyRange(1)

	vm.ClearMultiSelection()
	assert.False(t, vm.MultiSelectActive())
	assert.Empty(t, setHashes(vm))
	assert.Equal(t, 3, vm.GetSelectedLineIdx(), "the cursor stays where the run left it")
	assert.Equal(t, []string{"h3"}, selectedHashes(vm))
}

func TestMultiSelectSetSurvivesListReplacement(t *testing.T) {
	vm, list := newMultiSelectModel("h0", "h1", "h2", "h3")
	vm.SetSelection(0)
	vm.ToggleCommitSelection()
	vm.SetSelection(2)
	vm.ToggleCommitSelection()

	// A refresh whose list omits h2, as a filter or the commit limit does. h2 stays in
	// the set: it is not shown, and it is counted as not shown.
	full := *list
	*list = full[:1:1]
	assert.Equal(t, []string{"h0"}, selectedHashes(vm), "only the loaded selection is listed")
	assert.ElementsMatch(t, []string{"h0", "h2"}, vm.SelectedCommitHashes())
	selected, notShown := vm.CommitSelectionCounts()
	assert.Equal(t, []int{2, 1}, []int{selected, notShown})

	// When h2 is loaded again, it is selected again.
	*list = full
	assert.Equal(t, []string{"h0", "h2"}, selectedHashes(vm))
	selected, notShown = vm.CommitSelectionCounts()
	assert.Equal(t, []int{2, 0}, []int{selected, notShown})
}

func TestMultiSelectSelectedCommitHashesCoversHiddenMembersAndTheRun(t *testing.T) {
	vm, list := newMultiSelectModel("h0", "h1", "h2", "h3", "h4")
	vm.SetSelection(0)
	vm.ToggleCommitSelection()
	vm.SetSelection(2)
	vm.ToggleCommitSelection()

	// h0 is hidden by the list, and the run covers h1 and h2 in the new list.
	*list = (*list)[1:]
	vm.SetSelection(0)
	vm.ExpandNonStickyRange(1)
	assert.ElementsMatch(t, []string{"h0", "h1", "h2"}, vm.SelectedCommitHashes(), "h2 is in both the set and the run, and is listed once")
	selected, notShown := vm.CommitSelectionCounts()
	assert.Equal(t, []int{3, 1}, []int{selected, notShown})
}

func TestMultiSelectSelectedCommitHashesIsEmptyInSingleSelectMode(t *testing.T) {
	vm, _ := newMultiSelectModel("h0", "h1")
	vm.SetSelection(1)
	assert.Empty(t, vm.SelectedCommitHashes())
	selected, notShown := vm.CommitSelectionCounts()
	assert.Equal(t, []int{0, 0}, []int{selected, notShown})
}

func TestMultiSelectRemovedAndNoticedHashes(t *testing.T) {
	vm, _ := newMultiSelectModel("h0", "h1", "h2")
	vm.SetSelection(0)
	vm.ToggleCommitSelection()
	vm.SetSelection(1)
	vm.ToggleCommitSelection()

	// A rewritten commit is noticed once, while it stays in the set.
	assert.Equal(t, []string{"h1"}, vm.NewUnreachableNotices([]string{"h1", "x"}))
	assert.Empty(t, vm.NewUnreachableNotices([]string{"h1"}))

	// Leaving the set forgets the notice, so the next time it is rewritten it is named again.
	vm.SetSelection(1)
	vm.ToggleCommitSelection()
	vm.SetSelection(1)
	vm.ToggleCommitSelection()
	assert.Equal(t, []string{"h1"}, vm.NewUnreachableNotices([]string{"h1"}))

	vm.RemoveFromCommitSet([]string{"h0"})
	assert.Equal(t, []string{"h1"}, setHashes(vm))
	assert.Empty(t, vm.NewUnreachableNotices([]string{"h0"}), "a hash that is no longer selected is not named")
}

func TestMultiSelectRunEndedAfterListReplacementUsesTheDrawnCommits(t *testing.T) {
	vm, list := newMultiSelectModel("h0", "h1", "h2", "h3", "h4")
	vm.SetSelection(1)
	vm.ExpandNonStickyRange(1)
	vm.ExpandNonStickyRange(1)
	vm.recordDrawn(vm.GetCommits())

	// A refresh adds a commit at the top, so the run's indices now point one higher.
	pool := &utils.StringPool{}
	*list = append([]*models.Commit{models.NewCommit(pool, models.NewCommitOpts{Hash: "new"})}, *list...)

	// The run that was drawn is h1 to h3; it joins the set, not the commits at its indices.
	vm.SetSelection(0)
	assert.Equal(t, []string{"h1", "h2", "h3"}, setHashes(vm))
}
