package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Fork: Reflog's diff-files route is key-only, so it adds no double-click and no
// focused main view diff source beside its own controller's.
func TestSwitchToDiffFilesControllerWithoutDoubleClickAddsNoMouseOrDiffSource(t *testing.T) {
	controller := (&SwitchToDiffFilesController{}).WithoutDoubleClick()
	assert.Nil(t, controller.GetOnDoubleClick())
	assert.Nil(t, controller.GetFocusedMainViewDiffSource())
}
