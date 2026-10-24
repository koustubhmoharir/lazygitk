package controllers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
)

func (s *selectedRebaseFlowState) GetRepoGeneration() int { return 0 }

func TestForkCombineSelection(t *testing.T) {
	valid := historicalTestCommit("0123456789abcdef0123456789abcdef01234567")
	for _, tc := range []struct {
		name  string
		items []*models.Commit
		start int
		end   int
		ok    bool
	}{
		{name: "multiple full IDs", items: []*models.Commit{valid, historicalTestCommit("1123456789abcdef0123456789abcdef01234567"), historicalTestCommit("2123456789abcdef0123456789abcdef01234567")}, start: 0, end: 2, ok: true},
		{name: "one commit", items: []*models.Commit{valid}, ok: false},
		{name: "abbreviated", items: []*models.Commit{historicalTestCommit("0123456789ab"), valid}, ok: false},
		{name: "placeholder", items: []*models.Commit{historicalTestCommit("HEAD"), valid}, ok: false},
		{name: "nil item", items: []*models.Commit{valid, nil}, ok: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			controller := historicalTestBasicController(tc.items, tc.start, tc.end)
			binding := controller.combineSelectedBinding()
			if binding.CommandID != "commit.combine-selected" || binding.Verb != "combine" || len(binding.Keys) != 0 || !binding.OpensMenu {
				t.Fatalf("binding identity/keys = %q/%q/%v", binding.CommandID, binding.Verb, binding.Keys)
			}
			got, ok := forkCombineSelection(controller.context)
			if ok != tc.ok {
				t.Fatalf("selection valid=%v, want %v", ok, tc.ok)
			}
			if ok && len(got) != len(tc.items) {
				t.Fatalf("captured IDs=%v", got)
			}
		})
	}
}

func TestForkCombineStaleReviewChoicesAndCancel(t *testing.T) {
	controller, ui, _, oids := selectedRebaseFlowFixture(t)
	flow := &forkCombineFlow{c: controller.c, git: ui.git, generation: 0, oids: append([]string(nil), oids[:2]...), draft: "edited subject\n\nretained body"}
	if err := flow.stale(git_commands.CapturedCombinePlan{}); err != nil {
		t.Fatal(err)
	}
	if len(ui.menus) != 1 || len(ui.menus[0].Items) != 3 {
		t.Fatalf("stale review choices = %v", ui.menus)
	}
	if err := ui.menus[0].Items[2].OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(ui.workers) != 0 {
		t.Fatal("cancel queued a new review")
	}
	if err := ui.menus[0].Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(flow.oids) != 2 || flow.draft != "edited subject\n\nretained body" || len(ui.workers) != 1 {
		t.Fatalf("fresh review did not retain captured input: oids=%v draft=%q workers=%d", flow.oids, flow.draft, len(ui.workers))
	}
}

func TestForkCombineDraftRetainedOnDirtyPreStartFailure(t *testing.T) {
	controller, ui, root, oids := selectedRebaseFlowFixture(t)
	plan, err := ui.git.Rebase.PrepareCombinePlan(oids[:2])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source"), []byte("modified after review\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	flow := &forkCombineFlow{c: controller.c, git: ui.git, generation: 0, oids: append([]string(nil), oids[:2]...)}
	draft := "user edited subject\n\nuser edited body"
	if err := flow.apply(plan, draft); err != nil {
		t.Fatal(err)
	}
	if err := ui.runWorker(t); err != nil {
		t.Fatalf("preparation error should reopen the draft without starting Git: %v", err)
	}
	if flow.draft != draft || !flow.started {
		t.Fatalf("draft=%q started=%v after pre-start failure", flow.draft, flow.started)
	}
	if len(ui.ui) != 1 || len(ui.refreshes) == 0 || ui.runCalls != 0 {
		t.Fatalf("failure handling: UI callbacks=%d refreshes=%d subprocesses=%d", len(ui.ui), len(ui.refreshes), ui.runCalls)
	}
}

func TestForkCombineRejectsRepeatedMutationApproval(t *testing.T) {
	controller, ui, root, oids := selectedRebaseFlowFixture(t)
	plan, err := ui.git.Rebase.PrepareCombinePlan(oids[:2])
	if err != nil {
		t.Fatal(err)
	}
	flow := &forkCombineFlow{c: controller.c, git: ui.git}
	if err := flow.apply(plan, "approved"); err != nil {
		t.Fatal(err)
	}
	if err := flow.apply(plan, "approved twice"); err == nil {
		t.Fatal("replayed approval queued another execution")
	}
	if len(ui.workers) != 1 || selectedRebaseFlowGit(t, root, "rev-parse", "HEAD") != oids[1] {
		t.Fatal("approval replay changed the queued operation or HEAD")
	}
}

func TestForkCombineTerminalFailureCleansPayloadAndPreservesError(t *testing.T) {
	for _, changedRepo := range []bool{false, true} {
		t.Run(map[bool]string{false: "terminal failure", true: "repository changed"}[changedRepo], func(t *testing.T) {
			controller, ui, _, _ := selectedRebaseFlowFixture(t)
			flow := &forkCombineFlow{c: controller.c, git: ui.git}
			cmd := oscommands.NewDummyOSCommand().Cmd.New([]string{"git", "status"})
			cleaned := false
			if changedRepo {
				ui.git = &commands.GitCommand{}
			}
			err := flow.runTerminal(cmd, func() { cleaned = true })
			if !cleaned {
				t.Fatal("terminal path leaked the initial message payload")
			}
			if changedRepo {
				if err == nil || ui.runCalls != 0 || len(ui.workers) != 0 {
					t.Fatal("terminal action followed a repository change")
				}
				return
			}
			if err != nil || ui.runCalls != 1 || len(ui.workers) != 1 {
				t.Fatalf("terminal completion did not queue native result handling: %v", err)
			}
			if err := ui.runWorker(t); err == nil {
				t.Fatal("terminal execution error was swallowed")
			}
			if len(ui.refreshes) == 0 {
				t.Fatal("terminal failure did not refresh native state")
			}
		})
	}
}
