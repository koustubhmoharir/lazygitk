package controllers

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func TestBranchSyncUpdateCollectionQueuesOpaqueForwardPlan(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "branch", "local-upstream", "topic")
	branchSyncPushGit(t, f.root, "checkout", "-q", "local-upstream")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-m", "upstream")
	targetOID := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", ".")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/local-upstream")
	if err := os.WriteFile(filepath.Join(f.root, "tracked"), []byte("dirty"), 0o600); err != nil {
		t.Fatal(err)
	}
	indexBefore, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	headBefore := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	refsBefore := branchSyncPushGit(t, f.root, "for-each-ref", "--format=%(refname) %(objectname)")
	workBefore, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	info, err := f.git.Sync.InspectBranchSync("topic")
	if err != nil {
		t.Fatal(err)
	}
	var accepted git_commands.BranchFastForwardPlan
	acceptedCount := 0
	collector := &branchSyncUpdateCollector{acceptForward: func(plan git_commands.BranchFastForwardPlan) error {
		accepted, acceptedCount = plan, acceptedCount+1
		return nil
	}}
	flow := &branchSyncUpdateFlow{target: newBranchSyncTarget(f.sync.c, branchSyncSource{name: "topic", selected: true}), info: info, collector: collector}
	plan, err := f.git.Sync.PrepareLocalBranchUpdate("topic", info.OID, "refs/heads/local-upstream")
	if err != nil {
		t.Fatal(err)
	}
	if plan.TargetOID() != targetOID {
		t.Fatalf("target %s, want %s", plan.TargetOID(), targetOID)
	}
	if err := flow.applyUpdate(plan); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.workers) != 1 {
		t.Fatalf("queued workers=%d", len(f.ui.workers))
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if acceptedCount != 0 {
		t.Fatal("plan accepted before UI callback")
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if acceptedCount != 1 || accepted != plan {
		t.Fatalf("accepted count=%d or plan differs", acceptedCount)
	}
	indexAfter, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	workAfter, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	if head := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); head != headBefore {
		t.Fatal("HEAD moved")
	}
	if refs := branchSyncPushGit(t, f.root, "for-each-ref", "--format=%(refname) %(objectname)"); refs != refsBefore {
		t.Fatal("refs changed")
	}
	if string(indexBefore) != string(indexAfter) || string(workBefore) != string(workAfter) {
		t.Fatal("index or worktree changed")
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "topic")) != 0 || f.ui.refreshes != 0 {
		t.Fatal("collector marked/refresh operation")
	}
}

func TestBranchSyncUpdateCollectionRejectsStaleAndChangedPlans(t *testing.T) {
	for _, change := range []string{"generation", "repository", "source", "owner"} {
		t.Run(change, func(t *testing.T) {
			f := newBranchSyncPushFixture(t)
			branchSyncPushGit(t, f.root, "branch", "local-upstream", "topic")
			branchSyncPushGit(t, f.root, "checkout", "-q", "local-upstream")
			branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-m", "upstream")
			branchSyncPushGit(t, f.root, "checkout", "-q", "main")
			branchSyncPushGit(t, f.root, "config", "branch.topic.remote", ".")
			branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/local-upstream")
			info, err := f.git.Sync.InspectBranchSync("topic")
			if err != nil {
				t.Fatal(err)
			}
			plan, err := f.git.Sync.PrepareLocalBranchUpdate("topic", info.OID, "refs/heads/local-upstream")
			if err != nil {
				t.Fatal(err)
			}
			accepted := 0
			flow := &branchSyncUpdateFlow{target: newBranchSyncTarget(f.sync.c, branchSyncSource{name: "topic", selected: true}), info: info, collector: &branchSyncUpdateCollector{acceptForward: func(git_commands.BranchFastForwardPlan) error { accepted++; return nil }}}
			if err := flow.applyUpdate(plan); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "generation":
				f.ui.state.generation++
			case "repository":
				f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths("/replacement")}
			case "source":
				branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
				branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-m", "move source")
				branchSyncPushGit(t, f.root, "checkout", "-q", "main")
			case "owner":
				// Replacing the UI's Git owner after capture must invalidate the target.
				f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths(f.root)}
			}
			workerErr := f.ui.runWorker(t)
			if workerErr == nil {
				if uiErr := f.ui.runUI(t); uiErr == nil {
					t.Fatal("stale collector accepted")
				}
			}
			if accepted != 0 {
				t.Fatal("stale plan accepted")
			}
		})
	}
}

