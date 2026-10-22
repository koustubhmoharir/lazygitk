package helpers

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// fakeIGuiCommon keeps this test focused on the popup/worker seam. Any GUI
// method not explicitly needed by named checkout panics through the embedded
// interface instead of silently acting like a real GUI.
type namedCheckoutFakeIGuiCommon struct {
	types.IGuiCommon
	git          *commands.GitCommand
	model        types.Model
	prompts      []types.PromptOpts
	confirms     []types.ConfirmOpts
	alerts       []string
	workers      []func(gocui.Task) error
	ui           []func() error
	refreshes    []types.RefreshOptions
	blockingOpts []types.WaitingStatusOpts
}

func (f *namedCheckoutFakeIGuiCommon) Git() *commands.GitCommand { return f.git }
func (f *namedCheckoutFakeIGuiCommon) Model() *types.Model       { return &f.model }
func (f *namedCheckoutFakeIGuiCommon) Prompt(opts types.PromptOpts) {
	f.prompts = append(f.prompts, opts)
}

func (f *namedCheckoutFakeIGuiCommon) Confirm(opts types.ConfirmOpts) {
	f.confirms = append(f.confirms, opts)
}

func (f *namedCheckoutFakeIGuiCommon) Alert(title, message string) {
	f.alerts = append(f.alerts, title+"\n"+message)
}
func (f *namedCheckoutFakeIGuiCommon) LogAction(string)           {}
func (f *namedCheckoutFakeIGuiCommon) OnUIThread(fn func() error) { f.ui = append(f.ui, fn) }
func (f *namedCheckoutFakeIGuiCommon) RefreshFromWorker(opts types.RefreshOptions) {
	f.refreshes = append(f.refreshes, opts)
}

func (f *namedCheckoutFakeIGuiCommon) WithWaitingStatus(_ string, fn func(gocui.Task) error) error {
	f.workers = append(f.workers, fn)
	return nil
}

func (f *namedCheckoutFakeIGuiCommon) WithWaitingStatusBlockingInput(opts types.WaitingStatusOpts, fn func(gocui.Task) error) error {
	f.blockingOpts = append(f.blockingOpts, opts)
	f.workers = append(f.workers, fn)
	return nil
}

func (f *namedCheckoutFakeIGuiCommon) runWorker(t *testing.T) error {
	t.Helper()
	if len(f.workers) == 0 {
		t.Fatal("expected a queued worker")
	}
	fn := f.workers[0]
	f.workers = f.workers[1:]
	return fn(nil)
}

func (f *namedCheckoutFakeIGuiCommon) runUI(t *testing.T) error {
	t.Helper()
	if len(f.ui) == 0 {
		t.Fatal("expected a queued UI callback")
	}
	fn := f.ui[0]
	f.ui = f.ui[1:]
	return fn()
}

type namedCheckoutFixture struct {
	root  string
	c     *HelperCommon
	ui    *namedCheckoutFakeIGuiCommon
	oid   string
	other string
}

func namedCheckoutGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func namedCheckoutWrite(t *testing.T, root, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func newNamedCheckoutFixture(t *testing.T) *namedCheckoutFixture {
	t.Helper()
	root := t.TempDir()
	namedCheckoutGit(t, root, "init", "-q", "-b", "main")
	namedCheckoutGit(t, root, "config", "user.name", "Test")
	namedCheckoutGit(t, root, "config", "user.email", "test@example.invalid")
	namedCheckoutWrite(t, root, "tracked", "base\n")
	namedCheckoutWrite(t, root, "carry", "base carry\n")
	namedCheckoutGit(t, root, "add", "tracked", "carry")
	namedCheckoutGit(t, root, "commit", "-qm", "base")
	oid := namedCheckoutGit(t, root, "rev-parse", "HEAD")
	namedCheckoutGit(t, root, "checkout", "-qb", "other")
	namedCheckoutWrite(t, root, "tracked", "other\n")
	namedCheckoutGit(t, root, "commit", "-qam", "other")
	other := namedCheckoutGit(t, root, "rev-parse", "HEAD")
	namedCheckoutGit(t, root, "checkout", "-q", "main")
	paths := git_commands.MockRepoPaths(root)
	osCmd := oscommands.NewDummyOSCommand()
	branch := git_commands.NewBranchCommands(git_commands.NewGitCommon(common.NewDummyCommon(), nil, osCmd.Cmd, osCmd, paths, nil, nil))
	ui := &namedCheckoutFakeIGuiCommon{git: &commands.GitCommand{Branch: branch, RepoPaths: paths}}
	c := &HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui}
	return &namedCheckoutFixture{root: root, c: c, ui: ui, oid: oid, other: other}
}

