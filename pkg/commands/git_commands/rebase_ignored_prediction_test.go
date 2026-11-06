package git_commands

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// predictionRebaseRepo builds the 07-F1 shape. Base tracks old/a and ignores
// new/b, new/c, new/unrelated and elsewhere. Target (the current branch's
// destination) renames old/ to new/, while topic, branched from base, adds old/b.
// With existingDestination, base already has new/x and target renames only old/a.
// The topic checkout is left with the ignored files written in place.
func predictionRebaseRepo(t *testing.T, existingDestination bool) (root, target string) {
	t.Helper()
	root = initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("/new/b\n/new/c\n/new/unrelated\n/elsewhere\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "old", "a"), []byte("base\n"), 0o644)
	if existingDestination {
		writeTestFile(t, filepath.Join(root, "new", "x"), []byte("x\n"), 0o644)
	}
	gitTest(t, root, "add", "-A")
	gitTest(t, root, "commit", "-qm", "base")
	baseBranch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	writeTestFile(t, filepath.Join(root, "old", "b"), []byte("topic\n"), 0o644)
	gitTest(t, root, "add", "old/b")
	gitTest(t, root, "commit", "-qm", "addition")
	gitTest(t, root, "checkout", "-q", baseBranch)
	if existingDestination {
		gitTest(t, root, "mv", "old/a", "new/a")
	} else {
		gitTest(t, root, "mv", "old", "new")
	}
	gitTest(t, root, "commit", "-qm", "rename")
	target = gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "checkout", "-q", "topic")
	return root, target
}

// predictionRebaseCommands uses the given Git version; nil keeps the shared test
// helper's unknown version, which must fall back to the rename-destination rule.
func predictionRebaseCommands(t *testing.T, root string, version *GitVersion) *RebaseCommands {
	t.Helper()
	commands := selectedCommitRebaseCommands(t, root)
	commands.version = version
	return commands
}

func currentGitVersion(t *testing.T) *GitVersion {
	t.Helper()
	version, err := GetGitVersion(oscommands.NewDummyOSCommand())
	selectedRebaseRequireNoError(t, err)
	return version
}

func writeIgnoredPredictionFiles(t *testing.T, root string, paths ...string) {
	t.Helper()
	for _, path := range paths {
		writeTestFile(t, filepath.Join(root, path), []byte("PRECIOUS "+path+"\n"), 0o600)
	}
}

// mergeTreeFailingBuilder makes every merge-tree invocation fail for a reason other
// than conflicts, as a Git that rejects the options would.
type mergeTreeFailingBuilder struct {
	rootedTestBuilder
}

func (builder mergeTreeFailingBuilder) New(args []string) *oscommands.CmdObj {
	if slices.Contains(args, "merge-tree") {
		args = []string{"git", "merge-tree", "--no-such-option"}
	}
	return builder.rootedTestBuilder.New(args)
}

func TestRebasePredictionDirectoryRenameFlagsCreatedPath(t *testing.T) {
	for _, mode := range []string{"default", "true"} {
		t.Run("merge.directoryRenames="+mode, func(t *testing.T) {
			root, target := predictionRebaseRepo(t, false)
			if mode == "true" {
				gitTest(t, root, "config", "merge.directoryRenames", "true")
			}
			writeIgnoredPredictionFiles(t, root, "new/b")
			plan, err := predictionRebaseCommands(t, root, currentGitVersion(t)).PrepareSelectedCommitRebase(target, true)
			selectedRebaseRequireNoError(t, err)
			assert.Equal(t, []string{"new/b"}, plan.IgnoredFiles())
		})
	}
}

func TestRebasePredictionExistingDestinationDirectory(t *testing.T) {
	root, target := predictionRebaseRepo(t, true)
	writeIgnoredPredictionFiles(t, root, "new/b", "new/unrelated")
	plan, err := predictionRebaseCommands(t, root, currentGitVersion(t)).PrepareSelectedCommitRebase(target, true)
	selectedRebaseRequireNoError(t, err)
	// The replay creates new/b and no other path, so only that file collides.
	assert.Equal(t, []string{"new/b"}, plan.IgnoredFiles())
}

