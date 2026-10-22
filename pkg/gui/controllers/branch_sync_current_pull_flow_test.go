package controllers

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func TestBranchSyncCurrentPullMenuUsesCurrentPullIdentityAndCopy(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "config", "branch.main.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.main.merge", "refs/heads/main")
	branchSyncPushGit(t, f.root, "config", "branch.main.pushRemote", "origin")
	if err := f.sync.pullBranchSource(branchSyncSource{name: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if got := f.ui.menus[0].Title; got != f.sync.c.Tr.BranchSyncPullCurrent {
		t.Fatalf("title %q", got)
	}
	if got := f.ui.menus[0].Items[0].CommandID; got != "branch.pull-current.fetch" {
		t.Fatalf("fetch ID %q", got)
	}
	if got := f.ui.menus[0].Prompt; got == "" {
		t.Fatal("missing current pull summary")
	}
}

func TestBranchSyncCurrentPullRoutingKeepsSelectedBranchOffBranch(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	if err := f.sync.pullBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	menu := f.ui.menus[0]
	if menu.Title != f.sync.c.Tr.BranchSyncUpdateMenu {
		t.Fatalf("off-branch title %q", menu.Title)
	}
	if menu.Items[0].CommandID != "branch.update-selected.fetch" {
		t.Fatalf("off-branch ID %q", menu.Items[0].CommandID)
	}
}

type branchSyncCurrentPullRepoState struct {
	types.IRepoStateAccessor
	started  bool
	setCalls int
}

func (state *branchSyncCurrentPullRepoState) SetMergeOrRebaseStartedInLazygit(started bool) {
	state.started = started
	state.setCalls++
}

type branchSyncCurrentPullState struct {
	*branchSyncTestState
	repo *branchSyncCurrentPullRepoState
}

func (state *branchSyncCurrentPullState) GetRepoState() types.IRepoStateAccessor { return state.repo }

type branchSyncCurrentPullUI struct {
	*branchSyncIntegrationExecUI
	state types.IStateAccessor
}

func (ui *branchSyncCurrentPullUI) State() types.IStateAccessor { return ui.state }

type branchSyncCurrentPullHelpers struct{ value *helpers.Helpers }

func (h branchSyncCurrentPullHelpers) Helpers() *helpers.Helpers { return h.value }

func newBranchSyncCurrentPullFixture(t *testing.T) (*branchSyncPushFixture, *branchSyncCurrentPullUI) {
	t.Helper()
	f, _, execUI := integrationExecFixture(t)
	state := &branchSyncCurrentPullState{
		branchSyncTestState: f.ui.state,
		repo:                &branchSyncCurrentPullRepoState{},
	}
	ui := &branchSyncCurrentPullUI{branchSyncIntegrationExecUI: execUI, state: state}
	f.sync.c.HelperCommon.IGuiCommon = ui
	f.sync.c.IGetHelpers = branchSyncCurrentPullHelpers{value: &helpers.Helpers{
		MergeAndRebase: helpers.NewMergeAndRebaseHelper(f.sync.c.HelperCommon),
	}}
	return f, ui
}

func setupBranchSyncCurrentPullRemote(t *testing.T, f *branchSyncPushFixture) string {
	t.Helper()
	branchSyncPushGit(t, f.root, "push", "-q", "-u", "origin", "main")
	branchSyncPushGit(t, f.root, "config", "branch.main.remote", "origin")
	branchSyncPushGit(t, f.root, "config", "branch.main.merge", "refs/heads/main")
	clone := filepath.Join(t.TempDir(), "remote-work")
	branchSyncPushGit(t, t.TempDir(), "clone", "-q", "--branch", "main", f.remote, clone)
	branchSyncPushGit(t, clone, "config", "user.name", "Remote Test")
	branchSyncPushGit(t, clone, "config", "user.email", "remote@example.invalid")
	return clone
}

func branchSyncCurrentPullRemoteCommit(t *testing.T, clone string, files map[string]string) string {
	t.Helper()
	for path, contents := range files {
		full := filepath.Join(clone, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	branchSyncPushGit(t, clone, "add", "-A")
	branchSyncPushGit(t, clone, "commit", "-qm", "remote incoming")
	branchSyncPushGit(t, clone, "push", "-q", "origin", "main")
	return branchSyncPushGit(t, clone, "rev-parse", "HEAD")
}

func branchSyncCurrentPullDivergeLocally(t *testing.T, f *branchSyncPushFixture) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.root, "local-only"), []byte("local commit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "add", "local-only")
	branchSyncPushGit(t, f.root, "commit", "-qm", "local divergent commit")
}

func branchSyncCurrentPullStart(t *testing.T, f *branchSyncPushFixture) types.CreateMenuOptions {
	t.Helper()
	if err := f.sync.pullBranchSource(branchSyncSource{name: "main"}); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	return f.ui.menus[len(f.ui.menus)-1]
}

func branchSyncCurrentPullToChoices(t *testing.T, f *branchSyncPushFixture) types.CreateMenuOptions {
	t.Helper()
	menu := branchSyncCurrentPullStart(t, f)
	if err := menuItem(t, menu, "branch.pull-current.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	return branchSyncCurrentPullApproveFetch(t, f)
}

func branchSyncCurrentPullApproveFetch(t *testing.T, f *branchSyncPushFixture) types.CreateMenuOptions {
	t.Helper()
	if err := f.ui.confirms[len(f.ui.confirms)-1].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	return f.ui.menus[len(f.ui.menus)-1]
}

func branchSyncCurrentPullChoose(t *testing.T, f *branchSyncPushFixture, menu types.CreateMenuOptions, id string) types.ConfirmOpts {
	t.Helper()
	if err := menuItem(t, menu, id).OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	return f.ui.confirms[len(f.ui.confirms)-1]
}

func branchSyncCurrentPullRunReviewed(t *testing.T, ui *branchSyncCurrentPullUI, confirmation types.ConfirmOpts) error {
	t.Helper()
	if err := confirmation.HandleConfirm(); err != nil {
		return err
	}
	if err := ui.runBlockingWorker(t); err != nil {
		return err
	}
	if err := ui.runUI(t); err != nil {
		return err
	}
	if err := ui.runBlockingWorker(t); err != nil {
		return err
	}
	return nil
}

type branchSyncCurrentPullSnapshot struct {
	head, index, tracked []byte
}

func branchSyncCurrentPullReadSnapshot(t *testing.T, f *branchSyncPushFixture) branchSyncCurrentPullSnapshot {
	t.Helper()
	head := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	index, err := os.ReadFile(filepath.Join(f.root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	tracked, err := os.ReadFile(filepath.Join(f.root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	return branchSyncCurrentPullSnapshot{head: []byte(head), index: index, tracked: tracked}
}

func branchSyncCurrentPullAssertUnchanged(t *testing.T, f *branchSyncPushFixture, before branchSyncCurrentPullSnapshot) {
	t.Helper()
	after := branchSyncCurrentPullReadSnapshot(t, f)
	if !bytes.Equal(before.head, after.head) || !bytes.Equal(before.index, after.index) || !bytes.Equal(before.tracked, after.tracked) {
		t.Fatal("HEAD, index or tracked worktree changed before integration approval")
	}
}

func branchSyncCurrentPullRiskListsPath(items []*types.MenuItem, path string) bool {
	quoted := `"` + path + `"`
	for _, item := range items {
		if item.Label == quoted {
			return true
		}
	}
	return false
}

func TestBranchSyncCurrentPullFetchThenReviewsPinnedFastForward(t *testing.T) {
	f, ui := newBranchSyncCurrentPullFixture(t)
	clone := setupBranchSyncCurrentPullRemote(t, f)
	incoming := branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"remote-file": "incoming one\n"})
	before := branchSyncCurrentPullReadSnapshot(t, f)
	menu := branchSyncCurrentPullStart(t, f)
	if err := menuItem(t, menu, "branch.pull-current.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	fetchConfirm := f.ui.confirms[len(f.ui.confirms)-1]
	branchSyncCurrentPullAssertUnchanged(t, f, before)
	if err := fetchConfirm.HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	branchSyncCurrentPullAssertUnchanged(t, f, before)
	choices := f.ui.menus[len(f.ui.menus)-1]
	cache := "refs/remotes/origin/main"
	if got := branchSyncPushGit(t, f.root, "rev-parse", cache); got != incoming {
		t.Fatalf("fetch tip %s, want %s", got, incoming)
	}
	// Move the mutable cache after fetch. The menu action must keep the exact
	// full incoming OID captured by the fetch plan.
	next := branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"remote-file": "incoming two\n"})
	branchSyncPushGit(t, f.root, "fetch", "-q", "--no-tags", "--no-write-fetch-head", "origin", next)
	branchSyncPushGit(t, f.root, "update-ref", cache, next)
	final := branchSyncCurrentPullChoose(t, f, choices, "branch.pull-current.default")
	if !strings.Contains(final.Prompt, incoming) || strings.Contains(final.Prompt, next) {
		t.Fatalf("final review did not pin fetched OID: %s", final.Prompt)
	}
	if err := branchSyncCurrentPullRunReviewed(t, ui, final); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != incoming {
		t.Fatalf("integrated %s, want pinned %s", got, incoming)
	}
	if got, err := os.ReadFile(filepath.Join(f.root, "remote-file")); err != nil || string(got) != "incoming one\n" {
		t.Fatalf("integrated file %q, err=%v", got, err)
	}
	if ui.subprocessCalls != 1 {
		t.Fatalf("native integration calls=%d", ui.subprocessCalls)
	}
	if repo := ui.State().GetRepoState().(*branchSyncCurrentPullRepoState); repo.setCalls != 1 || repo.started {
		t.Fatalf("merge/rebase state record calls=%d started=%t", repo.setCalls, repo.started)
	}
}

func TestBranchSyncCurrentPullCancelPhases(t *testing.T) {
	for _, phase := range []string{"fetch", "integration"} {
		t.Run(phase, func(t *testing.T) {
			f, _ := newBranchSyncCurrentPullFixture(t)
			clone := setupBranchSyncCurrentPullRemote(t, f)
			branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"remote-file": "incoming\n"})
			before := branchSyncCurrentPullReadSnapshot(t, f)
			cacheBefore := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/main")
			menu := branchSyncCurrentPullStart(t, f)
			if err := menuItem(t, menu, "branch.pull-current.fetch").OnPress(); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if phase == "fetch" {
				if err := f.ui.confirms[len(f.ui.confirms)-1].HandleClose(); err != nil {
					t.Fatal(err)
				}
				branchSyncCurrentPullAssertUnchanged(t, f, before)
				if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/main"); got != cacheBefore {
					t.Fatalf("cancelled fetch changed tracking ref from %s to %s", cacheBefore, got)
				}
				return
			}
			choices := branchSyncCurrentPullApproveFetch(t, f)
			final := branchSyncCurrentPullChoose(t, f, choices, "branch.pull-current.default")
			if err := final.HandleClose(); err != nil {
				t.Fatal(err)
			}
			branchSyncCurrentPullAssertUnchanged(t, f, before)
			if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/main"); got == string(before.head) {
				t.Fatal("approved fetch did not update tracking ref")
			}
		})
	}
}

