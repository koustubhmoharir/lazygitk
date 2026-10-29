package gocui

import (
	"io"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/stretchr/testify/assert"
)

// probeTty answers the probe with a fixed reply, or never if reply is empty,
// until it is drained.
type probeTty struct {
	tcell.Tty

	reply   string
	replies chan string
	written string
	started bool
}

func newProbeTty(reply string) *probeTty {
	return &probeTty{reply: reply, replies: make(chan string, 1)}
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
		reply    string
		expected bool
	}{
		{"SGR with semicolons", "\x1bP1$r0;48;2;1;2;3m\x1b\\\x1b[?61;4c", true},
		{"SGR with colons", "\x1bP1$r48:2::1:2:3m\x1b\\\x1b[?1;2c", true},
		{"RGB capability", "\x1bP0$r\x1b\\\x1bP1+r524742=383B383B38\x1b\\\x1b[?62c", true},
		{"SGR rounded to the palette", "\x1bP1$r0;48;5;16m\x1b\\\x1b[?62c", false},
		{"device attributes only", "\x1b[?1;2c", false},
		{"no answer", "", false},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			tty := newProbeTty(s.reply)
			assert.Equal(t, s.expected, probeTrueColor(tty))
			assert.Equal(t, requestTrueColorSgr+requestRgbCapability+requestDeviceAttributes, tty.written)
			assert.False(t, tty.started, "the tty is left stopped for tcell")
		})
	}
}
