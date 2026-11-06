package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

// These tests drive g m n against a real repository. The fixture's main is at
// f.oid. Each test makes origin/main and a feature branch with two unpushed
// commits, so the move has a real source, upstream and base.

// moveCommitsUI records menus as well as the popup/worker fake.
type moveCommitsUI struct {
	*checkoutByNameUI
	menus []types.CreateMenuOptions
	repo  *moveCommitsRepoState
}

func (ui *moveCommitsUI) Menu(opts types.CreateMenuOptions) error {
	ui.menus = append(ui.menus, opts)
	return nil
}

// State adds the repository-state accessor that the merge helper writes to.
func (ui *moveCommitsUI) State() types.IStateAccessor {
	return &moveCommitsStateAccessor{checkoutStateAccessor: &ui.checkoutByNameUI.checkoutStateUI.state, repo: ui.repo}
}

// moveCommitsRepoState records whether an operation was started by the app.
type moveCommitsRepoState struct {
	types.IRepoStateAccessor
	started bool
}

func (s *moveCommitsRepoState) SetMergeOrRebaseStartedInLazygit(started bool) { s.started = started }

type moveCommitsStateAccessor struct {
	*checkoutStateAccessor
	repo *moveCommitsRepoState
}

func (s *moveCommitsStateAccessor) GetRepoState() types.IRepoStateAccessor { return s.repo }

// moveCommitsWireGit gives the fixture the Git command groups the flow uses,
// with the real branch loader for the base-branch lookup.
func moveCommitsWireGit(f *namedCheckoutFixture, ui *moveCommitsUI) {
	osCmd := oscommands.NewDummyOSCommand()
	paths := git_commands.MockRepoPaths(f.root)
	backend := git_commands.NewGitCommon(common.NewDummyCommon(), &git_commands.GitVersion{Major: 2, Minor: 51, Patch: 1}, osCmd.Cmd, osCmd, paths, nil, nil)
	commit := git_commands.NewCommitCommands(backend)
	ui.git = &commands.GitCommand{
		Branch: git_commands.NewBranchCommands(backend),
		Commit: commit,
		Rebase: git_commands.NewRebaseCommands(backend, commit, nil),
		Status: git_commands.NewStatusCommands(backend),
		Stash:  git_commands.NewStashCommands(backend, nil, git_commands.NewWorkingTreeCommands(backend, nil, nil)),
		Loaders: commands.Loaders{
			BranchLoader: git_commands.NewBranchLoader(common.NewDummyCommon(), backend, osCmd.Cmd,
				func() (git_commands.BranchInfo, error) { return git_commands.BranchInfo{}, nil },
				func(*models.Branch) (bool, error) { return false, nil }, nil),
		},
		RepoPaths: paths,
	}
}

// moveCommitsFixture makes feature with two unpushed commits above its upstream,
// which is origin/feature at f.oid. origin/main is f.oid too, so main is the
// base. The checked-out branch is feature.
type moveCommitsFixture struct {
	*namedCheckoutFixture
	ui       *moveCommitsUI
	helper   *RefsHelper
	unpushed []string // feature's unpushed commits, oldest first
}

func newMoveCommitsFixture(t *testing.T) *moveCommitsFixture {
	t.Helper()
	f, base := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	ui := &moveCommitsUI{checkoutByNameUI: base, repo: &moveCommitsRepoState{}}
	f.c.IGuiCommon = ui
	moveCommitsWireGit(f, ui)

	namedCheckoutGit(t, f.root, "remote", "add", "origin", "https://example.invalid/repo.git")
	namedCheckoutGit(t, f.root, "update-ref", "refs/remotes/origin/main", f.oid)
	namedCheckoutGit(t, f.root, "config", "branch.main.remote", "origin")
	namedCheckoutGit(t, f.root, "config", "branch.main.merge", "refs/heads/main")
	namedCheckoutGit(t, f.root, "checkout", "-q", "-b", "feature", f.oid)
	namedCheckoutGit(t, f.root, "update-ref", "refs/remotes/origin/feature", f.oid)
	namedCheckoutGit(t, f.root, "config", "branch.feature.remote", "origin")
	namedCheckoutGit(t, f.root, "config", "branch.feature.merge", "refs/heads/feature")
	m := &moveCommitsFixture{namedCheckoutFixture: f, ui: ui, helper: &RefsHelper{c: f.c, rebaseHelper: NewMergeAndRebaseHelper(f.c)}}
	for _, name := range []string{"unpushed-1", "unpushed-2"} {
		namedCheckoutWrite(t, f.root, name, name+"\n")
		namedCheckoutGit(t, f.root, "add", name)
		namedCheckoutGit(t, f.root, "commit", "-qm", name)
		m.unpushed = append(m.unpushed, namedCheckoutGit(t, f.root, "rev-parse", "HEAD"))
	}
	ui.model.Branches = []*models.Branch{{Name: "feature", Head: true}}
	ui.model.MainBranches = git_commands.NewMainBranches(common.NewDummyCommon(), oscommands.NewDummyOSCommand().Cmd)
	return m
}

