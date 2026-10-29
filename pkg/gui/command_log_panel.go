package gui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/constants"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/theme"
)

// our UI command log looks like this:
// Stage File:
// git add -- 'filename'
// Unstage File:
// git reset HEAD 'filename'
//
// The 'Stage File' and 'Unstage File' lines are actions i.e they group up a set
// of command logs (typically there's only one command under an action but there may be more).
// So we call logAction to log the 'Stage File' part and then we call logCommand to log the command itself.
// We pass logCommand to our OSCommand struct so that it can handle logging commands
// for us.
func (gui *Gui) LogAction(action string) {
	if gui.Views.Extras == nil {
		return
	}

	// LogAction and LogCommand are called both from the UI thread and from git
	// worker goroutines, so bounce the writes onto the UI thread: they touch the
	// view's autoscroll flag and the GuiLog slice, which the layout/draw code
	// reads. Ordering between successive log calls is preserved by the FIFO the
	// bounce enqueues onto. It's a background bounce because writing the command
	// log is incidental display work that must not count towards lazygit being
	// busy (otherwise it could block a repo switch).
	gui.onUIThreadBackground(func() error {
		gui.Views.Extras.Autoscroll = true

		gui.GuiLog = append(gui.GuiLog, action)
		fmt.Fprint(gui.Views.Extras, "\n"+style.FgYellow.Sprint(action))
		return nil
	})
}

func (gui *Gui) LogCommand(cmdStr string, commandLine bool) {
	if gui.Views.Extras == nil {
		return
	}

	textStyle := theme.DefaultTextColor
	if !commandLine {
		// if we're not dealing with a direct command that could be run on the command line,
		// we style it differently to communicate that
		textStyle = style.FgMagenta
	}
	indentedCmdStr := "  " + strings.ReplaceAll(cmdStr, "\n", "\n  ")

	// See the comment in LogAction: bounce onto the UI thread since we may be
	// called from a git worker, in the background so it can't block a repo switch.
	gui.onUIThreadBackground(func() error {
		gui.Views.Extras.Autoscroll = true

		gui.GuiLog = append(gui.GuiLog, cmdStr)
		fmt.Fprint(gui.Views.Extras, "\n"+textStyle.Sprint(indentedCmdStr))
		return nil
	})
}

func (gui *Gui) printCommandLogHeader() {
	introStr := fmt.Sprintf(
		gui.c.Tr.CommandLogHeader,
		// Fork: the options menu is reached through its command path, not the legacy key field.
		gui.c.CommandKeyLabel("command-log.options-menu", ""),
	)
	fmt.Fprintln(gui.Views.Extras, style.FgCyan.Sprint(introStr))

	if gui.c.UserConfig().Gui.ShowRandomTip {
		fmt.Fprintf(
			gui.Views.Extras,
			"%s: %s",
			style.FgYellow.Sprint(gui.c.Tr.RandomTip),
			style.FgGreen.Sprint(gui.getRandomTip()),
		)
	}
}

// commandRoute names a command and the view whose effective key label is shown.
type commandRoute struct {
	commandID string
	view      string
}

// keyTip formats a tip with the effective label of each route. It returns ""
// when any route has no key, so that the tip is dropped rather than naming a
// key that does not exist.
func (gui *Gui) keyTip(format string, routes ...commandRoute) string {
	labels := make([]any, len(routes))
	for i, route := range routes {
		label := gui.c.CommandKeyLabel(route.commandID, route.view)
		if label == "" {
			return ""
		}
		labels[i] = label
	}
	return fmt.Sprintf(format, labels...)
}

