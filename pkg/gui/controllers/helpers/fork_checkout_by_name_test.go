package helpers

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

// checkoutByNameUI runs the Then callback of a refresh on the queued UI
// callbacks, as the GUI does after the refresh completes.
type checkoutByNameUI struct {
	*selectedRefCheckoutUI
}

func (ui *checkoutByNameUI) Refresh(opts types.RefreshOptions) {
	if opts.Then != nil {
		ui.ui = append(ui.ui, opts.Then)
	}
}

// checkoutByNameFixture gives a repository with main at f.oid and a branch
// other at f.other, with the UI fake's state available for the captured repository.
func checkoutByNameFixture(t *testing.T) (*namedCheckoutFixture, *checkoutByNameUI) {
	t.Helper()
	f := newNamedCheckoutFixture(t)
	ui := &checkoutByNameUI{selectedRefCheckoutUI: &selectedRefCheckoutUI{namedCheckoutFakeIGuiCommon: f.ui}}
	f.c.IGuiCommon = ui
	return f, ui
}

// drainCheckoutByName runs queued workers and UI callbacks until none remain,
// and returns the last error a callback returned.
func drainCheckoutByName(t *testing.T, ui *checkoutByNameUI) error {
	t.Helper()
	var last error
	for len(ui.workers) > 0 || len(ui.ui) > 0 {
		if len(ui.workers) > 0 {
			if err := ui.runWorker(t); err != nil {
				last = err
			}
			continue
		}
		if err := ui.runUI(t); err != nil {
			last = err
		}
	}
	return last
}

// checkoutByNameHead returns the symbolic HEAD ref, empty when detached, and
// the commit HEAD points at.
func checkoutByNameHead(t *testing.T, f *namedCheckoutFixture) (ref, oid string) {
	t.Helper()
	out, _ := exec.Command("git", "-C", f.root, "symbolic-ref", "-q", "HEAD").Output()
	return strings.TrimSpace(string(out)), namedCheckoutGit(t, f.root, "rev-parse", "HEAD")
}

func TestCheckoutByNameTagChecksOutDetachedAtTagCommit(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	namedCheckoutGit(t, f.root, "tag", "v1", f.other)

	if err := (&RefsHelper{c: f.c}).CaptureCheckoutByName(nil).Confirm("v1"); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutByName(t, ui); err != nil {
		t.Fatal(err)
	}
	ref, head := checkoutByNameHead(t, f)
	assert.Equal(t, "", ref, "HEAD should be detached")
	assert.Equal(t, f.other, head)
}

func TestCheckoutByNameCommitChecksOutDetachedAtCommit(t *testing.T) {
	f, ui := checkoutByNameFixture(t)

	if err := (&RefsHelper{c: f.c}).CaptureCheckoutByName(nil).Confirm(f.other[:10]); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutByName(t, ui); err != nil {
		t.Fatal(err)
	}
	ref, head := checkoutByNameHead(t, f)
	assert.Equal(t, "", ref, "HEAD should be detached")
	assert.Equal(t, f.other, head)
}

func TestCheckoutByNameLocalBranchWinsOverSameNamedTag(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	namedCheckoutGit(t, f.root, "tag", "other-name", f.oid)
	namedCheckoutGit(t, f.root, "branch", "other-name", f.other)

	if err := (&RefsHelper{c: f.c}).CaptureCheckoutByName(nil).Confirm("other-name"); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutByName(t, ui); err != nil {
		t.Fatal(err)
	}
	// The branch is already at its own commit, so the named checkout attaches
	// it without a confirmation. A detached checkout here would mean the tag won.
	assert.Empty(t, ui.confirms)
	ref, _ := checkoutByNameHead(t, f)
	assert.Equal(t, "refs/heads/other-name", ref)
}

func TestCheckoutByNameRemoteBranchUsesRemoteFlow(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	namedCheckoutGit(t, f.root, "remote", "add", "origin", "https://example.invalid/repo.git")
	namedCheckoutGit(t, f.root, "update-ref", "refs/remotes/origin/topic", f.other)
	// CreateWithUpstream runs git in the working directory, as the app does
	// inside the repository, so the test enters the repository for this call.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(f.root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	ui.model.RemoteBranches = []*models.RemoteBranch{{Name: "topic", RemoteName: "origin"}}

	if err := (&RefsHelper{c: f.c}).CaptureCheckoutByName(nil).Confirm("origin/topic"); err != nil {
		t.Fatal(err)
	}
	// The remote flow creates the tracking branch first, which is how s c on a
	// remote branch starts; that creation is what this test observes.
	if err := drainCheckoutByName(t, ui); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, f.other, namedCheckoutGit(t, f.root, "rev-parse", "refs/heads/topic"))
	assert.Equal(t, "origin", namedCheckoutGit(t, f.root, "config", "branch.topic.remote"))
}