func TestBranchSyncCurrentPullConfiguredRebaseOnDivergenceAndUntracked(t *testing.T) {
	f, ui := newBranchSyncCurrentPullFixture(t)
	clone := setupBranchSyncCurrentPullRemote(t, f)
	branchSyncCurrentPullDivergeLocally(t, f)
	incoming := branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"remote-file": "remote\n"})
	branchSyncPushGit(t, f.root, "config", "branch.main.rebase", "true")
	if err := os.WriteFile(filepath.Join(f.root, "harmless-untracked"), []byte("keep\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	choices := branchSyncCurrentPullToChoices(t, f)
	defaultItem := menuItem(t, choices, "branch.pull-current.default")
	if defaultItem.Label != f.sync.c.Tr.BranchSyncIntegrationConfiguredRebase {
		t.Fatalf("default label %q", defaultItem.Label)
	}
	final := branchSyncCurrentPullChoose(t, f, choices, "branch.pull-current.default")
	if !strings.Contains(final.Prompt, f.sync.c.Tr.BranchSyncIntegrationRebaseOptions) {
		t.Fatalf("configured rebase note missing: %s", final.Prompt)
	}
	if err := branchSyncCurrentPullRunReviewed(t, ui, final); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got == incoming {
		t.Fatal("divergent rebase unexpectedly fast-forwarded")
	}
	if got := branchSyncPushGit(t, f.root, "merge-base", "HEAD", incoming); got != incoming {
		t.Fatalf("rebased HEAD does not contain incoming tip %s", incoming)
	}
	if got, err := os.ReadFile(filepath.Join(f.root, "harmless-untracked")); err != nil || string(got) != "keep\n" {
		t.Fatalf("untracked file %q, err=%v", got, err)
	}
}

func TestBranchSyncCurrentPullUnconfiguredDivergenceKeepsAlternatives(t *testing.T) {
	f, _ := newBranchSyncCurrentPullFixture(t)
	clone := setupBranchSyncCurrentPullRemote(t, f)
	branchSyncCurrentPullDivergeLocally(t, f)
	branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"remote-file": "remote\n"})
	choices := branchSyncCurrentPullToChoices(t, f)
	defaultItem := menuItem(t, choices, "branch.pull-current.default")
	if defaultItem.DisabledReason == nil || !strings.Contains(defaultItem.DisabledReason.Text, git_commands.ErrPullIntegrationChoiceRequired.Error()) {
		t.Fatalf("default disabled reason = %#v", defaultItem.DisabledReason)
	}
	for _, id := range []string{"ff-only", "merge", "merge-no-ff", "rebase", "rebase-merges", "rebase-interactive"} {
		if menuItem(t, choices, "branch.pull-current."+id).DisabledReason != nil {
			t.Errorf("explicit choice %s disabled", id)
		}
	}
}

