package helpers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

type capturedMergeChoice struct {
	id, label, verb string
	mode            git_commands.BranchActionMergeMode
	key             rune
}

type capturedMergeFlow struct {
	repository      *capturedActionRepository
	review          git_commands.BranchActionMergeReview
	squashMessage   string
	skipHookPrefix  string
	signOff         bool
	applicationArgs string
}

type capturedMergeReview struct {
	flow          *capturedMergeFlow
	plan          git_commands.BranchActionMergePlan
	choice        capturedMergeChoice
	commitOptions git_commands.BranchActionSquashCommitOptions
	started       bool
}

// MergeSelectedIntoCurrent captures the selected source, repository and all
// application merge/commit configuration before background inspection begins.
func MergeSelectedIntoCurrent(c *HelperCommon, ref string) error {
	config := c.UserConfig().Git
	flow := &capturedMergeFlow{
		repository:      captureActionRepository(c),
		squashMessage:   config.Merging.SquashMergeMessage,
		skipHookPrefix:  config.SkipHookPrefix,
		signOff:         config.Commit.SignOff,
		applicationArgs: config.Merging.Args,
	}
	return c.WithWaitingStatus(c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.repository.checkRepository(); err != nil {
			return err
		}
		target, err := flow.captureTarget(ref)
		if err != nil {
			return err
		}
		flow.review, err = flow.repository.git.Branch.PrepareBranchActionMergeReview(target, flow.applicationArgs)
		if err != nil {
			return err
		}
		c.OnUIThread(flow.menu)
		return nil
	})
}

func (flow *capturedMergeFlow) captureTarget(ref string) (git_commands.BranchActionTarget, error) {
	oid, err := flow.repository.git.Branch.ResolveBranchActionCommit(ref)
	if err != nil {
		return git_commands.BranchActionTarget{}, err
	}
	return flow.repository.git.Branch.PrepareBranchActionTarget(ref, oid)
}

func (flow *capturedMergeFlow) check() error {
	// UI callbacks only guard repository identity; native review/plan validation
	// runs in worker execution paths and must not block the UI thread.
	return flow.repository.checkRepository()
}

func (flow *capturedMergeFlow) prompt(choice capturedMergeChoice) string {
	target := flow.review.Target()
	mode := choice.label
	text := fmt.Sprintf(flow.repository.c.Tr.BranchActionMergeReviewPrompt,
		ShortBranchName(target.HeadRef()), target.HeadOID(), ShortBranchName(target.SourceRef()), target.SourceRef(), target.SourceOID(), mode)
	args := flow.review.Arguments()
	if len(args) > 0 {
		quoted := make([]string, len(args))
		for i, arg := range args {
			quoted[i] = fmt.Sprintf("%q", arg)
		}
		text += fmt.Sprintf(flow.repository.c.Tr.BranchActionMergeOptions, strings.Join(quoted, " "))
	}
	return text
}

func (flow *capturedMergeFlow) menu() error {
	if err := flow.check(); err != nil {
		return err
	}
	c := flow.repository.c
	target := flow.review.Target()
	var disabled *types.DisabledReason
	if target.SourceRef() == target.HeadRef() {
		disabled = &types.DisabledReason{Text: "cannot merge the current branch into itself"}
	}
	choices := []capturedMergeChoice{
		{id: "branch.merge.regular", label: c.Tr.BranchActionMergeConfigured, verb: "merge", mode: git_commands.BranchActionMergeConfigured, key: 'm'},
		{id: "branch.merge.ff", label: c.Tr.BranchActionMergeAllowFF, verb: "merge", mode: git_commands.BranchActionMergeAllowFF, key: 'f'},
		{id: "branch.merge.no-ff", label: c.Tr.BranchActionMergeNoFF, verb: "merge", mode: git_commands.BranchActionMergeNoFF, key: 'n'},
		{id: "branch.merge.squash-uncommitted", label: c.Tr.SquashMergeUncommitted, verb: "squash-merge", mode: git_commands.BranchActionMergeSquash, key: 's'},
		{id: "branch.merge.squash-committed", label: c.Tr.SquashMergeCommitted, verb: "squash-merge", mode: git_commands.BranchActionMergeSquashAndCommit, key: 'S'},
	}
	items := make([]*types.MenuItem, 0, len(choices))
	for _, choice := range choices {
		items = append(items, &types.MenuItem{
			CommandID: choice.id, Verb: choice.verb, Label: choice.label, Keys: menuKey(choice.key), DisabledReason: disabled,
			OpensMenu: true, OnPress: func() error { return flow.choose(choice) },
		})
	}
	return c.Menu(types.CreateMenuOptions{Title: c.Tr.Merge, Prompt: flow.prompt(choices[0]), Items: items, OnCancel: flow.repository.checkRepository})
}

