package config

import (
	"errors"
	"fmt"
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
		default:
			return fmt.Errorf("git.fileRenderers: unknown type '%s'. Allowed values: builtin, plain, stdinFilter", fileRenderer.Type)
		}
	}
	return nil
}
