package gui

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/gui/mergeconflicts"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

type forkOriginTestList struct {
	types.Context
	ids         []string
	first, last int
}

func TestForkCommandOriginChecksConflictFileAndSelectedHunk(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	pane := gui.State.Contexts.MergeConflicts
	gui.State.ContextMgr.ContextStack = []types.Context{gui.State.Contexts.Files, pane}
	state := mergeconflicts.NewState()
	content := "<<<<<<< ours\nours\n=======\ntheirs\n>>>>>>> theirs\n"
	state.SetContent(content, "file", 7)
	pane.SetState(state)
	origin := gui.captureForkCommandOrigin()
	assert.True(t, origin.valid(gui))
	state.SelectNextConflictHunk()
	assert.False(t, origin.valid(gui), "a prefix cannot resolve a newly selected side")
	state.SelectPrevConflictHunk()
	assert.True(t, origin.valid(gui))
	state.SetContent(content, "different-file", 7)
	assert.False(t, origin.valid(gui), "identical text in another file is a different target")
	state.SetContent(content+"changed\n", "file", 7)
	assert.False(t, origin.valid(gui), "refreshed conflict contents require a fresh prefix")
	pane.SetState(nil)
	assert.False(t, origin.valid(gui))
	origin = gui.captureForkCommandOrigin()
	pane.SetState(state)
	assert.False(t, origin.valid(gui), "loading conflict state cannot activate an old empty prefix")
}

func (list *forkOriginTestList) GetSelectedItemIds() ([]string, int, int) {
	return list.ids, list.first, list.last
}
func (list *forkOriginTestList) IsListContext() {}

// Other list methods are deliberately unavailable: capturing a command target
// must read stable selection identity without inspecting or modifying models.
func (list *forkOriginTestList) GetSelectedItemId() string       { return list.ids[0] }
func (list *forkOriginTestList) IsItemVisible(types.HasUrn) bool { panic("unused") }
func (list *forkOriginTestList) GetList() types.IList            { panic("unused") }
func (list *forkOriginTestList) ViewIndexToModelIndex(int) int   { panic("unused") }
func (list *forkOriginTestList) ModelIndexToViewIndex(int) int   { panic("unused") }
func (list *forkOriginTestList) RangeSelectEnabled() bool        { panic("unused") }
func (list *forkOriginTestList) RenderOnlyVisibleLines() bool    { panic("unused") }
func (list *forkOriginTestList) SetNeedRerenderVisibleLines()    { panic("unused") }
func (list *forkOriginTestList) IndexForGotoBottom() int         { panic("unused") }

func TestForkCommandOriginRejectsSelectionContextAndRepositoryChanges(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	selected := &forkOriginTestList{Context: gui.State.Contexts.Files, ids: []string{"a", "b"}, first: 1, last: 2}
	gui.State.ContextMgr.ContextStack = []types.Context{selected}
	origin := gui.captureForkCommandOrigin()
	assert.True(t, origin.valid(gui))
	selected.ids[1] = "other"
	assert.False(t, origin.valid(gui), "selection backing slice must be copied")
	selected.ids[1] = "b"
	selected.first = 0
	assert.False(t, origin.valid(gui), "same IDs at a different range need a new prefix")
	selected.first = 1
	gui.repoGeneration.Add(1)
	assert.False(t, origin.valid(gui))
	origin = gui.captureForkCommandOrigin()
	gui.git = &commands.GitCommand{}
	assert.False(t, origin.valid(gui))
	origin = gui.captureForkCommandOrigin()
	gui.State.ContextMgr.ContextStack = []types.Context{gui.State.Contexts.Files}
	assert.False(t, origin.valid(gui), "same view with a replaced context cannot reuse a prefix")
}

func TestForkCommandOriginChecksDiffContentAndUnderlyingSelection(t *testing.T) {
	gui := newForkKeymapTestGui(t)
	side := &forkOriginTestList{Context: gui.State.Contexts.Files, ids: []string{"a"}}
	pane := gui.State.Contexts.Normal
	gui.State.ContextMgr.ContextStack = []types.Context{side, pane}
	pane.GetView().WriteString("first\nsecond\n")
	origin := gui.captureForkCommandOrigin()
	assert.True(t, origin.valid(gui))
	pane.DiffSelectState().UserEnabledHunkMode = true
	assert.False(t, origin.valid(gui))
	pane.DiffSelectState().UserEnabledHunkMode = false
	side.ids[0] = "different-file"
	assert.False(t, origin.valid(gui))
	side.ids[0] = "a"
	pane.GetView().WriteString("new content\n")
	assert.False(t, origin.valid(gui), "refreshed diff text invalidates the captured line selection")
}
