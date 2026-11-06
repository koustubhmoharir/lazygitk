package controllers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

const patchSourceRefusal = "The commit this patch was built from is not in the Commits list. Clear the filter, or reset the patch."

// patchSourceFixture returns the patch menu action for a patch built from ids[1], a
// commit that is not in the Commits list, and for one built from ids[0], which is.
func patchSourceFixture(t *testing.T, sourceInList bool) (*CustomPatchOptionsMenuAction, *selectedRebaseFlowUI, []string) {
	t.Helper()
	basic, ui, _, ids := selectedRebaseFlowFixture(t)
	ui.model.Commits = []*models.Commit{historicalTestCommit(ids[2]), historicalTestCommit(ids[0])}
	source := ids[1]
	if sourceInList {
		source = ids[0]
	}
	ui.git.Patch.PatchBuilder.To = source
	ui.git.Patch.PatchBuilder.CanRebase = true
	return &CustomPatchOptionsMenuAction{c: basic.c}, ui, ids
}

// Each source-editing action refuses, before it queues any work, when the patch's source
// commit is not in the Commits list. The index is -1 there, and the backend would panic.
func TestPatchSourceEditingRefusesMissingSource(t *testing.T) {
	actions := map[string]func(*CustomPatchOptionsMenuAction) error{
		"remove from source": (*CustomPatchOptionsMenuAction).handleDeletePatchFromCommit,
		"move to index":      (*CustomPatchOptionsMenuAction).handleMovePatchIntoWorkingTree,
		"move to new commit": (*CustomPatchOptionsMenuAction).handlePullPatchIntoNewCommit,
		"move to new before": (*CustomPatchOptionsMenuAction).handlePullPatchIntoNewCommitBefore,
		"move to selected":   (*CustomPatchOptionsMenuAction).handleMovePatchToSelectedCommit,
	}
	for name, action := range actions {
		t.Run(name, func(t *testing.T) {
			controller, ui, _ := patchSourceFixture(t, false)
			err := action(controller)
			if assert.Error(t, err) {
				assert.Equal(t, patchSourceRefusal, err.Error())
			}
			assert.Empty(t, ui.workers, "no rebase may be started")
		})
	}
}

func TestPatchSourceDisabledReasonOnlyWhenSourceIsMissing(t *testing.T) {
	controller, _, _ := patchSourceFixture(t, false)
	reason := controller.patchSourceDisabledReason()
	if assert.NotNil(t, reason) {
		assert.Equal(t, patchSourceRefusal, reason.Text)
	}

	controller, _, _ = patchSourceFixture(t, true)
	assert.Nil(t, controller.patchSourceDisabledReason())
}
