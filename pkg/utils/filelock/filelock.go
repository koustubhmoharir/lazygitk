// Package filelock is an exclusive lock between lazygitk instances on files that
// the application owns, such as command usage counts and commit drafts.
//
// The lock is an OS advisory lock on a lock file, which is separate from the file
// it protects. The OS releases the lock when the holding process exits, so a
// crashed instance cannot leave it held. The lock file is never deleted: unlinking
// it would let one process lock the old file while another locks a new file at
// the same path, which would break mutual exclusion.
package filelock

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// DefaultTimeout is how long Acquire waits for another instance to release a lock.
const DefaultTimeout = 2 * time.Second

const retryInterval = 10 * time.Millisecond

// errBusy means another holder has the lock. Platform code returns it from tryLock.
var errBusy = errors.New("lock is held by another process")

// Lock is an acquired exclusive lock. Release it when done.
type Lock struct {
	file *os.File
}

// Acquire creates the lock file at path if needed, then waits up to timeout for
// an exclusive lock on it. It returns an error if the lock is still held by
// another holder after timeout.
func Acquire(path string, timeout time.Duration) (*Lock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	for {
		err := tryLock(file)
		if err == nil {
			return &Lock{file: file}, nil
		}
		if !errors.Is(err, errBusy) {
			_ = file.Close()
			return nil, fmt.Errorf("could not lock %s: %w", path, err)
		}
		if !time.Now().Before(deadline) {
			_ = file.Close()
			return nil, fmt.Errorf("timed out after %s waiting for another lazygitk instance to release %s", timeout, path)
		}
		time.Sleep(retryInterval)
	}
}

// Release unlocks and closes the lock file. The lock file stays on disk.
func (l *Lock) Release() error {
	if l.file == nil {
		return nil
	}
	unlockErr := unlock(l.file)
	closeErr := l.file.Close()
	l.file = nil
	return errors.Join(unlockErr, closeErr)
}
