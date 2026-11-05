package context

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

type commitDraftGuiFake struct {
	types.IGuiCommon
	git *commands.GitCommand
}

func (f *commitDraftGuiFake) Git() *commands.GitCommand { return f.git }

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
	ctx.saveCommitDrafts(drafts)
	return ctx, dir
}

func TestCommitDraftsStoredBesidePreservedMessage(t *testing.T) {
	ctx, dir := newCommitDraftTestContext(t, "")

	assert.Equal(t, filepath.Join(dir, ".git", "LAZYGITK_COMMIT_DRAFTS"), ctx.GetCommitDraftsPath())
	assert.NotEqual(t, ctx.GetPreservedMessagePath(), ctx.GetCommitDraftsPath())
}

func TestCommitDraftsRoundTripAndRemoval(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "")

	assert.Empty(t, ctx.loadCommitDrafts())

	ctx.saveCommitDrafts([]string{"subject\nbody \"quoted\"", "older"})
	assert.Equal(t, []string{"subject\nbody \"quoted\"", "older"}, ctx.loadCommitDrafts())

	ctx.saveCommitDrafts(nil)
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

	_, ok := ctx.StepToNewerCommitDraft("typed")
	assert.False(t, ok, "Down from the typed text does nothing")

	for i, want := range []string{"newest", "middle", "oldest"} {
		text := "typed"
		if i > 0 {
			text = []string{"newest", "middle", "oldest"}[i-1]
		}
		got, ok := ctx.StepToOlderCommitDraft(text)
		assert.True(t, ok)
		assert.Equal(t, want, got)
		index, total := ctx.commitDraftPosition()
		assert.Equal(t, i+1, index)
		assert.Equal(t, 3, total)
	}

	_, ok = ctx.StepToOlderCommitDraft("oldest")
	assert.False(t, ok, "Up from the oldest draft does nothing")

	got, _ := ctx.StepToNewerCommitDraft("oldest")
	assert.Equal(t, "middle", got)
	got, _ = ctx.StepToNewerCommitDraft("middle")
	assert.Equal(t, "newest", got)
	got, _ = ctx.StepToNewerCommitDraft("newest")
	assert.Equal(t, "typed", got)

	index, _ := ctx.commitDraftPosition()
	assert.Equal(t, 0, index)
}

func TestStepSkipsDraftEqualToTypedText(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "only", "only")

	_, ok := ctx.StepToOlderCommitDraft("only")
	assert.False(t, ok)
}

func TestStepKeepsEditsToShownDraft(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "newest", "oldest")

	_, _ = ctx.StepToOlderCommitDraft("typed")
	got, _ := ctx.StepToOlderCommitDraft("newest edited")
	assert.Equal(t, "oldest", got)
	assert.Equal(t, []string{"newest edited", "oldest"}, ctx.loadCommitDrafts())

	got, _ = ctx.StepToNewerCommitDraft("oldest")
	assert.Equal(t, "newest edited", got)
}

func TestClearingShownDraftRemovesIt(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "newest", "middle", "oldest")

	_, _ = ctx.StepToOlderCommitDraft("typed")
	got, _ := ctx.StepToOlderCommitDraft("")
	assert.Equal(t, "middle", got, "Up moves to the draft after the cleared one")
	assert.Equal(t, []string{"middle", "oldest"}, ctx.loadCommitDrafts())

	got, _ = ctx.StepToOlderCommitDraft("middle")
	assert.Equal(t, "oldest", got)
	got, _ = ctx.StepToOlderCommitDraft("")
	assert.Equal(t, "middle", got, "clearing the oldest moves to the one before it")

	got, _ = ctx.StepToNewerCommitDraft("")
	assert.Equal(t, "typed", got, "clearing the last one returns to the typed text")
	assert.Empty(t, ctx.loadCommitDrafts())
}

func TestCancelKeepsTypedText(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "older")

	ctx.KeepCommitDraftsOnCancel("typed")

	assert.Equal(t, []string{"typed", "older"}, ctx.loadCommitDrafts())
}

func TestCancelIgnoresBlankAndUnchangedText(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "opened", "older")

	ctx.KeepCommitDraftsOnCancel("opened")
	assert.Equal(t, []string{"older"}, ctx.loadCommitDrafts())

	ctx.KeepCommitDraftsOnCancel(" \n")
	assert.Equal(t, []string{"older"}, ctx.loadCommitDrafts())
}

func TestCancelWhileDraftShownKeepsBothTexts(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "newest", "oldest")

	_, _ = ctx.StepToOlderCommitDraft("typed")
	_, _ = ctx.StepToOlderCommitDraft("newest")
	ctx.KeepCommitDraftsOnCancel("oldest edited")

	assert.Equal(t, []string{"typed", "newest", "oldest edited"}, ctx.loadCommitDrafts())
}

func TestEditingTheDraftTheBoxOpenedWithReplacesIt(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "pending", "other", "pending")

	ctx.KeepCommitDraftsOnCancel("pending edited")

	assert.Equal(t, []string{"pending edited", "other"}, ctx.loadCommitDrafts())
}

func TestSubmitUsesUpTheShownDraftAndKeepsTypedText(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "newest", "oldest")

	_, _ = ctx.StepToOlderCommitDraft("typed")
	used := ctx.CommitDraftsUsedBy("newest edited")
	ctx.EndCommitDraftSession() // the box closes before the drafts are updated
	used()

	assert.Equal(t, []string{"typed", "oldest"}, ctx.loadCommitDrafts())
}

func TestSubmitUsesUpMatchingAndOpenedDrafts(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "pending", "same", "pending", "other")
	ctx.CommitDraftsUsedBy("same")()
	assert.Equal(t, []string{"other"}, ctx.loadCommitDrafts())

	ctx, _ = newCommitDraftTestContext(t, "pending", "pending", "other")
	ctx.CommitDraftsUsedBy("pending edited")()
	assert.Equal(t, []string{"other"}, ctx.loadCommitDrafts())
}

func TestSwitchToEditorKeepsOnlyTypedText(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t, "", "older")

	ctx.KeepTypedCommitDraftOnSwitchToEditor()
	assert.Equal(t, []string{"older"}, ctx.loadCommitDrafts(), "nothing was replaced")

	_, _ = ctx.StepToOlderCommitDraft("typed")
	ctx.KeepTypedCommitDraftOnSwitchToEditor()
	assert.Equal(t, []string{"typed", "older"}, ctx.loadCommitDrafts())
}
