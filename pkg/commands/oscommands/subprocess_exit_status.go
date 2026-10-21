package oscommands

import (
	"errors"
	"os/exec"
)

// AcceptSubprocessExitCodes marks the given subprocess exit codes as successful
// when SubprocessError is used to normalize its result.
func (self *CmdObj) AcceptSubprocessExitCodes(codes ...int) *CmdObj {
	self.subprocessSuccessExitCodes = append([]int(nil), codes...)
	return self
}

// SubprocessError returns nil for an allowed subprocess exit code, and
// otherwise preserves the original error.
func (self *CmdObj) SubprocessError(err error) error {
	if err == nil {
		return nil
	}

	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		return err
	}

	for _, code := range self.subprocessSuccessExitCodes {
		if exitError.ExitCode() == code {
			return nil
		}
	}

	return err
}