func (m *moveCommitsFixture) head(t *testing.T) (ref, oid string) {
	t.Helper()
	return checkoutByNameHead(t, m.namedCheckoutFixture)
}

// begin starts g m n and runs its capture worker, so the choice is on screen.
func (m *moveCommitsFixture) begin(t *testing.T) {
	t.Helper()
	if err := m.helper.MoveCommitsToNewBranch(); err != nil {
		t.Fatal(err)
	}
	if err := drainCheckoutByName(t, m.ui.checkoutByNameUI); err != nil {
		t.Fatal(err)
	}
}

// choose picks the variant: the stacked item, or the base item, of the menu.
// A non-main branch always shows a menu; the main branch confirms the stacked
// variant instead.
func (m *moveCommitsFixture) choose(t *testing.T, variant git_commands.MoveCommitsVariant) {
	t.Helper()
	if len(m.ui.menus) != 1 {
		t.Fatalf("expected one menu, got %d", len(m.ui.menus))
	}
	items := m.ui.menus[0].Items
	idx := 1
	if variant == git_commands.MoveCommitsFromBase {
		idx = 0
	}
	if err := items[idx].OnPress(); err != nil {
		t.Fatal(err)
	}
}

// name answers the name prompt and runs the write worker and its UI callbacks.
func (m *moveCommitsFixture) name(t *testing.T, name string) error {
	t.Helper()
	if len(m.ui.prompts) == 0 {
		t.Fatal("expected a name prompt")
	}
	prompt := m.ui.prompts[len(m.ui.prompts)-1]
	if err := prompt.HandleConfirm(name); err != nil {
		return err
	}
	return drainCheckoutByName(t, m.ui.checkoutByNameUI)
}

func (m *moveCommitsFixture) run(t *testing.T, variant git_commands.MoveCommitsVariant, name string) error {
	t.Helper()
	m.begin(t)
	m.choose(t, variant)
	return m.name(t, name)
}