func (flow *capturedMergeFlow) choose(choice capturedMergeChoice) error {
	if err := flow.check(); err != nil {
		return err
	}
	target := flow.review.Target()
	if target.SourceRef() == target.HeadRef() {
		return errors.New("cannot merge the current branch into itself")
	}
	if choice.mode == git_commands.BranchActionMergeSquashAndCommit {
		initial := utils.ResolvePlaceholderString(flow.squashMessage, map[string]string{
			"selectedRef": ShortBranchName(target.SourceRef()), "currentBranch": ShortBranchName(target.HeadRef()),
		})
		flow.repository.c.Prompt(types.PromptOpts{
			Title: flow.repository.c.Tr.BranchActionMergeMessageTitle, InitialContent: initial,
			PreserveWhitespace: true, AllowEditSuggestion: true, HandleClose: flow.check,
			HandleConfirm: func(message string) error {
				if err := flow.check(); err != nil {
					return err
				}
				if strings.TrimSpace(message) == "" || strings.ContainsRune(message, '\x00') {
					return errors.New("commit message must not be blank or contain NUL")
				}
				return flow.prepare(choice, message)
			},
		})
		return nil
	}
	return flow.prepare(choice, fmt.Sprintf(flow.repository.c.Tr.BranchActionMergeDefaultMessage,
		ShortBranchName(target.SourceRef()), ShortBranchName(target.HeadRef())))
}

func (flow *capturedMergeFlow) prepare(choice capturedMergeChoice, message string) error {
	if err := flow.check(); err != nil {
		return err
	}
	c := flow.repository.c
	return c.WithWaitingStatus(c.Tr.BranchSyncInspecting, func(gocui.Task) error {
		if err := flow.check(); err != nil {
			return err
		}
		plan, err := flow.repository.git.Branch.PrepareBranchActionMerge(flow.review, choice.mode, message)
		if err != nil {
			return err
		}
		c.OnUIThread(func() error {
			if err := flow.check(); err != nil {
				return err
			}
			review := &capturedMergeReview{
				flow: flow, plan: plan, choice: choice,
				commitOptions: git_commands.BranchActionSquashCommitOptions{
					Message: message, SignOff: flow.signOff,
					SkipHooks: flow.skipHookPrefix != "" && strings.HasPrefix(message, flow.skipHookPrefix),
				},
			}
			prompt := flow.prompt(choice)
			if choice.mode == git_commands.BranchActionMergeSquashAndCommit {
				prompt += "\n\n" + fmt.Sprintf(c.Tr.BranchActionMergeChosenMessage, message)
			}
			c.Confirm(types.ConfirmOpts{Title: c.Tr.Merge, Prompt: prompt, HandleClose: flow.check, HandleConfirm: review.reviewRisks})
			return nil
		})
		return nil
	})
}

func (review *capturedMergeReview) check() error {
	if err := review.flow.check(); err != nil {
		return err
	}
	if review.started {
		return git_commands.ErrBranchActionChanged
	}
	return nil
}

func (review *capturedMergeReview) reviewRisks() error {
	if err := review.check(); err != nil {
		return err
	}
	if review.choice.mode == git_commands.BranchActionMergeSquashAndCommit && !review.plan.Review().AlreadyMerged() && len(review.plan.Review().StagedFiles()) > 0 {
		return review.reviewStaged()
	}
	return review.reviewIgnored()
}

func (review *capturedMergeReview) reviewStaged() error {
	if err := review.check(); err != nil {
		return err
	}
	c := review.flow.repository.c
	items := []*types.MenuItem{
		{CommandID: review.choice.id + ".cancel-staged-risk", Verb: "cancel", Label: c.Tr.Cancel, OnPress: review.flow.repository.checkRepository},
		{CommandID: review.choice.id + ".approve-staged", Verb: review.choice.verb, Label: c.Tr.BranchActionMergeApproveStaged, OnPress: func() error {
			review.plan = review.plan.WithStagedFilesApproved()
			return review.reviewIgnored()
		}},
	}
	for _, path := range review.plan.Review().StagedFiles() {
		quoted := fmt.Sprintf("%q", path)
		items = append(items, &types.MenuItem{CommandID: review.choice.id + ".inspect-staged-path", Verb: "inspect", Label: quoted, Tooltip: quoted, OpensMenu: true, OnPress: review.reviewStaged})
	}
	return c.Menu(types.CreateMenuOptions{Title: c.Tr.BranchActionMergeStagedTitle, Prompt: c.Tr.BranchActionMergeStagedPrompt, HideCancel: true, Items: items, OnCancel: review.flow.repository.checkRepository})
}

func (review *capturedMergeReview) reviewIgnored() error {
	if err := review.check(); err != nil {
		return err
	}
	if len(review.plan.IgnoredFiles()) == 0 {
		return review.execute(review.plan)
	}
	c := review.flow.repository.c
	prompt := c.Tr.BranchActionMergeIgnoredPrompt
	if review.plan.ConservativeIgnoredReview() {
		prompt = c.Tr.BranchActionMergeAllIgnoredPrompt
	}
	items := []*types.MenuItem{
		{CommandID: review.choice.id + ".cancel-ignored-risk", Verb: "cancel", Label: c.Tr.Cancel, OnPress: review.flow.repository.checkRepository},
		{CommandID: review.choice.id + ".approve-ignored", Verb: review.choice.verb, Label: c.Tr.BranchActionMergeApproveIgnored, OnPress: func() error {
			return review.execute(review.plan.WithIgnoredFilesApproved())
		}},
	}
	for _, path := range review.plan.IgnoredFiles() {
		quoted := fmt.Sprintf("%q", path)
		items = append(items, &types.MenuItem{CommandID: review.choice.id + ".inspect-ignored-path", Verb: "inspect", Label: quoted, Tooltip: quoted, OpensMenu: true, OnPress: review.reviewIgnored})
	}
	return c.Menu(types.CreateMenuOptions{Title: c.Tr.BranchActionMergeIgnoredTitle, Prompt: prompt, HideCancel: true, Items: items, OnCancel: review.flow.repository.checkRepository})
}