func TestBranchSyncCurrentPullPolicyChangeBeforeChoicesReinspectsWithoutRefetch(t *testing.T) {
	f, ui := newBranchSyncCurrentPullFixture(t)
	clone := setupBranchSyncCurrentPullRemote(t, f)
	branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"remote-file": "incoming\n"})
	before := branchSyncCurrentPullReadSnapshot(t, f)
	menu := branchSyncCurrentPullStart(t, f)
	if err := menuItem(t, menu, "branch.pull-current.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.confirms[len(f.ui.confirms)-1].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	// The fetch is already approved and complete; invalidate its immutable
	// review before the queued worker prepares integration choices.
	branchSyncPushGit(t, f.root, "config", "branch.main.rebase", "true")
	branchSyncPushGit(t, f.root, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "unavailable-remote"))
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 2 || f.ui.menus[len(f.ui.menus)-1].Title != f.sync.c.Tr.BranchSyncPullCurrent {
		t.Fatalf("stale integration choices were shown instead of fresh top-level review: %#v", f.ui.menus)
	}
	branchSyncCurrentPullAssertUnchanged(t, f, before)
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/main"); got == string(before.head) {
		t.Fatal("approved fetch did not update tracking ref")
	}
	if ui.subprocessCalls != 0 {
		t.Fatalf("unexpected native integration/refetch subprocess calls=%d", ui.subprocessCalls)
	}
}

