package git_commands

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

type branchPushHistory struct {
	root, bare, base, mastertip, selected string
}

func branchPushRequireNoError(t *testing.T, err error, details ...any) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v %v", err, details)
	}
}

func branchPushTestHistory(t *testing.T) branchPushHistory {
	t.Helper()
	root, base, mastertip := transactionHistory(t)
	bare := filepath.Join(t.TempDir(), "origin.git")
	gitTest(t, filepath.Dir(bare), "init", "--bare", "-q", bare)
	gitTest(t, root, "branch", "selected", base)
	gitTest(t, root, "push", "-q", bare, "refs/heads/selected:refs/heads/selected", "refs/heads/master:refs/heads/master")
	gitTest(t, root, "remote", "add", "origin", bare)
	gitTest(t, root, "update-ref", "refs/remotes/origin/selected", base)
	gitTest(t, root, "checkout", "-q", "selected")
	selected := branchPlacementTestCommit(t, root, "separatefile", "selected content\n", "selected")
	gitTest(t, root, "checkout", "-q", "master")
	return branchPushHistory{root: root, bare: bare, base: base, mastertip: mastertip, selected: selected}
}

func (h branchPushHistory) options() BranchPushOptions {
	return BranchPushOptions{Remote: "origin", DestinationRef: "refs/heads/selected"}
}

func branchPushPlan(t *testing.T, h branchPushHistory, options BranchPushOptions) (*SyncCommands, BranchPushPlan) {
	t.Helper()
	commands := branchSyncTestCommands(t, h.root)
	plan, err := commands.PrepareBranchPush("selected", h.selected, options)
	branchPushRequireNoError(t, err)
	return commands, plan
}

func runBranchPush(t *testing.T, commands *SyncCommands, plan BranchPushPlan) ([]byte, error) {
	t.Helper()
	cmd, err := commands.BranchPushCommand(plan, nil)
	if err != nil {
		return nil, err
	}
	return cmd.GetCmd().CombinedOutput()
}

func branchPushGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	branchPushRequireNoError(t, err, string(output))
	return strings.TrimSpace(string(output))
}

func branchPushConfig(t *testing.T, root, key string) string {
	t.Helper()
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			return ""
		}
		t.Fatalf("read config %s: %v", key, err)
	}
	return strings.TrimSpace(string(output))
}

func TestBranchPushSelectedBranchPreservesWorktreeAndIgnoresPushExpansion(t *testing.T) {
	h := branchPushTestHistory(t)
	writeTestFile(t, filepath.Join(h.root, "tracked"), []byte("staged change\n"), 0o644)
	gitTest(t, h.root, "add", "tracked")
	writeTestFile(t, filepath.Join(h.root, "tracked"), []byte("unstaged change\n"), 0o644)
	writeTestFile(t, filepath.Join(h.root, "binary-untracked"), []byte{0, 1, 2, 255}, 0o600)
	gitTest(t, h.root, "tag", "unrelated-tag", h.base)
	gitTest(t, h.root, "tag", "-a", "annotated-tag", "-m", "tag", h.base)
	gitTest(t, h.root, "config", "push.followTags", "true")
	gitTest(t, h.root, "config", "push.autoSetupRemote", "true")
	gitTest(t, h.root, "config", "remote.origin.push", "refs/heads/master:refs/heads/other")
	head := gitTest(t, h.root, "rev-parse", "HEAD")
	index := gitTest(t, h.root, "write-tree")
	rawHead := mustRead(t, filepath.Join(h.root, ".git", "HEAD"))
	rawIndex := mustRead(t, filepath.Join(h.root, ".git", "index"))
	status := gitTest(t, h.root, "status", "--porcelain=v1", "-z")
	tag := gitTest(t, h.root, "rev-parse", "refs/tags/unrelated-tag")
	commands, plan := branchPushPlan(t, h, h.options())
	cmd, err := commands.BranchPushCommand(plan, nil)
	branchPushRequireNoError(t, err)
	assert.Equal(t, []string{"git", "-c", "push.autoSetupRemote=false", "push", "--porcelain", "--no-follow-tags", "--recurse-submodules=no", "--", "origin", "refs/heads/selected:refs/heads/selected"}, cmd.Args())
	assert.True(t, cmd.LockRetriesDisabled())
	assert.Equal(t, oscommands.PROMPT, cmd.GetCredentialStrategy())
	output, err := cmd.GetCmd().CombinedOutput()
	branchPushRequireNoError(t, err, string(output))
	assert.Equal(t, rawHead, mustRead(t, filepath.Join(h.root, ".git", "HEAD")), "raw HEAD bytes remain unchanged immediately after push")
	assert.Equal(t, rawIndex, mustRead(t, filepath.Join(h.root, ".git", "index")), "raw index bytes remain unchanged immediately after push")
	assert.Equal(t, h.selected, branchPushGit(t, h.bare, "rev-parse", "refs/heads/selected"))
	assert.Equal(t, h.mastertip, branchPushGit(t, h.bare, "rev-parse", "refs/heads/master"))
	assert.Empty(t, branchPushGit(t, h.bare, "for-each-ref", "--format=%(refname)", "refs/heads/other", "refs/tags"))
	assert.Equal(t, h.selected, gitTest(t, h.root, "rev-parse", "refs/heads/selected"))
	assert.Equal(t, h.mastertip, gitTest(t, h.root, "rev-parse", "refs/heads/master"))
	assert.Equal(t, head, gitTest(t, h.root, "rev-parse", "HEAD"))
	assert.Equal(t, index, gitTest(t, h.root, "write-tree"))
	assert.Equal(t, status, gitTest(t, h.root, "status", "--porcelain=v1", "-z"))
	assert.Equal(t, tag, gitTest(t, h.root, "rev-parse", "refs/tags/unrelated-tag"))
	assert.Equal(t, h.selected, gitTest(t, h.root, "rev-parse", "refs/remotes/origin/selected"))
	assert.Equal(t, "unstaged change\n", string(mustRead(t, filepath.Join(h.root, "tracked"))))
	assert.Equal(t, "staged change", gitTest(t, h.root, "show", ":tracked"))
	assert.Equal(t, []byte{0, 1, 2, 255}, mustRead(t, filepath.Join(h.root, "binary-untracked")))
	assert.Empty(t, branchPushConfig(t, h.root, "branch.selected.remote"))
}

func TestBranchPushSetUpstreamAndLease(t *testing.T) {
	t.Run("set upstream only for selected branch", func(t *testing.T) {
		h := branchPushTestHistory(t)
		opts := h.options()
		opts.SetUpstream = true
		commands, plan := branchPushPlan(t, h, opts)
		out, err := runBranchPush(t, commands, plan)
		branchPushRequireNoError(t, err, string(out))
		assert.Equal(t, "origin", gitTest(t, h.root, "config", "branch.selected.remote"))
		assert.Equal(t, "refs/heads/selected", gitTest(t, h.root, "config", "branch.selected.merge"))
		assert.Empty(t, branchPushConfig(t, h.root, "branch.master.remote"))
	})
	t.Run("stale lease rejects remote advance", func(t *testing.T) {
		h := branchPushTestHistory(t)
		opts := h.options()
		opts.ForceWithLease = true
		commands, plan := branchPushPlan(t, h, opts)
		cmd, err := commands.BranchPushCommand(plan, nil)
		branchPushRequireNoError(t, err)
		remoteTip := branchPlacementTestCommit(t, h.root, "remote-only", "advance\n", "remote advance")
		gitTest(t, h.root, "push", "-q", h.bare, remoteTip+":refs/heads/selected")
		_, err = cmd.GetCmd().CombinedOutput()
		assert.Error(t, err)
		assert.Equal(t, remoteTip, branchPushGit(t, h.bare, "rev-parse", "refs/heads/selected"))
	})
	t.Run("changed source and lease fail preparation", func(t *testing.T) {
		h := branchPushTestHistory(t)
		gitTest(t, h.root, "update-ref", "refs/heads/selected", h.base, h.selected)
		_, err := branchSyncTestCommands(t, h.root).PrepareBranchPush("selected", h.selected, h.options())
		assert.ErrorIs(t, err, ErrBranchSyncChanged)
	})
}

func TestBranchPushLeaseChoicesAndValidation(t *testing.T) {
	h := branchPushTestHistory(t)
	commands := branchSyncTestCommands(t, h.root)
	opts := h.options()
	opts.ForceWithLease = true
	gitTest(t, h.root, "config", "remote.origin.fetch", "+refs/heads/*:refs/cache/origin/*")
	_, err := commands.PrepareBranchPush("selected", h.selected, opts)
	assert.True(t, errors.Is(err, ErrBranchPushLeaseChoice))
	gitTest(t, h.root, "update-ref", "refs/tags/reviewed", h.base)
	opts.LeaseRef = "refs/tags/reviewed"
	plan, err := commands.PrepareBranchPush("selected", h.selected, opts)
	branchPushRequireNoError(t, err)
	assert.Equal(t, h.base, plan.LeaseOID())
	cmd, err := commands.BranchPushCommand(plan, nil)
	branchPushRequireNoError(t, err)
	assert.Contains(t, strings.Join(cmd.Args(), " "), "--force-with-lease=refs/heads/selected:"+h.base)
	gitTest(t, h.root, "update-ref", "refs/tags/reviewed", h.selected, h.base)
	_, err = commands.BranchPushCommand(plan, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
}

func TestBranchPushExplicitLeasesAndConfigurationGuards(t *testing.T) {
	h := branchPushTestHistory(t)
	commands := branchSyncTestCommands(t, h.root)
	for _, dest := range []string{"selected", "refs/tags", "refs/tags/no", "refs/notes", "refs/notes/review", "refs/heads/"} {
		opts := h.options()
		opts.DestinationRef = dest
		_, err := commands.PrepareBranchPush("selected", h.selected, opts)
		assert.Error(t, err, "destination %q", dest)
	}
	gitTest(t, h.root, "config", "remote.origin.mirror", "true")
	_, err := commands.PrepareBranchPush("selected", h.selected, h.options())
	assert.ErrorContains(t, err, "mirror")
	gitTest(t, h.root, "config", "--unset", "remote.origin.mirror")
	gitTest(t, h.root, "config", "remote.origin.fetch", "+refs/heads/*:refs/heads/*")
	_, err = commands.PrepareBranchPush("selected", h.selected, h.options())
	assert.Error(t, err)
	assert.Equal(t, h.base, branchPushGit(t, h.bare, "rev-parse", "refs/heads/selected"), "rejected fetch mappings leave the remote destination untouched")
	gitTest(t, h.root, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	gitTest(t, h.root, "symbolic-ref", "refs/remotes/origin/selected-alias", "refs/remotes/origin/selected")
	gitTest(t, h.root, "config", "remote.origin.fetch", "+refs/heads/selected:refs/remotes/origin/selected-alias")
	_, err = commands.PrepareBranchPush("selected", h.selected, h.options())
	assert.ErrorContains(t, err, "symbolic alias")
	assert.Equal(t, h.base, branchPushGit(t, h.bare, "rev-parse", "refs/heads/selected"), "rejected symbolic aliases leave the remote destination untouched")
	gitTest(t, h.root, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	gitTest(t, h.root, "config", "--replace-all", "remote.origin.pushurl", h.bare)
	gitTest(t, h.root, "config", "--add", "remote.origin.pushurl", h.bare+"-second")
	plan, err := commands.PrepareBranchPush("selected", h.selected, h.options())
	branchPushRequireNoError(t, err)
	if _, err := commands.BranchPushCommand(plan, nil); !assert.ErrorIs(t, err, ErrBranchPushMultipleURLs) {
		return
	}
	approved := plan.WithMultipleURLsApproved()
	cmd, err := commands.BranchPushCommand(approved, nil)
	assert.NoError(t, err)
	if cmd != nil {
		assert.Contains(t, strings.Join(cmd.GetCmd().Args, " "), "refs/heads/selected:refs/heads/selected")
	}
	gitTest(t, h.root, "config", "--replace-all", "remote.origin.pushurl", h.bare)
	_, err = commands.BranchPushCommand(approved, nil)
	assert.Error(t, err)

	gitTest(t, h.root, "config", "--unset-all", "remote.origin.pushurl")
	force := h.options()
	force.ForceWithLease = true
	gitTest(t, h.root, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	_, err = commands.PrepareBranchPush("selected", h.selected, force)
	assert.NoError(t, err)
	commands.UserConfig().Git.DisableForcePushing = true
	_, err = commands.PrepareBranchPush("selected", h.selected, force)
	assert.ErrorContains(t, err, "force pushing is disabled")
}

func TestBranchPushBackendAndCurrentBranchGuards(t *testing.T) {
	h := branchPushTestHistory(t)
	commands, plan := branchPushPlan(t, h, h.options())
	foreign := branchSyncTestCommands(t, h.root)
	_, err := foreign.BranchPushCommand(plan, nil)
	assert.ErrorContains(t, err, "not prepared by this repository instance")

	opts := h.options()
	opts.RequireCurrent = true
	_, err = commands.PrepareBranchPush("selected", h.selected, opts)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	selectedOpts := h.options()
	selectedPlan, err := commands.PrepareBranchPush("selected", h.selected, selectedOpts)
	branchPushRequireNoError(t, err)
	cmd, err := commands.BranchPushCommand(selectedPlan, nil)
	branchPushRequireNoError(t, err)
	assert.Contains(t, strings.Join(cmd.GetCmd().Args, " "), "refs/heads/selected:refs/heads/selected")
}

func TestBranchPushPreviousUpstreamSnapshot(t *testing.T) {
	for _, setUpstream := range []bool{true, false} {
		name := "plain push preserves previous upstream"
		if setUpstream {
			name = "changed upstream invalidates approval"
		}
		t.Run(name, func(t *testing.T) {
			h := branchPushTestHistory(t)
			gitTest(t, h.root, "remote", "add", "old-origin", h.bare)
			gitTest(t, h.root, "config", "branch.selected.remote", "old-origin")
			merges := []string{"refs/heads/old-one", "refs/heads/old-two"}
			for _, ref := range merges {
				gitTest(t, h.root, "config", "--add", "branch.selected.merge", ref)
			}
			gitTest(t, h.root, "config", "push.autoSetupRemote", "true")
			opts := h.options()
			opts.SetUpstream = setUpstream
			commands, plan := branchPushPlan(t, h, opts)
			if setUpstream {
				assert.Equal(t, "old-origin", plan.PreviousRemote())
				assert.Equal(t, merges, plan.PreviousMergeRefs())
				returned := plan.PreviousMergeRefs()
				returned[0] = "refs/heads/mutated"
				assert.Equal(t, merges, plan.PreviousMergeRefs())
				gitTest(t, h.root, "config", "--replace-all", "branch.selected.merge", "refs/heads/changed", "^refs/heads/old-two$")
			}
			configPath := filepath.Join(h.root, ".git", "config")
			before := mustRead(t, configPath)
			cmd, err := commands.BranchPushCommand(plan, nil)
			if setUpstream {
				assert.ErrorIs(t, err, ErrBranchSyncChanged)
				assert.Nil(t, cmd)
				assert.Equal(t, h.base, branchPushGit(t, h.bare, "rev-parse", "refs/heads/selected"))
			} else {
				branchPushRequireNoError(t, err)
				output, err := cmd.GetCmd().CombinedOutput()
				branchPushRequireNoError(t, err, string(output))
				assert.Equal(t, h.selected, branchPushGit(t, h.bare, "rev-parse", "refs/heads/selected"))
			}
			assert.Equal(t, before, mustRead(t, configPath), "push planning/execution must preserve the previously configured upstream")
		})
	}
}

func TestBranchPushMultipleRealURLsApproval(t *testing.T) {
	h := branchPushTestHistory(t)
	second := filepath.Join(t.TempDir(), "second.git")
	gitTest(t, filepath.Dir(second), "init", "--bare", "-q", second)
	gitTest(t, h.root, "push", "-q", second, h.base+":refs/heads/selected", h.mastertip+":refs/heads/master")
	for _, bare := range []string{h.bare, second} {
		gitTest(t, bare, "update-ref", "refs/heads/unrelated", h.mastertip)
		gitTest(t, h.root, "config", "--add", "remote.origin.pushurl", bare)
	}
	commands, plan := branchPushPlan(t, h, h.options())
	urls := []string{h.bare, second}
	assert.Equal(t, urls, plan.ConfiguredPushURLs())
	returned := plan.ConfiguredPushURLs()
	returned[0] = "mutated"
	assert.Equal(t, urls, plan.ConfiguredPushURLs())
	cmd, err := commands.BranchPushCommand(plan, nil)
	assert.ErrorIs(t, err, ErrBranchPushMultipleURLs)
	assert.Nil(t, cmd)
	for _, bare := range urls {
		assert.Equal(t, h.base, branchPushGit(t, bare, "rev-parse", "refs/heads/selected"))
	}
	approved := plan.WithMultipleURLsApproved()
	output, err := runBranchPush(t, commands, approved)
	branchPushRequireNoError(t, err, string(output))
	for _, bare := range urls {
		assert.Equal(t, "refs/heads/master\nrefs/heads/selected\nrefs/heads/unrelated", branchPushGit(t, bare, "for-each-ref", "--format=%(refname)"))
		assert.Equal(t, h.selected, branchPushGit(t, bare, "rev-parse", "refs/heads/selected"))
		assert.Equal(t, h.mastertip, branchPushGit(t, bare, "rev-parse", "refs/heads/master"))
		assert.Equal(t, h.mastertip, branchPushGit(t, bare, "rev-parse", "refs/heads/unrelated"))
	}
	gitTest(t, h.root, "config", "--unset-all", "remote.origin.pushurl")
	gitTest(t, h.root, "config", "--add", "remote.origin.pushurl", h.bare)
	gitTest(t, h.root, "config", "--add", "remote.origin.pushurl", second+"/.")
	cmd, err = commands.BranchPushCommand(approved, nil)
	assert.ErrorIs(t, err, ErrBranchSyncChanged)
	assert.Nil(t, cmd)
}

func TestBranchPushPreparedInputsMustRemainCurrent(t *testing.T) {
	for _, name := range []string{"automatic lease", "remote URL", "URL rewrite", "force disabled"} {
		t.Run(name, func(t *testing.T) {
			h := branchPushTestHistory(t)
			rewriteKey := "url." + h.bare + ".insteadOf"
			if name == "URL rewrite" {
				gitTest(t, h.root, "config", rewriteKey, "reviewed:")
			}
			opts := h.options()
			opts.ForceWithLease = true
			commands, plan := branchPushPlan(t, h, opts)
			assert.Equal(t, "refs/remotes/origin/selected", plan.LeaseRef())
			assert.Equal(t, h.base, plan.LeaseOID())
			switch name {
			case "automatic lease":
				gitTest(t, h.root, "update-ref", plan.LeaseRef(), h.mastertip, h.base)
			case "remote URL":
				gitTest(t, h.root, "config", "remote.origin.url", h.bare+"/.")
			case "URL rewrite":
				gitTest(t, h.root, "config", rewriteKey, "changed:")
			case "force disabled":
				commands.UserConfig().Git.DisableForcePushing = true
			}
			cmd, err := commands.BranchPushCommand(plan, nil)
			if name == "force disabled" {
				assert.ErrorContains(t, err, "force pushing is disabled")
			} else {
				assert.ErrorIs(t, err, ErrBranchSyncChanged)
			}
			assert.Nil(t, cmd)
			assert.Equal(t, h.base, branchPushGit(t, h.bare, "rev-parse", "refs/heads/selected"))
			if name == "automatic lease" {
				assert.Equal(t, h.mastertip, gitTest(t, h.root, "rev-parse", plan.LeaseRef()), "command validation must not fetch and replace the changed lease")
			}
		})
	}
}

func TestBranchPushExcludedLeaseAndSubmoduleChoices(t *testing.T) {
	t.Run("exact negative fetch mapping needs an explicit reviewed lease", func(t *testing.T) {
		h := branchPushTestHistory(t)
		gitTest(t, h.root, "config", "--add", "remote.origin.fetch", "^refs/heads/selected")
		gitTest(t, h.root, "update-ref", "refs/tags/reviewed", h.base)
		commands := branchSyncTestCommands(t, h.root)
		opts := h.options()
		opts.ForceWithLease = true
		_, err := commands.PrepareBranchPush("selected", h.selected, opts)
		assert.ErrorIs(t, err, ErrBranchPushLeaseChoice)
		opts.LeaseRef = "refs/tags/reviewed"
		plan, err := commands.PrepareBranchPush("selected", h.selected, opts)
		branchPushRequireNoError(t, err)
		assert.Equal(t, opts.LeaseRef, plan.LeaseRef())
		assert.Equal(t, h.base, plan.LeaseOID())
		output, err := runBranchPush(t, commands, plan)
		branchPushRequireNoError(t, err, string(output))
		assert.Equal(t, h.selected, branchPushGit(t, h.bare, "rev-parse", "refs/heads/selected"))
	})
	t.Run("submodule choice is explicit", func(t *testing.T) {
		h := branchPushTestHistory(t)
		commands := branchSyncTestCommands(t, h.root)
		opts := h.options()
		opts.Submodules = BranchPushSubmodules(255)
		_, err := commands.PrepareBranchPush("selected", h.selected, opts)
		assert.Error(t, err)
		opts.Submodules = BranchPushCheckSubmodules
		plan, err := commands.PrepareBranchPush("selected", h.selected, opts)
		branchPushRequireNoError(t, err)
		cmd, err := commands.BranchPushCommand(plan, nil)
		branchPushRequireNoError(t, err)
		assert.Contains(t, cmd.Args(), "--recurse-submodules=check")
		assert.Equal(t, h.base, branchPushGit(t, h.bare, "rev-parse", "refs/heads/selected"))
	})
}
