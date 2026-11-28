// Package promptmark lets prompt strings carry semantic role markers, such as
// "this text is a branch name". The markers are resolved to theme styles just
// before the prompt is displayed, so the styles follow the theme and the
// message text itself stays plain.
//
// A marked span is encoded as:
//
//	OPEN ROLE text... CLOSE
//
// where OPEN is U+E000, ROLE is one rune identifying the role (see the Role
// constants), and CLOSE is U+E001. Both delimiters are Unicode private-use
// characters, so they do not occur in ordinary text.
//
// Marking happens before values are substituted, so a marked span may contain
// a placeholder such as "{{.name}}" or a format verb such as "%s". The
// markers do not interfere with either.
//
// Nesting is not supported. A span that is not closed before another OPEN
// (or the end of the string) is treated as unbalanced: its delimiters are
// dropped and its text is kept unstyled. A stray CLOSE is dropped. An
// unknown role is kept as unstyled text. Resolve and Strip never panic.
//
// This package is a leaf: it imports nothing from pkg/gui or pkg/i18n, so
// that translation code can use it too.
package promptmark

import (
	"strings"
)

// Role identifies what a marked span of text is. The rune value is the one
// written into the marker.
type Role rune

const (
	RoleBranch       Role = 'b' // local branch name
	RoleRemoteBranch Role = 'r' // remote branch name
	RoleRemote       Role = 'R' // remote name
	RoleTag          Role = 't' // tag name
	RoleHash         Role = 'h' // commit hash
	RolePath         Role = 'p' // file or directory path
	RoleKey          Role = 'k' // key label, as shown in the options bar
	RoleCount        Role = 'n' // count of items
	RoleWarn         Role = 'w' // destructive or irreversible consequence
	RoleLabel        Role = 'l' // attribute name in a "name: value" line, with its colon
)

const (
	open  = '\uE000'
	close = '\uE001'
)

// Branch marks s as a local branch name.
func Branch(s string) string { return mark(RoleBranch, s) }

// RemoteBranch marks s as a remote branch name.
func RemoteBranch(s string) string { return mark(RoleRemoteBranch, s) }

// Remote marks s as a remote name.
func Remote(s string) string { return mark(RoleRemote, s) }

// Tag marks s as a tag name.
func Tag(s string) string { return mark(RoleTag, s) }

// Hash marks s as a commit hash.
func Hash(s string) string { return mark(RoleHash, s) }

// Path marks s as a file or directory path.
func Path(s string) string { return mark(RolePath, s) }

// Key marks s as a key label.
func Key(s string) string { return mark(RoleKey, s) }

// Count marks s as a count of items.
func Count(s string) string { return mark(RoleCount, s) }

// Warn marks s as a destructive or irreversible consequence.
func Warn(s string) string { return mark(RoleWarn, s) }

// Label marks s as the name of an attribute in a "name: value" line. s includes
// the colon, so the colon is styled with the name.
func Label(s string) string { return mark(RoleLabel, s) }

// RefAs marks text that names ref with the role of the kind of ref: a full ref
// by its prefix (refs/heads/ is a branch, refs/remotes/ a remote branch,
// refs/tags/ a tag), a full object ID as a hash, and anything else, such as a
// short branch name from a reflog entry, as a branch.
func RefAs(ref, text string) string {
	switch {
	case strings.HasPrefix(ref, "refs/remotes/"):
		return RemoteBranch(text)
	case strings.HasPrefix(ref, "refs/tags/"):
		return Tag(text)
	case isFullObjectID(ref):
		return Hash(text)
	default:
		return Branch(text)
	}
}

// Ref marks ref, as its own text, with the role of the kind of ref. See RefAs.
func Ref(ref string) string { return RefAs(ref, ref) }

// isFullObjectID reports whether s is a full SHA-1 or SHA-256 object ID.
func isFullObjectID(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func mark(role Role, s string) string {
	if s == "" {
		return ""
	}
	return string(open) + string(role) + s + string(close)
}

// Resolve replaces each marked span in s with styleFor(role)(text). If
// styleFor returns nil for a role, the text is kept unstyled. A nil styleFor
// behaves like Strip.
func Resolve(s string, styleFor func(Role) func(string) string) string {
	return render(s, styleFor)
}

// Strip removes all markers from s and keeps the text.
func Strip(s string) string {
	return render(s, nil)
}

func render(s string, styleFor func(Role) func(string) string) string {
	if !strings.ContainsRune(s, open) && !strings.ContainsRune(s, close) {
		return s
	}

	rs := []rune(s)
	var b strings.Builder
	for i := 0; i < len(rs); {
		switch rs[i] {
		case close:
			// A stray close marker.
			i++
		case open:
			i = renderSpan(rs, i, styleFor, &b)
		default:
			b.WriteRune(rs[i])
			i++
		}
	}
	return b.String()
}

// renderSpan handles the marked span that starts at rs[start] (an OPEN rune)
// and writes its output to b. It returns the index to continue from.
func renderSpan(rs []rune, start int, styleFor func(Role) func(string) string, b *strings.Builder) int {
	// rs[start+1] must be a role. If it is missing or is itself a delimiter,
	// the OPEN marker is malformed and dropped.
	if start+1 >= len(rs) || rs[start+1] == open || rs[start+1] == close {
		return start + 1
	}
	role := Role(rs[start+1])

	textStart := start + 2
	end := textStart
	for end < len(rs) && rs[end] != open && rs[end] != close {
		end++
	}

	text := string(rs[textStart:end])
	if end >= len(rs) || rs[end] == open {
		// Unbalanced: keep the text, unstyled. The next OPEN, if any, is
		// handled by the caller's loop.
		b.WriteString(text)
		return end
	}

	// rs[end] is CLOSE.
	if text == "" {
		return end + 1
	}
	if styleFor == nil {
		b.WriteString(text)
		return end + 1
	}
	if style := styleFor(role); style != nil {
		b.WriteString(style(text))
	} else {
		b.WriteString(text)
	}
	return end + 1
}
