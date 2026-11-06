package git_commands

import (
	"strconv"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_config"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// Every patch action that edits its source commit, or the commit it moves to, must
// refuse an index outside the commits list with an error. Before the check, -1 reached
// commits[-1] or, for a destination before the first commit, silently became index 0.
func TestPatchSourceCommitIndexOutsideCommitsIsRejected(t *testing.T) {
	pool := &utils.StringPool{}
	commits := []*models.Commit{
		models.NewCommit(pool, models.NewCommitOpts{Hash: "1111111111111111111111111111111111111111"}),
		models.NewCommit(pool, models.NewCommitOpts{Hash: "2222222222222222222222222222222222222222"}),
	}
	deps := commonDeps{runner: oscommands.NewFakeRunner(t), gitConfig: git_config.NewFakeGitConfig(nil)}
	gitCommon := buildGitCommon(deps)
	rebase := buildRebaseCommands(deps)
	patch := NewPatchCommands(gitCommon, rebase, buildCommitCommands(deps), buildStatusCommands(deps), nil, nil)
	bad := []int{-1, len(commits)}

	for _, index := range bad {
		name := strconv.Itoa(index)
		t.Run("delete from commit "+name, func(t *testing.T) {
			assert.Error(t, patch.DeletePatchesFromCommit(commits, index))
		})
		t.Run("move into index "+name, func(t *testing.T) {
			assert.Error(t, patch.MovePatchIntoIndex(commits, index))
		})
		t.Run("move into new commit "+name, func(t *testing.T) {
			assert.Error(t, patch.PullPatchIntoNewCommit(commits, index, "summary", ""))
		})
		t.Run("move into new commit before "+name, func(t *testing.T) {
			assert.Error(t, patch.PullPatchIntoNewCommitBefore(commits, index, "summary", ""))
		})
		for _, destination := range []int{0, 1} {
			t.Run("move to selected commit from "+name+" to "+strconv.Itoa(destination), func(t *testing.T) {
				assert.Error(t, patch.MovePatchToSelectedCommit(commits, index, destination))
			})
			t.Run("move to selected commit from "+strconv.Itoa(destination)+" to "+name, func(t *testing.T) {
				assert.Error(t, patch.MovePatchToSelectedCommit(commits, destination, index))
			})
		}
	}
}
