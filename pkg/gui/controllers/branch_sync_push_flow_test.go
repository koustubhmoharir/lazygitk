package controllers

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_config"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func TestBranchSyncPushSafeURLDisplay(t *testing.T) {
	for _, raw := range []string{
		"https://user:secret@example.com/repo?token=secret#private",
		"ssh://user:secret@example.com/repo",
		"git@example.com:secret/repo",
		"https://user:pass@[broken",
		"https://example.com/repo\n\x1b[31msecret",
	} {
		got := safeBranchSyncURL(raw)
		if strings.Contains(got, "secret") || strings.Contains(got, "token=") || strings.Contains(got, "private") {
			t.Fatalf("credential-bearing URL data was exposed: %s", got)
		}
	}
	if got := safeBranchSyncURL("../local/repo"); !strings.Contains(got, "local/repo") {
		t.Fatalf("local path was not retained: %s", got)
	}
	if got := safeBranchSyncURL(`C:\repos\project`); !strings.Contains(got, `C:`) || !strings.Contains(got, `project`) {
		t.Fatalf("Windows path was not retained: %s", got)
	}
}

type branchSyncPushTestUI struct {
	types.IGuiCommon
	git       *commands.GitCommand
	model     types.Model
	state     *branchSyncTestState
	menus     []types.CreateMenuOptions
	prompts   []types.PromptOpts
	confirms  []types.ConfirmOpts
	workers   []func(gocui.Task) error
	ui        []func() error
	refreshes int
}

func (ui *branchSyncPushTestUI) Git() *commands.GitCommand   { return ui.git }
func (ui *branchSyncPushTestUI) State() types.IStateAccessor { return ui.state }
func (ui *branchSyncPushTestUI) Model() *types.Model         { return &ui.model }
func (ui *branchSyncPushTestUI) Menu(opts types.CreateMenuOptions) error {
	ui.menus = append(ui.menus, opts)
	return nil
}

func (ui *branchSyncPushTestUI) Prompt(opts types.PromptOpts) { ui.prompts = append(ui.prompts, opts) }

func (ui *branchSyncPushTestUI) Confirm(opts types.ConfirmOpts) {
	ui.confirms = append(ui.confirms, opts)
}
func (ui *branchSyncPushTestUI) OnUIThread(f func() error) { ui.ui = append(ui.ui, f) }
func (ui *branchSyncPushTestUI) WithWaitingStatus(_ string, f func(gocui.Task) error) error {
	ui.workers = append(ui.workers, f)
	return nil
}
func (ui *branchSyncPushTestUI) RefreshFromWorker(types.RefreshOptions) { ui.refreshes++ }

func (ui *branchSyncPushTestUI) runWorker(t *testing.T) error {
	t.Helper()
	if len(ui.workers) == 0 {
		t.Fatal("expected queued worker")
	}
	f := ui.workers[0]
	ui.workers = ui.workers[1:]
	return f(nil)
}

func (ui *branchSyncPushTestUI) runUI(t *testing.T) error {
	t.Helper()
	if len(ui.ui) == 0 {
		t.Fatal("expected queued UI callback")
	}
	f := ui.ui[0]
	ui.ui = ui.ui[1:]
	return f()
}

type branchSyncPushFixture struct {
	root      string
	remote    string
	git       *commands.GitCommand
	ui        *branchSyncPushTestUI
	sync      *SyncController
	current   string
	sourceOID string
}

func branchSyncPushGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func newBranchSyncPushFixture(t *testing.T) *branchSyncPushFixture {
	t.Helper()
	root := t.TempDir()
	branchSyncPushGit(t, root, "init", "-q", "-b", "main")
	branchSyncPushGit(t, root, "config", "user.name", "Test")
	branchSyncPushGit(t, root, "config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(root, "tracked"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, root, "add", "tracked")
	branchSyncPushGit(t, root, "commit", "-qm", "base")
	branchSyncPushGit(t, root, "branch", "topic")
	branchSyncPushGit(t, root, "checkout", "-q", "topic")
	if err := os.WriteFile(filepath.Join(root, "tracked"), []byte("topic"), 0o600); err != nil {
		t.Fatal(err)
	}
	branchSyncPushGit(t, root, "commit", "-qam", "topic")
	sourceOID := branchSyncPushGit(t, root, "rev-parse", "HEAD")
	branchSyncPushGit(t, root, "checkout", "-q", "main")
	remote := filepath.Join(t.TempDir(), "remote.git")
	branchSyncPushGit(t, t.TempDir(), "init", "--bare", "-q", remote)
	branchSyncPushGit(t, root, "remote", "add", "origin", remote)
	branchSyncPushGit(t, root, "config", "branch.topic.pushRemote", "origin")
	branchSyncPushGit(t, root, "config", "branch.main.pushRemote", "origin")
	paths := git_commands.MockRepoPaths(root)
	osCmd := oscommands.NewDummyOSCommand()
	cmn := common.NewDummyCommon()
	git := commands.NewGitCommandAux(cmn, nil, osCmd, git_config.NewFakeGitConfig(nil), paths, nil)
	state := &branchSyncTestState{generation: 1, operations: map[string]types.ItemOperation{}}
	ui := &branchSyncPushTestUI{git: git, state: state}
	hc := &helpers.HelperCommon{Common: cmn, IGuiCommon: ui}
	c := NewControllerCommon(hc, nil)
	return &branchSyncPushFixture{root: root, remote: remote, git: git, ui: ui, sync: NewSyncController(c), current: "main", sourceOID: sourceOID}
}

func (f *branchSyncPushFixture) open(t *testing.T, source branchSyncSource) {
	t.Helper()
	if err := f.sync.pushBranchSource(source); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
}

func menuItem(t *testing.T, menu types.CreateMenuOptions, id string) *types.MenuItem {
	t.Helper()
	for _, item := range menu.Items {
		if item.CommandID == id {
			return item
		}
	}
	t.Fatalf("menu item %q not found", id)
	return nil
}

func TestBranchSyncPushSelectedBranchExecutesWithoutConfirmation(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	source := branchSyncSource{name: "topic", selected: true}
	if err := f.sync.pushBranchSource(source); err != nil {
		t.Fatal(err)
	}
	source.name = "main"
	f.ui.model.Branches = []*models.Branch{{Name: "main"}}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.menus) != 1 {
		t.Fatalf("expected initial push menu, got %d", len(f.ui.menus))
	}
	menu := f.ui.menus[0]
	if !strings.Contains(menu.Prompt, `"topic"`) || !strings.Contains(menu.Prompt, "origin") || !strings.Contains(menu.Prompt, "topic") {
		t.Fatalf("menu omitted captured target/defaults: %q", menu.Prompt)
	}
	if !strings.Contains(menu.Prompt, "Set upstream for selected branch: false") || !strings.Contains(menu.Prompt, "Check submodules: false") {
		t.Fatalf("push defaults were not explicit: %q", menu.Prompt)
	}
	if err := menuItem(t, menu, "branch.push-selected.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	// The menu summary is the review for an ordinary push, so execution is
	// queued directly after preparation without a confirmation.
	if len(f.ui.confirms) != 0 {
		t.Fatalf("ordinary push asked for a second confirmation: %+v", f.ui.confirms)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.remote, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
		t.Fatalf("remote got %s, want selected branch tip %s", got, f.sourceOID)
	}
	if got := branchSyncPushGit(t, f.root, "symbolic-ref", "--short", "HEAD"); got != f.current {
		t.Fatalf("push changed current branch to %s", got)
	}
}

func TestBranchSyncPushMovedBranchReopensMenu(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	f.open(t, branchSyncSource{name: "topic", selected: true})
	oldTip := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic^")
	// The branch moves after the menu captured sourceOID. Choosing the push
	// must not send the captured OID; it reopens the menu with fresh details.
	branchSyncPushGit(t, f.root, "update-ref", "refs/heads/topic", oldTip, f.sourceOID)
	if err := menuItem(t, f.ui.menus[0], "branch.push-selected.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := f.ui.runWorker(t); err != nil {
			t.Fatal(err)
		}
		if err := f.ui.runUI(t); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.ui.confirms) != 0 {
		t.Fatal("moved branch produced a confirmation instead of reopening the menu")
	}
	if len(f.ui.menus) != 2 {
		t.Fatalf("expected the push menu to reopen, got %d menus", len(f.ui.menus))
	}
	if !strings.Contains(f.ui.menus[1].Prompt, oldTip[:7]) {
		t.Fatalf("fresh menu omitted changed tip: %q", f.ui.menus[1].Prompt)
	}
	if _, err := exec.Command("git", "-C", f.remote, "show-ref", "--verify", "--quiet", "refs/heads/topic").Output(); err == nil {
		t.Fatal("moved branch contacted remote")
	}
}

