package helpers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Fork: every difftool launch goes through WithExternalTool (see
// docs-master/fork/MergeGuide.md). With no tool configured for the mode, Git
// would print "... is not configured" and guess one, so the user is asked. Every
// launch pins its tool with --tool=<name>, so Git never guesses: an unknown
// configured tool fails with Git's error. The chosen tool is applied to this
// launch only, unless the user saves it to Git's global config. Nothing is
// launched and cancel runs on Escape, on a missing tool, on a changed repository
// and on a failed save, so that a temporary file held by the caller is released
// exactly once.

// ExternalToolHelper launches external tools for the fork's difftool paths.
type ExternalToolHelper struct {
	c *HelperCommon
}

func NewExternalToolHelper(c *HelperCommon) *ExternalToolHelper {
	return &ExternalToolHelper{c: c}
}

// WithExternalTool runs launch with the tool Git should use for mode. apply sets
// that tool on the command launch runs, and launch must run the command and
// return its error. cancel, if not nil, runs once when nothing is launched.
//
// A configured tool launches at once, pinned to that tool. Otherwise a menu lists
// the tools Git reports, then offers "Just this time" or saving the tool to Git's
// global config. The repository is captured when this
// is called and checked before each step, so a repository switch while a menu
// is open refuses the launch instead of running it in the wrong repository.
//
// Callers must invoke this from the UI thread, since it opens menus.
func (self *ExternalToolHelper) WithExternalTool(mode git_commands.ExternalToolMode, launch func(apply func(*oscommands.CmdObj) error) error, cancel func() error) error {
	flow := &externalToolFlow{
		c:          self.c,
		mode:       mode,
		repository: captureActionRepository(self.c),
		launch:     launch,
		cancel:     cancel,
	}
	return flow.start()
}

type externalToolFlow struct {
	c          *HelperCommon
	mode       git_commands.ExternalToolMode
	repository *capturedActionRepository
	launch     func(apply func(*oscommands.CmdObj) error) error
	cancel     func() error
	// settled is set once the flow launches or cancels, so each ends it once.
	settled bool
}

type externalToolTexts struct {
	chooseTitle string
	saveAndUse  string
	notFound    string
	searching   string
}

// pinTo returns the apply function that makes this launch use tool.
func (flow *externalToolFlow) pinTo(tool string) func(*oscommands.CmdObj) error {
	return func(cmd *oscommands.CmdObj) error {
		return git_commands.ApplyExternalTool(cmd, flow.mode, tool)
	}
}

func (flow *externalToolFlow) texts() externalToolTexts {
	tr := flow.c.Tr
	if flow.mode == git_commands.ExternalToolMerge {
		return externalToolTexts{tr.ExternalToolChooseMergeTitle, tr.ExternalToolSaveMerge, tr.ExternalToolNotFoundMerge, tr.ExternalToolSearching}
	}
	return externalToolTexts{tr.ExternalToolChooseDiffTitle, tr.ExternalToolSaveDiff, tr.ExternalToolNotFoundDiff, tr.ExternalToolSearching}
}

// commandID names a menu item, such as diff-tool.save for the diff mode.
func (flow *externalToolFlow) commandID(action string) string {
	if flow.mode == git_commands.ExternalToolMerge {
		return "merge-tool." + action
	}
	return "diff-tool." + action
}

func (flow *externalToolFlow) settle() bool {
	if flow.settled {
		return false
	}
	flow.settled = true
	return true
}

func (flow *externalToolFlow) launchWith(apply func(*oscommands.CmdObj) error) error {
	if !flow.settle() {
		return errors.New("external tool launch was already settled")
	}
	return flow.launch(apply)
}

func (flow *externalToolFlow) cancelFlow() error {
	if !flow.settle() || flow.cancel == nil {
		return nil
	}
	return flow.cancel()
}

// abort ends the flow without launching and returns err with any cancel error.
func (flow *externalToolFlow) abort(err error) error {
	return errors.Join(err, flow.cancelFlow())
}

func (flow *externalToolFlow) start() error {
	if err := flow.repository.checkRepository(); err != nil {
		return flow.abort(err)
	}
	git := flow.repository.git
	configured, err := git.Diff.ConfiguredExternalTool(flow.mode)
	if err != nil {
		return flow.abort(err)
	}
	if configured != "" {
		return flow.launchWith(flow.pinTo(configured))
	}
	// git difftool --tool-help probes for each tool and can take over a second,
	// so it runs on a worker. The menu opens on the UI thread, after the
	// repository is checked again.
	return flow.c.WithWaitingStatus(flow.texts().searching, func(gocui.Task) error {
		tools, err := git.Diff.AvailableExternalTools(flow.mode)
		flow.c.OnUIThread(func() error { return flow.offerTools(tools, err) })
		return nil
	})
}

func (flow *externalToolFlow) offerTools(tools []git_commands.ExternalTool, listErr error) error {
	if err := flow.repository.checkRepository(); err != nil {
		return flow.abort(err)
	}
	if listErr != nil {
		return flow.abort(listErr)
	}
	if len(tools) == 0 {
		return flow.abort(errors.New(flow.texts().notFound))
	}
	items := make([]*types.MenuItem, 0, len(tools))
	for _, tool := range tools {
		items = append(items, &types.MenuItem{
			LabelColumns: []string{tool.Name, tool.Description},
			CommandID:    flow.commandID("choose"),
			Verb:         "choose",
			OnPress:      func() error { return flow.chooseTool(tool) },
		})
	}
	return flow.c.Menu(types.CreateMenuOptions{
		Title:    flow.texts().chooseTitle,
		Items:    items,
		OnCancel: flow.cancelFlow,
	})
}

func (flow *externalToolFlow) chooseTool(tool git_commands.ExternalTool) error {
	if err := flow.repository.checkRepository(); err != nil {
		return flow.abort(err)
	}
	return flow.c.Menu(types.CreateMenuOptions{
		Title: fmt.Sprintf(flow.c.Tr.ExternalToolUseTitle, tool.Name),
		Items: []*types.MenuItem{
			{
				CommandID: flow.commandID("use-once"),
				Verb:      "use-once",
				Label:     flow.c.Tr.ExternalToolUseOnce,
				OnPress:   func() error { return flow.useOnce(tool) },
			},
			{
				CommandID: flow.commandID("save"),
				Verb:      "save",
				Label:     flow.texts().saveAndUse,
				OnPress:   func() error { return flow.saveAndUse(tool) },
			},
		},
		OnCancel: flow.cancelFlow,
	})
}

func (flow *externalToolFlow) useOnce(tool git_commands.ExternalTool) error {
	if err := flow.repository.checkRepository(); err != nil {
		return flow.abort(err)
	}
	return flow.launchWith(flow.pinTo(tool.Name))
}

// saveAndUse writes the tool to Git's global config, then launches pinned to it,
// as useOnce does. If the save fails, nothing launches and the save's error is
// returned.
func (flow *externalToolFlow) saveAndUse(tool git_commands.ExternalTool) error {
	if err := flow.repository.checkRepository(); err != nil {
		return flow.abort(err)
	}
	if err := flow.repository.git.Diff.SaveExternalToolGlobally(flow.mode, tool.Name); err != nil {
		return flow.abort(err)
	}
	return flow.launchWith(flow.pinTo(tool.Name))
}
