package controllers

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: branch and tag panels offer the flat `y` copy menu's commit actions for
// the commit their selected ref points at (see the copy table in
// docs-master/fork/Guide.md, Prefix menus). The commit is resolved when an action runs,
// so it is never stale. Annotated tags resolve to their commit, not the tag
// object.
type RefCommitCopyController struct {
	baseController
	c       *ControllerCommon
	context types.Context
	// selectedRef returns the full name of the selected ref, or "" if none is selected.
	selectedRef func() string
}

var _ types.IController = &RefCommitCopyController{}

func NewRefCommitCopyController(
	c *ControllerCommon,
	context types.Context,
	selectedRef func() string,
) *RefCommitCopyController {
	return &RefCommitCopyController{
		baseController: baseController{},
		c:              c,
		context:        context,
		selectedRef:    selectedRef,
	}
}

func (self *RefCommitCopyController) Context() types.Context {
	return self.context
}

func (self *RefCommitCopyController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	// Refs have no commit-level state to check, so every action is disabled only
	// when no ref is selected. The tags action is not checked here; it checks at
	// press time instead.
	return commitCopyBindingList(
		self.c,
		// The copy methods read only self.c (see commitCopyBindingList).
		&BasicCommitsController{c: self.c},
		self.selectedRefCommit,
		self.requireSelectedRef,
		self.requireSelectedRef,
	)
}

func (self *RefCommitCopyController) requireSelectedRef() *types.DisabledReason {
	if self.selectedRef() == "" {
		return &types.DisabledReason{Text: self.c.Tr.NoItemSelected, Inapplicable: true}
	}
	return nil
}

// selectedRefCommit resolves the selected ref to a commit, with its tags, at
// press time.
func (self *RefCommitCopyController) selectedRefCommit() (*models.Commit, error) {
	ref := self.selectedRef()
	if ref == "" {
		return nil, errors.New(self.c.Tr.NoItemSelected)
	}

	hash, err := self.c.Git().Commit.ResolveRefCommitHash(ref)
	if err != nil {
		return nil, err
	}
	tags, err := self.c.Git().Commit.TagsPointingAt(hash)
	if err != nil {
		return nil, err
	}

	return models.NewCommit(self.c.Model().HashPool, models.NewCommitOpts{
		Hash: hash,
		Tags: tags,
	}), nil
}
