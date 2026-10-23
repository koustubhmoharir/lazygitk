package helpers

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type capturedTagState struct {
	types.IStateAccessor
	generation int
}

func (s *capturedTagState) GetRepoGeneration() int { return s.generation }

type capturedTagUI struct {
	*namedCheckoutFakeIGuiCommon
	state         *capturedTagState
	subprocess    []*oscommands.CmdObj
	subprocessErr error
}

func (u *capturedTagUI) State() types.IStateAccessor { return u.state }
func (u *capturedTagUI) Refresh(opts types.RefreshOptions) {
	u.refreshes = append(u.refreshes, opts)
}

func (u *capturedTagUI) RunSubprocess(cmd *oscommands.CmdObj) (bool, error) {
	u.subprocess = append(u.subprocess, cmd)
	return false, u.subprocessErr
}

type capturedTagFixture struct {
	*namedCheckoutFixture
	ui   *capturedTagUI
	flow *capturedTagCreateFlow
}

func newCapturedTagFixture(t *testing.T) *capturedTagFixture {
	t.Helper()
	f := newNamedCheckoutFixture(t)
	base := f.ui
	base.git.Tag = git_commands.NewTagCommands(base.git.Branch.GitCommon)
	u := &capturedTagUI{namedCheckoutFakeIGuiCommon: base, state: &capturedTagState{generation: 10}}
	f.c.IGuiCommon = u
	flow := &capturedTagCreateFlow{repository: captureActionRepository(f.c)}
	flow.target, _ = base.git.Branch.PrepareBranchActionHeadTarget()
	if flow.target.SourceOID() == "" {
		t.Fatal("native HEAD target was not prepared")
	}
	return &capturedTagFixture{namedCheckoutFixture: f, ui: u, flow: flow}
}

func (f *capturedTagFixture) submit(t *testing.T, name, message string) error {
	t.Helper()
	if err := f.flow.submitName(name, message); err != nil {
		return err
	}
	return f.ui.runWorker(t)
}

func capturedTagArgsContain(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func capturedTagSnapshot(t *testing.T, root string) (string, string, []byte, map[string][]byte) {
	t.Helper()
	head := namedCheckoutGit(t, root, "rev-parse", "HEAD")
	index, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	rawHead, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, name := range []string{"tracked", "carry", "loose"} {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err == nil {
			files[name] = b
		}
	}
	return head, string(index), rawHead, files
}

func assertCapturedTagSnapshot(t *testing.T, f *capturedTagFixture, beforeHead, beforeIndex string, beforeRawHead []byte, beforeFiles map[string][]byte) {
	t.Helper()
	assertCapturedTagSnapshotAt(t, f.root, beforeHead, beforeIndex, beforeRawHead, beforeFiles)
}

func assertCapturedTagSnapshotAt(t *testing.T, root, beforeHead, beforeIndex string, beforeRawHead []byte, beforeFiles map[string][]byte) {
	t.Helper()
	head, index, rawHead, files := capturedTagSnapshot(t, root)
	if head != beforeHead || index != beforeIndex || !reflect.DeepEqual(rawHead, beforeRawHead) || !reflect.DeepEqual(files, beforeFiles) {
		t.Fatalf("tag flow changed checkout: HEAD %s/%s rawHEAD %q/%q index %s/%s files %#v/%#v", beforeHead, head, beforeRawHead, rawHead, beforeIndex, index, beforeFiles, files)
	}
}

func TestCapturedTagCreateLightweightAndAnnotatedPreserveCheckout(t *testing.T) {
	for _, tc := range []struct {
		name, message string
		annotated     bool
	}{{"light", "", false}, {"annotated", "release note", true}} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCapturedTagFixture(t)
			// Selection differs from current HEAD. Tag placement remains bound to selected.
			selected := f.other
			f.flow.target, _ = f.ui.git.Branch.PrepareBranchActionTarget("refs/heads/other", selected)
			namedCheckoutWrite(t, f.root, "tracked", "staged bytes\n")
			namedCheckoutGit(t, f.root, "add", "tracked")
			namedCheckoutWrite(t, f.root, "tracked", "working bytes\n")
			namedCheckoutWrite(t, f.root, "carry", "unstaged bytes\n")
			beforeHead, beforeIndex, beforeRawHead, beforeFiles := capturedTagSnapshot(t, f.root)
			if err := f.submit(t, tc.name, tc.message); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if len(f.ui.workers) != 1 {
				t.Fatalf("expected one queued execution worker; got %d", len(f.ui.workers))
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			tagType := namedCheckoutGit(t, f.root, "cat-file", "-t", "refs/tags/"+tc.name)
			if tc.annotated && tagType != "tag" || !tc.annotated && tagType != "commit" {
				t.Fatalf("tag object type %q", tagType)
			}
			if namedCheckoutGit(t, f.root, "rev-parse", "refs/tags/"+tc.name+"^{commit}") != selected {
				t.Fatal("tag did not target selected commit")
			}
			assertCapturedTagSnapshot(t, f, beforeHead, beforeIndex, beforeRawHead, beforeFiles)
		})
	}
}