func TestBranchSyncUpdateCollectionRequiresAcceptCallback(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	info, err := f.git.Sync.InspectBranchSync("topic")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := f.git.Sync.PrepareBranchFastForward("topic", info.OID, info.OID)
	if err != nil {
		t.Fatal(err)
	}
	flow := &branchSyncUpdateFlow{target: newBranchSyncTarget(f.sync.c, branchSyncSource{name: "topic", selected: true}), info: info, collector: &branchSyncUpdateCollector{}}
	if err := flow.collectFastForward(plan); err == nil {
		t.Fatal("expected missing callback error")
	}
	if errors.Is(err, git_commands.ErrBranchSyncChanged) {
		t.Fatal(err)
	}
}

func TestBranchSyncUpdateCollectionCurrentPlanIsNativeReadOnly(t *testing.T) {
	f, ui := newBranchSyncCurrentPullFixture(t)
	source := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main")
	targetOID := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	review, err := f.git.Sync.PrepareCurrentPullReview("main", source, "")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := f.git.Sync.PrepareCurrentPullIntegration(review, targetOID, git_commands.PullIntegrationMerge)
	if err != nil {
		t.Fatal(err)
	}
	accepted := 0
	var acceptedPlan git_commands.CurrentPullIntegrationPlan
	flow := &branchSyncUpdateFlow{target: newBranchSyncTarget(f.sync.c, branchSyncSource{name: "main"}), currentMode: true, collector: &branchSyncUpdateCollector{acceptCurrent: func(plan git_commands.CurrentPullIntegrationPlan) error { accepted++; acceptedPlan = plan; return nil }}}
	if err := flow.executeIntegration(plan); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if accepted != 0 || ui.subprocessCalls != 0 || ui.state.(*branchSyncCurrentPullState).repo.setCalls != 0 {
		t.Fatal("collector acted or called helper before callback")
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if accepted != 1 || !reflect.DeepEqual(acceptedPlan, plan) || ui.subprocessCalls != 0 || ui.state.(*branchSyncCurrentPullState).repo.setCalls != 0 || f.ui.refreshes != 0 {
		t.Fatal("current plan was not collected read-only")
	}
	if f.ui.state.GetItemOperation(syncOperationItem(f.git, "main")) != types.ItemOperationNone {
		t.Fatal("collector marked current branch busy")
	}
}

func TestBranchSyncUpdateCollectionIgnoredApprovalAndFingerprint(t *testing.T) {
	f, _ := newBranchSyncCurrentPullFixture(t)
	clone := setupBranchSyncCurrentPullRemote(t, f)
	branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"collision": "incoming\n"})
	if err := os.WriteFile(filepath.Join(f.root, "collision"), []byte("ignored\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, ".git", "info", "exclude"), []byte("collision\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "fetch", "-q", "origin", "main")
	source := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	targetOID := branchSyncPushGit(t, clone, "rev-parse", "HEAD")
	review, err := f.git.Sync.PrepareCurrentPullReview("main", source, "refs/remotes/origin/main")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := f.git.Sync.PrepareCurrentPullIntegration(review, targetOID, git_commands.PullIntegrationMerge)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.IgnoredFiles()) == 0 {
		t.Fatal("native plan did not capture ignored collision")
	}
	approved := plan.WithIgnoredFilesApproved()
	accepted := 0
	flow := &branchSyncUpdateFlow{target: newBranchSyncTarget(f.sync.c, branchSyncSource{name: "main"}), currentMode: true, collector: &branchSyncUpdateCollector{acceptCurrent: func(git_commands.CurrentPullIntegrationPlan) error { accepted++; return nil }}}
	if err := flow.executeIntegration(approved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, "collision"), []byte("changed after approval\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	workerErr := f.ui.runWorker(t)
	if workerErr == nil {
		if err := f.ui.runUI(t); err == nil {
			t.Fatal("accepted plan after ignored collision changed")
		}
	}
	if accepted != 0 {
		t.Fatal("changed ignored-file plan reached collector")
	}
}

