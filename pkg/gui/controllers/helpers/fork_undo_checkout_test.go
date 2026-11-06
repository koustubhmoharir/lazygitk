package helpers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

var undoCheckoutEnv = []string{"GIT_REFLOG_ACTION=[lazygit undo]"}

// undoCheckoutFixture is the named-checkout fixture with the repository state the
// captured-action helpers read, as the captured reset tests provide it.
func undoCheckoutFixture(t *testing.T) *namedCheckoutFixture {
	t.Helper()
	f := newNamedCheckoutFixture(t)
	ui := &capturedResetTestUI{namedCheckoutFakeIGuiCommon: f.ui, state: &capturedResetTestState{}}
	f.c = &HelperCommon{Common: common.NewDummyCommon(), IGuiCommon: ui}
	return f
}

// drainCheckoutFlow runs queued workers and UI callbacks until the checkout flow
// settles, and returns the first error a worker or callback reports.
func drainCheckoutFlow(t *testing.T, ui *namedCheckoutFakeIGuiCommon) error {
	t.Helper()
	for i := 0; len(ui.workers) > 0 || len(ui.ui) > 0; i++ {
		if i > 10 {
			t.Fatal("checkout flow did not settle")
		}
		var err error
		if len(ui.workers) > 0 {
			err = ui.runWorker(t)
		} else {
			err = ui.runUI(t)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func undoCheckoutLastReflog(t *testing.T, root string) string {
	t.Helper()
	return namedCheckoutGit(t, root, "log", "-g", "-1", "--format=%gs", "HEAD")
}

// TestUndoCheckoutToBranchUsesNamedCheckout covers undo of a checkout whose
// target is an existing local branch. The branch is at its own tip, so the named
// flow shows no review of its own. Uncommitted changes that Git can carry are
// kept, nothing is stashed, and the reflog entry carries the undo marker.
func TestUndoCheckoutToBranchUsesNamedCheckout(t *testing.T) {
	f := undoCheckoutFixture(t)
	enterCheckoutFixtureRepo(t, f)
	namedCheckoutWrite(t, f.root, "carry", "dirty carry\n")

	h := &RefsHelper{c: f.c}
	if err := h.CheckoutReflogTarget("other", undoCheckoutEnv); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutFlow(t, f.ui); err != nil {
		t.Fatal(err)
	}

	assert.Empty(t, f.ui.confirms, "named checkout of an existing branch must not add a second review")
	assert.Equal(t, "other", namedCheckoutGit(t, f.root, "rev-parse", "--abbrev-ref", "HEAD"))
	assert.Equal(t, "dirty carry", strings.TrimSpace(readFileForUndo(t, f.root, "carry")))
	assert.Empty(t, namedCheckoutGit(t, f.root, "stash", "list"), "nothing may be stashed")
	assert.True(t, strings.HasPrefix(undoCheckoutLastReflog(t, f.root), "[lazygit undo]"), "reflog entry must carry the undo marker")
}

// TestUndoCheckoutToDetachedCommitUsesCapturedCheckout covers undo of a checkout
// whose target is a commit ID rather than a branch. HEAD is detached at that
// commit, and the reflog entry carries the undo marker.
func TestUndoCheckoutToDetachedCommitUsesCapturedCheckout(t *testing.T) {
	f := undoCheckoutFixture(t)
	enterCheckoutFixtureRepo(t, f)

	h := &RefsHelper{c: f.c}
	if err := h.CheckoutReflogTarget(f.oid, undoCheckoutEnv); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutFlow(t, f.ui); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, f.oid, namedCheckoutGit(t, f.root, "rev-parse", "HEAD"))
	assert.Equal(t, "HEAD", namedCheckoutGit(t, f.root, "rev-parse", "--abbrev-ref", "HEAD"), "HEAD must be detached")
	assert.True(t, strings.HasPrefix(undoCheckoutLastReflog(t, f.root), "[lazygit undo]"), "reflog entry must carry the undo marker")
}

// TestUndoCheckoutAbbreviatedCommitResolvesToOneCommit covers a reflog that
// recorded an abbreviated commit ID. It resolves to the full commit, and only if
// the ID names exactly that commit.
func TestUndoCheckoutAbbreviatedCommitResolvesToOneCommit(t *testing.T) {
	f := undoCheckoutFixture(t)
	enterCheckoutFixtureRepo(t, f)

	h := &RefsHelper{c: f.c}
	if err := h.CheckoutReflogTarget(f.oid[:9], []string{"GIT_REFLOG_ACTION=[lazygit redo]"}); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutFlow(t, f.ui); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, f.oid, namedCheckoutGit(t, f.root, "rev-parse", "HEAD"))
	assert.True(t, strings.HasPrefix(undoCheckoutLastReflog(t, f.root), "[lazygit redo]"), "redo entry must carry the redo marker")
}

// TestUndoCheckoutRefusesUnknownTarget covers a target that is neither a local
// branch nor an ID of a commit. The refusal names it and changes nothing.
func TestUndoCheckoutRefusesUnknownTarget(t *testing.T) {
	for _, target := range []string{"no-such-branch", "abcdef1"} {
		t.Run(target, func(t *testing.T) {
			f := undoCheckoutFixture(t)
			enterCheckoutFixtureRepo(t, f)
			before := namedCheckoutGit(t, f.root, "rev-parse", "HEAD")

			h := &RefsHelper{c: f.c}
			err := h.CheckoutReflogTarget(target, undoCheckoutEnv)
			if err == nil {
				err = drainCheckoutFlow(t, f.ui)
			}
			if err == nil {
				t.Fatal("unknown target was accepted")
			}
			assert.Contains(t, err.Error(), target)
			assert.Equal(t, before, namedCheckoutGit(t, f.root, "rev-parse", "HEAD"))
		})
	}
}

// TestUndoCheckoutRefusesPendingMerge covers both routes while a merge is
// pending: neither changes HEAD or the merge state.
func TestUndoCheckoutRefusesPendingMerge(t *testing.T) {
	t.Run("branch", func(t *testing.T) {
		f := undoCheckoutFixture(t)
		enterCheckoutFixtureRepo(t, f)
		startPendingMerge(t, f)
		before := namedCheckoutGit(t, f.root, "rev-parse", "HEAD")

		h := &RefsHelper{c: f.c}
		err := h.CheckoutReflogTarget("other", undoCheckoutEnv)
		if err == nil {
			err = drainCheckoutFlow(t, f.ui)
		}
		if err == nil {
			t.Fatal("branch checkout accepted during a pending merge")
		}
		assert.Equal(t, before, namedCheckoutGit(t, f.root, "rev-parse", "HEAD"))
		assert.FileExists(t, filepath.Join(f.root, ".git", "MERGE_HEAD"))
	})

	t.Run("commit", func(t *testing.T) {
		f := undoCheckoutFixture(t)
		enterCheckoutFixtureRepo(t, f)
		startPendingMerge(t, f)
		before := namedCheckoutGit(t, f.root, "rev-parse", "HEAD")

		h := &RefsHelper{c: f.c}
		err := h.CheckoutReflogTarget(f.other, undoCheckoutEnv)
		if err == nil {
			err = drainCheckoutFlow(t, f.ui)
		}
		if err == nil {
			t.Fatal("detached checkout accepted during a pending merge")
		}
		assert.Equal(t, before, namedCheckoutGit(t, f.root, "rev-parse", "HEAD"))
		assert.FileExists(t, filepath.Join(f.root, ".git", "MERGE_HEAD"))
	})
}

// TestUndoCheckoutKeepsIgnoredFiles covers an ignored file in the way of the
// target branch. Git must refuse rather than overwrite it, and the file keeps
// its content.
func TestUndoCheckoutKeepsIgnoredFiles(t *testing.T) {
	f := undoCheckoutFixture(t)
	enterCheckoutFixtureRepo(t, f)
	namedCheckoutWrite(t, f.root, ".gitignore", "ignored.txt\n")
	namedCheckoutGit(t, f.root, "add", ".gitignore")
	namedCheckoutGit(t, f.root, "commit", "-qm", "ignore rule")
	namedCheckoutGit(t, f.root, "checkout", "-qb", "tracks-ignored")
	namedCheckoutWrite(t, f.root, "ignored.txt", "tracked on the branch\n")
	namedCheckoutGit(t, f.root, "add", "-f", "ignored.txt")
	namedCheckoutGit(t, f.root, "commit", "-qm", "track ignored file")
	namedCheckoutGit(t, f.root, "checkout", "-q", "main")
	namedCheckoutWrite(t, f.root, "ignored.txt", "local ignored content\n")
	before := namedCheckoutGit(t, f.root, "rev-parse", "HEAD")

	h := &RefsHelper{c: f.c}
	err := h.CheckoutReflogTarget("tracks-ignored", undoCheckoutEnv)
	if err == nil {
		err = drainCheckoutFlow(t, f.ui)
	}
	if err == nil {
		t.Fatal("checkout overwrote an ignored file")
	}
	assert.Contains(t, err.Error(), "ignored.txt", "Git must name the ignored file it refused to overwrite")
	assert.Equal(t, before, namedCheckoutGit(t, f.root, "rev-parse", "HEAD"))
	content, readErr := os.ReadFile(filepath.Join(f.root, "ignored.txt"))
	assert.NoError(t, readErr)
	assert.Equal(t, "local ignored content\n", string(content))
}

func readFileForUndo(t *testing.T, root, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
