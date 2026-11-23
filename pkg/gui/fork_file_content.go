package gui

import (
	"bytes"
	"io"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/tasks"
)

// Fork: renders a types.FileContentTask. It runs like a plain command render, except that
// the first bytes of the output are read before anything is shown, to tell whether the
// file is binary the way git does: a NUL byte among the first binaryDetectionSize bytes.
// A binary file's command is stopped at once, so a large file is never read through; a
// text file's output is streamed as it is, lazily, like a RunCommandTask's.
//
// The reading happens in the render's start function, which the task runs on its own
// goroutine, not on the UI thread.

// binaryDetectionSize is how much of a file is read to tell whether it is binary.
const binaryDetectionSize = 8000

func (gui *Gui) newFileContentTask(view *gocui.View, task *types.FileContentTask) error {
	return gui.newCmdTaskWithRender(view, task.Cmd, nil, gui.fileContentRender)
}

// fileContentRender runs the command as a plain render does, and decides between streaming
// its output and showing the binary note once it has the first bytes.
func (gui *Gui) fileContentRender(spec renderSpec) (startRender, onCloseRender) {
	plainStart, plainClose := gui.plainRender(spec)
	start := func() (tasks.Cmd, io.Reader) {
		cmd, output := plainStart()
		return sniffFileContent(cmd, output, plainClose, gui.c.Tr.BinaryFileNote)
	}
	return start, plainClose
}

// sniffFileContent reads at most binaryDetectionSize bytes of output. For text it returns
// the command with output that yields those bytes and then the rest. For binary it stops
// the command, closes the output (closeOutput), reaps the command, and returns a reader of
// note.
func sniffFileContent(cmd tasks.Cmd, output io.Reader, closeOutput func(), note string) (tasks.Cmd, io.Reader) {
	head := make([]byte, binaryDetectionSize)
	// A short read is the end of the output. A read error is not handled here: the stream
	// repeats it when the view reads on, where it is reported as any command's error is.
	n, _ := io.ReadFull(output, head)
	head = head[:n]

	if bytes.IndexByte(head, 0) < 0 {
		return cmd, io.MultiReader(bytes.NewReader(head), output)
	}

	_ = cmd.Terminate()
	closeOutput()
	_ = cmd.Wait()
	return reapedCmd{cmd: cmd}, strings.NewReader(note)
}

// reapedCmd is a command that has been stopped and reaped already, so that the task has
// nothing left to wait for or to terminate.
type reapedCmd struct {
	cmd tasks.Cmd
}

func (c reapedCmd) Wait() error      { return nil }
func (c reapedCmd) Terminate() error { return nil }
func (c reapedCmd) String() string   { return c.cmd.String() }
