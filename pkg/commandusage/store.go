// Package commandusage persists per-user command usage counts. Several lazygit
// processes may share one counts file, so writes merge deltas into the file
// under a lock instead of overwriting it.
package commandusage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const formatVersion = 1

var errCorrupt = errors.New("corrupt command usage file")

type fileFormat struct {
	Version int            `json:"version"`
	Counts  map[string]int `json:"counts"`
}

// Store holds usage counts in memory and flushes deltas to a shared file.
// It is safe for concurrent use.
type Store struct {
	path         string
	lockTimeout  time.Duration
	staleLockAge time.Duration

	// flushMu serializes file operations, so only one Flush or Reset runs at a time.
	flushMu sync.Mutex

	// mu guards the maps below. It is never held during file I/O, so Add never blocks on disk.
	mu       sync.Mutex
	base     map[string]int // counts as last read from or written to disk
	inFlight map[string]int // deltas handed to the Flush that is currently writing
	pending  map[string]int // deltas not yet handed to a Flush
}

// Open loads the counts file at path. A missing file yields an empty store.
// If the file is unparseable or has an unsupported version, it is removed and
// reset is true so the caller can warn the user. Other read errors return an
// empty store and the error; they are not fatal.
func Open(path string) (store *Store, reset bool, err error) {
	s := newStore(path)
	counts, err := readFile(path)
	if errors.Is(err, errCorrupt) {
		if rmErr := os.Remove(path); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
			return s, true, rmErr
		}
		return s, true, nil
	}
	if err != nil {
		return s, false, err
	}
	s.base = counts
	return s, false, nil
}

func newStore(path string) *Store {
	return &Store{
		path:         path,
		lockTimeout:  defaultLockTimeout,
		staleLockAge: defaultStaleLockAge,
		base:         map[string]int{},
		inFlight:     map[string]int{},
		pending:      map[string]int{},
	}
}

// Add records a usage delta for id in memory. Call Flush to persist it.
func (s *Store) Add(id string, delta int) {
	if id == "" || delta == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	addDelta(s.pending, id, delta)
}

// Count returns the persisted count for id plus any unflushed deltas.
func (s *Store) Count(id string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.base[id] + s.inFlight[id] + s.pending[id]
}

// Snapshot returns a copy of all non-zero counts, including unflushed deltas.
func (s *Store) Snapshot() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := map[string]int{}
	for _, m := range []map[string]int{s.base, s.inFlight, s.pending} {
		for id, n := range m {
			result[id] += n
		}
	}
	for id, n := range result {
		if n == 0 {
			delete(result, id)
		}
	}
	return result
}

// HasPending reports whether there are deltas that have not been written yet.
func (s *Store) HasPending() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.inFlight) > 0 || len(s.pending) > 0
}

// Flush merges the pending deltas into the counts file under the lock file.
// The file is re-read first, so deltas written by other processes are kept.
// On failure the deltas stay pending for a later Flush. reset is true if a
// corrupt file was replaced.
func (s *Store) Flush() (reset bool, err error) {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	s.mu.Lock()
	if len(s.pending) == 0 {
		s.mu.Unlock()
		return false, nil
	}
	s.inFlight, s.pending = s.pending, map[string]int{}
	deltas := s.inFlight
	s.mu.Unlock()

	counts, reset, err := s.update(func(counts map[string]int) {
		for id, delta := range deltas {
			counts[id] += delta
		}
	})

	s.mu.Lock()
	defer s.mu.Unlock()
	s.inFlight = map[string]int{}
	if err != nil {
		for id, delta := range deltas {
			addDelta(s.pending, id, delta)
		}
		return false, err
	}
	s.base = counts
	return reset, nil
}

// Reset empties the counts file and clears all counts in memory.
func (s *Store) Reset() error {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	if _, _, err := s.update(func(counts map[string]int) {
		clear(counts)
	}); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.base = map[string]int{}
	s.pending = map[string]int{}
	return nil
}

// update re-reads the counts file under the lock, applies change, and writes
// the result back. A corrupt file is treated as empty, and reset is reported
// only if the replacement succeeded.
func (s *Store) update(change func(map[string]int)) (counts map[string]int, reset bool, err error) {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return nil, false, err
	}

	unlock, err := s.lock()
	if err != nil {
		return nil, false, err
	}
	defer unlock()

	counts, err = readFile(s.path)
	corrupt := errors.Is(err, errCorrupt)
	if corrupt {
		counts, err = map[string]int{}, nil
	}
	if err != nil {
		return nil, false, err
	}

	change(counts)
	for id, n := range counts {
		if n == 0 {
			delete(counts, id)
		}
	}
	if err := writeFile(s.path, counts); err != nil {
		return nil, false, err
	}
	return counts, corrupt, nil
}

// addDelta adds delta to m[id], removing the entry if it nets to zero.
func addDelta(m map[string]int, id string, delta int) {
	m[id] += delta
	if m[id] == 0 {
		delete(m, id)
	}
}

// readFile returns the counts in path. A missing file has no counts. A file
// that cannot be parsed, or that has an unsupported version, returns errCorrupt.
func readFile(path string) (map[string]int, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]int{}, nil
	}
	if err != nil {
		return nil, err
	}

	var f fileFormat
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("%w: %w", errCorrupt, err)
	}
	if f.Version != formatVersion {
		return nil, fmt.Errorf("%w: unsupported version %d", errCorrupt, f.Version)
	}
	if f.Counts == nil {
		return nil, fmt.Errorf("%w: missing counts", errCorrupt)
	}
	return f.Counts, nil
}

// writeFile atomically replaces path with counts. It writes a temp file in the
// same directory, syncs it, and renames it over path.
func writeFile(path string, counts map[string]int) error {
	data, err := json.Marshal(fileFormat{Version: formatVersion, Counts: counts})
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	// Removes the temp file if we fail before the rename; after a successful rename this is a no-op.
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// CreateTemp uses 0o600, but the counts file is meant to be readable like state.yml.
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
