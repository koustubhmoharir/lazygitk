package gocui

import (
	"io"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/stretchr/testify/assert"
)

// probeTty answers the probe with a fixed reply, or never if reply is empty,
// until it is drained. typed is input that the user types before the reply.
type probeTty struct {
	tcell.Tty

	typed   string
	reply   string
	replies chan string
	written string
	started bool
}

func newProbeTty(typed string, reply string) *probeTty {
	return &probeTty{typed: typed, reply: reply, replies: make(chan string, 4)}
}

func (self *probeTty) Start() error {
	self.started = true
	return nil
}

func (self *probeTty) Stop() error {
	self.started = false
	return nil
}

func (self *probeTty) Drain() error {
	close(self.replies)
	return nil
}

func (self *probeTty) Write(p []byte) (int, error) {
	self.written += string(p)
	if self.typed != "" {
		self.replies <- self.typed
	}
	if self.reply != "" {
		self.replies <- self.reply
	}
	return len(p), nil
}

func (self *probeTty) Read(p []byte) (int, error) {
	reply, ok := <-self.replies
	if !ok {
		return 0, io.EOF
	}
	return copy(p, reply), nil
}

func TestProbeTrueColor(t *testing.T) {
	scenarios := []struct {
		name     string
		typed    string
		reply    string
		expected bool
		// The input that the probe leaves for tcell
		leftover string
	}{
		{"SGR with semicolons", "", "\x1bP1$r0;48;2;1;2;3m\x1b\\\x1b[?61;4c", true, ""},
		{"SGR with colons", "", "\x1bP1$r48:2::1:2:3m\x1b\\\x1b[?1;2c", true, ""},
		{"RGB capability", "", "\x1bP0$r\x1b\\\x1bP1+r524742=383B383B38\x1b\\\x1b[?62c", true, ""},
		{"SGR rounded to the palette", "", "\x1bP1$r0;48;5;16m\x1b\\\x1b[?62c", false, ""},
		{"device attributes only", "", "\x1b[?1;2c", false, ""},
		{"no answer", "", "", false, ""},
		{"key typed before the replies", "j", "\x1bP1$r0;48;2;1;2;3m\x1b\\\x1b[?61;4c", true, "j"},
		{"bytes after the device attributes reply", "", "\x1bP1$r0;48;2;1;2;3m\x1b\\\x1b[?61;4ck", true, "k"},
		{"key typed and no answer", "j", "", false, "j"},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			tty := newProbeTty(s.typed, s.reply)
			trueColor, leftover := probeTrueColor(tty)
			assert.Equal(t, s.expected, trueColor)
			assert.Equal(t, s.leftover, string(leftover))
			assert.Equal(t, requestTrueColorSgr+requestRgbCapability+requestDeviceAttributes, tty.written)
			assert.False(t, tty.started, "the tty is left stopped for tcell")
		})
	}
}

func TestWithoutProbeReplies(t *testing.T) {
	scenarios := []struct {
		name     string
		input    string
		timedOut bool
		expected string
	}{
		{"keys around a DCS reply", "j\x1bP1$r0m\x1b\\k", false, "jk"},
		{"keys around the device attributes reply", "a\x1b[?62cb", false, "ab"},
		{"XTGETTCAP reply", "\x1bP1+r524742=383B\x1b\\\x1b[?62c", false, ""},
		{"empty DCS", "x\x1bP\x1b\\y", false, "xy"},
		{"incomplete DCS after a reply", "\x1bP1$r0m\x1b\\x\x1bP1$r", false, "x\x1bP1$r"},
		{"incomplete DCS on the timeout path", "x\x1bP1$r0", true, "x"},
		{"complete DCS on the timeout path", "x\x1bP1$r0m\x1b\\y", true, "xy"},
		{"partial device attributes reply on the timeout path", "x\x1b[?62", true, "x"},
		{"partial device attributes reply after a reply", "x\x1b[?62cy\x1b[?6", false, "xy\x1b[?6"},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, string(withoutProbeReplies([]byte(s.input), s.timedOut)))
		})
	}
}