func TestBranchSyncCurrentPullExplicitMergeOverridesConfiguredRebase(t *testing.T) {
	f, ui := newBranchSyncCurrentPullFixture(t)
	clone := setupBranchSyncCurrentPullRemote(t, f)
	branchSyncCurrentPullDivergeLocally(t, f)
	branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"remote-file": "remote\n"})
	branchSyncPushGit(t, f.root, "config", "branch.main.rebase", "true")
	choices := branchSyncCurrentPullToChoices(t, f)
	final := branchSyncCurrentPullChoose(t, f, choices, "branch.pull-current.merge")
	if strings.Contains(final.Prompt, f.sync.c.Tr.BranchSyncIntegrationRebaseOptions) || !strings.Contains(final.Prompt, f.sync.c.Tr.BranchSyncIntegrationMerge) {
		t.Fatalf("explicit merge review is inaccurate: %s", final.Prompt)
	}
	if err := branchSyncCurrentPullRunReviewed(t, ui, final); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-list", "--parents", "-n", "1", "HEAD"); len(strings.Fields(got)) != 3 {
		t.Fatalf("explicit merge did not create merge commit: %s", got)
	}
}

func TestBranchSyncCurrentPullConfiguredMergeOptionsAreQuoted(t *testing.T) {
	f, _ := newBranchSyncCurrentPullFixture(t)
	clone := setupBranchSyncCurrentPullRemote(t, f)
	branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"remote-file": "remote\n"})
	branchSyncPushGit(t, f.root, "config", "branch.main.mergeOptions", "--no-ff --log=7")
	choices := branchSyncCurrentPullToChoices(t, f)
	defaultItem := menuItem(t, choices, "branch.pull-current.default")
	if defaultItem.Label != f.sync.c.Tr.BranchSyncIntegrationConfiguredMerge {
		t.Fatalf("configured merge label %q", defaultItem.Label)
	}
	final := branchSyncCurrentPullChoose(t, f, choices, "branch.pull-current.default")
	for _, expected := range []string{`"--no-ff --log=7"`, `pull.ff override: ""`} {
		if !strings.Contains(final.Prompt, expected) {
			t.Errorf("configured merge details missing %q: %s", expected, final.Prompt)
		}
	}
}

