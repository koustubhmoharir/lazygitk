package theme

import (
	"regexp"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/jesseduffield/lazygit/pkg/utils/promptmark"
	"github.com/stretchr/testify/assert"
)

var sgrPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func TestResolvePromptMarksUsesPaneStyles(t *testing.T) {
	cases := map[string]struct {
		marked   string
		expected string
	}{
		"branch":        {promptmark.Branch("x"), style.FgCyan.SetBold().Sprint("x")},
		"remote branch": {promptmark.RemoteBranch("x"), style.FgYellow.Sprint("x")},
		"remote":        {promptmark.Remote("x"), style.FgYellow.Sprint("x")},
		"tag":           {promptmark.Tag("x"), style.FgMagenta.SetBold().Sprint("x")},
		"hash":          {promptmark.Hash("x"), style.FgYellow.Sprint("x")},
		"path":          {promptmark.Path("x"), style.AttrBold.Sprint("x")},
		"key":           {promptmark.Key("x"), OptionsFgColor.Sprint("x")},
		"count":         {promptmark.Count("x"), style.AttrBold.Sprint("x")},
		"warn":          {promptmark.Warn("x"), style.FgRed.SetBold().Sprint("x")},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, ResolvePromptMarks(c.marked))
		})
	}
}

func TestResolvePromptMarksDoesNotLeakStyleIntoFollowingText(t *testing.T) {
	for _, marked := range []string{
		promptmark.Warn("cannot"),
		promptmark.Branch("main"),
		promptmark.Key("q"),
	} {
		resolved := ResolvePromptMarks("before " + marked + " after")
		span := ResolvePromptMarks(marked)

		// A styled span must end with a reset, and the text after it must
		// be plain. (The default key colour is unstyled, so it has no codes.)
		if sgrPattern.MatchString(span) {
			assert.True(t, strings.HasSuffix(span, "\x1b[0m"), "span %q should end with a reset", span)
		}
		assert.Equal(t, "before "+span+" after", resolved)
		assert.True(t, strings.HasSuffix(resolved, " after"))
	}
}

func TestResolvePromptMarksKeepsWrapWidth(t *testing.T) {
	plain := "Are you sure you want to delete branch main (remote origin/main), tag v1.0, path src/app/server.go, and 12 commits? This cannot be undone."
	marked := "Are you sure you want to delete branch " + promptmark.Branch("main") +
		" (remote " + promptmark.RemoteBranch("origin/main") + "), tag " + promptmark.Tag("v1.0") +
		", path " + promptmark.Path("src/app/server.go") + ", and " + promptmark.Count("12") +
		" commits? " + promptmark.Warn("This cannot be undone.")

	for _, width := range []int{10, 24, 40, 78} {
		plainLines, _, _ := utils.WrapViewLinesToWidth(true, false, plain, width, 4)
		resolvedLines, _, _ := utils.WrapViewLinesToWidth(true, false, ResolvePromptMarks(marked), width, 4)

		assert.Len(t, resolvedLines, len(plainLines), "width %d", width)
		for i := range plainLines {
			assert.Equal(t, plainLines[i], sgrPattern.ReplaceAllString(resolvedLines[i], ""), "width %d, line %d", width, i)
		}
	}
}