func TestBranchSyncPushReplacesTracking(t *testing.T) {
	for _, tc := range []struct {
		name        string
		setUp       bool
		config      [][]string
		wantReplace bool
	}{
		{name: "no set-upstream", setUp: false, config: [][]string{{"config", "branch.topic.remote", "oldremote"}, {"config", "branch.topic.merge", "refs/heads/old-one"}}},
		{name: "no previous tracking", setUp: true},
		{name: "same remote and single same merge ref", setUp: true, config: [][]string{{"config", "branch.topic.remote", "origin"}, {"config", "branch.topic.merge", "refs/heads/topic"}}},
		{name: "different remote", setUp: true, wantReplace: true, config: [][]string{{"config", "branch.topic.remote", "oldremote"}, {"config", "branch.topic.merge", "refs/heads/topic"}}},
		{name: "different merge ref", setUp: true, wantReplace: true, config: [][]string{{"config", "branch.topic.remote", "origin"}, {"config", "branch.topic.merge", "refs/heads/old-one"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newBranchSyncPushFixture(t)
			for _, args := range tc.config {
				branchSyncPushGit(t, f.root, args...)
			}
			opts := git_commands.BranchPushOptions{Remote: "origin", DestinationRef: "refs/heads/topic", SetUpstream: tc.setUp, Submodules: git_commands.BranchPushNoSubmoduleRecursion}
			plan, err := f.git.Sync.PrepareBranchPush("topic", f.sourceOID, opts)
			if err != nil {
				t.Fatal(err)
			}
			if got := branchPushReplacesTracking(plan); got != tc.wantReplace {
				t.Fatalf("branchPushReplacesTracking() = %t, want %t", got, tc.wantReplace)
			}
		})
	}
}

func TestBranchSyncPushQueuedUICallbackRejectsRepositoryChange(t *testing.T) {
	for _, generationChange := range []bool{false, true} {
		t.Run(map[bool]string{false: "git-pointer", true: "generation"}[generationChange], func(t *testing.T) {
			f := newBranchSyncPushFixture(t)
			if err := f.sync.pushBranchSource(branchSyncSource{name: "topic", selected: true}); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if generationChange {
				f.ui.state.generation++
			} else {
				f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths("/other/repo")}
			}
			if err := f.ui.runUI(t); err == nil {
				t.Fatal("queued modal callback accepted a changed repository")
			}
			if len(f.ui.menus) != 0 {
				t.Fatal("stale callback opened a modal for another repository")
			}
		})
	}
}