func TestBranchSyncCurrentPullIgnoredRiskCancelAndReapprovalAfterChange(t *testing.T) {
	f, ui := newBranchSyncCurrentPullFixture(t)
	clone := setupBranchSyncCurrentPullRemote(t, f)
	branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"collision": "from incoming\n"})
	if err := os.WriteFile(filepath.Join(f.root, "collision"), []byte("ignored original\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, ".git", "info", "exclude"), []byte("collision\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "check-ignore", "-q", "collision")
	choices := branchSyncCurrentPullToChoices(t, f)
	final := branchSyncCurrentPullChoose(t, f, choices, "branch.pull-current.merge")
	if err := final.HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	risk := f.ui.menus[len(f.ui.menus)-1]
	if risk.Items[0].Verb != "cancel" {
		t.Fatalf("first risk action is %q", risk.Items[0].Verb)
	}
	if !strings.Contains(risk.Prompt, "Integrate reviewed commit") || !branchSyncCurrentPullRiskListsPath(risk.Items, "collision") {
		t.Fatalf("risk review omitted full context or quoted collision path: prompt=%s items=%v", risk.Prompt, risk.Items)
	}
	if err := menuItem(t, risk, "branch.pull-current.cancel-ignored-risk").OnPress(); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("cancel executed integration")
	}
	if err := os.WriteFile(filepath.Join(f.root, "collision"), []byte("ignored changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := menuItem(t, risk, "branch.pull-current.approve-ignored-risk").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("changed ignored contents auto-executed integration")
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) < 3 {
		t.Fatalf("fresh review not shown after changed file; confirmations=%d", len(f.ui.confirms))
	}
	fresh := f.ui.confirms[len(f.ui.confirms)-1]
	if err := fresh.HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	freshRisk := f.ui.menus[len(f.ui.menus)-1]
	if !branchSyncCurrentPullRiskListsPath(freshRisk.Items, "collision") {
		t.Fatalf("fresh risk review omitted quoted changed path: %v", freshRisk.Items)
	}
	if err := menuItem(t, freshRisk, "branch.pull-current.approve-ignored-risk").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 1 {
		t.Fatalf("approved native integration calls=%d", ui.subprocessCalls)
	}
	if got, err := os.ReadFile(filepath.Join(f.root, "collision")); err != nil || string(got) != "from incoming\n" {
		t.Fatalf("approved ignored collision result %q, err=%v", got, err)
	}
}

func TestBranchSyncCurrentPullLocalUpstreamPinsCommitWithoutFetch(t *testing.T) {
	f, _ := newBranchSyncCurrentPullFixture(t)
	branchSyncPushGit(t, f.root, "config", "branch.main.remote", ".")
	branchSyncPushGit(t, f.root, "config", "branch.main.merge", "refs/heads/topic")
	upstream := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic")
	menu := branchSyncCurrentPullStart(t, f)
	if got := menu.Items[0].Label; got != f.sync.c.Tr.BranchSyncReviewLocalUpdate {
		t.Fatalf("local upstream label %q", got)
	}
	if err := menu.Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	choices := f.ui.menus[len(f.ui.menus)-1]
	if len(f.ui.confirms) != 0 {
		t.Fatal("local upstream showed a network fetch confirmation")
	}
	if err := menuItem(t, choices, "branch.pull-current.default").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.ui.confirms[len(f.ui.confirms)-1].Prompt, upstream) {
		t.Fatal("local upstream integration did not pin full commit ID")
	}
	if _, err := os.Stat(filepath.Join(f.root, ".git", "refs", "remotes", "origin")); !os.IsNotExist(err) {
		t.Fatalf("local upstream unexpectedly created remote cache: %v", err)
	}
}

