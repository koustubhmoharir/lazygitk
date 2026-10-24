package daemon

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// CombineRebasePayload travels in a private file rather than the environment:
// neither a large commit selection nor its complete messages must hit ARG_MAX.
type CombineRebasePayload struct {
	KeeperOID string
	Todo      string
	Message   string
}

type CombineRebaseInstruction struct {
	PayloadPath string
}

func NewCombineRebaseInstruction(payloadPath string) Instruction {
	return &CombineRebaseInstruction{PayloadPath: payloadPath}
}

func (self *CombineRebaseInstruction) Kind() DaemonKind { return DaemonKindCombineRebase }
func (self *CombineRebaseInstruction) SerializedInstructions() string {
	return serializeInstruction(self)
}

func (self *CombineRebaseInstruction) run(log *logrus.Entry) error {
	return handleInteractiveRebase(log, func(path string) error {
		return installCombineRebaseTodo(path, self.PayloadPath)
	})
}

// A native reword may bypass its editor after a hook fails, losing the approved
// message on --continue. Compile that message action to a native exec amendment
// instead. --reschedule-failed-exec leaves failures for an explicit continue.
// Git owns the payload's lifetime in rebase-merge, including success and abort.
const combineAmendTodo = `exec git commit --amend --allow-empty --cleanup=verbatim --file="$(git rev-parse --git-path rebase-merge/fork-combine-message)"`

func installCombineRebaseTodo(todoPath, payloadPath string) error {
	data, err := os.ReadFile(payloadPath)
	if err != nil {
		return err
	}
	var payload CombineRebasePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	if len(payload.KeeperOID) != 40 && len(payload.KeeperOID) != 64 {
		return fmt.Errorf("invalid combine keeper object ID")
	}
	if _, err := hex.DecodeString(payload.KeeperOID); err != nil {
		return fmt.Errorf("invalid combine keeper object ID: %w", err)
	}
	rest, ok := strings.CutPrefix(payload.Todo, "reword "+payload.KeeperOID+"\n")
	if !ok || strings.ContainsRune(payload.Message, '\x00') {
		return fmt.Errorf("invalid combine message or todo")
	}
	messagePath := filepath.Join(filepath.Dir(todoPath), "fork-combine-message")
	message, err := os.OpenFile(messagePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := message.WriteString(payload.Message)
	closeErr := message.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	// This static exec line contains no interpolated path or message text. Git's
	// shell resolves its own state path, even for worktrees or newline paths.
	content := "pick " + payload.KeeperOID + "\n" + combineAmendTodo + "\n" + rest
	return os.WriteFile(todoPath, []byte(content), 0o600)
}
