package gui

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeFileCmd records whether the command was stopped and waited for.
type fakeFileCmd struct {
	terminated bool
	waited     bool
}

func (c *fakeFileCmd) Wait() error      { c.waited = true; return nil }
func (c *fakeFileCmd) Terminate() error { c.terminated = true; return nil }
func (c *fakeFileCmd) String() string   { return "git cat-file blob" }

// countingReader counts the bytes read from it.
type countingReader struct {
	r    io.Reader
	read int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.read += n
	return n, err
}

func TestSniffFileContentStreamsTextAndReadsOnlyAPrefix(t *testing.T) {
	content := strings.Repeat("a", 20000)
	output := &countingReader{r: strings.NewReader(content)}
	cmd := &fakeFileCmd{}
	closed := false

	gotCmd, gotOutput := sniffFileContent(cmd, output, func() { closed = true }, "note")

	assert.Equal(t, 8000, output.read, "only the detection prefix is read before the view takes over")
	assert.False(t, cmd.terminated)
	assert.False(t, closed)
	assert.Same(t, cmd, gotCmd)

	rest, err := io.ReadAll(gotOutput)
	assert.NoError(t, err)
	assert.Equal(t, content, string(rest), "text is streamed in full, the prefix included")
}

func TestSniffFileContentShowsNoteForBinaryAndStopsTheCommand(t *testing.T) {
	output := &countingReader{r: strings.NewReader("a\x00b" + strings.Repeat("c", 20000))}
	cmd := &fakeFileCmd{}
	closed := false

	gotCmd, gotOutput := sniffFileContent(cmd, output, func() { closed = true }, "Binary file, not shown")

	assert.True(t, cmd.terminated)
	assert.True(t, cmd.waited)
	assert.True(t, closed)
	assert.Equal(t, 8000, output.read)
	assert.Equal(t, "git cat-file blob", gotCmd.String())
	assert.NoError(t, gotCmd.Wait())
	assert.NoError(t, gotCmd.Terminate())

	shown, err := io.ReadAll(gotOutput)
	assert.NoError(t, err)
	assert.Equal(t, "Binary file, not shown", string(shown))
}

func TestSniffFileContentDetectsNulAsGitDoes(t *testing.T) {
	tests := []struct {
		name   string
		size   int
		nulAt  int
		binary bool
	}{
		{"NUL at the start", 100, 0, true},
		{"NUL at the last byte of the detection window", 20000, binaryDetectionSize - 1, true},
		{"NUL just past the detection window", 20000, binaryDetectionSize, false},
		{"NUL far into a long text file", 20000, 15000, false},
		{"empty file", 0, -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := bytes.Repeat([]byte("a"), tt.size)
			if tt.nulAt >= 0 {
				content[tt.nulAt] = 0
			}
			cmd := &fakeFileCmd{}
			_, output := sniffFileContent(cmd, bytes.NewReader(content), func() {}, "note")
			assert.Equal(t, tt.binary, cmd.terminated)
			shown, err := io.ReadAll(output)
			assert.NoError(t, err)
			if tt.binary {
				assert.Equal(t, "note", string(shown))
			} else {
				assert.Equal(t, content, shown)
			}
		})
	}
}
