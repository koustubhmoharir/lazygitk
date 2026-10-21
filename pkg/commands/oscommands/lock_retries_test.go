package oscommands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLockRetriesDisabledDefaultsToFalse(t *testing.T) {
	assert.False(t, newLockRetryTestCmd().LockRetriesDisabled())
}

func TestLockRetriesDisabledSurvivesClone(t *testing.T) {
	cmd := newLockRetryTestCmd().DisableLockRetries()

	assert.True(t, cmd.LockRetriesDisabled())
	assert.True(t, cmd.Clone().LockRetriesDisabled())
}

func TestLockRetriesDisabledIsIsolatedBetweenCommands(t *testing.T) {
	cmd := newLockRetryTestCmd()
	clone := cmd.Clone()

	clone.DisableLockRetries()

	assert.False(t, cmd.LockRetriesDisabled())
	assert.True(t, clone.LockRetriesDisabled())
}

func newLockRetryTestCmd() *CmdObj {
	return NewDummyCmdObjBuilder(nil).New([]string{"git", "status"})
}
