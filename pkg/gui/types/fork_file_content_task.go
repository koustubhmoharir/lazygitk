package types

import "os/exec"

// Fork: FileContentTask shows a command's output as the content of a file: a binary file
// shows a note instead of its bytes, and text streams as a RunCommandTask's output does.
// Whether it is binary is decided by the render, off the UI thread, from the first bytes
// the command writes (see fork_file_content.go).
type FileContentTask struct {
	Cmd *exec.Cmd
}

func (t *FileContentTask) IsUpdateTask() {}

func NewFileContentTask(cmd *exec.Cmd) *FileContentTask {
	return &FileContentTask{Cmd: cmd}
}