func (gui *Gui) getRandomTip() string {
	tips := []string{
		// keybindings and lazygit-specific advice
		gui.keyTip(
			"To force push, press '%s' and then if the push is rejected you will be asked if you want to force push",
			commandRoute{"branch.push-current", ""},
		),
		gui.keyTip(
			"To filter commits by path, press '%s'",
			commandRoute{"filter.by-path", ""},
		),
		gui.keyTip(
			"To start an interactive rebase, press '%s' on a commit. You can always abort the rebase by pressing '%s' and selecting 'abort'",
			commandRoute{"commit.rebase-interactive", "commits"},
			commandRoute{"branch.merge-rebase-menu", ""},
		),
		gui.keyTip(
			"In flat file view, merge conflicts are sorted to the top. To switch to flat file view press '%s'",
			commandRoute{"ui.file-tree.toggle", "files"},
		),
		gui.keyTip(
			"If you press '%s'/'%s' you can undo/redo your changes. Be wary though, this only applies to branches/commits, so only do this if your worktree is clear.\nDocs: "+constants.Links.Docs.Undoing,
			commandRoute{"branch.reflog-undo", "commits"},
			commandRoute{"branch.reflog-redo", "commits"},
		),
		gui.keyTip(
			"to hard reset onto your current upstream branch, press '%s' in the files panel",
			commandRoute{"branch.reset-upstream-menu", "files"},
		),
		gui.keyTip(
			"To push a tag, navigate to the tag in the tags tab and press '%s'",
			commandRoute{"tag.push", "tags"},
		),
		gui.keyTip(
			"You can view the individual files of a stash entry by pressing '%s'",
			commandRoute{"navigation.open-diff-files", "stash"},
		),
		gui.keyTip(
			"You can diff two commits by pressing '%s' on one commit and then navigating to the other. You can then press '%s' to view the files of the diff",
			commandRoute{"diff.options-menu", ""},
			commandRoute{"navigation.open-diff-files", "commits"},
		),
		gui.keyTip(
			"press '%s' on a commit to drop it (delete it)",
			commandRoute{"commit.drop", "commits"},
		),
		gui.keyTip(
			"If you need to pull out the big guns to resolve merge conflicts, you can press '%s' in the files panel to open merge options",
			commandRoute{"conflict.options-menu", "files"},
		),
		gui.keyTip(
			"To revert a commit, press '%s' on that commit",
			commandRoute{"commit.revert", "commits"},
		),
		gui.keyTip(
			"To escape a mode, for example cherry-picking, patch-building, diffing, or filtering mode, you can just spam the '%s' button. Unless of course you have `quitOnTopLevelReturn` enabled in your config",
			commandRoute{"app.cancel", ""},
		),
		gui.keyTip(
			"You can page through the items of a panel using '%s' and '%s'",
			commandRoute{"navigation.page.previous", "commits"},
			commandRoute{"navigation.page.next", "commits"},
		),
		gui.keyTip(
			"You can jump to the top/bottom of a panel using '%s' and '%s'",
			commandRoute{"navigation.list.first", "commits"},
			commandRoute{"navigation.list.last", "commits"},
		),
		gui.keyTip(
			"To collapse/expand a directory in the file tree, press '%s'/'%s'",
			commandRoute{"navigation.tree.collapse", "files"},
			commandRoute{"navigation.tree.expand", "files"},
		),
		gui.keyTip(
			"You can append your staged changes to an older commit by pressing '%s' on that commit",
			commandRoute{"commit.amend-selected", "commits"},
		),
		gui.keyTip(
			"You can amend the last commit with your new file changes by pressing '%s' in the files panel",
			commandRoute{"commit.amend-head", "files"},
		),
		gui.keyTip(
			"You can now navigate the side panels with '%s' and '%s'",
			commandRoute{"navigation.panel.next", "files"},
			commandRoute{"navigation.panel.previous", "files"},
		),

		"You can use lazygit with a bare repo by passing the --git-dir and --work-tree arguments as you would for the git CLI",

		// general advice
		"`git commit` is really just the programmer equivalent of saving your game. Always do it before embarking on an ambitious change!",
		"Try to separate commits that refactor code from commits that add new functionality: if they're squashed into one commit, it can be hard to spot what's new.",
		"If you ever want to experiment, it's easy to create a new branch off your current one and go nuts, then delete it afterwards",
		"Always read through the diff of your changes before assigning somebody to review your code. Better for you to catch any silly mistakes than your colleagues!",
		"If something goes wrong, you can always checkout a commit from your reflog to return to an earlier state",
		"The stash is a good place to save snippets of code that you always find yourself adding when debugging.",

		// links
		fmt.Sprintf(
			"If you want a git diff with syntax colouring, check out lazygit's integration with delta:\n%s",
			constants.Links.Docs.CustomDiffRenderers,
		),
		fmt.Sprintf(
			"You can build your own custom menus and commands to run from within lazygit. For examples see:\n%s",
			constants.Links.Docs.CustomCommands,
		),
		fmt.Sprintf(
			"If you ever find a bug, do not hesitate to raise an issue on the repo:\n%s",
			constants.Links.Issues,
		),
		"You can alt-click a line in a diff to open it in your editor. Not all terminals support this; if yours doesn't, try shift-click instead.",
	}

	// Tips that name a key with no effective label are dropped.
	available := make([]string, 0, len(tips))
	for _, tip := range tips {
		if tip != "" {
			available = append(available, tip)
		}
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomIndex := rnd.Intn(len(available))
	return available[randomIndex]
}
