package components

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/stretchr/testify/assert"
)

func TestEventuallyContent(t *testing.T) {
	gui := &fakeGuiDriver{}
	view := gocui.NewView("test", 0, 0, 20, 5, gocui.OutputNormal)
	view.SetContent("before")
	captures := 0
	testDriver := NewTestDriver(gui, nil, config.KeybindingConfig{}, 0)
	driver := &ViewDriver{
		context: "test view",
		getView: func() *gocui.View {
			assert.True(t, gui.onUIThread, "view reads must stay on the UI thread")
			captures++
			if captures == 2 {
				view.SetContent("after")
			}
			return view
		},
		t: testDriver,
	}

	driver.EventuallyContent(Contains("after"))

	assert.Empty(t, gui.failureMessage)
	assert.Equal(t, 2, gui.onUIThreadCallCount)
	assert.Equal(t, 2, captures)
	assert.False(t, gui.onUIThread)
}
