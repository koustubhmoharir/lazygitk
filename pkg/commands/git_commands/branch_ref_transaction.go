package git_commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/jesseduffield/lazygit/pkg/utils"
)

// branchRefCommitError means commit was sent but Git did not finish cleanly.
// Callers must refresh/report this uncertainty, never silently retry the write.
type branchRefCommitError struct{ cause error }

func (err *branchRefCommitError) Error() string {
	return "Git's ref transaction did not finish cleanly; the branch may have changed. Refresh before retrying: " + err.cause.Error()
}

func (err *branchRefCommitError) Unwrap() error { return err.cause }

// guardedBranchRefUpdate holds Git's target-ref lock while guard inspects the
// still-committed ref type and fresh worktree reservations. An OID-only CAS
// cannot distinguish a direct ref from a symbolic alias with the same OID.
// All mutation remains inside Git's normal ref transaction and hooks.
func (self *BranchCommands) guardedBranchRefUpdate(ref, newOID, oldOID, reason string, guard func() error) (resultErr error) {
	if !strings.HasPrefix(ref, "refs/heads/") || ref == "refs/heads/" || strings.ContainsAny(ref, "\x00 \t\r\n\v\f") {
		return fmt.Errorf("invalid local branch ref for transaction")
	}
	if guard == nil {
		return fmt.Errorf("branch ref transaction requires a final safety guard")
	}
	if err := validateObjectID(newOID); err != nil {
		return err
	}
	if strings.Trim(newOID, "0") == "" {
		return fmt.Errorf("branch placement cannot delete a ref")
	}
	if err := validateObjectID(oldOID); err != nil {
		return err
	}
	if len(oldOID) != len(newOID) {
		return fmt.Errorf("branch transaction object ID lengths differ")
	}

	cmdObj := self.cmd.New(NewGitCmd("update-ref").
		Arg("--stdin", "--no-deref", "--create-reflog", "-m", reason).ToArgv()).
		SetWd(self.repoPaths.WorktreePath())
	cmd := cmdObj.GetCmd()
	output := newBranchRefProtocolOutput()
	stderr := &branchRefDiagnosticTail{}
	cmd.Stdout, cmd.Stderr = output, stderr
	// A hook's detached descendant must not hold output-copy pipes forever
	// after Git exits. A timeout is reported, not mistaken for a clean commit.
	cmd.WaitDelay = 2 * time.Second
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	// The normal runner owns stdout/stdin and retries whole commands. This
	// interactive protocol needs acknowledgement-driven IO and must never retry
	// a possibly committed transaction. Keep the builder's cwd/environment.
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}
	done := make(chan struct{})
	var waitErr error
	go utils.Safe(func() {
		defer close(done)
		waitErr = cmd.Wait()
	})
	commitSent := false
	defer func() {
		// EOF aborts an uncommitted transaction, including guard/protocol errors.
		_ = stdin.Close()
		<-done
		if waitErr != nil {
			diagnostic := strings.TrimSpace(string(stderr.data))
			if diagnostic == "" {
				diagnostic = strings.TrimSpace(string(output.diagnostic.data))
			}
			resultErr = errors.Join(resultErr, fmt.Errorf("Git ref transaction: %w: %s", waitErr, diagnostic))
		}
		if self.Common != nil && self.Log != nil && len(output.diagnostic.data) > 0 {
			self.Log.Debugf("Git ref transaction output: %s", output.diagnostic.data)
		}
		if commitSent && resultErr != nil {
			resultErr = &branchRefCommitError{cause: resultErr}
		}
	}()
	send := func(command string) error {
		_, err := io.WriteString(stdin, command+"\n")
		return err
	}
	acknowledge := func(phase string) error {
		for {
			if output.acknowledged(phase) {
				return nil
			}
			select {
			case <-output.changed:
			case <-done:
				// The final ACK and process completion may arrive together.
				if output.acknowledged(phase) {
					return nil
				}
				return fmt.Errorf("Git ref transaction ended before %s acknowledgement", phase)
			}
		}
	}
	if err := send("start"); err != nil {
		return err
	}
	if err := acknowledge("start"); err != nil {
		return err
	}
	if err := send("update " + ref + " " + newOID + " " + oldOID); err != nil {
		return err
	}
	if err := send("prepare"); err != nil {
		return err
	}
	if err := acknowledge("prepare"); err != nil {
		return err
	}
	if err := guard(); err != nil {
		if abortErr := send("abort"); abortErr != nil {
			return errors.Join(err, abortErr)
		}
		return errors.Join(err, acknowledge("abort"))
	}
	commitSent = true
	if err := send("commit"); err != nil {
		return err
	}
	return acknowledge("commit")
}

// Git reference-transaction hooks can write to either stream, including text
// without a trailing newline. Track ACKs independently of bounded diagnostics
// so verbose hooks cannot fill a channel, hide an ACK, or grow memory forever.
type branchRefProtocolOutput struct {
	mutex      sync.Mutex
	seen       map[string]bool
	changed    chan struct{}
	line       []byte
	diagnostic branchRefDiagnosticTail
}

func newBranchRefProtocolOutput() *branchRefProtocolOutput {
	return &branchRefProtocolOutput{seen: make(map[string]bool), changed: make(chan struct{}, 1)}
}

func (output *branchRefProtocolOutput) acknowledged(phase string) bool {
	output.mutex.Lock()
	defer output.mutex.Unlock()
	return output.seen[phase]
}

func (output *branchRefProtocolOutput) Write(data []byte) (int, error) {
	output.mutex.Lock()
	defer output.mutex.Unlock()
	count := len(data)
	_, _ = output.diagnostic.Write(data)
	for len(data) > 0 {
		end := bytes.IndexByte(data, '\n')
		chunk := data
		if end >= 0 {
			chunk = data[:end]
		}
		const lineLimit = 64 // only the suffix containing an ACK is needed
		if len(chunk) >= lineLimit {
			output.line = append(output.line[:0], chunk[len(chunk)-lineLimit:]...)
		} else {
			if excess := len(output.line) + len(chunk) - lineLimit; excess > 0 {
				output.line = output.line[excess:]
			}
			output.line = append(output.line, chunk...)
		}
		if end < 0 {
			break
		}
		line := strings.TrimSuffix(string(output.line), "\r")
		for _, phase := range []string{"start", "prepare", "commit", "abort"} {
			if strings.HasSuffix(line, phase+": ok") {
				output.seen[phase] = true
				select {
				case output.changed <- struct{}{}:
				default:
				}
			}
		}
		output.line = output.line[:0]
		data = data[end+1:]
	}
	return count, nil
}

type branchRefDiagnosticTail struct{ data []byte }

func (tail *branchRefDiagnosticTail) Write(data []byte) (int, error) {
	const limit = 64 * 1024
	count := len(data)
	if count >= limit {
		tail.data = append(tail.data[:0], data[count-limit:]...)
	} else {
		if excess := len(tail.data) + count - limit; excess > 0 {
			tail.data = tail.data[excess:]
		}
		tail.data = append(tail.data, data...)
	}
	return count, nil
}
