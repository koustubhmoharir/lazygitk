package updates

import (
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/stretchr/testify/assert"
)

// fakeConfig supplies only the build identity and state the updater reads.
// Embedding a nil AppConfigurer makes any unexpected call panic.
type fakeConfig struct {
	config.AppConfigurer
	version     string
	buildSource string
	dir         string
	appState    *config.AppState
}

func (c *fakeConfig) GetVersion() string     { return c.version }
func (c *fakeConfig) GetBuildSource() string { return c.buildSource }
func (c *fakeConfig) GetUserConfigDir() string {
	return c.dir
}
func (c *fakeConfig) GetAppState() *config.AppState { return c.appState }
func (c *fakeConfig) SaveAppState() error           { return nil }

// stubNetwork replaces the HTTP transport with one that counts requests and
// fails them, so no test reaches a real server. It returns the request counter.
func stubNetwork(t *testing.T) *atomic.Int32 {
	t.Helper()
	var requests atomic.Int32
	previous := http.DefaultClient.Transport
	http.DefaultClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, errors.New("network disabled in tests")
	})
	t.Cleanup(func() { http.DefaultClient.Transport = previous })
	return &requests
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newTestUpdater(t *testing.T, version string, buildSource string) *Updater {
	t.Helper()
	return &Updater{
		Common: common.NewDummyCommon(),
		Config: &fakeConfig{
			version:     version,
			buildSource: buildSource,
			dir:         t.TempDir(),
			appState:    &config.AppState{},
		},
	}
}

// Fork: lazygitk builds must never check for or install upstream releases,
// even when the user asks for a manual check. These cases are the ones that
// previously reached the download path because userRequested skipped the
// build-source guard.
func TestCheckForNewUpdateSkipsForkBuildsWhenUserRequested(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		buildSource string
	}{
		{name: "forkDist release", version: "0.66.2", buildSource: "forkDist"},
		{name: "forkDist unversioned archive", version: "unversioned", buildSource: "forkDist"},
		{name: "unversioned dev build", version: "unversioned", buildSource: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := stubNetwork(t)
			updater := newTestUpdater(t, tt.version, tt.buildSource)

			finished := false
			updater.CheckForNewUpdate(func(string, error) error {
				finished = true
				return nil
			}, true)

			assert.False(t, finished, "no check result should be reported")
			assert.Equal(t, int32(0), requests.Load(), "no release request should be made")
		})
	}
}

// Fork: Update is the install boundary and must refuse for lazygitk builds
// even if it is reached without going through CheckForNewUpdate.
func TestUpdateRefusesForkBuilds(t *testing.T) {
	requests := stubNetwork(t)
	updater := newTestUpdater(t, "0.66.2", "forkDist")

	err := updater.Update("v0.66.2")

	assert.Error(t, err)
	assert.Equal(t, updater.Common.Tr.ForkCannotSelfUpdate, err.Error())
	assert.Equal(t, int32(0), requests.Load(), "no download should be attempted")
}

func TestCanSelfUpdate(t *testing.T) {
	assert.True(t, newTestUpdater(t, "0.66.2", "buildBinary").CanSelfUpdate())
	assert.False(t, newTestUpdater(t, "0.66.2", "forkDist").CanSelfUpdate())
	assert.False(t, newTestUpdater(t, "unversioned", "").CanSelfUpdate())
}