func (f *namedCheckoutFixture) item(oid string) *types.MenuItem {
	return (&RefsHelper{c: f.c}).namedCheckoutItem(oid)
}

func (f *namedCheckoutFixture) begin(t *testing.T, name string) {
	t.Helper()
	if err := f.item(f.oid).OnPress(); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.prompts) != 1 {
		t.Fatal("expected name prompt")
	}
	if err := f.ui.prompts[0].HandleConfirm(name); err != nil {
		t.Fatal(err)
	}
}

func (f *namedCheckoutFixture) prepare(t *testing.T) {
	t.Helper()
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
}

func TestNamedCheckoutItemAndTipLabels(t *testing.T) {
	f := newNamedCheckoutFixture(t)
	item := f.item(f.oid)
	if item.CommandID != "branch.checkout-at-selected-commit" || item.Verb != "checkout" || item.Label != f.c.Tr.CheckoutAsBranch || !item.OpensMenu || len(item.Keys) != 1 || item.Keys[0] != gocui.NewKeyRune('b') {
		t.Fatalf("unexpected menu item: %+v", item)
	}
	invalid := f.item("HEAD")
	if invalid.DisabledReason == nil {
		t.Fatal("malformed OID should be disabled")
	}
	if err := invalid.OnPress(); err == nil || len(f.ui.prompts) != 0 || len(f.ui.workers) != 0 {
		t.Fatal("malformed OID invoked the flow")
	}
	noGit := &HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: &namedCheckoutFakeIGuiCommon{}}
	noGitItem := (&RefsHelper{c: noGit}).namedCheckoutItem("HEAD")
	if err := noGitItem.OnPress(); err == nil {
		t.Fatal("malformed OID should be rejected before requiring Git")
	}
	if gotOld, gotNew := namedCheckoutTipLabels("", ""); gotOld != "" || gotNew != "" {
		t.Fatalf("empty tips: %q %q", gotOld, gotNew)
	}
	old := "1234567000000000000000000000000000000000"
	new := "1234567fffffffffffffffffffffffffffffffff"
	gotOld, gotNew := namedCheckoutTipLabels(old, new)
	if gotOld != old[:8] || gotNew != new[:8] {
		t.Fatalf("overlapping labels = %q, %q", gotOld, gotNew)
	}
}

func TestNamedCheckoutCapturesOIDAndRepositoryAtMenuCreation(t *testing.T) {
	f := newNamedCheckoutFixture(t)
	item := f.item(f.oid)
	capturedOID := f.oid
	f.oid = f.other
	if err := item.OnPress(); err != nil {
		t.Fatal(err)
	}
	if !f.ui.prompts[0].PreserveWhitespace {
		t.Fatal("branch input must preserve whitespace")
	}
	if f.ui.prompts[0].InitialContent != "" {
		t.Fatalf("unexpected initial branch name %q", f.ui.prompts[0].InitialContent)
	}
	if !strings.Contains(f.ui.prompts[0].Title, capturedOID[:7]) {
		t.Fatalf("prompt did not use captured OID %s: %q", capturedOID, f.ui.prompts[0].Title)
	}
	f.ui.model.Branches = nil
	if err := f.ui.prompts[0].HandleConfirm("captured"); err != nil {
		t.Fatal(err)
	}
	// A queued worker belongs to its original GitCommand. It prepares there;
	// repository identity is checked back on the UI thread before any apply.
	f.ui.git = &commands.GitCommand{}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatalf("captured repository worker failed: %v", err)
	}
	if err := f.ui.runUI(t); err == nil {
		t.Fatal("repository replacement was accepted at UI continuation")
	}
	if len(f.ui.workers) != 0 || len(f.ui.confirms) != 0 {
		t.Fatal("stale repository queued apply or confirmation")
	}
	if _, err := os.Stat(filepath.Join(f.root, ".git", "refs", "heads", "captured")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unexpected branch creation: %v", err)
	}
}

