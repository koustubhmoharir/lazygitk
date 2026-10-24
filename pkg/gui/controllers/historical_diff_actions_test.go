package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	guicontext "github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Only selection access is exercised by these guards. Embedding the interface
// makes an unexpected Git or controller dependency fail loudly in the test.
type historicalDiffCommitsStub struct {
	ContainsCommits
	items []*models.Commit
	start int
	end   int
}

type historicalDiffContextsStub struct {
	tree *guicontext.ContextTree
}

func (s historicalDiffContextsStub) Contexts() *guicontext.ContextTree { return s.tree }

func historicalTestFileController(t *testing.T, ref models.Ref, files []*models.CommitFile, showTree bool) (*CommitFilesController, *filetree.CommitFileTreeViewModel) {
	t.Helper()
	cmn := common.NewDummyCommon()
	vm := filetree.NewCommitFileTreeViewModel(func() []*models.CommitFile { return files }, cmn, showTree)
	vm.SetTree()
	vm.SetRef(ref)
	fileContext := &guicontext.CommitFilesContext{CommitFileTreeViewModel: vm}
	helperCommon := &helpers.HelperCommon{
		Common: cmn,
		IGetContexts: historicalDiffContextsStub{tree: &guicontext.ContextTree{
			CommitFiles: fileContext,
		}},
	}
	return &CommitFilesController{c: NewControllerCommon(helperCommon, nil)}, vm
}

func (s *historicalDiffCommitsStub) GetSelectedItems() ([]*models.Commit, int, int) {
	return s.items, s.start, s.end
}

func (s *historicalDiffCommitsStub) GetSelected() *models.Commit {
	if len(s.items) == 0 {
		return nil
	}
	return s.items[s.start]
}

func historicalTestCommit(hash string, parents ...string) *models.Commit {
	return models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{Hash: hash, Parents: parents})
}

func historicalTestBasicController(items []*models.Commit, start, end int) *BasicCommitsController {
	context := &historicalDiffCommitsStub{items: items, start: start, end: end}
	common := &helpers.HelperCommon{Common: common.NewDummyCommon()}
	return &BasicCommitsController{c: NewControllerCommon(common, nil), context: context}
}

func TestHistoricalCommitDiffGuard(t *testing.T) {
	valid := historicalTestCommit("0123456789abcdef0123456789abcdef01234567", "1123456789abcdef0123456789abcdef01234567")
	root := historicalTestCommit("2123456789abcdef0123456789abcdef01234567")
	for _, tc := range []struct {
		name     string
		items    []*models.Commit
		start    int
		end      int
		previous bool
		allowed  bool
	}{
		{name: "none"},
		{name: "nil item", items: []*models.Commit{nil}},
		{name: "range", items: []*models.Commit{valid, valid}, start: 0, end: 1},
		{name: "placeholder hash", items: []*models.Commit{historicalTestCommit("HEAD")}},
		{name: "root working tree", items: []*models.Commit{root}, allowed: true},
		{name: "root previous", items: []*models.Commit{root}, previous: true},
		{name: "commit working tree", items: []*models.Commit{valid}, allowed: true},
		{name: "commit previous", items: []*models.Commit{valid}, previous: true, allowed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			controller := historicalTestBasicController(tc.items, tc.start, tc.end)
			reason := controller.canHistoricalDiffCommit(tc.previous)
			if (reason == nil) != tc.allowed {
				t.Fatalf("disabled reason = %v, allowed = %v; want allowed %v", reason, reason == nil, tc.allowed)
			}
		})
	}
}

func TestHistoricalCommitDiffInvocationRechecksGuard(t *testing.T) {
	controller := historicalTestBasicController(nil, 0, 0)
	for name, handler := range map[string]func() error{
		"working tree": controller.diffCommitWithWorktree,
		"previous":     controller.diffCommitWithPrevious,
	} {
		t.Run(name, func(t *testing.T) {
			if err := handler(); err == nil {
				t.Fatal("expected invalid selection error")
			}
		})
	}
}

