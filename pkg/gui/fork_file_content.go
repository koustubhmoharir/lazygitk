package gui

import (
	"bytes"
	"io"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/tasks"
)

// Fork: renders a types.FileContentTask. It runs like a plain command render, except that
// the first bytes of the output are read before anything is shown, to tell whether the
// file is binary the way git does: a NUL byte among the first binaryDetectionSize bytes.
// A binary file's command is stopped at once, so a large file is never read through; a
// text file's output is streamed as it is, lazily, like a RunCommandTask's, unless the
// built-in renderer shows it (see fork_file_highlight.go).
//
// The reading happens in the render's start function, which the task runs on its own
// goroutine, not on the UI thread.

// binaryDetectionSize is how much of a file is read to tell whether it is binary.
const binaryDetectionSize = 8000

func (gui *Gui) newFileContentTask(view *gocui.View, task *types.FileContentTask) error {
	// The terminal's appearance is read here, on the UI thread, as the task is made.
	appearance := fileHighlightAppearance{
		lightBackground: gui.terminalHasLightBackground(),
		trueColor:       gui.g.TrueColor(),
	}
	run := func(spec renderSpec) (startRender, onCloseRender) {
		return gui.fileContentRender(spec, task, appearance)
	}
	return gui.newCmdTaskWithRender(view, task.Cmd, nil, run)
}

// fileContentRender runs the command as a plain render does, and decides between streaming
// its output and showing the binary note once it has the first bytes. A text file goes to
// the built-in renderer when the renderer is that one, and to a stdin filter's command when
// the renderer is one (see fork_file_render_cmd.go); any other renderer shows it as it is.
func (gui *Gui) fileContentRender(spec renderSpec, task *types.FileContentTask, appearance fileHighlightAppearance) (startRender, onCloseRender) {
	if task.Renderer == config.FileRendererType_StdinFilter {
		return gui.commandFileRender(spec, task.Command, gui.c.Tr.BinaryFileNote)
	}

	plainStart, plainClose := gui.plainRender(spec)
	start := func() (tasks.Cmd, io.Reader) {
		cmd, output := plainStart()
		cmd, output, isText := sniffFileContent(cmd, output, plainClose, gui.c.Tr.BinaryFileNote)
		if !isText || task.Renderer != config.FileRendererType_Builtin {
			return cmd, output
		}
		return gui.highlightFileStream(spec, cmd, task.Path, output, appearance)
	}
	return start, plainClose
}

// highlightFileStream returns the built-in renderer's output for a text file whose output
// is output. A file over maxHighlightedFileSize, or one that can't be read whole, is
// streamed as plain text; a large one says so in the view's subtitle.
func (gui *Gui) highlightFileStream(spec renderSpec, cmd tasks.Cmd, path string, output io.Reader, appearance fileHighlightAppearance) (tasks.Cmd, io.Reader) {
	data, tooLarge, err := readForHighlight(output)
	if tooLarge {
		gui.showFileNotHighlighted(spec)
	}
	if tooLarge || err != nil {
		// A read error is left to repeat in the stream, as it is for plain text.
		return cmd, io.MultiReader(bytes.NewReader(data), output)
	}
	return cmd, strings.NewReader(highlightFileContent(path, data, appearance))
}

// showFileNotHighlighted says in the view's subtitle that the file isn't highlighted. The
// subtitle is set on the UI thread, and only if no newer render has been asked for since
// this one, which would set the subtitle for its own content.
func (gui *Gui) showFileNotHighlighted(spec renderSpec) {
	err := gui.g.OnUIThreadAndWaitBackground(func() {
		if spec.isSuperseded != nil && spec.isSuperseded() {
			return
		}
		spec.view.Subtitle = gui.c.Tr.FileContentNotHighlightedLarge
	})
	if err != nil {
		gui.c.Log.Error(err)
	}
}

// sniffFileContent reads at most binaryDetectionSize bytes of output. For text it returns
// the command with output that yields those bytes and then the rest, and true. For binary
// it stops the command, closes the output (closeOutput), reaps the command, and returns a
// reader of note, and false.
func sniffFileContent(cmd tasks.Cmd, output io.Reader, closeOutput func(), note string) (tasks.Cmd, io.Reader, bool) {
	head := make([]byte, binaryDetectionSize)
	// A short read is the end of the output. A read error is not handled here: the stream
	// repeats it when the view reads on, where it is reported as any command's error is.
	n, _ := io.ReadFull(output, head)
	head = head[:n]

	if bytes.IndexByte(head, 0) < 0 {
		return cmd, io.MultiReader(bytes.NewReader(head), output), true
	}

	_ = cmd.Terminate()
	closeOutput()
	_ = cmd.Wait()
	return reapedCmd{cmd: cmd}, strings.NewReader(note), false
}

// reapedCmd is a command that has been stopped and reaped already, so that the task has
// nothing left to wait for or to terminate.
type reapedCmd struct {
	cmd tasks.Cmd
}

func (c reapedCmd) Wait() error      { return nil }
func (c reapedCmd) Terminate() error { return nil }
func (c reapedCmd) String() string   { return c.cmd.String() }