func TestBranchSyncPushMissingRemoteCancelAndOrdinaryEditAfterForceCancel(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "remote", "remove", "origin")
	f.open(t, branchSyncSource{name: "topic", selected: true})
	menu := f.ui.menus[0]
	if err := menuItem(t, menu, "branch.push-selected.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.prompts) != 1 {
		t.Fatal("missing remote did not prompt")
	}
	if err := f.ui.prompts[0].HandleClose(); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Command("git", "-C", f.remote, "show-ref", "--verify", "--quiet", "refs/heads/topic").Output(); err == nil {
		t.Fatal("cancelled remote prompt changed remote")
	}
	// Refresh the captured menu context after remote configuration changed.
	f = newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "update-ref", "refs/remotes/origin/topic", f.sourceOID)
	f.open(t, branchSyncSource{name: "topic", selected: true})
	initial := f.ui.menus[0]
	if err := menuItem(t, initial, "branch.push-selected.force-with-lease").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	forceReview := f.ui.menus[len(f.ui.menus)-1]
	if err := menuItem(t, forceReview, "branch.push-selected.force-cancel").OnPress(); err != nil {
		t.Fatal(err)
	}
	menusBeforeEdit := len(f.ui.menus)
	if err := menuItem(t, initial, "branch.push-selected.remote").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.prompts[len(f.ui.prompts)-1].HandleConfirm("origin"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	// An ordinary push goes straight to execution, so it must not show a
	// confirmation or a force menu.
	if len(f.ui.confirms) != 0 {
		t.Fatalf("ordinary remote edit asked for a confirmation: %+v", f.ui.confirms)
	}
	if len(f.ui.menus) != menusBeforeEdit {
		t.Fatal("ordinary remote edit reopened a menu, such as a force menu")
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
}

func TestBranchSyncPushRemoteURLNeverAppearsAsRemoteName(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	secretURL := "https://user:secret@example.invalid/repository"
	branchSyncPushGit(t, f.root, "config", "branch.topic.pushRemote", secretURL)
	f.open(t, branchSyncSource{name: "topic", selected: true})
	if strings.Contains(f.ui.menus[0].Prompt, "secret") || strings.Contains(f.ui.menus[0].Prompt, secretURL) {
		t.Fatalf("URL-shaped remote leaked into initial menu: %q", f.ui.menus[0].Prompt)
	}
	if err := menuItem(t, f.ui.menus[0], "branch.push-selected.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.prompts) != 1 {
		t.Fatal("URL-shaped pushRemote did not require a named remote")
	}
	if f.ui.prompts[0].InitialContent != "" || !strings.Contains(f.ui.prompts[0].Title, "configured remote name") {
		t.Fatalf("unsafe URL was suggested as a remote name: %+v", f.ui.prompts[0])
	}
	if err := f.ui.prompts[0].HandleConfirm("origin"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 0 {
		t.Fatalf("ordinary push to the named remote asked for a confirmation: %+v", f.ui.confirms)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.remote, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
		t.Fatalf("named remote got %s, want selected branch tip %s", got, f.sourceOID)
	}
}

func TestBranchSyncPushGlobalCurrentBranchRechecksAtExecution(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	f.open(t, branchSyncSource{name: "main", selected: false})
	if err := menuItem(t, f.ui.menus[0], "branch.push-selected.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 0 {
		t.Fatal("ordinary push asked for a confirmation")
	}
	// Execution is queued without a review, so HEAD changes after queuing.
	branchSyncPushGit(t, f.root, "checkout", "-q", "topic")
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err == nil {
		t.Fatal("switching HEAD during review was accepted")
	}
	if _, err := exec.Command("git", "-C", f.remote, "show-ref", "--verify", "--quiet", "refs/heads/main").Output(); err == nil {
		t.Fatal("stale current-branch review pushed")
	}
}

func TestBranchSyncPushUnknownLeaseRequiresExplicitReference(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	f.open(t, branchSyncSource{name: "topic", selected: true})
	if err := menuItem(t, f.ui.menus[0], "branch.push-selected.force-with-lease").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.prompts) != 1 {
		t.Fatal("unknown lease did not require a local ref choice")
	}
	if err := f.ui.prompts[0].HandleClose(); err != nil {
		t.Fatal(err)
	}
	if _, err := exec.Command("git", "-C", f.remote, "show-ref", "--verify", "--quiet", "refs/heads/topic").Output(); err == nil {
		t.Fatal("lease prompt cancel performed a push")
	}
}

func TestBranchSyncPushMultipleURLsCancelFirstThenApprove(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	second := filepath.Join(t.TempDir(), "second.git")
	branchSyncPushGit(t, t.TempDir(), "init", "--bare", "-q", second)
	oldTip := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/topic^")
	branchSyncPushGit(t, f.root, "push", "-q", f.remote, "refs/heads/topic:refs/heads/topic")
	branchSyncPushGit(t, f.root, "push", "-q", second, "refs/heads/topic:refs/heads/topic")
	branchSyncPushGit(t, f.root, "update-ref", "refs/remotes/origin/topic", oldTip)
	branchSyncPushGit(t, f.remote, "update-ref", "refs/heads/topic", oldTip)
	branchSyncPushGit(t, second, "update-ref", "refs/heads/topic", oldTip)
	branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.pushurl", f.remote)
	branchSyncPushGit(t, f.root, "config", "--add", "remote.origin.pushurl", second)
	f.open(t, branchSyncSource{name: "topic", selected: true})
	if err := menuItem(t, f.ui.menus[0], "branch.push-selected.force-with-lease").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	forceReview := f.ui.menus[len(f.ui.menus)-1]
	if !strings.Contains(forceReview.Prompt, "topic") || !strings.Contains(forceReview.Prompt, "origin") || !strings.Contains(forceReview.Prompt, oldTip[:7]) {
		t.Fatalf("force review omitted target or lease details: %q", forceReview.Prompt)
	}
	if err := menuItem(t, forceReview, "branch.push-selected.force-confirm").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	urls := f.ui.menus[len(f.ui.menus)-1]
	if len(urls.Items) < 4 || urls.Items[0].Verb != "cancel" {
		t.Fatalf("multi-URL review is not cancel-first: %+v", urls.Items)
	}
	if !strings.Contains(urls.Prompt, "topic") || !strings.Contains(urls.Prompt, "origin") || !strings.Contains(urls.Prompt, "refs/heads/topic") {
		t.Fatalf("multi-URL menu omitted reviewed force context: %q", urls.Prompt)
	}
	if got := branchSyncPushGit(t, f.remote, "rev-parse", "refs/heads/topic"); got != oldTip {
		t.Fatalf("remote changed before approval: %s", got)
	}
	if err := urls.Items[0].OnPress(); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, second, "rev-parse", "refs/heads/topic"); got != oldTip {
		t.Fatalf("cancel changed second destination: %s", got)
	}
	if err := urls.Items[len(urls.Items)-1].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	for _, remote := range []string{f.remote, second} {
		if got := branchSyncPushGit(t, remote, "rev-parse", "refs/heads/topic"); got != f.sourceOID {
			t.Fatalf("approved destination %s got %s", remote, got)
		}
	}
}

