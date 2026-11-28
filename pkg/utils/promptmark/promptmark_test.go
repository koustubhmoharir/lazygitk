package promptmark

import (
	"fmt"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// brackets styles every role as "<role:text>", so the tests can see which
// role each span got.
func brackets(role Role) func(string) string {
	switch role {
	case RoleBranch, RoleRemoteBranch, RoleRemote, RoleTag, RoleHash,
		RolePath, RoleKey, RoleCount, RoleWarn, RoleLabel:
		return func(s string) string { return "<" + string(rune(role)) + ":" + s + ">" }
	}
	return nil
}

func TestConstructorsAddTheirRole(t *testing.T) {
	cases := map[string]struct {
		marked string
		role   rune
	}{
		"branch":        {Branch("x"), 'b'},
		"remote branch": {RemoteBranch("x"), 'r'},
		"remote":        {Remote("x"), 'R'},
		"tag":           {Tag("x"), 't'},
		"hash":          {Hash("x"), 'h'},
		"path":          {Path("x"), 'p'},
		"key":           {Key("x"), 'k'},
		"count":         {Count("x"), 'n'},
		"warn":          {Warn("x"), 'w'},
		"label":         {Label("x"), 'l'},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, ""+string(c.role)+"x", c.marked)
			assert.Equal(t, "x", Strip(c.marked))
			assert.Equal(t, "<"+string(c.role)+":x>", Resolve(c.marked, brackets))
		})
	}
}

func TestEmptySpanVanishes(t *testing.T) {
	assert.Equal(t, "", Branch(""))
	assert.Equal(t, "a..b", Resolve("a."+Branch("")+".b", brackets))
}

func TestResolveInsideText(t *testing.T) {
	s := "Delete " + Branch("main") + " and tag " + Tag("v1") + "?"
	assert.Equal(t, "Delete <b:main> and tag <t:v1>?", Resolve(s, brackets))
	assert.Equal(t, "Delete main and tag v1?", Strip(s))
}

func TestResolveWithNilStyleKeepsText(t *testing.T) {
	s := "a " + Warn("b") + " c"
	unstyled := func(Role) func(string) string { return nil }
	assert.Equal(t, "a b c", Resolve(s, unstyled))
	assert.Equal(t, "a b c", Resolve(s, nil))
}

func TestPlainTextIsUnchanged(t *testing.T) {
	for _, s := range []string{"", "plain", "100% sure: {{.x}} %s\n\tindented"} {
		assert.Equal(t, s, Strip(s))
		assert.Equal(t, s, Resolve(s, brackets))
	}
}

func TestPlaceholderSubstitutedAfterMarking(t *testing.T) {
	marked := "Delete " + Branch("{{.name}}") + "?"
	substituted := utils.ResolvePlaceholderString(marked, map[string]string{"name": "feature/x"})
	assert.Equal(t, "Delete <b:feature/x>?", Resolve(substituted, brackets))
}

func TestFormatVerbSubstitutedAfterMarking(t *testing.T) {
	marked := fmt.Sprintf(Branch("%s")+" and "+Warn("%d")+" more", "main", 3)
	assert.Equal(t, "<b:main> and <w:3> more", Resolve(marked, brackets))
}

func TestUnbalancedOpenKeepsText(t *testing.T) {
	// The first rune after OPEN is the role, so "b" below is a branch role
	// and the text is what follows it. No CLOSE comes before the end.
	assert.Equal(t, "ac", Strip("a\uE000bc"))
	assert.Equal(t, "a", Strip("a\uE000b"))
	// A lone OPEN at the very end, with no role.
	assert.Equal(t, "a", Strip("a\uE000"))
	// An unbalanced span followed by a balanced one.
	assert.Equal(t, "unclosed<b:x>", Resolve("\uE000bunclosed"+Branch("x"), brackets))
}

func TestStrayCloseIsDropped(t *testing.T) {
	assert.Equal(t, "ab", Strip("ab"))
	assert.Equal(t, "<b:x>", Resolve(Branch("x")+"", brackets))
}

func TestUnknownRoleKeepsText(t *testing.T) {
	s := "azxyb"
	assert.Equal(t, "axyb", Resolve(s, brackets))
	assert.Equal(t, "axyb", Strip(s))
}

func TestNestingIsNotSupported(t *testing.T) {
	// The inner span is resolved; the outer span is unbalanced and loses
	// its styling but keeps its text.
	nested := "b" + "a" + Tag("t") + "c"
	assert.Equal(t, "a<t:t>c", Resolve(nested, brackets))
}

func TestMalformedDelimiterPairsDoNotPanic(t *testing.T) {
	inputs := []string{
		"", "", "", "", "",
		"b", "", "b",
	}
	for _, in := range inputs {
		assert.NotPanics(t, func() {
			_ = Strip(in)
			_ = Resolve(in, brackets)
		}, "input %q", in)
	}
}

func TestRefAsClassifiesByKindOfRef(t *testing.T) {
	const sha1 = "0123456789abcdef0123456789abcdef01234567"
	const sha256 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	cases := map[string]struct {
		ref  string
		role rune
	}{
		"local branch ref":     {"refs/heads/main", 'b'},
		"remote branch ref":    {"refs/remotes/origin/main", 'r'},
		"tag ref":              {"refs/tags/v1.0", 't'},
		"full sha-1 object id": {sha1, 'h'},
		"full sha-256 id":      {sha256, 'h'},
		"short branch name":    {"main", 'b'},
		"short hash":           {"0123456", 'b'},
		"sha-1 length, upper":  {"0123456789ABCDEF0123456789ABCDEF01234567", 'b'},
		"sha-1 length, other":  {"0123456789abcdef0123456789abcdef0123456g", 'b'},
		"sha-1 minus one":      {sha1[:39], 'b'},
		"sha-1 plus one":       {sha1 + "0", 'b'},
		"sha-256 minus one":    {sha256[:63], 'b'},
		"empty ref":            {"", 'b'},
		"HEAD":                 {"HEAD", 'b'},
		"upstream expression":  {"@{upstream}", 'b'},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, "<"+string(c.role)+":x>", Resolve(RefAs(c.ref, "x"), brackets))
			assert.Equal(t, "x", Strip(RefAs(c.ref, "x")))
		})
	}
}

func TestRefMarksTheRefItself(t *testing.T) {
	assert.Equal(t, "<b:refs/heads/main>", Resolve(Ref("refs/heads/main"), brackets))
	assert.Equal(t, "<r:refs/remotes/origin/main>", Resolve(Ref("refs/remotes/origin/main"), brackets))
	assert.Equal(t, "<t:refs/tags/v1.0>", Resolve(Ref("refs/tags/v1.0"), brackets))
	assert.Equal(t, "<h:0123456789abcdef0123456789abcdef01234567>", Resolve(Ref("0123456789abcdef0123456789abcdef01234567"), brackets))
	assert.Equal(t, "<b:feature>", Resolve(Ref("feature"), brackets))
}