func TestCapturedTagCreateOverwriteCancelAndStaleApproval(t *testing.T) {
	f := newCapturedTagFixture(t)
	name := "replace-me"
	namedCheckoutGit(t, f.root, "tag", "-a", name, "-m", "old", f.oid)
	oldObject := namedCheckoutGit(t, f.root, "rev-parse", "refs/tags/"+name)
	beforeHead, beforeIndex, beforeRawHead, beforeFiles := capturedTagSnapshot(t, f.root)
	if err := f.submit(t, name, "new"); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatal("overwrite did not ask for approval")
	}
	if err := f.ui.confirms[0].HandleClose(); err != nil {
		t.Fatal(err)
	}
	if namedCheckoutGit(t, f.root, "rev-parse", "refs/tags/"+name) != oldObject || len(f.ui.refreshes) != 0 {
		t.Fatal("cancel changed tag or refreshed")
	}
	if err := f.ui.confirms[0].HandleConfirm(); err == nil {
		t.Fatal("stale confirmation resurrected cancelled flow")
	}
	assertCapturedTagSnapshot(t, f, beforeHead, beforeIndex, beforeRawHead, beforeFiles)

	g := newCapturedTagFixture(t)
	namedCheckoutGit(t, g.root, "tag", "-a", name, "-m", "old", g.oid)
	oldObject = namedCheckoutGit(t, g.root, "rev-parse", "refs/tags/"+name)
	if err := g.submit(t, name, "new"); err != nil {
		t.Fatal(err)
	}
	if err := g.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	confirm := g.ui.confirms[0]
	if err := confirm.HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	namedCheckoutGit(t, g.root, "tag", "-fa", name, "-m", "concurrent", g.other)
	if err := g.ui.runWorker(t); err == nil {
		t.Fatal("approved replacement accepted changed old tag object")
	}
	if got := namedCheckoutGit(t, g.root, "rev-parse", "refs/tags/"+name); got == oldObject {
		t.Fatal("concurrent tag update absent")
	}
}

func TestCapturedTagCreateDuplicateNameSubmissionAndConfirmationAreGuarded(t *testing.T) {
	f := newCapturedTagFixture(t)
	namedCheckoutGit(t, f.root, "tag", "single", f.oid)
	if err := f.flow.submitName("single", ""); err != nil {
		t.Fatal(err)
	}
	if err := f.flow.submitName("duplicate", ""); err == nil {
		t.Fatal("duplicate name submission accepted")
	}
	if len(f.ui.workers) != 1 {
		t.Fatalf("duplicate submission queued work: %d workers", len(f.ui.workers))
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.confirms) != 1 {
		t.Fatal("expected a single overwrite confirmation")
	}
	if err := f.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.confirms[0].HandleConfirm(); err == nil {
		t.Fatal("duplicate confirmation accepted")
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.refreshes) != 1 {
		t.Fatalf("duplicate execution caused %d refreshes", len(f.ui.refreshes))
	}
}

