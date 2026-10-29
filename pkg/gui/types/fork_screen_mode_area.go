package types

// Fork: a screen mode (half or full) enlarges one area of the screen. The side
// area is the side panels; the main area is the main and secondary views. The
// area is recorded when the user changes the mode with + and -, and it is the
// area of the focused window at that moment (fork_screen_mode_area.go). Every
// other way of setting the mode means the side area.
//
// The zero value is the side area, so a fresh repo state starts in it.
type ScreenModeArea int

const (
	SCREEN_AREA_SIDE ScreenModeArea = iota
	SCREEN_AREA_MAIN
)

func (self ScreenModeArea) String() string {
	switch self {
	case SCREEN_AREA_MAIN:
		return "main"
	default:
		return "side"
	}
}