func TestBranchSyncPushPreviousUpstreamURLIsRedacted(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "https://user:secret@example.invalid/repo")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/old")
	f.open(t, branchSyncSource{name: "topic", selected: true})
	if err := menuItem(t, f.ui.menus[0], "branch.push-selected.set-upstream").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-selected.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 || strings.Contains(f.ui.confirms[0].Prompt, "secret") || !strings.Contains(f.ui.confirms[0].Prompt, "example.invalid") {
		t.Fatal("previous upstream URL was not safely displayed")
	}
}

func TestBranchSyncPushSingleUpstreamReplacementIsReviewed(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "oldremote")
	branchSyncPushGit(t, f.root, "config", "--add", "branch.topic.merge", "refs/heads/old-one")
	f.open(t, branchSyncSource{name: "topic", selected: true})
	initial := f.ui.menus[0]
	if err := menuItem(t, initial, "branch.push-selected.set-upstream").OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 0 {
		t.Fatal("toggle wrote or reviewed before choosing push")
	}
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-selected.submodules").OnPress(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.ui.menus[len(f.ui.menus)-1].Prompt, "Check submodules: true") {
		t.Fatal("submodule recursion toggle was not reflected in the menu")
	}
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-selected.execute").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.ui.confirms[0].Prompt, "oldremote") || !strings.Contains(f.ui.confirms[0].Prompt, "old-one") {
		t.Fatalf("review omitted replaced upstream settings: %q", f.ui.confirms[0].Prompt)
	}
	if got := branchSyncPushGit(t, f.root, "config", "--get", "branch.topic.remote"); got != "oldremote" {
		t.Fatalf("upstream changed before approval: %s", got)
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if got := branchSyncPushGit(t, f.root, "config", "--get", "branch.topic.remote"); got != "origin" {
		t.Fatalf("approved SetUpstream did not update remote: %s", got)
	}
	if got := branchSyncPushGit(t, f.root, "config", "--get-all", "branch.topic.merge"); got != "refs/heads/topic" {
		t.Fatalf("single upstream merge ref was not replaced: %s", got)
	}
}

func requestBranchSyncTrackingPush(t *testing.T, f *branchSyncPushFixture, force bool) {
	t.Helper()
	f.open(t, branchSyncSource{name: "topic", selected: true})
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-selected.set-upstream").OnPress(); err != nil {
		t.Fatal(err)
	}
	id := "branch.push-selected.execute"
	if force {
		id = "branch.push-selected.force-with-lease"
	}
	if err := menuItem(t, f.ui.menus[len(f.ui.menus)-1], id).OnPress(); err != nil {
		t.Fatal(err)
	}
}

