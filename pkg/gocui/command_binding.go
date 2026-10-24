package gocui

// SetCommandKeybinding adds an opt-in keybinding that matches ASCII command
// letters case-insensitively. It is intended for commands, not literal text.
func (g *Gui) SetCommandKeybinding(viewname string, key Key, handler func(*Gui, *View) error) {
	kb := newKeybinding(viewname, NormalizeCommandKey(key), handler)
	kb.command = true
	g.keybindings = append(g.keybindings, kb)
}
