package context

import (
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

// Fork: a commit message context on a real temporary worktree.
func newCommitDraftTestContext(t *testing.T) (*CommitMessageContext, string) {
	t.Helper()
	dir := t.TempDir()
	assert.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	gui := &commitDraftGuiFake{git: &commands.GitCommand{RepoPaths: git_commands.MockRepoPaths(dir)}}
	cc := &ContextCommon{Common: common.NewDummyCommon(), IGuiCommon: gui}
	return &CommitMessageContext{c: cc, viewModel: &CommitMessageViewModel{}}, dir
}

func TestCommitDraftStoredBesidePreservedMessage(t *testing.T) {
	ctx, dir := newCommitDraftTestContext(t)

	assert.Equal(t, filepath.Join(dir, ".git", "LAZYGITK_COMMIT_DRAFT"), ctx.GetCommitDraftPath())
	assert.NotEqual(t, ctx.GetPreservedMessagePath(), ctx.GetCommitDraftPath())
}

func TestCommitDraftRoundTripAndRemoval(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t)

	assert.Equal(t, "", ctx.GetCommitDraftAndLogError())

	ctx.SetCommitDraftAndLogError("subject\n\nbody line")
	assert.Equal(t, "subject\n\nbody line", ctx.GetCommitDraftAndLogError())

	ctx.SetCommitDraftAndLogError("")
	assert.Equal(t, "", ctx.GetCommitDraftAndLogError())
	_, err := os.Stat(ctx.GetCommitDraftPath())
	assert.True(t, os.IsNotExist(err), "setting an empty draft removes the file")
}

func TestCommitDraftRemovalWhenAbsentIsNotAnError(t *testing.T) {
	ctx, _ := newCommitDraftTestContext(t)

	assert.NoError(t, ctx.setCommitDraft(""))
}