func TestBranchSyncUpdateCollectionFastForwardOnlySkipsMethodMenu(t *testing.T) {
	f, _ := newBranchSyncCurrentPullFixture(t)
	branchSyncPushGit(t, f.root, "config", "branch.main.rebase", "true")
	source := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	targetOID := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	review, err := f.git.Sync.PrepareCurrentPullReview("main", source, "")
	if err != nil {
		t.Fatal(err)
	}
	flow := &branchSyncUpdateFlow{target: newBranchSyncTarget(f.sync.c, branchSyncSource{name: "main"}), currentMode: true, fastForwardOnly: true, collector: &branchSyncUpdateCollector{acceptCurrent: func(git_commands.CurrentPullIntegrationPlan) error { return nil }}}
	if err := flow.chooseIntegration(review, targetOID); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 0 {
		t.Fatal("fast-forward-only offered integration method menu")
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 || !strings.Contains(f.ui.confirms[0].Prompt, f.sync.c.Tr.BranchSyncIntegrationFFOnly) {
		t.Fatalf("missing explicit ff-only review: %+v", f.ui.confirms)
	}
}

func TestBranchSyncUpdateCollectionFastForwardOnlyDivergenceHasNoChoices(t *testing.T) {
	f, _ := newBranchSyncCurrentPullFixture(t)
	branchSyncCurrentPullDivergeLocally(t, f)
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	branchSyncPushGit(t, f.root, "commit", "--allow-empty", "-m", "incoming divergent")
	targetOID := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	branchSyncPushGit(t, f.root, "checkout", "-q", "main")
	source := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	review, err := f.git.Sync.PrepareCurrentPullReview("main", source, "")
	if err != nil {
		t.Fatal(err)
	}
	flow := &branchSyncUpdateFlow{target: newBranchSyncTarget(f.sync.c, branchSyncSource{name: "main"}), currentMode: true, fastForwardOnly: true, collector: &branchSyncUpdateCollector{acceptCurrent: func(git_commands.CurrentPullIntegrationPlan) error { return nil }}}
	if err := flow.chooseIntegration(review, targetOID); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); !errors.Is(err, git_commands.ErrBranchSyncDiverged) {
		t.Fatalf("got %v", err)
	}
	if len(f.ui.menus) != 0 || len(f.ui.confirms) != 0 {
		t.Fatal("divergence offered alternate integration or approval")
	}
}

