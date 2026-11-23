package gui

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/tasks"
)

// Fork: a stdinFilter file renderer (see fork_file_renderers.go) shows a text file's content
// through its command. git's output, which is the content, goes to the command's stdin, and
// the command's stdout is what the view shows. The two processes are one render: fileRenderCmd
// stops and reaps both, and a failure of either is the render's failure.

// commandStderr collects what the processes of a render write to stderr. Both write to it
// from their own goroutines, so it is locked.
type commandStderr struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *commandStderr) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *commandStderr) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// fileRenderCmd is the git command and the renderer's command of one render, as a tasks.Cmd.
// Terminate stops both; Wait waits for both, once, and reports the render's failure.
type fileRenderCmd struct {
	git      *exec.Cmd
	gitOut   io.Closer // our end of git's output, which the renderer's stdin is fed from
	renderer *exec.Cmd
	stderr   *commandStderr

	// stopped is set by Terminate. A render that was stopped has no failure to show: its
	// processes exited because they were asked to.
	stopped  atomic.Bool
	waitOnce sync.Once
	failure  error
}

func (c *fileRenderCmd) String() string {
	return c.git.String() + " | " + c.renderer.String()
}

// Terminate asks both processes to stop, without waiting for them. A process that has
// already exited is not an error.
func (c *fileRenderCmd) Terminate() error {
	c.stopped.Store(true)
	var firstErr error
	for _, proc := range []*os.Process{c.git.Process, c.renderer.Process} {
		err := oscommands.TerminateProcessGracefully(proc)
		if err != nil && !errors.Is(err, os.ErrProcessDone) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Wait waits until both processes have exited, and returns the render's failure, or nil
// when it was stopped.
func (c *fileRenderCmd) Wait() error {
	c.waitOnce.Do(func() {
		rendererErr := c.renderer.Wait()
		// The renderer no longer reads git's output, so git's next write has to fail, or it
		// would block on a full pipe. Closing our end of the pipe makes it fail.
		_ = c.gitOut.Close()
		gitErr := c.git.Wait()
		if !c.stopped.Load() {
			c.failure = renderFailure(rendererErr, gitErr)
		}
	})
	return c.failure
}

// renderFailure is the failure of a render whose renderer and git have exited with the given
// errors. The renderer's failure always counts. git's counts only when git exited with a
// status of its own: when the renderer stopped reading early, git is stopped by the closed
// pipe, which is a consequence of the renderer's exit and not a failure of its own.
func renderFailure(rendererErr, gitErr error) error {
	if rendererErr != nil {
		return rendererErr
	}
	var exitErr *exec.ExitError
	if errors.As(gitErr, &exitErr) && exitErr.ExitCode() > 0 {
		return gitErr
	}
	return nil
}

// renderFailureText is what a failed render shows after its output: the commands' stderr if
// they wrote any, otherwise the error.
func renderFailureText(err error, stderr *commandStderr) string {
	if text := strings.TrimRight(stderr.String(), " \t\r\n"); text != "" {
		return text + "\n"
	}
	return err.Error() + "\n"
}

// failureReader shows the failure of a render once its output has ended. It is read after
// the renderer's output, so the failure is only known, and waited for, at the end.
type failureReader struct {
	cmd  *fileRenderCmd
	text *strings.Reader
}

func (r *failureReader) Read(p []byte) (int, error) {
	if r.text == nil {
		r.text = strings.NewReader("")
		if err := r.cmd.Wait(); err != nil {
			r.text = strings.NewReader(renderFailureText(err, r.cmd.stderr))
		}
	}
	return r.text.Read(p)
}

// commandFileRender renders the content of a text file through the command of a stdinFilter
// file renderer. The content is read by the start function, which gets git's output going
// before the renderer starts; the binary check (see sniffFileContent) comes first, so a
// binary file never reaches the command.
func (gui *Gui) commandFileRender(spec renderSpec, command string, binaryNote string) (startRender, onCloseRender) {
	var gitOut, renderOut io.ReadCloser
	start := func() (tasks.Cmd, io.Reader) {
		// The view wraps to this width; see plainRender.
		spec.view.SetContentWidth(spec.width)

		git := spec.cmd
		stderr := &commandStderr{}
		git.Stderr = stderr
		out, err := git.StdoutPipe()
		if err != nil {
			gui.c.Log.Error(err)
			return reapedCmd{cmd: tasks.ExecCmd{Cmd: git}}, strings.NewReader(renderFailureText(err, stderr))
		}
		gitOut = out
		if err := git.Start(); err != nil {
			return abandonFileRender(git, gitOut, stderr, err)
		}

		cmd, content, isText := sniffFileContent(tasks.ExecCmd{Cmd: git}, gitOut, func() { _ = gitOut.Close() }, binaryNote)
		if !isText {
			return cmd, content
		}

		// The renderer runs as the diff renderer's stdin filter does: in a plain shell, with
		// git's environment (see pipedRender).
		cmdObj := gui.os.Cmd.NewShell(command, "").SetEnviron(git.Env).DontLog()
		gui.os.LogCommand(cmdObj.ToString(), true)
		renderer := cmdObj.GetCmd()
		renderer.Stdin = content
		renderer.Stderr = stderr
		out, err = renderer.StdoutPipe()
		if err != nil {
			return abandonFileRender(git, gitOut, stderr, err)
		}
		renderOut = out
		if err := renderer.Start(); err != nil {
			return abandonFileRender(git, gitOut, stderr, err)
		}

		fileRender := &fileRenderCmd{git: git, gitOut: gitOut, renderer: renderer, stderr: stderr}
		return fileRender, io.MultiReader(renderOut, &failureReader{cmd: fileRender})
	}

	onClose := func() {
		// Closing the renderer's output makes its next write fail, so it exits; that in
		// turn stops git, as its output has no reader left (see fileRenderCmd.Wait).
		if renderOut != nil {
			renderOut.Close()
			renderOut = nil
		}
		if gitOut != nil {
			gitOut.Close()
			gitOut = nil
		}
	}

	return start, onClose
}

// abandonFileRender stops git, which has started and whose output nothing will read, and
// reaps it, for a render whose renderer could not be started. The error is what the view shows.
func abandonFileRender(git *exec.Cmd, gitOut io.Closer, stderr *commandStderr, err error) (tasks.Cmd, io.Reader) {
	gitCmd := tasks.ExecCmd{Cmd: git}
	_ = gitCmd.Terminate()
	_ = gitOut.Close()
	_ = git.Wait()
	return reapedCmd{cmd: gitCmd}, strings.NewReader(renderFailureText(err, stderr))
}
