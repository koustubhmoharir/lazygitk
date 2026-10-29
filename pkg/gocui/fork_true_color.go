package gocui

import (
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
	// The answer to requestTrueColorSgr, in any of the ways that terminals
	// write a true colour background: 48;2;1;2;3, 48:2:1:2:3 or 48:2::1:2:3
	trueColorSgrReply  = regexp.MustCompile(`\x1bP1\$r[0-9;:]*48[:;]2[:;]{1,2}1[:;]2[:;]3[0-9;:]*m`)
	rgbCapabilityReply = regexp.MustCompile(`\x1bP1\+r524742`)
)

// trueColorScreenOptions returns the options that make tcell use true colour
// when the terminal shows it but COLORTERM doesn't say so.
func trueColorScreenOptions(tty tcell.Tty) []tcell.TerminfoScreenOption {
	if !shouldProbeTrueColor() || !probeTrueColor(tty) {
		return nil
	}
	return []tcell.TerminfoScreenOption{tcell.OptColors(1 << 24)}
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
// the tty stopped for tcell to start.
func probeTrueColor(tty tcell.Tty) bool {
	if err := tty.Start(); err != nil {
		return false
	}
	defer func() { _ = tty.Stop() }()

	if _, err := tty.Write([]byte(requestTrueColorSgr + requestRgbCapability + requestDeviceAttributes)); err != nil {
		return false
	}

	replies := make(chan string, 1)
	go func() {
		replies <- readUntilDeviceAttributes(tty)
	}()

	select {
	case reply := <-replies:
		return trueColorSgrReply.MatchString(reply) || rgbCapabilityReply.MatchString(reply)
	case <-time.After(trueColorProbeTimeout):
		// Ends the read; tcell flushes whatever arrives later when it starts
		// the tty.
		_ = tty.Drain()
		<-replies
		return false
	}
}

// readUntilDeviceAttributes reads the terminal's answers up to the one to
// requestDeviceAttributes, or until reading fails.
func readUntilDeviceAttributes(tty tcell.Tty) string {
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
	return string(input)
}

// TrueColor says whether the screen shows true colour; otherwise colours are
// rounded to the terminal's palette.
func (g *Gui) TrueColor() bool {
	return g.screen != nil && g.screen.Colors() > 256
}
