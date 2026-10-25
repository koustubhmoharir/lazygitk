package commandusage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

const (
	defaultLockTimeout  = time.Second
	defaultStaleLockAge = 10 * time.Second
	lockRetryInterval   = 10 * time.Millisecond
)

// lock creates the lock file next to the counts file, which serializes writers
// across processes. It returns a function that releases the lock. A lock file
// older than staleLockAge is assumed to be left behind by a crashed process
// and is removed.
func (s *Store) lock() (unlock func(), err error) {
	lockPath := s.path + ".lock"
	deadline := time.Now().Add(s.lockTimeout)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			if err := f.Close(); err != nil {
				_ = os.Remove(lockPath)
				return nil, err
			}
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
		if s.breakStaleLock(lockPath) {
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for lock %s", lockPath)
		}
		time.Sleep(lockRetryInterval)
	}
}

// breakStaleLock removes lockPath if it is older than staleLockAge. It reports
// whether the lock was removed.
func (s *Store) breakStaleLock(lockPath string) bool {
	info, err := os.Stat(lockPath)
	if err != nil || time.Since(info.ModTime()) < s.staleLockAge {
		return false
	}
	return os.Remove(lockPath) == nil
}