func TestBranchSyncCurrentPullRefusesRepositoryGenerationChangeAtFetchConfirmation(t *testing.T) {
	f, _ := newBranchSyncCurrentPullFixture(t)
	clone := setupBranchSyncCurrentPullRemote(t, f)
	branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"remote-file": "incoming\n"})
	menu := branchSyncCurrentPullStart(t, f)
	cacheBefore := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/main")
	if err := menuItem(t, menu, "branch.pull-current.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	f.ui.state.generation++
	if err := f.ui.confirms[len(f.ui.confirms)-1].HandleConfirm(); err == nil {
		t.Fatal("stale repository generation accepted")
	}
	if uiCalls := f.ui.refreshes; uiCalls != 0 {
		t.Fatalf("unexpected refreshes=%d", uiCalls)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/main"); got != cacheBefore {
		t.Fatalf("stale confirmation changed tracking ref from %s to %s", cacheBefore, got)
	}
}

func TestBranchSyncCurrentPullHeadChangeAtFetchConfirmationDoesNotFetch(t *testing.T) {
	f, _ := newBranchSyncCurrentPullFixture(t)
	clone := setupBranchSyncCurrentPullRemote(t, f)
	branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"remote-file": "incoming\n"})
	menu := branchSyncCurrentPullStart(t, f)
	cacheBefore := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/main")
	if err := menuItem(t, menu, "branch.pull-current.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	if err := f.ui.confirms[len(f.ui.confirms)-1].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err == nil {
		t.Fatal("reinspection accepted a target that is no longer current")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/main"); got != cacheBefore {
		t.Fatalf("HEAD change moved tracking ref from %s to %s", cacheBefore, got)
	}
}

func TestBranchSyncCurrentPullPolicyChangeBeforeExecutionRequiresFreshReview(t *testing.T) {
	f, ui := newBranchSyncCurrentPullFixture(t)
	clone := setupBranchSyncCurrentPullRemote(t, f)
	branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"remote-file": "incoming\n"})
	choices := branchSyncCurrentPullToChoices(t, f)
	cacheBefore := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/main")
	headBefore := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	final := branchSyncCurrentPullChoose(t, f, choices, "branch.pull-current.default")
	branchSyncPushGit(t, f.root, "config", "pull.rebase", "true")
	if err := final.HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := ui.runBlockingWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if ui.subprocessCalls != 0 {
		t.Fatal("changed integration policy launched native command")
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/main"); got != cacheBefore {
		t.Fatalf("policy change caused refetch: tracking ref %s changed to %s", cacheBefore, got)
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != headBefore {
		t.Fatalf("policy change moved current branch from %s to %s", headBefore, got)
	}
	if !strings.Contains(f.ui.menus[len(f.ui.menus)-1].Items[0].CommandID, "branch.pull-current") {
		t.Fatal("fresh source review did not return to pull menu")
	}
}

func TestBranchSyncCurrentPullPolicyChangeBetweenFetchAndChoices(t *testing.T) {
	f, ui := newBranchSyncCurrentPullFixture(t)
	clone := setupBranchSyncCurrentPullRemote(t, f)
	incoming := branchSyncCurrentPullRemoteCommit(t, clone, map[string]string{"remote-file": "incoming\n"})
	head := branchSyncPushGit(t, f.root, "rev-parse", "HEAD")
	menu := branchSyncCurrentPullStart(t, f)
	if err := menuItem(t, menu, "branch.pull-current.fetch").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	// The approved fetch has finished; the integration-choice worker is queued.
	if got := branchSyncPushGit(t, f.root, "rev-parse", "refs/remotes/origin/main"); got != incoming {
		t.Fatalf("fetch did not reach incoming commit: %s", got)
	}
	branchSyncPushGit(t, f.root, "config", "pull.rebase", "true")
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 2 || f.ui.menus[1].Title != f.sync.c.Tr.BranchSyncPullCurrent {
		t.Fatal("changed policy presented stale integration choices instead of top-level review")
	}
	if len(f.ui.workers) != 0 || len(f.ui.confirms) != 1 || ui.subprocessCalls != 0 {
		t.Fatal("reinspection queued an unapproved fetch or integration")
	}
	if got := branchSyncPushGit(t, f.root, "rev-parse", "HEAD"); got != head {
		t.Fatalf("reinspection changed HEAD to %s", got)
	}
}