func branchSyncReadConfig(t *testing.T, path string) []byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func assertBranchSyncConfigUnchanged(t *testing.T, path string, before []byte) {
	t.Helper()
	if !bytes.Equal(branchSyncReadConfig(t, path), before) {
		t.Fatal("push flow changed tracking configuration bytes")
	}
}

func assertBranchSyncRemoteTip(t *testing.T, f *branchSyncPushFixture, want string) {
	t.Helper()
	if got := branchSyncPushGit(t, f.remote, "for-each-ref", "--format=%(objectname)", "refs/heads/topic"); got != want {
		t.Fatalf("remote topic tip = %q, want %q", got, want)
	}
}

func branchSyncUpstreamRefusalMenu(t *testing.T, f *branchSyncPushFixture) types.CreateMenuOptions {
	t.Helper()
	menu := f.ui.menus[len(f.ui.menus)-1]
	if menu.Title != "Upstream settings need attention" || !menu.HideCancel || len(menu.Items) != 2 || menu.Items[0].Label != f.sync.c.Tr.Cancel {
		t.Fatalf("upstream refusal is not Cancel-first: %+v", menu)
	}
	if menu.Items[1].Label != "Push without changing tracking" || !strings.Contains(menu.Prompt, "new push review") {
		t.Fatalf("upstream fallback did not explain another review: %+v", menu)
	}
	return menu
}

func TestBranchSyncPushAmbiguousUpstreamCancelAndFallback(t *testing.T) {
	for _, phase := range []string{"prepare", "execute"} {
		for _, fallback := range []bool{false, true} {
			action := map[bool]string{false: "cancel", true: "fallback"}[fallback]
			t.Run(phase+"/"+action, func(t *testing.T) {
				f := newBranchSyncPushFixture(t)
				branchSyncPushGit(t, f.root, "config", "push.autoSetupRemote", "true")
				branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "oldremote")
				branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/old-one")
				if phase == "prepare" {
					branchSyncPushGit(t, f.root, "config", "--add", "branch.topic.merge", "refs/heads/old-two")
				}
				configPath := filepath.Join(f.root, ".git", "config")
				before := branchSyncReadConfig(t, configPath)
				requestBranchSyncTrackingPush(t, f, false)
				if err := f.ui.runWorker(t); err != nil {
					t.Fatal(err)
				}
				if err := f.ui.runUI(t); err != nil {
					t.Fatal(err)
				}
				if phase == "execute" {
					branchSyncPushGit(t, f.root, "config", "--add", "branch.topic.merge", "refs/heads/old-two")
					before = branchSyncReadConfig(t, configPath)
				}
				if phase == "execute" {
					if len(f.ui.confirms) != 1 {
						t.Fatal("expected original valid tracking review")
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
				}
				refusal := branchSyncUpstreamRefusalMenu(t, f)
				assertBranchSyncConfigUnchanged(t, configPath, before)
				assertBranchSyncRemoteTip(t, f, "")
				if !fallback {
					if err := refusal.Items[0].OnPress(); err != nil {
						t.Fatal(err)
					}
					if len(f.ui.workers) != 0 {
						t.Fatal("cancellation scheduled a push")
					}
					assertBranchSyncConfigUnchanged(t, configPath, before)
					assertBranchSyncRemoteTip(t, f, "")
					return
				}
				if err := refusal.Items[1].OnPress(); err != nil {
					t.Fatal(err)
				}
				if err := f.ui.runWorker(t); err != nil {
					t.Fatal(err)
				}
				if err := f.ui.runUI(t); err != nil {
					t.Fatal(err)
				}
				// Dropping set-upstream makes this an ordinary push: no confirmation,
				// and execution is queued directly with tracking left unchanged.
				if len(f.ui.confirms) != (map[bool]int{false: 0, true: 1}[phase == "execute"]) {
					t.Fatalf("fallback asked for a confirmation: %+v", f.ui.confirms)
				}
				assertBranchSyncConfigUnchanged(t, configPath, before)
				assertBranchSyncRemoteTip(t, f, "")
				if err := f.ui.runWorker(t); err != nil {
					t.Fatal(err)
				}
				assertBranchSyncRemoteTip(t, f, f.sourceOID)
				assertBranchSyncConfigUnchanged(t, configPath, before)
			})
		}
	}
}

