package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	guicontext "github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type branchSyncBindingSide struct {
	types.IContextMgr
	side types.Context
}

func (mgr *branchSyncBindingSide) CurrentSide() types.Context { return mgr.side }

type branchSyncBindingState struct {
	types.IStateAccessor
	operations map[string]types.ItemOperation
}

func (state *branchSyncBindingState) GetItemOperation(item types.HasUrn) types.ItemOperation {
	return state.operations[item.URN()]
}

type branchSyncBindingUI struct {
	types.IGuiCommon
	git     *commands.GitCommand
	model   types.Model
	state   types.IStateAccessor
	context types.IContextMgr
}

func (ui *branchSyncBindingUI) Git() *commands.GitCommand   { return ui.git }
func (ui *branchSyncBindingUI) Model() *types.Model         { return &ui.model }
func (ui *branchSyncBindingUI) State() types.IStateAccessor { return ui.state }
func (ui *branchSyncBindingUI) Context() types.IContextMgr  { return ui.context }

func branchSyncBindingFixture(side types.Context, branches []*models.Branch, history models.Ref) *SyncController {
	cmn := common.NewDummyCommon()
	branchContext := &guicontext.BranchesContext{FilteredListViewModel: guicontext.NewFilteredListViewModel(
		func() []*models.Branch { return branches }, func(branch *models.Branch) []string { return []string{branch.Name} },
	)}
	branchContext.SetSelection(0)
	subCommits := &guicontext.SubCommitsContext{SubCommitsViewModel: &guicontext.SubCommitsViewModel{}}
	subCommits.SetRef(history)
	state := &branchSyncBindingState{}
	ui := &branchSyncBindingUI{
		git:   &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths("/repo/branch-sync-binding")},
		model: types.Model{Branches: branches}, state: state,
		context: &branchSyncBindingSide{side: side},
	}
	cmnHelper := &helpers.HelperCommon{Common: cmn, IGuiCommon: ui, IGetContexts: branchSyncBindingContexts{tree: &guicontext.ContextTree{Branches: branchContext, SubCommits: subCommits}}}
	return NewSyncController(NewControllerCommon(cmnHelper, nil))
}

type branchSyncBindingContexts struct{ tree *guicontext.ContextTree }

func (contexts branchSyncBindingContexts) Contexts() *guicontext.ContextTree { return contexts.tree }

func branchSyncBindingContext(key types.ContextKey) types.Context {
	return guicontext.NewSimpleContext(guicontext.NewBaseContext(guicontext.NewBaseContextOpts{Key: key}))
}

func TestBranchSyncBindingFocusedSourceAndIdentity(t *testing.T) {
	selected := &models.Branch{Name: "selected"}
	current := &models.Branch{Name: "current", Head: true}
	controller := branchSyncBindingFixture(branchSyncBindingContext(guicontext.LOCAL_BRANCHES_CONTEXT_KEY), []*models.Branch{selected, current}, nil)
	source, problem := controller.focusedBranchSyncSource()
	if problem != branchSyncSourceOK || source.name != "selected" || !source.selected {
		t.Fatalf("focused source = %+v, %v; want selected local branch", source, problem)
	}
	controller.c.Context().(*branchSyncBindingSide).side = branchSyncBindingContext(guicontext.NORMAL_MAIN_CONTEXT_KEY)
	source, problem = controller.focusedBranchSyncSource()
	if problem != branchSyncSourceOK || source.name != "current" || source.selected {
		t.Fatalf("global source = %+v, %v; want current branch", source, problem)
	}
	if got := controller.branchSyncCommandID("push"); got != "branch.push-current" {
		t.Fatalf("global command ID = %q", got)
	}
	controller.c.Context().(*branchSyncBindingSide).side = branchSyncBindingContext(guicontext.LOCAL_BRANCHES_CONTEXT_KEY)
	controller.c.Contexts().Branches.SetSelectionRangeAndMode(1, 0, 1)
	if _, problem := controller.focusedBranchSyncSource(); problem != branchSyncSelectOneLocalBranch {
		t.Errorf("range selection problem = %v, want exactly one local branch", problem)
	}
	if got := controller.branchSyncCommandID("push"); got != "branch.push-selected" {
		t.Errorf("invalid range command ID = %q, want selected scope", got)
	}
	controller.c.Contexts().Branches.SetSelection(0)
	if got := controller.branchSyncCommandID("pull"); got != "branch.pull-selected" {
		t.Fatalf("selected command ID = %q", got)
	}
	if got := controller.branchSyncDescription(true); got != controller.c.Tr.BranchSyncPushSelected {
		t.Fatalf("selected push description = %q", got)
	}
	if got := controller.branchSyncDescription(false); got != controller.c.Tr.BranchSyncPullSelected {
		t.Fatalf("selected pull description = %q", got)
	}
	for _, verb := range []string{"push", "pull"} {
		if got, want := controller.branchSyncCommandID(verb), "branch."+verb+"-selected"; got != want {
			t.Errorf("verb %q selected command ID = %q, want %q", verb, got, want)
		}
	}
}

