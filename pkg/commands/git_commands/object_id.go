package git_commands

import (
	"fmt"
	"regexp"
)

var fullObjectID = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)

// IsFullObjectID excludes revision expressions and synthetic UI rows. It does
// not establish the object's existence or type; Git must still check those.
func IsFullObjectID(oid string) bool { return fullObjectID.MatchString(oid) }

func validateObjectID(oid string) error {
	if !IsFullObjectID(oid) {
		return fmt.Errorf("expected a full 40- or 64-character hexadecimal object ID")
	}
	return nil
}
