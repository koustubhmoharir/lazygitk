package context

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils/filelock"
	"github.com/stretchr/testify/assert"
)

type commitDraftGuiFake struct {
	types.IGuiCommon
	git    *commands.GitCommand
	toasts []string
}

func (f *commitDraftGuiFake) Git() *commands.GitCommand { return f.git }

func (f *commitDraftGuiFake) ErrorToast(message string) { f.toasts = append(f.toasts, message) }

// Up and Down with no error expected.
func stepOlder(t *testing.T, ctx *CommitMessageContext, text string) (string, bool) {
	t.Helper()
	got, ok, err := ctx.StepToOlderCommitDraft(text)
	assert.NoError(t, err)
	return got, ok
}

func stepNewer(t *testing.T, ctx *CommitMessageContext, text string) (string, bool) {
	t.Helper()
	got, ok, err := ctx.StepToNewerCommitDraft(text)
	assert.NoError(t, err)
	return got, ok
}

// Fork: a commit message context on a real temporary worktree, with the box
// open on the given text.
func newCommitDraftTestContext(t *testing.T, opened string, drafts ...string) (*CommitMessageContext, string) {
	t.Helper()
	dir := t.TempDir()
	assert.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	gui := &commitDraftGuiFake{git: &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths(dir)}}
	cc := &ContextCommon{Common: common.NewDummyCommon(), IGuiCommon: gui}
	ctx := &CommitMessageContext{c: cc, viewModel: &CommitMessageViewModel{}}
	ctx.resetCommitDraftSession()
	ctx.SetCommitDraftOpenedText(opened)
	setCommitDrafts(t, ctx, drafts)
	return ctx, dir
}

// Replaces the drafts on disk, as a change through the locked path does.
func setCommitDrafts(t *testing.T, ctx *CommitMessageContext, drafts []string) {
	t.Helper()
	assert.NoError(t, ctx.updateCommitDrafts(func([]string) []string { return drafts }))
}

func TestCommitDraftsStoredBesidePreservedMessage(t *testing.T) {
	ctx, dir := newCommitDraftTestContext(t, "")

	assert.Equal(t, filepath.Join(dir, ".git", "LAZYGITK_COMMIT_DRAFTS"), ctx.GetCommitDraftsPath())
	assert.NotEqual(t, ctx.GetPreservedMessagePath(), ctx.GetCommitDraftsPath())
}

func TestCommitDraftsRoundTripAndRemoval(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "")

	assert.Empty(t, ctx.loadCommitDrafts())

	setCommitDrafts(t, ctx, []string{"subject\nbody \"quoted\"", "older"})
	assert.Equal(t, []string{"subject\nbody \"quoted\"", "older"}, ctx.loadCommitDrafts())

	setCommitDrafts(t, ctx, nil)
	assert.Empty(t, ctx.loadCommitDrafts())
	_, err := os.Stat(ctx.GetCommitDraftsPath())
	assert.True(t, os.IsNotExist(err), "an empty list removes the file")

	assert.NoError(t, ctx.writeCommitDrafts(nil), "removing an absent file is not an error")
}

func TestCommitDraftsCorruptFileReadsAsEmpty(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "")
	assert.NoError(t, os.WriteFile(ctx.GetCommitDraftsPath(), []byte("not json"), 0o644))

	assert.Empty(t, ctx.loadCommitDrafts())
}

func TestAddCommitDraftMovesCopiesAndCaps(t *testing.T) {
	assert.Equal(t, []string{"b", "a", "c"}, addCommitDraft([]string{"a", "b", "c"}, "b"))

	drafts := []string{}
	for i := range maxCommitDrafts + 1 {
		drafts = addCommitDraft(drafts, fmt.Sprintf("draft %d", i))
	}
	assert.Len(t, drafts, maxCommitDrafts)
	assert.Equal(t, fmt.Sprintf("draft %d", maxCommitDrafts), drafts[0])
	assert.NotContains(t, drafts, "draft 0")
}

func TestReplaceCommitDraft(t *testing.T) {
	drafts := []string{"a", "b", "c"}

	assert.Equal(t, []string{"a", "edited", "c"}, replaceCommitDraft(drafts, "b", "edited"))
	assert.Equal(t, drafts, replaceCommitDraft(drafts, "b", "b"))
	assert.Equal(t, []string{"a", "c"}, replaceCommitDraft(drafts, "b", "  "), "clearing removes")
	assert.Equal(t, []string{"a", "c"}, replaceCommitDraft(drafts, "b", "c"), "a match removes")
}