func TestBranchSyncBindingHistoryAndInvalidContexts(t *testing.T) {
	current := &models.Branch{Name: "current", Head: true}
	stored := &models.Branch{Name: "history-source"}
	controller := branchSyncBindingFixture(branchSyncBindingContext(guicontext.SUB_COMMITS_CONTEXT_KEY), []*models.Branch{current}, stored)
	source, problem := controller.focusedBranchSyncSource()
	if problem != branchSyncSourceOK || source.name != "history-source" || !source.selected {
		t.Fatalf("history source = %+v, %v", source, problem)
	}
	for _, key := range []types.ContextKey{guicontext.TAGS_CONTEXT_KEY, guicontext.LOCAL_COMMITS_CONTEXT_KEY} {
		controller.c.Context().(*branchSyncBindingSide).side = branchSyncBindingContext(key)
		paneSource, paneProblem := controller.focusedBranchSyncSource()
		if paneProblem != branchSyncSourceOK || paneSource.name != "current" || paneSource.selected {
			t.Errorf("other pane %q source = %+v, %v; want global current branch", key, paneSource, paneProblem)
		}
		if got := controller.branchSyncCommandID("push"); got != "branch.push-current" {
			t.Errorf("other pane %q command ID = %q, want global scope", key, got)
		}
	}
	controller.c.Context().(*branchSyncBindingSide).side = branchSyncBindingContext(guicontext.REMOTE_BRANCHES_CONTEXT_KEY)
	if _, problem := controller.focusedBranchSyncSource(); problem != branchSyncSelectOneLocalBranch {
		t.Errorf("remote branch pane problem = %v; want local selection requirement", problem)
	}
	if got := controller.branchSyncCommandID("push"); got != "branch.push-selected" {
		t.Errorf("remote branch pane command ID = %q, want selected scope", got)
	}
	for name, history := range map[string]models.Ref{
		"remote branch": &models.RemoteBranch{Name: "origin/topic"},
		"tag":           &models.Tag{Name: "release"},
		"commit":        &models.Commit{},
	} {
		controller = branchSyncBindingFixture(branchSyncBindingContext(guicontext.SUB_COMMITS_CONTEXT_KEY), []*models.Branch{current}, history)
		if _, problem := controller.focusedBranchSyncSource(); problem == branchSyncSourceOK {
			t.Errorf("sub-commits %s ref resolved from HEAD; non-local history must not fall back", name)
		}
	}
	controller.c.Context().(*branchSyncBindingSide).side = branchSyncBindingContext(guicontext.NORMAL_MAIN_CONTEXT_KEY)
	controller.c.Model().Branches = []*models.Branch{{Name: "not-head"}, {Name: "detached", Head: true, DetachedHead: true}}
	if _, problem := controller.focusedBranchSyncSource(); problem != branchSyncNoCurrentLocalBranch {
		t.Fatalf("missing attached HEAD problem = %v", problem)
	}
}

func TestBranchSyncBindingBusyOperationUsesSourceAndRepository(t *testing.T) {
	branch := &models.Branch{Name: "picked"}
	controller := branchSyncBindingFixture(branchSyncBindingContext(guicontext.LOCAL_BRANCHES_CONTEXT_KEY), []*models.Branch{branch}, nil)
	state := controller.c.State().(*branchSyncBindingState)
	state.operations = map[string]types.ItemOperation{}
	git := controller.c.Git()
	current := &models.Branch{Name: "current"}
	state.operations[current.URN()] = types.ItemOperationPushing
	if reason := controller.getDisabledReasonForBranchSync(); reason != nil {
		t.Fatalf("unrelated current branch operation disabled selected source: %v", reason)
	}
	state.operations[syncOperationItem(&commands.GitCommand{RepoPaths: git_commands.MockRepoPaths("/repo/other")}, "picked").URN()] = types.ItemOperationPushing
	if reason := controller.getDisabledReasonForBranchSync(); reason != nil {
		t.Fatalf("same-name operation in another repository disabled source: %v", reason)
	}
	state.operations[syncOperationItem(git, "picked").URN()] = types.ItemOperationPulling
	if reason := controller.getDisabledReasonForBranchSync(); reason == nil {
		t.Fatal("expected selected source's scoped operation to disable sync")
	}
	delete(state.operations, syncOperationItem(git, "picked").URN())
	state.operations[branch.URN()] = types.ItemOperationPushing
	if reason := controller.getDisabledReasonForBranchSync(); reason == nil {
		t.Fatal("expected legacy ordinary branch operation to disable sync")
	}
}
