package controllers

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/modes/filtering"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: w b shows a ref with its history: a local branch with its upstream, or a remote
// branch or tag alone. This file holds what Commits, Branches, Remote branches and Tags
// share: the scope itself, the menu labels and the decoration check. The integration
// points are in branches_controller.go (see fork_branch_scope_filter.go),
// remote_branches_controller.go and tags_controller.go, and fork_commit_scope_filters.go
// for Commits.

// applyRefScope shows fullRefName, with the cursor on cursorHash, the commit the ref
// points at.
func applyRefScope(c *ControllerCommon, fullRefName string, cursorHash string) error {
	return c.Helpers().Mode.SetCommitScopeFilter(filtering.CommitScope{Ref: fullRefName}, cursorHash)
}

// applyRefScopeAtTip is applyRefScope for a ref whose tip is not in the model, which is
// the case for remote branches and tags. The tip is resolved now, when the command runs.
func applyRefScopeAtTip(c *ControllerCommon, fullRefName string) error {
	hash, err := c.Git().Commit.ResolveRefCommitHash(fullRefName)
	if err != nil {
		return err
	}
	return applyRefScope(c, fullRefName, hash)
}

// refScopeLabel is the menu label of a full ref name: a local branch with its upstream
// when it has one, a remote branch as origin/name, and a tag as "tag name".
func refScopeLabel(c *ControllerCommon, fullRefName string) string {
	if name, isBranch := strings.CutPrefix(fullRefName, "refs/heads/"); isBranch {
		if upstream, ok := helpers.LocalBranchScopeUpstream(c.Model().Branches, name); ok {
			return fmt.Sprintf(c.Tr.RefScopeLocalWithUpstream, name, upstream)
		}
		return name
	}
	if name, isTag := strings.CutPrefix(fullRefName, "refs/tags/"); isTag {
		return fmt.Sprintf(c.Tr.RefScopeTagLabel, name)
	}
	return strings.TrimPrefix(fullRefName, "refs/remotes/")
}

// commitHasRefScope reports whether a local branch, a remote branch or a tag decorates
// commit. It reads only the model, so that the key can be checked when the list is
// drawn. A bare HEAD decoration does not count, and neither does origin/HEAD, which the
// model does not list as a remote branch.
func commitHasRefScope(commit *models.Commit, branches []*models.Branch, remotes []*models.Remote) bool {
	if len(commit.Tags) > 0 || len(localBranchNamesAt(branches, commit.Hash())) > 0 {
		return true
	}
	// The model lists the branches of every remote, not only of the selected one.
	remoteNames := map[string]bool{}
	for _, remote := range remotes {
		for _, branch := range remote.Branches {
			remoteNames[branch.FullName()] = true
		}
	}
	for _, name := range commitDecorationNames(commit.ExtraInfo) {
		if remoteNames[name] {
			return true
		}
	}
	return false
}

// commitDecorationNames returns the names in a commit's decoration, such as
// "(HEAD -> main, origin/main, tag: v1)": "main", "origin/main" and "tag: v1". The
// "HEAD -> " prefix is dropped, so the branch that HEAD points at is returned. A bare
// HEAD is returned as "HEAD", since it names no ref.
func commitDecorationNames(extraInfo string) []string {
	inner := strings.TrimSuffix(strings.TrimPrefix(extraInfo, "("), ")")
	if inner == "" {
		return nil
	}
	names := []string{}
	for _, field := range strings.Split(inner, ", ") {
		names = append(names, strings.TrimPrefix(strings.TrimSpace(field), "HEAD -> "))
	}
	return names
}

// localBranchNamesAt returns the names of the local branches whose tip is hash, in
// the model's order. A detached HEAD is not a branch.
func localBranchNamesAt(branches []*models.Branch, hash string) []string {
	names := []string{}
	for _, branch := range branches {
		if !branch.DetachedHead && branch.CommitHash == hash {
			names = append(names, branch.Name)
		}
	}
	return names
}

// refScopeFilterBinding is w b in Remote branches. The commit is resolved when it runs.
func (self *RemoteBranchesController) refScopeFilterBinding() *types.Binding {
	return &types.Binding{
		CommandID:         "filter.ref",
		Verb:              "filter",
		Description:       self.c.Tr.FilterRef,
		Handler:           self.withItem(self.applyRefScopeFilter),
		GetDisabledReason: self.require(self.singleItemSelected()),
	}
}

func (self *RemoteBranchesController) applyRefScopeFilter(branch *models.RemoteBranch) error {
	return applyRefScopeAtTip(self.c, branch.FullRefName())
}

// refScopeFilterBinding is w b in Tags. The commit is resolved when it runs.
func (self *TagsController) refScopeFilterBinding() *types.Binding {
	return &types.Binding{
		CommandID:         "filter.ref",
		Verb:              "filter",
		Description:       self.c.Tr.FilterRef,
		Handler:           self.withItem(self.applyRefScopeFilter),
		GetDisabledReason: self.require(self.singleItemSelected()),
	}
}

func (self *TagsController) applyRefScopeFilter(tag *models.Tag) error {
	return applyRefScopeAtTip(self.c, tag.FullRefName())
}