func TestNamedCheckoutCreateAcceptChecksOutAndCarriesSafeDirtyChanges(t *testing.T) {
	for _, differentTree := range []bool{false, true} {
		t.Run(fmt.Sprintf("different-tree-%v", differentTree), func(t *testing.T) {
			f := newNamedCheckoutFixture(t)
			if differentTree {
				namedCheckoutGit(t, f.root, "checkout", "-q", "other")
			}
			namedCheckoutWrite(t, f.root, "carry", "staged carry\n")
			namedCheckoutGit(t, f.root, "add", "carry")
			index := namedCheckoutGit(t, f.root, "write-tree")
			namedCheckoutWrite(t, f.root, "carry", "working carry\n")
			namedCheckoutWrite(t, f.root, "loose", "untracked\n")
			f.begin(t, "new-topic")
			f.prepare(t)
			if len(f.ui.confirms) != 0 || len(f.ui.blockingOpts) != 1 {
				t.Fatal("new branch should apply after name acceptance")
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if got := namedCheckoutGit(t, f.root, "symbolic-ref", "--no-recurse", "HEAD"); got != "refs/heads/new-topic" {
				t.Fatalf("HEAD ref = %q", got)
			}
			if got := namedCheckoutGit(t, f.root, "rev-parse", "HEAD"); got != f.oid {
				t.Fatalf("HEAD = %q want %q", got, f.oid)
			}
			if got := namedCheckoutGit(t, f.root, "show", ":carry"); got != "staged carry" {
				t.Fatalf("staged carried content = %q", got)
			}
			if !differentTree && namedCheckoutGit(t, f.root, "write-tree") != index {
				t.Fatal("same-tree checkout changed the index")
			}
			for file, want := range map[string]string{"carry": "working carry\n", "loose": "untracked\n"} {
				data, err := os.ReadFile(filepath.Join(f.root, file))
				if err != nil || string(data) != want {
					t.Fatalf("%s = %q, %v", file, data, err)
				}
			}
			if len(f.ui.refreshes) != 1 || !reflect.DeepEqual(f.ui.refreshes[0].Scope, namedCheckoutRefreshOptions().Scope) || f.ui.refreshes[0].Then == nil {
				t.Fatalf("missing success refresh: %+v", f.ui.refreshes)
			}
		})
	}
}

func TestNamedCheckoutHeadIdentityChangeRequiresFreshConfirmation(t *testing.T) {
	f := newNamedCheckoutFixture(t)
	namedCheckoutGit(t, f.root, "branch", "topic", f.other)
	namedCheckoutGit(t, f.root, "branch", "same-oid")
	f.begin(t, "topic")
	f.prepare(t)
	if len(f.ui.confirms) != 1 {
		t.Fatal("expected initial move confirmation")
	}
	if !strings.Contains(f.ui.confirms[0].Prompt, `"main" (`+f.oid[:8]+`)`) {
		t.Fatalf("initial confirmation omitted current checkout identity: %s", f.ui.confirms[0].Prompt)
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	// Switch to a different branch at exactly the same OID. The warning must
	// become stale based on HEAD's symbolic identity, not only its commit hash.
	namedCheckoutGit(t, f.root, "checkout", "-q", "same-oid")
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.ui) != 1 || len(f.ui.workers) != 0 {
		t.Fatal("HEAD identity change should prepare on the apply worker and queue one UI callback")
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 2 || f.ui.confirms[1].Title != f.c.Tr.NamedCheckoutChangedTitle {
		t.Fatal("HEAD identity change did not require fresh confirmation")
	}
	if !strings.Contains(f.ui.confirms[1].Prompt, `"same-oid" (`+f.oid[:8]+`)`) {
		t.Fatalf("fresh confirmation omitted changed HEAD identity/OID: %s", f.ui.confirms[1].Prompt)
	}
	if len(f.ui.workers) != 0 {
		t.Fatal("stale checkout was applied without fresh confirmation")
	}
}

func TestNamedCheckoutMoveWaitsForConfirmationAndRechecksChangedDestination(t *testing.T) {
	for _, change := range []string{"move", "delete"} {
		t.Run(change, func(t *testing.T) {
			f := newNamedCheckoutFixture(t)
			namedCheckoutGit(t, f.root, "branch", "topic", f.other)
			f.begin(t, "topic")
			f.prepare(t)
			if len(f.ui.confirms) != 1 || len(f.ui.blockingOpts) != 0 {
				t.Fatal("existing branch must warn before applying")
			}
			warning := f.ui.confirms[0]
			if !strings.Contains(warning.Prompt, f.other[:7]) || !strings.Contains(warning.Prompt, f.oid[:7]) || !strings.Contains(warning.Prompt, "topic") {
				t.Fatalf("incomplete warning: %s", warning.Prompt)
			}
			if got := namedCheckoutGit(t, f.root, "rev-parse", "topic"); got != f.other {
				t.Fatal("warning moved branch before confirmation")
			}
			if err := warning.HandleConfirm(); err != nil {
				t.Fatal(err)
			}
			if change == "move" {
				namedCheckoutGit(t, f.root, "update-ref", "refs/heads/topic", f.oid, f.other)
			} else {
				namedCheckoutGit(t, f.root, "branch", "-D", "topic")
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			if len(f.ui.ui) != 1 || len(f.ui.workers) != 0 {
				t.Fatal("stale plan should prepare on its apply worker and queue one UI callback")
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if len(f.ui.confirms) != 2 || f.ui.confirms[1].Title != f.c.Tr.NamedCheckoutChangedTitle {
				t.Fatal("changed destination did not require fresh confirmation")
			}
			if len(f.ui.workers) != 0 {
				t.Fatal("fresh plan applied without confirmation")
			}
			if change == "move" && !strings.Contains(f.ui.confirms[1].Prompt, f.oid[:7]) {
				t.Fatal("fresh warning omitted new tip")
			}
		})
	}
}

func TestNamedCheckoutSameTipStillChecksOutAndRepositoryGuards(t *testing.T) {
	f := newNamedCheckoutFixture(t)
	namedCheckoutGit(t, f.root, "branch", "same", f.oid)
	f.begin(t, "same")
	f.prepare(t)
	if len(f.ui.confirms) != 0 || len(f.ui.blockingOpts) != 1 {
		t.Fatal("same-tip destination should checkout without move warning")
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if namedCheckoutGit(t, f.root, "symbolic-ref", "--no-recurse", "HEAD") != "refs/heads/same" {
		t.Fatal("same-tip checkout did not attach HEAD")
	}

	g := newNamedCheckoutFixture(t)
	if err := g.item(g.oid).OnPress(); err != nil {
		t.Fatal(err)
	}
	g.ui.git = &commands.GitCommand{}
	if err := g.ui.prompts[0].HandleConfirm("refused"); err == nil || len(g.ui.workers) != 0 {
		t.Fatal("repository change before name acceptance queued work")
	}

	h := newNamedCheckoutFixture(t)
	namedCheckoutGit(t, h.root, "branch", "topic", h.other)
	h.begin(t, "topic")
	h.prepare(t)
	h.ui.git = &commands.GitCommand{}
	if err := h.ui.confirms[0].HandleConfirm(); err == nil || len(h.ui.workers) != 0 {
		t.Fatal("repository change before move confirmation queued work")
	}
}

func TestNamedCheckoutFailedApplyRefreshesActualStateAndReturnsError(t *testing.T) {
	for _, failure := range []string{"dirty-overwrite", "post-checkout-hook"} {
		t.Run(failure, func(t *testing.T) {
			f := newNamedCheckoutFixture(t)
			f.oid = f.other
			if failure == "dirty-overwrite" {
				namedCheckoutWrite(t, f.root, "tracked", "local change\n")
			} else {
				hooks := filepath.Join(f.root, ".git", "hooks")
				if err := os.MkdirAll(hooks, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(hooks, "post-checkout"), []byte("#!/bin/sh\necho checkout-hook-failed >&2\nexit 23\n"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			f.begin(t, "new-topic")
			f.prepare(t)
			err := f.ui.runWorker(t)
			if err == nil {
				t.Fatal("expected checkout failure")
			}
			if !strings.Contains(err.Error(), "checkout") && !strings.Contains(err.Error(), "hook") && !strings.Contains(err.Error(), "overwritten") {
				t.Fatalf("unexpected failure: %v", err)
			}
			if len(f.ui.refreshes) != 1 {
				t.Fatalf("failed operation did not refresh actual state: %+v", f.ui.refreshes)
			}
			want := namedCheckoutRefreshOptions()
			got := f.ui.refreshes[0]
			if !reflect.DeepEqual(got.Scope, want.Scope) || got.BranchSelection != types.SelectCheckedOutBranch || got.CommitSelection != types.KeepCommitSelectionByHash || got.SelectTopReflogCommit {
				t.Fatalf("incorrect failure refresh: %+v", got)
			}
			if got.Then != nil {
				t.Fatal("failed checkout must not run the success continuation")
			}
			if failure == "post-checkout-hook" && namedCheckoutGit(t, f.root, "rev-parse", "HEAD") != f.other {
				t.Fatal("hook failure did not preserve Git's already-completed checkout")
			}
			if len(f.ui.alerts) != 0 {
				t.Fatal("checkout failure must not offer autostash")
			}
			if len(f.ui.confirms) != 0 {
				t.Fatal("checkout failure must not offer autostash confirmation")
			}
		})
	}
}

func TestNamedCheckoutTipLabelsEmptyAndSharedPrefix(t *testing.T) {
	if a, b := namedCheckoutTipLabels(strings.Repeat("a", 40), strings.Repeat("a", 40)); a != strings.Repeat("a", 8) || b != a {
		t.Fatalf("matching tips need only a short label: %q %q", a, b)
	}
	if a, b := namedCheckoutTipLabels("", "abcdef0"); a != "" || b != "abcdef0" {
		t.Fatalf("empty tip labels: %q %q", a, b)
	}
	if a, b := namedCheckoutTipLabels("abcdef0123456789", "abcdef0fedcba987"); a != "abcdef01" || b != "abcdef0f" {
		t.Fatalf("shared-prefix labels: %q %q", a, b)
	}
}
