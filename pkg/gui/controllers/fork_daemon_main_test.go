package controllers

import (
	"os"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/app/daemon"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: a test that starts an interactive rebase runs this test binary as the sequence
// editor (see oscommands.GetLazygitPath). The daemon invocation is handled here as the
// entry point handles it, so that the editor edits the todo file instead of starting
// the tests again.
func TestMain(m *testing.M) {
	if daemon.InDaemonMode() {
		daemon.Handle(utils.NewDummyLog())
		os.Exit(0)
	}
	os.Exit(m.Run())
}
