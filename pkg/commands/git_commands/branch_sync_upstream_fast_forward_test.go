package git_commands

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// upstreamFFConfigure points branch "selected" at an upstream of the given
// kind: a named remote-tracking ref, or a local branch via remote ".".
func upstreamFFConfigure(t *testing.T, root, kind string) string {
	t.Helper()
	switch kind {
	case "remote":
		gitTest(t, root, "remote", "add", "origin", "https://example.invalid/origin.git")
		gitTest(t, root, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
		gitTest(t, root, "config", "branch.selected.remote", "origin")
		gitTest(t, root, "config", "branch.selected.merge", "refs/heads/main")
		return "refs/remotes/origin/main"
	case "local":
		gitTest(t, root, "config", "branch.selected.remote", ".")
		gitTest(t, root, "config", "branch.selected.merge", "refs/heads/upstream")
		return "refs/heads/upstream"
	}
	t.Fatalf("unknown upstream kind %q", kind)
	return ""
}

// upstreamFFHistory returns the base commit, a descendant of it on master,
// and a sibling of that descendant that diverges from it.
func upstreamFFHistory(t *testing.T, root string) (base, ahead, sibling string) {
	t.Helper()
	base = gitTest(t, root, "rev-parse", "HEAD")
	ahead = branchPlacementTestCommit(t, root, "ahead", "ahead\n", "ahead")
	gitTest(t, root, "checkout", "-q", "-b", "sibling", base)
	sibling = branchPlacementTestCommit(t, root, "sibling", "sibling\n", "sibling")
	gitTest(t, root, "checkout", "-q", "master")
	return base, ahead, sibling
}

func TestUpstreamFastForwardClassifiesLocalTrackingRef(t *testing.T) {
	for _, kind := range []string{"remote", "local"} {
		t.Run(kind, func(t *testing.T) {
			root, _ := branchSyncTestRepo(t)
			base, ahead, sibling := upstreamFFHistory(t, root)
			trackingRef := upstreamFFConfigure(t, root, kind)
			commands := branchSyncTestCommands(t, root)
			cases := []struct {
				name               string
				selected, upstream string
				want               UpstreamFastForwardState
			}{
				{"behind", base, ahead, UpstreamFastForwardBehind},
				{"ahead", ahead, base, UpstreamFastForwardUpToDate},
				{"equal", ahead, ahead, UpstreamFastForwardUpToDate},
				{"diverged", ahead, sibling, UpstreamFastForwardDiverged},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					gitTest(t, root, "update-ref", "refs/heads/selected", tc.selected)
					gitTest(t, root, "update-ref", trackingRef, tc.upstream)
					got, err := commands.InspectUpstreamFastForward("selected")
					if err != nil {
						t.Fatal(err)
					}
					assert.Equal(t, tc.want, got.State)
					assert.Equal(t, tc.upstream, got.UpstreamOID)
					assert.Equal(t, trackingRef, got.Info.UpstreamTrackingRef)
					assert.Equal(t, tc.selected, got.Info.OID)
					assert.False(t, got.Current())
				})
			}
		})
	}
}

func TestUpstreamFastForwardNoUpstreamConfigured(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	base := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "branch", "selected", base)
	got, err := branchSyncTestCommands(t, root).InspectUpstreamFastForward("selected")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, UpstreamFastForwardNoUpstream, got.State)
	assert.Empty(t, got.UpstreamOID)
}

func TestUpstreamFastForwardMissingTrackingRef(t *testing.T) {
	for _, kind := range []string{"remote", "local"} {
		t.Run(kind, func(t *testing.T) {
			root, _ := branchSyncTestRepo(t)
			base := gitTest(t, root, "rev-parse", "HEAD")
			gitTest(t, root, "branch", "selected", base)
			upstreamFFConfigure(t, root, kind)
			got, err := branchSyncTestCommands(t, root).InspectUpstreamFastForward("selected")
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, UpstreamFastForwardMissingTrackingRef, got.State)
			assert.Empty(t, got.UpstreamOID)
		})
	}
}

func TestUpstreamFastForwardRejectsSymbolicTrackingRef(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	base := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "branch", "selected", base)
	trackingRef := upstreamFFConfigure(t, root, "remote")
	gitTest(t, root, "symbolic-ref", trackingRef, "refs/heads/master")
	_, err := branchSyncTestCommands(t, root).InspectUpstreamFastForward("selected")
	assert.Error(t, err)
}

func TestUpstreamFastForwardCheckedOutBranchIsCurrent(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	base, ahead, _ := upstreamFFHistory(t, root)
	gitTest(t, root, "branch", "selected", base)
	gitTest(t, root, "config", "branch.selected.remote", ".")
	gitTest(t, root, "config", "branch.selected.merge", "refs/heads/upstream")
	gitTest(t, root, "update-ref", "refs/heads/upstream", ahead)
	gitTest(t, root, "checkout", "-q", "selected")
	got, err := branchSyncTestCommands(t, root).InspectUpstreamFastForward("selected")
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, got.Current())
	assert.Equal(t, UpstreamFastForwardBehind, got.State)
	assert.Equal(t, ahead, got.UpstreamOID)
}

func TestUpstreamFastForwardInspectionChangesNothing(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	base, ahead, _ := upstreamFFHistory(t, root)
	gitTest(t, root, "branch", "selected", ahead)
	trackingRef := upstreamFFConfigure(t, root, "remote")
	gitTest(t, root, "update-ref", trackingRef, base)
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty\n"), 0o644)
	beforeRefs := gitTest(t, root, "for-each-ref", "--format=%(refname) %(objectname)")
	beforeHead := gitTest(t, root, "symbolic-ref", "HEAD")
	beforeIndex := mustRead(t, filepath.Join(root, ".git", "index"))
	commands := branchSyncTestCommands(t, root)
	for range 3 {
		got, err := commands.InspectUpstreamFastForward("selected")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, UpstreamFastForwardUpToDate, got.State)
	}
	assert.Equal(t, beforeRefs, gitTest(t, root, "for-each-ref", "--format=%(refname) %(objectname)"))
	assert.Equal(t, beforeHead, gitTest(t, root, "symbolic-ref", "HEAD"))
	assert.Equal(t, beforeIndex, mustRead(t, filepath.Join(root, ".git", "index")))
}
