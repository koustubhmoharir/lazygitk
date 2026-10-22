package git_commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func branchSyncTestCommands(t *testing.T, root string) *SyncCommands {
	t.Helper()
	osCommand := oscommands.NewDummyOSCommand()
	common := NewGitCommon(common.NewDummyCommon(), nil, rootedTestBuilder{inner: osCommand.Cmd, root: root}, osCommand, MockRepoPaths(root), nil, nil)
	return NewSyncCommands(common)
}

func branchSyncTestRepo(t *testing.T) (string, string) {
	t.Helper()
	root := initHistoricalRepo(t)
	oid := branchPlacementTestCommit(t, root, "tracked", "initial\n", "initial")
	return root, oid
}

func TestBranchSyncInfoSelectedBranchMetadata(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	gitTest(t, root, "branch", "selected")
	gitTest(t, root, "remote", "add", "pullside", "https://example.invalid/pull.git")
	gitTest(t, root, "remote", "add", "pushside", "https://example.invalid/push.git")
	gitTest(t, root, "config", "remote.pullside.fetch", "+refs/heads/*:refs/cache/pullside/*")
	gitTest(t, root, "update-ref", "refs/cache/pullside/reviewed", oid)
	gitTest(t, root, "config", "branch.selected.remote", "pullside")
	gitTest(t, root, "config", "branch.selected.merge", "refs/heads/reviewed")
	gitTest(t, root, "config", "remote.pushside.push", "refs/heads/*:refs/for/review/*")
	gitTest(t, root, "config", "branch.selected.pushRemote", "pushside")
	// The selected branch is not HEAD; dirty state must not affect inspection.
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty\n"), 0o644)
	gitTest(t, root, "add", "tracked")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("unstaged after staging\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "untracked"), []byte("untracked\n"), 0o644)
	beforeHead := gitTest(t, root, "rev-parse", "HEAD")
	beforeIndex := gitTest(t, root, "write-tree")

	info, err := branchSyncTestCommands(t, root).InspectBranchSync("selected")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "selected", info.Name)
	assert.Equal(t, "refs/heads/selected", info.FullRef)
	assert.Equal(t, oid, info.OID)
	assert.Equal(t, gitTest(t, root, "symbolic-ref", "HEAD"), info.HeadRef)
	assert.Equal(t, beforeHead, info.HeadOID)
	assert.Equal(t, "pullside", info.UpstreamRemote)
	assert.Equal(t, "refs/heads/reviewed", info.UpstreamRef)
	assert.Equal(t, "refs/cache/pullside/reviewed", info.UpstreamTrackingRef)
	assert.Equal(t, "pushside", info.PushRemote)
	assert.Equal(t, "refs/for/review/selected", info.PushRef)
	assert.Equal(t, beforeHead, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Equal(t, beforeIndex, gitTest(t, root, "write-tree"))
	assert.Equal(t, "unstaged after staging\n", string(mustRead(t, filepath.Join(root, "tracked"))))
	assert.FileExists(t, filepath.Join(root, "untracked"))

	gitTest(t, root, "config", "branch.selected.merge", "refs/heads/deleted")
	gitTest(t, root, "update-ref", "refs/cache/pullside/deleted", oid)
	gitTest(t, root, "update-ref", "-d", "refs/cache/pullside/deleted", oid)
	info, err = branchSyncTestCommands(t, root).InspectBranchSync("selected")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "pullside", info.UpstreamRemote)
	assert.Equal(t, "refs/heads/deleted", info.UpstreamRef)
	assert.Equal(t, "refs/cache/pullside/deleted", info.UpstreamTrackingRef)

	second := branchPlacementTestCommit(t, root, "tracked", "second\n", "second")
	gitTest(t, root, "update-ref", "refs/heads/selected", second, oid)
	gitTest(t, root, "config", "branch.selected.pushRemote", "pullside")
	info, err = branchSyncTestCommands(t, root).InspectBranchSync("selected")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, second, info.OID, "source OID is freshly read")
	assert.Equal(t, "pullside", info.PushRemote, "push configuration is freshly read")
}

func TestBranchSyncInfoNoUpstreamAndExactParent(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	gitTest(t, root, "branch", "topic/child")
	headBranch := strings.TrimPrefix(gitTest(t, root, "symbolic-ref", "HEAD"), "refs/heads/")
	info, err := branchSyncTestCommands(t, root).InspectBranchSync(headBranch)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, oid, info.OID)
	assert.Empty(t, info.UpstreamRemote)
	assert.Empty(t, info.UpstreamRef)
	assert.Empty(t, info.UpstreamTrackingRef)
	assert.Empty(t, info.PushRemote)
	assert.Empty(t, info.PushRef)
	_, err = branchSyncTestCommands(t, root).InspectBranchSync("topic")
	if err == nil {
		t.Fatal("prefix-matched child refs must not be treated as a parent branch")
	}
}

