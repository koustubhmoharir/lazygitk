package controllers

import "github.com/jesseduffield/lazygit/pkg/commands/git_commands"

// isFullSelectedCommitOID excludes synthetic rows and revision expressions.
// Git still verifies the object's existence/type before an operation starts.
func isFullSelectedCommitOID(oid string) bool { return git_commands.IsFullObjectID(oid) }
