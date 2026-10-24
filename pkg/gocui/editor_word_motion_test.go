package gocui

import "testing"

func TestEditorCtrlArrowWordMotion(t *testing.T) {
	oldLeft, oldRight := moveWordLeftKeybinding, moveWordRightKeybinding
	oldBackspace, oldDelete := backspaceWordKeybinding, forwardDeleteWordKeybinding
	defer func() {
		moveWordLeftKeybinding, moveWordRightKeybinding = oldLeft, oldRight
		backspaceWordKeybinding, forwardDeleteWordKeybinding = oldBackspace, oldDelete
	}()
	moveWordLeftKeybinding = []Key{NewKey(KeyArrowLeft, "", ModCtrl)}
	moveWordRightKeybinding = []Key{NewKey(KeyArrowRight, "", ModCtrl)}

	tests := []struct {
		name    string
		content string
		cursor  int
		key     Key
		want    int
	}{
		{"left ascii", "one two", 7, NewKey(KeyArrowLeft, "", ModCtrl), 4},
		{"left boundary", "one two", 0, NewKey(KeyArrowLeft, "", ModCtrl), 0},
		{"right ascii", "one two", 0, NewKey(KeyArrowRight, "", ModCtrl), 3},
		{"right boundary", "one two", 7, NewKey(KeyArrowRight, "", ModCtrl), 7},
		{"left unicode", "éclair 世界", len("éclair 世界"), NewKey(KeyArrowLeft, "", ModCtrl), len("éclair ")},
		{"right multiline", "one\ntwo", 0, NewKey(KeyArrowRight, "", ModCtrl), 3},
		{"left multiline", "one\ntwo", len("one\ntwo"), NewKey(KeyArrowLeft, "", ModCtrl), 4},
		{"ordinary left", "one", 3, NewKeyName(KeyArrowLeft), 2},
		{"ordinary right", "one", 0, NewKeyName(KeyArrowRight), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewView("editor", 0, 0, 20, 5, OutputNormal)
			view.TextArea.TypeString(tt.content)
			view.TextArea.cursor = tt.cursor
			if !SimpleEditor(view, tt.key) {
				t.Fatal("editor did not handle key")
			}
			if got := view.TextArea.cursor; got != tt.want {
				t.Errorf("cursor = %d, want %d", got, tt.want)
			}
			if got := view.TextArea.GetContent(); got != tt.content {
				t.Errorf("content changed to %q", got)
			}
		})
	}
}
