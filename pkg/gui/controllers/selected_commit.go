package controllers

import "regexp"

var selectedCommitOIDPattern = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)

// isFullSelectedCommitOID excludes synthetic rows and revision expressions.
// Git still verifies the object's existence/type before an operation starts.
func isFullSelectedCommitOID(oid string) bool { return selectedCommitOIDPattern.MatchString(oid) }