func TestHistoricalDiffBindingMetadata(t *testing.T) {
	controller := historicalTestBasicController(nil, 0, 0)
	controller.ListControllerTrait = &ListControllerTrait[*models.Commit]{}
	bindings := controller.historicalDiffBindings(types.KeybindingsOpts{
		Config:  config.GetDefaultConfig().Keybinding,
		GetKeys: config.GetValidatedKeyBindingKeys,
	})
	want := map[string]string{"diff.commit.worktree": "diff-worktree", "diff.commit.previous": "diff-previous", "diff.commit.selected-commit": "diff-commit"}
	seen := map[string]bool{}
	for _, binding := range bindings {
		if seen[binding.CommandID] {
			t.Fatalf("duplicate command ID %q", binding.CommandID)
		}
		seen[binding.CommandID] = true
		if want[binding.CommandID] != binding.Verb {
			t.Errorf("%s verb = %q, want %q", binding.CommandID, binding.Verb, want[binding.CommandID])
		}
		if binding.CommandID == "diff.commit.worktree" && !binding.DisplayOnScreen {
			t.Error("working-tree action should be shown in the bottom hints")
		}
		if binding.CommandID == "diff.commit.previous" && binding.DisplayOnScreen {
			t.Error("previous-commit action should not be shown in the bottom hints")
		}
		var wantedKey config.Keybinding
		switch binding.CommandID {
		case "diff.commit.worktree":
			wantedKey = config.GetDefaultConfig().Keybinding.Universal.DiffWithWorkingTree
		case "diff.commit.previous":
			wantedKey = config.GetDefaultConfig().Keybinding.Universal.DiffWithPreviousCommit
		}
		wantKeys := config.GetValidatedKeyBindingKeys(wantedKey)
		if len(binding.Keys) != len(wantKeys) {
			t.Errorf("%s keys = %v, want %v", binding.CommandID, binding.Keys, wantKeys)
		} else {
			for i := range wantKeys {
				if binding.Keys[i] != wantKeys[i] {
					t.Errorf("%s keys = %v, want %v", binding.CommandID, binding.Keys, wantKeys)
					break
				}
			}
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("got IDs %v, want %v", seen, want)
	}
}

func TestHistoricalFileDiffGuards(t *testing.T) {
	const oid = "0123456789abcdef0123456789abcdef01234567"
	commit := historicalTestCommit(oid, "1123456789abcdef0123456789abcdef01234567")
	root := historicalTestCommit("2123456789abcdef0123456789abcdef01234567")
	tests := []struct {
		name      string
		ref       models.Ref
		files     []*models.CommitFile
		showTree  bool
		previous  bool
		arbitrary bool
		selection int
		multi     bool
		refRange  bool
		wantOK    bool
	}{
		{name: "missing ref", files: []*models.CommitFile{{Path: "a.txt"}}},
		{name: "non commit ref", ref: &models.Branch{Name: "topic"}, files: []*models.CommitFile{{Path: "a.txt"}}},
		{name: "invalid full hash", ref: historicalTestCommit("HEAD"), files: []*models.CommitFile{{Path: "a.txt"}}},
		{name: "ref range", ref: commit, files: []*models.CommitFile{{Path: "a.txt"}}, refRange: true},
		{name: "missing node", ref: commit},
		{name: "multiple nodes", ref: commit, files: []*models.CommitFile{{Path: "a.txt"}, {Path: "b.txt"}}, multi: true},
		{name: "directory direct diff", ref: commit, files: []*models.CommitFile{{Path: "dir/a.txt"}}, showTree: true, wantOK: true},
		{name: "directory arbitrary file disabled", ref: commit, files: []*models.CommitFile{{Path: "dir/a.txt"}}, showTree: true, arbitrary: true},
		{name: "deleted previous allowed", ref: commit, files: []*models.CommitFile{{Path: "gone.txt", ChangeStatus: "D"}}, previous: true, wantOK: true},
		{name: "deleted worktree disabled", ref: commit, files: []*models.CommitFile{{Path: "gone.txt", ChangeStatus: "D"}}},
		{name: "deleted arbitrary disabled", ref: commit, files: []*models.CommitFile{{Path: "gone.txt", ChangeStatus: "D"}}, arbitrary: true},
		{name: "root previous disabled", ref: root, files: []*models.CommitFile{{Path: "a.txt"}}, previous: true},
		{name: "valid single file", ref: commit, files: []*models.CommitFile{{Path: "a.txt"}}, arbitrary: true, wantOK: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			controller, vm := historicalTestFileController(t, tc.ref, tc.files, tc.showTree)
			if tc.refRange {
				vm.SetRefRange(&types.RefRange{From: commit, To: commit})
			}
			if tc.multi {
				vm.ToggleStickyRange()
				vm.MoveSelectedLine(1)
			} else if tc.selection != 0 {
				vm.SetSelectedLineIdx(tc.selection)
			}
			reason := controller.canHistoricalFileDiff(tc.previous, tc.arbitrary)
			if (reason == nil) != tc.wantOK {
				t.Fatalf("disabled reason = %v, want allowed %v", reason, tc.wantOK)
			}
		})
	}
}

