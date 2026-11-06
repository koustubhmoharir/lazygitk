package filelock

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// helperLockEnv names the lock file for TestHelperProcess. It is set only when
// the test binary runs as a child process.
const helperLockEnv = "FILELOCK_TEST_HELPER_LOCK"

// noErr stops the test immediately on error, since later steps depend on it.
func noErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func lockPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "target.lock")
}

func TestSecondAcquireTimesOutWhileHeld(t *testing.T) {
	path := lockPath(t)
	first, err := Acquire(path, time.Second)
	noErr(t, err)
	defer func() { assert.NoError(t, first.Release()) }()

	start := time.Now()
	second, err := Acquire(path, 100*time.Millisecond)
	assert.Nil(t, second)
	assert.ErrorContains(t, err, "timed out")
	assert.GreaterOrEqual(t, time.Since(start), 100*time.Millisecond)
}

func TestAcquireSucceedsAfterRelease(t *testing.T) {
	path := lockPath(t)
	first, err := Acquire(path, time.Second)
	noErr(t, err)
	noErr(t, first.Release())

	second, err := Acquire(path, 100*time.Millisecond)
	noErr(t, err)
	assert.NoError(t, second.Release())
}

func TestLockFileStaysOnDisk(t *testing.T) {
	path := lockPath(t)
	lock, err := Acquire(path, time.Second)
	noErr(t, err)
	noErr(t, lock.Release())

	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestReleaseTwiceIsHarmless(t *testing.T) {
	lock, err := Acquire(lockPath(t), time.Second)
	noErr(t, err)
	assert.NoError(t, lock.Release())
	assert.NoError(t, lock.Release())
}

// TestHelperProcess is not a real test. TestLockHeldByExitedProcessIsReleased
// runs the test binary with helperLockEnv set, and this function then takes the
// lock, reports it, and exits without releasing it.
func TestHelperProcess(t *testing.T) {
	path := os.Getenv(helperLockEnv)
	if path == "" {
		t.Skip("only runs as a child process")
	}
	if _, err := Acquire(path, time.Second); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("locked")
	// Wait until the parent closes stdin, so the lock is held while the parent checks it.
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	os.Exit(0)
}

func TestLockHeldByExitedProcessIsReleased(t *testing.T) {
	path := lockPath(t)

	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(), helperLockEnv+"="+path)
	stdin, err := cmd.StdinPipe()
	noErr(t, err)
	stdout, err := cmd.StdoutPipe()
	noErr(t, err)
	noErr(t, cmd.Start())
	exited := false
	defer func() {
		if !exited {
			_ = stdin.Close()
			_ = cmd.Wait()
		}
	}()

	line, err := bufio.NewReader(stdout).ReadString('\n')
	noErr(t, err)
	assert.Equal(t, "locked\n", line)
	if line != "locked\n" {
		t.FailNow()
	}

	_, err = Acquire(path, 100*time.Millisecond)
	assert.ErrorContains(t, err, "timed out", "the child process holds the lock while it runs")

	noErr(t, stdin.Close())
	noErr(t, cmd.Wait())
	exited = true

	lock, err := Acquire(path, time.Second)
	// The lock must be released when the child process exits.
	noErr(t, err)
	assert.NoError(t, lock.Release())
}
