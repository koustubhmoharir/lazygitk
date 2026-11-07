package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// pendingStep registers a step whose callback and notice count their runs.
func pendingStep(helper *MergeAndRebaseHelper) (ran *int, abandoned *int) {
	ran = new(int)
	abandoned = new(int)
	helper.SetOnOperationFinished(func() error {
		*ran++
		return nil
	}, func() {
		*abandoned++
	})
	return ran, abandoned
}

func TestDropAbandonedOperationStepDropsStepEndedOutside(t *testing.T) {
	helper := &MergeAndRebaseHelper{}
	ran, abandoned := pendingStep(helper)

	helper.DropAbandonedOperationStep(helper.OperationFinishedEpoch())

	assert.Equal(t, 0, *ran)
	assert.Equal(t, 1, *abandoned)

	// The step is cleared, so a second drop does nothing.
	helper.DropAbandonedOperationStep(helper.OperationFinishedEpoch())
	assert.Equal(t, 0, *ran)
	assert.Equal(t, 1, *abandoned)
}

func TestDropAbandonedOperationStepKeepsStepRegisteredAfterRead(t *testing.T) {
	helper := &MergeAndRebaseHelper{}
	// The refresh loads its epoch before the step is registered.
	staleEpoch := helper.OperationFinishedEpoch()
	ran, abandoned := pendingStep(helper)

	helper.DropAbandonedOperationStep(staleEpoch)

	assert.Equal(t, 0, *ran)
	assert.Equal(t, 0, *abandoned)
}

func TestDropAbandonedOperationStepKeepsStepWhileOperationInFlight(t *testing.T) {
	helper := &MergeAndRebaseHelper{}
	ran, abandoned := pendingStep(helper)

	helper.operationsInFlight.Add(1)
	helper.DropAbandonedOperationStep(helper.OperationFinishedEpoch())
	assert.Equal(t, 0, *ran)
	assert.Equal(t, 0, *abandoned)

	helper.operationsInFlight.Add(-1)
	helper.DropAbandonedOperationStep(helper.OperationFinishedEpoch())
	assert.Equal(t, 0, *ran)
	assert.Equal(t, 1, *abandoned)
}
