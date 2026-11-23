package gui

import (
	"errors"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/stretchr/testify/assert"
)

// startedRender starts a git-like command whose output is the renderer's stdin, and a
// renderer command, the way commandFileRender does, for tests of fileRenderCmd.
func startedRender(t *testing.T, git, renderer *exec.Cmd) (*fileRenderCmd, io.ReadCloser) {
	t.Helper()
	stderr := &commandStderr{}
	git.Stderr = stderr
	gitOut, err := git.StdoutPipe()
	mustNoError(t, err)
	mustNoError(t, git.Start())
	renderer.Stdin = gitOut
	renderer.Stderr = stderr
	renderer.Stdout = io.Discard
	mustNoError(t, renderer.Start())
	return &fileRenderCmd{git: git, gitOut: gitOut, renderer: renderer, stderr: stderr}, gitOut
}

// waitWithin returns the result of cmd.Wait, failing the test if it takes longer than d.
func waitWithin(t *testing.T, cmd *fileRenderCmd, d time.Duration) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(d):
		t.Fatal("the render's processes were not reaped in time")
		return nil
	}
}

// mustNoError fails the test at once if err is set, as require.NoError would.
func mustNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func skipWithoutShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
}

func TestFileRenderCmdTerminateStopsAndReapsBothProcesses(t *testing.T) {
	skipWithoutShell(t)
	git := exec.Command("sh", "-c", "exec sleep 60")
	renderer := exec.Command("sh", "-c", "exec sleep 60")
	cmd, _ := startedRender(t, git, renderer)

	assert.NoError(t, cmd.Terminate())
	assert.NoError(t, waitWithin(t, cmd, 10*time.Second), "a stopped render has no failure")

	assert.NotNil(t, git.ProcessState, "git was reaped")
	assert.NotNil(t, renderer.ProcessState, "the renderer was reaped")
	assert.False(t, git.ProcessState.Success())
	assert.False(t, renderer.ProcessState.Success())
}

func TestFileRenderCmdTerminateWorksWhenGitHasAlreadyExited(t *testing.T) {
	skipWithoutShell(t)
	git := exec.Command("sh", "-c", "exit 0")
	renderer := exec.Command("sh", "-c", "exec sleep 60")
	cmd, gitOut := startedRender(t, git, renderer)
	// git's output ends when git exits; reading it to the end is how we know it has.
	_, err := io.ReadAll(gitOut)
	mustNoError(t, err)

	assert.NoError(t, cmd.Terminate(), "a process that has exited is not an error to stop")
	assert.NoError(t, waitWithin(t, cmd, 10*time.Second), "git exited cleanly, and the renderer was stopped")
	assert.NotNil(t, renderer.ProcessState)
}

func TestFileRenderCmdReportsTheRenderersStderrWhenItFails(t *testing.T) {
	skipWithoutShell(t)
	git := exec.Command("sh", "-c", "printf text")
	renderer := exec.Command("sh", "-c", "echo boom >&2; exit 3")
	cmd, _ := startedRender(t, git, renderer)

	err := waitWithin(t, cmd, 10*time.Second)

	assert.Error(t, err)
	assert.Equal(t, "boom\n", renderFailureText(err, cmd.stderr))
}

func TestFileRenderCmdReportsTheErrorWhenTheRendererWritesNothingToStderr(t *testing.T) {
	skipWithoutShell(t)
	git := exec.Command("sh", "-c", "printf text")
	renderer := exec.Command("sh", "-c", "exit 4")
	cmd, _ := startedRender(t, git, renderer)

	err := waitWithin(t, cmd, 10*time.Second)

	assert.Error(t, err)
	assert.Equal(t, err.Error()+"\n", renderFailureText(err, cmd.stderr))
}

func TestFileRenderCmdDoesNotReportGitBeingStoppedByAFinishedRenderer(t *testing.T) {
	skipWithoutShell(t)
	// The renderer reads one byte and exits 0 while git writes forever; git must be stopped
	// by the closed pipe, and that is not a failure of the render.
	git := exec.Command("sh", "-c", "exec yes")
	renderer := exec.Command("sh", "-c", "head -c 1 >/dev/null")
	cmd, _ := startedRender(t, git, renderer)

	assert.NoError(t, waitWithin(t, cmd, 10*time.Second))
}

func TestFileRenderCmdReportsGitsOwnFailure(t *testing.T) {
	skipWithoutShell(t)
	git := exec.Command("sh", "-c", "echo 'fatal: no such path' >&2; exit 128")
	renderer := exec.Command("sh", "-c", "cat >/dev/null")
	cmd, _ := startedRender(t, git, renderer)

	err := waitWithin(t, cmd, 10*time.Second)

	assert.Error(t, err)
	assert.Equal(t, "fatal: no such path\n", renderFailureText(err, cmd.stderr))
}

func TestFailureReaderShowsTheFailureAfterTheOutput(t *testing.T) {
	skipWithoutShell(t)
	git := exec.Command("sh", "-c", "printf text")
	renderer := exec.Command("sh", "-c", "printf 'partial\\n'; echo boom >&2; exit 1")
	cmd, _ := startedRender(t, git, renderer)

	// The renderer's output is read first; the failure follows it, once the render has ended.
	output, err := io.ReadAll(io.MultiReader(strings.NewReader("partial\n"), &failureReader{cmd: cmd}))
	mustNoError(t, err)
	assert.Equal(t, "partial\nboom\n", string(output))
}

func TestFailureReaderShowsNothingForASuccessfulRender(t *testing.T) {
	skipWithoutShell(t)
	git := exec.Command("sh", "-c", "printf text")
	renderer := exec.Command("sh", "-c", "cat")
	cmd, _ := startedRender(t, git, renderer)

	output, err := io.ReadAll(&failureReader{cmd: cmd})
	mustNoError(t, err)
	assert.Empty(t, output)
}

func TestFileRendererFilenameReachesTheShellIntact(t *testing.T) {
	skipWithoutShell(t)
	osCmd := oscommands.NewDummyOSCommand()
	// Quotes, dollars, backslashes, backticks and spaces, all of which the shell would act on.
	for _, name := range []string{"my notes.txt", `it's "quoted".txt`, "$HOME `id` \\back.txt"} {
		fileRenderer := config.FileRendererConfig{Type: "stdinFilter", Command: "printf '%s' {{filename}}"}
		command, err := fileRenderer.ResolveCommand(name, osCmd.Quote)
		mustNoError(t, err)

		out, err := osCmd.Cmd.NewShell(command, "").GetCmd().Output()

		mustNoError(t, err)
		assert.Equal(t, name, string(out))
	}
}

func TestFileRenderCmdStringNamesBothCommands(t *testing.T) {
	skipWithoutShell(t)
	git := exec.Command("git", "show", "rev:path")
	renderer := exec.Command("sh", "-c", "cat")
	cmd := &fileRenderCmd{git: git, renderer: renderer}

	assert.Equal(t, git.String()+" | "+renderer.String(), cmd.String())
}

func TestRenderFailureIsTheRenderersAndOtherwiseGits(t *testing.T) {
	rendererErr := errors.New("renderer")
	gitErr := errors.New("git")

	assert.Equal(t, rendererErr, renderFailure(rendererErr, nil))
	assert.Equal(t, rendererErr, renderFailure(rendererErr, gitErr))
	assert.Nil(t, renderFailure(nil, gitErr), "an error that is not an exit status of git's own is not a failure")
	assert.Nil(t, renderFailure(nil, nil))
}
