package theme

import (
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/utils/promptmark"
)

// ResolvePromptMarks replaces the role markers in a prompt with the styles the
// panes use for the same kind of text, so that a branch name in a prompt
// looks like a branch name in the Branches panel. Styles are read at call
// time, so theme changes apply.
//
// Upstream code cannot show these markers, so this runs just before a prompt
// is measured and displayed (see the Fork: comments at the call sites).
func ResolvePromptMarks(s string) string {
	return promptmark.Resolve(s, promptMarkStyle)
}

func promptMarkStyle(role promptmark.Role) func(string) string {
	var textStyle style.TextStyle
	switch role {
	case promptmark.RoleBranch:
		textStyle = style.FgCyan.SetBold()
	case promptmark.RoleRemoteBranch, promptmark.RoleRemote:
		textStyle = style.FgYellow
	case promptmark.RoleTag:
		textStyle = style.FgMagenta.SetBold()
	case promptmark.RoleHash:
		textStyle = style.FgYellow
	case promptmark.RolePath, promptmark.RoleCount:
		textStyle = style.AttrBold
	case promptmark.RoleKey:
		textStyle = OptionsFgColor
	case promptmark.RoleWarn:
		textStyle = style.FgRed.SetBold()
	default:
		return nil
	}
	return func(s string) string { return textStyle.Sprint(s) }
}