func TestHistoricalFileDiffCapturesRenameAndCopiesSelection(t *testing.T) {
	commit := historicalTestCommit("0123456789abcdef0123456789abcdef01234567", "1123456789abcdef0123456789abcdef01234567")
	file := &models.CommitFile{Path: "new/name.txt", PreviousPath: "old/name.txt"}
	controller, vm := historicalTestFileController(t, commit, []*models.CommitFile{file}, false)
	captured, reason := controller.captureHistoricalFileSelection()
	if reason != nil {
		t.Fatalf("capture failed: %v", reason)
	}
	if captured.oid != commit.Hash() || captured.path != "new/name.txt" || captured.previousPath != "old/name.txt" || !captured.isFile || captured.deleted || len(captured.parents) != 1 || captured.parents[0] != "1123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("captured selection = %#v", captured)
	}
	captured.parents[0] = "changed-copy"
	recaptured, reason := controller.captureHistoricalFileSelection()
	if reason != nil || recaptured.parents[0] != "1123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("captured parent list aliases commit data: selection=%#v reason=%v", recaptured, reason)
	}
	file.Path = "mutated.txt"
	file.PreviousPath = "mutated-old.txt"
	file.ChangeStatus = "D"
	vm.SetRef(historicalTestCommit("2123456789abcdef0123456789abcdef01234567"))
	if captured.oid != commit.Hash() || captured.path != "new/name.txt" || captured.previousPath != "old/name.txt" || !captured.isFile || captured.deleted || len(captured.parents) != 1 || captured.parents[0] != "changed-copy" {
		t.Fatalf("captured selection changed with model: %#v", captured)
	}
	deletedCapture, reason := controller.captureHistoricalFileSelection()
	if reason != nil || !deletedCapture.deleted {
		t.Fatalf("deleted selection capture = %#v, reason = %v", deletedCapture, reason)
	}
}

func TestHistoricalFileDiffInvocationRechecksGuard(t *testing.T) {
	controller, _ := historicalTestFileController(t, nil, nil, false)
	for name, handler := range map[string]func() error{
		"working tree": controller.diffHistoricalFileWithWorktree,
		"previous":     controller.diffHistoricalFileWithPrevious,
		"working file": controller.promptHistoricalFileWorkingDiff,
	} {
		t.Run(name, func(t *testing.T) {
			if err := handler(); err == nil {
				t.Fatal("expected invalid selection error")
			}
		})
	}
}

func TestHistoricalFileDiffBindingMetadata(t *testing.T) {
	controller, _ := historicalTestFileController(t, nil, nil, false)
	bindings := controller.historicalDiffBindings(types.KeybindingsOpts{
		Config:  config.GetDefaultConfig().Keybinding,
		GetKeys: config.GetValidatedKeyBindingKeys,
	})
	want := map[string]string{
		"diff.history-file.worktree":        "diff-worktree",
		"diff.history-file.previous":        "diff-previous",
		"diff.history-file.working-file":    "diff-working-file",
		"diff.history-file.selected-commit": "diff-commit",
	}
	seen := map[string]bool{}
	for _, binding := range bindings {
		if seen[binding.CommandID] {
			t.Fatalf("duplicate command ID %q", binding.CommandID)
		}
		seen[binding.CommandID] = true
		if binding.Verb != want[binding.CommandID] {
			t.Errorf("%s verb = %q, want %q", binding.CommandID, binding.Verb, want[binding.CommandID])
		}
		if binding.CommandID == "diff.history-file.worktree" && !binding.DisplayOnScreen {
			t.Error("working-tree action should be shown in bottom hints")
		}
		if binding.CommandID == "diff.history-file.working-file" {
			if len(binding.Keys) != 0 {
				t.Errorf("arbitrary file action keys = %v, want no default keys", binding.Keys)
			}
			if !binding.OpensMenu {
				t.Error("arbitrary file action should open a menu")
			}
		}
		var wantedKey config.Keybinding
		switch binding.CommandID {
		case "diff.history-file.worktree":
			wantedKey = config.GetDefaultConfig().Keybinding.Universal.DiffWithWorkingTree
		case "diff.history-file.previous":
			wantedKey = config.GetDefaultConfig().Keybinding.Universal.DiffWithPreviousCommit
		}
		wantKeys := config.GetValidatedKeyBindingKeys(wantedKey)
		if len(binding.Keys) != len(wantKeys) {
			t.Errorf("%s keys = %v, want %v", binding.CommandID, binding.Keys, wantKeys)
		} else {
			for i := range wantKeys {
				if binding.Keys[i] != wantKeys[i] {
					t.Errorf("%s keys = %v, want %v", binding.CommandID, binding.Keys, wantKeys)
					break
				}
			}
		}
		if binding.GetDisabledReason == nil || binding.GetDisabledReason() == nil {
			t.Errorf("%s should report disabled without a selection", binding.CommandID)
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("got IDs %v, want %v", seen, want)
	}
}
