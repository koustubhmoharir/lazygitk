package oscommands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

const subprocessExitStatusHelperEnv = "LAZYGITK_SUBPROCESS_EXIT_STATUS_HELPER"

func TestSubprocessExitStatusHelper(t *testing.T) {
	codeString := os.Getenv(subprocessExitStatusHelperEnv)
	if codeString == "" {
		return
	}
	code, err := strconv.Atoi(codeString)
	if err != nil {
		t.Fatalf("invalid helper exit code %q: %v", codeString, err)
	}
	os.Exit(code)
}

func subprocessExitError(t *testing.T, code int) error {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestSubprocessExitStatusHelper$")
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%d", subprocessExitStatusHelperEnv, code))
	err := cmd.Run()
	if err == nil {
		t.Fatalf("helper unexpectedly exited successfully; wanted %d", code)
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitError.ExitCode() != code {
		t.Fatalf("expected exit code %d, got %d", code, exitError.ExitCode())
	}
	return err
}

func TestSubprocessError(t *testing.T) {
	t.Run("nil remains nil", func(t *testing.T) {
		if err := (&CmdObj{}).SubprocessError(nil); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("default preserves exit errors", func(t *testing.T) {
		err := subprocessExitError(t, 1)
		assert.Same(t, err, (&CmdObj{}).SubprocessError(err))
	})

	for _, code := range []int{1, 2, 128} {
		t.Run(fmt.Sprintf("allowed %d", code), func(t *testing.T) {
			exitErr := subprocessExitError(t, code)
			wrappedErr := fmt.Errorf("subprocess failed: %w", exitErr)
			if got := (&CmdObj{}).AcceptSubprocessExitCodes(code).SubprocessError(wrappedErr); got != nil {
				t.Fatalf("expected nil, got %v", got)
			}
		})
	}

	for _, code := range []int{2, 128} {
		t.Run(fmt.Sprintf("disallowed wrapped exit error %d is preserved", code), func(t *testing.T) {
			exitErr := subprocessExitError(t, code)
			wrappedErr := fmt.Errorf("subprocess failed: %w", exitErr)
			got := (&CmdObj{}).AcceptSubprocessExitCodes(1).SubprocessError(wrappedErr)
			assert.Same(t, wrappedErr, got)
		})
	}

	t.Run("start failure is never accepted", func(t *testing.T) {
		cmd := exec.Command("this-subprocess-exit-status-binary-does-not-exist")
		err := cmd.Run()
		if err == nil {
			t.Fatal("expected command start failure")
		}
		got := (&CmdObj{}).AcceptSubprocessExitCodes(-1).SubprocessError(err)
		assert.Same(t, err, got)
	})

	t.Run("non-exit sentinel error is preserved", func(t *testing.T) {
		err := errors.New("sentinel")
		got := (&CmdObj{}).AcceptSubprocessExitCodes(1).SubprocessError(err)
		assert.Same(t, err, got)
	})

	t.Run("allowlist is copied and command local", func(t *testing.T) {
		codes := []int{1}
		first := (&CmdObj{}).AcceptSubprocessExitCodes(codes...)
		codes[0] = 2
		second := &CmdObj{}

		if got := first.SubprocessError(subprocessExitError(t, 1)); got != nil {
			t.Fatalf("first command should still allow original code 1, got %v", got)
		}
		if got := first.SubprocessError(subprocessExitError(t, 2)); got == nil {
			t.Fatal("first command unexpectedly allowed mutated code 2")
		}
		if got := second.SubprocessError(subprocessExitError(t, 1)); got == nil {
			t.Fatal("second command inherited first command's allowlist")
		}
	})
}
