package git_commands

import (
	"strings"
	"testing"
)

func TestIsFullObjectID(t *testing.T) {
	for _, oid := range []string{strings.Repeat("a", 40), strings.Repeat("F", 64)} {
		if !IsFullObjectID(oid) || validateObjectID(oid) != nil {
			t.Errorf("valid full object ID rejected: %q", oid)
		}
	}
	for _, oid := range []string{"", "HEAD", "HEAD~1", "-leading", strings.Repeat("g", 40), strings.Repeat("a", 39), strings.Repeat("a", 41), strings.Repeat("a", 63), strings.Repeat("a", 65), strings.Repeat("a", 40) + "\n"} {
		if IsFullObjectID(oid) || validateObjectID(oid) == nil {
			t.Errorf("invalid/synthetic object ID accepted: %q", oid)
		}
	}
}