func TestCheckoutByNameRefusesRemoteBranchNotInModel(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	namedCheckoutGit(t, f.root, "update-ref", "refs/remotes/origin/topic", f.other)

	if err := (&RefsHelper{c: f.c}).CaptureCheckoutByName(nil).Confirm("origin/topic"); err != nil {
		t.Fatal(err)
	}
	err := drainCheckoutByName(t, ui)
	if err == nil || !strings.Contains(err.Error(), "origin/topic") {
		t.Fatalf("expected a refusal naming the remote branch, got %v", err)
	}
	assert.Equal(t, "", namedCheckoutGit(t, f.root, "branch", "--list", "topic"))
}

func TestCheckoutByNameAmbiguousRemoteAndTagRefuses(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	namedCheckoutGit(t, f.root, "update-ref", "refs/remotes/origin/topic", f.other)
	namedCheckoutGit(t, f.root, "tag", "origin/topic", f.other)

	if err := (&RefsHelper{c: f.c}).CaptureCheckoutByName(nil).Confirm("origin/topic"); err != nil {
		t.Fatal(err)
	}
	err := drainCheckoutByName(t, ui)
	if err == nil {
		t.Fatal("expected an ambiguity refusal")
	}
	assert.Contains(t, err.Error(), "refs/remotes/origin/topic")
	assert.Contains(t, err.Error(), "refs/tags/origin/topic")
	_, head := checkoutByNameHead(t, f)
	assert.Equal(t, f.oid, head)
	assert.Empty(t, ui.confirms)
}

func TestCheckoutByNameUnknownNameOffersCreationWithoutCheckout(t *testing.T) {
	f, ui := checkoutByNameFixture(t)

	if err := (&RefsHelper{c: f.c}).CaptureCheckoutByName(nil).Confirm("new-branch"); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutByName(t, ui); err != nil {
		t.Fatal(err)
	}
	if len(ui.confirms) != 1 {
		t.Fatalf("expected the create confirmation, got %d", len(ui.confirms))
	}
	assert.Equal(t, f.c.Tr.BranchNotFoundTitle, ui.confirms[0].Title)
	assert.Equal(t, f.c.Tr.BranchNotFoundPrompt+" new-branch?", ui.confirms[0].Prompt)
	ref, head := checkoutByNameHead(t, f)
	assert.Equal(t, "refs/heads/main", ref)
	assert.Equal(t, f.oid, head)
}

func TestCheckoutByNameTrackedPathNeverCheckedOut(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	namedCheckoutWrite(t, f.root, "tracked", "edited but not committed\n")

	// "tracked" is a file, not a ref, so it must not reach checkout as a path.
	if err := (&RefsHelper{c: f.c}).CaptureCheckoutByName(nil).Confirm("tracked"); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutByName(t, ui); err != nil {
		t.Fatal(err)
	}
	assert.Len(t, ui.confirms, 1, "a path name should only offer creation")
	ref, _ := checkoutByNameHead(t, f)
	assert.Equal(t, "refs/heads/main", ref)
	contents, err := os.ReadFile(f.root + "/tracked")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "edited but not committed\n", string(contents))
}

func TestCheckoutByNameRefusesMissingFullRefAndEmptyName(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	flow := (&RefsHelper{c: f.c}).CaptureCheckoutByName(nil)

	assert.Error(t, flow.Confirm("   "))
	assert.Empty(t, ui.workers)

	if err := flow.Confirm("refs/heads/missing"); err != nil {
		t.Fatal(err)
	}
	err := drainCheckoutByName(t, ui)
	if err == nil || !strings.Contains(err.Error(), "refs/heads/missing") {
		t.Fatalf("expected a missing-ref refusal, got %v", err)
	}
	assert.Empty(t, ui.confirms, "a full ref is never offered as a new branch name")
}

func TestCheckoutByNameRefusesChangedRepository(t *testing.T) {
	f, ui := checkoutByNameFixture(t)
	namedCheckoutGit(t, f.root, "tag", "v1", f.other)
	flow := (&RefsHelper{c: f.c}).CaptureCheckoutByName(nil)

	ui.state.generation++
	assert.Error(t, flow.Confirm("v1"))
	assert.Empty(t, ui.workers)
}
