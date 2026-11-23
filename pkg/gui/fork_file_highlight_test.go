package gui

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

var darkTrueColor = fileHighlightAppearance{lightBackground: false, trueColor: true}

func TestHighlightFileContentColoursGoAndNumbersLines(t *testing.T) {
	out := highlightFileContent("main.go", []byte("package main\n\nfunc main() {}\n"), darkTrueColor)

	assert.Contains(t, out, "\x1b[38;2;", "24-bit colour escapes for true colour")
	assert.Equal(t, "1 package main\n2 \n3 func main() {}\n", stripAnsi(out))
}

func TestHighlightFileContentFindsTheLanguageFromAPathInADirectory(t *testing.T) {
	out := highlightFileContent("pkg/gui/main.go", []byte("package main\n"), darkTrueColor)

	assert.Contains(t, out, "\x1b[38;2;")
}

func TestHighlightFileContentUses256ColoursWithoutTrueColour(t *testing.T) {
	out := highlightFileContent("main.go", []byte("package main\n"), fileHighlightAppearance{trueColor: false})

	assert.Contains(t, out, "\x1b[38;5;")
	assert.NotContains(t, out, "\x1b[38;2;")
	assert.Equal(t, "1 package main\n", stripAnsi(out))
}

func TestHighlightFileContentUsesTheLightStyleOnALightBackground(t *testing.T) {
	dark := highlightFileContent("main.go", []byte("package main\n"), darkTrueColor)
	light := highlightFileContent("main.go", []byte("package main\n"), fileHighlightAppearance{lightBackground: true, trueColor: true})

	assert.NotEqual(t, dark, light)
	assert.Equal(t, stripAnsi(dark), stripAnsi(light))
}

func TestHighlightFileContentNumbersLinesToTheWidthOfTheLargestNumber(t *testing.T) {
	content := strings.Repeat("x\n", 12)
	out := highlightFileContent("notes.unknownext", []byte(content), darkTrueColor)

	lines := strings.Split(stripAnsi(out), "\n")
	assert.Equal(t, " 1 x", lines[0])
	assert.Equal(t, " 9 x", lines[8])
	assert.Equal(t, "10 x", lines[9])
	assert.Equal(t, "12 x", lines[11])
	assert.Equal(t, "", lines[12])
}

func TestHighlightFileContentOfAnUnknownTypeIsNumberedPlainText(t *testing.T) {
	out := highlightFileContent("notes.unknownext", []byte("alpha\nbeta\n"), darkTrueColor)

	assert.Equal(t, "1 alpha\n2 beta\n", stripAnsi(out))
	// Only the numbers are coloured: after the number and its space, the text has no escapes.
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		_, text, found := strings.Cut(line, " ")
		assert.True(t, found)
		assert.NotContains(t, text, "\x1b")
	}
}

func TestHighlightFileContentWithoutTrailingNewlineKeepsItsLines(t *testing.T) {
	out := highlightFileContent("main.go", []byte("package main\nfunc main() {}"), darkTrueColor)

	assert.Equal(t, "1 package main\n2 func main() {}", stripAnsi(out))
}

func TestHighlightFileContentKeepsCRLFLinesWhole(t *testing.T) {
	out := highlightFileContent("main.go", []byte("package main\r\n\r\nfunc main() {}\r\n"), darkTrueColor)

	assert.NotContains(t, out, "\r")
	assert.Equal(t, "1 package main\n2 \n3 func main() {}\n", stripAnsi(out))
}

func TestHighlightFileContentKeepsALineOfAMultiLineTokenWhole(t *testing.T) {
	out := highlightFileContent("main.go", []byte("/* one\ntwo */\nvar x = 1\n"), darkTrueColor)

	assert.Equal(t, "1 /* one\n2 two */\n3 var x = 1\n", stripAnsi(out))
	assert.Equal(t, 3, strings.Count(out, "\n"))
}

func TestHighlightFileContentOfEmptyAndBlankFiles(t *testing.T) {
	assert.Equal(t, "", highlightFileContent("main.go", []byte(""), darkTrueColor))
	assert.Equal(t, "1 \n", stripAnsi(highlightFileContent("main.go", []byte("\n"), darkTrueColor)))
}

func TestReadForHighlightFitsTheLimit(t *testing.T) {
	content := bytes.Repeat([]byte("a"), maxHighlightedFileSize)

	data, tooLarge, err := readForHighlight(bytes.NewReader(content))

	assert.NoError(t, err)
	assert.False(t, tooLarge)
	assert.Equal(t, content, data)
}

func TestReadForHighlightOneByteOverTheLimitIsTooLarge(t *testing.T) {
	content := append(bytes.Repeat([]byte("a"), maxHighlightedFileSize), 'b', 'c')
	rest := bytes.NewReader(content)

	data, tooLarge, err := readForHighlight(rest)

	assert.NoError(t, err)
	assert.True(t, tooLarge)
	assert.Len(t, data, maxHighlightedFileSize+1)
	// What is not read is still there, so that the stream can carry on with it.
	remaining, err := io.ReadAll(rest)
	assert.NoError(t, err)
	assert.Equal(t, "c", string(remaining))
	streamed, err := io.ReadAll(io.MultiReader(bytes.NewReader(data), rest))
	assert.NoError(t, err)
	assert.Equal(t, content[:maxHighlightedFileSize+1], streamed[:maxHighlightedFileSize+1])
}

func TestReadForHighlightReportsAReadError(t *testing.T) {
	failure := errors.New("read failed")
	_, tooLarge, err := readForHighlight(io.MultiReader(strings.NewReader("abc"), errReader{failure}))

	assert.False(t, tooLarge)
	assert.ErrorIs(t, err, failure)
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }
