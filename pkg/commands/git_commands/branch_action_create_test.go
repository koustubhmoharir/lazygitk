package git_commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestBranchActionCreate(t *testing.T) {
	root, first, second := transactionHistory(t)
	branches := transactionCommands(t, root)
	gitTest(t, root, "reset", "--hard", first)
	target, err := branches.PrepareBranchActionTarget("refs/heads/master", first)
	if err != nil {
		t.Fatal(err)
	}
	for _, tracking := range []BranchActionCreateTracking{BranchActionCreateNoTracking, BranchActionCreateConfiguredTracking, BranchActionCreateDirectTracking} {
		name := map[BranchActionCreateTracking]string{BranchActionCreateNoTracking: "created-no-track", BranchActionCreateConfiguredTracking: "created-configured", BranchActionCreateDirectTracking: "created-direct"}[tracking]
		plan, err := branches.PrepareBranchActionCreate(target, name, BranchActionCreateOptions{Tracking: tracking})
		if err != nil {
			t.Fatal(err)
		}
		cmd, err := branches.BranchActionCreateCommand(plan)
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/heads/"+name))
	}
	assert.Error(t, func() error {
		_, e := branches.PrepareBranchActionCreate(target, "created-no-track", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
		return e
	}())
	_, err = branches.PrepareBranchActionTarget("refs/heads/master", second)
	assert.ErrorIs(t, err, ErrBranchActionChanged)
	plan, err := branches.PrepareBranchActionCreate(target, "=literal", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := branches.BranchActionCreateCommand(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/heads/=literal"))
}

func TestBranchActionCreateCheckoutDirtyAndCollision(t *testing.T) {
	root, first, _ := transactionHistory(t)
	branches := transactionCommands(t, root)
	gitTest(t, root, "reset", "--hard", first)
	target, err := branches.PrepareBranchActionTarget("refs/heads/master", first)
	if err != nil {
		t.Fatal(err)
	}
	// Harmless dirty and untracked files travel through Git's native checkout.
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("staged dirty bytes\n"), 0o644)
	gitTest(t, root, "add", "tracked")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("unstaged dirty bytes\n"), 0o644)
	if err := os.WriteFile(filepath.Join(root, "dirty"), []byte("work"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "loose"), []byte("untracked"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := branches.PrepareBranchActionCreate(target, "created-checkout", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking, Checkout: true})
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := branches.BranchActionCreateCommand(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "created-checkout", gitTest(t, root, "branch", "--show-current"))
	assert.Equal(t, "unstaged dirty bytes\n", string(mustReadBranchRefFile(t, filepath.Join(root, "tracked"))))
	assert.Equal(t, "staged dirty bytes", gitTest(t, root, "show", ":tracked"))
	assert.FileExists(t, filepath.Join(root, "dirty"))
	assert.FileExists(t, filepath.Join(root, "loose"))
	_, err = branches.PrepareBranchActionCreate(target, "bad name", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
	assert.Error(t, err)
}

func TestBranchActionCreateStalePlans(t *testing.T) {
	t.Run("source", func(t *testing.T) {
		root, first, second := transactionHistory(t)
		commands := transactionCommands(t, root)
		gitTest(t, root, "checkout", "-q", "--detach", first)
		gitTest(t, root, "branch", "source", first)
		target, err := commands.PrepareBranchActionTarget("refs/heads/source", first)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := commands.PrepareBranchActionCreate(target, "new-branch", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
		if err != nil {
			t.Fatal(err)
		}
		gitTest(t, root, "branch", "-f", "source", second)
		_, err = commands.BranchActionCreateCommand(plan)
		assert.Error(t, err)
	})
	t.Run("head", func(t *testing.T) {
		root, first, second := transactionHistory(t)
		commands := transactionCommands(t, root)
		gitTest(t, root, "reset", "--hard", first)
		target, err := commands.PrepareBranchActionTarget(first, first)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := commands.PrepareBranchActionCreate(target, "new-branch", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
		if err != nil {
			t.Fatal(err)
		}
		gitTest(t, root, "checkout", "-q", "--detach", second)
		_, err = commands.BranchActionCreateCommand(plan)
		assert.ErrorIs(t, err, ErrBranchActionChanged)
	})
	t.Run("config", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		commands := transactionCommands(t, root)
		gitTest(t, root, "reset", "--hard", first)
		target, err := commands.PrepareBranchActionTarget("refs/heads/master", first)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := commands.PrepareBranchActionCreate(target, "new-branch", BranchActionCreateOptions{Tracking: BranchActionCreateConfiguredTracking})
		if err != nil {
			t.Fatal(err)
		}
		gitTest(t, root, "config", "branch.autoSetupMerge", "always")
		_, err = commands.BranchActionCreateCommand(plan)
		assert.ErrorIs(t, err, ErrBranchActionChanged)
	})
	t.Run("destination", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		commands := transactionCommands(t, root)
		gitTest(t, root, "reset", "--hard", first)
		target, err := commands.PrepareBranchActionTarget("refs/heads/master", first)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := commands.PrepareBranchActionCreate(target, "new-branch", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
		if err != nil {
			t.Fatal(err)
		}
		gitTest(t, root, "branch", "new-branch", first)
		_, err = commands.BranchActionCreateCommand(plan)
		assert.Error(t, err)
	})
	t.Run("foreign and zero", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		otherRoot, _, _ := transactionHistory(t)
		commands := transactionCommands(t, root)
		gitTest(t, root, "reset", "--hard", first)
		target, err := commands.PrepareBranchActionTarget("refs/heads/master", first)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := commands.PrepareBranchActionCreate(target, "new-branch", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
		if err != nil {
			t.Fatal(err)
		}
		_, err = transactionCommands(t, otherRoot).BranchActionCreateCommand(plan)
		assert.Error(t, err)
		_, err = commands.BranchActionCreateCommand(BranchActionCreatePlan{})
		assert.Error(t, err)
	})
}

func TestBranchActionCreateRefusesOperationReservations(t *testing.T) {
	root, first, _ := transactionHistory(t)
	commands := transactionCommands(t, root)
	target, err := commands.PrepareBranchActionTarget(first, first)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git", "rebase-merge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, ".git", "rebase-merge", "head-name"), []byte("refs/heads/reserved-target\n"), 0o644)
	_, err = commands.PrepareBranchActionCreate(target, "reserved-target", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
	assert.Error(t, err)
}

func TestBranchActionCreatePointerOnlyPreservesRawState(t *testing.T) {
	root, first, _ := transactionHistory(t)
	commands := transactionCommands(t, root)
	gitTest(t, root, "reset", "--hard", first)
	target, err := commands.PrepareBranchActionTarget("refs/heads/master", first)
	if err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "tracked")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty tracked bytes\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "untracked"), []byte("raw untracked\x00bytes"), 0o600)
	head, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	tracked, err := os.ReadFile(filepath.Join(root, "tracked"))
	if err != nil {
		t.Fatal(err)
	}
	untracked, err := os.ReadFile(filepath.Join(root, "untracked"))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := commands.PrepareBranchActionCreate(target, "pointer-only", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := commands.BranchActionCreateCommand(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	gotHead, _ := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	gotIndex, _ := os.ReadFile(filepath.Join(root, ".git", "index"))
	gotTracked, _ := os.ReadFile(filepath.Join(root, "tracked"))
	gotUntracked, _ := os.ReadFile(filepath.Join(root, "untracked"))
	assert.Equal(t, head, gotHead)
	assert.Equal(t, index, gotIndex)
	assert.Equal(t, tracked, gotTracked)
	assert.Equal(t, untracked, gotUntracked)
}

func TestBranchActionCreateNativeTrackingPolicies(t *testing.T) {
	for _, policy := range []struct {
		name, config, destination string
		wantTracked               bool
	}{{"true", "true", "different-name", true}, {"always", "always", "different-name", true}, {"inherit", "inherit", "different-name", true}, {"simple-match", "simple", "source", true}, {"simple-mismatch", "simple", "different-name", false}, {"configured-no-track", "false", "different-name", false}} {
		t.Run(policy.name, func(t *testing.T) {
			root, first, _ := transactionHistory(t)
			gitTest(t, root, "remote", "add", "origin", root)
			gitTest(t, root, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
			gitTest(t, root, "update-ref", "refs/remotes/origin/source", first)
			gitTest(t, root, "config", "branch.autoSetupMerge", policy.config)
			gitTest(t, root, "reset", "--hard", first)
			gitTest(t, root, "branch", "--set-upstream-to=refs/remotes/origin/source", "master")
			commands := transactionCommands(t, root)
			source := "refs/remotes/origin/source"
			if policy.config == "inherit" {
				source = "refs/heads/master"
			}
			target, err := commands.PrepareBranchActionTarget(source, first)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := commands.PrepareBranchActionCreate(target, policy.destination, BranchActionCreateOptions{Tracking: BranchActionCreateConfiguredTracking})
			if err != nil {
				t.Fatal(err)
			}
			cmd, err := commands.BranchActionCreateCommand(plan)
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Run(); err != nil {
				t.Fatal(err)
			}
			remoteErr := runBranchCreateGit(root, "config", "--get", "branch."+policy.destination+".remote")
			assert.Equal(t, policy.wantTracked, remoteErr == nil)
			if policy.wantTracked {
				assert.Equal(t, "origin", gitTest(t, root, "config", "--get", "branch."+policy.destination+".remote"))
			}
		})
	}
	t.Run("explicit no-track overrides configured true", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		gitTest(t, root, "remote", "add", "origin", root)
		gitTest(t, root, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
		gitTest(t, root, "update-ref", "refs/remotes/origin/source", first)
		gitTest(t, root, "config", "branch.autoSetupMerge", "always")
		commands := transactionCommands(t, root)
		target, err := commands.PrepareBranchActionTarget("refs/remotes/origin/source", first)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := commands.PrepareBranchActionCreate(target, "chosen-name", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
		if err != nil {
			t.Fatal(err)
		}
		cmd, err := commands.BranchActionCreateCommand(plan)
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		err = runBranchCreateGit(root, "config", "--get", "branch.chosen-name.remote")
		assert.Error(t, err)
	})
	t.Run("no-track preserves orphan upstream metadata", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		gitTest(t, root, "remote", "add", "origin", root)
		gitTest(t, root, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
		gitTest(t, root, "update-ref", "refs/remotes/origin/source", first)
		gitTest(t, root, "config", "branch.orphan-metadata.remote", "origin")
		gitTest(t, root, "config", "branch.orphan-metadata.merge", "refs/heads/old-upstream")
		commands := transactionCommands(t, root)
		target, err := commands.PrepareBranchActionTarget("refs/remotes/origin/source", first)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := commands.PrepareBranchActionCreate(target, "orphan-metadata", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
		if err != nil {
			t.Fatal(err)
		}
		cmd, err := commands.BranchActionCreateCommand(plan)
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "origin", gitTest(t, root, "config", "--get", "branch.orphan-metadata.remote"))
		assert.Equal(t, "refs/heads/old-upstream", gitTest(t, root, "config", "--get", "branch.orphan-metadata.merge"))
	})
	t.Run("direct custom fetch namespace", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		gitTest(t, root, "remote", "add", "origin", root)
		gitTest(t, root, "config", "--add", "remote.origin.fetch", "+refs/changes/*:refs/remotes/origin/changes/*")
		gitTest(t, root, "config", "branch.autoSetupRebase", "always")
		gitTest(t, root, "update-ref", "refs/remotes/origin/changes/42", first)
		commands := transactionCommands(t, root)
		target, err := commands.PrepareBranchActionTarget("refs/remotes/origin/changes/42", first)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := commands.PrepareBranchActionCreate(target, "review-42", BranchActionCreateOptions{Tracking: BranchActionCreateDirectTracking})
		if err != nil {
			t.Fatal(err)
		}
		cmd, err := commands.BranchActionCreateCommand(plan)
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "true", gitTest(t, root, "config", "--get", "branch.review-42.rebase"))
	})
}

func TestBranchActionCreateBareUnbornAndIgnoredCollision(t *testing.T) {
	t.Run("unborn HEAD explicit OID", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		gitTest(t, root, "checkout", "--orphan", "unborn")
		commands := transactionCommands(t, root)
		target, err := commands.PrepareBranchActionTarget(first, first)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := commands.PrepareBranchActionCreate(target, "from-oid", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
		if err != nil {
			t.Fatal(err)
		}
		cmd, err := commands.BranchActionCreateCommand(plan)
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/heads/from-oid"))
	})
	t.Run("pointer destination reserved by unborn HEAD", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		gitTest(t, root, "checkout", "--orphan", "unborn")
		commands := transactionCommands(t, root)
		target, err := commands.PrepareBranchActionTarget(first, first)
		if err != nil {
			t.Fatal(err)
		}
		_, err = commands.PrepareBranchActionCreate(target, "unborn", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
		assert.Error(t, err)
	})
	t.Run("bare pointer create and checkout refusal", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		bare := t.TempDir()
		gitTest(t, filepath.Dir(root), "clone", "--bare", root, bare)
		commands := transactionCommandsForBareRepo(t, bare)
		gitTest(t, bare, "branch", "-f", "master", first)
		target, err := commands.PrepareBranchActionTarget(first, first)
		if err != nil {
			t.Fatal(err)
		}
		_, err = commands.PrepareBranchActionCreate(target, "bare-checkout", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking, Checkout: true})
		assert.Error(t, err)
		plan, err := commands.PrepareBranchActionCreate(target, "bare-pointer", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking})
		if err != nil {
			t.Fatal(err)
		}
		cmd, err := commands.BranchActionCreateCommand(plan)
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("ignored collision fails and leaves destination absent", func(t *testing.T) {
		root, _, _ := transactionHistory(t)
		writeTestFile(t, filepath.Join(root, ".gitignore"), []byte("ignored-target\n"), 0o644)
		writeTestFile(t, filepath.Join(root, "ignored-target"), []byte("committed target"), 0o644)
		gitTest(t, root, "add", "-f", ".gitignore", "ignored-target")
		gitTest(t, root, "commit", "-qm", "add collision target")
		targetOID := gitTest(t, root, "rev-parse", "HEAD")
		gitTest(t, root, "reset", "--hard", "HEAD^")
		writeTestFile(t, filepath.Join(root, ".git", "info", "exclude"), []byte("ignored-target\n"), 0o644)
		writeTestFile(t, filepath.Join(root, "ignored-target"), []byte("local ignored"), 0o644)
		commands := transactionCommands(t, root)
		target, err := commands.PrepareBranchActionTarget(targetOID, targetOID)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := commands.PrepareBranchActionCreate(target, "ignored-fail", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking, Checkout: true})
		if err != nil {
			t.Fatal(err)
		}
		cmd, err := commands.BranchActionCreateCommand(plan)
		if err != nil {
			t.Fatal(err)
		}
		assert.Error(t, cmd.Run())
		err = runBranchCreateGit(root, "show-ref", "--verify", "refs/heads/ignored-fail")
		assert.Error(t, err)
	})
}