func TestStepThroughCommitDrafts(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "newest", "middle", "oldest")

	_, ok := stepNewer(t, ctx, "typed")
	assert.False(t, ok, "Down from the typed text does nothing")

	for i, want := range []string{"newest", "middle", "oldest"} {
		text := "typed"
		if i > 0 {
			text = []string{"newest", "middle", "oldest"}[i-1]
		}
		got, ok := stepOlder(t, ctx, text)
		assert.True(t, ok)
		assert.Equal(t, want, got)
		index, total := ctx.commitDraftPosition()
		assert.Equal(t, i+1, index)
		assert.Equal(t, 3, total)
	}

	_, ok = stepOlder(t, ctx, "oldest")
	assert.False(t, ok, "Up from the oldest draft does nothing")

	got, _ := stepNewer(t, ctx, "oldest")
	assert.Equal(t, "middle", got)
	got, _ = stepNewer(t, ctx, "middle")
	assert.Equal(t, "newest", got)
	got, _ = stepNewer(t, ctx, "newest")
	assert.Equal(t, "typed", got)

	index, _ := ctx.commitDraftPosition()
	assert.Equal(t, 0, index)
}

func TestStepSkipsDraftEqualToTypedText(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "only", "only")

	_, ok := stepOlder(t, ctx, "only")
	assert.False(t, ok)
}

func TestStepKeepsEditsToShownDraft(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "newest", "oldest")

	_, _ = stepOlder(t, ctx, "typed")
	got, _ := stepOlder(t, ctx, "newest edited")
	assert.Equal(t, "oldest", got)
	assert.Equal(t, []string{"newest edited", "oldest"}, ctx.loadCommitDrafts())

	got, _ = stepNewer(t, ctx, "oldest")
	assert.Equal(t, "newest edited", got)
}

func TestClearingShownDraftRemovesIt(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "newest", "middle", "oldest")

	_, _ = stepOlder(t, ctx, "typed")
	got, _ := stepOlder(t, ctx, "")
	assert.Equal(t, "middle", got, "Up moves to the draft after the cleared one")
	assert.Equal(t, []string{"middle", "oldest"}, ctx.loadCommitDrafts())

	got, _ = stepOlder(t, ctx, "middle")
	assert.Equal(t, "oldest", got)
	got, _ = stepOlder(t, ctx, "")
	assert.Equal(t, "middle", got, "clearing the oldest moves to the one before it")

	got, _ = stepNewer(t, ctx, "")
	assert.Equal(t, "typed", got, "clearing the last one returns to the typed text")
	assert.Empty(t, ctx.loadCommitDrafts())
}

func TestCancelKeepsTypedText(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "older")

	assert.NoError(t, ctx.KeepCommitDraftsOnCancel("typed"))

	assert.Equal(t, []string{"typed", "older"}, ctx.loadCommitDrafts())
}

func TestCancelIgnoresBlankAndUnchangedText(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "opened", "older")

	assert.NoError(t, ctx.KeepCommitDraftsOnCancel("opened"))
	assert.Equal(t, []string{"older"}, ctx.loadCommitDrafts())

	assert.NoError(t, ctx.KeepCommitDraftsOnCancel(" \n"))
	assert.Equal(t, []string{"older"}, ctx.loadCommitDrafts())
}

func TestCancelWhileDraftShownKeepsBothTexts(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "newest", "oldest")

	_, _ = stepOlder(t, ctx, "typed")
	_, _ = stepOlder(t, ctx, "newest")
	assert.NoError(t, ctx.KeepCommitDraftsOnCancel("oldest edited"))

	assert.Equal(t, []string{"typed", "newest", "oldest edited"}, ctx.loadCommitDrafts())
}

func TestEditingTheDraftTheBoxOpenedWithReplacesIt(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "pending", "other", "pending")

	assert.NoError(t, ctx.KeepCommitDraftsOnCancel("pending edited"))

	assert.Equal(t, []string{"pending edited", "other"}, ctx.loadCommitDrafts())
}

func TestSubmitUsesUpTheShownDraftAndKeepsTypedText(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "newest", "oldest")

	_, _ = stepOlder(t, ctx, "typed")
	used := ctx.CommitDraftsUsedBy("newest edited")
	ctx.EndCommitDraftSession() // the box closes before the drafts are updated
	assert.NoError(t, used())

	assert.Equal(t, []string{"typed", "oldest"}, ctx.loadCommitDrafts())
}

func TestSubmitUsesUpMatchingAndOpenedDrafts(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "pending", "same", "pending", "other")
	assert.NoError(t, ctx.CommitDraftsUsedBy("same")())
	assert.Equal(t, []string{"other"}, ctx.loadCommitDrafts())

	ctx, _ = newCommitDraftTestContext(t, "pending", "pending", "other")
	assert.NoError(t, ctx.CommitDraftsUsedBy("pending edited")())
	assert.Equal(t, []string{"other"}, ctx.loadCommitDrafts())
}

func TestSwitchToEditorKeepsOnlyTypedText(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "older")

	assert.NoError(t, ctx.KeepTypedCommitDraftOnSwitchToEditor())
	assert.Equal(t, []string{"older"}, ctx.loadCommitDrafts(), "nothing was replaced")

	_, _ = stepOlder(t, ctx, "typed")
	assert.NoError(t, ctx.KeepTypedCommitDraftOnSwitchToEditor())
	assert.Equal(t, []string{"typed", "older"}, ctx.loadCommitDrafts())
}

