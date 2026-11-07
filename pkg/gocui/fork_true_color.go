package gocui

import (
	"bytes"
	"os"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/gdamore/tcell/v3"
)

// Fork: tcell only uses true colour when COLORTERM says the terminal has it,
// and rounds every other colour to the 256-colour palette. Many terminals that
// show true colour don't set COLORTERM, notably Windows Terminal inside WSL,
// where the palette's dark tints round to greys. So when COLORTERM doesn't say,
// we ask the terminal before tcell starts.

const (
	// Sets a true colour background and asks for the current SGR (DECRQSS). A
	// terminal with true colour answers with the same colour.
	requestTrueColorSgr = "\x1b[48;2;1;2;3m\x1bP$qm\x1b\\\x1b[0m"
	// Asks for the RGB capability (XTGETTCAP, "RGB" in hex), which tmux answers
	requestRgbCapability = "\x1bP+q524742\x1b\\"
	// Asks for the primary device attributes, which every terminal answers. The
	// answer comes after those to the queries before it, or after where they
	// would have been, so it ends the wait.
	requestDeviceAttributes = "\x1b[c"

	// How long to wait for the terminal's answers. The answer to the device
	// attributes ends the wait long before this on any terminal nearby.
	trueColorProbeTimeout = 300 * time.Millisecond
)

var (
	deviceAttributesReply = regexp.MustCompile(`\x1b\[\?[0-9;]*c`)
	// The start of a device attributes reply at the end of the input
	partialDeviceAttributesReply = regexp.MustCompile(`\x1b\[\?[0-9;]*$`)
	// The answer to requestTrueColorSgr, in any of the ways that terminals
	// write a true colour background: 48;2;1;2;3, 48:2:1:2:3 or 48:2::1:2:3
	trueColorSgrReply  = regexp.MustCompile(`\x1bP1\$r[0-9;:]*48[:;]2[:;]{1,2}1[:;]2[:;]3[0-9;:]*m`)
	rgbCapabilityReply = regexp.MustCompile(`\x1bP1\+r524742`)
)

// trueColorScreenOptions returns the options that make tcell use true colour
// when the terminal shows it but COLORTERM doesn't say so. It also returns the
// input that the probe read but didn't use up; tcell must read it first (see
// probeTrueColor).
func trueColorScreenOptions(tty tcell.Tty) ([]tcell.TerminfoScreenOption, []byte) {
	if !shouldProbeTrueColor() {
		return nil, nil
	}
	trueColor, typedAhead := probeTrueColor(tty)
	if !trueColor {
		return nil, typedAhead
	}
	return []tcell.TerminfoScreenOption{tcell.OptColors(1 << 24)}, typedAhead
}

// shouldProbeTrueColor says whether tcell's own detection leaves true colour
// open. tcell always uses it on Windows. It leaves out the terminals that we
// don't send queries to (see colorSchemeQueries).
func shouldProbeTrueColor() bool {
	if runtime.GOOS == "windows" ||
		slices.Contains([]string{"truecolor", "24bit"}, os.Getenv("COLORTERM")) ||
		os.Getenv("NO_COLOR") != "" || os.Getenv("TCELL_TRUECOLOR") == "disable" {
		return false
	}
	return strings.Contains(
		colorSchemeQueries(os.Getenv("TERM"), os.Getenv("TERM_PROGRAM"), os.Getenv("TCELL_NEGOTIATE")),
		requestColorScheme)
}

// probeTrueColor asks the terminal whether it shows true colour, and leaves
// the tty stopped for tcell to start. It also returns the input it read that
// isn't a reply to its queries, in the order it arrived, so that the caller can
// hand it to tcell. The user may have typed while we waited. tcell's Start
// flushes the tty, so that input must not be left there.
func probeTrueColor(tty tcell.Tty) (bool, []byte) {
	if err := tty.Start(); err != nil {
		return false, nil
	}
	defer func() { _ = tty.Stop() }()

	if _, err := tty.Write([]byte(requestTrueColorSgr + requestRgbCapability + requestDeviceAttributes)); err != nil {
		return false, nil
	}

	replies := make(chan []byte, 1)
	go safe(func() {
		replies <- readUntilDeviceAttributes(tty)
	})

	select {
	case reply := <-replies:
		trueColor := trueColorSgrReply.Match(reply) || rgbCapabilityReply.Match(reply)
		return trueColor, withoutProbeReplies(reply, false)
	case <-time.After(trueColorProbeTimeout):
		// Ends the read; the reader returns what it has read so far, and the
		// input that arrives later stays in the tty for tcell.
		_ = tty.Drain()
		reply := <-replies
		return false, withoutProbeReplies(reply, true)
	}
}

// readUntilDeviceAttributes reads the terminal's answers up to the one to
// requestDeviceAttributes, or until reading fails. It returns everything read,
// including any input after that answer in the same read.
func readUntilDeviceAttributes(tty tcell.Tty) []byte {
	var input []byte
	buffer := make([]byte, 256)
	for len(input) < 4096 {
		n, err := tty.Read(buffer)
		input = append(input, buffer[:n]...)
		if deviceAttributesReply.Match(input) {
			break
		}
		if err != nil || n == 0 {
			break
		}
	}
	return input
}

// withoutProbeReplies removes the terminal's replies to the probe's queries
// (the complete DCS strings and the device attributes reply) from input, and
// keeps every other byte in order. A DCS with no terminating ST runs to the end
// of the input. On the timeout path, that is a partial reply, so it is dropped.
// Otherwise it is kept, because the reply to the device attributes ends the read
// and nothing says that the DCS is a reply.
func withoutProbeReplies(input []byte, timedOut bool) []byte {
	var kept []byte
	rest := input
	for {
		start := bytes.Index(rest, []byte("\x1bP"))
		if start < 0 {
			kept = append(kept, rest...)
			break
		}
		kept = append(kept, rest[:start]...)
		end := bytes.Index(rest[start+2:], []byte("\x1b\\"))
		if end < 0 {
			if !timedOut {
				kept = append(kept, rest[start:]...)
			}
			break
		}
		rest = rest[start+2+end+2:]
	}
	kept = deviceAttributesReply.ReplaceAll(kept, nil)
	if timedOut {
		// A device attributes reply cut off by the timeout is partial too.
		kept = partialDeviceAttributesReply.ReplaceAll(kept, nil)
	}
	return kept
}

// TrueColor says whether the screen shows true colour; otherwise colours are
// rounded to the terminal's palette.
func (g *Gui) TrueColor() bool {
	return g.screen != nil && g.screen.Colors() > 256
}
