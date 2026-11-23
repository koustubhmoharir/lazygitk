package config

import (
	"errors"
	"fmt"
	"text/template"
	"text/template/parse"

	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Fork: a file renderer shows the content of a file in the full listing of a commit's
// files (see git.fileRenderers). The renderers are cycled with t d n and t d p while the
// main view shows a file's content, as diff renderers are for a diff. Unset or empty
// means the built-in highlighter, then plain text. A non-empty list is used as given.

// FileRendererType is the kind of a file renderer.
type FileRendererType int

const (
	// FileRendererType_Builtin highlights the file's syntax, with line numbers.
	FileRendererType_Builtin FileRendererType = iota
	// FileRendererType_Plain shows the file as it is.
	FileRendererType_Plain
	// FileRendererType_StdinFilter runs the renderer's command with the file's content on stdin.
	FileRendererType_StdinFilter
)

// forkDefaultFileRenderers is the list used when git.fileRenderers is empty.
func forkDefaultFileRenderers() []FileRendererConfig {
	return []FileRendererConfig{
		{Type: "builtin"},
		{Type: "plain"},
	}
}

// getType returns the kind of the renderer. The config is validated when it is loaded,
// so an unknown type is only possible for a renderer made in code; it shows plain text.
func (self *FileRendererConfig) getType() FileRendererType {
	switch self.Type {
	case "builtin":
		return FileRendererType_Builtin
	case "stdinFilter":
		return FileRendererType_StdinFilter
	default:
		return FileRendererType_Plain
	}
}

func validateFileRenderers(fileRenderers []FileRendererConfig) error {
	for _, fileRenderer := range fileRenderers {
		switch fileRenderer.Type {
		case "builtin", "plain":
			if fileRenderer.Command != "" {
				return fmt.Errorf("git.fileRenderers: 'command' cannot be used with file renderer type '%s'.", fileRenderer.Type)
			}
		case "stdinFilter":
			if fileRenderer.Command == "" {
				return errors.New("git.fileRenderers: 'command' must be specified for file renderer type 'stdinFilter'.")
			}
			if err := checkFileRendererCommand(fileRenderer.Command); err != nil {
				return fmt.Errorf("git.fileRenderers: can't use the command '%s': %w", fileRenderer.Command, err)
			}
		default:
			return fmt.Errorf("git.fileRenderers: unknown type '%s'. Allowed values: builtin, plain, stdinFilter", fileRenderer.Type)
		}
	}
	return nil
}

// fileRendererFilenamePlaceholder is the only placeholder a file renderer's command can
// use: the file's path, shell-quoted. The command is a Go template, so it can't contain
// anything else.
const fileRendererFilenamePlaceholder = "filename"

// fileRendererFuncs are the functions a file renderer's command template can call.
func fileRendererFuncs(filename func() string) template.FuncMap {
	return template.FuncMap{fileRendererFilenamePlaceholder: filename}
}

// checkFileRendererCommand reports why a stdinFilter command can't be used: it doesn't
// parse, or it has anything but text and {{filename}}. Nothing else is looked at, so
// the command is never run with a placeholder that doesn't exist.
func checkFileRendererCommand(command string) error {
	tmpl, err := template.New("command").
		Funcs(fileRendererFuncs(func() string { return "" })).
		Option("missingkey=error").
		Parse(command)
	if err != nil {
		return err
	}

	for _, node := range tmpl.Tree.Root.Nodes {
		if _, ok := node.(*parse.TextNode); ok {
			continue
		}
		action, ok := node.(*parse.ActionNode)
		if !ok || !isFilenamePlaceholder(action.Pipe) {
			return fmt.Errorf("the only placeholder allowed is {{%s}}", fileRendererFilenamePlaceholder)
		}
	}
	return nil
}

// isFilenamePlaceholder reports whether a pipeline is exactly the filename function call.
func isFilenamePlaceholder(pipe *parse.PipeNode) bool {
	if pipe == nil || len(pipe.Decl) > 0 || len(pipe.Cmds) != 1 {
		return false
	}
	args := pipe.Cmds[0].Args
	if len(args) != 1 {
		return false
	}
	ident, ok := args[0].(*parse.IdentifierNode)
	return ok && ident.Ident == fileRendererFilenamePlaceholder
}

// ResolveCommand returns the command of a stdinFilter renderer for the file at filename,
// with {{filename}} replaced by quote applied to the path. quote is the platform's shell
// quoting, so the path reaches the shell as one argument.
func (self *FileRendererConfig) ResolveCommand(filename string, quote func(string) string) (string, error) {
	command, err := utils.ResolveTemplate(self.Command, nil, fileRendererFuncs(func() string {
		return quote(filename)
	}))
	if err != nil {
		return "", fmt.Errorf("git.fileRenderers: can't use the command '%s': %w", self.Command, err)
	}
	return command, nil
}