// assertUntouched checks that feature keeps its unpushed commits, HEAD is still
// headRef, and nothing was stashed.
func (m *moveCommitsFixture) assertUntouched(t *testing.T, headRef string) {
	t.Helper()
	assert.Equal(t, m.unpushed[1], namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/feature"), "feature must keep its commits")
	ref, _ := m.head(t)
	assert.Equal(t, headRef, ref)
	assert.Empty(t, namedCheckoutGit(t, m.root, "stash", "list"), "nothing may be stashed")
}

func TestMoveCommitsStackedMovesCommitsAndChecksOutNewBranch(t *testing.T) {
	m := newMoveCommitsFixture(t)
	if err := m.run(t, git_commands.MoveCommitsStacked, "moved"); err != nil {
		t.Fatal(err)
	}
	ref, head := m.head(t)
	assert.Equal(t, "refs/heads/moved", ref)
	assert.Equal(t, m.unpushed[1], head, "the new branch holds the captured HEAD")
	assert.Equal(t, m.oid, namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/feature"), "feature is back at its upstream")
	assert.Equal(t, "origin/feature", namedCheckoutGit(t, m.root, "rev-parse", "--abbrev-ref", "feature@{upstream}"))
	// config --get exits 1 when the key is absent, which is the expected result here.
	assert.Error(t, exec.Command("git", "-C", m.root, "config", "--get", "branch.moved.remote").Run(), "the new branch sets no tracking")
	assert.Empty(t, namedCheckoutGit(t, m.root, "stash", "list"))
}

func TestMoveCommitsFromBaseCherryPicksOntoNewBranchAndMovesOld(t *testing.T) {
	m := newMoveCommitsFixture(t)
	if err := m.run(t, git_commands.MoveCommitsFromBase, "moved"); err != nil {
		t.Fatal(err)
	}
	ref, _ := m.head(t)
	assert.Equal(t, "refs/heads/moved", ref)
	assert.Equal(t, m.oid, namedCheckoutGit(t, m.root, "rev-parse", "moved~2"), "the new branch starts at the base")
	assert.Equal(t, "2", namedCheckoutGit(t, m.root, "rev-list", "--count", m.oid+"..moved"))
	assert.Equal(t, "unpushed-2", namedCheckoutGit(t, m.root, "log", "-1", "--format=%s", "moved"))
	assert.Equal(t, m.oid, namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/feature"))
	assert.Empty(t, namedCheckoutGit(t, m.root, "stash", "list"))
}

func TestMoveCommitsFromMainBranchConfirmsStackedVariant(t *testing.T) {
	m := newMoveCommitsFixture(t)
	namedCheckoutGit(t, m.root, "checkout", "-q", "main")
	namedCheckoutGit(t, m.root, "merge", "-q", "--ff-only", "feature")
	m.ui.model.Branches = []*models.Branch{{Name: "main", Head: true}}
	m.begin(t)
	if len(m.ui.confirms) != 1 || len(m.ui.menus) != 0 {
		t.Fatalf("main should confirm, got %d confirms and %d menus", len(m.ui.confirms), len(m.ui.menus))
	}
	if err := m.ui.confirms[0].HandleConfirm(); err != nil {
		t.Fatal(err)
	}
	if err := m.name(t, "moved"); err != nil {
		t.Fatal(err)
	}
	ref, head := m.head(t)
	assert.Equal(t, "refs/heads/moved", ref)
	assert.Equal(t, m.unpushed[1], head)
	assert.Equal(t, m.oid, namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/main"))
}

func TestMoveCommitsRefusesExistingNameWithNothingChanged(t *testing.T) {
	for _, variant := range []git_commands.MoveCommitsVariant{git_commands.MoveCommitsStacked, git_commands.MoveCommitsFromBase} {
		t.Run(map[git_commands.MoveCommitsVariant]string{git_commands.MoveCommitsStacked: "stacked", git_commands.MoveCommitsFromBase: "base"}[variant], func(t *testing.T) {
			m := newMoveCommitsFixture(t)
			headRef, _ := m.head(t)
			other := namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/other")
			err := m.run(t, variant, "other")
			if err == nil || !strings.Contains(err.Error(), "already exists") {
				t.Fatalf("expected an existing-name refusal, got %v", err)
			}
			assert.Equal(t, other, namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/other"), "the existing branch must not move")
			m.assertUntouched(t, headRef)
		})
	}
}

func TestMoveCommitsRefusesOptionLookingNameWithNothingChanged(t *testing.T) {
	for _, variant := range []git_commands.MoveCommitsVariant{git_commands.MoveCommitsStacked, git_commands.MoveCommitsFromBase} {
		t.Run(map[git_commands.MoveCommitsVariant]string{git_commands.MoveCommitsStacked: "stacked", git_commands.MoveCommitsFromBase: "base"}[variant], func(t *testing.T) {
			m := newMoveCommitsFixture(t)
			headRef, _ := m.head(t)
			err := m.run(t, variant, "--show-current")
			if err == nil || !strings.Contains(err.Error(), "'-'") {
				t.Fatalf("expected an option-name refusal, got %v", err)
			}
			m.assertUntouched(t, headRef)
			assert.Empty(t, namedCheckoutGit(t, m.root, "for-each-ref", "--format=%(refname)", "refs/heads/--show-current"), "no branch named by the option")
		})
	}
}

func TestMoveCommitsRefusesTrackedChangeBeforeAnyPromptAndNeverStashes(t *testing.T) {
	m := newMoveCommitsFixture(t)
	namedCheckoutWrite(t, m.root, "tracked", "edited\n")
	err := m.helper.MoveCommitsToNewBranch()
	if err == nil {
		err = drainCheckoutByName(t, m.ui.checkoutByNameUI)
	}
	if err == nil || !strings.Contains(err.Error(), "tracked files") {
		t.Fatalf("expected a tracked-changes refusal, got %v", err)
	}
	assert.Empty(t, m.ui.menus)
	assert.Empty(t, m.ui.prompts)
	assert.Empty(t, namedCheckoutGit(t, m.root, "stash", "list"))
	assert.Equal(t, "edited", strings.TrimSpace(readFileForMoveCommits(t, filepath.Join(m.root, "tracked"))))
}

func TestMoveCommitsRefusesTrackedChangeMadeDuringPromptWithNothingChanged(t *testing.T) {
	m := newMoveCommitsFixture(t)
	headRef, _ := m.head(t)
	m.begin(t)
	m.choose(t, git_commands.MoveCommitsStacked)
	namedCheckoutWrite(t, m.root, "tracked", "edited during prompt\n")
	err := m.name(t, "moved")
	if err == nil || !strings.Contains(err.Error(), "tracked files") {
		t.Fatalf("expected a tracked-changes refusal, got %v", err)
	}
	assert.Empty(t, namedCheckoutGit(t, m.root, "branch", "--list", "moved"))
	assert.Empty(t, namedCheckoutGit(t, m.root, "stash", "list"))
	assert.Equal(t, "edited during prompt", strings.TrimSpace(readFileForMoveCommits(t, filepath.Join(m.root, "tracked"))))
	m.assertUntouched(t, headRef)
}

func TestMoveCommitsAllowsUntrackedFiles(t *testing.T) {
	m := newMoveCommitsFixture(t)
	namedCheckoutWrite(t, m.root, "scratch", "untracked\n")
	if err := m.run(t, git_commands.MoveCommitsStacked, "moved"); err != nil {
		t.Fatal(err)
	}
	ref, head := m.head(t)
	assert.Equal(t, "refs/heads/moved", ref)
	assert.Equal(t, m.unpushed[1], head)
	assert.Equal(t, "untracked\n", readFileForMoveCommits(t, filepath.Join(m.root, "scratch")))
}

func TestMoveCommitsRefusesHeadChangedAfterCaptureWithNothingChanged(t *testing.T) {
	m := newMoveCommitsFixture(t)
	m.begin(t)
	m.choose(t, git_commands.MoveCommitsStacked)
	// Another branch is checked out while the name prompt is open.
	namedCheckoutGit(t, m.root, "checkout", "-q", "main")
	err := m.name(t, "moved")
	if err == nil || !strings.Contains(err.Error(), "changed since the action started") {
		t.Fatalf("expected a changed-branch refusal, got %v", err)
	}
	assert.Empty(t, namedCheckoutGit(t, m.root, "branch", "--list", "moved"))
	assert.Equal(t, m.unpushed[1], namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/feature"))
	ref, _ := checkoutByNameHead(t, m.namedCheckoutFixture)
	assert.Equal(t, "refs/heads/main", ref)
}

func TestMoveCommitsRefusesUpstreamMovedAfterCapture(t *testing.T) {
	m := newMoveCommitsFixture(t)
	m.begin(t)
	m.choose(t, git_commands.MoveCommitsFromBase)
	namedCheckoutGit(t, m.root, "update-ref", "refs/remotes/origin/feature", m.unpushed[0])
	err := m.name(t, "moved")
	if err == nil || !strings.Contains(err.Error(), "changed since the action started") {
		t.Fatalf("expected a changed-upstream refusal, got %v", err)
	}
	assert.Empty(t, namedCheckoutGit(t, m.root, "branch", "--list", "moved"))
	assert.Equal(t, m.unpushed[1], namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/feature"))
}

// A failed write leaves the commits on a branch, and the flow says which.

func TestMoveCommitsStackedCheckoutFailureKeepsCommitsOnBothBranches(t *testing.T) {
	m := newMoveCommitsFixture(t)
	// An index lock makes the checkout step fail after the create step succeeded.
	namedCheckoutWrite(t, m.root, filepath.Join(".git", "index.lock"), "")
	err := m.run(t, git_commands.MoveCommitsStacked, "moved")
	if err == nil || !strings.Contains(err.Error(), "could not check it out") {
		t.Fatalf("expected a checkout-step failure, got %v", err)
	}
	assert.Equal(t, m.unpushed[1], namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/moved"))
	assert.Equal(t, m.unpushed[1], namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/feature"), "feature keeps its commits")
	assert.Empty(t, namedCheckoutGit(t, m.root, "stash", "list"))
}

func TestMoveCommitsStackedOldBranchLockedKeepsCommitsOnNewBranch(t *testing.T) {
	m := newMoveCommitsFixture(t)
	// A lock on the old branch's loose ref makes the final move fail, after the
	// new branch exists and is checked out.
	namedCheckoutWrite(t, m.root, filepath.Join(".git", "refs", "heads", "feature.lock"), "")
	err := m.run(t, git_commands.MoveCommitsStacked, "moved")
	if err == nil || !strings.Contains(err.Error(), "was not moved") {
		t.Fatalf("expected an old-branch failure, got %v", err)
	}
	ref, head := m.head(t)
	assert.Equal(t, "refs/heads/moved", ref)
	assert.Equal(t, m.unpushed[1], head)
	assert.Equal(t, m.unpushed[1], namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/feature"), "feature keeps its commits")
}

func TestMoveCommitsFromBaseCreateFailureCreatesNothing(t *testing.T) {
	m := newMoveCommitsFixture(t)
	namedCheckoutWrite(t, m.root, filepath.Join(".git", "index.lock"), "")
	headRef, _ := m.head(t)
	err := m.run(t, git_commands.MoveCommitsFromBase, "moved")
	if err == nil || !strings.Contains(err.Error(), "Nothing was changed") {
		t.Fatalf("expected a create-step failure, got %v", err)
	}
	assert.Empty(t, namedCheckoutGit(t, m.root, "branch", "--list", "moved"))
	m.assertUntouched(t, headRef)
}

func TestMoveCommitsFromBaseConflictKeepsOldBranchUntilContinue(t *testing.T) {
	m := newMoveCommitsFixtureWithConflict(t)
	err := m.run(t, git_commands.MoveCommitsFromBase, "moved")
	if err != nil {
		t.Fatal(err)
	}
	// The pick stopped on a conflict: the operation is in progress and the old
	// branch still has its commit.
	assert.FileExists(t, filepath.Join(m.root, ".git", "CHERRY_PICK_HEAD"))
	assert.Equal(t, m.unpushed[0], namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/feature"), "feature must keep its commit while the pick is stopped")
	assert.NotEmpty(t, m.ui.menus, "the conflict menu is offered")

	// Resolve and continue with Git, then report the operation as ended with
	// continue. The settlement callback runs the old branch move.
	namedCheckoutWrite(t, m.root, "conflict.txt", "resolved\n")
	namedCheckoutGit(t, m.root, "add", "conflict.txt")
	namedCheckoutGitEnv(t, m.root, []string{"GIT_EDITOR=true"}, "cherry-pick", "--continue")
	m.helper.rebaseHelper.settleOperationFinished(REBASE_OPTION_CONTINUE, nil)
	if err := drainCheckoutByName(t, m.ui.checkoutByNameUI); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, namedCheckoutGit(t, m.root, "rev-parse", "origin/feature"), namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/feature"), "after continue feature is at its upstream")
	assert.Equal(t, "1", namedCheckoutGit(t, m.root, "rev-list", "--count", m.conflictBase+"..moved"), "the one unpushed commit is on the new branch")
	ref, _ := m.head(t)
	assert.Equal(t, "refs/heads/moved", ref)
}

func TestMoveCommitsFromBaseAbortKeepsOldBranchUntouched(t *testing.T) {
	m := newMoveCommitsFixtureWithConflict(t)
	if err := m.run(t, git_commands.MoveCommitsFromBase, "moved"); err != nil {
		t.Fatal(err)
	}
	namedCheckoutGit(t, m.root, "cherry-pick", "--abort")
	m.helper.rebaseHelper.settleOperationFinished(REBASE_OPTION_ABORT, nil)
	if err := drainCheckoutByName(t, m.ui.checkoutByNameUI); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, m.unpushed[0], namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/feature"), "an abort must not move the old branch")
}

func TestMoveCommitsFromBaseOldBranchMovedWhileStoppedIsReported(t *testing.T) {
	m := newMoveCommitsFixtureWithConflict(t)
	if err := m.run(t, git_commands.MoveCommitsFromBase, "moved"); err != nil {
		t.Fatal(err)
	}
	namedCheckoutWrite(t, m.root, "conflict.txt", "resolved\n")
	namedCheckoutGit(t, m.root, "add", "conflict.txt")
	namedCheckoutGitEnv(t, m.root, []string{"GIT_EDITOR=true"}, "cherry-pick", "--continue")
	// The old branch moves while the operation is in progress; the move must refuse.
	namedCheckoutGit(t, m.root, "update-ref", "refs/heads/feature", m.conflictBase)
	m.helper.rebaseHelper.settleOperationFinished(REBASE_OPTION_CONTINUE, nil)
	err := drainCheckoutByName(t, m.ui.checkoutByNameUI)
	if err == nil || !strings.Contains(err.Error(), "was not moved") || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("expected a changed-old-branch report, got %v", err)
	}
	assert.Equal(t, m.conflictBase, namedCheckoutGit(t, m.root, "rev-parse", "refs/heads/feature"))
}

// newMoveCommitsFixtureWithConflict makes the base variant's first pick conflict:
// main changes conflict.txt after the feature commit was made from the common
// ancestor. conflictBase is main's tip, where the new branch starts.
func newMoveCommitsFixtureWithConflict(t *testing.T) *moveCommitsFixtureConflict {
	t.Helper()
	f, base := checkoutByNameFixture(t)
	enterCheckoutFixtureRepo(t, f)
	ui := &moveCommitsUI{checkoutByNameUI: base, repo: &moveCommitsRepoState{}}
	f.c.IGuiCommon = ui
	moveCommitsWireGit(f, ui)
	namedCheckoutGit(t, f.root, "remote", "add", "origin", "https://example.invalid/repo.git")

	namedCheckoutWrite(t, f.root, "conflict.txt", "ancestor\n")
	namedCheckoutGit(t, f.root, "add", "conflict.txt")
	namedCheckoutGit(t, f.root, "commit", "-qm", "ancestor")
	ancestor := namedCheckoutGit(t, f.root, "rev-parse", "HEAD")

	namedCheckoutGit(t, f.root, "checkout", "-q", "-b", "feature", ancestor)
	namedCheckoutGit(t, f.root, "update-ref", "refs/remotes/origin/feature", ancestor)
	namedCheckoutGit(t, f.root, "config", "branch.feature.remote", "origin")
	namedCheckoutGit(t, f.root, "config", "branch.feature.merge", "refs/heads/feature")
	namedCheckoutWrite(t, f.root, "conflict.txt", "feature\n")
	namedCheckoutGit(t, f.root, "commit", "-qam", "feature conflicting edit")
	unpushed := namedCheckoutGit(t, f.root, "rev-parse", "HEAD")

	namedCheckoutGit(t, f.root, "checkout", "-q", "main")
	namedCheckoutWrite(t, f.root, "conflict.txt", "main\n")
	namedCheckoutGit(t, f.root, "commit", "-qam", "main conflicting edit")
	conflictBase := namedCheckoutGit(t, f.root, "rev-parse", "HEAD")
	namedCheckoutGit(t, f.root, "checkout", "-q", "feature")

	m := &moveCommitsFixture{namedCheckoutFixture: f, ui: ui, helper: &RefsHelper{c: f.c, rebaseHelper: NewMergeAndRebaseHelper(f.c)}}
	m.unpushed = []string{unpushed}
	ui.model.Branches = []*models.Branch{{Name: "feature", Head: true}}
	ui.model.MainBranches = git_commands.NewMainBranches(common.NewDummyCommon(), oscommands.NewDummyOSCommand().Cmd)
	return &moveCommitsFixtureConflict{moveCommitsFixture: m, conflictBase: conflictBase}
}

type moveCommitsFixtureConflict struct {
	*moveCommitsFixture
	conflictBase string
}

func readFileForMoveCommits(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// namedCheckoutGitEnv runs git with extra environment, for steps that must not
// open an editor.
func namedCheckoutGitEnv(t *testing.T, root string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