func TestBranchSyncUpdateCollectionNoopCancelFirstContinueCollects(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	linked := filepath.Join(t.TempDir(), "occupied-topic")
	branchSyncPushGit(t, f.root, "worktree", "add", "-q", linked, "topic")
	info, err := f.git.Sync.InspectBranchSync("topic")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := f.git.Sync.PrepareBranchFastForward("topic", info.OID, info.OID)
	if err != nil {
		t.Fatal(err)
	}
	accepted := 0
	flow := &branchSyncUpdateFlow{target: newBranchSyncTarget(f.sync.c, branchSyncSource{name: "topic", selected: true}), info: info, collector: &branchSyncUpdateCollector{context: "parent batch: ", acceptForward: func(git_commands.BranchFastForwardPlan) error { accepted++; return nil }}}
	flow.update = &plan
	if err := flow.reviewUpdate(); err != nil {
		t.Fatal(err)
	}
	menu := f.ui.menus[len(f.ui.menus)-1]
	if len(menu.Items) < 2 || menu.Items[0].Verb != "cancel" {
		t.Fatalf("no-op menu isn't cancel-first: %+v", menu.Items)
	}
	if accepted != 0 {
		t.Fatal("no-op was automatically accepted")
	}
	if err := menuItem(t, menu, "branch.update-selected.noop-cancel").OnPress(); err != nil {
		t.Fatal(err)
	}
	if accepted != 0 {
		t.Fatal("cancel accepted no-op")
	}
	if err := menuItem(t, menu, "branch.update-selected.noop-continue").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatalf("read-only occupied no-op preflight: %v", err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if accepted != 1 || f.ui.refreshes != 0 {
		t.Fatalf("accepted=%d refresh=%d", accepted, f.ui.refreshes)
	}
}

func TestBranchSyncUpdateCollectionContextAndNilCallbacks(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	info, err := f.git.Sync.InspectBranchSync("topic")
	if err != nil {
		t.Fatal(err)
	}
	context := "batch 2 of 4: "
	forward, err := f.git.Sync.PrepareBranchFastForward("topic", info.OID, info.OID)
	if err != nil {
		t.Fatal(err)
	}
	collector := &branchSyncUpdateCollector{context: context}
	flow := &branchSyncUpdateFlow{target: newBranchSyncTarget(f.sync.c, branchSyncSource{name: "topic", selected: true}), info: info, collector: collector}
	if got := flow.summary(); !strings.HasPrefix(got, context) {
		t.Fatalf("summary lacks context: %q", got)
	}
	flow.update = &forward
	if err := flow.reviewUpdate(); err != nil {
		t.Fatal(err)
	}
	if got := f.ui.menus[len(f.ui.menus)-1].Prompt; !strings.HasPrefix(got, context) {
		t.Fatalf("final review lacks context: %q", got)
	}
	if err := flow.collectFastForward(forward); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("nil forward callback: %v", err)
	}
	cur, ui := newBranchSyncCurrentPullFixture(t)
	source := branchSyncPushGit(t, cur.root, "rev-parse", "HEAD")
	target := branchSyncPushGit(t, cur.root, "rev-parse", "refs/heads/topic")
	review, err := cur.git.Sync.PrepareCurrentPullReview("main", source, "")
	if err != nil {
		t.Fatal(err)
	}
	integration, err := cur.git.Sync.PrepareCurrentPullIntegration(review, target, git_commands.PullIntegrationMerge)
	if err != nil {
		t.Fatal(err)
	}
	currentFlow := &branchSyncUpdateFlow{target: newBranchSyncTarget(cur.sync.c, branchSyncSource{name: "main"}), currentMode: true, collector: &branchSyncUpdateCollector{}}
	currentFlow.collector.context = context
	if got := currentFlow.summary(); !strings.HasPrefix(got, context) {
		t.Fatalf("current summary lacks context: %q", got)
	}
	if got := currentFlow.integrationReviewPrompt(integration); !strings.HasPrefix(got, context) {
		t.Fatalf("current final review lacks context: %q", got)
	}
	if err := currentFlow.collectIntegration(integration); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("nil current callback: %v", err)
	}
	_ = ui
}

func TestBranchSyncUpdateCollectionReinspectionReturnsChanged(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	info, err := f.git.Sync.InspectBranchSync("topic")
	if err != nil {
		t.Fatal(err)
	}
	flow := &branchSyncUpdateFlow{target: newBranchSyncTarget(f.sync.c, branchSyncSource{name: "topic", selected: true}), info: info, collector: &branchSyncUpdateCollector{acceptForward: func(git_commands.BranchFastForwardPlan) error { return nil }}}
	if err := flow.reinspectOffBranch(); !errors.Is(err, git_commands.ErrBranchSyncChanged) {
		t.Fatalf("off-branch reinspection: %v", err)
	}
	current, _ := newBranchSyncCurrentPullFixture(t)
	currentInfo, err := current.git.Sync.InspectBranchSync("main")
	if err != nil {
		t.Fatal(err)
	}
	currentFlow := &branchSyncUpdateFlow{target: newBranchSyncTarget(current.sync.c, branchSyncSource{name: "main"}), info: currentInfo, currentMode: true, collector: &branchSyncUpdateCollector{acceptCurrent: func(git_commands.CurrentPullIntegrationPlan) error { return nil }}}
	if err := currentFlow.reinspectCurrent(); !errors.Is(err, git_commands.ErrBranchSyncChanged) {
		t.Fatalf("current reinspection: %v", err)
	}
	if len(f.ui.workers) != 0 || len(f.ui.menus) != 0 || len(f.ui.confirms) != 0 {
		t.Fatalf("reinspection queued/reopened work: workers=%d menus=%d confirms=%d", len(f.ui.workers), len(f.ui.menus), len(f.ui.confirms))
	}
}