func TestBranchSyncInfoRejectsInvalidNamesAndAliases(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	commands := branchSyncTestCommands(t, root)
	for _, name := range []string{"@{-1}", "HEAD~1", "missing", ""} {
		_, err := commands.InspectBranchSync(name)
		assert.Error(t, err, "name %q", name)
	}
	headRef := gitTest(t, root, "symbolic-ref", "HEAD")
	gitTest(t, root, "symbolic-ref", "refs/heads/alias", headRef)
	_, err := commands.InspectBranchSync("alias")
	if err == nil {
		t.Fatal("expected alias rejection")
	}
	gitTest(t, root, "symbolic-ref", "refs/heads/broken", "refs/heads/absent")
	_, err = commands.InspectBranchSync("broken")
	if err == nil {
		t.Fatal("expected broken alias rejection")
	}
	_, err = branchSyncTestCommands(t, root).InspectBranchSync(strings.TrimPrefix(gitTest(t, root, "symbolic-ref", "HEAD"), "refs/heads/"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, oid, gitTest(t, root, "rev-parse", "HEAD"))
}

func TestBranchSyncInfoReservationAndDetachedHead(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	gitTest(t, root, "branch", "reserved")
	linked := filepath.Join(t.TempDir(), "linked")
	gitTest(t, root, "worktree", "add", "-q", linked, "reserved")
	gitTest(t, root, "checkout", "--detach", "-q", oid)
	info, err := branchSyncTestCommands(t, root).InspectBranchSync("reserved")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "", info.HeadRef)
	assert.Equal(t, oid, info.HeadOID)
	assert.NotEmpty(t, info.Reservation.GitDir)
	assert.False(t, info.Reservation.Current)
	assert.Equal(t, "checkout", info.Reservation.Kind)
}

func TestBranchSyncInfoParserMalformedDuplicateAndChildren(t *testing.T) {
	row := func(ref, oid, remote string) string {
		return strings.Join([]string{ref, oid, "", remote, "", "", ""}, "\x00") + "\x00\n"
	}
	good := row("refs/heads/topic", strings.Repeat("a", 40), "")
	got, err := parseBranchSyncInfoRows(row("refs/heads/topic/child", strings.Repeat("b", 40), "")+good, "refs/heads/topic")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "refs/heads/topic", got.FullRef)
	withNewline, err := parseBranchSyncInfoRows(row("refs/heads/topic", strings.Repeat("a", 40), "remote\nname"), "refs/heads/topic")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "remote\nname", withNewline.UpstreamRemote)
	_, err = parseBranchSyncInfoRows("broken\n", "refs/heads/topic")
	if err == nil {
		t.Fatal("expected malformed parser row rejection")
	}
	_, err = parseBranchSyncInfoRows(good+good, "refs/heads/topic")
	if err == nil {
		t.Fatal("expected duplicate parser row rejection")
	}
	_, err = parseBranchSyncInfoRows("", "refs/heads/topic")
	if err == nil {
		t.Fatal("expected missing parser row rejection")
	}
}

func TestBranchSyncInfoRequiresCommit(t *testing.T) {
	root, _ := branchSyncTestRepo(t)
	blob := gitTest(t, root, "rev-parse", "HEAD:tracked")
	refPath := filepath.Join(root, ".git", "refs", "heads", "blob")
	if err := os.MkdirAll(filepath.Dir(refPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(refPath, []byte(blob+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, blob, gitTest(t, root, "rev-parse", "refs/heads/blob"), "the deliberately corrupt ref must be visible to Git")
	_, err := branchSyncTestCommands(t, root).InspectBranchSync("blob")
	assert.ErrorContains(t, err, "does not point to a commit")
	assert.FileExists(t, refPath)
}

func TestBranchSyncInfoSelectedBranchWithUnbornCheckout(t *testing.T) {
	root, oid := branchSyncTestRepo(t)
	selected := strings.TrimPrefix(gitTest(t, root, "symbolic-ref", "HEAD"), "refs/heads/")
	gitTest(t, root, "symbolic-ref", "HEAD", "refs/heads/unborn")
	info, err := branchSyncTestCommands(t, root).InspectBranchSync(selected)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, oid, info.OID)
	assert.Equal(t, "refs/heads/unborn", info.HeadRef)
	assert.Empty(t, info.HeadOID)
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