func TestRebasePredictionMergeCommitRangeUsesRenameDestinationRule(t *testing.T) {
	root, target := predictionRebaseRepo(t, false)
	gitTest(t, root, "checkout", "-qb", "side", "master~1")
	writeTestFile(t, filepath.Join(root, "side"), []byte("side\n"), 0o644)
	gitTest(t, root, "add", "side")
	gitTest(t, root, "commit", "-qm", "side")
	gitTest(t, root, "checkout", "-q", "topic")
	gitTest(t, root, "merge", "-q", "--no-ff", "-m", "merge side", "side")
	writeIgnoredPredictionFiles(t, root, "new/b", "new/unrelated", "elsewhere")
	plan, err := predictionRebaseCommands(t, root, currentGitVersion(t)).PrepareSelectedCommitRebase(target, true)
	selectedRebaseRequireNoError(t, err)
	// Merges have no exact chain here, so the mirrored rule applies. It flags the
	// mirrored replayed path only: the unrelated file in the renamed directory and
	// the file outside it are not flagged.
	assert.Equal(t, []string{"new/b"}, plan.IgnoredFiles())
}

func TestRebasePredictionFallbackForOldGitAndMergeTreeFailure(t *testing.T) {
	cases := map[string]func(*RebaseCommands, string){
		"git before merge-tree --merge-base": func(commands *RebaseCommands, _ string) {
			commands.version = &GitVersion{Major: 2, Minor: 39, Patch: 0}
		},
		"merge-tree fails without conflicts": func(commands *RebaseCommands, root string) {
			commands.cmd = mergeTreeFailingBuilder{rootedTestBuilder{inner: oscommands.NewDummyOSCommand().Cmd, root: root}}
		},
	}
	for name, configure := range cases {
		t.Run(name, func(t *testing.T) {
			root, target := predictionRebaseRepo(t, false)
			writeIgnoredPredictionFiles(t, root, "new/b", "new/unrelated", "elsewhere")
			commands := predictionRebaseCommands(t, root, currentGitVersion(t))
			configure(commands, root)
			plan, err := commands.PrepareSelectedCommitRebase(target, true)
			selectedRebaseRequireNoError(t, err)
			assert.Equal(t, []string{"new/b"}, plan.IgnoredFiles())
		})
	}
}

func TestRebasePredictionLeavesRepositoryUnchanged(t *testing.T) {
	root, target := predictionRebaseRepo(t, false)
	writeIgnoredPredictionFiles(t, root, "new/b")
	before := mergePreviewSnapshot(t, root)
	objectsBefore := gitTest(t, root, "count-objects", "-v")
	_, err := predictionRebaseCommands(t, root, currentGitVersion(t)).PrepareSelectedCommitRebase(target, true)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, objectsBefore, gitTest(t, root, "count-objects", "-v"))
	assert.Equal(t, before, mergePreviewSnapshot(t, root))
}

func TestRebasePredictionConflictedStepContinuesChain(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("/new/c\n/new/unrelated\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "old", "a"), []byte("base\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "f"), []byte("base\n"), 0o644)
	gitTest(t, root, "add", "-A")
	gitTest(t, root, "commit", "-qm", "base")
	baseBranch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	writeTestFile(t, filepath.Join(root, "f"), []byte("topic\n"), 0o644)
	gitTest(t, root, "commit", "-qam", "conflicting edit")
	writeTestFile(t, filepath.Join(root, "old", "c"), []byte("topic\n"), 0o644)
	gitTest(t, root, "add", "old/c")
	gitTest(t, root, "commit", "-qm", "addition after conflict")
	gitTest(t, root, "checkout", "-q", baseBranch)
	writeTestFile(t, filepath.Join(root, "f"), []byte("main\n"), 0o644)
	gitTest(t, root, "mv", "old", "new")
	gitTest(t, root, "commit", "-qam", "rename and edit")
	target := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "checkout", "-q", "topic")
	writeIgnoredPredictionFiles(t, root, "new/c", "new/unrelated")
	plan, err := predictionRebaseCommands(t, root, currentGitVersion(t)).PrepareSelectedCommitRebase(target, true)
	selectedRebaseRequireNoError(t, err)
	// The exact chain continues past the conflicted step and predicts new/c. The
	// fallback would also flag new/unrelated, so the exact list shows the chain ran.
	assert.Equal(t, []string{"new/c"}, plan.IgnoredFiles())
}