func TestBranchActionCreateCheckoutCollisions(t *testing.T) {
	for _, tc := range []struct {
		name          string
		prepareTarget func(*testing.T, string)
		prepareLocal  func(*testing.T, string)
	}{{
		name: "dirty tracked", prepareTarget: func(t *testing.T, root string) {
			t.Helper()
			writeTestFile(t, filepath.Join(root, "tracked"), []byte("target change\n"), 0o644)
		},
		prepareLocal: func(t *testing.T, root string) {
			t.Helper()
			writeTestFile(t, filepath.Join(root, "tracked"), []byte("local dirty change\n"), 0o644)
		},
	}, {
		name: "untracked", prepareTarget: func(t *testing.T, root string) {
			t.Helper()
			writeTestFile(t, filepath.Join(root, "collision-path"), []byte("target file"), 0o644)
			gitTest(t, root, "add", "collision-path")
		},
		prepareLocal: func(t *testing.T, root string) {
			t.Helper()
			writeTestFile(t, filepath.Join(root, "collision-path"), []byte("local untracked file"), 0o644)
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			root, _, _ := transactionHistory(t)
			parent := gitTest(t, root, "rev-parse", "HEAD")
			tc.prepareTarget(t, root)
			if tc.name == "dirty tracked" {
				gitTest(t, root, "add", "tracked")
			}
			gitTest(t, root, "commit", "-qm", "collision target")
			targetOID := gitTest(t, root, "rev-parse", "HEAD")
			gitTest(t, root, "checkout", "-q", "--detach", parent)
			tc.prepareLocal(t, root)
			commands := transactionCommands(t, root)
			target, err := commands.PrepareBranchActionTarget(targetOID, targetOID)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := commands.PrepareBranchActionCreate(target, "collision-fail", BranchActionCreateOptions{Tracking: BranchActionCreateNoTracking, Checkout: true})
			if err != nil {
				t.Fatal(err)
			}
			cmd, err := commands.BranchActionCreateCommand(plan)
			if err != nil {
				t.Fatal(err)
			}
			assert.Error(t, cmd.Run())
			err = runBranchCreateGit(root, "show-ref", "--verify", "refs/heads/collision-fail")
			assert.Error(t, err)
		})
	}
}

func runBranchCreateGit(root string, args ...string) error {
	commandArgs := append([]string{"-C", root}, args...)
	return exec.Command("git", commandArgs...).Run()
}

func transactionCommandsForBareRepo(t *testing.T, root string) *BranchCommands {
	t.Helper()
	osCommand := oscommands.NewDummyOSCommand()
	paths, err := repoPathsForDir(root, osCommand.Cmd)
	if err != nil {
		t.Fatal(err)
	}
	paths.worktreePath = root
	return NewBranchCommands(NewGitCommon(common.NewDummyCommon(), nil, osCommand.Cmd, osCommand, paths, nil, nil))
}
