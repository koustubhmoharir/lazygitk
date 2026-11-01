package helpers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

// fakeSwitchContext stands in for a context in the pane selection tests.
type fakeSwitchContext struct {
	types.Context
	kind     types.ContextKind
	viewName string
}

func (c *fakeSwitchContext) GetKind() types.ContextKind { return c.kind }
func (c *fakeSwitchContext) GetViewName() string        { return c.viewName }

func TestRepoSwitchPaneViewNames(t *testing.T) {
	contexts := []types.Context{
		&fakeSwitchContext{kind: types.GLOBAL_CONTEXT, viewName: ""},
		&fakeSwitchContext{kind: types.SIDE_CONTEXT, viewName: "status"},
		&fakeSwitchContext{kind: types.SIDE_CONTEXT, viewName: "files"},
		&fakeSwitchContext{kind: types.SIDE_CONTEXT, viewName: "localBranches"},
		&fakeSwitchContext{kind: types.SIDE_CONTEXT, viewName: "remotes"},
		&fakeSwitchContext{kind: types.SIDE_CONTEXT, viewName: "stash"},
		&fakeSwitchContext{kind: types.MAIN_CONTEXT, viewName: "main"},
		&fakeSwitchContext{kind: types.MAIN_CONTEXT, viewName: "secondary"},
		&fakeSwitchContext{kind: types.TEMPORARY_POPUP, viewName: "menu"},
		&fakeSwitchContext{kind: types.PERSISTENT_POPUP, viewName: "commitMessage"},
		&fakeSwitchContext{kind: types.EXTRAS_CONTEXT, viewName: "commandLog"},
		&fakeSwitchContext{kind: types.DISPLAY_CONTEXT, viewName: "options"},
		&fakeSwitchContext{kind: types.SIDE_CONTEXT, viewName: "files"},
	}

	assert.Equal(t,
		[]string{"status", "files", "localBranches", "remotes", "stash", "main", "secondary"},
		repoSwitchPaneViewNames(contexts))
}
