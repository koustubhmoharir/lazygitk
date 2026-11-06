package commandusage

import (
	"github.com/jesseduffield/lazygit/pkg/utils/filelock"
)

// lock takes the exclusive lock that serializes writers to the counts file across
// processes. It returns a function that releases the lock. The OS releases the
// lock if the holding process dies, so no stale lock is ever broken.
func (s *Store) lock() (unlock func(), err error) {
	lock, err := filelock.Acquire(s.path+".lock", s.lockTimeout)
	if err != nil {
		return nil, err
	}
	return func() { _ = lock.Release() }, nil
}
