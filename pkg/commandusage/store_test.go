package commandusage

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// noErr stops the test immediately on error, since later steps depend on it.
func noErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func countsPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "command_usage.json")
}

func openStore(t *testing.T, path string) *Store {
	t.Helper()
	s, reset, err := Open(path)
	noErr(t, err)
	assert.False(t, reset)
	return s
}

func reopen(t *testing.T, path string) *Store {
	t.Helper()
	s, _, err := Open(path)
	noErr(t, err)
	return s
}

func TestOpenMissingFile(t *testing.T) {
	path := countsPath(t)
	s, reset, err := Open(path)
	noErr(t, err)
	assert.False(t, reset)
	assert.Equal(t, 0, s.Count("x"))
	assert.Empty(t, s.Snapshot())
	assert.False(t, s.HasPending())
}

func TestAddCountSnapshot(t *testing.T) {
	s := openStore(t, countsPath(t))

	s.Add("a", 3)
	s.Add("a", -1)
	s.Add("b", -2)
	s.Add("", 5)
	s.Add("c", 0)

	assert.Equal(t, 2, s.Count("a"))
	assert.Equal(t, -2, s.Count("b"))
	assert.Equal(t, 0, s.Count("c"))
	assert.Equal(t, 0, s.Count(""))
	assert.Equal(t, map[string]int{"a": 2, "b": -2}, s.Snapshot())
	assert.True(t, s.HasPending())

	s.Add("b", 2)
	assert.Equal(t, map[string]int{"a": 2}, s.Snapshot())
}

func TestFlushAndReopen(t *testing.T) {
	path := countsPath(t)
	s := openStore(t, path)
	s.Add("x", 4)
	s.Add("y", -1)

	reset, err := s.Flush()
	noErr(t, err)
	assert.False(t, reset)
	assert.False(t, s.HasPending())
	assert.Equal(t, 4, s.Count("x"))

	data, err := os.ReadFile(path)
	noErr(t, err)
	var f fileFormat
	noErr(t, json.Unmarshal(data, &f))
	assert.Equal(t, formatVersion, f.Version)
	assert.Equal(t, map[string]int{"x": 4, "y": -1}, f.Counts)

	again := reopen(t, path)
	assert.Equal(t, map[string]int{"x": 4, "y": -1}, again.Snapshot())
}

func TestFlushWithoutPendingDoesNothing(t *testing.T) {
	path := countsPath(t)
	reset, err := openStore(t, path).Flush()
	noErr(t, err)
	assert.False(t, reset)
	_, err = os.Stat(path)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestFlushMergesDeltasFromTwoStores(t *testing.T) {
	path := countsPath(t)
	a := openStore(t, path)
	b := openStore(t, path)

	a.Add("x", 1)
	b.Add("x", -1)
	b.Add("y", 2)

	_, err := a.Flush()
	noErr(t, err)
	_, err = b.Flush()
	noErr(t, err)

	result := reopen(t, path)
	assert.Equal(t, 0, result.Count("x"))
	assert.Equal(t, 2, result.Count("y"))
	assert.Equal(t, map[string]int{"y": 2}, result.Snapshot())
	assert.Equal(t, map[string]int{"y": 2}, b.Snapshot())
}

func TestCorruptFileIsRemovedOnOpen(t *testing.T) {
	tests := map[string]string{
		"garbage":        "\x00\xffnot json",
		"wrong version":  `{"version": 2, "counts": {"x": 1}}`,
		"non-int count":  `{"version": 1, "counts": {"x": "many"}}`,
		"missing counts": `{"version": 1}`,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := countsPath(t)
			noErr(t, os.WriteFile(path, []byte(content), 0o644))

			s, reset, err := Open(path)
			noErr(t, err)
			assert.True(t, reset)
			assert.Empty(t, s.Snapshot())
			_, err = os.Stat(path)
			assert.ErrorIs(t, err, fs.ErrNotExist)
		})
	}
}

func TestCorruptionAppearingBeforeFlushIsReset(t *testing.T) {
	path := countsPath(t)
	s := openStore(t, path)
	s.Add("x", 3)

	noErr(t, os.WriteFile(path, []byte("garbage"), 0o644))

	reset, err := s.Flush()
	noErr(t, err)
	assert.True(t, reset)
	assert.False(t, s.HasPending())

	result := reopen(t, path)
	assert.Equal(t, map[string]int{"x": 3}, result.Snapshot())
}

func TestStaleLockIsBroken(t *testing.T) {
	path := countsPath(t)
	lockPath := path + ".lock"
	noErr(t, os.WriteFile(lockPath, nil, 0o644))
	old := time.Now().Add(-time.Hour)
	noErr(t, os.Chtimes(lockPath, old, old))

	s := openStore(t, path)
	s.Add("x", 1)
	_, err := s.Flush()
	noErr(t, err)

	_, err = os.Stat(lockPath)
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.Equal(t, 1, reopen(t, path).Count("x"))
}

func TestFreshLockTimesOutAndKeepsPending(t *testing.T) {
	path := countsPath(t)
	lockPath := path + ".lock"
	noErr(t, os.WriteFile(lockPath, nil, 0o644))

	s := openStore(t, path)
	s.lockTimeout = 50 * time.Millisecond
	s.Add("x", 2)

	_, err := s.Flush()
	assert.Error(t, err)
	assert.True(t, s.HasPending())
	assert.Equal(t, 2, s.Count("x"))
	_, statErr := os.Stat(path)
	assert.ErrorIs(t, statErr, fs.ErrNotExist)

	noErr(t, os.Remove(lockPath))
	_, err = s.Flush()
	noErr(t, err)
	assert.False(t, s.HasPending())
	assert.Equal(t, 2, reopen(t, path).Count("x"))
}

func TestUnwritableDirectoryKeepsPending(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not apply on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "command_usage.json")
	s := openStore(t, path)
	s.Add("x", 5)

	noErr(t, os.Chmod(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, err := s.Flush()
	assert.Error(t, err)
	assert.True(t, s.HasPending())
	assert.Equal(t, 5, s.Count("x"))
}

func TestMissingParentDirectoryIsCreated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "command_usage.json")
	s := openStore(t, path)
	s.Add("x", 1)
	_, err := s.Flush()
	noErr(t, err)
	assert.Equal(t, 1, reopen(t, path).Count("x"))
}

func TestResetClearsFileAndMemory(t *testing.T) {
	path := countsPath(t)
	s := openStore(t, path)
	s.Add("x", 2)
	_, err := s.Flush()
	noErr(t, err)
	s.Add("y", 1)

	noErr(t, s.Reset())
	assert.Empty(t, s.Snapshot())
	assert.False(t, s.HasPending())

	data, err := os.ReadFile(path)
	noErr(t, err)
	var f fileFormat
	noErr(t, json.Unmarshal(data, &f))
	assert.Equal(t, fileFormat{Version: formatVersion, Counts: map[string]int{}}, f)
	assert.Empty(t, reopen(t, path).Snapshot())
}

func TestConcurrentAddIsRaceFree(t *testing.T) {
	s := openStore(t, countsPath(t))
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				s.Add("x", 1)
				_ = s.Count("x")
				_ = s.Snapshot()
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, 800, s.Count("x"))
}

func TestLockFileIsReleased(t *testing.T) {
	path := countsPath(t)
	s := openStore(t, path)
	s.Add("x", 1)
	_, err := s.Flush()
	noErr(t, err)
	_, err = os.Stat(path + ".lock")
	assert.True(t, errors.Is(err, fs.ErrNotExist))
	assert.Equal(t, map[string]int{"x": 1}, reopen(t, path).Snapshot())
}
