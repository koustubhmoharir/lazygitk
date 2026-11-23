package gui

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/jesseduffield/lazygit/pkg/config"
)

// Fork: the built-in file renderer highlights a file's syntax with chroma, and numbers its
// lines (see types.FileContentTask). It holds the whole file in memory, so a file over
// maxHighlightedFileSize is not highlighted; it is streamed as plain text instead.

// maxHighlightedFileSize is the largest file the built-in renderer highlights.
const maxHighlightedFileSize = 1 << 20

// fileHighlightAppearance is what the highlighting depends on in the terminal.
type fileHighlightAppearance struct {
	// lightBackground selects the Catppuccin Latte style; otherwise Mocha.
	lightBackground bool
	// trueColor selects a formatter for 24-bit colour; otherwise 256 colours.
	trueColor bool
}

// readForHighlight reads the rest of a file's content, up to one byte more than
// maxHighlightedFileSize. tooLarge reports whether the file is larger: then data holds the
// bytes read, which the caller streams in front of the rest of rest.
func readForHighlight(rest io.Reader) (data []byte, tooLarge bool, err error) {
	data, err = io.ReadAll(io.LimitReader(rest, maxHighlightedFileSize+1))
	return data, len(data) > maxHighlightedFileSize, err
}

// highlightFileContent returns content as its lines, numbered, and each coloured with the
// syntax of the language that path names. A path that names no language, or a file that
// chroma can't highlight, is shown as plain text, with its line numbers.
func highlightFileContent(path string, content []byte, appearance fileHighlightAppearance) string {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	if text == "" {
		return ""
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	style := forkFileHighlightStyle(appearance.lightBackground)
	formatter := forkFileHighlightFormatter(appearance.trueColor)
	coloured, err := colourFileLines(path, text, len(lines), style, formatter)
	if err != nil {
		coloured = lines
	}

	numberStyle := forkLineNumberStyle(appearance.lightBackground)
	width := len(strconv.Itoa(len(lines)))
	var out strings.Builder
	for i, line := range coloured {
		var number bytes.Buffer
		// The style is built here from a constant, so the number is formatted in the
		// same way the lines are, and a formatting error leaves it uncoloured.
		_ = formatter.Format(&number, numberStyle, chroma.Literator(chroma.Token{
			Type:  chroma.LineNumbers,
			Value: fmt.Sprintf("%*d", width, i+1),
		}))
		out.Write(number.Bytes())
		out.WriteString(" ")
		out.WriteString(line)
		out.WriteString("\n")
	}
	result := out.String()
	if !strings.HasSuffix(text, "\n") {
		result = strings.TrimSuffix(result, "\n")
	}
	return result
}

// colourFileLines returns the formatted text of each of the file's lines, which are
// numLines in number. Each line is formatted on its own, so a token that spans lines
// doesn't carry its colour onto the next line's number or leave it open at the end of one.
func colourFileLines(path string, text string, numLines int, style *chroma.Style, formatter chroma.Formatter) ([]string, error) {
	lexer := lexers.Match(path)
	if lexer == nil {
		return nil, fmt.Errorf("no lexer for %s", path)
	}
	iterator, err := chroma.Coalesce(lexer).Tokenise(nil, text)
	if err != nil {
		return nil, err
	}

	lineTokens := splitTokensIntoLines(iterator.Tokens())
	if len(lineTokens) < numLines {
		return nil, fmt.Errorf("chroma gave %d lines for %d", len(lineTokens), numLines)
	}
	coloured := make([]string, numLines)
	for i := range coloured {
		var buf bytes.Buffer
		if err := formatter.Format(&buf, style, chroma.Literator(lineTokens[i]...)); err != nil {
			return nil, err
		}
		coloured[i] = buf.String()
	}
	return coloured, nil
}

// splitTokensIntoLines splits tokens at their newlines. The lines have no newlines in
// them, and the last one holds whatever follows the last newline, which may be nothing.
func splitTokensIntoLines(tokens []chroma.Token) [][]chroma.Token {
	lines := [][]chroma.Token{{}}
	for _, token := range tokens {
		for i, part := range strings.Split(token.Value, "\n") {
			if i > 0 {
				lines = append(lines, []chroma.Token{})
			}
			if part != "" {
				last := len(lines) - 1
				lines[last] = append(lines[last], chroma.Token{Type: token.Type, Value: part})
			}
		}
	}
	return lines
}

// forkFileHighlightStyle is the Catppuccin style for the appearance.
func forkFileHighlightStyle(lightBackground bool) *chroma.Style {
	if lightBackground {
		return styles.Get("catppuccin-latte")
	}
	return styles.Get("catppuccin-mocha")
}

// forkFileHighlightFormatter is the terminal formatter for the appearance: 24-bit colour
// when the screen shows it, 256 colours otherwise.
func forkFileHighlightFormatter(trueColor bool) chroma.Formatter {
	if trueColor {
		return formatters.Get("terminal16m")
	}
	return formatters.Get("terminal256")
}

// forkLineNumberStyle colours line numbers with the palette's subdued text colour.
func forkLineNumberStyle(lightBackground bool) *chroma.Style {
	return chroma.MustNewStyle("fork-line-numbers", chroma.StyleEntries{
		chroma.LineNumbers: config.ForkPaletteSubtext(lightBackground),
	})
}
