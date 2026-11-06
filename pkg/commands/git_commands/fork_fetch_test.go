package git_commands

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/stretchr/testify/assert"
)

func TestFetchRefspecRisks(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	// "remote add" gives origin the default refspec, which writes only into
	// remote-tracking refs.
	gitTest(t, root, "remote", "add", "origin", "https://example.invalid/origin.git")
	gitTest(t, root, "remote", "add", "other", "https://example.invalid/other.git")
	gitTest(t, root, "remote", "add", "quiet", "https://example.invalid/quiet.git")
	gitTest(t, root, "config", "--unset-all", "remote.quiet.fetch")
	commands := branchSyncTestCommands(t, root)

	risks, err := commands.FetchRefspecRisks([]string{"origin", "quiet", "missing"})
	assert.NoError(t, err)
	assert.Empty(t, risks, "the default refspec, a remote without refspecs and an absent remote are not risks")

	gitTest(t, root, "config", "--add", "remote.origin.fetch", "+refs/heads/topic:refs/heads/topic")
	risks, err = commands.FetchRefspecRisks([]string{"origin"})
	assert.NoError(t, err)
	assert.Equal(t, []FetchRefspecRisk{{Remote: "origin", Refspec: "+refs/heads/topic:refs/heads/topic"}}, risks)

	// Negative refspecs and refspecs without a destination write nothing.
	gitTest(t, root, "config", "--add", "remote.origin.fetch", "^refs/heads/test")
	gitTest(t, root, "config", "--add", "remote.other.fetch", "refs/heads/topic")
	risks, err = commands.FetchRefspecRisks([]string{"origin", "other"})
	assert.NoError(t, err)
	assert.Equal(t, []FetchRefspecRisk{{Remote: "origin", Refspec: "+refs/heads/topic:refs/heads/topic"}}, risks)

	// Several refspecs of several remotes keep their order.
	gitTest(t, root, "config", "--add", "remote.other.fetch", "refs/heads/*:refs/heads/mirror/*")
	gitTest(t, root, "config", "--add", "remote.other.fetch", "+refs/heads/*:refs/remotes/other/*")
	risks, err = commands.FetchRefspecRisks([]string{"other", "origin"})
	assert.NoError(t, err)
	assert.Equal(t, []FetchRefspecRisk{
		{Remote: "other", Refspec: "refs/heads/*:refs/heads/mirror/*"},
		{Remote: "origin", Refspec: "+refs/heads/topic:refs/heads/topic"},
	}, risks)
}

func TestRefspecWritesLocalBranch(t *testing.T) {
	assert.True(t, refspecWritesLocalBranch("+refs/heads/topic:refs/heads/topic"))
	assert.True(t, refspecWritesLocalBranch("refs/heads/*:refs/heads/*"))
	assert.False(t, refspecWritesLocalBranch("+refs/heads/*:refs/remotes/origin/*"))
	assert.False(t, refspecWritesLocalBranch("^refs/heads/test"))
	assert.False(t, refspecWritesLocalBranch("refs/heads/topic"))
	assert.False(t, refspecWritesLocalBranch("refs/heads/topic:"))
}

func TestPrimaryFetchRemote(t *testing.T) {
	cases := []struct {
		remotes []string
		want    string
		ok      bool
	}{
		{remotes: []string{"origin"}, want: "origin", ok: true},
		{remotes: []string{"alpha", "origin"}, want: "origin", ok: true},
		{remotes: []string{"upstream"}, want: "upstream", ok: true},
		{remotes: []string{"alpha", "beta"}, ok: false},
		{remotes: nil, ok: false},
	}
	for _, c := range cases {
		remote, ok := PrimaryFetchRemote(c.remotes)
		assert.Equal(t, c.ok, ok, "%v", c.remotes)
		assert.Equal(t, c.want, remote, "%v", c.remotes)
	}
}

func TestFetchBackgroundRefspecRisks(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	gitTest(t, root, "remote", "add", "origin", "https://example.invalid/origin.git")
	gitTest(t, root, "remote", "add", "other", "https://example.invalid/other.git")
	gitTest(t, root, "config", "--add", "remote.other.fetch", "+refs/heads/topic:refs/heads/topic")
	commands := branchSyncTestCommands(t, root)
	risky := []FetchRefspecRisk{{Remote: "other", Refspec: "+refs/heads/topic:refs/heads/topic"}}

	// Without git.fetchAll, a plain fetch goes to origin, since the branch has
	// no remote of its own.
	commands.UserConfig().Git.FetchAll = false
	risks, err := commands.FetchBackgroundRefspecRisks()
	assert.NoError(t, err)
	assert.Empty(t, risks)

	gitTest(t, root, "config", "branch.master.remote", "other")
	risks, err = commands.FetchBackgroundRefspecRisks()
	assert.NoError(t, err)
	assert.Equal(t, risky, risks)

	// With git.fetchAll, every remote is fetched.
	gitTest(t, root, "config", "--unset", "branch.master.remote")
	commands.UserConfig().Git.FetchAll = true
	risks, err = commands.FetchBackgroundRefspecRisks()
	assert.NoError(t, err)
	assert.Equal(t, risky, risks)
}

func TestSyncFetchAllRemotesArgsIgnoreFetchAllConfig(t *testing.T) {
	// The global f and "Fetch all remotes" pass --all themselves, whatever
	// git.fetchAll says.
	for _, fetchAll := range []bool{false, true} {
		instance := buildSyncCommands(commonDeps{})
		instance.UserConfig().Git.FetchAll = fetchAll
		cmdObj := instance.fetchAllRemotesCmdObj(gocui.NewFakeTask())
		assert.True(t, cmdObj.ShouldLog())
		assert.Equal(t, cmdObj.GetCredentialStrategy(), oscommands.PROMPT)
		assert.False(t, cmdObj.ShouldSuppressOutputUnlessError())
		assert.Equal(t, []string{"git", "fetch", "--all", "--no-write-fetch-head"}, cmdObj.Args())
	}
}
