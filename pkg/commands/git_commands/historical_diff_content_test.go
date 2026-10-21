package git_commands

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrepareHistoricalFileDiffMaterializesLiteralPathContent(t *testing.T) {
	root := initHistoricalRepo(t)
	largeBinary := bytes.Repeat([]byte{0x00, 0xff, '\n', 'x', 0x80}, (3*1024*1024)/5)
	fixtures := []struct {
		path string
		data []byte
	}{
		{path: "nested/with spaces/λ.txt", data: []byte("unicode and spaced path\n")},
		{path: "-leading.txt", data: []byte("leading option-like path\n")},
		{path: "empty.bin", data: []byte{}},
		{path: "large.bin", data: largeBinary},
	}
	if runtime.GOOS != "windows" {
		fixtures = append(fixtures, struct {
			path string
			data []byte
		}{path: ":(glob)*.txt", data: []byte("literal pathspec magic\n")})
	}

	for _, fixture := range fixtures {
		writeTestFile(t, filepath.Join(root, filepath.FromSlash(fixture.path)), fixture.data, 0o644)
		gitTest(t, root, "--literal-pathspecs", "add", "--", fixture.path)
	}
	gitTest(t, root, "commit", "-qm", "historical content fixtures")
	commit := gitTest(t, root, "rev-parse", "HEAD")
	destination := filepath.Join(root, "chosen file")
	destinationBytes := []byte("destination is independent of all source paths\x00\xff\n")
	writeTestFile(t, destination, destinationBytes, 0o644)
	commands := realHistoricalDiffCommands(t, root)

	for _, fixture := range fixtures {
		t.Run(fixture.path, func(t *testing.T) {
			diff, err := commands.PrepareHistoricalFileDiff(commit, fixture.path, destination)
			if !assert.NoError(t, err) || !assert.NotNil(t, diff) {
				return
			}
			t.Cleanup(func() {
				if err := diff.Close(); err != nil {
					t.Errorf("close historical file diff: %v", err)
				}
			})
			sourcePath := diff.SourcePath
			assert.Equal(t, filepath.Base(filepath.FromSlash(fixture.path)), filepath.Base(sourcePath))
			assert.Equal(t, filepath.Ext(filepath.FromSlash(fixture.path)), filepath.Ext(sourcePath))
			actualBytes := mustReadFile(t, sourcePath)
			if !bytes.Equal(fixture.data, actualBytes) {
				t.Fatalf("source bytes differ: got %d bytes, want %d bytes", len(actualBytes), len(fixture.data))
			}
			assert.Equal(t, root, diff.Command.GetCmd().Dir)
			assert.Equal(t, destination, diff.DestinationPath)
			args := diff.Command.Args()
			if !assert.GreaterOrEqual(t, len(args), 2) {
				return
			}
			assert.Equal(t, []string{sourcePath, destination}, args[len(args)-2:])
			assert.True(t, filepath.IsAbs(sourcePath))
			assert.True(t, filepath.IsAbs(destination))

			if !assert.NoError(t, diff.Close()) {
				return
			}
			assert.NoError(t, diff.Close())
			_, statErr := os.Stat(sourcePath)
			assert.ErrorIs(t, statErr, os.ErrNotExist)
		})
	}

	assert.Equal(t, destinationBytes, mustReadFile(t, destination))
	assert.Equal(t, commit, gitTest(t, root, "rev-parse", "HEAD"))
	assert.Empty(t, gitTest(t, root, "diff", "--cached", "--name-only"))
}

func TestValidateObjectID(t *testing.T) {
	valid := []struct {
		name string
		oid  string
	}{
		{name: "40 lowercase", oid: repeatByte('a', 40)},
		{name: "40 uppercase", oid: repeatByte('F', 40)},
		{name: "64 lowercase", oid: repeatByte('b', 64)},
		{name: "64 uppercase", oid: repeatByte('A', 64)},
	}
	for _, testCase := range valid {
		t.Run(testCase.name, func(t *testing.T) {
			assert.NoError(t, validateObjectID(testCase.oid))
		})
	}
	invalid := []string{"", repeatByte('a', 39), repeatByte('a', 41), repeatByte('a', 63), repeatByte('a', 65), repeatByte('g', 40), "--help", "HEAD", "-" + repeatByte('a', 40)}
	for _, oid := range invalid {
		t.Run("invalid/"+oid, func(t *testing.T) {
			assert.Error(t, validateObjectID(oid))
		})
	}
}

func repeatByte(value byte, count int) string {
	return string(bytes.Repeat([]byte{value}, count))
}
