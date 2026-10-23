package git_commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchActionTagCreate(t *testing.T) {
	root, first, second := transactionHistory(t)
	branch := gitTest(t, root, "symbolic-ref", "--short", "HEAD")
	gitTest(t, root, "checkout", "-q", "--detach", first)
	gitTest(t, root, "branch", "-f", branch, first)
	branches := transactionCommands(t, root)
	target, err := branches.PrepareBranchActionTarget("refs/heads/"+branch, first)
	if !assert.NoError(t, err) {
		return
	}
	tags := NewTagCommands(branches.GitCommon)
	plan, err := tags.PrepareBranchActionTagCreate(target, "literal-tag", "")
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, target, plan.Target())
	assert.Equal(t, "literal-tag", plan.Name())
	assert.Empty(t, plan.Description())
	assert.False(t, plan.Annotated())
	assert.False(t, plan.Signed())
	assert.False(t, plan.DestinationExists())
	cmd, err := tags.BranchActionTagCreateCommand(plan)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NoError(t, cmd.Run()) {
		return
	}
	assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/tags/literal-tag^{commit}"))

	for _, name := range []string{"-option", "bad name", "bad\nname", "bad\x00name"} {
		_, err := tags.PrepareBranchActionTagCreate(target, name, "")
		assert.Error(t, err, name)
	}
	_, err = tags.PrepareBranchActionTagCreate(target, "nul-message", "bad\x00message")
	assert.Error(t, err)

	// An equals sign and option-looking suffix remain one literal ref name.
	plan, err = tags.PrepareBranchActionTagCreate(target, "name=--help", "")
	if assert.NoError(t, err) {
		cmd, commandErr := tags.BranchActionTagCreateCommand(plan)
		if assert.NoError(t, commandErr) {
			assert.Equal(t, "name=--help", cmd.Args()[len(cmd.Args())-2])
			assert.Equal(t, first+"^{commit}", cmd.Args()[len(cmd.Args())-1])
			assert.NoError(t, cmd.Run())
			assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/tags/name=--help^{commit}"))
		}
	}
	// A same-named branch at another commit cannot substitute for the captured tag target.
	gitTest(t, root, "tag", "tag-shadow", second)
	gitTest(t, root, "branch", "tag-shadow", second)
	shadow, err := tags.PrepareBranchActionTagCreate(target, "tag-shadow", "")
	if assert.NoError(t, err) {
		assert.Equal(t, second, shadow.OldOID())
		cmd, commandErr := tags.BranchActionTagCreateCommand(shadow.WithOverwriteApproved())
		if assert.NoError(t, commandErr) {
			assert.Equal(t, first+"^{commit}", cmd.Args()[len(cmd.Args())-1])
			assert.NoError(t, cmd.Run())
			assert.Equal(t, first, gitTest(t, root, "rev-parse", "refs/tags/tag-shadow^{commit}"))
		}
	}

	gitTest(t, root, "tag", "-a", "same-commit", "-m", "old", first)
	oldObject := gitTest(t, root, "rev-parse", "refs/tags/same-commit")
	plan, err = tags.PrepareBranchActionTagCreate(target, "same-commit", "replacement")
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, plan.DestinationExists())
	assert.Equal(t, oldObject, plan.OldOID())
	assert.Error(t, func() error { _, e := tags.BranchActionTagCreateCommand(plan); return e }())
	cmd, err = tags.BranchActionTagCreateCommand(plan.WithOverwriteApproved())
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NoError(t, cmd.Run()) {
		return
	}
	assert.NotEqual(t, oldObject, gitTest(t, root, "rev-parse", "refs/tags/same-commit"))
}

func TestBranchActionTagCreateStalePlans(t *testing.T) {
	for _, scenario := range []string{"source", "head", "config", "destination changed", "destination added", "symbolic destination", "foreign", "zero plan"} {
		t.Run(scenario, func(t *testing.T) {
			root, first, second := transactionHistory(t)
			branch := gitTest(t, root, "symbolic-ref", "--short", "HEAD")
			commands := transactionCommands(t, root)
			target, err := commands.PrepareBranchActionTarget("refs/heads/"+branch, second)
			if !assert.NoError(t, err) {
				return
			}
			tags := NewTagCommands(commands.GitCommon)
			name := "stale-tag"
			if scenario == "destination changed" {
				gitTest(t, root, "tag", "-a", name, "-m", "before", first)
			}
			plan, err := tags.PrepareBranchActionTagCreate(target, name, "")
			if !assert.NoError(t, err) {
				return
			}
			switch scenario {
			case "source":
				gitTest(t, root, "update-ref", "refs/heads/"+branch, first)
			case "head":
				gitTest(t, root, "checkout", "-q", "--detach", first)
			case "config":
				gitTest(t, root, "config", "user.name", "Changed Name")
			case "destination changed":
				gitTest(t, root, "tag", "-fa", name, "-m", "after", second)
			case "destination added":
				gitTest(t, root, "tag", name, second)
			case "symbolic destination":
				gitTest(t, root, "symbolic-ref", "refs/tags/"+name, "refs/heads/"+branch)
			case "foreign":
				otherRoot, _, _ := transactionHistory(t)
				other := transactionCommands(t, otherRoot)
				_, err = NewTagCommands(other.GitCommon).BranchActionTagCreateCommand(plan)
				assert.Error(t, err)
				return
			case "zero plan":
				_, err = tags.BranchActionTagCreateCommand(BranchActionTagCreatePlan{})
				assert.Error(t, err)
				return
			}
			_, err = tags.BranchActionTagCreateCommand(plan.WithOverwriteApproved())
			assert.Error(t, err, "stale review must be blocked")
		})
	}
}

