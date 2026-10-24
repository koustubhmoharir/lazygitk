package gui

import (
	"crypto/sha256"
	"slices"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// A prefix captures its target before opening a popup. Background refreshes may
// replace models or move rows while the menu is open; never retarget a leaf to
// whatever happens to be selected when the continuation arrives.
type forkCommandOrigin struct {
	context    types.Context
	side       types.Context
	git        *commands.GitCommand
	generation int32
	selections []forkCommandSelection
	diff       *forkCommandDiffSelection
}

type forkCommandSelection struct {
	context     types.IListContext
	ids         []string
	first, last int
}

type forkCommandDiffSelection struct {
	context     types.DiffPaneContext
	content     [32]byte
	first, last int
	selectable  bool
	state       types.DiffSelectState
}

func (gui *Gui) captureForkCommandOrigin() *forkCommandOrigin {
	origin := &forkCommandOrigin{
		context: gui.State.ContextMgr.CurrentStatic(),
		side:    gui.State.ContextMgr.CurrentSide(),
		git:     gui.c.Git(), generation: gui.repoGeneration.Load(),
	}
	for _, ctx := range []types.Context{origin.context, origin.side} {
		if ctx == origin.side && ctx == origin.context && len(origin.selections) > 0 {
			continue
		}
		if list, ok := ctx.(types.IListContext); ok {
			ids, first, last := list.GetSelectedItemIds()
			origin.selections = append(origin.selections, forkCommandSelection{list, slices.Clone(ids), first, last})
		}
	}
	if pane, ok := origin.context.(types.DiffPaneContext); ok {
		first, last, selectable := pane.GetView().SelectedBufferLineRange()
		origin.diff = &forkCommandDiffSelection{
			context: pane, content: sha256.Sum256([]byte(pane.GetView().Buffer())),
			first: first, last: last, selectable: selectable, state: *pane.DiffSelectState(),
		}
	}
	return origin
}

func (origin *forkCommandOrigin) valid(gui *Gui) bool {
	if origin == nil || origin.generation != gui.repoGeneration.Load() || origin.git != gui.c.Git() ||
		origin.context != gui.State.ContextMgr.CurrentStatic() || origin.side != gui.State.ContextMgr.CurrentSide() {
		return false
	}
	for _, selected := range origin.selections {
		ids, first, last := selected.context.GetSelectedItemIds()
		if first != selected.first || last != selected.last || !slices.Equal(ids, selected.ids) {
			return false
		}
	}
	if diff := origin.diff; diff != nil {
		first, last, selectable := diff.context.GetView().SelectedBufferLineRange()
		if first != diff.first || last != diff.last || selectable != diff.selectable ||
			*diff.context.DiffSelectState() != diff.state || sha256.Sum256([]byte(diff.context.GetView().Buffer())) != diff.content {
			return false
		}
	}
	return true
}
