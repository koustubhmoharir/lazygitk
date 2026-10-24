package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestCombineRebaseEditorProcess(t *testing.T) {
	if os.Getenv("LAZYGIT_TEST_COMBINE_EDITOR") != "1" {
		return
	}
	os.Args = []string{os.Args[0], os.Args[len(os.Args)-1]}
	Handle(logrus.NewEntry(logrus.New()))
	os.Exit(0)
}

type combineRebaseFixture struct {
	root, payload, editor string
}

func (f *combineRebaseFixture) run(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", f.root, "-c", "commit.gpgSign=false"}, args...)...)
	cmd.Env = append(os.Environ(), "LAZYGIT_TEST_COMBINE_EDITOR=1", "GIT_SEQUENCE_EDITOR="+f.editor, "GIT_EDITOR=true", "GIT_TERMINAL_PROMPT=0")
	cmd.Env = append(cmd.Env, ToEnvVars(NewCombineRebaseInstruction(f.payload))...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (f *combineRebaseFixture) git(t *testing.T, args ...string) string {
	t.Helper()
	out, err := f.run(args...)
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(out)
}

func (f *combineRebaseFixture) write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.root, path), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (f *combineRebaseFixture) commit(t *testing.T, file, content, subject string) string {
	t.Helper()
	f.write(t, file, content)
	f.git(t, "add", "--", file)
	f.git(t, "commit", "-qm", subject)
	return f.git(t, "rev-parse", "HEAD")
}

func newCombineRebaseFixture(t *testing.T) *combineRebaseFixture {
	t.Helper()
	name := "repo with spaces"
	if runtime.GOOS != "windows" {
		name += "\nand newline"
	}
	root := filepath.Join(t.TempDir(), name)
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	f := &combineRebaseFixture{root: root, payload: filepath.Join(t.TempDir(), "payload"), editor: `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`").Replace(filepath.ToSlash(binary)) + `" -test.run=^TestCombineRebaseEditorProcess$ --`}
	f.git(t, "init", "-q", "-b", "main")
	f.git(t, "config", "user.name", "Test")
	f.git(t, "config", "user.email", "test@example.invalid")
	f.git(t, "config", "core.hooksPath", filepath.Join(root, "hooks"))
	f.git(t, "config", "rebase.autoStash", "false")
	return f
}

func (f *combineRebaseFixture) prepare(t *testing.T, keeper, todo, message string) {
	t.Helper()
	data, err := json.Marshal(CombineRebasePayload{KeeperOID: keeper, Todo: todo, Message: message})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.payload, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func (f *combineRebaseFixture) message(t *testing.T, ref string) string {
	t.Helper()
	out, err := f.run("cat-file", "commit", ref)
	if err != nil {
		t.Fatal(err)
	}
	_, message, ok := strings.Cut(out, "\n\n")
	if !ok {
		t.Fatal("missing commit message")
	}
	return message
}

func (f *combineRebaseFixture) assertNoState(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(f.root, ".git", "rebase-merge")); !os.IsNotExist(err) {
		t.Fatalf("rebase state survived: %v", err)
	}
}

func TestCombineRebasePersistsLargeMessageAndUnselectedCommits(t *testing.T) {
	for _, root := range []bool{true, false} {
		t.Run(fmt.Sprint("root=", root), func(t *testing.T) {
			f := newCombineRebaseFixture(t)
			base := "--root"
			if !root {
				base = f.commit(t, "base", "base\n", "base")
			}
			keeper := f.commit(t, "a", "a\n", "keeper")
			middle := f.commit(t, "b", "b\n", "unselected middle")
			selected := f.commit(t, "c", "c\n", "selected later")
			tail := f.commit(t, "d", "d\n", "unselected tail")
			message := "Approved summary\n\n" + strings.Repeat("full message text\n", 12000)
			f.prepare(t, keeper, "reword "+keeper+"\nfixup "+selected+"\npick "+middle+"\npick "+tail+"\n", message)
			f.git(t, "rebase", "-i", "--reschedule-failed-exec", "--no-autosquash", "--no-autostash", base)
			if got := f.message(t, "HEAD~2"); got != message {
				t.Fatalf("approved message changed (got %d bytes, want %d)", len(got), len(message))
			}
			if f.message(t, "HEAD~1") != "unselected middle\n" || f.message(t, "HEAD") != "unselected tail\n" {
				t.Fatal("an unselected message was changed")
			}
			for _, name := range []string{"a", "b", "c", "d"} {
				if f.git(t, "show", "HEAD:"+name) != name {
					t.Fatalf("lost %s", name)
				}
			}
			f.assertNoState(t)
		})
	}
}

func TestCombineRebaseHookFailureResumesApprovedMessage(t *testing.T) {
	f := newCombineRebaseFixture(t)
	keeper := f.commit(t, "a", "a\n", "keeper")
	selected := f.commit(t, "b", "b\n", "selected")
	message := "Approved summary\n\nComplete approved description.\n"
	f.prepare(t, keeper, "reword "+keeper+"\nfixup "+selected+"\n", message)
	hooks := filepath.Join(f.root, "hooks")
	if err := os.Mkdir(hooks, 0o700); err != nil {
		t.Fatal(err)
	}
	hook := filepath.Join(hooks, "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if out, err := f.run("rebase", "-i", "--reschedule-failed-exec", "--root"); err == nil || !strings.Contains(out, "rescheduled") {
		t.Fatalf("expected failed exec to remain for explicit continue: %v\n%s", err, out)
	}
	if err := os.Remove(hook); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(f.payload); err != nil {
		t.Fatal(err)
	}
	f.git(t, "rebase", "--continue")
	if f.message(t, "HEAD") != message {
		t.Fatal("approved message lost after hook failure")
	}
	f.assertNoState(t)
}

func TestCombineRebaseFixupAndUnselectedConflicts(t *testing.T) {
	for _, abort := range []bool{false, true} {
		t.Run(fmt.Sprint("abort=", abort), func(t *testing.T) {
			f := newCombineRebaseFixture(t)
			keeper := f.commit(t, "file", "keeper\n", "keeper")
			middle := f.commit(t, "file", "middle\n", "unselected middle")
			selected := f.commit(t, "file", "selected\n", "selected")
			message := "Approved summary\n\nAll selected text.\n"
			f.prepare(t, keeper, "reword "+keeper+"\nfixup "+selected+"\npick "+middle+"\n", message)
			if out, err := f.run("rebase", "-i", "--reschedule-failed-exec", "--root"); err == nil || !strings.Contains(out, "CONFLICT") {
				t.Fatalf("expected selected fixup conflict: %v\n%s", err, out)
			}
			if err := os.Remove(f.payload); err != nil {
				t.Fatal(err)
			}
			if abort {
				f.git(t, "rebase", "--abort")
				if f.git(t, "rev-parse", "HEAD") != selected {
					t.Fatal("abort did not restore original HEAD")
				}
				f.assertNoState(t)
				return
			}
			f.write(t, "file", "selected resolved\n")
			f.git(t, "add", "file")
			if out, err := f.run("rebase", "--continue"); err == nil || !strings.Contains(out, "CONFLICT") {
				t.Fatalf("expected unselected pick conflict: %v\n%s", err, out)
			}
			f.write(t, "file", "middle resolved\n")
			f.git(t, "add", "file")
			f.git(t, "rebase", "--continue")
			if f.message(t, "HEAD~1") != message || f.message(t, "HEAD") != "unselected middle\n" {
				t.Fatal("continuation lost combined message or rewrote unselected message")
			}
			f.assertNoState(t)
		})
	}
}