func TestBranchActionTagCreateReadOnlyAndConfigSigning(t *testing.T) {
	root, _, second := transactionHistory(t)
	branch := gitTest(t, root, "symbolic-ref", "--short", "HEAD")
	writeTestFile(t, filepath.Join(root, "tracked"), []byte("dirty worktree bytes\x00\n"), 0o644)
	writeTestFile(t, filepath.Join(root, "staged"), []byte("staged bytes\n"), 0o644)
	gitTest(t, root, "add", "staged")
	headBefore, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if !assert.NoError(t, err) {
		return
	}
	indexPath := gitTest(t, root, "rev-parse", "--git-path", "index")
	if !filepath.IsAbs(indexPath) {
		indexPath = filepath.Join(root, indexPath)
	}
	indexBefore, err := os.ReadFile(indexPath)
	if !assert.NoError(t, err) {
		return
	}
	trackedPath := filepath.Join(root, "tracked")
	worktreeBefore, err := os.ReadFile(trackedPath)
	if !assert.NoError(t, err) {
		return
	}
	commands := transactionCommands(t, root)
	target, err := commands.PrepareBranchActionTarget("refs/heads/"+branch, second)
	if !assert.NoError(t, err) {
		return
	}
	tags := NewTagCommands(commands.GitCommon)
	plan, err := tags.PrepareBranchActionTagCreate(target, "read-only", "")
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, target, plan.Target())
	cmd, err := tags.BranchActionTagCreateCommand(plan)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NoError(t, cmd.Run()) {
		return
	}
	assert.Equal(t, headBefore, mustRead(t, filepath.Join(root, ".git", "HEAD")))
	indexAfter, err := os.ReadFile(indexPath)
	if assert.NoError(t, err) {
		assert.Equal(t, indexBefore, indexAfter)
	}
	worktreeAfter, err := os.ReadFile(trackedPath)
	if assert.NoError(t, err) {
		assert.Equal(t, worktreeBefore, worktreeAfter)
	}

	gitTest(t, root, "config", "tag.gpgSign", "true")
	signed, err := tags.PrepareBranchActionTagCreate(target, "signed-plan", "")
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, signed.Annotated())
	assert.True(t, signed.Signed())
	descriptionPlan, err := tags.PrepareBranchActionTagCreate(target, "annotated-only", "message")
	if assert.NoError(t, err) {
		assert.True(t, descriptionPlan.Annotated())
		assert.True(t, descriptionPlan.Signed(), "native tag.gpgSign remains enabled")
	}
	cmd, err = tags.BranchActionTagCreateCommand(signed)
	if assert.NoError(t, err) {
		assert.Contains(t, cmd.Args(), "-m")
		assert.Contains(t, cmd.Args(), "")
		assert.NotContains(t, cmd.Args(), "--force")
	}
	gitTest(t, root, "config", "tag.gpgSign", "not-a-bool")
	_, err = tags.PrepareBranchActionTagCreate(target, "invalid-sign-config", "")
	assert.Error(t, err)
	assert.Equal(t, second, gitTest(t, root, "rev-parse", "refs/tags/read-only^{commit}"))
}

func TestBranchActionTagCreateHeadTarget(t *testing.T) {
	t.Run("named", func(t *testing.T) {
		root, _, _ := transactionHistory(t)
		commands := transactionCommands(t, root)
		target, err := commands.PrepareBranchActionHeadTarget()
		if assert.NoError(t, err) {
			assert.Equal(t, gitTest(t, root, "rev-parse", "HEAD"), target.SourceOID())
			assert.Equal(t, "refs/heads/"+gitTest(t, root, "symbolic-ref", "--short", "HEAD"), target.SourceRef())
		}
	})
	t.Run("detached", func(t *testing.T) {
		root, first, _ := transactionHistory(t)
		gitTest(t, root, "checkout", "-q", "--detach", first)
		commands := transactionCommands(t, root)
		target, err := commands.PrepareBranchActionHeadTarget()
		if assert.NoError(t, err) {
			assert.Equal(t, first, target.SourceOID())
			assert.Equal(t, first, target.SourceRef())
		}
	})
	t.Run("unborn", func(t *testing.T) {
		root := initHistoricalRepo(t)
		commands := transactionCommands(t, root)
		_, err := commands.PrepareBranchActionHeadTarget()
		assert.Error(t, err)
	})
	t.Run("head moved", func(t *testing.T) {
		root, first, second := transactionHistory(t)
		commands := transactionCommands(t, root)
		target, err := commands.PrepareBranchActionHeadTarget()
		if !assert.NoError(t, err) {
			return
		}
		tags := NewTagCommands(commands.GitCommon)
		plan, err := tags.PrepareBranchActionTagCreate(target, "head-moved", "")
		if !assert.NoError(t, err) {
			return
		}
		gitTest(t, root, "reset", "--hard", first)
		assert.NotEqual(t, first, second)
		_, err = tags.BranchActionTagCreateCommand(plan)
		assert.Error(t, err)
	})
}
