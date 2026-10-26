package oscommands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOSCommandRun(t *testing.T) {
	type scenario struct {
		args []string
		test func(error)
	}

	scenarios := []scenario{
		{
			[]string{"rmdir", "unexisting-folder"},
			func(err error) {
				assert.Regexp(t, "rmdir.*unexisting-folder.*", err.Error())
			},
		},
	}

	for _, s := range scenarios {
		c := NewDummyOSCommand()
		s.test(c.Cmd.New(s.args).Run())
	}
}

func TestOSCommandQuote(t *testing.T) {
	osCommand := NewDummyOSCommand()

	osCommand.Platform.OS = "linux"

	actual := osCommand.Quote("hello `test`")

	expected := "\"hello \\`test\\`\""

	assert.EqualValues(t, expected, actual)
}

// TestOSCommandQuoteSingleQuote tests the quote function with ' quotes explicitly for Linux
func TestOSCommandQuoteSingleQuote(t *testing.T) {
	osCommand := NewDummyOSCommand()

	osCommand.Platform.OS = "linux"

	actual := osCommand.Quote("hello 'test'")

	expected := `"hello 'test'"`

	assert.EqualValues(t, expected, actual)
}

// TestOSCommandQuoteDoubleQuote tests the quote function with " quotes explicitly for Linux
func TestOSCommandQuoteDoubleQuote(t *testing.T) {
	osCommand := NewDummyOSCommand()

	osCommand.Platform.OS = "linux"

	actual := osCommand.Quote(`hello "test"`)

	expected := `"hello \"test\""`

	assert.EqualValues(t, expected, actual)
}

// TestOSCommandQuoteWindows tests the quote function for Windows
func TestOSCommandQuoteWindows(t *testing.T) {
	osCommand := NewDummyOSCommand()

	osCommand.Platform.OS = "windows"

	actual := osCommand.Quote(`hello "test" 'test2'`)

	expected := `"hello \"test\" 'test2'"`

	assert.EqualValues(t, expected, actual)
}

// On Windows, NewShell must hand the command to cmd.exe verbatim.
func TestNewShellWindowsPassesMetacharactersVerbatim(t *testing.T) {
	osCommand := NewDummyOSCommand()
	platform := &Platform{OS: "windows", Shell: "cmd", ShellArg: "/c"}
	osCommand.Platform = platform
	osCommand.Cmd.platform = platform

	command := `echo a && echo b | sort > out.txt < in.txt %PATH%`

	assert.Equal(t,
		[]string{"cmd", "/s", "/c", command},
		osCommand.Cmd.NewShell(command, "").Args(),
	)
}

func TestOSCommandFileType(t *testing.T) {
	type scenario struct {
		path  string
		setup func()
		test  func(string)
	}

	scenarios := []scenario{
		{
			"testFile",
			func() {
				f, err := os.Create("testFile")
				if err != nil {
					panic(err)
				}
				if err := f.Close(); err != nil {
					panic(err)
				}
			},
			func(output string) {
				assert.EqualValues(t, "file", output)
			},
		},
		{
			"file with spaces",
			func() {
				f, err := os.Create("file with spaces")
				if err != nil {
					panic(err)
				}
				if err := f.Close(); err != nil {
					panic(err)
				}
			},
			func(output string) {
				assert.EqualValues(t, "file", output)
			},
		},
		{
			"testDirectory",
			func() {
				if err := os.Mkdir("testDirectory", 0o644); err != nil {
					panic(err)
				}
			},
			func(output string) {
				assert.EqualValues(t, "directory", output)
			},
		},
		{
			"nonExistent",
			func() {},
			func(output string) {
				assert.EqualValues(t, "other", output)
			},
		},
	}

	for _, s := range scenarios {
		s.setup()
		s.test(FileType(s.path))
		assert.NoError(t, os.RemoveAll(s.path))
	}
}

func TestOSCommandAppendLineToFile(t *testing.T) {
	type scenario struct {
		path  string
		setup func(string)
		test  func(string)
	}

	scenarios := []scenario{
		{
			filepath.Join(os.TempDir(), "testFile"),
			func(path string) {
				if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
					panic(err)
				}
			},
			func(output string) {
				assert.EqualValues(t, "hello\nworld\n", output)
			},
		},
		{
			filepath.Join(os.TempDir(), "emptyTestFile"),
			func(path string) {
				if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
					panic(err)
				}
			},
			func(output string) {
				assert.EqualValues(t, "world\n", output)
			},
		},
		{
			filepath.Join(os.TempDir(), "testFileWithNewline"),
			func(path string) {
				if err := os.WriteFile(path, []byte("hello\n"), 0o600); err != nil {
					panic(err)
				}
			},
			func(output string) {
				assert.EqualValues(t, "hello\nworld\n", output)
			},
		},
	}

	for _, s := range scenarios {
		s.setup(s.path)
		osCommand := NewDummyOSCommand()
		if err := osCommand.AppendLineToFile(s.path, "world"); err != nil {
			panic(err)
		}
		f, err := os.ReadFile(s.path)
		if err != nil {
			panic(err)
		}
		s.test(string(f))
		_ = os.RemoveAll(s.path)
	}
}

// Fork: copies use the configured command when there is one, and otherwise
// the injected terminal clipboard writer (roadmap item 21).
func TestCopyToClipboardFork(t *testing.T) {
	type scenario struct {
		name   string
		cmd    string
		writer bool
		// nil when no command should run
		runner  func(t *testing.T) *FakeCmdObjRunner
		text    string
		wantErr string
		// the text the writer receives; nil when it must not be called
		wantWrites []string
	}

	scenarios := []scenario{
		{
			name:   "configured command runs and the writer is not used",
			cmd:    "printf '%s' {{text}} > clipboard",
			writer: true,
			runner: func(t *testing.T) *FakeCmdObjRunner {
				t.Helper()
				return NewFakeRunner(t).
					ExpectArgs([]string{"bash", "-c", `printf '%s' "hello" > clipboard`}, "", nil)
			},
			text: "hello",
		},
		{
			name:       "without a command the writer gets the exact text",
			writer:     true,
			text:       "line one\nline two ✓",
			wantWrites: []string{"line one\nline two ✓"},
		},
		{
			name:    "without a command and without a writer an error is returned",
			text:    "hello",
			wantErr: "no terminal clipboard writer is configured",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			c := NewDummyOSCommand()
			c.Platform.OS = "darwin"
			c.UserConfig().OS.CopyToClipboardCmd = s.cmd
			var runner *FakeCmdObjRunner
			if s.runner != nil {
				runner = s.runner(t)
				c.Cmd = NewDummyCmdObjBuilder(runner)
			}

			var writes []string
			if s.writer {
				c.SetClipboardWriter(func(text string) error {
					writes = append(writes, text)
					return nil
				})
			}

			err := c.CopyToClipboard(s.text)
			if s.wantErr != "" {
				assert.ErrorContains(t, err, s.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, s.wantWrites, writes)
			if runner != nil {
				runner.CheckForMissingCalls()
			}
		})
	}
}
