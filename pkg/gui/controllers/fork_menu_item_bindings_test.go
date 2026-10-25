package controllers

import (
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	guicontext "github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type flatCopyTestGui struct {
	types.IGuiCommon
	files []*models.File
}

func (g flatCopyTestGui) Model() *types.Model { return &types.Model{Files: g.files} }

// Views is needed only so that the real context constructors can bind their
// views; the tests never draw them.
func (g flatCopyTestGui) Views() types.Views { return types.Views{} }

type flatCopyTestHelpers struct {
	helpers *helpers.Helpers
}

func (h flatCopyTestHelpers) Helpers() *helpers.Helpers { return h.helpers }

func TestFlatMenuItemBindingsEnabledRunsItem(t *testing.T) {
	pressed := 0
	items := []*types.MenuItem{
		{CommandID: "other.command", OnPress: func() error { t.Fatal("unrelated item pressed"); return nil }},
		{CommandID: "test.copy", OnPress: func() error { pressed++; return nil }},
	}
	bindings := flatMenuItemBindings(func() []*types.MenuItem { return items },
		flatMenuEntry{"test.copy", "Copy test"})

	if len(bindings) != 1 {
		t.Fatalf("got %d bindings, want 1", len(bindings))
	}
	binding := bindings[0]
	if binding.CommandID != "test.copy" || binding.Verb != "copy" || binding.Description != "Copy test" {
		t.Errorf("binding identity = %q/%q/%q", binding.CommandID, binding.Verb, binding.Description)
	}
	if len(binding.Keys) != 0 {
		t.Errorf("binding unexpectedly has keys %v", binding.Keys)
	}
	if reason := binding.GetDisabledReason(); reason != nil {
		t.Fatalf("enabled item reported disabled: %q", reason.Text)
	}
	if err := binding.Handler(); err != nil {
		t.Fatalf("handler returned %v", err)
	}
	if pressed != 1 {
		t.Errorf("OnPress called %d times, want 1", pressed)
	}
}

func TestFlatMenuItemBindingsDisabledReturnsReason(t *testing.T) {
	const reasonText = "Nothing to copy"
	items := []*types.MenuItem{
		{
			CommandID:      "test.copy",
			DisabledReason: &types.DisabledReason{Text: reasonText},
			OnPress:        func() error { t.Fatal("disabled item pressed"); return nil },
		},
	}
	bindings := flatMenuItemBindings(func() []*types.MenuItem { return items },
		flatMenuEntry{"test.copy", "Copy test"})
	binding := bindings[0]

	reason := binding.GetDisabledReason()
	if reason == nil || reason.Text != reasonText {
		t.Fatalf("GetDisabledReason = %v, want text %q", reason, reasonText)
	}
	err := binding.Handler()
	if err == nil || err.Error() != reasonText {
		t.Fatalf("handler error = %v, want %q", err, reasonText)
	}
}

func TestFlatMenuItemBindingsMissingIDIsError(t *testing.T) {
	bindings := flatMenuItemBindings(func() []*types.MenuItem { return nil },
		flatMenuEntry{"test.missing", "Copy missing"})
	binding := bindings[0]

	reason := binding.GetDisabledReason()
	if reason == nil || !strings.Contains(reason.Text, "test.missing") {
		t.Fatalf("GetDisabledReason = %v, want a reason naming the missing ID", reason)
	}
	err := binding.Handler()
	if err == nil || !strings.Contains(err.Error(), "test.missing") {
		t.Fatalf("handler error = %v, want one naming the missing ID", err)
	}
}

// flatCopyFilesController builds a FilesController with no files, so the copy
// items are built with no selection.
func flatCopyFilesController(t *testing.T) *FilesController {
	t.Helper()
	cmn := common.NewDummyCommon()
	gui := flatCopyTestGui{}
	contextCommon := &guicontext.ContextCommon{Common: cmn, IGuiCommon: gui}
	helperCommon := &helpers.HelperCommon{
		Common:     cmn,
		IGuiCommon: gui,
		IGetContexts: historicalDiffContextsStub{tree: &guicontext.ContextTree{
			Files: guicontext.NewWorkingTreeContext(contextCommon),
		}},
	}
	helpersStub := flatCopyTestHelpers{helpers: &helpers.Helpers{
		WorkingTree: helpers.NewWorkingTreeHelper(helperCommon, nil, nil, nil, nil),
	}}
	return NewFilesController(NewControllerCommon(helperCommon, helpersStub))
}

// flatCopyCommitFilesController builds a CommitFilesController with no commit
// files, so the copy items are built with no selection.
func flatCopyCommitFilesController(t *testing.T) *CommitFilesController {
	t.Helper()
	cmn := common.NewDummyCommon()
	gui := flatCopyTestGui{}
	contextCommon := &guicontext.ContextCommon{Common: cmn, IGuiCommon: gui}
	helperCommon := &helpers.HelperCommon{
		Common:     cmn,
		IGuiCommon: gui,
		IGetContexts: historicalDiffContextsStub{tree: &guicontext.ContextTree{
			CommitFiles: guicontext.NewCommitFilesContext(contextCommon),
		}},
	}
	return NewCommitFilesController(NewControllerCommon(helperCommon, nil))
}

func TestFlatCopyBindingIDsExistInMenuItems(t *testing.T) {
	tests := []struct {
		name     string
		bindings func() []*types.Binding
		items    func() []*types.MenuItem
		wantIDs  []string
	}{
		{
			name: "files",
			bindings: func() []*types.Binding {
				return flatCopyFilesController(t).copyBindings()
			},
			items: func() []*types.MenuItem {
				return flatCopyFilesController(t).copyMenuItems()
			},
			wantIDs: []string{
				"clipboard.copy-file-name",
				"clipboard.copy-absolute-file-path",
				"clipboard.copy-working-file-diff",
				"clipboard.copy-working-tree-diff",
			},
		},
		{
			name: "commit files",
			bindings: func() []*types.Binding {
				return flatCopyCommitFilesController(t).copyBindings()
			},
			items: func() []*types.MenuItem {
				return flatCopyCommitFilesController(t).copyMenuItems()
			},
			wantIDs: []string{
				"clipboard.copy-file-name",
				"clipboard.copy-absolute-file-path",
				"clipboard.copy-history-file-diff",
				"clipboard.copy-history-file-content",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bindings := test.bindings()
			if len(bindings) != len(test.wantIDs) {
				t.Fatalf("got %d bindings, want %d", len(bindings), len(test.wantIDs))
			}
			items := test.items()
			for i, binding := range bindings {
				if binding.CommandID != test.wantIDs[i] {
					t.Errorf("binding %d command ID = %q, want %q", i, binding.CommandID, test.wantIDs[i])
				}
				if _, err := findMenuItem(items, binding.CommandID); err != nil {
					t.Errorf("binding %q: %v", binding.CommandID, err)
				}
				if binding.Description == "" {
					t.Errorf("binding %q has no description", binding.CommandID)
				}
			}
		})
	}
}