func TestRebasePredictionChainLimitAndReorderedReplay(t *testing.T) {
	root, target := predictionRebaseRepo(t, false)
	writeIgnoredPredictionFiles(t, root, "new/b", "new/unrelated")
	commands := predictionRebaseCommands(t, root, currentGitVersion(t))
	replay := rebaseReplay{destination: target, upstream: target, head: "HEAD", logRevs: []string{target + "..HEAD"}}

	t.Run("replay beyond the cap uses the rename-destination rule", func(t *testing.T) {
		// The range holds one commit, which is over a cap of zero.
		_, ok := commands.mergeTreeChainPaths(replay, 0)
		assert.False(t, ok)
		_, ok = commands.mergeTreeChainPaths(replay, rebaseMergeTreeChainLimit)
		assert.True(t, ok)
	})
	t.Run("reordered replay adds the rename-destination rule", func(t *testing.T) {
		replay.reordered = true
		paths, _, err := commands.rebaseCandidateRisks(replay)
		selectedRebaseRequireNoError(t, err)
		assert.Equal(t, []string{"new/b"}, paths)
	})
}

func TestCompoundStagedRenameMirrorsReplayedPaths(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("/new/m\n/new/unrelated\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "old", "a"), []byte("base\n"), 0o644)
	gitTest(t, root, "add", "-A")
	gitTest(t, root, "commit", "-qm", "base")
	base := gitTest(t, root, "rev-parse", "HEAD")
	// The replayed commit adds old/m. The staged rename moves old/a, and so old/ itself, to new/.
	writeTestFile(t, filepath.Join(root, "old", "m"), []byte("middle\n"), 0o644)
	gitTest(t, root, "add", "old/m")
	gitTest(t, root, "commit", "-qm", "middle")
	middle := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "rm", "-q", "old/a")
	writeTestFile(t, filepath.Join(root, "new", "a"), []byte("base\n"), 0o644)
	gitTest(t, root, "add", "new/a")
	writeIgnoredPredictionFiles(t, root, "new/m", "new/unrelated")
	hashPool := &utils.StringPool{}
	commits := []*models.Commit{
		models.NewCommit(hashPool, models.NewCommitOpts{Hash: middle}),
		models.NewCommit(hashPool, models.NewCommitOpts{Hash: base}),
	}
	err := predictionRebaseCommands(t, root, currentGitVersion(t)).CheckCompoundRebaseIgnoredRisks(commits, 0, true)
	riskErr := requireIgnoredRebaseRisk(t, err)
	// The rename moves the replayed old/m to new/m. The unrelated file is not flagged.
	assert.Equal(t, []string{"new/m"}, riskErr.Paths)
}

func TestRebasePredictionRenameIntoRootFlagsOnlyMirroredPaths(t *testing.T) {
	root := initHistoricalRepo(t)
	writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("/b\n/build/\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "old", "a"), []byte("base\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "old", "x"), []byte("base\n"), 0o644)
	gitTest(t, root, "add", "-A")
	gitTest(t, root, "commit", "-qm", "base")
	baseBranch := gitTest(t, root, "branch", "--show-current")
	gitTest(t, root, "checkout", "-qb", "topic")
	writeTestFile(t, filepath.Join(root, "old", "b"), []byte("topic\n"), 0o644)
	gitTest(t, root, "add", "old/b")
	gitTest(t, root, "commit", "-qm", "addition")
	gitTest(t, root, "checkout", "-q", baseBranch)
	// Renaming old/ into the top level moves old/b to b. Nothing else is at risk.
	gitTest(t, root, "mv", "old/a", "a")
	gitTest(t, root, "mv", "old/x", "x")
	gitTest(t, root, "commit", "-qm", "rename into root")
	target := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "checkout", "-q", "topic")
	writeIgnoredPredictionFiles(t, root, "b", "build/out.o")
	plan, err := predictionRebaseCommands(t, root, currentGitVersion(t)).PrepareSelectedCommitRebase(target, true)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, []string{"b"}, plan.IgnoredFiles())
}

func TestRebasePredictionChainSkippedWithoutMirroredCollision(t *testing.T) {
	root, target := predictionRebaseRepo(t, false)
	commands := predictionRebaseCommands(t, root, currentGitVersion(t))

	writeIgnoredPredictionFiles(t, root, "build/out.o")
	before := mergeTreeChainRuns.Load()
	plan, err := commands.PrepareSelectedCommitRebase(target, true)
	selectedRebaseRequireNoError(t, err)
	assert.Empty(t, plan.IgnoredFiles())
	assert.Equal(t, before, mergeTreeChainRuns.Load(), "the gate must skip the merge-tree chain")

	writeIgnoredPredictionFiles(t, root, "new/b")
	before = mergeTreeChainRuns.Load()
	plan, err = commands.PrepareSelectedCommitRebase(target, true)
	selectedRebaseRequireNoError(t, err)
	assert.Equal(t, []string{"new/b"}, plan.IgnoredFiles())
	assert.Equal(t, before+1, mergeTreeChainRuns.Load(), "a mirrored collision runs the chain once")
}