// Two instances of lazygitk share one drafts file. Each adds drafts while the
// other does, and every draft must survive.
func TestDraftsAddedByTwoInstancesAreAllKept(t *testing.T) {
	first, _ := newCommitDraftTestContext(t, "")
	second := &CommitMessageContext{c: first.c, viewModel: &CommitMessageViewModel{}}
	const perInstance = 10 // with the cap of maxCommitDrafts, all of them fit

	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i, ctx := range []*CommitMessageContext{first, second} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range perInstance {
				message := fmt.Sprintf("instance %d draft %d", i, n)
				if err := ctx.updateCommitDrafts(func(drafts []string) []string {
					return addCommitDraft(drafts, message)
				}); err != nil {
					errs[i] = err
					return
				}
			}
		}()
	}
	wg.Wait()
	assert.NoError(t, errs[0])
	assert.NoError(t, errs[1])

	drafts := first.loadCommitDrafts()
	assert.Len(t, drafts, 2*perInstance)
	for i := range 2 {
		for n := range perInstance {
			assert.Contains(t, drafts, fmt.Sprintf("instance %d draft %d", i, n))
		}
	}
}

func TestMalformedDraftsFileIsMovedAsideNotLost(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "")
	assert.NoError(t, os.WriteFile(ctx.GetCommitDraftsPath(), []byte("not json"), 0o644))

	assert.NoError(t, ctx.updateCommitDrafts(func(drafts []string) []string {
		return addCommitDraft(drafts, "fresh")
	}))

	assert.Equal(t, []string{"fresh"}, ctx.loadCommitDrafts())
	asideFiles, err := filepath.Glob(ctx.GetCommitDraftsPath() + ".corrupt-*")
	assert.NoError(t, err)
	if assert.Len(t, asideFiles, 1) {
		content, err := os.ReadFile(asideFiles[0])
		assert.NoError(t, err)
		assert.Equal(t, "not json", string(content))
		// The user is told once, with the path the file was moved to.
		toasts := ctx.c.IGuiCommon.(*commitDraftGuiFake).toasts
		assert.Equal(t, []string{"The commit drafts file could not be read, so it was moved to " + asideFiles[0] + ". The drafts start empty."}, toasts)
	}
}

// A drafts location that cannot be read or written makes the save fail, and the
// error comes back so that the caller can keep the message box open.
func TestDraftSaveFailureIsReturned(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "older")
	// A directory where the file should be is unusable for the drafts.
	assert.NoError(t, os.Remove(ctx.GetCommitDraftsPath()))
	assert.NoError(t, os.Mkdir(ctx.GetCommitDraftsPath(), 0o755))

	assert.Error(t, ctx.KeepCommitDraftsOnCancel("typed"))
	assert.Error(t, ctx.CommitDraftsUsedBy("typed")())
	assert.Error(t, ctx.updateCommitDrafts(func(drafts []string) []string { return drafts }))

	info, err := os.Stat(ctx.GetCommitDraftsPath())
	assert.NoError(t, err)
	assert.True(t, info.IsDir(), "the failed saves leave the location as it was")
}

// If another instance holds the drafts lock for too long, the change fails and
// changes nothing.
func TestDraftChangeFailsWhileLockIsHeld(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "older")
	held, err := filelock.Acquire(ctx.commitDraftsLockPath(), time.Second)
	assert.NoError(t, err)
	defer func() { assert.NoError(t, held.Release()) }()

	assert.Error(t, ctx.KeepCommitDraftsOnCancel("typed"))
	assert.Equal(t, []string{"older"}, ctx.loadCommitDrafts())
}

// Up and Down step only if the edits to the shown draft are saved. If they are
// not, the text stays as it is, and the drafts are unchanged.
func TestStepFailureKeepsTextAndDraftsUnchanged(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "newest", "oldest")
	_, _ = stepOlder(t, ctx, "typed")
	assert.Equal(t, 0, ctx.viewModel.drafts.index, "the newest draft is shown")

	// The drafts location becomes unusable, so saving the shown draft fails.
	assert.NoError(t, os.Remove(ctx.GetCommitDraftsPath()))
	assert.NoError(t, os.Mkdir(ctx.GetCommitDraftsPath(), 0o755))

	_, ok, err := ctx.StepToOlderCommitDraft("newest edited")
	assert.Error(t, err)
	assert.False(t, ok)
	_, ok, err = ctx.StepToNewerCommitDraft("newest edited")
	assert.Error(t, err)
	assert.False(t, ok)

	assert.Equal(t, 0, ctx.viewModel.drafts.index, "nothing moved")
	assert.Equal(t, []string{"newest", "oldest"}, ctx.viewModel.drafts.stops)
	assert.Equal(t, "typed", ctx.viewModel.drafts.typed)
}
