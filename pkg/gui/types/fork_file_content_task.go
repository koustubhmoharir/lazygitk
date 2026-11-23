package types

import (
	"os/exec"

	"github.com/jesseduffield/lazygit/pkg/config"
)

// Fork: FileContentTask shows a command's output as the content of a file: a binary file
// shows a note instead of its bytes, and text is shown by the file renderer it was made
// with (see fork_file_content.go), which is chosen when the task is made, on the UI thread.
type FileContentTask struct {
	Cmd *exec.Cmd
	// Path is the file's path, which the built-in renderer reads the language from.
	Path string
	// Renderer is the kind of file renderer that shows the text.
	Renderer config.FileRendererType
}

func (t *FileContentTask) IsUpdateTask() {}

func NewFileContentTask(cmd *exec.Cmd, path string, renderer config.FileRendererType) *FileContentTask {
	return &FileContentTask{Cmd: cmd, Path: path, Renderer: renderer}
}