func TestCapturedTagCreateGuardsQueuedWorkAndCallbacks(t *testing.T) {
	callbackCalls := 0
	for _, change := range []string{"source", "head", "config", "destination", "repository", "generation"} {
		t.Run(change, func(t *testing.T) {
			f := newCapturedTagFixture(t)
			f.flow.onCreate = func() { callbackCalls++ }
			if change == "destination" {
				namedCheckoutGit(t, f.root, "tag", "guarded", f.oid)
			}
			if err := f.submit(t, "guarded", ""); err != nil {
				t.Fatal(err)
			}
			if change == "repository" {
				f.ui.git = &commands.GitCommand{}
			}
			if change == "generation" {
				f.ui.state.generation++
			}
			err := f.ui.runUI(t)
			if change == "repository" || change == "generation" {
				if err == nil || len(f.ui.workers) != 0 || len(f.ui.refreshes) != 0 {
					t.Fatal("repository guard queued worker or refresh")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(f.ui.confirms) > 0 {
				if err := f.ui.confirms[0].HandleConfirm(); err != nil {
					t.Fatal(err)
				}
			}
			if change == "source" {
				namedCheckoutGit(t, f.root, "update-ref", f.flow.target.SourceRef(), f.other)
			}
			if change == "head" {
				namedCheckoutGit(t, f.root, "checkout", "-q", "other")
			}
			if change == "config" {
				namedCheckoutGit(t, f.root, "config", "tag.gpgSign", "true")
			}
			if change == "destination" {
				namedCheckoutGit(t, f.root, "tag", "-fa", "guarded", "-m", "changed", f.oid)
			}
			postMutationHead, postMutationIndex, postMutationRawHead, postMutationFiles := capturedTagSnapshot(t, f.root)
			postMutationTag := ""
			if change == "destination" {
				postMutationTag = namedCheckoutGit(t, f.root, "rev-parse", "refs/tags/guarded")
			}
			if err := f.ui.runWorker(t); err == nil {
				t.Fatal("stale native snapshot accepted by worker")
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if len(f.ui.refreshes) != 1 || f.ui.refreshes[0].Then == nil {
				t.Fatal("failed operation did not queue its guarded refresh continuation")
			}
			if err := f.ui.refreshes[0].Then(); err != nil {
				t.Fatal(err)
			}
			if callbackCalls != 0 {
				t.Fatal("failed operation ran onCreate callback")
			}
			assertCapturedTagSnapshot(t, f, postMutationHead, postMutationIndex, postMutationRawHead, postMutationFiles)
			if change == "destination" && namedCheckoutGit(t, f.root, "rev-parse", "refs/tags/guarded") != postMutationTag {
				t.Fatal("failed operation replaced externally updated destination")
			}
		})
	}

	f := newCapturedTagFixture(t)
	called := 0
	f.flow.onCreate = func() { called++ }
	if err := f.submit(t, "once", ""); err != nil {
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
	if len(f.ui.refreshes) != 1 || f.ui.refreshes[0].Then == nil {
		t.Fatal("success refresh continuation missing")
	}
	if err := f.ui.refreshes[0].Then(); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.refreshes[0].Then(); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("onCreate called %d times", called)
	}
}

func TestCapturedTagCreateSignedTerminalIsCapturedAndRepositoryGuarded(t *testing.T) {
	f := newCapturedTagFixture(t)
	// Exercise the captured configuration branch without invoking a GPG tool.
	namedCheckoutGit(t, f.root, "config", "tag.gpgSign", "true")
	f.flow.overrideGpg = false
	if err := f.submit(t, "signed", ""); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.ui) != 1 {
		t.Fatal("signed tag did not queue terminal continuation")
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.subprocess) != 1 {
		t.Fatal("signed command was not sent to subprocess")
	}
	args := f.ui.subprocess[0].GetCmd().Args
	if !capturedTagArgsContain(args, "-m") || !capturedTagArgsContain(args, "--") || capturedTagArgsContain(args, "--force") {
		t.Fatalf("unexpected signed argv: %#v", args)
	}
	if _, err := os.Stat(filepath.Join(f.root, ".git", "refs", "tags", "signed")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("mock terminal modified tag: %v", err)
	}

	g := newCapturedTagFixture(t)
	namedCheckoutGit(t, g.root, "config", "tag.gpgSign", "true")
	g.flow.overrideGpg = false
	if err := g.submit(t, "signed-stale", ""); err != nil {
		t.Fatal(err)
	}
	if err := g.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := g.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	g.ui.state.generation++
	if err := g.ui.runUI(t); err == nil {
		t.Fatal("stale terminal callback accepted")
	}
	if len(g.ui.subprocess) != 0 {
		t.Fatal("stale terminal callback launched subprocess")
	}
}

func TestCapturedTagCreateSignedTerminalRejectsStaleNativeState(t *testing.T) {
	for _, change := range []string{"source", "head", "config", "destination"} {
		t.Run(change, func(t *testing.T) {
			f := newCapturedTagFixture(t)
			name := "signed-stale"
			if change == "destination" {
				namedCheckoutGit(t, f.root, "tag", "-a", name, "-m", "before", f.oid)
			}
			namedCheckoutGit(t, f.root, "config", "tag.gpgSign", "true")
			f.flow.overrideGpg = false
			callbackCalls := 0
			f.flow.onCreate = func() { callbackCalls++ }
			if err := f.submit(t, name, ""); err != nil {
				t.Fatal(err)
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if len(f.ui.confirms) > 0 {
				if err := f.ui.confirms[0].HandleConfirm(); err != nil {
					t.Fatal(err)
				}
			}
			if err := f.ui.runWorker(t); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "source":
				namedCheckoutGit(t, f.root, "update-ref", f.flow.target.SourceRef(), f.other)
			case "head":
				namedCheckoutGit(t, f.root, "checkout", "-q", "other")
			case "config":
				namedCheckoutGit(t, f.root, "config", "tag.gpgSign", "false")
			case "destination":
				namedCheckoutGit(t, f.root, "-c", "tag.gpgSign=false", "tag", "-fa", name, "-m", "concurrent", f.other)
			}
			postHead, postIndex, postRawHead, postFiles := capturedTagSnapshot(t, f.root)
			postSource := namedCheckoutGit(t, f.root, "rev-parse", f.flow.target.SourceRef())
			postTag := ""
			if change == "destination" {
				postTag = namedCheckoutGit(t, f.root, "rev-parse", "refs/tags/"+name)
			}
			if err := f.ui.runUI(t); err == nil {
				t.Fatal("stale signed terminal callback accepted")
			}
			if len(f.ui.subprocess) != 0 {
				t.Fatal("stale signed command launched subprocess")
			}
			if err := f.ui.runUI(t); err != nil {
				t.Fatal(err)
			}
			if len(f.ui.refreshes) != 1 || f.ui.refreshes[0].Then == nil {
				t.Fatal("stale terminal failure did not queue guarded refresh")
			}
			if err := f.ui.refreshes[0].Then(); err != nil {
				t.Fatal(err)
			}
			if callbackCalls != 0 {
				t.Fatal("stale signed operation ran onCreate callback")
			}
			assertCapturedTagSnapshot(t, f, postHead, postIndex, postRawHead, postFiles)
			if got := namedCheckoutGit(t, f.root, "rev-parse", f.flow.target.SourceRef()); got != postSource {
				t.Fatalf("stale signed flow changed source ref: %s != %s", got, postSource)
			}
			if change == "destination" && namedCheckoutGit(t, f.root, "rev-parse", "refs/tags/"+name) != postTag {
				t.Fatal("stale signed flow replaced concurrent tag")
			}
		})
	}
}

func TestCapturedTagCreateActualRepositoryReplacementAndRefreshGuard(t *testing.T) {
	f := newCapturedTagFixture(t)
	other := newNamedCheckoutFixture(t)
	firstHead, firstIndex, firstRawHead, firstFiles := capturedTagSnapshot(t, f.root)
	secondHead, secondIndex, secondRawHead, secondFiles := capturedTagSnapshot(t, other.root)
	if err := f.submit(t, "replacement", ""); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runUI(t); err != nil {
		t.Fatal(err)
	}
	if err := f.ui.runWorker(t); err != nil {
		t.Fatal(err)
	}
	if len(f.ui.ui) != 1 {
		t.Fatal("expected UI finish callback")
	}
	f.ui.git = other.ui.git
	if err := f.ui.runUI(t); err == nil {
		t.Fatal("finish accepted replacement backend")
	}
	if len(f.ui.refreshes) != 0 {
		t.Fatal("replacement backend caused refresh")
	}
	assertCapturedTagSnapshotAt(t, f.root, firstHead, firstIndex, firstRawHead, firstFiles)
	assertCapturedTagSnapshotAt(t, other.root, secondHead, secondIndex, secondRawHead, secondFiles)
	if namedCheckoutGit(t, f.root, "rev-parse", "refs/tags/replacement^{commit}") != firstHead {
		t.Fatal("tag was not created in captured repository")
	}
	if _, err := os.Stat(filepath.Join(other.root, ".git", "refs", "tags", "replacement")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("replacement repository received tag: %v", err)
	}
}