func TestBranchSyncPushAmbiguousRemoteOrInheritedTrackingCancels(t *testing.T) {
	for _, kind := range []string{"multiple-remotes", "inherited"} {
		t.Run(kind, func(t *testing.T) {
			f := newBranchSyncPushFixture(t)
			var includedPath string
			var includedBefore []byte
			if kind == "multiple-remotes" {
				branchSyncPushGit(t, f.root, "config", "--add", "branch.topic.remote", "oldremote")
				branchSyncPushGit(t, f.root, "config", "--add", "branch.topic.remote", "otherremote")
				branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/old")
			} else {
				includedPath = filepath.Join(t.TempDir(), "tracking.inc")
				includedBefore = []byte("[branch \"topic\"]\n\tremote = oldremote\n\tmerge = refs/heads/old\n")
				if err := os.WriteFile(includedPath, includedBefore, 0o600); err != nil {
					t.Fatal(err)
				}
				branchSyncPushGit(t, f.root, "config", "include.path", includedPath)
			}
			configPath := filepath.Join(f.root, ".git", "config")
			before := branchSyncReadConfig(t, configPath)
			requestBranchSyncTrackingPush(t, f, false)
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			refusal := branchSyncUpstreamRefusalMenu(t, f)
			if err := refusal.Items[0].OnPress(); err != nil {
				t.Fatal(err)
			}
			assertBranchSyncConfigUnchanged(t, configPath, before)
			if includedPath != "" {
				assertBranchSyncConfigUnchanged(t, includedPath, includedBefore)
			}
			assertBranchSyncRemoteTip(t, f, "")
		})
	}
}

func TestBranchSyncPushAmbiguousUpstreamForceFallbackPreservesReviewedLease(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "config", "push.autoSetupRemote", "true")
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "oldremote")
	branchSyncPushGit(t, f.root, "config", "branch.topic.merge", "refs/heads/old-one")
	oldTip := branchSyncPushGit(t, f.root, "rev-parse", "refs/heads/main")
	branchSyncPushGit(t, f.root, "push", "-q", f.remote, "refs/heads/main:refs/heads/topic")
	leaseRef := "refs/reviewed/topic-lease"
	branchSyncPushGit(t, f.root, "update-ref", leaseRef, oldTip)
	requestBranchSyncTrackingPush(t, f, true)
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.prompts) != 1 {
		t.Fatal("expected explicit lease choice before initial force review")
	}
	if err := f.ui.prompts[0].HandleConfirm(leaseRef); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	forceReview := f.ui.menus[len(f.ui.menus)-1]
	branchSyncPushGit(t, f.root, "config", "--add", "branch.topic.merge", "refs/heads/old-two")
	configPath := filepath.Join(f.root, ".git", "config")
	before := branchSyncReadConfig(t, configPath)
	if err := menuItem(t, forceReview, "branch.push-selected.force-confirm").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	refusal := branchSyncUpstreamRefusalMenu(t, f)
	assertBranchSyncRemoteTip(t, f, oldTip)
	assertBranchSyncConfigUnchanged(t, configPath, before)
	if err := refusal.Items[1].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	review := f.ui.menus[len(f.ui.menus)-1]
	for _, text := range []string{"Set upstream for selected local branch: false", "origin", "refs/heads/topic", leaseRef, oldTip[:7], f.sourceOID[:7]} {
		if !strings.Contains(review.Prompt, text) {
			t.Fatalf("force fallback review omitted %q: %q", text, review.Prompt)
		}
	}
	if review.Items[0].Verb != "cancel" || len(f.ui.prompts) != 1 || len(f.ui.workers) != 0 {
		t.Fatal("force fallback lost its lease, defaulted to push, or automatically executed")
	}
	assertBranchSyncRemoteTip(t, f, oldTip)
	assertBranchSyncConfigUnchanged(t, configPath, before)
	if err := menuItem(t, review, "branch.push-selected.force-confirm").OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	assertBranchSyncRemoteTip(t, f, f.sourceOID)
	assertBranchSyncConfigUnchanged(t, configPath, before)
}

