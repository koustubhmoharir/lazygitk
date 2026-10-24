package controllers

import (
	"reflect"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	commonpkg "github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/modes/cherrypicking"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type forkCherryMarksGuiCommon struct {
	types.IGuiCommon
	modes     *types.Modes
	contexts  *context.ContextTree
	refreshed []types.Context
}

func (f *forkCherryMarksGuiCommon) Modes() *types.Modes { return f.modes }
func (f *forkCherryMarksGuiCommon) PostRefreshUpdate(ctx types.Context) {
	f.refreshed = append(f.refreshed, ctx)
}

func forkCherryMarksController(selection *historicalDiffCommitsStub) (*BasicCommitsController, *forkCherryMarksGuiCommon) {
	gui := &forkCherryMarksGuiCommon{
		modes:    &types.Modes{CherryPicking: cherrypicking.New()},
		contexts: &context.ContextTree{},
	}
	common := commonpkg.NewDummyCommon()
	helpersCommon := &helpers.HelperCommon{
		Common:       common,
		IGetContexts: historicalDiffContextsStub{tree: gui.contexts},
		IGuiCommon:   gui,
	}
	return NewBasicCommitsController(NewControllerCommon(helpersCommon, nil), selection), gui
}

func TestForkCherryMarksBindings(t *testing.T) {
	controller := historicalTestBasicController(nil, 0, 0)
	bindings := controller.addCherryMarkBindings(nil)
	if len(bindings) != 2 {
		t.Fatalf("got %d mark bindings, want 2", len(bindings))
	}
	want := []struct{ id, verb string }{
		{"commit.mark-cherry-pick", "mark-cherry-pick"},
		{"commit.unmark-cherry-pick", "unmark-cherry-pick"},
	}
	for i, binding := range bindings {
		if binding.CommandID != want[i].id || binding.Verb != want[i].verb {
			t.Errorf("binding %d identity = %q/%q, want %q/%q", i, binding.CommandID, binding.Verb, want[i].id, want[i].verb)
		}
		if len(binding.Keys) != 0 {
			t.Errorf("binding %q unexpectedly has keys %v", binding.CommandID, binding.Keys)
		}
	}
}

func TestForkCherryMarksBindingGuard(t *testing.T) {
	valid := historicalTestCommit("0123456789abcdef0123456789abcdef01234567")
	for _, tc := range []struct {
		name       string
		items      []*models.Commit
		start, end int
		blocked    bool
	}{
		{name: "empty", blocked: true},
		{name: "nil", items: []*models.Commit{nil}, blocked: true},
		{name: "synthetic", items: []*models.Commit{historicalTestCommit("HEAD")}, blocked: true},
		{name: "short oid", items: []*models.Commit{historicalTestCommit("0123456")}, blocked: true},
		{name: "full oid", items: []*models.Commit{valid}},
		{name: "selected range includes invalid", items: []*models.Commit{valid, historicalTestCommit("HEAD")}, start: 0, end: 1, blocked: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			controller, _ := forkCherryMarksController(&historicalDiffCommitsStub{items: tc.items, start: tc.start, end: tc.end})
			for _, binding := range controller.addCherryMarkBindings(nil) {
				got := binding.GetDisabledReason() != nil
				if got != tc.blocked {
					t.Errorf("%q blocked = %v, want %v", binding.CommandID, got, tc.blocked)
				}
			}
		})
	}
}

func TestForkCherryMarksHandlersUseCurrentSelectionAndRefreshAllContexts(t *testing.T) {
	a := historicalTestCommit("0123456789abcdef0123456789abcdef01234567")
	b := historicalTestCommit("1123456789abcdef0123456789abcdef01234567")
	c := historicalTestCommit("2123456789abcdef0123456789abcdef01234567")
	selection := &historicalDiffCommitsStub{items: []*models.Commit{a}}
	controller, gui := forkCherryMarksController(selection)
	local, reflog, sub := &context.LocalCommitsContext{}, &context.ReflogCommitsContext{}, &context.SubCommitsContext{}
	gui.contexts.LocalCommits, gui.contexts.ReflogCommits, gui.contexts.SubCommits = local, reflog, sub
	bindings := controller.addCherryMarkBindings(nil)
	mark, unmark := bindings[0].Handler, bindings[1].Handler

	if err := mark(); err != nil {
		t.Fatal(err)
	}
	if err := mark(); err != nil {
		t.Fatal(err)
	}
	selection.items = []*models.Commit{b}
	if err := mark(); err != nil {
		t.Fatal(err)
	}
	if got := cherryMarkHashes(gui.modes.CherryPicking.CherryPickedCommits); !reflect.DeepEqual(got, []string{a.Hash(), b.Hash()}) {
		t.Fatalf("marks after repeated marks = %v", got)
	}
	selection.items = []*models.Commit{c}
	if err := unmark(); err != nil {
		t.Fatal(err)
	}
	if got := cherryMarkHashes(gui.modes.CherryPicking.CherryPickedCommits); !reflect.DeepEqual(got, []string{a.Hash(), b.Hash()}) {
		t.Fatalf("unmark changed unrelated marks: %v", got)
	}
	selection.items = []*models.Commit{b}
	if err := unmark(); err != nil {
		t.Fatal(err)
	}
	if got := cherryMarkHashes(gui.modes.CherryPicking.CherryPickedCommits); !reflect.DeepEqual(got, []string{a.Hash()}) {
		t.Fatalf("unmark selected marks = %v", got)
	}
	if got, want := gui.refreshed, []types.Context{local, reflog, sub, local, reflog, sub, local, reflog, sub, local, reflog, sub, local, reflog, sub}; !reflect.DeepEqual(got, want) {
		t.Fatalf("refresh contexts = %v, want %v", got, want)
	}
}

func TestForkCherryMarksStoreSnapshots(t *testing.T) {
	commit := historicalTestCommit("0123456789abcdef0123456789abcdef01234567")
	controller, gui := forkCherryMarksController(&historicalDiffCommitsStub{items: []*models.Commit{commit}})
	if err := controller.setCherryMarks(true); err != nil {
		t.Fatal(err)
	}
	stored := gui.modes.CherryPicking.CherryPickedCommits[0]
	if stored == commit || stored.Hash() != commit.Hash() {
		t.Fatalf("stored commit = %p (%s), source = %p (%s); want same hash in a snapshot", stored, stored.Hash(), commit, commit.Hash())
	}
}

func cherryMarkHashes(commits []*models.Commit) []string {
	hashes := make([]string, 0, len(commits))
	for _, commit := range commits {
		hashes = append(hashes, commit.Hash())
	}
	return hashes
}