func TestBranchSyncPushAmbiguousForceLeasePromptCapturesOptionsAndSource(t *testing.T) {
	f := newBranchSyncPushFixture(t)
	branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "oldremote")
	branchSyncPushGit(t, f.root, "config", "--add", "branch.topic.merge", "refs/heads/old-one")
	branchSyncPushGit(t, f.root, "config", "--add", "branch.topic.merge", "refs/heads/old-two")
	configPath := filepath.Join(f.root, ".git", "config")
	before := branchSyncReadConfig(t, configPath)
	info, err := f.git.Sync.InspectBranchSync("topic")
	if err != nil {
		t.Fatal(err)
	}
	flow := &branchSyncPushFlow{
		target: newBranchSyncTarget(f.sync.c, branchSyncSource{name: "topic", selected: true}),
		info:   info,
		opts:   git_commands.BranchPushOptions{Remote: "origin", DestinationRef: "refs/heads/topic", SetUpstream: true},
	}
	if err := flow.prepare(true); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	refusal := branchSyncUpstreamRefusalMenu(t, f)
	// A later flow edit cannot redirect a retained refusal/lease callback.
	flow.opts.Remote = "unconfigured-later-remote"
	flow.opts.DestinationRef = "refs/heads/later-destination"
	flow.info.OID = strings.Repeat("a", len(f.sourceOID))
	if err := refusal.Items[1].OnPress(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.prompts) != 1 {
		t.Fatal("force fallback did not request its missing lease")
	}
	if err := f.ui.prompts[0].HandleConfirm("refs/heads/main"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if flow.plan == nil {
		t.Fatal("missing force fallback plan")
	}
	opts := flow.plan.Options()
	if !opts.ForceWithLease || opts.SetUpstream || opts.Remote != "origin" || opts.DestinationRef != "refs/heads/topic" || flow.plan.LeaseRef() != "refs/heads/main" || flow.plan.SourceOID() != f.sourceOID {
		t.Fatalf("lease callback substituted mutable flow inputs: %+v", opts)
	}
	menuItem(t, f.ui.menus[len(f.ui.menus)-1], "branch.push-selected.force-confirm")
	assertBranchSyncRemoteTip(t, f, "")
	assertBranchSyncConfigUnchanged(t, configPath, before)
}

func TestBranchSyncPushAmbiguousUpstreamRejectsStaleCallbacks(t *testing.T) {
	for _, phase := range []string{"queued-modal", "shown-modal"} {
		for _, generationChange := range []bool{false, true} {
			kind := map[bool]string{false: "git-pointer", true: "generation"}[generationChange]
			t.Run(phase+"/"+kind, func(t *testing.T) {
				f := newBranchSyncPushFixture(t)
				branchSyncPushGit(t, f.root, "config", "branch.topic.remote", "oldremote")
				branchSyncPushGit(t, f.root, "config", "--add", "branch.topic.merge", "refs/heads/old-one")
				branchSyncPushGit(t, f.root, "config", "--add", "branch.topic.merge", "refs/heads/old-two")
				configPath := filepath.Join(f.root, ".git", "config")
				before := branchSyncReadConfig(t, configPath)
				requestBranchSyncTrackingPush(t, f, false)
				if err := f.ui.runWorker(t); err != nil {
					t.Fatal(err)
				}
				var refusal types.CreateMenuOptions
				if phase == "shown-modal" {
					if err := f.ui.runUI(t); err != nil {
						t.Fatal(err)
					}
					refusal = branchSyncUpstreamRefusalMenu(t, f)
				}
				menuCount := len(f.ui.menus)
				if generationChange {
					f.ui.state.generation++
				} else {
					f.ui.git = &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths("/other/repo")}
				}
				if phase == "queued-modal" {
					if err := f.ui.runUI(t); err == nil {
						t.Fatal("queued upstream refusal ignored repository change")
					}
				} else {
					if err := refusal.Items[0].OnPress(); err == nil {
						t.Fatal("stale refusal cancellation ignored repository change")
					}
					if err := refusal.Items[1].OnPress(); err == nil {
						t.Fatal("stale refusal fallback ignored repository change")
					}
				}
				if len(f.ui.workers) != 0 || len(f.ui.menus) != menuCount {
					t.Fatal("stale callback scheduled work or presented another modal")
				}
				assertBranchSyncRemoteTip(t, f, "")
				assertBranchSyncConfigUnchanged(t, configPath, before)
			})
		}
	}
}
